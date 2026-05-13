package qid_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/qid/qid/job"
	"go.uber.org/mock/gomock"
)

func TestGetPipelineJob_InvalidCanonical(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	_, err := s.S.GetPipelineJob(ctx, "INVALID", "pp", "jn")
	require.Error(t, err)

	_, err = s.S.GetPipelineJob(ctx, "main", "INVALID", "jn")
	require.Error(t, err)

	_, err = s.S.GetPipelineJob(ctx, "main", "pp", "INVALID")
	require.Error(t, err)
}

func TestTriggerPipelineJob_InvalidCanonical(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	err := s.S.TriggerPipelineJob(ctx, "INVALID", "pp", "jn")
	require.Error(t, err)

	err = s.S.TriggerPipelineJob(ctx, "main", "INVALID", "jn")
	require.Error(t, err)

	err = s.S.TriggerPipelineJob(ctx, "main", "pp", "INVALID")
	require.Error(t, err)
}

func TestGetPipelineJob_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	s.Jobs.EXPECT().Find(ctx, "main", "pp", "jn").Return(nil, assert.AnError)

	_, err := s.S.GetPipelineJob(ctx, "main", "pp", "jn")
	require.Error(t, err)
}

func TestTriggerPipelineJob_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	s.Jobs.EXPECT().Find(ctx, "main", "pp", "jn").Return(nil, assert.AnError)

	err := s.S.TriggerPipelineJob(ctx, "main", "pp", "jn")
	require.Error(t, err)
}

func TestTriggerPipelineJob_SendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	s.Jobs.EXPECT().Find(ctx, "main", "pp", "jn").Return(&job.Job{ID: 1}, nil)
	s.Topic.EXPECT().Send(ctx, gomock.Any()).Return(assert.AnError)

	err := s.S.TriggerPipelineJob(ctx, "main", "pp", "jn")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to Trigger Job")
}
