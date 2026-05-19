package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/xescugc/pikoci/pikoci"
	"github.com/xescugc/pikoci/pikoci/build"
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

func updateJobBuild(s pikoci.Service) http.HandlerFunc {
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

func deleteJobBuild(s pikoci.Service) http.HandlerFunc {
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

type InsertBuildGetVersionRequest struct {
	TeamCanonical string `json:"team_canonical"`
	PipelineName  string `json:"pipeline_name"`
	JobName       string `json:"job_name"`
	BuildID       uint32 `json:"build_id"`
	StepName      string `json:"step_name"`
	VersionID     uint32 `json:"version_id"`
}
type InsertBuildGetVersionResponse struct {
	Err string `json:"error,omitempty"`
}

func (r InsertBuildGetVersionResponse) Error() string { return r.Err }

func insertBuildGetVersion(s pikoci.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req InsertBuildGetVersionRequest
			ctx = r.Context()
		)
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			encodeResponse(InsertBuildGetVersionResponse{Err: err.Error()}, w)
			return
		}
		vars := mux.Vars(r)
		req.TeamCanonical = vars["team_canonical"]
		req.PipelineName = vars["pipeline_name"]
		req.JobName = vars["job_name"]
		bid, _ := strconv.Atoi(vars["build_id"])
		req.BuildID = uint32(bid)

		err = s.InsertBuildGetVersion(ctx, req.TeamCanonical, req.PipelineName, req.JobName, req.BuildID, req.StepName, req.VersionID)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		encodeResponse(InsertBuildGetVersionResponse{Err: errs}, w)
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

func listJobBuilds(s pikoci.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req ListJobBuildsRequest
			ctx = r.Context()
		)
		vars := mux.Vars(r)
		req.TeamCanonical = vars["team_canonical"]
		req.PipelineName = vars["pipeline_name"]
		req.JobName = vars["job_name"]
		var builds []*build.Build
		var err error
		if isPublic, _ := ctx.Value(IsPublicAccessKey).(bool); isPublic {
			builds, err = s.ListPublicJobBuilds(ctx, req.TeamCanonical, req.PipelineName, req.JobName)
		} else {
			builds, err = s.ListJobBuilds(ctx, req.TeamCanonical, req.PipelineName, req.JobName)
		}
		var errs string
		if err != nil {
			errs = err.Error()
		}
		encodeResponse(ListJobBuildsResponse{Builds: builds, Err: errs}, w)
	}
}
