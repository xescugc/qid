package job

import "context"

//go:generate go tool mockgen -destination=../mock/job_repository.go -mock_names=Repository=JobRepository -package mock github.com/xescugc/qid/qid/job Repository

type Repository interface {
	Create(ctx context.Context, pn string, j Job) (uint32, error)
	Find(ctx context.Context, pn, jn string) (*Job, error)
	Filter(ctx context.Context, pn string) ([]*Job, error)
}
