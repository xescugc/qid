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
	GetPipelineImage    endpoint.Endpoint
	CreatePipelineImage endpoint.Endpoint

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

	UpdatePipelineResource endpoint.Endpoint

	CreateResourceVersion endpoint.Endpoint
	ListResourceVersions  endpoint.Endpoint

	GetPipelineResource endpoint.Endpoint
}

func MakeServerEndpoints(s qid.Service) Endpoints {
	return Endpoints{
		GetPipelineImage:    MakeGetPipelineImageEndpoint(s),
		CreatePipelineImage: MakeCreatePipelineImageEndpoint(s),

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

		UpdatePipelineResource: MakeUpdatePipelineResourceEndpoint(s),

		CreateResourceVersion: MakeCreateResourceVersionEndpoint(s),
		ListResourceVersions:  MakeListResourceVersionsEndpoint(s),

		GetPipelineResource: MakeGetPipelineResourceEndpoint(s),
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
	Err string `json:"error,omitempty"`
}

func (r CreateResourceVersionResponse) Error() string { return r.Err }

func MakeCreateResourceVersionEndpoint(s qid.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateResourceVersionRequest)
		err := s.CreateResourceVersion(ctx, req.PipelineName, req.ResourceCanonical, req.Version)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		return CreateResourceVersionResponse{Err: errs}, nil
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
