package qid_test

import (
	"context"

	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/mock"
	"go.uber.org/mock/gomock"
)

type MockService struct {
	Topic         *mock.Topic
	Pipelines     *mock.PipelineRepository
	Jobs          *mock.JobRepository
	Resources     *mock.ResourceRepository
	ResourceTypes *mock.ResourceTypeRepository
	Builds        *mock.BuildRepository

	S qid.Service
}

func newService(ctrl *gomock.Controller) MockService {
	pr := mock.NewPipelineRepository(ctrl)
	jr := mock.NewJobRepository(ctrl)
	rr := mock.NewResourceRepository(ctrl)
	rtr := mock.NewResourceTypeRepository(ctrl)
	br := mock.NewBuildRepository(ctrl)
	t := mock.NewTopic(ctrl)

	return MockService{
		Topic:         t,
		Pipelines:     pr,
		Jobs:          jr,
		Resources:     rr,
		ResourceTypes: rtr,
		Builds:        br,

		S: qid.New(context.TODO(), t, pr, jr, rr, rtr, br, nil),
	}
}
