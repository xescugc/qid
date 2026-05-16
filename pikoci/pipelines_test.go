package pikoci_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/pikoci/pikoci/job"
	"github.com/xescugc/pikoci/pikoci/pipeline"
	"github.com/xescugc/pikoci/pikoci/resource"
	"github.com/xescugc/pikoci/pikoci/secret"
	"github.com/xescugc/pikoci/pikoci/sectype"
	"github.com/xescugc/pikoci/pikoci/utils"
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

func TestCreatePipeline_OrderedPlanWithPut(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
resource_type "git" {
  params = ["url"]
  check "exec" {
    path = "echo"
    args = ["check"]
  }
  pull "exec" {
    path = "echo"
    args = ["pull"]
  }
  push "exec" {
    path = "echo"
    args = ["push"]
  }
}

resource "git" "repo" {
  params {
    url = "http://example.com"
  }
}

resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "deploy" {
  get "cron" "timer" {
    trigger = true
  }
  task "build" {
    run "exec" {
      path = "echo"
      args = ["building"]
    }
  }
  put "git" "repo" {
    tag = "latest"
  }
}
`)

	// Expect all the create calls
	s.Pipelines.EXPECT().Create(ctx, "main", gomock.Any()).Return(uint32(1), nil)
	s.Jobs.EXPECT().Create(ctx, "main", "test-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, j job.Job) (uint32, error) {
			// Verify the plan is ordered: get, task, put
			require.Len(t, j.Plan, 3)
			assert.Equal(t, job.StepTypeGet, j.Plan[0].Type)
			assert.Equal(t, "timer", j.Plan[0].Get.Name)
			assert.Equal(t, job.StepTypeTask, j.Plan[1].Type)
			assert.Equal(t, "build", j.Plan[1].Task.Name)
			assert.Equal(t, job.StepTypePut, j.Plan[2].Type)
			assert.Equal(t, "repo", j.Plan[2].Put.Name)
			assert.Equal(t, "latest", j.Plan[2].Put.Params["tag"])
			return uint32(1), nil
		})
	s.ResourceTypes.EXPECT().Create(ctx, "main", "test-pipeline", gomock.Any()).Return(uint32(1), nil)
	s.Resources.EXPECT().Create(ctx, "main", "test-pipeline", gomock.Any()).Return(uint32(1), nil).Times(2)
	s.Pipelines.EXPECT().Find(ctx, "main", "test-pipeline").Return(&pipeline.Pipeline{ID: 1, Name: "test-pipeline"}, nil)

	_, err := s.S.CreatePipeline(ctx, "main", "test-pipeline", hclConfig, nil)
	require.NoError(t, err)
}

func TestCreatePipeline_BackwardsCompat_GetThenTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "test" {
  get "cron" "timer" {
    trigger = true
  }
  task "echo" {
    run "exec" {
      path = "echo"
      args = ["hello"]
    }
  }
}
`)

	s.Pipelines.EXPECT().Create(ctx, "main", gomock.Any()).Return(uint32(1), nil)
	s.Jobs.EXPECT().Create(ctx, "main", "compat-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, j job.Job) (uint32, error) {
			// Verify backwards compat: get before task
			require.Len(t, j.Plan, 2)
			assert.Equal(t, job.StepTypeGet, j.Plan[0].Type)
			assert.Equal(t, job.StepTypeTask, j.Plan[1].Type)
			return uint32(1), nil
		})
	s.Resources.EXPECT().Create(ctx, "main", "compat-pipeline", gomock.Any()).Return(uint32(1), nil)
	s.Pipelines.EXPECT().Find(ctx, "main", "compat-pipeline").Return(&pipeline.Pipeline{ID: 1, Name: "compat-pipeline"}, nil)

	_, err := s.S.CreatePipeline(ctx, "main", "compat-pipeline", hclConfig, nil)
	require.NoError(t, err)
}

