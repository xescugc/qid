package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"gocloud.dev/pubsub"

	"github.com/go-kit/log"
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
			w.logger.Log("error", fmt.Errorf("failed Unmarshal Message body: %w", err))
			continue
		}

		cwd, err := os.MkdirTemp("", "qid")
		if err != nil {
			return fmt.Errorf("failed to creat Temp Dir: %w", err)
		}

		//err = os.Mkdir(cwd, 0755)
		//if err != nil {
		//return fmt.Errorf("failed to create dir %q: %w", cwd, err)
		//}

		pp, err := w.qid.GetPipeline(ctx, m.PipelineName)
		if err != nil {
			w.logger.Log("error", fmt.Errorf("failed GetPipeline: %w", err))
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
				w.logger.Log("error", fmt.Errorf("failed create Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err))
				continue
			}
			// We keep 'b' as a reference
			b.ID = nb.ID
			j, err := w.qid.GetPipelineJob(ctx, m.PipelineName, m.JobName)
			if err != nil {
				ferr := fmt.Errorf("failed Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
				w.failBuild(ctx, m, b, ferr)
				w.logger.Log("error", ferr)
				continue
			}

			for _, g := range j.Get {
				for _, r := range pp.Resources {
					for _, rt := range pp.ResourceTypes {
						if rt.Name == r.Type {
							cmd := exec.CommandContext(ctx, rt.Pull.Path, rt.Pull.Args...)
							cmd.Dir = cwd
							if g.Name == m.ResourceName && g.Type == m.ResourceType && m.VersionHash != "" {
								cmd.Env = append(cmd.Environ(), fmt.Sprintf("VERSION_HASH=%s", m.VersionHash))
							} else {
								rCan := strings.Join([]string{g.Type, g.Name}, ".")
								vers, err := w.qid.ListResourceVersions(ctx, m.PipelineName, rCan)
								if err != nil {
									ferr := fmt.Errorf("failed Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
									w.failBuild(ctx, m, b, ferr)
									w.logger.Log("error", ferr)
									goto END
								}
								slices.Reverse(vers)
								cmd.Env = append(cmd.Environ(), fmt.Sprintf("VERSION_HASH=%s", vers[0].Hash))
							}

							for k, v := range r.Inputs {
								if slices.Contains(rt.Inputs, k) {
									cmd.Env = append(cmd.Environ(), fmt.Sprintf("%s=%s", strings.ToUpper(k), v))
								}
							}
							stdouterr, err := cmd.CombinedOutput()
							if err != nil {
								b.Get = append(b.Get, build.Step{
									Name: g.Name,
									Logs: err.Error(),
								})
								w.failBuild(ctx, m, b, nil)
								w.logger.Log("error", fmt.Errorf("failed to run command %q with args %q (%s): %w", rt.Pull.Path, rt.Pull.Args, stdouterr, err))
								goto END
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
								w.logger.Log("error", ferr)
								continue
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
						Logs: err.Error(),
					})
					w.failBuild(ctx, m, b, nil)
					w.logger.Log("error", fmt.Errorf("failed to run command %q with args %q: %w", t.Run.Path, t.Run.Args, err))
					goto END
				}
				b.Task = append(b.Task, build.Step{
					Name: t.Name,
					Logs: string(stdouterr),
				})
				err = w.qid.UpdateJobBuild(ctx, m.PipelineName, m.JobName, b.ID, b)
				if err != nil {
					ferr := fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err)
					w.failBuild(ctx, m, b, ferr)
					w.logger.Log("error", ferr)
					continue
				}
				spew.Dump(string(stdouterr))
				for _, nj := range pp.Jobs {
					for _, g := range nj.Get {
						if slices.Contains(g.Passed, j.Name) && g.Trigger {
							qb := queue.Body{
								PipelineName: pp.Name,
								JobName:      nj.Name,
								ResourceName: g.Name,
								VersionHash:  m.VersionHash,
							}
							mb, err := json.Marshal(qb)
							if err != nil {
								ferr := fmt.Errorf("failed to run marshal body: %w", err)
								w.failBuild(ctx, m, b, ferr)
								w.logger.Log("error", ferr)
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
				w.logger.Log("error", fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err))
				continue
			}
		} else if m.PipelineName != "" && m.ResourceName != "" {
			// This is for the periodic resource checks
			for _, r := range pp.Resources {
				if r.Name == m.ResourceName {
					for _, rt := range pp.ResourceTypes {
						if rt.Name == r.Type {
							cmd := exec.CommandContext(ctx, rt.Check.Path, rt.Check.Args...)
							cmd.Dir = cwd

							vers, err := w.qid.ListResourceVersions(ctx, m.PipelineName, r.Canonical)
							if err != nil {
								ferr := fmt.Errorf("failed to list resource versions: %w", err)
								w.logger.Log("error", ferr)
								goto END
							}
							if len(vers) != 0 {
								cmd.Env = append(cmd.Environ(), fmt.Sprintf("LAST_VERSION_HASH=%s", vers[0].Hash))
							}
							for k, v := range r.Inputs {
								if slices.Contains(rt.Inputs, k) {
									cmd.Env = append(cmd.Environ(), fmt.Sprintf("%s=%s", strings.ToUpper(k), v))
								}
							}
							stdouterr, err := cmd.CombinedOutput()
							if err != nil {
								r.Logs = string(stdouterr)
								err = w.qid.UpdatePipelineResource(ctx, m.PipelineName, r.Canonical, r)
								if err != nil {
									w.logger.Log("error", fmt.Errorf("failed update Resource %q.%q from Pipeline %q: %w", r.Type, r.Name, m.PipelineName, err))
								}
								w.logger.Log("error", fmt.Errorf("failed to run command %q with args %q (%s): %w", rt.Check.Path, rt.Check.Args, stdouterr, err))
								goto END
							}
							if r.Logs != "" {
								r.Logs = ""
								err = w.qid.UpdatePipelineResource(ctx, m.PipelineName, r.Canonical, r)
								if err != nil {
									w.logger.Log("error", fmt.Errorf("failed update Resource %q.%q from Pipeline %q: %w", r.Type, r.Name, m.PipelineName, err))
									goto END
								}
							}
							hashs := strings.Split(string(stdouterr), "\n")
							if len(hashs) == 0 || string(stdouterr) == "" {
								// Nothing new so we can skip
								goto END
							}
							for _, h := range hashs {
								err = w.qid.CreateResourceVersion(ctx, m.PipelineName, r.Canonical, resource.Version{
									Hash: h,
								})
								if err != nil {
									w.logger.Log("error", fmt.Errorf("failed to create Resource Version body: %w", err))
									goto END
								}
								for _, j := range pp.Jobs {
									for _, g := range j.Get {
										// If Passed is not 0 it means is waiting for another job
										// and this trigger is only for resources
										if g.Name == r.Name && g.Type == r.Type && g.Trigger && len(g.Passed) == 0 {
											b := queue.Body{
												PipelineName: pp.Name,
												JobName:      j.Name,
												ResourceName: r.Name,
												ResourceType: r.Type,
												VersionHash:  h,
											}
											mb, err := json.Marshal(b)
											if err != nil {
												w.logger.Log("error", fmt.Errorf("failed to run marshal body: %w", err))
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
		w.logger.Log("error", fmt.Errorf("failed update Build for Job %q from Pipeline %q: %w", m.PipelineName, m.JobName, err))
	}
}
