package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/xescugc/qid/qid/transport"
)

func encodeCreatePipelineRequest(_ context.Context, r *http.Request, request interface{}) error {
	cfr := request.(transport.CreatePipelineRequest)
	b, err := json.Marshal(cfr)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(b))

	return nil
}

func decodeCreatePipelineResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.CreatePipelineResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeGetPipelineRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.GetPipelineRequest)
	r.URL.Path = strings.Replace(r.URL.Path, "{pipeline_name}", req.Name, 1)

	return nil
}

func decodeGetPipelineResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.GetPipelineResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeDeletePipelineRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.DeletePipelineRequest)
	r.URL.Path = strings.Replace(r.URL.Path, "{pipeline_name}", req.Name, 1)

	return nil
}

func decodeDeletePipelineResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.DeletePipelineResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeTriggerPipelineJobRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.TriggerPipelineJobRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{job_name}", req.JobName, 1)

	return nil
}

func decodeTriggerPipelineJobResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.TriggerPipelineJobResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeGetPipelineJobRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.GetPipelineJobRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{job_name}", req.JobName, 1)

	return nil
}

func decodeGetPipelineJobResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.GetPipelineJobResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}
