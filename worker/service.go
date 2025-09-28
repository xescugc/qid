package worker

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/davecgh/go-spew/spew"
	"github.com/xescugc/qid/qid"
)

type Service interface {
	ProcessTask(ctx context.Context, q, t string) error
}

type Worker struct {
	qid qid.Service
}

func New(s qid.Service) *Worker {
	return &Worker{
		qid: s,
	}
}

func (w *Worker) ProcessTask(ctx context.Context, q, t string) error {
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
	return nil
}
