package transport

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang-jwt/jwt/v5"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/team"
	"github.com/xescugc/qid/qid/user"
)

type Endpoints struct {
	// Web
	GetPipelineImage    endpoint.Endpoint
	CreatePipelineImage endpoint.Endpoint

	// API
	UserLogin endpoint.Endpoint

	ListUsers  endpoint.Endpoint
	CreateUser endpoint.Endpoint

	CreateTeam       endpoint.Endpoint
	UpdateTeam       endpoint.Endpoint
	GetTeam          endpoint.Endpoint
	ListTeams        endpoint.Endpoint
	DeleteTeam       endpoint.Endpoint
	CreateTeamMember endpoint.Endpoint
	UpdateTeamMember endpoint.Endpoint
	DeleteTeamMember endpoint.Endpoint

	CreatePipeline endpoint.Endpoint
	UpdatePipeline endpoint.Endpoint
	ListPipelines  endpoint.Endpoint
	GetPipeline    endpoint.Endpoint
	DeletePipeline endpoint.Endpoint

	TriggerPipelineJob endpoint.Endpoint
	GetPipelineJob     endpoint.Endpoint

	CreateJobBuild endpoint.Endpoint
	UpdateJobBuild endpoint.Endpoint
	DeleteJobBuild endpoint.Endpoint
	ListJobBuilds  endpoint.Endpoint

	UpdatePipelineResource  endpoint.Endpoint
	TriggerPipelineResource endpoint.Endpoint

	CreateResourceVersion endpoint.Endpoint
	ListResourceVersions  endpoint.Endpoint

	GetPipelineResource endpoint.Endpoint
}

func MakeServerEndpoints(s qid.Service, ts []byte) Endpoints {
	return Endpoints{
		GetPipelineImage:    MakeGetPipelineImageEndpoint(s),
		CreatePipelineImage: MakeCreatePipelineImageEndpoint(s),

		UserLogin: MakeUserLoginEndpoint(s, ts),

		ListUsers:  MakeListUsersEndpoint(s),
		CreateUser: MakeCreateUserEndpoint(s),

		CreateTeam: MakeCreateTeamEndpoint(s),
		UpdateTeam: MakeUpdateTeamEndpoint(s),
		GetTeam:    MakeGetTeamEndpoint(s),
		ListTeams:  MakeListTeamsEndpoint(s),
		DeleteTeam: MakeDeleteTeamEndpoint(s),

		CreateTeamMember: MakeCreateTeamMemberEndpoint(s),
		UpdateTeamMember: MakeUpdateTeamMemberEndpoint(s),
		DeleteTeamMember: MakeDeleteTeamMemberEndpoint(s),

		CreatePipeline: MakeCreatePipelineEndpoint(s),
		UpdatePipeline: MakeUpdatePipelineEndpoint(s),
		ListPipelines:  MakeListPipelinesEndpoint(s),
		GetPipeline:    MakeGetPipelineEndpoint(s),
		DeletePipeline: MakeDeletePipelineEndpoint(s),

		TriggerPipelineJob: MakeTriggerPipelineJobEndpoint(s),
		GetPipelineJob:     MakeGetPipelineJobEndpoint(s),

		CreateJobBuild: MakeCreateJobBuildEndpoint(s),
		UpdateJobBuild: MakeUpdateJobBuildEndpoint(s),
		DeleteJobBuild: MakeDeleteJobBuildEndpoint(s),
		ListJobBuilds:  MakeListJobBuildsEndpoint(s),

		UpdatePipelineResource:  MakeUpdatePipelineResourceEndpoint(s),
		TriggerPipelineResource: MakeTriggerPipelineResourceEndpoint(s),

		CreateResourceVersion: MakeCreateResourceVersionEndpoint(s),
		ListResourceVersions:  MakeListResourceVersionsEndpoint(s),

		GetPipelineResource: MakeGetPipelineResourceEndpoint(s),
	}
}

type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type UserLoginResponse struct {
	User *user.User `json:"user,omitempty"`
	JWT  string     `json:"jwt,omitempty"`
	Err  string     `json:"error,omitempty"`
}

