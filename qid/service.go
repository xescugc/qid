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

	"log/slog"
)

//go:generate go tool mockgen -destination=mock/service.go -mock_names=Service=Service -package mock github.com/xescugc/qid/qid Service

type Service interface {
	UserLogin(ctx context.Context, un, pass string) (*user.WithMemberships, string, error)
	RefreshToken(ctx context.Context, un string) (*user.WithMemberships, string, error)

	GetUser(ctx context.Context, un string) (*user.WithMemberships, error)
	CreateUser(ctx context.Context, u user.User, isHash bool) (*user.User, error)
	ListUsers(ctx context.Context) ([]*user.User, error)

	CreateTeam(ctx context.Context, un string, t team.Team) (*team.WithMembers, error)
	ListTeams(ctx context.Context, un string) ([]*team.WithMembers, error)
	GetTeam(ctx context.Context, tc string) (*team.WithMembers, error)
	UpdateTeam(ctx context.Context, tc string, t team.Team) (*team.WithMembers, error)
	DeleteTeam(ctx context.Context, tc string) error

	CreateTeamMember(ctx context.Context, tc string, tm team.Member) (*team.Member, error)
	UpdateTeamMember(ctx context.Context, tc, mc string, tm team.Member) (*team.Member, error)
	DeleteTeamMember(ctx context.Context, tc, mc string) error

	CreatePipeline(ctx context.Context, tc, pn string, pp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error)
	UpdatePipeline(ctx context.Context, tc, pn string, pp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error)
	GetPipeline(ctx context.Context, tc, pn string) (*pipeline.Pipeline, error)
	DeletePipeline(ctx context.Context, tc, pn string) error
	ListPipelines(ctx context.Context, tc string) ([]*pipeline.Pipeline, error)

	GetPipelineImage(ctx context.Context, tc, pn, format string) ([]byte, error)
	CreatePipelineImage(ctx context.Context, tc string, pp []byte, vars map[string]interface{}, format string) ([]byte, error)

	TriggerPipelineJob(ctx context.Context, tc, pn, jn string) error
	GetPipelineJob(ctx context.Context, tc, pn, jn string) (*job.Job, error)

	CreateJobBuild(ctx context.Context, tc, pn, jn string, b build.Build) (*build.Build, error)
	UpdateJobBuild(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error
	DeleteJobBuild(ctx context.Context, tc, pn, jn string, bID uint32) error
	ListJobBuilds(ctx context.Context, tc, pn, jn string) ([]*build.Build, error)

	GetPipelineResource(ctx context.Context, tc, pn, rCan string) (*resource.Resource, error)
	UpdatePipelineResource(ctx context.Context, tc, pn, rCan string, r resource.Resource) error
	TriggerPipelineResource(ctx context.Context, tc, pn, rCan string) error
	CreateResourceVersion(ctx context.Context, tc, pn, rCan string, v resource.Version) (*resource.Version, error)
	ListResourceVersions(ctx context.Context, tc, pn, rCan string) ([]*resource.Version, error)
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

	JWTSecret []byte

	cron   *cron.Cron
	logger *slog.Logger
}

func New(ctx context.Context, t queue.Topic, ur user.Repository, tr team.Repository, pr pipeline.Repository, jr job.Repository, rr resource.Repository, rt restype.Repository, br build.Repository, rur runner.Repository, js []byte, l *slog.Logger) *Qid {
	q := &Qid{
		Ctx:           ctx,
		Topic:         t,
		Users:         ur,
		Teams:         tr,
		Pipelines:     pr,
		Jobs:          jr,
		Resources:     rr,
		ResourceTypes: rt,
		Builds:        br,
		Runners:       rur,
		JWTSecret:     js,
		logger:        l,
		cron:          cron.New(cron.WithContext(ctx)),
	}

	q.cron.Start()

	return q
}
