package asynq

import (
	"context"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/xescugc/qid/worker"
)

func Handler(w worker.Service) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		s := strings.Split(t.Type(), ":")
		err := w.ProcessTask(ctx, s[0], s[1])
		if err != nil {
			return err
		}
		return nil
	}
}
