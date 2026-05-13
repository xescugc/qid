package unitwork

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/mysql"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
	"github.com/xescugc/qid/qid/runner"
	"github.com/xescugc/qid/qid/team"
	"github.com/xescugc/qid/qid/user"
)

type unitOfWork struct {
	tx *sql.Tx

	users         user.Repository
	teams         team.Repository
	pipelines     pipeline.Repository
	jobs          job.Repository
	resources     resource.Repository
	resourceTypes restype.Repository
	builds        build.Repository
	runners       runner.Repository
}

func NewStartUnitOfWork(db *sql.DB) StartUnitOfWork {
	return func(ctx context.Context, uowFn func(uow UnitOfWork) error) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		uow := &unitOfWork{tx: tx}

		if err := uowFn(uow); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("failed to rollback: %w (original error: %w)", rbErr, err)
			}
			return err
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	}
}

func (u *unitOfWork) Users() user.Repository {
	if u.users == nil {
		u.users = mysql.NewUserRepository(u.tx)
	}
	return u.users
}

func (u *unitOfWork) Teams() team.Repository {
	if u.teams == nil {
		u.teams = mysql.NewTeamRepository(u.tx)
	}
	return u.teams
}

func (u *unitOfWork) Pipelines() pipeline.Repository {
	if u.pipelines == nil {
		u.pipelines = mysql.NewPipelineRepository(u.tx)
	}
	return u.pipelines
}

func (u *unitOfWork) Jobs() job.Repository {
	if u.jobs == nil {
		u.jobs = mysql.NewJobRepository(u.tx)
	}
	return u.jobs
}

func (u *unitOfWork) Resources() resource.Repository {
	if u.resources == nil {
		u.resources = mysql.NewResourceRepository(u.tx)
	}
	return u.resources
}

func (u *unitOfWork) ResourceTypes() restype.Repository {
	if u.resourceTypes == nil {
		u.resourceTypes = mysql.NewResourceTypeRepository(u.tx)
	}
	return u.resourceTypes
}

func (u *unitOfWork) Builds() build.Repository {
	if u.builds == nil {
		u.builds = mysql.NewBuildRepository(u.tx)
	}
	return u.builds
}

func (u *unitOfWork) Runners() runner.Repository {
	if u.runners == nil {
		u.runners = mysql.NewRunnerRepository(u.tx)
	}
	return u.runners
}
