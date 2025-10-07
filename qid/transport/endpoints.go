package transport

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/resource"
)

type Endpoints struct {
	// Web
	ShowPipeline     endpoint.Endpoint
	GetPipelineImage endpoint.Endpoint

	// API
	CreatePipeline endpoint.Endpoint
	UpdatePipeline endpoint.Endpoint
	ListPipelines  endpoint.Endpoint
	GetPipeline    endpoint.Endpoint
	DeletePipeline endpoint.Endpoint

	TriggerPipelineJob endpoint.Endpoint
	GetPipelineJob     endpoint.Endpoint

	CreateJobBuild endpoint.Endpoint
	UpdateJobBuild endpoint.Endpoint
	ListJobBuilds  endpoint.Endpoint

	CreateResourceVersion endpoint.Endpoint
	ListResourceVersions  endpoint.Endpoint
}

func MakeServerEndpoints(s qid.Service) Endpoints {
	return Endpoints{
		ShowPipeline:     MakeShowPipelineEndpoint(s),
		GetPipelineImage: MakeGetPipelineImageEndpoint(s),

		CreatePipeline: MakeCreatePipelineEndpoint(s),
		UpdatePipeline: MakeUpdatePipelineEndpoint(s),
		ListPipelines:  MakeListPipelinesEndpoint(s),
		GetPipeline:    MakeGetPipelineEndpoint(s),
		DeletePipeline: MakeDeletePipelineEndpoint(s),

		TriggerPipelineJob: MakeTriggerPipelineJobEndpoint(s),
		GetPipelineJob:     MakeGetPipelineJobEndpoint(s),

		CreateJobBuild: MakeCreateJobBuildEndpoint(s),
		UpdateJobBuild: MakeUpdateJobBuildEndpoint(s),
		ListJobBuilds:  MakeListJobBuildsEndpoint(s),

		CreateResourceVersion: MakeCreateResourceVersionEndpoint(s),
		ListResourceVersions:  MakeListResourceVersionsEndpoint(s),
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

type UpdatePipelineRequest struct {
	Name   string `json:"name"`
	Config []byte `json:"config"`
}
type UpdatePipelineResponse struct {
	Err string `json:"error,omitempty"`
}

func MakeUpdatePipelineEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdatePipelineRequest)
		err := s.UpdatePipeline(ctx, req.Name, req.Config)
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

type CreateResourceVersionRequest struct {
	PipelineName string           `json:"pipeline_name"`
	ResourceName string           `json:"resource_name"`
	ResourceType string           `json:"resource_type"`
	Version      resource.Version `json:"version"`
}
type CreateResourceVersionResponse struct {
	Err string `json:"error,omitempty"`
}

func MakeCreateResourceVersionEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateResourceVersionRequest)
		err := s.CreateResourceVersion(ctx, req.PipelineName, req.ResourceType, req.ResourceName, req.Version)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateResourceVersionResponse{Err: errs}, nil
	}
}

type ListResourceVersionsRequest struct {
	PipelineName string `json:"pipeline_name"`
	ResourceName string `json:"resource_name"`
	ResourceType string `json:"resource_type"`
}
type ListResourceVersionsResponse struct {
	Versions []*resource.Version
	Err      string `json:"error,omitempty"`
}

func MakeListResourceVersionsEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListResourceVersionsRequest)
		vers, err := s.ListResourceVersions(ctx, req.PipelineName, req.ResourceType, req.ResourceName)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return ListResourceVersionsResponse{Versions: vers, Err: errs}, nil
	}
}

type ShowPipelineRequest struct {
	Name string `json:"name"`
}
type ShowPipelineResponse struct {
	Pipeline *pipeline.Pipeline `json:"pipeline,omitempty"`
	Err      string             `json:"error,omitempty"`
}

func MakeShowPipelineEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ShowPipelineRequest)
		pp, err := s.GetPipeline(ctx, req.Name)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return ShowPipelineResponse{Pipeline: pp, Err: errs}, nil
	}
}

type GetPipelineImageRequest struct {
	Name   string `json:"name"`
	Format string `json:"name"`
}
type GetPipelineImageResponse struct {
	Image []byte `json:"image,omitempty"`
	Err   string `json:"error,omitempty"`
}

func MakeGetPipelineImageEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetPipelineImageRequest)
		img, err := s.GetPipelineImage(ctx, req.Name, req.Format)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return GetPipelineImageResponse{Image: img, Err: errs}, nil
	}
}

type ListJobBuildsRequest struct {
	PipelineName string `json:"pipeline_name"`
	JobName      string `json:"job_name"`
}
type ListJobBuildsResponse struct {
	Pipeline *pipeline.Pipeline `json:"pipeline,omitempty"`
	Job      *job.Job           `json:"pipeline,omitempty"`
	Builds   []*build.Build     `json:"builds,omitempty"`
	Err      string             `json:"error,omitempty"`
}

func MakeListJobBuildsEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListJobBuildsRequest)
		pp, err := s.GetPipeline(ctx, req.PipelineName)
		if err != nil {
			return ListJobBuildsResponse{Err: err.Error()}, nil
		}
		job, err := s.GetPipelineJob(ctx, req.PipelineName, req.JobName)
		if err != nil {
			return ListJobBuildsResponse{Err: err.Error()}, nil
		}
		builds, err := s.ListJobBuilds(ctx, req.PipelineName, req.JobName)
		if err != nil {
			return ListJobBuildsResponse{Err: err.Error()}, nil
		}
		return ListJobBuildsResponse{Pipeline: pp, Job: job, Builds: builds}, nil
	}
}
