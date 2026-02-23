package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/transport"
	"github.com/xescugc/qid/qid/transport/http/assets"
	"github.com/xescugc/qid/qid/transport/http/templates"

	kittransport "github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

const (
	UsernameContextKey string = "username_context_key"
)

func Handler(s qid.Service, ts []byte, l log.Logger) http.Handler {
	r := mux.NewRouter()
	e := transport.MakeServerEndpoints(s, ts)

	options := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(kittransport.NewLogErrorHandler(l)),
	}

	auth := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, rr *http.Request) {
			// Aauthentication
			reqToken := rr.Header.Get("Authorization")
			splitToken := strings.Split(reqToken, " ")
			if len(splitToken) != 2 {
				encodeError(rr.Context(), "Authentication required", rw)
				return
			}
			tokenString := splitToken[1]
			if reqToken == "" {
				encodeError(rr.Context(), "Authentication required", rw)
				return
			}
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
				return ts, nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
			if err != nil {
				l.Log("error", err)
				encodeError(rr.Context(), "Authentication required", rw)
				return
			}

			var un string
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				un = claims["user"].(map[string]interface{})["username"].(string)
				rr = rr.WithContext(context.WithValue(rr.Context(), UsernameContextKey, un))
			} else {
				l.Log("error", err)
				encodeError(rr.Context(), "Authentication required", rw)
				return
			}

			// Authorization

			cr := mux.CurrentRoute(rr)
			crns := cr.GetName()
			if crns == "" {
				pt, _ := cr.GetPathTemplate()
				encodeError(rr.Context(), fmt.Sprintf("Route %s has no name", pt), rw)
				return
			}

			crn, err := RouteNameString(crns)
			if err != nil {
				pt, _ := cr.GetPathTemplate()
				encodeError(rr.Context(), fmt.Sprintf("Route %s has no name conversion(%s)", pt, crns), rw)
				return
			}

			afn, ok := routeAuthorization[crn]
			if !ok {
				pt, _ := cr.GetPathTemplate()
				encodeError(rr.Context(), fmt.Sprintf("Route %s has no auth", pt), rw)
				return
			}

			vars := mux.Vars(rr)
			tc := vars["team_canonical"]
			err = afn(rr.Context(), s, un, tc)
			if err != nil {
				l.Log("error", err)
				encodeError(rr.Context(), "Authentication required", rw)
				return
			}

			h.ServeHTTP(rw, rr)
		})
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

	jsonr := r.Headers("Content-Type", "application/json").Subrouter()

	jsonr.Methods(http.MethodPost).Path("/login").Handler(kithttp.NewServer(
		e.UserLogin,
		decodeUserLoginRequest,
		encodeJSONResponse,
		options...,
	))

	api := jsonr.PathPrefix("/").Subrouter()

	api.Use(auth)

	api.Methods(http.MethodGet).Path("/users").Name(ListUsers.String()).Handler(kithttp.NewServer(
		e.ListUsers,
		decodeListUsersRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/users").Name(CreateUser.String()).Handler(kithttp.NewServer(
		e.CreateUser,
		decodeCreateUserRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/teams").Name(CreateTeam.String()).Handler(kithttp.NewServer(
		e.CreateTeam,
		decodeCreateTeamRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams").Name(ListTeams.String()).Handler(kithttp.NewServer(
		e.ListTeams,
		decodeListTeamsRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams/{team_canonical}").Name(GetTeam.String()).Handler(kithttp.NewServer(
		e.GetTeam,
		decodeGetTeamRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/teams/{team_canonical}").Name(UpdateTeam.String()).Handler(kithttp.NewServer(
		e.UpdateTeam,
		decodeUpdateTeamRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodDelete).Path("/teams/{team_canonical}").Name(DeleteTeam.String()).Handler(kithttp.NewServer(
		e.DeleteTeam,
		decodeDeleteTeamRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/teams/{team_canonical}/members").Name(CreateTeamMember.String()).Handler(kithttp.NewServer(
		e.CreateTeamMember,
		decodeCreateTeamMemberRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/teams/{team_canonical}/members/{member_username}").Name(UpdateTeamMember.String()).Handler(kithttp.NewServer(
		e.UpdateTeamMember,
		decodeUpdateTeamMemberRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodDelete).Path("/teams/{team_canonical}/members/{member_username}").Name(DeleteTeamMember.String()).Handler(kithttp.NewServer(
		e.DeleteTeamMember,
		decodeDeleteTeamMemberRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/teams/{team_canonical}/pipelines").Handler(kithttp.NewServer(
		e.CreatePipeline,
		decodeCreatePipelineRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams/{team_canonical}/pipelines").Handler(kithttp.NewServer(
		e.ListPipelines,
		decodeListPipelinesRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/teams/{team_canonical}/pipelines/image{ext}").Handler(kithttp.NewServer(
		e.CreatePipelineImage,
		decodeCreatePipelineImageRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams/{team_canonical}/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.GetPipeline,
		decodeGetPipelineRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/teams/{team_canonical}/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.UpdatePipeline,
		decodeUpdatePipelineRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodDelete).Path("/teams/{team_canonical}/pipelines/{pipeline_name}").Handler(kithttp.NewServer(
		e.DeletePipeline,
		decodeDeletePipelineRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/jobs/{job_name}/trigger").Handler(kithttp.NewServer(
		e.TriggerPipelineJob,
		decodeTriggerPipelineJobRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/jobs/{job_name}").Handler(kithttp.NewServer(
		e.GetPipelineJob,
		decodeGetPipelineJobRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/jobs/{job_name}/builds").Handler(kithttp.NewServer(
		e.ListJobBuilds,
		decodeListJobBuildsRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/jobs/{job_name}/builds/{build_id}").Handler(kithttp.NewServer(
		e.UpdateJobBuild,
		decodeUpdateJobBuildRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodDelete).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/jobs/{job_name}/builds/{build_id}").Handler(kithttp.NewServer(
		e.DeleteJobBuild,
		decodeDeleteJobBuildRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/resources/{resource_canonical}/versions").Handler(kithttp.NewServer(
		e.CreateResourceVersion,
		decodeCreateResourceVersionRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/resources/{resource_canonical}/versions").Handler(kithttp.NewServer(
		e.ListResourceVersions,
		decodeListResourceVersionsRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/resources/{resource_canonical}").Handler(kithttp.NewServer(
		e.GetPipelineResource,
		decodeGetPipelineResourceRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPut).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/resources/{resource_canonical}").Handler(kithttp.NewServer(
		e.UpdatePipelineResource,
		decodeUpdatePipelineResourceRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodPost).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/resources/{resource_canonical}/trigger").Handler(kithttp.NewServer(
		e.TriggerPipelineResource,
		decodeTriggerPipelineResourceRequest,
		encodeJSONResponse,
		options...,
	))

	api.Methods(http.MethodGet).Path("/teams/{team_canonical}/pipelines/{pipeline_name}/image{ext}").Handler(kithttp.NewServer(
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

func decodeUserLoginRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.UserLoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	return req, err
}

func decodeListUsersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.ListUsersRequest

	return req, nil
}

func decodeCreateUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	return req, err
}

func decodeCreateTeamRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreateTeamRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	req.Username = r.Context().Value(UsernameContextKey).(string)

	return req, err
}

func decodeListTeamsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req transport.ListTeamsRequest
	req.Username = r.Context().Value(UsernameContextKey).(string)

	return req, nil
}

func decodeGetTeamRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.GetTeamRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]

	return req, nil
}

func decodeUpdateTeamRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.UpdateTeamRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]

	return req, err
}

func decodeDeleteTeamRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.DeleteTeamRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]

	return req, nil
}

func decodeCreateTeamMemberRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreateTeamMemberRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]

	return req, err
}

func decodeUpdateTeamMemberRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.UpdateTeamMemberRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.MemberUsername = vars["member_username"]

	return req, err
}

func decodeDeleteTeamMemberRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.DeleteTeamMemberRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.MemberUsername = vars["member_username"]

	return req, nil
}

func decodeCreatePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreatePipelineRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	err := json.NewDecoder(r.Body).Decode(&req)

	return req, err
}

func decodeUpdatePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.UpdatePipelineRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.Name = vars["pipeline_name"]

	return req, err
}

func decodeListPipelinesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.ListPipelinesRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]

	return req, nil
}

func decodeGetPipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.GetPipelineRequest
	vars := mux.Vars(r)
	req.Name = vars["pipeline_name"]
	req.TeamCanonical = vars["team_canonical"]

	return req, nil
}

func decodeDeletePipelineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.DeletePipelineRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.Name = vars["pipeline_name"]

	return req, nil
}

func decodeTriggerPipelineJobRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.TriggerPipelineJobRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.JobName = vars["job_name"]

	return req, nil
}

func decodeGetPipelineJobRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.GetPipelineJobRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.JobName = vars["job_name"]

	return req, nil
}

func decodeListJobBuildsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.ListJobBuildsRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.JobName = vars["job_name"]

	return req, nil
}

func decodeCreateJobBuildRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreateJobBuildRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.JobName = vars["job_name"]
	err := json.NewDecoder(r.Body).Decode(&req.Build)

	return req, err
}

func decodeUpdateJobBuildRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.UpdateJobBuildRequest
	vars := mux.Vars(r)
	bid, _ := strconv.Atoi(vars["build_id"])
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.JobName = vars["job_name"]
	req.BuildID = uint32(bid)

	err := json.NewDecoder(r.Body).Decode(&req.Build)

	return req, err
}

func decodeDeleteJobBuildRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.DeleteJobBuildRequest
	vars := mux.Vars(r)
	bid, _ := strconv.Atoi(vars["build_id"])
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.JobName = vars["job_name"]
	req.BuildID = uint32(bid)

	return req, nil
}

func decodeCreateResourceVersionRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreateResourceVersionRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.ResourceCanonical = vars["resource_canonical"]
	err := json.NewDecoder(r.Body).Decode(&req.Version)

	return req, err
}

func decodeListResourceVersionsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.ListResourceVersionsRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.ResourceCanonical = vars["resource_canonical"]

	return req, nil
}

func decodeGetPipelineResourceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.GetPipelineResourceRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.ResourceCanonical = vars["resource_canonical"]

	return req, nil
}

func decodeUpdatePipelineResourceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.UpdatePipelineResourceRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.ResourceCanonical = vars["resource_canonical"]
	err := json.NewDecoder(r.Body).Decode(&req.Resource)

	return req, err
}

func decodeTriggerPipelineResourceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.TriggerPipelineResourceRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.PipelineName = vars["pipeline_name"]
	req.ResourceCanonical = vars["resource_canonical"]

	return req, nil
}

func decodeGetPipelineImageRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.GetPipelineImageRequest
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
	req.Name = vars["pipeline_name"]
	req.Format = vars["format"]

	return req, nil
}

func decodeCreatePipelineImageRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req transport.CreatePipelineImageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	vars := mux.Vars(r)
	req.TeamCanonical = vars["team_canonical"]
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
