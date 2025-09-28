package qid_test

import (
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/mock"
	"go.uber.org/mock/gomock"
)

type MockService struct {
	Queue     *mock.Queue
	Pipelines *mock.PipelineRepository
	Jobs      *mock.JobRepository

	S qid.Service
}

func newService(ctrl *gomock.Controller) MockService {
	pr := mock.NewPipelineRepository(ctrl)
	jr := mock.NewJobRepository(ctrl)
	q := mock.NewQueue(ctrl)

	return MockService{
		Queue:     q,
		Pipelines: pr,
		Jobs:      jr,

		S: qid.New(q, pr, jr),
	}
}
