package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"github.com/xescugc/pikoci/pikoci"
	"github.com/xescugc/pikoci/pikoci/build"
	"github.com/xescugc/pikoci/pikoci/job"
	"github.com/xescugc/pikoci/pikoci/pipeline"
	"github.com/xescugc/pikoci/pikoci/queue"
	"github.com/xescugc/pikoci/pikoci/resource"
	"github.com/xescugc/pikoci/pikoci/restype"
	"github.com/xescugc/pikoci/pikoci/runner"
	"github.com/xescugc/pikoci/pikoci/service"
	"github.com/xescugc/pikoci/pikoci/utils"
	"gocloud.dev/pubsub"
)

type Service interface {
	Run(ctx context.Context, q, t string) error
}

type Worker struct {
	topic        queue.Topic
	pikoci       pikoci.Service
	subscription queue.Subscription

	logger *slog.Logger
}

func New(s pikoci.Service, t queue.Topic, ss queue.Subscription, l *slog.Logger) *Worker {
	return &Worker{
		pikoci:       s,
		topic:        t,
		subscription: ss,
		logger:       l,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		msg, err := w.subscription.Receive(ctx)
		if err != nil {
			return fmt.Errorf("failed to receive message: %w", err)
		}

		var m queue.Body
		if err := json.Unmarshal(msg.Body, &m); err != nil {
			w.logger.Error("failed unmarshal message body", "error", err)
			msg.Ack()
			continue
		}

		cwd, err := w.createWorkDir()
		if err != nil {
			return err
		}

		w.processMessage(ctx, m, cwd)

		msg.Ack()
		os.RemoveAll(cwd)
	}
}

func (w *Worker) createWorkDir() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to create UUID: %w", err)
	}
	// We append a file "pikoci" just so CacheFile creates the full dir,
	// afterward we just get the Dir of the cwd
	cwd, err := xdg.CacheFile(filepath.Join("pikoci", id.String(), "pikoci"))
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	return filepath.Dir(cwd), nil
}

func (w *Worker) processMessage(ctx context.Context, m queue.Body, cwd string) {
	pp, err := w.pikoci.GetPipeline(ctx, m.TeamCanonical, m.PipelineName)
	if err != nil {
		w.logger.Error("failed GetPipeline", "error", err)
		return
	}

	// Parse services from the pipeline's raw HCL since they are not stored
	// in a separate DB table.
	if len(pp.Raw) > 0 && len(pp.Services) == 0 {
		svcs, err := pipeline.ParseServicesFromRaw(ctx, pp.Raw)
		if err != nil {
			w.logger.Error("failed to parse services from pipeline raw", "error", err)
		} else {
			pp.Services = svcs
		}
	}

	// Parse secret-backed variables from the pipeline's raw HCL since they
	// are not stored in a separate DB table.
	if len(pp.Raw) > 0 && len(pp.SecretVars) == 0 {
		svars, err := pipeline.ParseSecretVarsFromRaw(pp.Raw, nil)
		if err != nil {
			w.logger.Error("failed to parse secret vars from pipeline raw", "error", err)
		} else {
			pp.SecretVars = svars
		}
	}

	if m.PipelineName != "" && m.JobName != "" {
		w.processJob(ctx, m, cwd, pp)
	} else if m.PipelineName != "" && m.ResourceCanonical != "" {
		w.processResourceCheck(ctx, m, cwd, pp)
	}
}

// processJob handles executing a job: creates a build, runs the plan steps,
// runs hooks, and triggers downstream jobs.
func (w *Worker) processJob(ctx context.Context, m queue.Body, cwd string, pp *pipeline.Pipeline) {
	b := build.Build{
		Status:    build.Started,
		Steps:     []build.Step{},
		StartedAt: time.Now(),
	}
	nb, err := w.pikoci.CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, b)
	if err != nil {
		w.logger.Error("failed create build", "pipeline", m.PipelineName, "job", m.JobName, "error", err)
		return
	}
	b.ID = nb.ID

	j, err := w.pikoci.GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName)
	if err != nil {
		w.failBuild(ctx, m, b, fmt.Errorf("failed to get job: %w", err))
		return
	}

	if !w.checkPassedConstraints(ctx, m, &b, j) {
		return
	}

	// checkVersionAvailability verifies that the get step can pull a version.
	// If no version is available (e.g. manual trigger with no resource versions),
	// the build is deleted silently — no hooks run, no failure recorded.
	if !w.checkVersionAvailability(ctx, m, &b, j, pp) {
		return
	}

	failed := w.runPlan(ctx, m, &b, cwd, pp, j)

	if !failed {
		b.Status = build.Succeeded
		if err := w.updateBuild(ctx, m, b); err != nil {
			return
		}
		w.triggerDownstreamJobs(ctx, m, &b, pp, j)
		w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "", j.OnSuccess, "on_success", "succeeded")
	} else {
		w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "", j.OnFailure, "on_failure", "failed")
	}
	status := "succeeded"
	if b.Status == build.Failed {
		status = "failed"
	}
	w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "", j.Ensure, "ensure", status)
}

