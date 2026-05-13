package http

import (
	"encoding/json"
	"net/http"

	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/user"
)

type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type UserLoginResponse struct {
	Err  string `json:"error,omitempty"`
	Data struct {
		User *user.WithMemberships `json:"user,omitempty"`
		JWT  string                `json:"jwt,omitempty"`
	} `json:"data,omitempty"`
}

func (r UserLoginResponse) Error() string {
	return r.Err
}

func userLogin(s qid.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx = r.Context()
			req UserLoginRequest
		)
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			encodeResponse(UserLoginResponse{Err: err.Error()}, w)
			return
		}

		u, jwt, err := s.UserLogin(ctx, req.Username, req.Password)
		var errs string
		if err != nil {
			errs = err.Error()
		}

		resp := UserLoginResponse{
			Data: struct {
				User *user.WithMemberships `json:"user,omitempty"`
				JWT  string                `json:"jwt,omitempty"`
			}{
				User: u,
				JWT:  jwt,
			},
			Err: errs,
		}
		encodeResponse(resp, w)
	}
}

type ListUsersResponse struct {
	Err   string       `json:"error,omitempty"`
	Users []*user.User `json:"data,omitempty"`
}

func (r ListUsersResponse) Error() string {
	return r.Err
}

func listUsers(s qid.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx = r.Context()
		)
		us, err := s.ListUsers(ctx)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		encodeResponse(ListUsersResponse{Users: us, Err: errs}, w)
	}
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsHash   bool   `json:"is_hash"`
}
type CreateUserResponse struct {
	User *user.User `json:"data,omitempty"`
	Err  string     `json:"error,omitempty"`
}

func (r CreateUserResponse) Error() string {
	return r.Err
}

func createUser(s qid.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req CreateUserRequest
			ctx = r.Context()
		)
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			encodeResponse(CreateUserResponse{Err: err.Error()}, w)
			return
		}
		u, err := s.CreateUser(ctx, user.User{Username: req.Username, Password: req.Password}, req.IsHash)
		var errs string
		if err != nil {
			errs = err.Error()
		}
		encodeResponse(CreateUserResponse{User: u, Err: errs}, w)
	}
}
