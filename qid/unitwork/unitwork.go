package unitwork

import (
	"context"

	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
	"github.com/xescugc/qid/qid/runner"
	"github.com/xescugc/qid/qid/team"
	"github.com/xescugc/qid/qid/user"
)

type StartUnitOfWork func(ctx context.Context, uowFn func(uow UnitOfWork) error) error

type UnitOfWork interface {
	Users() user.Repository
	Teams() team.Repository
	Pipelines() pipeline.Repository
	Jobs() job.Repository
	Resources() resource.Repository
	ResourceTypes() restype.Repository
	Builds() build.Repository
	Runners() runner.Repository
}

// Repositories holds all repository interfaces, used to construct a noop UoW for testing.
type Repositories struct {
	UsersRepo         user.Repository
	TeamsRepo         team.Repository
	PipelinesRepo     pipeline.Repository
	JobsRepo          job.Repository
	ResourcesRepo     resource.Repository
	ResourceTypesRepo restype.Repository
	BuildsRepo        build.Repository
	RunnersRepo       runner.Repository
}