// checkPassedConstraints verifies that all jobs in the "passed" list have a
// successful latest build. If not, the build is deleted and false is returned.
func (w *Worker) checkPassedConstraints(ctx context.Context, m queue.Body, b *build.Build, j *job.Job) bool {
	for _, ps := range j.Plan {
		if ps.Type != job.StepTypeGet || ps.Get == nil {
			continue
		}
		g := ps.Get
		for _, p := range g.Passed {
			builds, err := w.pikoci.ListJobBuilds(ctx, m.TeamCanonical, m.PipelineName, p)
			if err != nil {
				w.failBuild(ctx, m, *b, fmt.Errorf("failed to list builds for passed job %q: %w", p, err))
				return false
			}
			if len(builds) == 0 {
				w.logger.Info("job will not run: passed job has no builds",
					"job", m.JobName, "pipeline", m.PipelineName, "passed_job", p)
				w.deleteBuild(ctx, m, *b)
				return false
			}
			if builds[0].Status != build.Succeeded {
				w.logger.Info("job will not run: passed job is not succeeded",
					"job", m.JobName, "pipeline", m.PipelineName, "passed_job", p)
				w.deleteBuild(ctx, m, *b)
				return false
			}
		}
	}
	return true
}

// checkVersionAvailability verifies that all get steps in the plan have a
// version available to pull. If any get step has no version, the build is
// deleted and false is returned (same behavior as checkPassedConstraints).
// This prevents hooks from running when no work can be done.
func (w *Worker) checkVersionAvailability(ctx context.Context, m queue.Body, b *build.Build, j *job.Job, pp *pipeline.Pipeline) bool {
	for _, ps := range j.Plan {
		if ps.Type != job.StepTypeGet || ps.Get == nil {
			continue
		}
		g := ps.Get
		rCan := utils.ResourceCanonical(g.Type, g.Name)
		r, ok := pp.Resource(rCan)
		if !ok {
			w.logger.Warn("get step references unknown resource", "resource", rCan, "job", m.JobName)
			continue
		}

		dbvers, err := w.pikoci.ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, r.Canonical)
		if err != nil {
			// Transient errors (DB, network) should fail the build, not silently delete it.
			w.failBuild(ctx, m, *b, fmt.Errorf("failed to list resource versions: %w", err))
			return false
		}

		if len(dbvers) == 0 {
			w.logger.Info("job will not run: no versions available",
				"job", m.JobName, "pipeline", m.PipelineName, "resource", r.Canonical)
			w.deleteBuild(ctx, m, *b)
			return false
		}
	}
	return true
}

// runPlan runs all plan steps (service/get/task/put) in order.
// Service steps are collected and started before other steps, then stopped unconditionally after.
// Returns true if the job failed during plan execution.
func (w *Worker) runPlan(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, j *job.Job) bool {
	// Collect service steps and remaining steps
	var serviceSteps []job.ServiceStep
	var remainingPlan []job.PlanStep
	for _, ps := range j.Plan {
		if ps.Type == job.StepTypeService && ps.Service != nil {
			serviceSteps = append(serviceSteps, *ps.Service)
		} else {
			remainingPlan = append(remainingPlan, ps)
		}
	}

	// Start all services, defer stop
	if len(serviceSteps) > 0 {
		startedServices := w.startServices(ctx, m, b, cwd, pp, serviceSteps)
		defer w.stopServices(m, b, cwd, pp, startedServices)

		// If any service failed to start, fail the build
		if len(startedServices) != len(serviceSteps) {
			return true
		}

		// Wait for ready checks
		if !w.waitForServices(ctx, m, b, cwd, pp, startedServices) {
			return true
		}
	}

	// Resolve secret-backed variables once for the entire job execution.
	resolved, err := w.resolveSecretVars(ctx, cwd, pp)
	if err != nil {
		w.failBuild(ctx, m, *b, fmt.Errorf("failed to resolve secret vars: %w", err))
		return true
	}

	// Run remaining plan steps (get/task/put)
	for _, ps := range remainingPlan {
		switch ps.Type {
		case job.StepTypeGet:
			if ps.Get == nil {
				continue
			}
			if w.runGetStep(ctx, m, b, cwd, pp, *ps.Get, ps, resolved) {
				return true
			}
		case job.StepTypeTask:
			if ps.Task == nil {
				continue
			}
			if w.runTaskStep(ctx, m, b, cwd, pp, *ps.Task, ps, resolved) {
				return true
			}
		case job.StepTypePut:
			if ps.Put == nil {
				continue
			}
			if w.runPutStep(ctx, m, b, cwd, pp, *ps.Put, ps, resolved) {
				return true
			}
		}
	}
	return false
}

