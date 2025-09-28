package worker

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/davecgh/go-spew/spew"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/queue"
)

type Service interface {
	Run(ctx context.Context, q, t string) error
}

type Worker struct {
	qid          qid.Service
	subscription queue.Subscription
}

func New(s qid.Service, ss queue.Subscription) *Worker {
	return &Worker{
		qid:          s,
		subscription: ss,
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
		q, t := msg.Metadata["pipeline_name"], msg.Metadata["job_name"]

		j, err := w.qid.GetPipelineJob(ctx, q, t)
		if err != nil {
			return fmt.Errorf("failed Job %q from Pipeline %q: %w", t, q, err)
		}
		for _, p := range j.Plan {
			if p.Task != "" {
				cmd := exec.CommandContext(ctx, p.TConfig.Run.Path, p.TConfig.Run.Args...)
				stdouterr, err := cmd.CombinedOutput()
				if err != nil {
					//if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to run command %q with args %q: %w", p.TConfig.Run.Path, p.TConfig.Run.Args, err)
				}
				spew.Dump(stdouterr)
				pp, err := w.qid.GetPipeline(ctx, q)
				if err != nil {
					return fmt.Errorf("failed to get Pipeline %q: %w", q, err)
				}
				for _, j := range pp.Jobs {
					for _, s := range j.Plan {
						if s.Get == t && s.GConfig.Trigger {
							err = w.qid.TriggerPipelineJob(ctx, q, j.Name)
							if err != nil {
								return fmt.Errorf("failed to Trigger Pipeline %q with Job %q: %w", q, j.Name, err)
							}
						}
					}
				}
			}
		}
		// Messages must always be acknowledged with Ack.
		msg.Ack()
	}
	return nil
}
