package queue

import "context"

//go:generate go tool mockgen -destination=../mock/queue.go -mock_names=Queue=Queue -package mock github.com/xescugc/qid/qid/queue Queue

type Queue interface {
	// Push will push to the Queue(q) the Task(t)
	Push(ctx context.Context, q, t string) error
}
