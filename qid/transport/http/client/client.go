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
	"github.com/xescugc/qid/qid/transport"
)

type Client struct {
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
	listJobBuilds  endpoint.Endpoint

	getPipelineResource    endpoint.Endpoint
	updatePipelineResource endpoint.Endpoint

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

		getPipelineResource:    makeGetPipelineResourceEndpoint(*u),
		updatePipelineResource: makeUpdatePipelineResourceEndpoint(*u),

		createResourceVersion: makeCreateResourceVersionEndpoint(*u),
		listResourceVersions:  makeListResourceVersionsEndpoint(*u),
	}

	return cl, nil
}

func (cl *Client) CreatePipeline(ctx context.Context, pn string, pp []byte, vars map[string]interface{}) error {
	response, err := cl.createPipeline(ctx, transport.CreatePipelineRequest{Name: pn, Config: pp, Vars: vars})
	if err != nil {
		return err
	}

	resp := response.(transport.CreatePipelineResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) UpdatePipeline(ctx context.Context, pn string, pp []byte, vars map[string]interface{}) error {
	response, err := cl.updatePipeline(ctx, transport.UpdatePipelineRequest{Name: pn, Config: pp, Vars: vars})
	if err != nil {
		return err
	}

	resp := response.(transport.UpdatePipelineResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) GetPipeline(ctx context.Context, pn string) (*pipeline.Pipeline, error) {
	response, err := cl.getPipeline(ctx, transport.GetPipelineRequest{Name: pn})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.GetPipelineResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Pipeline, nil
}

func (cl *Client) GetPipelineImage(ctx context.Context, pn, format string) ([]byte, error) {
	response, err := cl.getPipelineImage(ctx, transport.GetPipelineImageRequest{Name: pn, Format: format})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.GetPipelineImageResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return []byte(resp.Image), nil
}

func (cl *Client) CreatePipelineImage(ctx context.Context, pp []byte, vars map[string]interface{}, format string) ([]byte, error) {
	response, err := cl.createPipelineImage(ctx, transport.CreatePipelineImageRequest{Config: pp, Vars: vars, Format: format})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.CreatePipelineImageResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return []byte(resp.Image), nil
}

func (cl *Client) ListPipelines(ctx context.Context) ([]*pipeline.Pipeline, error) {
	response, err := cl.listPipelines(ctx, transport.ListPipelinesRequest{})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.ListPipelinesResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Pipelines, nil
}

func (cl *Client) DeletePipeline(ctx context.Context, pn string) error {
	response, err := cl.deletePipeline(ctx, transport.DeletePipelineRequest{Name: pn})
	if err != nil {
		return err
	}

	resp := response.(transport.DeletePipelineResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) TriggerPipelineJob(ctx context.Context, ppn, jn string) error {
	response, err := cl.triggerPipelineJob(ctx, transport.TriggerPipelineJobRequest{PipelineName: ppn, JobName: jn})
	if err != nil {
		return err
	}

	resp := response.(transport.TriggerPipelineJobResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) GetPipelineJob(ctx context.Context, ppn, jn string) (*job.Job, error) {
	response, err := cl.getPipelineJob(ctx, transport.GetPipelineJobRequest{PipelineName: ppn, JobName: jn})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.GetPipelineJobResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Job, nil
}

func (cl *Client) CreateJobBuild(ctx context.Context, pn, jn string, b build.Build) (*build.Build, error) {
	response, err := cl.createJobBuild(ctx, transport.CreateJobBuildRequest{PipelineName: pn, JobName: jn, Build: b})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.CreateJobBuildResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Build, nil
}

func (cl *Client) UpdateJobBuild(ctx context.Context, pn, jn string, bID uint32, b build.Build) error {
	response, err := cl.updateJobBuild(ctx, transport.UpdateJobBuildRequest{PipelineName: pn, JobName: jn, BuildID: bID, Build: b})
	if err != nil {
		return err
	}

	resp := response.(transport.UpdateJobBuildResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) ListJobBuilds(ctx context.Context, pn, jn string) ([]*build.Build, error) {
	response, err := cl.listJobBuilds(ctx, transport.ListJobBuildsRequest{PipelineName: pn, JobName: jn})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.ListJobBuildsResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Builds, nil
}

func (cl *Client) CreateResourceVersion(ctx context.Context, pn, rCan string, rv resource.Version) error {
	response, err := cl.updateJobBuild(ctx, transport.CreateResourceVersionRequest{PipelineName: pn, ResourceCanonical: rCan, Version: rv})
	if err != nil {
		return err
	}

	resp := response.(transport.CreateResourceVersionResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

func (cl *Client) ListResourceVersions(ctx context.Context, pn, rCan string) ([]*resource.Version, error) {
	response, err := cl.listResourceVersions(ctx, transport.ListResourceVersionsRequest{PipelineName: pn, ResourceCanonical: rCan})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.ListResourceVersionsResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Versions, nil
}

func (cl *Client) GetPipelineResource(ctx context.Context, pn, rCan string) (*resource.Resource, error) {
	response, err := cl.getPipelineResource(ctx, transport.GetPipelineResourceRequest{PipelineName: pn, ResourceCanonical: rCan})
	if err != nil {
		return nil, err
	}

	resp := response.(transport.GetPipelineResourceResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.Resource, nil
}

func (cl *Client) UpdatePipelineResource(ctx context.Context, pn, rCan string, r resource.Resource) error {
	response, err := cl.updatePipelineResource(ctx, transport.UpdatePipelineResourceRequest{PipelineName: pn, ResourceCanonical: rCan, Resource: r})
	if err != nil {
		return err
	}

	resp := response.(transport.UpdatePipelineResourceResponse)
	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}