func TestCreatePipeline_HCLFunctions(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
variable "greeting" {
  type    = string
  default = "hello"
}

resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "test" {
  get "cron" "timer" {
    trigger = true
  }
  task "echo" {
    run "exec" {
      path = "echo"
      args = [upper(var.greeting), join(",", ["a", "b", "c"])]
    }
  }
}
`)

	s.Pipelines.EXPECT().Create(ctx, "main", gomock.Any()).Return(uint32(1), nil)
	s.Jobs.EXPECT().Create(ctx, "main", "func-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, j job.Job) (uint32, error) {
			require.Len(t, j.Plan, 2)
			assert.Equal(t, job.StepTypeTask, j.Plan[1].Type)
			require.Len(t, j.Plan[1].Task.Run.Args, 2)
			assert.Equal(t, "HELLO", j.Plan[1].Task.Run.Args[0])
			assert.Equal(t, "a,b,c", j.Plan[1].Task.Run.Args[1])
			return uint32(1), nil
		})
	s.Resources.EXPECT().Create(ctx, "main", "func-pipeline", gomock.Any()).Return(uint32(1), nil)
	s.Pipelines.EXPECT().Find(ctx, "main", "func-pipeline").Return(&pipeline.Pipeline{ID: 1, Name: "func-pipeline"}, nil)

	_, err := s.S.CreatePipeline(ctx, "main", "func-pipeline", hclConfig, nil)
	require.NoError(t, err)
}

func TestCreatePipeline_SourceAndInlineConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
resource_type "my-git" {
  source = "pikoci://git"
  params = ["url"]
  check "exec" {
    path = "echo"
    args = ["check"]
  }
  pull "exec" { }
  push "exec" { }
}

resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "test" {
  get "cron" "timer" { trigger = true }
  task "echo" {
    run "exec" {
      path = "echo"
      args = ["hello"]
    }
  }
}
`)

	_, err := s.S.CreatePipeline(ctx, "main", "conflict-pipeline", hclConfig, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both source and inline commands")
}

func TestCreatePipeline_WithTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "test" {
  get "cron" "timer" {
    trigger = true
    timeout = "2m"
  }
  task "build" {
    timeout = "10m"
    run "exec" {
      path = "echo"
      args = ["building"]
    }
  }
}
`)

	s.Pipelines.EXPECT().Create(ctx, "main", gomock.Any()).Return(uint32(1), nil)
	s.Jobs.EXPECT().Create(ctx, "main", "timeout-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, j job.Job) (uint32, error) {
			require.Len(t, j.Plan, 2)
			assert.Equal(t, 2*time.Minute, j.Plan[0].Timeout)
			assert.Equal(t, 10*time.Minute, j.Plan[1].Timeout)
			return uint32(1), nil
		})
	s.Resources.EXPECT().Create(ctx, "main", "timeout-pipeline", gomock.Any()).Return(uint32(1), nil)
	s.Pipelines.EXPECT().Find(ctx, "main", "timeout-pipeline").Return(&pipeline.Pipeline{ID: 1, Name: "timeout-pipeline"}, nil)

	_, err := s.S.CreatePipeline(ctx, "main", "timeout-pipeline", hclConfig, nil)
	require.NoError(t, err)
}

func TestCreatePipeline_InvalidTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "test" {
  get "cron" "timer" {
    trigger = true
  }
  task "build" {
    timeout = "invalid"
    run "exec" {
      path = "echo"
      args = ["building"]
    }
  }
}
`)

	_, err := s.S.CreatePipeline(ctx, "main", "invalid-timeout-pipeline", hclConfig, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timeout")
}

func TestCreatePipeline_WithAttempts(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "test" {
  get "cron" "timer" {
    trigger  = true
    attempts = 3
  }
  task "build" {
    attempts = 2
    run "exec" {
      path = "echo"
      args = ["building"]
    }
  }
}
`)

	s.Pipelines.EXPECT().Create(ctx, "main", gomock.Any()).Return(uint32(1), nil)
	s.Jobs.EXPECT().Create(ctx, "main", "attempts-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, j job.Job) (uint32, error) {
			require.Len(t, j.Plan, 2)
			assert.Equal(t, 3, j.Plan[0].Attempts)
			assert.Equal(t, 2, j.Plan[1].Attempts)
			return uint32(1), nil
		})
	s.Resources.EXPECT().Create(ctx, "main", "attempts-pipeline", gomock.Any()).Return(uint32(1), nil)
	s.Pipelines.EXPECT().Find(ctx, "main", "attempts-pipeline").Return(&pipeline.Pipeline{ID: 1, Name: "attempts-pipeline"}, nil)

	_, err := s.S.CreatePipeline(ctx, "main", "attempts-pipeline", hclConfig, nil)
	require.NoError(t, err)
}

func TestCreatePipeline_InvalidAttempts(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "test" {
  task "build" {
    attempts = -1
    run "exec" {
      path = "echo"
      args = ["building"]
    }
  }
}
`)

	_, err := s.S.CreatePipeline(ctx, "main", "invalid-attempts-pipeline", hclConfig, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid attempts")
}

