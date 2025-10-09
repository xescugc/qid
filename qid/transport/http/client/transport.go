package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
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
	r.Header.Set("Content-Type", "application/json")

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

func encodeUpdatePipelineRequest(_ context.Context, r *http.Request, request interface{}) error {
	cfr := request.(transport.UpdatePipelineRequest)
	b, err := json.Marshal(cfr)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(b))
	r.Header.Set("Content-Type", "application/json")

	return nil
}

func decodeUpdatePipelineResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.UpdatePipelineResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeListPipelinesRequest(_ context.Context, r *http.Request, request interface{}) error {
	r.Header.Set("Content-Type", "application/json")
	return nil
}

func decodeListPipelinesResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.ListPipelinesResponse
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
	r.Header.Set("Content-Type", "application/json")

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

func encodeGetPipelineImageRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.GetPipelineImageRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.Name, 1), "{format}", req.Format, 1)

	return nil
}

func decodeGetPipelineImageResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.GetPipelineImageResponse
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
	r.Header.Set("Content-Type", "application/json")

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
	r.Header.Set("Content-Type", "application/json")

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
	r.Header.Set("Content-Type", "application/json")

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

func encodeCreateJobBuildRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.CreateJobBuildRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{job_name}", req.JobName, 1)
	r.Header.Set("Content-Type", "application/json")

	return nil
}

func decodeCreateJobBuildResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.CreateJobBuildResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeUpdateJobBuildRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.UpdateJobBuildRequest)
	r.URL.Path = strings.Replace(strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{job_name}", req.JobName, 1), "{build_id}", strconv.Itoa(int(req.BuildID)), 1)
	r.Header.Set("Content-Type", "application/json")

	return nil
}

func decodeUpdateJobBuildResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.UpdateJobBuildResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeListJobBuildsRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.ListJobBuildsRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{job_name}", req.JobName, 1)
	r.Header.Set("Content-Type", "application/json")

	return nil
}

func decodeListJobBuildsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.ListJobBuildsResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeCreateResourceVersionRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.CreateResourceVersionRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{resource_canonical}", strings.Join([]string{req.ResourceType, req.ResourceName}, ":"), 1)
	r.Header.Set("Content-Type", "application/json")

	return nil
}

func decodeCreateResourceVersionResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.CreateResourceVersionResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeListResourceVersionsRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.ListResourceVersionsRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{resource_canonical}", strings.Join([]string{req.ResourceType, req.ResourceName}, ":"), 1)
	r.Header.Set("Content-Type", "application/json")

	return nil
}

func decodeListResourceVersionsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.ListResourceVersionsResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeGetPipelineResourceRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.GetPipelineResourceRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{resource_canonical}", strings.Join([]string{req.ResourceType, req.ResourceName}, ":"), 1)
	r.Header.Set("Content-Type", "application/json")

	return nil
}

func decodeGetPipelineResourceResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.GetPipelineResourceResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeUpdatePipelineResourceRequest(_ context.Context, r *http.Request, request interface{}) error {
	req := request.(transport.UpdatePipelineResourceRequest)
	r.URL.Path = strings.Replace(strings.Replace(r.URL.Path, "{pipeline_name}", req.PipelineName, 1), "{resource_canonical}", strings.Join([]string{req.ResourceType, req.ResourceName}, ":"), 1)
	r.Header.Set("Content-Type", "application/json")

	return nil
}

func decodeUpdatePipelineResourceResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response transport.UpdatePipelineResourceResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}
