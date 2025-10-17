package pipeline

import (
	"context"
)

//go:generate go tool mockgen -destination=../mock/pipeline_repository.go -mock_names=Repository=PipelineRepository -package mock github.com/xescugc/qid/qid/pipeline Repository

type Repository interface {
	Create(ctx context.Context, pp Pipeline) (uint32, error)
	Update(ctx context.Context, ppn string, pp Pipeline) error
	Find(ctx context.Context, pn string) (*Pipeline, error)
	Filter(ctx context.Context) ([]*Pipeline, error)
	Delete(ctx context.Context, pn string) error
}
