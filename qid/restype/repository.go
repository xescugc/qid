package restype

import "context"

//go:generate go tool mockgen -destination=../mock/resource_type_repository.go -mock_names=Repository=ResourceTypeRepository -package mock github.com/xescugc/qid/qid/restype Repository

type Repository interface {
	Create(ctx context.Context, pn string, rt ResourceType) (uint32, error)
	Update(ctx context.Context, pn, tn string, rt ResourceType) error
	Find(ctx context.Context, pn, tn string) (*ResourceType, error)
	Filter(ctx context.Context, pn string) ([]*ResourceType, error)
	Delete(ctx context.Context, pn, tn string) error
}
