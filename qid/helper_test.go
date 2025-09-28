package qid_test

import (
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/mock"
	"go.uber.org/mock/gomock"
)

type MockService struct {
	Topic     *mock.Topic
	Pipelines *mock.PipelineRepository
	Jobs      *mock.JobRepository

	S qid.Service
}

func newService(ctrl *gomock.Controller) MockService {
	pr := mock.NewPipelineRepository(ctrl)
	jr := mock.NewJobRepository(ctrl)
	t := mock.NewTopic(ctrl)

	return MockService{
		Topic:     t,
		Pipelines: pr,
		Jobs:      jr,

		S: qid.New(t, pr, jr),
	}
}
