package qid

import (
	"context"

	cron "github.com/netresearch/go-cron"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
	"github.com/xescugc/qid/qid/runner"
	"github.com/xescugc/qid/qid/team"
	"github.com/xescugc/qid/qid/user"

	"github.com/go-kit/kit/log"
)

type Service interface {
	UserLogin(ctx context.Context, un, pass string) (*user.User, error)

	GetUser(ctx context.Context, un string) (*user.WithMemberships, error)
	CreateUser(ctx context.Context, u user.User, isHash bool) (*user.User, error)
	ListUsers(ctx context.Context) ([]*user.User, error)

	CreateTeam(ctx context.Context, un string, t team.Team) (*team.WithMembers, error)
	ListTeams(ctx context.Context, un string) ([]*team.WithMembers, error)
	GetTeam(ctx context.Context, un, tc string) (*team.WithMembers, error)
	UpdateTeam(ctx context.Context, un, tc string, t team.Team) (*team.WithMembers, error)
	DeleteTeam(ctx context.Context, un, tc string) error

	CreateTeamMember(ctx context.Context, un, tc string, tm team.Member) (*team.Member, error)
	UpdateTeamMember(ctx context.Context, un, tc, mc string, tm team.Member) (*team.Member, error)
	DeleteTeamMember(ctx context.Context, un, tc, mc string) error

	CreatePipeline(ctx context.Context, pn string, pp []byte, vars map[string]interface{}) error
	UpdatePipeline(ctx context.Context, pn string, pp []byte, vars map[string]interface{}) error
	GetPipeline(ctx context.Context, pn string) (*pipeline.Pipeline, error)
	DeletePipeline(ctx context.Context, pn string) error
	ListPipelines(ctx context.Context) ([]*pipeline.Pipeline, error)

	GetPipelineImage(ctx context.Context, pn, format string) ([]byte, error)
	CreatePipelineImage(ctx context.Context, pp []byte, vars map[string]interface{}, format string) ([]byte, error)

	TriggerPipelineJob(ctx context.Context, pn, jn string) error
	GetPipelineJob(ctx context.Context, pn, jn string) (*job.Job, error)

	CreateJobBuild(ctx context.Context, pn, jn string, b build.Build) (*build.Build, error)
	UpdateJobBuild(ctx context.Context, pn, jn string, bID uint32, b build.Build) error
	DeleteJobBuild(ctx context.Context, pn, jn string, bID uint32) error
	ListJobBuilds(ctx context.Context, pn, jn string) ([]*build.Build, error)

	GetPipelineResource(ctx context.Context, pn, rCan string) (*resource.Resource, error)
	UpdatePipelineResource(ctx context.Context, pn, rCan string, r resource.Resource) error
	TriggerPipelineResource(ctx context.Context, pn, rCan string) error
	CreateResourceVersion(ctx context.Context, pn, rCan string, v resource.Version) (*resource.Version, error)
	ListResourceVersions(ctx context.Context, pn, rCan string) ([]*resource.Version, error)
}

type Qid struct {
	Topic         queue.Topic
	Users         user.Repository
	Teams         team.Repository
	Pipelines     pipeline.Repository
	Jobs          job.Repository
	Resources     resource.Repository
	ResourceTypes restype.Repository
	Builds        build.Repository
	Runners       runner.Repository
	Ctx           context.Context

	cron   *cron.Cron
	logger log.Logger
}

func New(ctx context.Context, t queue.Topic, ur user.Repository, tr team.Repository, pr pipeline.Repository, jr job.Repository, rr resource.Repository, rt restype.Repository, br build.Repository, rur runner.Repository, l log.Logger) *Qid {
	q := &Qid{
		Topic:         t,
		Users:         ur,
		Teams:         tr,
		Pipelines:     pr,
		Jobs:          jr,
		Resources:     rr,
		ResourceTypes: rt,
		Builds:        br,
		Runners:       rur,
		Ctx:           ctx,
		logger:        l,
		cron:          cron.New(cron.WithContext(ctx)),
	}

	q.cron.Start()

	return q
}
