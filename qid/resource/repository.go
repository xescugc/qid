package resource

import "context"

//go:generate go tool mockgen -destination=../mock/resource_repository.go -mock_names=Repository=ResourceRepository -package mock github.com/xescugc/qid/qid/resource Repository

type Repository interface {
	Create(ctx context.Context, pn string, r Resource) (uint32, error)
	Find(ctx context.Context, pn, rn, rt string) (*Resource, error)
	Filter(ctx context.Context, pn string) ([]*Resource, error)
}