func (r UserLoginResponse) Error() string { return r.Err }

func MakeUserLoginEndpoint(s qid.Service, ts []byte) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UserLoginRequest)
		u, err := s.UserLogin(ctx, req.Username, req.Password)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user": u,
		})
		tokenString, err := token.SignedString(ts)
		if err != nil {
			errs = err.Error()
		}
		return UserLoginResponse{User: u, Err: errs, JWT: tokenString}, nil
	}
}

type ListUsersRequest struct{}
type ListUsersResponse struct {
	Users []*user.User `json:"data,omitempty"`
	Err   string       `json:"error,omitempty"`
}

func (r ListUsersResponse) Error() string { return r.Err }

func MakeListUsersEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		_ = request.(ListUsersRequest)
		us, err := s.ListUsers(ctx)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return ListUsersResponse{Users: us, Err: errs}, nil
	}
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsHash   bool   `json:"is_hash"`
}
type CreateUserResponse struct {
	User *user.User `json:"data,omitempty"`
	Err  string     `json:"error,omitempty"`
}

func (r CreateUserResponse) Error() string { return r.Err }

func MakeCreateUserEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateUserRequest)
		u, err := s.CreateUser(ctx, user.User{Username: req.Username, Password: req.Password}, req.IsHash)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateUserResponse{User: u, Err: errs}, nil
	}
}

type CreateTeamRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}
type CreateTeamResponse struct {
	Team *team.WithMembers `json:"data,omitempty"`
	Err  string            `json:"error,omitempty"`
}

func (r CreateTeamResponse) Error() string { return r.Err }

func MakeCreateTeamEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateTeamRequest)
		t, err := s.CreateTeam(ctx, req.Username, team.Team{Name: req.Name})
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateTeamResponse{Team: t, Err: errs}, nil
	}
}

type UpdateTeamRequest struct {
	Name          string `json:"name"`
	TeamCanonical string `json:"team_canonical"`
}
type UpdateTeamResponse struct {
	Team *team.WithMembers `json:"data,omitempty"`
	Err  string            `json:"error,omitempty"`
}

func (r UpdateTeamResponse) Error() string { return r.Err }

func MakeUpdateTeamEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateTeamRequest)
		t, err := s.UpdateTeam(ctx, req.TeamCanonical, team.Team{Name: req.Name})
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateTeamResponse{Team: t, Err: errs}, nil
	}
}

type GetTeamRequest struct {
	TeamCanonical string `json:"team_canonical"`
}
type GetTeamResponse struct {
	Team *team.WithMembers `json:"data,omitempty"`
	Err  string            `json:"error,omitempty"`
}

func (r GetTeamResponse) Error() string { return r.Err }

func MakeGetTeamEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetTeamRequest)
		t, err := s.GetTeam(ctx, req.TeamCanonical)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return GetTeamResponse{Team: t, Err: errs}, nil
	}
}

type ListTeamsRequest struct {
	TeamCanonical string `json:"team_canonical"`
	Username      string `json:"username"`
}
type ListTeamsResponse struct {
	Teams []*team.WithMembers `json:"data,omitempty"`
	Err   string              `json:"error,omitempty"`
}

func (r ListTeamsResponse) Error() string { return r.Err }

func MakeListTeamsEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListTeamsRequest)
		ts, err := s.ListTeams(ctx, req.Username)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return ListTeamsResponse{Teams: ts, Err: errs}, nil
	}
}

type DeleteTeamRequest struct {
	TeamCanonical string `json:"team_canonical"`
}
type DeleteTeamResponse struct {
	Err string `json:"error,omitempty"`
}

func (r DeleteTeamResponse) Error() string { return r.Err }

func MakeDeleteTeamEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteTeamRequest)
		err := s.DeleteTeam(ctx, req.TeamCanonical)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return DeleteTeamResponse{Err: errs}, nil
	}
}

type CreateTeamMemberRequest struct {
	TeamCanonical string `json:"team_canonical"`

	team.Member
}
type CreateTeamMemberResponse struct {
	Member *team.Member `json:"data,omitempty"`
	Err    string       `json:"error,omitempty"`
}

