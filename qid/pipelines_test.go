package qid_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/resource"
	"go.uber.org/mock/gomock"
)

func TestGetPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	expected := &pipeline.Pipeline{
		ID:   1,
		Name: "my-pipeline",
		Jobs: []job.Job{{ID: 1, Name: "echo"}},
		Resources: []resource.Resource{{ID: 1, Canonical: "cron.my-cron"}},
	}
	s.Pipelines.EXPECT().Find(ctx, "main", "my-pipeline").Return(expected, nil)

	pp, err := s.S.GetPipeline(ctx, "main", "my-pipeline")
	require.NoError(t, err)
	assert.Equal(t, expected, pp)
}

func TestGetPipeline_InvalidCanonical(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	_, err := s.S.GetPipeline(ctx, "INVALID", "my-pipeline")
	require.Error(t, err)

	_, err = s.S.GetPipeline(ctx, "main", "INVALID")
	require.Error(t, err)
}

func TestListPipelines(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	expected := []*pipeline.Pipeline{
		{ID: 1, Name: "pipeline-a"},
		{ID: 2, Name: "pipeline-b"},
	}
	s.Pipelines.EXPECT().Filter(ctx, "main").Return(expected, nil)

	pps, err := s.S.ListPipelines(ctx, "main")
	require.NoError(t, err)
	assert.Len(t, pps, 2)
}

func TestListPipelines_InvalidCanonical(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	_, err := s.S.ListPipelines(ctx, "INVALID")
	require.Error(t, err)
}

func TestDeletePipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	s.Resources.EXPECT().Filter(ctx, "main", "my-pipeline").Return([]*resource.Resource{
		{ID: 1, Canonical: "cron.my-cron", CronID: 5},
	}, nil)
	s.Pipelines.EXPECT().Delete(ctx, "main", "my-pipeline").Return(nil)

	err := s.S.DeletePipeline(ctx, "main", "my-pipeline")
	require.NoError(t, err)
}

func TestDeletePipeline_InvalidCanonical(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	err := s.S.DeletePipeline(ctx, "INVALID", "my-pipeline")
	require.Error(t, err)

	err = s.S.DeletePipeline(ctx, "main", "INVALID")
	require.Error(t, err)
}
