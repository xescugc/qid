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

type noopUnitOfWork struct {
	repos Repositories
}

func NewNoopStartUnitOfWork(repos Repositories) StartUnitOfWork {
	return func(ctx context.Context, uowFn func(uow UnitOfWork) error) error {
		uow := &noopUnitOfWork{repos: repos}
		return uowFn(uow)
	}
}

func (u *noopUnitOfWork) Users() user.Repository         { return u.repos.UsersRepo }
func (u *noopUnitOfWork) Teams() team.Repository         { return u.repos.TeamsRepo }
func (u *noopUnitOfWork) Pipelines() pipeline.Repository { return u.repos.PipelinesRepo }
func (u *noopUnitOfWork) Jobs() job.Repository           { return u.repos.JobsRepo }
func (u *noopUnitOfWork) Resources() resource.Repository { return u.repos.ResourcesRepo }
func (u *noopUnitOfWork) ResourceTypes() restype.Repository {
	return u.repos.ResourceTypesRepo
}
func (u *noopUnitOfWork) Builds() build.Repository   { return u.repos.BuildsRepo }
func (u *noopUnitOfWork) Runners() runner.Repository { return u.repos.RunnersRepo }
