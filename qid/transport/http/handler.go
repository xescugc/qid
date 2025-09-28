package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
