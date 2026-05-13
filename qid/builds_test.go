package qid_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/qid/qid/build"
	"go.uber.org/mock/gomock"
)

func TestCreateJobBuild(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	s.Builds.EXPECT().Create(ctx, "main", "my-pipeline", "my-job", gomock.Any()).Return(uint32(1), nil)

	b, err := s.S.CreateJobBuild(ctx, "main", "my-pipeline", "my-job", build.Build{Status: build.Started})
	require.NoError(t, err)
	assert.Equal(t, uint32(1), b.ID)
}

func TestCreateJobBuild_InvalidCanonical(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	_, err := s.S.CreateJobBuild(ctx, "INVALID", "my-pipeline", "my-job", build.Build{})
	require.Error(t, err)

	_, err = s.S.CreateJobBuild(ctx, "main", "INVALID", "my-job", build.Build{})
	require.Error(t, err)

	_, err = s.S.CreateJobBuild(ctx, "main", "my-pipeline", "INVALID", build.Build{})
	require.Error(t, err)
}

func TestListJobBuilds(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	// Builds returned in ascending order from DB, reversed by service
	s.Builds.EXPECT().Filter(ctx, "main", "my-pipeline", "my-job").Return([]*build.Build{
		{ID: 1, Status: build.Succeeded},
		{ID: 2, Status: build.Started},
	}, nil)

	builds, err := s.S.ListJobBuilds(ctx, "main", "my-pipeline", "my-job")
	require.NoError(t, err)
	require.Len(t, builds, 2)
	// Should be reversed (newest first)
	assert.Equal(t, uint32(2), builds[0].ID)
	assert.Equal(t, uint32(1), builds[1].ID)
}

func TestUpdateJobBuild(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	s.Builds.EXPECT().Update(ctx, "main", "my-pipeline", "my-job", uint32(1), gomock.Any()).Return(nil)

	err := s.S.UpdateJobBuild(ctx, "main", "my-pipeline", "my-job", 1, build.Build{Status: build.Succeeded})
	require.NoError(t, err)
}

func TestDeleteJobBuild(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	s.Builds.EXPECT().Delete(ctx, "main", "my-pipeline", "my-job", uint32(1)).Return(nil)

	err := s.S.DeleteJobBuild(ctx, "main", "my-pipeline", "my-job", 1)
	require.NoError(t, err)
}