// runGetStep runs a single get step (resource pull).
// Returns true if the step failed.
func (w *Worker) runGetStep(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, g job.GetStep, ps job.PlanStep, resolved ...map[string]string) bool {
	r, ok := pp.Resource(utils.ResourceCanonical(g.Type, g.Name))
	if !ok {
		return false
	}
	rt, ok := pp.ResourceType(g.Type)
	if !ok {
		return false
	}

	if rt.Pull == nil {
		return false
	}

	params := w.buildPullParams(ctx, m, b, rt, r, g)
	if params == nil {
		return true
	}

	ru, ok := pp.Runner(rt.Pull.Runner)
	if !ok {
		return false
	}

	var secretResolved map[string]string
	if len(resolved) > 0 {
		secretResolved = resolved[0]
	}
	replaceSecretPlaceholders(params, secretResolved)

	for k, v := range buildMetadataParams(b, m) {
		params[k] = v
	}

	pullArgs := make([]string, len(rt.Pull.Args))
	copy(pullArgs, rt.Pull.Args)
	replaceSecretPlaceholdersInSlice(pullArgs, secretResolved)

	rc := utils.RunnerCommand{
		Runner: rt.Pull.Runner,
		Args:   pullArgs,
		Params: params,
	}

	maxAttempts := ps.Attempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var out string
	var d time.Duration
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 && maxAttempts > 1 {
			out += fmt.Sprintf("\n--- attempt %d/%d ---\n", attempt, maxAttempts)
		}

		runCtx := ctx
		var cancel context.CancelFunc
		if ps.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, ps.Timeout)
		}

		var attemptOut string
		attemptOut, d, err = w.runRunner(runCtx, ru, cwd, rc)
		out += attemptOut

		if cancel != nil {
			cancel()
		}

		if err == nil {
			break
		}

		if runCtx.Err() == context.DeadlineExceeded {
			out += fmt.Sprintf("\nstep timed out after %s", ps.Timeout)
		}
	}

	if err != nil {
		b.Steps = append(b.Steps, build.Step{Type: "get", Name: g.Name, Logs: out, Duration: d})
		b.Status = build.Failed
		w.failBuild(ctx, m, *b, nil)
		w.logger.Error("failed to run get step", "step", g.Name, "error", err)
		w.runHooks(ctx, m, b, &b.Steps, cwd, pp, g.Name, ps.OnFailure, "on_failure")
		w.runHooks(ctx, m, b, &b.Steps, cwd, pp, g.Name, ps.Ensure, "ensure")
		return true
	}

	b.Steps = append(b.Steps, build.Step{
		Type:      "get",
		Name:      g.Name,
		VersionID: m.VersionID,
		Logs:      out,
		Duration:  d,
	})
	if err := w.updateBuild(ctx, m, *b); err != nil {
		return true
	}
	w.runHooks(ctx, m, b, &b.Steps, cwd, pp, g.Name, ps.OnSuccess, "on_success")
	w.runHooks(ctx, m, b, &b.Steps, cwd, pp, g.Name, ps.Ensure, "ensure")
	return false
}

// runTaskStep runs a single task step.
// Returns true if the step failed.
func (w *Worker) runTaskStep(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, t job.TaskStep, ps job.PlanStep, resolved ...map[string]string) bool {
	ru, ok := pp.Runner(t.Run.Runner)
	if !ok {
		return false
	}

	if t.Run.Params == nil {
		t.Run.Params = make(map[string]string)
	}

	var secretResolved map[string]string
	if len(resolved) > 0 {
		secretResolved = resolved[0]
	}
	replaceSecretPlaceholders(t.Run.Params, secretResolved)
	replaceSecretPlaceholdersInSlice(t.Run.Args, secretResolved)

	for k, v := range buildMetadataParams(b, m) {
		t.Run.Params[k] = v
	}

	maxAttempts := ps.Attempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var out string
	var d time.Duration
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 && maxAttempts > 1 {
			out += fmt.Sprintf("\n--- attempt %d/%d ---\n", attempt, maxAttempts)
		}

		runCtx := ctx
		var cancel context.CancelFunc
		if ps.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, ps.Timeout)
		}

		var attemptOut string
		attemptOut, d, err = w.runRunner(runCtx, ru, cwd, t.Run)
		out += attemptOut

		if cancel != nil {
			cancel()
		}

		if err == nil {
			break
		}

		if runCtx.Err() == context.DeadlineExceeded {
			out += fmt.Sprintf("\nstep timed out after %s", ps.Timeout)
		}
	}

	if err != nil {
		b.Steps = append(b.Steps, build.Step{Type: "task", Name: t.Name, Logs: out, Duration: d})
		b.Status = build.Failed
		w.failBuild(ctx, m, *b, nil)
		w.runHooks(ctx, m, b, &b.Steps, cwd, pp, t.Name, ps.OnFailure, "on_failure")
		w.runHooks(ctx, m, b, &b.Steps, cwd, pp, t.Name, ps.Ensure, "ensure")
		return true
	}

	b.Steps = append(b.Steps, build.Step{Type: "task", Name: t.Name, Logs: out, Duration: d})
	if err := w.updateBuild(ctx, m, *b); err != nil {
		return true
	}
	w.runHooks(ctx, m, b, &b.Steps, cwd, pp, t.Name, ps.OnSuccess, "on_success")
	w.runHooks(ctx, m, b, &b.Steps, cwd, pp, t.Name, ps.Ensure, "ensure")
	return false
}

