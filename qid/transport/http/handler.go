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

	// If the URL ends in `.json` it'll match a URL with `Content-Type=application/json`
	jsm := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, rr *http.Request) {
			if strings.HasSuffix(rr.URL.String(), ".json") {
				rr.URL.Path = strings.TrimSuffix(rr.URL.Path, ".json")
				rr.Header.Set("Content-Type", "application/json")
				r.ServeHTTP(rw, rr)
				return
			}
			h.ServeHTTP(rw, rr)
		})
	}

	r.Use(jsm)

	api := r.Headers("Content-Type", "application/json").Subrouter()

	api.Methods(http.MethodPost).Path("/pipelines").Handler(kithttp.NewServer(
		e.CreatePipeline,
		decodeCreatePipelineRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines").Handler(kithttp.NewServer(
		e.ListPipelines,
		decodeListPipelinesRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/pipelines/image{ext}").Handler(kithttp.NewServer(
		e.CreatePipelineImage,
		decodeCreatePipelineImageRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.GetPipeline,
		decodeGetPipelineRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.UpdatePipeline,
		decodeUpdatePipelineRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodDelete).Path("/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.DeletePipeline,
		decodeDeletePipelineRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}/jobs/{job_name}/trigger").Handler(kithttp.NewServer(
		e.TriggerPipelineJob,
		decodeTriggerPipelineJobRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/jobs/{job_name}").Handler(kithttp.NewServer(
		e.GetPipelineJob,
		decodeGetPipelineJobRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/jobs/{job_name}/builds").Handler(kithttp.NewServer(
		e.ListJobBuilds,
		decodeListJobBuildsRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/pipelines/{pipeline_name}/jobs/{job_name}/builds/{build_id}").Handler(kithttp.NewServer(
		e.UpdateJobBuild,
		decodeUpdateJobBuildRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}/versions").Handler(kithttp.NewServer(
		e.CreateResourceVersion,
		decodeCreateResourceVersionRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}/versions").Handler(kithttp.NewServer(
		e.ListResourceVersions,
		decodeListResourceVersionsRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}").Handler(kithttp.NewServer(
		e.GetPipelineResource,
		decodeGetPipelineResourceRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/pipelines/{pipeline_name}/resources/{resource_canonical}").Handler(kithttp.NewServer(
		e.UpdatePipelineResource,
		decodeUpdatePipelineResourceRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/pipelines/{pipeline_name}/image{ext}").Handler(kithttp.NewServer(
		e.GetPipelineImage,
		decodeGetPipelineImageRequest,
		encodeJSONResponse,
		options...,
	))

	api.NotFoundHandler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Path not found"}`)
		},
	)

	r.PathPrefix("/css/").Handler(http.FileServer(http.FS(assets.Assets)))
	r.PathPrefix("/js/").Handler(http.FileServer(http.FS(assets.Assets)))
	r.PathPrefix("/images/").Handler(http.FileServer(http.FS(assets.Assets)))

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t, _ := templates.Templates["views/layouts/index.tmpl"]
		t.Execute(w, nil)
	})

	return r
}

func decodeCreatePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreatePipelineRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	return req, err
}

func decodeUpdatePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.UpdatePipelineRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	vars := mux.Vars(r)
	req.Name = vars["pipeline_name"]

	return req, err
}

func decodeListPipelinesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.ListPipelinesRequest

	return req, nil
}

func decodeGetPipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.GetPipelineRequest{
		Name: vars["pipeline_name"],
	}, nil
}

func decodeDeletePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.DeletePipelineRequest{
		Name: vars["pipeline_name"],
	}, nil
}

func decodeTriggerPipelineJobRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.TriggerPipelineJobRequest{
		PipelineName: vars["pipeline_name"],
		JobName:      vars["job_name"],
	}, nil
}

func decodeGetPipelineJobRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.GetPipelineJobRequest{
		PipelineName: vars["pipeline_name"],
		JobName:      vars["job_name"],
	}, nil
}

func decodeListJobBuildsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.ListJobBuildsRequest{
		PipelineName: vars["pipeline_name"],
		JobName:      vars["job_name"],
	}, nil
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

func decodeCreateResourceVersionRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	req := transport.CreateResourceVersionRequest{
		PipelineName:      vars["pipeline_name"],
		ResourceCanonical: vars["resource_canonical"],
	}
	err := json.NewDecoder(r.Body).Decode(&req.Version)

	return req, err
}

func decodeListResourceVersionsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	req := transport.ListResourceVersionsRequest{
		PipelineName:      vars["pipeline_name"],
		ResourceCanonical: vars["resource_canonical"],
	}

	return req, nil
}

func decodeGetPipelineResourceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	req := transport.GetPipelineResourceRequest{
		PipelineName:      vars["pipeline_name"],
		ResourceCanonical: vars["resource_canonical"],
	}

	return req, nil
}

func decodeUpdatePipelineResourceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	req := transport.UpdatePipelineResourceRequest{
		PipelineName:      vars["pipeline_name"],
		ResourceCanonical: vars["resource_canonical"],
	}

	err := json.NewDecoder(r.Body).Decode(&req.Resource)

	return req, err
}

func decodeGetPipelineImageRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	return transport.GetPipelineImageRequest{
		Name:   vars["pipeline_name"],
		Format: vars["format"],
	}, nil
}

func decodeCreatePipelineImageRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreatePipelineImageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	vars := mux.Vars(r)
	req.Format = vars["format"]

	return req, err
}

func encodeJSONResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(Errorer); ok && e.Error() != "" {
		encodeError(ctx, e.Error(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)

	w.Header().Set("Content-Type", "application/json")
	return nil
}

type Errorer interface {
	Error() string
}

func encodeError(_ context.Context, err string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err,
	})
}
