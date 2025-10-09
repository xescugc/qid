package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/transport"
	"github.com/xescugc/qid/qid/transport/http/assets"
	"github.com/xescugc/qid/qid/transport/http/templates"

	kittransport "github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

func Handler(s qid.Service, l log.Logger) http.Handler {
	r := mux.NewRouter()
	e := transport.MakeServerEndpoints(s)

	options := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(kittransport.NewLogErrorHandler(l)),
	}

	api := r.Headers("Content-Type", "application/json").Subrouter()

	api.Methods(http.MethodPost).Path("/pipelines").Handler(kithttp.NewServer(
		e.CreatePipeline,
		decodeCreatePipelineRequest,
		encodeCreatePipelineResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines").Handler(kithttp.NewServer(
		e.ListPipelines,
		decodeListPipelinesRequest,
		encodeListPipelinesResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.GetPipeline,
		decodeGetPipelineRequest,
		encodeGetPipelineResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.UpdatePipeline,
		decodeUpdatePipelineRequest,
		encodeUpdatePipelineResponse,
		options...,
	))

	api.Methods(http.MethodDelete).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.DeletePipeline,
		decodeDeletePipelineRequest,
		encodeDeletePipelineResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}/jobs/{job_name}/trigger").Handler(kithttp.NewServer(
		e.TriggerPipelineJob,
		decodeTriggerPipelineJobRequest,
		encodeTriggerPipelineJobResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/jobs/{job_name}").Handler(kithttp.NewServer(
		e.GetPipelineJob,
		decodeGetPipelineJobRequest,
		encodeGetPipelineJobResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}/jobs/{job_name}/builds").Handler(kithttp.NewServer(
		e.CreateJobBuild,
		decodeCreateJobBuildRequest,
		encodeCreateJobBuildResponse,
	))

	api.Methods(http.MethodPut).Path("/pipelines/{pipeline_name}/jobs/{job_name}/builds/{build_id}").Handler(kithttp.NewServer(
		e.UpdateJobBuild,
		decodeUpdateJobBuildRequest,
		encodeUpdateJobBuildResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}/versions").Handler(kithttp.NewServer(
		e.CreateResourceVersion,
		decodeCreateResourceVersionRequest,
		encodeCreateResourceVersionResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}/versions").Handler(kithttp.NewServer(
		e.ListResourceVersions,
		decodeListResourceVersionsRequest,
		encodeListResourceVersionsResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}").Handler(kithttp.NewServer(
		e.GetPipelineResource,
		decodeGetPipelineResourceRequest,
		encodeGetPipelineResourceResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}").Handler(kithttp.NewServer(
		e.UpdatePipelineResource,
		decodeUpdatePipelineResourceRequest,
		encodeUpdatePipelineResourceResponse,
		options...,
	))

	r.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.ShowPipeline,
		decodeShowPipelineRequest,
		encodeShowPipelineResponse,
		options...,
	))

	r.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/image{ext}").Handler(kithttp.NewServer(
		e.GetPipelineImage,
		decodeGetPipelineImageRequest,
		encodeGetPipelineImageResponse,
		options...,
	))

	r.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/jobs/{job_name}/builds").Handler(kithttp.NewServer(
		e.IndexJobBuilds,
		decodeIndexJobBuildsRequest,
		encodeIndexJobBuildsResponse,
		options...,
	))

	r.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/resources/{resource_type}/{resource_name}/versions").Handler(kithttp.NewServer(
		e.IndexResourceVersions,
		decodeIndexResourceVersionsRequest,
		encodeIndexResourceVersionsResponse,
		options...,
	))

	r.PathPrefix("/css/").Handler(http.FileServer(http.FS(assets.Assets)))
	r.PathPrefix("/js/").Handler(http.FileServer(http.FS(assets.Assets)))

	r.NotFoundHandler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Path not found"}`)
		},
	)

	return r
}

func decodeCreatePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreatePipelineRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	return req, err
}

func encodeCreatePipelineResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.CreatePipelineResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeUpdatePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.UpdatePipelineRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	return req, err
}

func encodeUpdatePipelineResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.UpdatePipelineResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeListPipelinesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.ListPipelinesRequest

	return req, nil
}

func encodeListPipelinesResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.ListPipelinesResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeGetPipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.GetPipelineRequest{
		Name: vars["pipeline_name"],
	}, nil
}

func encodeGetPipelineResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.GetPipelineResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeDeletePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.DeletePipelineRequest{
		Name: vars["pipeline_name"],
	}, nil
}

func encodeDeletePipelineResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.DeletePipelineResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeTriggerPipelineJobRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.TriggerPipelineJobRequest{
		PipelineName: vars["pipeline_name"],
		JobName:      vars["job_name"],
	}, nil
}

func encodeTriggerPipelineJobResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.TriggerPipelineJobResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeGetPipelineJobRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.GetPipelineJobRequest{
		PipelineName: vars["pipeline_name"],
		JobName:      vars["job_name"],
	}, nil
}

func encodeGetPipelineJobResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.GetPipelineJobResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeCreateJobBuildRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	req := transport.CreateJobBuildRequest{
		PipelineName: vars["pipeline_name"],
		JobName:      vars["job_name"],
	}
	err := json.NewDecoder(r.Body).Decode(&req.Build)

	return req, err
}

func encodeCreateJobBuildResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.CreateJobBuildResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeUpdateJobBuildRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	bid, _ := strconv.Atoi(vars["build_id"])
	req := transport.UpdateJobBuildRequest{
		PipelineName: vars["pipeline_name"],
		JobName:      vars["job_name"],
		BuildID:      uint32(bid),
	}
	err := json.NewDecoder(r.Body).Decode(&req.Build)

	return req, err
}

func encodeUpdateJobBuildResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.UpdateJobBuildResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeCreateResourceVersionRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	rc := strings.Split(vars["resource_canonical"], ":")
	req := transport.CreateResourceVersionRequest{
		PipelineName: vars["pipeline_name"],
		ResourceType: rc[0],
		ResourceName: rc[1],
	}
	err := json.NewDecoder(r.Body).Decode(&req.Version)

	return req, err
}

func encodeCreateResourceVersionResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.CreateResourceVersionResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeListResourceVersionsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	rc := strings.Split(vars["resource_canonical"], ":")
	req := transport.ListResourceVersionsRequest{
		PipelineName: vars["pipeline_name"],
		ResourceType: rc[0],
		ResourceName: rc[1],
	}

	return req, nil
}

func encodeListResourceVersionsResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.ListResourceVersionsResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeGetPipelineResourceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	rc := strings.Split(vars["resource_canonical"], ":")
	req := transport.GetPipelineResourceRequest{
		PipelineName: vars["pipeline_name"],
		ResourceType: rc[0],
		ResourceName: rc[1],
	}

	return req, nil
}

func encodeGetPipelineResourceResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.GetPipelineResourceResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeUpdatePipelineResourceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	rc := strings.Split(vars["resource_canonical"], ":")
	req := transport.UpdatePipelineResourceRequest{
		PipelineName: vars["pipeline_name"],
		ResourceType: rc[0],
		ResourceName: rc[1],
	}

	err := json.NewDecoder(r.Body).Decode(&req.Resource)

	return req, err
}

func encodeUpdatePipelineResourceResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.UpdatePipelineResourceResponse)

	json.NewEncoder(w).Encode(resp)

	w.Header().Set("Content-Type", "application/json")

	return nil
}

func decodeShowPipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.ShowPipelineRequest{
		Name: vars["pipeline_name"],
	}, nil
}

func encodeShowPipelineResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.ShowPipelineResponse)
	t, _ := templates.Templates["views/pipelines/show.tmpl"]
	return t.Execute(w, resp)
}

func decodeGetPipelineImageRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.GetPipelineImageRequest{
		Name:   vars["pipeline_name"],
		Format: vars["format"],
	}, nil
}

func encodeGetPipelineImageResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.GetPipelineImageResponse)
	fmt.Fprint(w, string(resp.Image))
	return nil
}

func decodeIndexJobBuildsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.IndexJobBuildsRequest{
		PipelineName: vars["pipeline_name"],
		JobName:      vars["job_name"],
	}, nil
}

func encodeIndexJobBuildsResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.IndexJobBuildsResponse)
	t, _ := templates.Templates["views/builds/index.tmpl"]
	return t.Execute(w, resp)
}

func decodeIndexResourceVersionsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.IndexResourceVersionsRequest{
		PipelineName: vars["pipeline_name"],
		ResourceType: vars["resource_type"],
		ResourceName: vars["resource_name"],
	}, nil
}

func encodeIndexResourceVersionsResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.IndexResourceVersionsResponse)
	t, _ := templates.Templates["views/resource_versions/index.tmpl"]
	return t.Execute(w, resp)
}
