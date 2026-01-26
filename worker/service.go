package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/runner"
	"github.com/xescugc/qid/qid/utils"
	"gocloud.dev/pubsub"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/kballard/go-shellquote"
)

type Service interface {
	Run(ctx context.Context, q, t string) error
}

type Worker struct {
	topic        queue.Topic
	qid          qid.Service
	subscription queue.Subscription

	logger log.Logger
}

func New(s qid.Service, t queue.Topic, ss queue.Subscription, l log.Logger) *Worker {
	return &Worker{
		qid:          s,
		topic:        t,
		subscription: ss,
		logger:       l,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	// Loop on received messages.
	for {
		msg, err := w.subscription.Receive(ctx)
		if err != nil {
			// Errors from Receive indicate that Receive will no longer succeed.
			return fmt.Errorf("Failed to Receiving message: %w", err)
		}
		var m queue.Body
		err = json.Unmarshal(msg.Body, &m)
		if err != nil {
			level.Error(w.logger).Log("msg", fmt.Errorf("failed Unmarshal Message body: %w", err))
			continue
		}

		uuiddir, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to creat UUID %w", err)
		}
		// We append a file "qid" just so the CacheFile creates the full dir,
		// afterward we just get the Dir of the cwd
		cwd, err := xdg.CacheFile(filepath.Join("qid", uuiddir.String(), "qid"))
		if err != nil {
			return fmt.Errorf("failed to creat Temp Dir: %w", err)
		}
		cwd = filepath.Dir(cwd)

		pp, err := w.qid.GetPipeline(ctx, m.PipelineName)
		if err != nil {
			level.Error(w.logger).Log("msg", fmt.Errorf("failed GetPipeline: %w", err))
			continue
		}
		if m.PipelineName != "" && m.JobName != "" {
			b := build.Build{
				Status:    build.Started,
				Get:       make([]build.Step, 0, 0),
				Task:      make([]build.Step, 0, 0),
				StartedAt: time.Now(),
			}
			nb, err := w.qid.CreateJobBuild(ctx, m.PipelineName, m.JobName, b)
			if err != nil {
				level.Error(w.logger).Log("msg", fmt.Errorf("failed create Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err))
				continue
			}
			// We keep 'b' as a reference
			b.ID = nb.ID
			j, err := w.qid.GetPipelineJob(ctx, m.PipelineName, m.JobName)
			if err != nil {
				ferr := fmt.Errorf("failed Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
				w.failBuild(ctx, m, b, ferr)
				level.Error(w.logger).Log("msg", ferr)
				continue
			}

			// First we need to check that all the 'Get'
			// are Succeeded
			passed := true
			// NOTE: As this could happen concurrently that a resource changes
			// an improvement could be to store the version that was validated
			// of the resource_type
			for _, g := range j.Get {
				if !passed {
					break
				}
				for _, p := range g.Passed {
					if !passed {
						break
					}
					builds, err := w.qid.ListJobBuilds(ctx, m.PipelineName, p)
					if err != nil {
						ferr := fmt.Errorf("failed Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
						w.failBuild(ctx, m, b, ferr)
						level.Error(w.logger).Log("msg", ferr)
						goto END
					}
					if len(builds) > 0 {
						if builds[0].Status != build.Succeeded {
							passed = false
							level.Info(w.logger).Log("msg", fmt.Sprintf("The Job %q from Pipeline %q will not run as the Job %q that is 'passed' is not 'Succeeded'", m.JobName, m.PipelineName, p))
							w.deleteBuild(ctx, m, b)
							break
						}
					} else {
						passed = false
						level.Info(w.logger).Log("msg", fmt.Sprintf("The Job %q from Pipeline %q will not run as the Job %q that is 'passed' has no builds", m.JobName, m.PipelineName, p))
						w.deleteBuild(ctx, m, b)
						break
					}
				}
			}
			if !passed {
				goto END
			}

			// tacks if the job failed
			var failed bool
			for _, g := range j.Get {
				r, ok := pp.Resource(utils.ResourceCanonical(g.Type, g.Name))
				if !ok {
					continue
				}
				rt, ok := pp.ResourceType(g.Type)
				if !ok {
					continue
				}
				params := rt.Pull.Params
				if params == nil {
					params = make(map[string]string)
				}
				// Set the VERSION_HASH either from the Job or from the last
				// Version of the resource
				dbvers, err := w.qid.ListResourceVersions(ctx, m.PipelineName, r.Canonical)
				if err != nil {
					ferr := fmt.Errorf("failed Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
					w.failBuild(ctx, m, b, ferr)
					level.Error(w.logger).Log("msg", ferr)
					goto END
				}
				if m.VersionID != 0 {
					var found bool
					for _, ver := range dbvers {
						if ver.ID == m.VersionID {
							for k, v := range ver.Version {
								found = true
								params["version_"+k] = fmt.Sprintf("%s", v)
							}
							break
						}
					}
					if !found {
						ferr := fmt.Errorf("failed Job %q from Pipeline %q no version found for resource %q", m.PipelineName, m.JobName, r.Canonical)
						w.failBuild(ctx, m, b, ferr)
						level.Error(w.logger).Log("msg", ferr)
						goto END
					}
				} else {
					if len(dbvers) == 0 {
						ferr := fmt.Errorf("failed Job %q from Pipeline %q no versions for the resource %q", m.PipelineName, m.JobName, r.Canonical)
						w.failBuild(ctx, m, b, ferr)
						level.Error(w.logger).Log("msg", ferr)
						goto END
					}
					slices.Reverse(dbvers)
					for k, v := range dbvers[0].Version {
						params["version_"+k] = fmt.Sprintf("%s", v)
					}
				}

				// Set the params as Env
				for k, v := range r.Params.Params {
					if slices.Contains(rt.Params, k) {
						params["param_"+k] = v
					}
				}
				ru, ok := pp.Runner(rt.Pull.Runner)
				if !ok {
					continue
				}
				out, d, err := w.runRunner(ctx, ru, cwd, params)
				if err != nil {
					b.Get = append(b.Get, build.Step{
						Name:     g.Name,
						Logs:     out,
						Duration: d,
					})
					b.Status = build.Failed
					w.failBuild(ctx, m, b, nil)
					level.Error(w.logger).Log("msg", fmt.Errorf("failed to run command: %w", err))
					for i, f := range g.OnFailure {
						ru, ok := pp.Runner(f.Runner)
						if !ok {
							continue
						}
						out, d, _ := w.runRunner(ctx, ru, cwd, f.Params)
						name := fmt.Sprintf("%s:on_failure", g.Name)
						if len(g.OnFailure) > 1 {
							name = fmt.Sprintf("%s:%d:on_failure", g.Name, i)
						}
						b.Get = append(b.Get, build.Step{
							Name:     name,
							Logs:     out,
							Duration: d,
						})
						err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
						if err != nil {
							ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
							w.failBuild(ctx, m, b, ferr)
							level.Error(w.logger).Log("msg", ferr)
							continue
						}
					}
					for i, e := range g.Ensure {
						ru, ok := pp.Runner(e.Runner)
						if !ok {
							continue
						}
						out, d, _ := w.runRunner(ctx, ru, cwd, e.Params)
						name := fmt.Sprintf("%s:ensure", g.Name)
						if len(g.Ensure) > 1 {
							name = fmt.Sprintf("%s:%d:ensure", g.Name, i)
						}
						b.Get = append(b.Get, build.Step{
							Name:     name,
							Logs:     out,
							Duration: d,
						})
						err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
						if err != nil {
							ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
							w.failBuild(ctx, m, b, ferr)
							level.Error(w.logger).Log("msg", ferr)
							continue
						}
					}
					failed = true
					goto FAILED_JOB
				}
				b.Get = append(b.Get, build.Step{
					Name:      g.Name,
					VersionID: m.VersionID,
					Logs:      out,
					Duration:  d,
				})
				err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
				if err != nil {
					ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
					w.failBuild(ctx, m, b, ferr)
					level.Error(w.logger).Log("msg", ferr)
					continue
				}
				for i, s := range g.OnSuccess {
					ru, ok := pp.Runner(s.Runner)
					if !ok {
						continue
					}
					out, d, _ := w.runRunner(ctx, ru, cwd, s.Params)
					name := fmt.Sprintf("%s:on_success", g.Name)
					if len(g.OnSuccess) > 1 {
						name = fmt.Sprintf("%s:%d:on_success", g.Name, i)
					}
					b.Get = append(b.Get, build.Step{
						Name:     name,
						Logs:     out,
						Duration: d,
					})
					err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
					if err != nil {
						ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
						w.failBuild(ctx, m, b, ferr)
						level.Error(w.logger).Log("msg", ferr)
						continue
					}
				}
				for i, e := range g.Ensure {
					ru, ok := pp.Runner(e.Runner)
					if !ok {
						continue
					}
					out, d, _ := w.runRunner(ctx, ru, cwd, e.Params)
					name := fmt.Sprintf("%s:ensure", g.Name)
					if len(g.Ensure) > 1 {
						name = fmt.Sprintf("%s:%d:ensure", g.Name, i)
					}
					b.Get = append(b.Get, build.Step{
						Name:     name,
						Logs:     out,
						Duration: d,
					})
					err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
					if err != nil {
						ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
						w.failBuild(ctx, m, b, ferr)
						level.Error(w.logger).Log("msg", ferr)
						continue
					}
				}
			}
			for _, t := range j.Task {
				ru, ok := pp.Runner(t.Run.Runner)
				if !ok {
					continue
				}
				out, d, err := w.runRunner(ctx, ru, cwd, t.Run.Params)
				if err != nil {
					b.Task = append(b.Task, build.Step{
						Name:     t.Name,
						Logs:     out,
						Duration: d,
					})
					b.Status = build.Failed
					w.failBuild(ctx, m, b, nil)
					for i, f := range t.OnFailure {
						ru, ok := pp.Runner(f.Runner)
						if !ok {
							continue
						}
						out, d, _ := w.runRunner(ctx, ru, cwd, f.Params)
						name := fmt.Sprintf("%s:on_failure", t.Name)
						if len(t.OnFailure) > 1 {
							name = fmt.Sprintf("%s:%d:on_failure", t.Name, i)
						}
						b.Task = append(b.Task, build.Step{
							Name:     name,
							Logs:     out,
							Duration: d,
						})
						err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
						if err != nil {
							ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
							w.failBuild(ctx, m, b, ferr)
							level.Error(w.logger).Log("msg", ferr)
							continue
						}
					}
					for i, e := range t.Ensure {
						ru, ok := pp.Runner(e.Runner)
						if !ok {
							continue
						}
						out, d, _ := w.runRunner(ctx, ru, cwd, e.Params)
						name := fmt.Sprintf("%s:ensure", t.Name)
						if len(t.Ensure) > 1 {
							name = fmt.Sprintf("%s:%d:ensure", t.Name, i)
						}
						b.Task = append(b.Task, build.Step{
							Name:     name,
							Logs:     out,
							Duration: d,
						})
						err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
						if err != nil {
							ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
							w.failBuild(ctx, m, b, ferr)
							level.Error(w.logger).Log("msg", ferr)
							continue
						}
					}
					failed = true
					goto FAILED_JOB
				}
				b.Task = append(b.Task, build.Step{
					Name:     t.Name,
					Logs:     out,
					Duration: d,
				})
				err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
				if err != nil {
					ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.JobName, m.PipelineName, err)
					w.failBuild(ctx, m, b, ferr)
					level.Error(w.logger).Log("msg", ferr)
					continue
				}
				for i, s := range t.OnSuccess {
					ru, ok := pp.Runner(s.Runner)
					if !ok {
						continue
					}
					out, d, _ := w.runRunner(ctx, ru, cwd, s.Params)
					name := fmt.Sprintf("%s:on_success", t.Name)
					if len(t.OnSuccess) > 1 {
						name = fmt.Sprintf("%s:%d:on_success", t.Name, i)
					}
					b.Task = append(b.Task, build.Step{
						Name:     name,
						Logs:     out,
						Duration: d,
					})
					err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
					if err != nil {
						ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
						w.failBuild(ctx, m, b, ferr)
						level.Error(w.logger).Log("msg", ferr)
						continue
					}
				}
				for i, e := range t.Ensure {
					ru, ok := pp.Runner(e.Runner)
					if !ok {
						continue
					}
					out, d, _ := w.runRunner(ctx, ru, cwd, e.Params)
					name := fmt.Sprintf("%s:ensure", t.Name)
					if len(t.Ensure) > 1 {
						name = fmt.Sprintf("%s:%d:ensure", t.Name, i)
					}
					b.Task = append(b.Task, build.Step{
						Name:     name,
						Logs:     out,
						Duration: d,
					})
					err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
					if err != nil {
						ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
						w.failBuild(ctx, m, b, ferr)
						level.Error(w.logger).Log("msg", ferr)
						continue
					}
				}
				for _, nj := range pp.Jobs {
					for _, g := range nj.Get {
						if slices.Contains(g.Passed, j.Name) && g.Trigger {
							qb := queue.Body{
								PipelineName:      pp.Name,
								JobName:           nj.Name,
								ResourceCanonical: g.ResourceCanonical(),
								VersionID:         m.VersionID,
							}
							mb, err := json.Marshal(qb)
							if err != nil {
								ferr := fmt.Errorf("failed to run marshal body: %w", err)
								w.failBuild(ctx, m, b, ferr)
								level.Error(w.logger).Log("msg", ferr)
								goto END
							}
							w.topic.Send(ctx, &pubsub.Message{
								Body: mb,
							})
						}
					}
				}
			}
			b.Status = build.Succeeded
			err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
			if err != nil {
				level.Error(w.logger).Log("msg", fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.JobName, m.PipelineName, err))
				continue
			}
			for i, s := range j.OnSuccess {
				ru, ok := pp.Runner(s.Runner)
				if !ok {
					continue
				}
				out, d, _ := w.runRunner(ctx, ru, cwd, s.Params)
				name := fmt.Sprintf("on_success")
				if len(j.OnSuccess) > 1 {
					name = fmt.Sprintf("%d:on_success", i)
				}
				b.Job = append(b.Job, build.Step{
					Name:     name,
					Logs:     out,
					Duration: d,
				})
				err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
				if err != nil {
					ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
					w.failBuild(ctx, m, b, ferr)
					level.Error(w.logger).Log("msg", ferr)
					continue
				}
			}
			// This is because the normal flow will not go here

		FAILED_JOB:
			if failed {
				for i, f := range j.OnFailure {
					ru, ok := pp.Runner(f.Runner)
					if !ok {
						continue
					}
					out, d, _ := w.runRunner(ctx, ru, cwd, f.Params)
					name := fmt.Sprintf("on_failure")
					if len(j.OnFailure) > 1 {
						name = fmt.Sprintf("%d:on_failure", i)
					}
					b.Job = append(b.Job, build.Step{
						Name:     name,
						Logs:     out,
						Duration: d,
					})
					err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
					if err != nil {
						ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
						w.failBuild(ctx, m, b, ferr)
						level.Error(w.logger).Log("msg", ferr)
						continue
					}
				}
			}
			for i, e := range j.Ensure {
				ru, ok := pp.Runner(e.Runner)
				if !ok {
					continue
				}
				out, d, _ := w.runRunner(ctx, ru, cwd, e.Params)
				name := fmt.Sprintf("ensure")
				if len(j.Ensure) > 1 {
					name = fmt.Sprintf("%d:ensure", i)
				}
				b.Job = append(b.Job, build.Step{
					Name:     name,
					Logs:     out,
					Duration: d,
				})
				err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
				if err != nil {
					ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
					w.failBuild(ctx, m, b, ferr)
					level.Error(w.logger).Log("msg", ferr)
					continue
				}
			}
			if failed {
				goto END
			}
		} else if m.PipelineName != "" && m.ResourceCanonical != "" {
			// This is for the periodic resource checks
			r, ok := pp.Resource(m.ResourceCanonical)
			if !ok {
				continue
			}
			rt, ok := pp.ResourceType(r.Type)
			if !ok {
				continue
			}

			params := rt.Check.Params

			dbvers, err := w.qid.ListResourceVersions(ctx, m.PipelineName, r.Canonical)
			if err != nil {
				ferr := fmt.Errorf("failed to list resource versions: %w", err)
				level.Error(w.logger).Log("msg", ferr)
				goto END
			}
			if len(dbvers) != 0 {
				// This version is already stored flatten and prefixed with version_
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
				continue
			}
			out, _, err := w.runRunner(ctx, ru, cwd, params)
			if err != nil {
				r.Logs = out
				nerr := w.qid.UpdatePipelineResource(ctx, m.PipelineName, r.Canonical, r)
				if nerr != nil {
					level.Error(w.logger).Log("msg", fmt.Errorf("failed update Resource %q.%q from Pipeline %q: %w", r.Canonical, m.PipelineName, nerr))
				}
				level.Error(w.logger).Log("msg", fmt.Errorf("failed to run command: %w", err))
				goto END
			}
			if r.Logs != "" {
				r.Logs = ""
				err = w.qid.UpdatePipelineResource(ctx, m.PipelineName, r.Canonical, r)
				if err != nil {
					level.Error(w.logger).Log("msg", fmt.Errorf("failed update Resource %q from Pipeline %q: %w", r.Canonical, m.PipelineName, err))
					goto END
				}
			}
			sout := strings.Split(strings.Trim(out, "\n"), "\n")
			rawVers := sout[len(sout)-1]
			if rawVers == "" {
				// Nothing new so we can skip
				goto END
			}
			vers := make([]map[string]interface{}, 0)
			err = json.Unmarshal([]byte(rawVers), &vers)
			if err != nil {
				ferr := fmt.Errorf("failed to Unmarshal versions(%s): %w", rawVers, err)
				level.Error(w.logger).Log("msg", ferr)
				r.Logs = ferr.Error()
				nerr := w.qid.UpdatePipelineResource(ctx, m.PipelineName, r.Canonical, r)
				if nerr != nil {
					level.Error(w.logger).Log("msg", fmt.Errorf("failed update Resource %q.%q from Pipeline %q: %w", r.Canonical, m.PipelineName, nerr))
				}
				level.Error(w.logger).Log("msg", fmt.Errorf("failed to run command: %w", err))
				goto END
			}
			for _, v := range vers {
				cv, err := w.qid.CreateResourceVersion(ctx, m.PipelineName, r.Canonical, resource.Version{
					Version: v,
				})
				if err != nil {
					level.Error(w.logger).Log("msg", fmt.Errorf("failed to create Resource Version body: %w", err))
					goto END
				}
				for _, j := range pp.Jobs {
					for _, g := range j.Get {
						// If Passed is not 0 it means is waiting for another job
						// and this trigger is only for resources
						if g.Name == r.Name && g.Type == r.Type && g.Trigger && len(g.Passed) == 0 {
							b := queue.Body{
								PipelineName:      pp.Name,
								JobName:           j.Name,
								ResourceCanonical: r.Canonical,
								VersionID:         cv.ID,
							}
							mb, err := json.Marshal(b)
							if err != nil {
								level.Error(w.logger).Log("msg", fmt.Errorf("failed to run marshal body: %w", err))
								goto END
							}
							w.topic.Send(ctx, &pubsub.Message{
								Body: mb,
							})
						}
					}
				}
			}
		}
	END:
		// Messages must always be acknowledged with Ack.
		//defer func() { msg.Ack() }()
		msg.Ack()
		os.RemoveAll(cwd)
	}
	return nil
}

// runRunner runs the Runner and retuns the aggregated output and the error (already included on the aggregated output)
// func (w *Worker) runRunner(ctx context.Context, ru runner.Runner, cwd string, params map[string]string) (string, error) {
func (w *Worker) runRunner(ctx context.Context, ru runner.Runner, cwd string, params map[string]string) (string, time.Duration, error) {
	var (
		cmd *exec.Cmd
		out string
	)
	envs := map[string]string{
		"WORKDIR": cwd,
	}
	for k, v := range params {
		envs[k] = v
	}
	envFn := func(p string) string {
		if v, ok := envs[p]; ok {
			return v
		}
		return os.Getenv(p)
	}

	args := make([]string, 0, 0)
	for _, a := range ru.Run.Args {
		ea := os.Expand(a, envFn)
		if ea == "" {
			continue
		}
		//ea = os.Expand(ea, envFn)
		sea, err := shellquote.Split(ea)
		if err != nil {
			out += "\n" + err.Error()
			return out, time.Duration(1), err
		}
		args = append(args, sea...)
	}
	cmd = exec.CommandContext(ctx, os.Expand(ru.Run.Path, envFn), args...)
	cmd.Dir = cwd
	for k, v := range envs {
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	level.Debug(w.logger).Log("msg", "running command", cmd.String(), "envs", createKeyValuePairs(envs))
	b := time.Now()
	stdouterr, err := cmd.CombinedOutput()
	duration := time.Now().Sub(b)
	out += string(stdouterr)
	if err != nil {
		out += "\n" + err.Error()
	}
	level.Debug(w.logger).Log("msg", "finished running command", "out", out)

	return out, duration, err
}

func (w *Worker) failBuild(ctx context.Context, m queue.Body, b build.Build, err error) {
	b.Status = build.Failed
	if err != nil {
		b.Error = err.Error()
	}
	err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
	if err != nil {
		level.Error(w.logger).Log("msg", fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err))
	}
}

func (w *Worker) deleteBuild(ctx context.Context, m queue.Body, b build.Build) {
	err := w.qid.DeleteJobBuild(ctx, m.PipelineName, m.JobName, b.ID)
	if err != nil {
		level.Error(w.logger).Log("msg", fmt.Errorf("failed delete Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err))
	}
}

func createKeyValuePairs(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s=%s ", key, value)
	}
	return b.String()
}
