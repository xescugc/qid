package qid_test

import (
	"context"

	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/mock"
	"go.uber.org/mock/gomock"
)

type MockService struct {
	Topic         *mock.Topic
	Users         *mock.UserRepository
	Teams         *mock.TeamRepository
	Pipelines     *mock.PipelineRepository
	Jobs          *mock.JobRepository
	Resources     *mock.ResourceRepository
	ResourceTypes *mock.ResourceTypeRepository
	Builds        *mock.BuildRepository
	Runners       *mock.RunnerRepository

	S qid.Service
}

func newService(ctrl *gomock.Controller) MockService {
	ur := mock.NewUserRepository(ctrl)
	tr := mock.NewTeamRepository(ctrl)
	pr := mock.NewPipelineRepository(ctrl)
	jr := mock.NewJobRepository(ctrl)
	rr := mock.NewResourceRepository(ctrl)
	rtr := mock.NewResourceTypeRepository(ctrl)
	br := mock.NewBuildRepository(ctrl)
	rur := mock.NewRunnerRepository(ctrl)
	t := mock.NewTopic(ctrl)

	return MockService{
		Topic:         t,
		Users:         ur,
		Teams:         tr,
		Pipelines:     pr,
		Jobs:          jr,
		Resources:     rr,
		ResourceTypes: rtr,
		Builds:        br,
		Runners:       rur,

		S: qid.New(context.TODO(), t, ur, tr, pr, jr, rr, rtr, br, rur, []byte("test-secret"), nil),
	}
}
