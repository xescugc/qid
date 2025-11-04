package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"gocloud.dev/pubsub"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
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
				Status: build.Started,
				Get:    make([]build.Step, 0, 0),
				Task:   make([]build.Step, 0, 0),
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
			// an improvemnt could be to store the version that was validated
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
				for _, r := range pp.Resources {
					for _, rt := range pp.ResourceTypes {
						if rt.Name == r.Type {
							cmd := exec.CommandContext(ctx, rt.Pull.Path, rt.Pull.Args...)
							cmd.Dir = cwd
							// Set the VERSION_HASH either from the Job or from the last
							// Version of the resource
							if g.Name == r.Name && g.Type == r.Type && m.VersionHash != "" {
								cmd.Env = append(cmd.Environ(), fmt.Sprintf("VERSION_HASH=%s", m.VersionHash))
							} else {
								rCan := strings.Join([]string{g.Type, g.Name}, ".")
								vers, err := w.qid.ListResourceVersions(ctx, m.PipelineName, rCan)
								if err != nil {
									ferr := fmt.Errorf("failed Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
									w.failBuild(ctx, m, b, ferr)
									level.Error(w.logger).Log("msg", ferr)
									goto END
								}
								if len(vers) == 0 {
									ferr := fmt.Errorf("failed Job %q from Pipeline %q no versions for the resource %q", m.PipelineName, m.JobName, r.Canonical)
									w.failBuild(ctx, m, b, ferr)
									level.Error(w.logger).Log("msg", ferr)
									goto END
								}
								slices.Reverse(vers)
								cmd.Env = append(cmd.Environ(), fmt.Sprintf("VERSION_HASH=%s", vers[0].Hash))
							}

							// Set the inputs as Env
							for k, v := range r.Inputs.Inputs {
								if slices.Contains(rt.Inputs, k) {
									cmd.Env = append(cmd.Environ(), fmt.Sprintf("%s=%s", strings.ToUpper(k), v))
								}
							}
							stdouterr, err := cmd.CombinedOutput()
							if err != nil {
								b.Get = append(b.Get, build.Step{
									Name: g.Name,
									Logs: string(stdouterr) + "\n" + err.Error(),
								})
								b.Status = build.Failed
								w.failBuild(ctx, m, b, nil)
								level.Error(w.logger).Log("msg", fmt.Errorf("failed to run command %q with args %q (%s): %w", rt.Pull.Path, rt.Pull.Args, stdouterr, err))
								for i, f := range g.OnFailure {
									cmd := exec.CommandContext(ctx, f.Path, f.Args...)
									cmd.Dir = cwd
									stdouterr, err := cmd.CombinedOutput()
									errs := ""
									if err != nil {
										errs = "\n" + err.Error()
									}
									name := fmt.Sprintf("%s:on_failure", g.Name)
									if len(g.OnFailure) > 1 {
										name = fmt.Sprintf("%s:%d:on_failure", g.Name, i)
									}
									b.Get = append(b.Get, build.Step{
										Name: name,
										Logs: string(stdouterr) + errs,
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
									cmd := exec.CommandContext(ctx, e.Path, e.Args...)
									cmd.Dir = cwd
									stdouterr, err := cmd.CombinedOutput()
									errs := ""
									if err != nil {
										errs = "\n" + err.Error()
									}
									name := fmt.Sprintf("%s:ensure", g.Name)
									if len(g.Ensure) > 1 {
										name = fmt.Sprintf("%s:%d:ensure", g.Name, i)
									}
									b.Get = append(b.Get, build.Step{
										Name: name,
										Logs: string(stdouterr) + errs,
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
								Name:        g.Name,
								VersionHash: m.VersionHash,
								Logs:        string(stdouterr),
							})
							err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
							if err != nil {
								ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
								w.failBuild(ctx, m, b, ferr)
								level.Error(w.logger).Log("msg", ferr)
								continue
							}
							for i, s := range g.OnSuccess {
								cmd := exec.CommandContext(ctx, s.Path, s.Args...)
								cmd.Dir = cwd
								stdouterr, err := cmd.CombinedOutput()
								errs := ""
								if err != nil {
									errs = "\n" + err.Error()
								}
								name := fmt.Sprintf("%s:on_success", g.Name)
								if len(g.OnSuccess) > 1 {
									name = fmt.Sprintf("%s:%d:on_success", g.Name, i)
								}
								b.Get = append(b.Get, build.Step{
									Name: name,
									Logs: string(stdouterr) + errs,
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
								cmd := exec.CommandContext(ctx, e.Path, e.Args...)
								cmd.Dir = cwd
								stdouterr, err := cmd.CombinedOutput()
								errs := ""
								if err != nil {
									errs = "\n" + err.Error()
								}
								name := fmt.Sprintf("%s:ensure", g.Name)
								if len(g.Ensure) > 1 {
									name = fmt.Sprintf("%s:%d:ensure", g.Name, i)
								}
								b.Get = append(b.Get, build.Step{
									Name: name,
									Logs: string(stdouterr) + errs,
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
					}
				}
			}
			for _, t := range j.Task {
				cmd := exec.CommandContext(ctx, t.Run.Path, t.Run.Args...)
				cmd.Dir = cwd
				stdouterr, err := cmd.CombinedOutput()
				if err != nil {
					b.Task = append(b.Task, build.Step{
						Name: t.Name,
						Logs: string(stdouterr) + "\n" + err.Error(),
					})
					b.Status = build.Failed
					w.failBuild(ctx, m, b, nil)
					level.Error(w.logger).Log("msg", fmt.Errorf("failed to run command %q with args %q: %w", t.Run.Path, t.Run.Args, err))
					for i, f := range t.OnFailure {
						cmd := exec.CommandContext(ctx, f.Path, f.Args...)
						cmd.Dir = cwd
						stdouterr, err := cmd.CombinedOutput()
						errs := ""
						if err != nil {
							errs = "\n" + err.Error()
						}
						name := fmt.Sprintf("%s:on_failure", t.Name)
						if len(t.OnFailure) > 1 {
							name = fmt.Sprintf("%s:%d:on_failure", t.Name, i)
						}
						b.Task = append(b.Task, build.Step{
							Name: name,
							Logs: string(stdouterr) + errs,
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
						cmd := exec.CommandContext(ctx, e.Path, e.Args...)
						cmd.Dir = cwd
						stdouterr, err := cmd.CombinedOutput()
						errs := ""
						if err != nil {
							errs = "\n" + err.Error()
						}
						name := fmt.Sprintf("%s:ensure", t.Name)
						if len(t.Ensure) > 1 {
							name = fmt.Sprintf("%s:%d:ensure", t.Name, i)
						}
						b.Task = append(b.Task, build.Step{
							Name: name,
							Logs: string(stdouterr) + errs,
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
					Name: t.Name,
					Logs: string(stdouterr),
				})
				err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
				if err != nil {
					ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.JobName, m.PipelineName, err)
					w.failBuild(ctx, m, b, ferr)
					level.Error(w.logger).Log("msg", ferr)
					continue
				}
				for i, s := range t.OnSuccess {
					cmd := exec.CommandContext(ctx, s.Path, s.Args...)
					cmd.Dir = cwd
					stdouterr, err := cmd.CombinedOutput()
					errs := ""
					if err != nil {
						errs = "\n" + err.Error()
					}
					name := fmt.Sprintf("%s:on_success", t.Name)
					if len(t.OnSuccess) > 1 {
						name = fmt.Sprintf("%s:%d:on_success", t.Name, i)
					}
					b.Task = append(b.Task, build.Step{
						Name: name,
						Logs: string(stdouterr) + errs,
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
					cmd := exec.CommandContext(ctx, e.Path, e.Args...)
					cmd.Dir = cwd
					stdouterr, err := cmd.CombinedOutput()
					errs := ""
					if err != nil {
						errs = "\n" + err.Error()
					}
					name := fmt.Sprintf("%s:ensure", t.Name)
					if len(t.Ensure) > 1 {
						name = fmt.Sprintf("%s:%d:ensure", t.Name, i)
					}
					b.Task = append(b.Task, build.Step{
						Name: name,
						Logs: string(stdouterr) + errs,
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
								VersionHash:       m.VersionHash,
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
				cmd := exec.CommandContext(ctx, s.Path, s.Args...)
				cmd.Dir = cwd
				stdouterr, err := cmd.CombinedOutput()
				errs := ""
				if err != nil {
					errs = "\n" + err.Error()
				}
				name := fmt.Sprintf("on_success")
				if len(j.OnSuccess) > 1 {
					name = fmt.Sprintf("%d:on_success", i)
				}
				b.Job = append(b.Job, build.Step{
					Name: name,
					Logs: string(stdouterr) + errs,
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
					cmd := exec.CommandContext(ctx, f.Path, f.Args...)
					cmd.Dir = cwd
					stdouterr, err := cmd.CombinedOutput()
					errs := ""
					if err != nil {
						errs = "\n" + err.Error()
					}
					name := fmt.Sprintf("on_failure")
					if len(j.OnFailure) > 1 {
						name = fmt.Sprintf("%d:on_failure", i)
					}
					b.Job = append(b.Job, build.Step{
						Name: name,
						Logs: string(stdouterr) + errs,
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
				cmd := exec.CommandContext(ctx, e.Path, e.Args...)
				cmd.Dir = cwd
				stdouterr, err := cmd.CombinedOutput()
				errs := ""
				if err != nil {
					errs = "\n" + err.Error()
				}
				name := fmt.Sprintf("ensure")
				if len(j.Ensure) > 1 {
					name = fmt.Sprintf("%d:ensure", i)
				}
				b.Job = append(b.Job, build.Step{
					Name: name,
					Logs: string(stdouterr) + errs,
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
			for _, r := range pp.Resources {
				if r.Canonical != m.ResourceCanonical {
					continue
				}
				for _, rt := range pp.ResourceTypes {
					if rt.Name == r.Type {
						cmd := exec.CommandContext(ctx, rt.Check.Path, rt.Check.Args...)
						cmd.Dir = cwd

						vers, err := w.qid.ListResourceVersions(ctx, m.PipelineName, r.Canonical)
						if err != nil {
							ferr := fmt.Errorf("failed to list resource versions: %w", err)
							level.Error(w.logger).Log("msg", ferr)
							goto END
						}
						if len(vers) != 0 {
							cmd.Env = append(cmd.Environ(), fmt.Sprintf("LAST_VERSION_HASH=%s", vers[0].Hash))
						}
						for k, v := range r.Inputs.Inputs {
							if slices.Contains(rt.Inputs, k) {
								cmd.Env = append(cmd.Environ(), fmt.Sprintf("%s=%s", strings.ToUpper(k), v))
							}
						}
						stdouterr, err := cmd.CombinedOutput()
						if err != nil {
							r.Logs = string(stdouterr) + "\n" + err.Error()
							nerr := w.qid.UpdatePipelineResource(ctx, m.PipelineName, r.Canonical, r)
							if nerr != nil {
								level.Error(w.logger).Log("msg", fmt.Errorf("failed update Resource %q.%q from Pipeline %q: %w", r.Type, r.Name, m.PipelineName, nerr))
							}
							level.Error(w.logger).Log("msg", fmt.Errorf("failed to run command %q with args %q (%s): %w", rt.Check.Path, rt.Check.Args, stdouterr, err))
							goto END
						}
						if r.Logs != "" {
							r.Logs = ""
							err = w.qid.UpdatePipelineResource(ctx, m.PipelineName, r.Canonical, r)
							if err != nil {
								level.Error(w.logger).Log("msg", fmt.Errorf("failed update Resource %q.%q from Pipeline %q: %w", r.Type, r.Name, m.PipelineName, err))
								goto END
							}
						}
						hashs := strings.Split(string(stdouterr), "\n")
						if len(hashs) == 0 || string(stdouterr) == "" {
							// Nothing new so we can skip
							goto END
						}
						for _, h := range hashs {
							if h == "" {
								continue
							}
							err = w.qid.CreateResourceVersion(ctx, m.PipelineName, r.Canonical, resource.Version{
								Hash: h,
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
											VersionHash:       h,
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
