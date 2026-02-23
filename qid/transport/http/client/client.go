package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/team"
	"github.com/xescugc/qid/qid/transport"
	"github.com/xescugc/qid/qid/user"
)

type Client struct {
	userLogin endpoint.Endpoint

	createUser endpoint.Endpoint

	createTeam endpoint.Endpoint
	updateTeam endpoint.Endpoint
	getTeam    endpoint.Endpoint
	listTeams  endpoint.Endpoint
	deleteTeam endpoint.Endpoint

	createTeamMember endpoint.Endpoint
	deleteTeamMember endpoint.Endpoint

	createPipeline      endpoint.Endpoint
	updatePipeline      endpoint.Endpoint
	getPipeline         endpoint.Endpoint
	getPipelineImage    endpoint.Endpoint
	createPipelineImage endpoint.Endpoint
	listPipelines       endpoint.Endpoint
	deletePipeline      endpoint.Endpoint

	triggerPipelineJob endpoint.Endpoint
	getPipelineJob     endpoint.Endpoint

	createJobBuild endpoint.Endpoint
	updateJobBuild endpoint.Endpoint
	deleteJobBuild endpoint.Endpoint
	listJobBuilds  endpoint.Endpoint

	getPipelineResource     endpoint.Endpoint
	updatePipelineResource  endpoint.Endpoint
	triggerPipelineResource endpoint.Endpoint

	createResourceVersion endpoint.Endpoint
	listResourceVersions  endpoint.Endpoint
}

