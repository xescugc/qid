package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/build"
)

type CreateJobBuildRequest struct {
	TeamCanonical string      `json:"team_canonical"`
	PipelineName  string      `json:"pipeline_name"`
	JobName       string      `json:"job_name"`
	Build         build.Build `json:"build"`
}
type CreateJobBuildResponse struct {
	Build *build.Build `json:"build,omitempty"`
	Err   string       `json:"error,omitempty"`
}

func (r CreateJobBuildResponse) Error() string { return r.Err }

func createJobBuild(s qid.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req CreateJobBuildRequest
			ctx = r.Context()
		)
		vars := mux.Vars(r)
		req.TeamCanonical = vars["team_canonical"]
		req.PipelineName = vars["pipeline_name"]
		req.JobName = vars["job_name"]
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			encodeResponse(CreateJobBuildResponse{Err: err.Error()}, w)
			return
		}
		b, err := s.CreateJobBuild(ctx, req.TeamCanonical, req.PipelineName, req.JobName, req.Build)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		encodeResponse(CreateJobBuildResponse{Build: b, Err: errs}, w)
	}
}

type UpdateJobBuildRequest struct {
	TeamCanonical string      `json:"team_canonical"`
	PipelineName  string      `json:"pipeline_name"`
	JobName       string      `json:"job_name"`
	BuildID       uint32      `json:"build_id"`
	Build         build.Build `json:"build"`
}
type UpdateJobBuildResponse struct {
	Err string `json:"error,omitempty"`
}

func (r UpdateJobBuildResponse) Error() string { return r.Err }

func updateJobBuild(s qid.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req UpdateJobBuildRequest
			ctx = r.Context()
		)
		vars := mux.Vars(r)
		bid, _ := strconv.Atoi(vars["build_id"])
		req.TeamCanonical = vars["team_canonical"]
		req.PipelineName = vars["pipeline_name"]
		req.JobName = vars["job_name"]
		req.BuildID = uint32(bid)
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			encodeResponse(UpdateJobBuildResponse{Err: err.Error()}, w)
			return
		}
		err = s.UpdateJobBuild(ctx, req.TeamCanonical, req.PipelineName, req.JobName, req.BuildID, req.Build)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		encodeResponse(UpdateJobBuildResponse{Err: errs}, w)
	}
}

type DeleteJobBuildRequest struct {
	TeamCanonical string `json:"team_canonical"`
	PipelineName  string `json:"pipeline_name"`
	JobName       string `json:"job_name"`
	BuildID       uint32 `json:"build_id"`
}
type DeleteJobBuildResponse struct {
	Err string `json:"error,omitempty"`
}

func (r DeleteJobBuildResponse) Error() string { return r.Err }

func deleteJobBuild(s qid.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req UpdateJobBuildRequest
			ctx = r.Context()
		)
		vars := mux.Vars(r)
		bid, _ := strconv.Atoi(vars["build_id"])
		req.TeamCanonical = vars["team_canonical"]
		req.PipelineName = vars["pipeline_name"]
		req.JobName = vars["job_name"]
		req.BuildID = uint32(bid)
		err := s.DeleteJobBuild(ctx, req.TeamCanonical, req.PipelineName, req.JobName, req.BuildID)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		encodeResponse(DeleteJobBuildResponse{Err: errs}, w)
	}
}

type ListJobBuildsRequest struct {
	TeamCanonical string `json:"team_canonical"`
	PipelineName  string `json:"pipeline_name"`
	JobName       string `json:"job_name"`
}
type ListJobBuildsResponse struct {
	Builds []*build.Build `json:"data,omitempty"`
	Err    string         `json:"error,omitempty"`
}

func (r ListJobBuildsResponse) Error() string { return r.Err }

func listJobBuilds(s qid.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req ListJobBuildsRequest
			ctx = r.Context()
		)
		vars := mux.Vars(r)
		req.TeamCanonical = vars["team_canonical"]
		req.PipelineName = vars["pipeline_name"]
		req.JobName = vars["job_name"]
		builds, err := s.ListJobBuilds(ctx, req.TeamCanonical, req.PipelineName, req.JobName)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		encodeResponse(ListJobBuildsResponse{Builds: builds, Err: errs}, w)
	}
}
