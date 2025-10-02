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

	r.Methods(http.MethodPost).Path("/pipelines").Handler(kithttp.NewServer(
		e.CreatePipeline,
		decodeCreatePipelineRequest,
		encodeCreatePipelineResponse,
		options...,
	))

	r.Methods(http.MethodGet).Path("/pipelines").Handler(kithttp.NewServer(
		e.ListPipelines,
		decodeListPipelinesRequest,
		encodeListPipelinesResponse,
		options...,
	))

	r.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.GetPipeline,
		decodeGetPipelineRequest,
		encodeGetPipelineResponse,
		options...,
	))

	r.Methods(http.MethodDelete).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.DeletePipeline,
		decodeDeletePipelineRequest,
		encodeDeletePipelineResponse,
		options...,
	))

	r.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}/jobs/{job_name}/trigger").Handler(kithttp.NewServer(
		e.TriggerPipelineJob,
		decodeTriggerPipelineJobRequest,
		encodeTriggerPipelineJobResponse,
		options...,
	))

	r.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/jobs/{job_name}").Handler(kithttp.NewServer(
		e.GetPipelineJob,
		decodeGetPipelineJobRequest,
		encodeGetPipelineJobResponse,
		options...,
	))

	r.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}/jobs/{job_name}/builds").Handler(kithttp.NewServer(
		e.CreateJobBuild,
		decodeCreateJobBuildRequest,
		encodeCreateJobBuildResponse,
		options...,
	))

	r.Methods(http.MethodPut).Path("/pipelines/{pipeline_name}/jobs/{job_name}/builds/{build_id}").Handler(kithttp.NewServer(
		e.UpdateJobBuild,
		decodeUpdateJobBuildRequest,
		encodeUpdateJobBuildResponse,
		options...,
	))

	r.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}/versions").Handler(kithttp.NewServer(
		e.CreateResourceVersion,
		decodeCreateResourceVersionRequest,
		encodeCreateResourceVersionResponse,
		options...,
	))

	r.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}/versions").Handler(kithttp.NewServer(
		e.ListResourceVersions,
		decodeListResourceVersionsRequest,
		encodeListResourceVersionsResponse,
		options...,
	))

	r.NotFoundHandler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Context-Type", "application/json; charset=utf-8")
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

	return nil
}

func decodeListPipelinesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.ListPipelinesRequest

	return req, nil
}

func encodeListPipelinesResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(transport.ListPipelinesResponse)

	json.NewEncoder(w).Encode(resp)

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

	return nil
}