// runPutStep runs a single put step (resource push).
// Returns true if the step failed.
func (w *Worker) runPutStep(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, p job.PutStep, ps job.PlanStep, resolved ...map[string]string) bool {
	rCan := utils.ResourceCanonical(p.Type, p.Name)
	r, ok := pp.Resource(rCan)
	if !ok {
		return false
	}
	rt, ok := pp.ResourceType(p.Type)
	if !ok {
		return false
	}

	if rt.Push == nil {
		return false
	}

	params := make(map[string]string)
	for k, v := range rt.Push.Params {
		params[k] = v
	}
	// Add resource-level params
	for k, v := range r.GetParams() {
		if slices.Contains(rt.Params, k) {
			params["param_"+k] = v
		}
	}
	// Add put-step-level params with put_ prefix
	for k, v := range p.Params {
		params["put_"+k] = v
	}
	for k, v := range buildMetadataParams(b, m) {
		params[k] = v
	}

	ru, ok := pp.Runner(rt.Push.Runner)
	if !ok {
		return false
	}

	var secretResolved map[string]string
	if len(resolved) > 0 {
		secretResolved = resolved[0]
	}
	replaceSecretPlaceholders(params, secretResolved)

	pushArgs := make([]string, len(rt.Push.Args))
	copy(pushArgs, rt.Push.Args)
	replaceSecretPlaceholdersInSlice(pushArgs, secretResolved)

	rc := utils.RunnerCommand{
		Runner: rt.Push.Runner,
		Args:   pushArgs,
		Params: params,
	}

	maxAttempts := ps.Attempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var out string
	var d time.Duration
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 && maxAttempts > 1 {
			out += fmt.Sprintf("\n--- attempt %d/%d ---\n", attempt, maxAttempts)
		}

		runCtx := ctx
		var cancel context.CancelFunc
		if ps.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, ps.Timeout)
		}

		var attemptOut string
		attemptOut, d, err = w.runRunner(runCtx, ru, cwd, rc)
		out += attemptOut

		if cancel != nil {
			cancel()
		}

		if err == nil {
			break
		}

		if runCtx.Err() == context.DeadlineExceeded {
			out += fmt.Sprintf("\nstep timed out after %s", ps.Timeout)
		}
	}

	if err != nil {
		b.Steps = append(b.Steps, build.Step{Type: "put", Name: p.Name, Logs: out, Duration: d})
		b.Status = build.Failed
		w.failBuild(ctx, m, *b, nil)
		w.logger.Error("failed to run put step", "step", p.Name, "error", err)
		w.runHooks(ctx, m, b, &b.Steps, cwd, pp, p.Name, ps.OnFailure, "on_failure")
		w.runHooks(ctx, m, b, &b.Steps, cwd, pp, p.Name, ps.Ensure, "ensure")
		return true
	}

	b.Steps = append(b.Steps, build.Step{Type: "put", Name: p.Name, Logs: out, Duration: d})
	if err := w.updateBuild(ctx, m, *b); err != nil {
		return true
	}
	w.runHooks(ctx, m, b, &b.Steps, cwd, pp, p.Name, ps.OnSuccess, "on_success")
	w.runHooks(ctx, m, b, &b.Steps, cwd, pp, p.Name, ps.Ensure, "ensure")
	return false
}

// buildPullParams assembles the environment parameters needed to pull a resource version.
// Returns nil if an error occurred (error is already handled via failBuild).
func (w *Worker) buildPullParams(ctx context.Context, m queue.Body, b *build.Build, rt restype.ResourceType, r resource.Resource, g job.GetStep) map[string]string {
	var params map[string]string
	if rt.Pull != nil && rt.Pull.Params != nil {
		params = make(map[string]string)
		for k, v := range rt.Pull.Params {
			params[k] = v
		}
	} else {
		params = make(map[string]string)
	}

	dbvers, err := w.pikoci.ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, r.Canonical)
	if err != nil {
		w.failBuild(ctx, m, *b, fmt.Errorf("failed to list resource versions: %w", err))
		return nil
	}

	if m.VersionID != 0 {
		var found bool
		for _, ver := range dbvers {
			if ver.ID == m.VersionID {
				found = true
				for k, v := range ver.Version {
					params["version_"+k] = fmt.Sprintf("%s", v)
				}
				break
			}
		}
		if !found {
			w.failBuild(ctx, m, *b, fmt.Errorf("no version found for resource %q", r.Canonical))
			return nil
		}
	} else {
		if len(dbvers) == 0 {
			w.failBuild(ctx, m, *b, fmt.Errorf("no versions for resource %q", r.Canonical))
			return nil
		}
		slices.Reverse(dbvers)
		for k, v := range dbvers[0].Version {
			params["version_"+k] = fmt.Sprintf("%s", v)
		}
	}

	for k, v := range r.GetParams() {
		if slices.Contains(rt.Params, k) {
			params["param_"+k] = v
		}
	}

	return params
}

// triggerDownstreamJobs finds jobs that depend on the current job via "passed"
// and triggers them.
func (w *Worker) triggerDownstreamJobs(ctx context.Context, m queue.Body, b *build.Build, pp *pipeline.Pipeline, j *job.Job) {
	for _, nj := range pp.Jobs {
		for _, ps := range nj.Plan {
			if ps.Type != job.StepTypeGet || ps.Get == nil {
				continue
			}
			g := ps.Get
			if slices.Contains(g.Passed, j.Name) && g.Trigger {
				qb := queue.Body{
					TeamCanonical:     m.TeamCanonical,
					PipelineName:      pp.Name,
					JobName:           nj.Name,
					ResourceCanonical: g.ResourceCanonical(),
					VersionID:         m.VersionID,
				}
				mb, err := json.Marshal(qb)
				if err != nil {
					w.failBuild(ctx, m, *b, fmt.Errorf("failed to marshal trigger body: %w", err))
					return
				}
				w.topic.Send(ctx, &pubsub.Message{Body: mb})
			}
		}
	}
}

