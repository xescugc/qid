package runner

import "context"

//go:generate go tool mockgen -destination=../mock/runner_repository.go -mock_names=Repository=RunnerRepository -package mock github.com/xescugc/qid/qid/runner Repository

type Repository interface {
	Create(ctx context.Context, pn string, ru Runner) (uint32, error)
	Find(ctx context.Context, pn, run string) (*Runner, error)
	Filter(ctx context.Context, pn string) ([]*Runner, error)
	Update(ctx context.Context, pn, run string, ru Runner) error
	Delete(ctx context.Context, pn, run string) error
}
