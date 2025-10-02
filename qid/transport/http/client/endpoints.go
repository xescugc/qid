package client

import (
	"net/http"
	"net/url"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
)

func makeCreatePipelineEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines"
	return kithttp.NewClient(
		http.MethodPost,
		&u,
		encodeCreatePipelineRequest,
		decodeCreatePipelineResponse,
	).Endpoint()
}

func makeListPipelinesEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines"
	return kithttp.NewClient(
		http.MethodGet,
		&u,
		encodeListPipelinesRequest,
		decodeListPipelinesResponse,
	).Endpoint()
}

func makeGetPipelineEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines/{pipeline_name}"
	return kithttp.NewClient(
		http.MethodGet,
		&u,
		encodeGetPipelineRequest,
		decodeGetPipelineResponse,
	).Endpoint()
}

func makeDeletePipelineEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines/{pipeline_name}"
	return kithttp.NewClient(
		http.MethodDelete,
		&u,
		encodeDeletePipelineRequest,
		decodeDeletePipelineResponse,
	).Endpoint()
}

func makeTriggerPipelineJobEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines/{pipeline_name}/jobs/{job_name}/trigger"
	return kithttp.NewClient(
		http.MethodPost,
		&u,
		encodeTriggerPipelineJobRequest,
		decodeTriggerPipelineJobResponse,
	).Endpoint()
}

func makeGetPipelineJobEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines/{pipeline_name}/jobs/{job_name}"
	return kithttp.NewClient(
		http.MethodGet,
		&u,
		encodeGetPipelineJobRequest,
		decodeGetPipelineJobResponse,
	).Endpoint()
}

func makeCreateJobBuildEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines/{pipeline_name}/jobs/{job_name}/builds"
	return kithttp.NewClient(
		http.MethodGet,
		&u,
		encodeCreateJobBuildRequest,
		decodeCreateJobBuildResponse,
	).Endpoint()
}

func makeUpdateJobBuildEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines/{pipeline_name}/jobs/{job_name}/builds/{build_id}"
	return kithttp.NewClient(
		http.MethodGet,
		&u,
		encodeUpdateJobBuildRequest,
		decodeUpdateJobBuildResponse,
	).Endpoint()
}

func makeCreateResourceVersionEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines/{pipeline_name}/resources/{resource_canonical}/versions"
	return kithttp.NewClient(
		http.MethodPost,
		&u,
		encodeCreateResourceVersionRequest,
		decodeCreateResourceVersionResponse,
	).Endpoint()
}

func makeListResourceVersionsEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/pipelines/{pipeline_name}/resources/{resource_canonical}/versions"
	return kithttp.NewClient(
		http.MethodGet,
		&u,
		encodeListResourceVersionsRequest,
		decodeListResourceVersionsResponse,
	).Endpoint()
}