// runHooks runs a list of hooks (on_success, on_failure, ensure) and appends
// the results as build steps.
func (w *Worker) runHooks(ctx context.Context, m queue.Body, b *build.Build, steps *[]build.Step, cwd string, pp *pipeline.Pipeline, stepName string, hooks []job.HookStep, hookType string, buildStatus ...string) {
	for i, h := range hooks {
		var out string
		var d time.Duration

		switch h.Type {
		case job.StepTypeRunner:
			if h.Runner == nil {
				continue
			}
			ru, ok := pp.Runner(h.Runner.Runner)
			if !ok {
				continue
			}
			rc := *h.Runner
			params := make(map[string]string)
			for k, v := range rc.Params {
				params[k] = v
			}
			for k, v := range buildMetadataParams(b, m) {
				params[k] = v
			}
			rc.Params = params
			if len(buildStatus) > 0 {
				rc.Params["BUILD_STATUS"] = buildStatus[0]
			}
			out, d, _ = w.runRunner(ctx, ru, cwd, rc)
		case job.StepTypePut:
			if h.Put == nil {
				continue
			}
			ps := job.PlanStep{
				Type: job.StepTypePut,
				Put:  h.Put,
			}
			w.runPutStep(ctx, m, b, cwd, pp, *h.Put, ps)
			continue
		default:
			continue
		}

		name := hookType
		if stepName != "" {
			name = stepName + ":" + hookType
		}
		if len(hooks) > 1 {
			if stepName != "" {
				name = fmt.Sprintf("%s:%d:%s", stepName, i, hookType)
			} else {
				name = fmt.Sprintf("%d:%s", i, hookType)
			}
		}

		*steps = append(*steps, build.Step{Type: "hook", Name: name, Logs: out, Duration: d})
		if err := w.updateBuild(ctx, m, *b); err != nil {
			return
		}
	}
}

// processResourceCheck handles periodic resource version checks.
func (w *Worker) processResourceCheck(ctx context.Context, m queue.Body, cwd string, pp *pipeline.Pipeline) {
	r, ok := pp.Resource(m.ResourceCanonical)
	if !ok {
		return
	}
	rt, ok := pp.ResourceType(r.Type)
	if !ok {
		return
	}

	if rt.Check == nil {
		return
	}

	params := make(map[string]string)
	for k, v := range rt.Check.Params {
		params[k] = v
	}

	dbvers, err := w.pikoci.ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, r.Canonical)
	if err != nil {
		w.logger.Error("failed to list resource versions", "error", err)
		return
	}
	if len(dbvers) != 0 {
		for k, v := range dbvers[0].Version {
			params["version_"+k] = fmt.Sprintf("%s", v)
		}
	}
	for k, v := range r.GetParams() {
		if slices.Contains(rt.Params, k) {
			params["param_"+k] = v
		}
	}

	resolved, err := w.resolveSecretVars(ctx, cwd, pp)
	if err != nil {
		w.logger.Error("failed to resolve secret vars for resource check", "error", err)
		return
	}
	replaceSecretPlaceholders(params, resolved)

	ru, ok := pp.Runner(rt.Check.Runner)
	if !ok {
		return
	}

	checkArgs := make([]string, len(rt.Check.Args))
	copy(checkArgs, rt.Check.Args)
	replaceSecretPlaceholdersInSlice(checkArgs, resolved)

	rc := utils.RunnerCommand{
		Runner: rt.Check.Runner,
		Args:   checkArgs,
		Params: params,
	}
	out, _, err := w.runRunner(ctx, ru, cwd, rc)
	if err != nil {
		r.Logs = out
		if nerr := w.pikoci.UpdatePipelineResource(ctx, m.TeamCanonical, m.PipelineName, r.Canonical, r); nerr != nil {
			w.logger.Error("failed update resource", "resource", r.Canonical, "pipeline", m.PipelineName, "error", nerr)
		}
		w.logger.Error("failed to run resource check", "error", err)
		return
	}

	if r.Logs != "" {
		r.Logs = ""
		if err := w.pikoci.UpdatePipelineResource(ctx, m.TeamCanonical, m.PipelineName, r.Canonical, r); err != nil {
			w.logger.Error("failed update resource", "resource", r.Canonical, "pipeline", m.PipelineName, "error", err)
			return
		}
	}

	sout := strings.Split(strings.Trim(out, "\n"), "\n")
	rawVers := sout[len(sout)-1]
	if rawVers == "" {
		return
	}

	vers := make([]map[string]interface{}, 0)
	if err := json.Unmarshal([]byte(rawVers), &vers); err != nil {
		w.logger.Error("failed to unmarshal versions", "raw", rawVers, "error", err)
		r.Logs = fmt.Sprintf("failed to Unmarshal versions(%s): %v", rawVers, err)
		if nerr := w.pikoci.UpdatePipelineResource(ctx, m.TeamCanonical, m.PipelineName, r.Canonical, r); nerr != nil {
			w.logger.Error("failed update resource", "resource", r.Canonical, "pipeline", m.PipelineName, "error", nerr)
		}
		return
	}

	for _, v := range vers {
		cv, err := w.pikoci.CreateResourceVersion(ctx, m.TeamCanonical, m.PipelineName, r.Canonical, resource.Version{
			Version: v,
		})
		if err != nil {
			w.logger.Error("failed to create resource version", "error", err)
			return
		}
		w.triggerResourceJobs(ctx, m, pp, r, cv)
	}
}