// New returns a new HTTP Client for QID
func New(host string) (*Client, error) {
	if host == "" {
		return nil, fmt.Errorf("can't initialize the %q with an empty host", "qid")
	}
	if !strings.HasPrefix(host, "http") {
		host = fmt.Sprintf("http://%s", host)
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	cl := &Client{
		userLogin: makeUserLoginEndpoint(*u),

		//createUser: makeCreateUserEndpoint(*u),
		//createTeam: makeCreateTeamEndpoint(*u),
		//updateTeam: makeUpdateTeamEndpoint(*u),
		//getTeam:    makeGetTeamEndpoint(*u),
		//listTeams:  makeListTeamsEndpoint(*u),
		//deleteTeam: makeDeleteTeamEndpoint(*u),
		//createTeamMember
		//deleteTeamMebmer

		createPipeline:      makeCreatePipelineEndpoint(*u),
		updatePipeline:      makeUpdatePipelineEndpoint(*u),
		getPipeline:         makeGetPipelineEndpoint(*u),
		getPipelineImage:    makeGetPipelineImageEndpoint(*u),
		createPipelineImage: makeCreatePipelineImageEndpoint(*u),
		listPipelines:       makeListPipelinesEndpoint(*u),
		deletePipeline:      makeDeletePipelineEndpoint(*u),

		triggerPipelineJob: makeTriggerPipelineJobEndpoint(*u),
		getPipelineJob:     makeGetPipelineJobEndpoint(*u),

		listJobBuilds:  makeListJobBuildsEndpoint(*u),
		createJobBuild: makeCreateJobBuildEndpoint(*u),
		updateJobBuild: makeUpdateJobBuildEndpoint(*u),
		deleteJobBuild: makeDeleteJobBuildEndpoint(*u),

		getPipelineResource:     makeGetPipelineResourceEndpoint(*u),
		updatePipelineResource:  makeUpdatePipelineResourceEndpoint(*u),
		triggerPipelineResource: makeTriggerPipelineResourceEndpoint(*u),

		createResourceVersion: makeCreateResourceVersionEndpoint(*u),
		listResourceVersions:  makeListResourceVersionsEndpoint(*u),
	}

	return cl, nil
}

func (cl *Client) UserLogin(ctx context.Context, un, pass string) (*user.User, error) {
	response, err := cl.userLogin(ctx, transport.UserLoginRequest{Username: un, Password: pass})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.UserLoginResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.User, nil
}

func (cl *Client) GetUser(ctx context.Context, un string) (*user.WithMemberships, error) {
	return nil, nil
}

func (cl *Client) CreateUser(ctx context.Context, u user.User, isHash bool) (*user.User, error) {
	return nil, nil
}

// TODO: Implement those client actions
func (cl *Client) ListUsers(ctx context.Context) ([]*user.User, error) {
	return nil, nil
}
func (cl *Client) CreateTeam(ctx context.Context, un string, t team.Team) (*team.WithMembers, error) {
	return nil, nil
}
func (cl *Client) ListTeams(ctx context.Context, un string) ([]*team.WithMembers, error) {
	return nil, nil
}
func (cl *Client) GetTeam(ctx context.Context, tc string) (*team.WithMembers, error) {
	return nil, nil
}
func (cl *Client) UpdateTeam(ctx context.Context, tc string, t team.Team) (*team.WithMembers, error) {
	return nil, nil
}
func (cl *Client) DeleteTeam(ctx context.Context, tc string) error {
	return nil
}
func (cl *Client) CreateTeamMember(ctx context.Context, tc string, tm team.Member) (*team.Member, error) {
	return nil, nil
}
func (cl *Client) UpdateTeamMember(ctx context.Context, tc, mu string, tm team.Member) (*team.Member, error) {
	return nil, nil
}
func (cl *Client) DeleteTeamMember(ctx context.Context, tc, mu string) error {
	return nil
}

func (cl *Client) CreatePipeline(ctx context.Context, tc, pn string, pp []byte, vars map[string]interface{}) error {
	response, err := cl.createPipeline(ctx, transport.CreatePipelineRequest{TeamCanonical: tc, Name: pn, Config: pp, Vars: vars})
	if err != nil {
		return err
	}

	resp := response.(transport.CreatePipelineResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) UpdatePipeline(ctx context.Context, tc, pn string, pp []byte, vars map[string]interface{}) error {
	response, err := cl.updatePipeline(ctx, transport.UpdatePipelineRequest{TeamCanonical: tc, Name: pn, Config: pp, Vars: vars})
	if err != nil {
		return err
	}

	resp := response.(transport.UpdatePipelineResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) GetPipeline(ctx context.Context, tc, pn string) (*pipeline.Pipeline, error) {
	response, err := cl.getPipeline(ctx, transport.GetPipelineRequest{TeamCanonical: tc, Name: pn})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.GetPipelineResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Pipeline, nil
}

func (cl *Client) GetPipelineImage(ctx context.Context, tc, pn, format string) ([]byte, error) {
	response, err := cl.getPipelineImage(ctx, transport.GetPipelineImageRequest{TeamCanonical: tc, Name: pn, Format: format})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.GetPipelineImageResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return []byte(resp.Image), nil
}

func (cl *Client) CreatePipelineImage(ctx context.Context, tc string, pp []byte, vars map[string]interface{}, format string) ([]byte, error) {
	response, err := cl.createPipelineImage(ctx, transport.CreatePipelineImageRequest{TeamCanonical: tc, Config: pp, Vars: vars, Format: format})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.CreatePipelineImageResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return []byte(resp.Image), nil
}

func (cl *Client) ListPipelines(ctx context.Context, tc string) ([]*pipeline.Pipeline, error) {
	response, err := cl.listPipelines(ctx, transport.ListPipelinesRequest{TeamCanonical: tc})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.ListPipelinesResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Pipelines, nil
}

func (cl *Client) DeletePipeline(ctx context.Context, tc, pn string) error {
	response, err := cl.deletePipeline(ctx, transport.DeletePipelineRequest{TeamCanonical: tc, Name: pn})
	if err != nil {
		return err
	}

	resp := response.(transport.DeletePipelineResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) TriggerPipelineJob(ctx context.Context, tc, ppn, jn string) error {
	response, err := cl.triggerPipelineJob(ctx, transport.TriggerPipelineJobRequest{TeamCanonical: tc, PipelineName: ppn, JobName: jn})
	if err != nil {
		return err
	}

	resp := response.(transport.TriggerPipelineJobResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) GetPipelineJob(ctx context.Context, tc, ppn, jn string) (*job.Job, error) {
	response, err := cl.getPipelineJob(ctx, transport.GetPipelineJobRequest{TeamCanonical: tc, PipelineName: ppn, JobName: jn})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.GetPipelineJobResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Job, nil
}

func (cl *Client) CreateJobBuild(ctx context.Context, tc, pn, jn string, b build.Build) (*build.Build, error) {
	response, err := cl.createJobBuild(ctx, transport.CreateJobBuildRequest{TeamCanonical: tc, PipelineName: pn, JobName: jn, Build: b})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.CreateJobBuildResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Build, nil
}

func (cl *Client) UpdateJobBuild(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
	response, err := cl.updateJobBuild(ctx, transport.UpdateJobBuildRequest{TeamCanonical: tc, PipelineName: pn, JobName: jn, BuildID: bID, Build: b})
	if err != nil {
		return err
	}

	resp := response.(transport.UpdateJobBuildResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) DeleteJobBuild(ctx context.Context, tc, pn, jn string, bID uint32) error {
	response, err := cl.deleteJobBuild(ctx, transport.DeleteJobBuildRequest{TeamCanonical: tc, PipelineName: pn, JobName: jn, BuildID: bID})
	if err != nil {
		return err
	}

	resp := response.(transport.DeleteJobBuildResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) ListJobBuilds(ctx context.Context, tc, pn, jn string) ([]*build.Build, error) {
	response, err := cl.listJobBuilds(ctx, transport.ListJobBuildsRequest{TeamCanonical: tc, PipelineName: pn, JobName: jn})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.ListJobBuildsResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Builds, nil
}

func (cl *Client) CreateResourceVersion(ctx context.Context, tc, pn, rCan string, rv resource.Version) (*resource.Version, error) {
	response, err := cl.updateJobBuild(ctx, transport.CreateResourceVersionRequest{TeamCanonical: tc, PipelineName: pn, ResourceCanonical: rCan, Version: rv})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.CreateResourceVersionResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Version, nil
}

func (cl *Client) ListResourceVersions(ctx context.Context, tc, pn, rCan string) ([]*resource.Version, error) {
	response, err := cl.listResourceVersions(ctx, transport.ListResourceVersionsRequest{TeamCanonical: tc, PipelineName: pn, ResourceCanonical: rCan})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.ListResourceVersionsResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Versions, nil
}

func (cl *Client) GetPipelineResource(ctx context.Context, tc, pn, rCan string) (*resource.Resource, error) {
	response, err := cl.getPipelineResource(ctx, transport.GetPipelineResourceRequest{TeamCanonical: tc, PipelineName: pn, ResourceCanonical: rCan})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.GetPipelineResourceResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Resource, nil
}

func (cl *Client) UpdatePipelineResource(ctx context.Context, tc, pn, rCan string, r resource.Resource) error {
	response, err := cl.updatePipelineResource(ctx, transport.UpdatePipelineResourceRequest{TeamCanonical: tc, PipelineName: pn, ResourceCanonical: rCan, Resource: r})
	if err != nil {
		return err
	}

	resp := response.(transport.UpdatePipelineResourceResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) TriggerPipelineResource(ctx context.Context, tc, pn, rCan string) error {
	response, err := cl.triggerPipelineResource(ctx, transport.TriggerPipelineResourceRequest{TeamCanonical: tc, PipelineName: pn, ResourceCanonical: rCan})
	if err != nil {
		return err
	}

	resp := response.(transport.TriggerPipelineResourceResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}