func TestCreatePipeline_SourceResolution(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
resource_type "my-git" {
  source = "pikoci://git"
}

resource "my-git" "repo" {
  params {
    url  = "https://example.com/repo.git"
    name = "repo"
  }
}

job "test" {
  get "my-git" "repo" { trigger = true }
  task "echo" {
    run "exec" {
      path = "echo"
      args = ["hello"]
    }
  }
}
`)

	s.Pipelines.EXPECT().Create(ctx, "main", gomock.Any()).Return(uint32(1), nil)
	s.Jobs.EXPECT().Create(ctx, "main", "source-pipeline", gomock.Any()).Return(uint32(1), nil)
	s.ResourceTypes.EXPECT().Create(ctx, "main", "source-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, rt interface{}) (uint32, error) {
			return uint32(1), nil
		})
	s.Resources.EXPECT().Create(ctx, "main", "source-pipeline", gomock.Any()).Return(uint32(1), nil)
	s.Pipelines.EXPECT().Find(ctx, "main", "source-pipeline").Return(&pipeline.Pipeline{ID: 1, Name: "source-pipeline"}, nil)

	_, err := s.S.CreatePipeline(ctx, "main", "source-pipeline", hclConfig, nil)
	require.NoError(t, err)
}

func TestCreatePipeline_WithSecrets(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
secret_type "vault" {
  params = ["path"]
  get "exec" {
    path = "/bin/sh"
    args = ["-ec", "echo '{\"username\":\"admin\"}'"]
  }
}

secret "vault" "db-creds" {
  path = "secret/data/db"
}

resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "deploy" {
  get "cron" "timer" {
    trigger = true
  }
  task "migrate" {
    secrets = ["vault.db-creds"]
    run "exec" {
      path = "make"
      args = ["migrate"]
    }
  }
}
`)

	s.Pipelines.EXPECT().Create(ctx, "main", gomock.Any()).Return(uint32(1), nil)
	s.Jobs.EXPECT().Create(ctx, "main", "secrets-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, j job.Job) (uint32, error) {
			require.Len(t, j.Plan, 2)
			assert.Equal(t, job.StepTypeTask, j.Plan[1].Type)
			assert.Equal(t, []string{"vault.db-creds"}, j.Plan[1].Secrets)
			return uint32(1), nil
		})
	s.Resources.EXPECT().Create(ctx, "main", "secrets-pipeline", gomock.Any()).Return(uint32(1), nil)
	s.SecretTypes.EXPECT().Create(ctx, "main", "secrets-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, st sectype.SecretType) (uint32, error) {
			assert.Equal(t, "vault", st.Name)
			assert.Equal(t, []string{"path"}, st.Params)
			assert.Equal(t, "exec", st.Get.Runner)
			return uint32(1), nil
		})
	s.Secrets.EXPECT().Create(ctx, "main", "secrets-pipeline", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn string, sec secret.Secret) (uint32, error) {
			assert.Equal(t, "vault", sec.Type)
			assert.Equal(t, "db-creds", sec.Name)
			assert.Equal(t, utils.ResourceCanonical("vault", "db-creds"), sec.Canonical)
			assert.Equal(t, "secret/data/db", sec.Params["path"])
			return uint32(1), nil
		})
	s.Pipelines.EXPECT().Find(ctx, "main", "secrets-pipeline").Return(&pipeline.Pipeline{ID: 1, Name: "secrets-pipeline"}, nil)

	_, err := s.S.CreatePipeline(ctx, "main", "secrets-pipeline", hclConfig, nil)
	require.NoError(t, err)
}

func TestCreatePipeline_SecretTypeSourceAndInlineConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hclConfig := []byte(`
secret_type "vault" {
  source = "pikoci://vault"
  params = ["path"]
  get "exec" {
    path = "/bin/sh"
    args = ["-ec", "echo test"]
  }
}

resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "test" {
  get "cron" "timer" { trigger = true }
  task "echo" {
    run "exec" {
      path = "echo"
      args = ["hello"]
    }
  }
}
`)

	_, err := s.S.CreatePipeline(ctx, "main", "conflict-secret-pipeline", hclConfig, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both source and inline commands")
}