// triggerResourceJobs triggers jobs that depend on a resource via "get" with trigger=true.
func (w *Worker) triggerResourceJobs(ctx context.Context, m queue.Body, pp *pipeline.Pipeline, r resource.Resource, cv *resource.Version) {
	for _, j := range pp.Jobs {
		for _, ps := range j.Plan {
			if ps.Type != job.StepTypeGet || ps.Get == nil {
				continue
			}
			g := ps.Get
			// If Passed is not 0 it means is waiting for another job
			// and this trigger is only for resources
			if g.Name == r.Name && g.Type == r.Type && g.Trigger && len(g.Passed) == 0 {
				qb := queue.Body{
					TeamCanonical:     m.TeamCanonical,
					PipelineName:      pp.Name,
					JobName:           j.Name,
					ResourceCanonical: r.Canonical,
					VersionID:         cv.ID,
				}
				mb, err := json.Marshal(qb)
				if err != nil {
					w.logger.Error("failed to marshal trigger body", "error", err)
					return
				}
				w.topic.Send(ctx, &pubsub.Message{Body: mb})
			}
		}
	}
}

// updateBuild persists the current build state to the DB.
func (w *Worker) updateBuild(ctx context.Context, m queue.Body, b build.Build) error {
	err := w.pikoci.UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, b.ID, b)
	if err != nil {
		w.logger.Error("failed update build", "pipeline", m.PipelineName, "job", m.JobName, "error", err)
	}
	return err
}

func (w *Worker) failBuild(ctx context.Context, m queue.Body, b build.Build, err error) {
	b.Status = build.Failed
	if err != nil {
		b.Error = err.Error()
		w.logger.Error(err.Error())
	}
	if uerr := w.pikoci.UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, b.ID, b); uerr != nil {
		w.logger.Error("failed update build", "pipeline", m.PipelineName, "job", m.JobName, "error", uerr)
	}
}

func (w *Worker) deleteBuild(ctx context.Context, m queue.Body, b build.Build) {
	if err := w.pikoci.DeleteJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, b.ID); err != nil {
		w.logger.Error("failed delete build", "pipeline", m.PipelineName, "job", m.JobName, "error", err)
	}
}

func (w *Worker) runRunner(ctx context.Context, ru runner.Runner, cwd string, rc utils.RunnerCommand) (string, time.Duration, error) {
	envs := map[string]string{"WORKDIR": cwd}
	for k, v := range rc.Params {
		envs[k] = v
	}
	envFn := func(p string) string {
		if v, ok := envs[p]; ok {
			return v
		}
		return os.Getenv(p)
	}

	var args []string
	var out string
	for _, a := range ru.Run.Args {
		if a == "$args" {
			for _, ca := range rc.Args {
				ea := os.Expand(ca, envFn)
				if ea != "" {
					args = append(args, ea)
				}
			}
			continue
		}
		ea := os.Expand(a, envFn)
		if ea != "" {
			args = append(args, ea)
		}
	}

	cmdPath := os.Expand(ru.Run.Path, envFn)
	if cmdPath == "" {
		// Empty command path (e.g. cron pull/push with empty block), skip execution.
		return "", 0, nil
	}

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Dir = cwd
	cmd.Env = cmd.Environ()
	for k, v := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	w.logger.Debug("running command", "cmd", cmd.String(), "envs", createKeyValuePairs(envs))
	start := time.Now()
	stdouterr, err := cmd.CombinedOutput()
	duration := time.Since(start)
	out += string(stdouterr)
	if err != nil {
		out += "\n" + err.Error()
	}
	w.logger.Debug("finished running command", "out", out)

	return out, duration, err
}

