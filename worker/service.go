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

	failed := w.runPlan(ctx, m, &b, cwd, pp, j)

	if !failed {
		b.Status = build.Succeeded
		if err := w.updateBuild(ctx, m, b); err != nil {
			return
		}
		w.triggerDownstreamJobs(ctx, m, &b, pp, j)
		w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "", j.OnSuccess, "on_success")
	} else {
		w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "", j.OnFailure, "on_failure")
	}
	w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "", j.Ensure, "ensure")
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

// runPlan runs all plan steps (get/task/put) in order.
// Returns true if the job failed during plan execution.
func (w *Worker) runPlan(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, j *job.Job) bool {
	for _, ps := range j.Plan {
		switch ps.Type {
		case job.StepTypeGet:
			if ps.Get == nil {
				continue
			}
			if w.runGetStep(ctx, m, b, cwd, pp, *ps.Get, ps) {
				return true
			}
		case job.StepTypeTask:
			if ps.Task == nil {
				continue
			}
			if w.runTaskStep(ctx, m, b, cwd, pp, *ps.Task, ps) {
				return true
			}
		case job.StepTypePut:
			if ps.Put == nil {
				continue
			}
			if w.runPutStep(ctx, m, b, cwd, pp, *ps.Put, ps) {
				return true
			}
		}
	}
	return false
}

// runGetStep runs a single get step (resource pull).
// Returns true if the step failed.
func (w *Worker) runGetStep(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, g job.GetStep, ps job.PlanStep) bool {
	r, ok := pp.Resource(utils.ResourceCanonical(g.Type, g.Name))
	if !ok {
		return false
	}
	rt, ok := pp.ResourceType(g.Type)
	if !ok {
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

	rc := utils.RunnerCommand{
		Runner: rt.Pull.Runner,
		Args:   rt.Pull.Args,
		Params: params,
	}
	out, d, err := w.runRunner(ctx, ru, cwd, rc)
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
func (w *Worker) runTaskStep(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, t job.TaskStep, ps job.PlanStep) bool {
	ru, ok := pp.Runner(t.Run.Runner)
	if !ok {
		return false
	}

	out, d, err := w.runRunner(ctx, ru, cwd, t.Run)
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
func (w *Worker) runPutStep(ctx context.Context, m queue.Body, b *build.Build, cwd string, pp *pipeline.Pipeline, p job.PutStep, ps job.PlanStep) bool {
	rCan := utils.ResourceCanonical(p.Type, p.Name)
	r, ok := pp.Resource(rCan)
	if !ok {
		return false
	}
	rt, ok := pp.ResourceType(p.Type)
	if !ok {
		return false
	}

	params := rt.Push.Params
	if params == nil {
		params = make(map[string]string)
	}
	// Add resource-level params
	for k, v := range r.Params.Params {
		if slices.Contains(rt.Params, k) {
			params["param_"+k] = v
		}
	}
	// Add put-step-level params with put_ prefix
	for k, v := range p.Params {
		params["put_"+k] = v
	}

	ru, ok := pp.Runner(rt.Push.Runner)
	if !ok {
		return false
	}

	rc := utils.RunnerCommand{
		Runner: rt.Push.Runner,
		Args:   rt.Push.Args,
		Params: params,
	}
	out, d, err := w.runRunner(ctx, ru, cwd, rc)
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
	params := rt.Pull.Params
	if params == nil {
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

	for k, v := range r.Params.Params {
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
func (w *Worker) runHooks(ctx context.Context, m queue.Body, b *build.Build, steps *[]build.Step, cwd string, pp *pipeline.Pipeline, stepName string, hooks []utils.RunnerCommand, hookType string) {
	for i, h := range hooks {
		ru, ok := pp.Runner(h.Runner)
		if !ok {
			continue
		}
		out, d, _ := w.runRunner(ctx, ru, cwd, h)

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

	params := rt.Check.Params

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
	for k, v := range r.Params.Params {
		if slices.Contains(rt.Params, k) {
			params["param_"+k] = v
		}
	}

	ru, ok := pp.Runner(rt.Check.Runner)
	if !ok {
		return
	}

	rc := utils.RunnerCommand{
		Runner: rt.Check.Runner,
		Args:   rt.Check.Args,
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
	for k, v := range envs {
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("%s=%s", k, v))
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

func createKeyValuePairs(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s=%s ", key, value)
	}
	return b.String()
}
