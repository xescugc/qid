package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/transport"
)

type Client struct {
	createPipeline     endpoint.Endpoint
	getPipeline        endpoint.Endpoint
	deletePipeline     endpoint.Endpoint
	triggerPipelineJob endpoint.Endpoint
	getPipelineJob     endpoint.Endpoint
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
		createPipeline:     makeCreatePipelineEndpoint(*u),
		getPipeline:        makeGetPipelineEndpoint(*u),
		deletePipeline:     makeDeletePipelineEndpoint(*u),
		triggerPipelineJob: makeTriggerPipelineJobEndpoint(*u),
		getPipelineJob:     makeGetPipelineJobEndpoint(*u),
	}

	return cl, nil
}

func (cl *Client) CreatePipeline(ctx context.Context, pn string, pp []byte) error {
	response, err := cl.createPipeline(ctx, transport.CreatePipelineRequest{Name: pn, Config: pp})
	if err != nil {
		return err
	}

	resp := response.(transport.CreatePipelineResponse)
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
