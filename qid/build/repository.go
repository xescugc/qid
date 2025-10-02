package build

import "context"

//go:generate go tool mockgen -destination=../mock/build_repository.go -mock_names=Repository=BuildRepository -package mock github.com/xescugc/qid/qid/build Repository

type Repository interface {
	Create(ctx context.Context, pn, jn string, b Build) (uint32, error)
	Find(ctx context.Context, pn, jn string, bID uint32) (*Build, error)
	Filter(ctx context.Context, pn, jn string) ([]*Build, error)
	Update(ctx context.Context, pn, jn string, bID uint32, b Build) error
}