func (r CreateTeamMemberResponse) Error() string { return r.Err }

func MakeCreateTeamMemberEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateTeamMemberRequest)
		tm, err := s.CreateTeamMember(ctx, req.TeamCanonical, req.Member)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateTeamMemberResponse{Member: tm, Err: errs}, nil
	}
}

type UpdateTeamMemberRequest struct {
	TeamCanonical  string `json:"team_canonical"`
	MemberUsername string `json:"member_username"`
	Admin          bool   `json:"admin"`
}
type UpdateTeamMemberResponse struct {
	Member *team.Member `json:"data,omitempty"`
	Err    string       `json:"error,omitempty"`
}

func (r UpdateTeamMemberResponse) Error() string { return r.Err }

func MakeUpdateTeamMemberEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateTeamMemberRequest)
		tm, err := s.UpdateTeamMember(ctx, req.TeamCanonical, req.MemberUsername, team.Member{
			Admin: req.Admin,
			User:  user.User{Username: req.MemberUsername},
		})
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateTeamMemberResponse{Member: tm, Err: errs}, nil
	}
}

type DeleteTeamMemberRequest struct {
	TeamCanonical  string `json:"team_canonical"`
	MemberUsername string `json:"member_username"`
}
type DeleteTeamMemberResponse struct {
	Err string `json:"error,omitempty"`
}

func (r DeleteTeamMemberResponse) Error() string { return r.Err }

func MakeDeleteTeamMemberEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteTeamMemberRequest)
		err := s.DeleteTeamMember(ctx, req.TeamCanonical, req.MemberUsername)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return DeleteTeamMemberResponse{Err: errs}, nil
	}
}

type CreatePipelineRequest struct {
	Name   string                 `json:"name"`
	Config []byte                 `json:"config"`
	Vars   map[string]interface{} `json:"vars"`
}
type CreatePipelineResponse struct {
	Err string `json:"error,omitempty"`
}

func (r CreatePipelineResponse) Error() string { return r.Err }

func MakeCreatePipelineEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreatePipelineRequest)
		err := s.CreatePipeline(ctx, req.Name, req.Config, req.Vars)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreatePipelineResponse{Err: errs}, nil
	}
}

type UpdatePipelineRequest struct {
	Name   string                 `json:"name"`
	Config []byte                 `json:"config"`
	Vars   map[string]interface{} `json:"vars"`
}
type UpdatePipelineResponse struct {
	Err string `json:"error,omitempty"`
}

func (r UpdatePipelineResponse) Error() string { return r.Err }

func MakeUpdatePipelineEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdatePipelineRequest)
		err := s.UpdatePipeline(ctx, req.Name, req.Config, req.Vars)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return UpdatePipelineResponse{Err: errs}, nil
	}
}

type ListPipelinesRequest struct {
}
type ListPipelinesResponse struct {
	Pipelines []*pipeline.Pipeline `json:"data,omitempty"`
	Err       string               `json:"error,omitempty"`
}

func (r ListPipelinesResponse) Error() string { return r.Err }

func MakeListPipelinesEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		_ = request.(ListPipelinesRequest)
		pps, err := s.ListPipelines(ctx)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return ListPipelinesResponse{Pipelines: pps, Err: errs}, nil
	}
}

type GetPipelineRequest struct {
	Name string `json:"name"`
}
type GetPipelineResponse struct {
	Pipeline *pipeline.Pipeline `json:"data,omitempty"`
	Err      string             `json:"error,omitempty"`
}

func (r GetPipelineResponse) Error() string { return r.Err }

func MakeGetPipelineEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetPipelineRequest)
		pp, err := s.GetPipeline(ctx, req.Name)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return GetPipelineResponse{Pipeline: pp, Err: errs}, nil
	}
}

type DeletePipelineRequest struct {
	Name string `json:"name"`
}
type DeletePipelineResponse struct {
	Err string `json:"error,omitempty"`
}

func (r DeletePipelineResponse) Error() string { return r.Err }

func MakeDeletePipelineEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeletePipelineRequest)
		err := s.DeletePipeline(ctx, req.Name)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return DeletePipelineResponse{Err: errs}, nil
	}
}

type TriggerPipelineJobRequest struct {
	PipelineName string `json:"pipeline_name"`
	JobName      string `json:"job_name"`
}
type TriggerPipelineJobResponse struct {
	Err string `json:"error,omitempty"`
}

func (r TriggerPipelineJobResponse) Error() string { return r.Err }

func MakeTriggerPipelineJobEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(TriggerPipelineJobRequest)
		err := s.TriggerPipelineJob(ctx, req.PipelineName, req.JobName)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return TriggerPipelineJobResponse{Err: errs}, nil
	}
}

type GetPipelineJobRequest struct {
	PipelineName string `json:"pipeline_name"`
	JobName      string `json:"job_name"`
}
type GetPipelineJobResponse struct {
	Job *job.Job `json:"data,omitempty"`
	Err string   `json:"error,omitempty"`
}

func (r GetPipelineJobResponse) Error() string { return r.Err }

func MakeGetPipelineJobEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetPipelineJobRequest)
		j, err := s.GetPipelineJob(ctx, req.PipelineName, req.JobName)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return GetPipelineJobResponse{Job: j, Err: errs}, nil
	}
}

type CreateJobBuildRequest struct {
	PipelineName string      `json:"pipeline_name"`
	JobName      string      `json:"job_name"`
	Build        build.Build `json:"build"`
}
type CreateJobBuildResponse struct {
	Build *build.Build `json:"build,omitempty"`
	Err   string       `json:"error,omitempty"`
}

func (r CreateJobBuildResponse) Error() string { return r.Err }

func MakeCreateJobBuildEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateJobBuildRequest)
		b, err := s.CreateJobBuild(ctx, req.PipelineName, req.JobName, req.Build)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateJobBuildResponse{Build: b, Err: errs}, nil
	}
}

type UpdateJobBuildRequest struct {
	PipelineName string      `json:"pipeline_name"`
	JobName      string      `json:"job_name"`
	BuildID      uint32      `json:"build_id"`
	Build        build.Build `json:"build"`
}
type UpdateJobBuildResponse struct {
	Err string `json:"error,omitempty"`
}

func (r UpdateJobBuildResponse) Error() string { return r.Err }

func MakeUpdateJobBuildEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateJobBuildRequest)
		err := s.UpdateJobBuild(ctx, req.PipelineName, req.JobName, req.BuildID, req.Build)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return UpdateJobBuildResponse{Err: errs}, nil
	}
}

type DeleteJobBuildRequest struct {
	PipelineName string `json:"pipeline_name"`
	JobName      string `json:"job_name"`
	BuildID      uint32 `json:"build_id"`
}
type DeleteJobBuildResponse struct {
	Err string `json:"error,omitempty"`
}

func (r DeleteJobBuildResponse) Error() string { return r.Err }

func MakeDeleteJobBuildEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteJobBuildRequest)
		err := s.DeleteJobBuild(ctx, req.PipelineName, req.JobName, req.BuildID)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return DeleteJobBuildResponse{Err: errs}, nil
	}
}

type ListJobBuildsRequest struct {
	PipelineName string `json:"pipeline_name"`
	JobName      string `json:"job_name"`
}
type ListJobBuildsResponse struct {
	Builds []*build.Build `json:"data,omitempty"`
	Err    string         `json:"error,omitempty"`
}

func (r ListJobBuildsResponse) Error() string { return r.Err }

func MakeListJobBuildsEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListJobBuildsRequest)
		builds, err := s.ListJobBuilds(ctx, req.PipelineName, req.JobName)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return ListJobBuildsResponse{Builds: builds, Err: errs}, nil
	}
}

type CreateResourceVersionRequest struct {
	PipelineName      string           `json:"pipeline_name"`
	ResourceCanonical string           `json:"resource_canonical"`
	Version           resource.Version `json:"version"`
}
type CreateResourceVersionResponse struct {
	Err     string            `json:"error,omitempty"`
	Version *resource.Version `json:"version,omitempty"`
}

