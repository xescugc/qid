package qid_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/queue"
	"go.uber.org/mock/gomock"
	"gocloud.dev/pubsub"
)

func TestCreatePipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()
	ppn := "pipeline-name"

	b, err := os.ReadFile("testdata/resource_pipeline.hcl")
	require.NoError(t, err)

	var pp pipeline.Pipeline
	err = hclsimple.Decode("pipeline.hcl", b, nil, &pp)
	require.NoError(t, err)

	pp.Name = ppn

	s.Pipelines.EXPECT().Create(ctx, pp).Return(uint32(1), nil)
	for _, j := range pp.Jobs {
		s.Jobs.EXPECT().Create(ctx, ppn, j).Return(uint32(1), nil)
	}
	for _, rt := range pp.ResourceTypes {
		s.ResourceTypes.EXPECT().Create(ctx, ppn, rt).Return(uint32(1), nil)
	}
	for _, r := range pp.Resources {
		s.Resources.EXPECT().Create(ctx, ppn, r).Return(uint32(1), nil)
	}

	err = s.S.CreatePipeline(ctx, ppn, b)
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
