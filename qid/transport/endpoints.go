package transport

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
)

type Endpoints struct {
	CreatePipeline     endpoint.Endpoint
	ListPipelines      endpoint.Endpoint
	GetPipeline        endpoint.Endpoint
	DeletePipeline     endpoint.Endpoint
	TriggerPipelineJob endpoint.Endpoint
	GetPipelineJob     endpoint.Endpoint
	CreateJobBuild     endpoint.Endpoint
	UpdateJobBuild     endpoint.Endpoint
}

func MakeServerEndpoints(s qid.Service) Endpoints {
	return Endpoints{
		CreatePipeline:     MakeCreatePipelineEndpoint(s),
		ListPipelines:      MakeListPipelinesEndpoint(s),
		GetPipeline:        MakeGetPipelineEndpoint(s),
		DeletePipeline:     MakeDeletePipelineEndpoint(s),
		TriggerPipelineJob: MakeTriggerPipelineJobEndpoint(s),
		GetPipelineJob:     MakeGetPipelineJobEndpoint(s),
		CreateJobBuild:     MakeCreateJobBuildEndpoint(s),
		UpdateJobBuild:     MakeUpdateJobBuildEndpoint(s),
	}
}

type CreatePipelineRequest struct {
	Name   string `json:"name"`
	Config []byte `json:"config"`
}
type CreatePipelineResponse struct {
	Err string `json:"error,omitempty"`
}

func MakeCreatePipelineEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreatePipelineRequest)
		err := s.CreatePipeline(ctx, req.Name, req.Config)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreatePipelineResponse{Err: errs}, nil
	}
}

type ListPipelinesRequest struct {
}
type ListPipelinesResponse struct {
	Pipelines []*pipeline.Pipeline `json:"pipeline,omitempty"`
	Err       string               `json:"error,omitempty"`
}

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
	Pipeline *pipeline.Pipeline `json:"pipeline,omitempty"`
	Err      string             `json:"error,omitempty"`
}

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
	Job *job.Job `json:"job,omitempty"`
	Err string   `json:"error,omitempty"`
}

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