func (r CreateResourceVersionResponse) Error() string { return r.Err }

func MakeCreateResourceVersionEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateResourceVersionRequest)
		ver, err := s.CreateResourceVersion(ctx, req.PipelineName, req.ResourceCanonical, req.Version)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateResourceVersionResponse{Version: ver, Err: errs}, nil
	}
}

type ListResourceVersionsRequest struct {
	PipelineName      string `json:"pipeline_name"`
	ResourceCanonical string `json:"resource_canonical"`
}
type ListResourceVersionsResponse struct {
	Versions []*resource.Version `json:"data,omitempty"`
	Err      string              `json:"error,omitempty"`
}

func (r ListResourceVersionsResponse) Error() string { return r.Err }

func MakeListResourceVersionsEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListResourceVersionsRequest)
		vers, err := s.ListResourceVersions(ctx, req.PipelineName, req.ResourceCanonical)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return ListResourceVersionsResponse{Versions: vers, Err: errs}, nil
	}
}

type GetPipelineResourceRequest struct {
	PipelineName      string `json:"pipeline_name"`
	ResourceCanonical string `json:"resource_canonical"`
}
type GetPipelineResourceResponse struct {
	Resource *resource.Resource `json:"data,omitempty"`
	Err      string             `json:"error,omitempty"`
}

func (r GetPipelineResourceResponse) Error() string { return r.Err }

func MakeGetPipelineResourceEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetPipelineResourceRequest)
		res, err := s.GetPipelineResource(ctx, req.PipelineName, req.ResourceCanonical)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return GetPipelineResourceResponse{Resource: res, Err: errs}, nil
	}
}

type UpdatePipelineResourceRequest struct {
	PipelineName      string            `json:"pipeline_name"`
	ResourceCanonical string            `json:"resource_canonical"`
	Resource          resource.Resource `json:"resource"`
}
type UpdatePipelineResourceResponse struct {
	Err string `json:"error,omitempty"`
}

func (r UpdatePipelineResourceResponse) Error() string { return r.Err }

func MakeUpdatePipelineResourceEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdatePipelineResourceRequest)
		err := s.UpdatePipelineResource(ctx, req.PipelineName, req.ResourceCanonical, req.Resource)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return UpdatePipelineResourceResponse{Err: errs}, nil
	}
}

type TriggerPipelineResourceRequest struct {
	PipelineName      string `json:"pipeline_name"`
	ResourceCanonical string `json:"resource_canonical"`
}
type TriggerPipelineResourceResponse struct {
	Err string `json:"error,omitempty"`
}

func (r TriggerPipelineResourceResponse) Error() string { return r.Err }

func MakeTriggerPipelineResourceEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(TriggerPipelineResourceRequest)
		err := s.TriggerPipelineResource(ctx, req.PipelineName, req.ResourceCanonical)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return TriggerPipelineResourceResponse{Err: errs}, nil
	}
}

type GetPipelineImageRequest struct {
	Name   string `json:"name"`
	Format string `json:"format"`
}
type GetPipelineImageResponse struct {
	Image string `json:"image,omitempty"`
	Err   string `json:"error,omitempty"`
}

func (r GetPipelineImageResponse) Error() string { return r.Err }

func MakeGetPipelineImageEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetPipelineImageRequest)
		img, err := s.GetPipelineImage(ctx, req.Name, req.Format)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return GetPipelineImageResponse{Image: string(img), Err: errs}, nil
	}
}

type CreatePipelineImageRequest struct {
	Config []byte                 `json:"config"`
	Vars   map[string]interface{} `json:"vars"`
	Format string                 `json:"format"`
}
type CreatePipelineImageResponse struct {
	Image string `json:"image,omitempty"`
	Err   string `json:"error,omitempty"`
}

func (r CreatePipelineImageResponse) Error() string { return r.Err }

func MakeCreatePipelineImageEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreatePipelineImageRequest)
		img, err := s.CreatePipelineImage(ctx, req.Config, req.Vars, req.Format)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreatePipelineImageResponse{Image: string(img), Err: errs}, nil
	}
}
