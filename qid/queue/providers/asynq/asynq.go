package asynq

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

type Asynq struct {
	asynq *asynq.Client
}

func New(redisAddr string) *Asynq {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})

	return &Asynq{
		asynq: client,
	}
}

func (a *Asynq) Push(ctx context.Context, q, t string) error {
	task := asynq.NewTask(fmt.Sprintf("%s:%s", q, t), nil, asynq.Queue("default"), asynq.MaxRetry(-1))

	// Asynq cannot register to all the queues
	_, err := a.asynq.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("could not enqueue task %q no queue %q: %w", t, q, err)
	}

	return nil
}