// fetchSecrets resolves secret values for the given secrets map (secret_type name -> path)
// and returns them as a map of "secret_<key>" env vars.
func (w *Worker) fetchSecrets(ctx context.Context, cwd string, pp *pipeline.Pipeline, secrets map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	for stName, path := range secrets {
		st, ok := pp.SecretType(stName)
		if !ok {
			return nil, fmt.Errorf("secret_type %q not found in pipeline", stName)
		}

		// Build params: config values + path param
		params := make(map[string]string)
		for k, v := range st.Get.Params {
			params[k] = v
		}
		// Add config values as param_<key>
		for k, v := range st.Config {
			params["param_"+k] = v
		}
		// Add path as param_path (the dynamic per-step value), only if set.
		// When empty, the secret_type's config path (from st.Config) is used as default.
		if path != "" {
			params["param_path"] = path
		}

		ru, ok := pp.Runner(st.Get.Runner)
		if !ok {
			return nil, fmt.Errorf("runner %q not found for secret_type %q", st.Get.Runner, st.Name)
		}

		rc := utils.RunnerCommand{
			Runner: st.Get.Runner,
			Args:   st.Get.Args,
			Params: params,
		}

		out, _, err := w.runRunner(ctx, ru, cwd, rc)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch secret from %q at %q: %s\n%w", stName, path, out, err)
		}

		// Parse last line of stdout as JSON object
		sout := strings.Split(strings.Trim(out, "\n"), "\n")
		rawJSON := sout[len(sout)-1]

		var secretData map[string]string
		if err := json.Unmarshal([]byte(rawJSON), &secretData); err != nil {
			return nil, fmt.Errorf("failed to parse secret output from %q as JSON: %w", stName, err)
		}

		for k, v := range secretData {
			result["secret_"+k] = v
		}
	}
	return result, nil
}

// startServices starts all service steps and returns the successfully started service steps.
func (w *Worker) startServices(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, serviceSteps []job.ServiceStep) []job.ServiceStep {
	var started []job.ServiceStep
	for _, ss := range serviceSteps {
		svc, ok := pp.Service(ss.Name)
		if !ok {
			w.logger.Error("service not found", "service", ss.Name)
			b.Status = build.Failed
			w.failBuild(ctx, m, *b, fmt.Errorf("service %q not found in pipeline", ss.Name))
			return started
		}

		ru, ok := pp.Runner(svc.Start.Runner)
		if !ok {
			w.logger.Error("runner not found for service start", "runner", svc.Start.Runner, "service", ss.Name)
			b.Status = build.Failed
			w.failBuild(ctx, m, *b, fmt.Errorf("runner %q not found for service %q start", svc.Start.Runner, ss.Name))
			return started
		}

		params := w.serviceParams(b, m, svc.Start.Params, ss.Params)

		rc := utils.RunnerCommand{
			Runner: svc.Start.Runner,
			Args:   svc.Start.Args,
			Params: params,
		}

		out, d, err := w.runRunner(ctx, ru, cwd, rc)
		if err != nil {
			b.Steps = append(b.Steps, build.Step{Type: "service", Name: ss.Name + ":start", Logs: out, Duration: d})
			b.Status = build.Failed
			w.failBuild(ctx, m, *b, nil)
			w.logger.Error("failed to start service", "service", ss.Name, "error", err)
			return started
		}

		b.Steps = append(b.Steps, build.Step{Type: "service", Name: ss.Name + ":start", Logs: out, Duration: d})
		if err := w.updateBuild(ctx, m, *b); err != nil {
			return started
		}
		started = append(started, ss)
	}
	return started
}

// buildMetadataParams returns the standard build metadata environment variables.
func buildMetadataParams(b *build.Build, m queue.Body) map[string]string {
	return map[string]string{
		"BUILD_ID":            fmt.Sprintf("%d", b.ID),
		"BUILD_JOB_NAME":     m.JobName,
		"BUILD_PIPELINE_NAME": m.PipelineName,
		"BUILD_TEAM_NAME":    m.TeamCanonical,
	}
}

// serviceParams builds the environment parameters for a service command,
// merging the command's own params with build info and per-job overrides.
func (w *Worker) serviceParams(b *build.Build, m queue.Body, cmdParams map[string]string, overrides map[string]string) map[string]string {
	params := make(map[string]string)
	for k, v := range cmdParams {
		params[k] = v
	}
	for k, v := range buildMetadataParams(b, m) {
		params[k] = v
	}
	for k, v := range overrides {
		params["param_"+k] = v
	}
	return params
}

