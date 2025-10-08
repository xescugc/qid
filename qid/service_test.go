package qid_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/queue"
	"go.uber.org/mock/gomock"
	"gocloud.dev/pubsub"
)

func TestCreatePipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()
	ppn := "pipeline-name"

	b, err := os.ReadFile("testdata/pipeline.hcl")
	require.NoError(t, err)

	mvars := map[string]interface{}{
		"repo_name": "repo",
	}

	s.Pipelines.EXPECT().Create(ctx, gomock.Any()).Return(uint32(1), nil)
	s.Jobs.EXPECT().Create(ctx, ppn, gomock.Any()).Return(uint32(1), nil).Times(3)
	s.ResourceTypes.EXPECT().Create(ctx, ppn, gomock.Any()).Return(uint32(1), nil).Times(1)
	s.Resources.EXPECT().Create(ctx, ppn, gomock.Any()).Return(uint32(1), nil).Times(1)

	err = s.S.CreatePipeline(ctx, ppn, b, mvars)
	require.NoError(t, err)
}

func TestTriggerPipelineJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()
	ppn := "pipeline-name"
	jn := "job-name"

	m := queue.Body{
		PipelineName: ppn,
		JobName:      jn,
	}

	mb, err := json.Marshal(m)
	require.NoError(t, err)
	s.Jobs.EXPECT().Find(ctx, ppn, jn).Return(&job.Job{ID: 2}, nil)
	s.Topic.EXPECT().Send(ctx, &pubsub.Message{
		Body: mb,
	}).Return(nil)

	err = s.S.TriggerPipelineJob(ctx, ppn, jn)
	require.NoError(t, err)
}

func TestGetPipelineJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()
	ppn := "pipeline-name"
	jn := "job-name"
	rj := &job.Job{ID: 2}

	s.Jobs.EXPECT().Find(ctx, ppn, jn).Return(rj, nil)

	j, err := s.S.GetPipelineJob(ctx, ppn, jn)
	require.NoError(t, err)
	assert.Equal(t, rj, j)
}