// waitForServices runs ready_check for all started services that have one.
// Returns false if any ready_check times out.
func (w *Worker) waitForServices(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, startedServices []job.ServiceStep) bool {
	type readyResult struct {
		name string
		out  string
		d    time.Duration
		err  error
	}

	var wg sync.WaitGroup
	results := make(chan readyResult, len(startedServices))

	for _, ss := range startedServices {
		svc, ok := pp.Service(ss.Name)
		if !ok || svc.ReadyCheck == nil {
			continue
		}

		ru, ok := pp.Runner(svc.ReadyCheck.Runner)
		if !ok {
			continue
		}

		buildID := b.ID
		wg.Add(1)
		go func(svcName string, rc service.ReadyCheck, ru runner.Runner, overrides map[string]string) {
			defer wg.Done()

			interval := 1 * time.Second
			if rc.Interval != "" {
				if d, err := time.ParseDuration(rc.Interval); err == nil {
					interval = d
				}
			}
			timeout := 60 * time.Second
			if rc.Timeout != "" {
				if d, err := time.ParseDuration(rc.Timeout); err == nil {
					timeout = d
				}
			}

			params := make(map[string]string)
			for k, v := range rc.Params {
				params[k] = v
			}
			bm := buildMetadataParams(&build.Build{ID: buildID}, m)
			for k, v := range bm {
				params[k] = v
			}
			for k, v := range overrides {
				params["param_"+k] = v
			}

			runCmd := utils.RunnerCommand{
				Runner: rc.Runner,
				Args:   rc.Args,
				Params: params,
			}

			deadline := time.After(timeout)
			start := time.Now()
			var lastOut string
			var lastErr error
			for {
				select {
				case <-deadline:
					results <- readyResult{
						name: svcName,
						out:  lastOut + fmt.Sprintf("\nready_check timed out after %s", timeout),
						d:    time.Since(start),
						err:  fmt.Errorf("ready_check timed out after %s", timeout),
					}
					return
				case <-ctx.Done():
					results <- readyResult{
						name: svcName,
						out:  "context cancelled",
						d:    time.Since(start),
						err:  ctx.Err(),
					}
					return
				default:
				}

				lastOut, _, lastErr = w.runRunner(ctx, ru, cwd, runCmd)
				if lastErr == nil {
					results <- readyResult{
						name: svcName,
						out:  lastOut,
						d:    time.Since(start),
					}
					return
				}
				time.Sleep(interval)
			}
		}(ss.Name, *svc.ReadyCheck, ru, ss.Params)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	allReady := true
	for r := range results {
		if r.err != nil {
			b.Steps = append(b.Steps, build.Step{Type: "service", Name: r.name + ":ready", Logs: r.out, Duration: r.d})
			b.Status = build.Failed
			w.failBuild(ctx, m, *b, nil)
			w.logger.Error("service ready_check failed", "service", r.name, "error", r.err)
			allReady = false
		} else {
			b.Steps = append(b.Steps, build.Step{Type: "service", Name: r.name + ":ready", Logs: r.out, Duration: r.d})
			w.updateBuild(ctx, m, *b)
		}
	}

	return allReady
}

// stopServices stops all started services unconditionally.
// Uses a fresh context to ensure cleanup runs even if the parent context is cancelled.
func (w *Worker) stopServices(m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, startedServices []job.ServiceStep) {
	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, ss := range startedServices {
		svc, ok := pp.Service(ss.Name)
		if !ok {
			continue
		}

		ru, ok := pp.Runner(svc.Stop.Runner)
		if !ok {
			w.logger.Error("runner not found for service stop", "runner", svc.Stop.Runner, "service", ss.Name)
			continue
		}

		params := w.serviceParams(b, m, svc.Stop.Params, ss.Params)

		rc := utils.RunnerCommand{
			Runner: svc.Stop.Runner,
			Args:   svc.Stop.Args,
			Params: params,
		}

		out, d, err := w.runRunner(stopCtx, ru, cwd, rc)
		if err != nil {
			w.logger.Error("failed to stop service", "service", ss.Name, "error", err)
		}
		b.Steps = append(b.Steps, build.Step{Type: "service", Name: ss.Name + ":stop", Logs: out, Duration: d})
		w.updateBuild(stopCtx, m, *b)
	}
}

// resolveSecretVars resolves all secret-backed variable placeholders by fetching
// the actual secret values from the configured secret types. Variables sharing
// the same secret type and path are batched into a single fetch call.
func (w *Worker) resolveSecretVars(ctx context.Context, cwd string, pp *pipeline.Pipeline) (map[string]string, error) {
	if len(pp.SecretVars) == 0 {
		return nil, nil
	}

	// Group variables by (type, path) to avoid duplicate fetches.
	type fetchKey struct{ typ, path string }
	groups := make(map[fetchKey][]string) // fetchKey -> []varName
	for varName, sv := range pp.SecretVars {
		k := fetchKey{sv.Type, sv.Path}
		groups[k] = append(groups[k], varName)
	}

	resolved := make(map[string]string)
	for k, varNames := range groups {
		secrets, err := w.fetchSecrets(ctx, cwd, pp, map[string]string{k.typ: k.path})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve secrets from %q at %q: %w", k.typ, k.path, err)
		}
		for _, varName := range varNames {
			sv := pp.SecretVars[varName]
			placeholder := fmt.Sprintf("__pikoci_secret:%s:%s:%s__", sv.Type, sv.Path, sv.Key)
			val, ok := secrets["secret_"+sv.Key]
			if !ok {
				return nil, fmt.Errorf("secret for variable %q: key %q not found in response", varName, sv.Key)
			}
			resolved[placeholder] = val
		}
	}
	return resolved, nil
}

// replaceSecretPlaceholders replaces secret placeholder strings in a params map
// with the actual resolved secret values.
func replaceSecretPlaceholders(params map[string]string, resolved map[string]string) {
	for k := range params {
		for placeholder, val := range resolved {
			if strings.Contains(params[k], placeholder) {
				params[k] = strings.ReplaceAll(params[k], placeholder, val)
			}
		}
	}
}

// replaceSecretPlaceholdersInSlice replaces secret placeholder strings in a
// string slice with the actual resolved secret values.
func replaceSecretPlaceholdersInSlice(ss []string, resolved map[string]string) {
	for i := range ss {
		for placeholder, val := range resolved {
			if strings.Contains(ss[i], placeholder) {
				ss[i] = strings.ReplaceAll(ss[i], placeholder, val)
			}
		}
	}
}

func createKeyValuePairs(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s=%s ", key, value)
	}
	return b.String()
}
