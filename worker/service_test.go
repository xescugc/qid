package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/pikoci/pikoci/build"
	"github.com/xescugc/pikoci/pikoci/job"
	"github.com/xescugc/pikoci/pikoci/mock"
	"github.com/xescugc/pikoci/pikoci/pipeline"
	"github.com/xescugc/pikoci/pikoci/queue"
	"github.com/xescugc/pikoci/pikoci/resource"
	"github.com/xescugc/pikoci/pikoci/restype"
	"github.com/xescugc/pikoci/pikoci/runner"
	"github.com/xescugc/pikoci/pikoci/utils"
	"go.uber.org/mock/gomock"
	"gocloud.dev/pubsub"
)

func newTestWorker(ctrl *gomock.Controller) (*Worker, *mock.Service, *mock.Topic) {
	svc := mock.NewService(ctrl)
	topic := mock.NewTopic(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	w := &Worker{
		pikoci:        svc,
		topic:  topic,
		logger: logger,
	}
	return w, svc, topic
}

func testPipeline() *pipeline.Pipeline {
	return &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "test-job",
				Get: []job.GetStep{
					{
						Type:    "cron",
						Name:    "my-cron",
						Trigger: true,
					},
				},
				Task: []job.TaskStep{
					{
						Name: "echo",
						Run: utils.RunnerCommand{
							Runner: "exec",
							Params: map[string]string{
								"path": "echo",
								"args": "hello",
							},
						},
					},
				},
			},
		},
		Resources: []resource.Resource{
			{
				ID:        1,
				Name:      "my-cron",
				Type:      "cron",
				Canonical: "cron.my-cron",
				Params:    resource.Params{},
			},
		},
		ResourceTypes: []restype.ResourceType{
			{
				ID:     1,
				Name:   "cron",
				Params: []string{},
				Check: utils.RunnerCommand{
					Runner: "exec",
					Params: map[string]string{
						"path": "/bin/sh",
						"args": `-ec 'echo [{"date":"now"}]'`,
					},
				},
				Pull: utils.RunnerCommand{
					Runner: "exec",
					Params: map[string]string{},
				},
			},
		},
		Runners: []runner.Runner{
			{
				ID:   1,
				Name: "exec",
				Run: utils.RunCommand{
					Path: "$path",
					Args: []string{"$args"},
				},
			},
		},
	}
}

func TestProcessJob_Success_TaskOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "echo-job",
	}
	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "echo-job",
				Task: []job.TaskStep{
					{
						Name: "echo",
						Run: utils.RunnerCommand{
							Runner: "exec",
							Params: map[string]string{
								"path": "echo",
								"args": "hello",
							},
						},
					},
				},
			},
		},
		Runners: []runner.Runner{
			{Name: "exec", Run: utils.RunCommand{Path: "$path", Args: []string{"$args"}}},
		},
	}
	cwd := t.TempDir()

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 10}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// UpdateJobBuild: after task step + after marking succeeded
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(10), gomock.Any()).
		Return(nil).Times(2)

	w.processJob(ctx, m, cwd, pp)
}

func TestProcessJob_Success_WithGetAndTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "test-job",
		VersionID:     1,
	}
	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "test-job",
				Get: []job.GetStep{
					{Type: "cron", Name: "my-cron", Trigger: true},
				},
				Task: []job.TaskStep{
					{
						Name: "echo",
						Run:  utils.RunnerCommand{Runner: "exec", Params: map[string]string{"path": "echo", "args": "hello"}},
					},
				},
			},
		},
		Resources: []resource.Resource{
			{ID: 1, Name: "my-cron", Type: "cron", Canonical: "cron.my-cron"},
		},
		ResourceTypes: []restype.ResourceType{
			{
				ID: 1, Name: "cron",
				Pull: utils.RunnerCommand{
					Runner: "exec",
					Params: map[string]string{"path": "echo", "args": "pulling"},
				},
			},
		},
		Runners: []runner.Runner{
			{Name: "exec", Run: utils.RunCommand{Path: "$path", Args: []string{"$args"}}},
		},
	}
	cwd := t.TempDir()

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 10}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)
	svc.EXPECT().ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, "cron.my-cron").
		Return([]*resource.Version{
			{ID: 1, Version: map[string]interface{}{"date": "now"}},
		}, nil)

	// UpdateJobBuild: after get step + after task step + after marking succeeded
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(10), gomock.Any()).
		Return(nil).Times(3)

	w.processJob(ctx, m, cwd, pp)
}

func TestProcessJob_FailedPassedConstraint_NoBuilds(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "downstream-job",
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   2,
				Name: "downstream-job",
				Get: []job.GetStep{
					{
						Type:    "cron",
						Name:    "my-cron",
						Passed:  []string{"upstream-job"},
						Trigger: true,
					},
				},
			},
		},
	}

	cwd := t.TempDir()
	createdBuild := &build.Build{ID: 20}

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(createdBuild, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// Passed check: upstream-job has no builds
	svc.EXPECT().ListJobBuilds(ctx, m.TeamCanonical, m.PipelineName, "upstream-job").
		Return([]*build.Build{}, nil)

	// Build should be deleted (not failed)
	svc.EXPECT().DeleteJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(20)).
		Return(nil)

	w.processJob(ctx, m, cwd, pp)
}

func TestProcessJob_FailedPassedConstraint_NotSucceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "downstream-job",
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   2,
				Name: "downstream-job",
				Get: []job.GetStep{
					{
						Type:    "cron",
						Name:    "my-cron",
						Passed:  []string{"upstream-job"},
						Trigger: true,
					},
				},
			},
		},
	}

	cwd := t.TempDir()
	createdBuild := &build.Build{ID: 21}

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(createdBuild, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// Passed check: upstream-job has a failed build
	svc.EXPECT().ListJobBuilds(ctx, m.TeamCanonical, m.PipelineName, "upstream-job").
		Return([]*build.Build{{ID: 5, Status: build.Failed}}, nil)

	// Build should be deleted
	svc.EXPECT().DeleteJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(21)).
		Return(nil)

	w.processJob(ctx, m, cwd, pp)
}

func TestProcessJob_TaskFailure_RunsHooks(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "failing-job",
		VersionID:     1,
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "failing-job",
				Task: []job.TaskStep{
					{
						Name: "will-fail",
						Run: utils.RunnerCommand{
							Runner: "exec",
							Params: map[string]string{
								"path": "false", // exits with code 1
								"args": "",
							},
						},
						OnFailure: []utils.RunnerCommand{
							{
								Runner: "exec",
								Params: map[string]string{
									"path": "echo",
									"args": "task failed",
								},
							},
						},
					},
				},
				OnFailure: []utils.RunnerCommand{
					{
						Runner: "exec",
						Params: map[string]string{
							"path": "echo",
							"args": "job failed",
						},
					},
				},
				Ensure: []utils.RunnerCommand{
					{
						Runner: "exec",
						Params: map[string]string{
							"path": "echo",
							"args": "always runs",
						},
					},
				},
			},
		},
		Runners: []runner.Runner{
			{
				ID:   1,
				Name: "exec",
				Run: utils.RunCommand{
					Path: "$path",
					Args: []string{"$args"},
				},
			},
		},
	}

	cwd := t.TempDir()
	createdBuild := &build.Build{ID: 30}

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(createdBuild, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// failBuild (task fails) + task on_failure hook update + job on_failure hook update + job ensure hook update = 4 updates
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(30), gomock.Any()).
		Return(nil).Times(4)

	w.processJob(ctx, m, cwd, pp)
}

func TestProcessJob_TriggersDownstream(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, topic := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "upstream-job",
		VersionID:     1,
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "upstream-job",
				Task: []job.TaskStep{
					{
						Name: "echo",
						Run: utils.RunnerCommand{
							Runner: "exec",
							Params: map[string]string{
								"path": "echo",
								"args": "hello",
							},
						},
					},
				},
			},
			{
				ID:   2,
				Name: "downstream-job",
				Get: []job.GetStep{
					{
						Type:    "cron",
						Name:    "my-cron",
						Passed:  []string{"upstream-job"},
						Trigger: true,
					},
				},
			},
		},
		Runners: []runner.Runner{
			{
				ID:   1,
				Name: "exec",
				Run: utils.RunCommand{
					Path: "$path",
					Args: []string{"$args"},
				},
			},
		},
	}

	cwd := t.TempDir()
	createdBuild := &build.Build{ID: 40}

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(createdBuild, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// UpdateJobBuild: after task step + after success mark
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(40), gomock.Any()).
		Return(nil).Times(2)

	// Should trigger downstream-job
	topic.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, msg *pubsub.Message) error {
		var body queue.Body
		err := json.Unmarshal(msg.Body, &body)
		require.NoError(t, err)
		assert.Equal(t, "downstream-job", body.JobName)
		assert.Equal(t, "test-pipeline", body.PipelineName)
		assert.Equal(t, "cron.my-cron", body.ResourceCanonical)
		return nil
	})

	w.processJob(ctx, m, cwd, pp)
}

func TestProcessResourceCheck_NewVersions(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, topic := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical:     "main",
		PipelineName:      "test-pipeline",
		ResourceCanonical: "cron.my-cron",
	}
	// Use a pipeline where the check command outputs valid JSON
	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "test-job",
				Get:  []job.GetStep{{Type: "cron", Name: "my-cron", Trigger: true}},
			},
		},
		Resources: []resource.Resource{
			{ID: 1, Name: "my-cron", Type: "cron", Canonical: "cron.my-cron"},
		},
		ResourceTypes: []restype.ResourceType{
			{
				ID: 1, Name: "cron",
				Check: utils.RunnerCommand{
					Runner: "exec",
					Params: map[string]string{
						"path": "/bin/sh",
						"args": `-ec 'printf "[{\"date\":\"now\"}]\n"'`,
					},
				},
			},
		},
		Runners: []runner.Runner{
			{Name: "exec", Run: utils.RunCommand{Path: "$path", Args: []string{"$args"}}},
		},
	}
	cwd := t.TempDir()

	// ListResourceVersions - no existing versions
	svc.EXPECT().ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, "cron.my-cron").
		Return([]*resource.Version{}, nil)

	// CreateResourceVersion for the new version found
	svc.EXPECT().CreateResourceVersion(ctx, m.TeamCanonical, m.PipelineName, "cron.my-cron", gomock.Any()).
		Return(&resource.Version{ID: 1, Version: map[string]interface{}{"date": "now"}}, nil)

	// Should trigger the job that depends on this resource
	topic.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, msg *pubsub.Message) error {
		var body queue.Body
		err := json.Unmarshal(msg.Body, &body)
		require.NoError(t, err)
		assert.Equal(t, "test-job", body.JobName)
		assert.Equal(t, "cron.my-cron", body.ResourceCanonical)
		assert.Equal(t, uint32(1), body.VersionID)
		return nil
	})

	w.processResourceCheck(ctx, m, cwd, pp)
}

func TestCheckPassedConstraints_AllPassed(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "downstream-job",
	}
	b := build.Build{ID: 50}
	j := &job.Job{
		Name: "downstream-job",
		Get: []job.GetStep{
			{
				Type:   "cron",
				Name:   "my-cron",
				Passed: []string{"job-a", "job-b"},
			},
		},
	}

	svc.EXPECT().ListJobBuilds(ctx, m.TeamCanonical, m.PipelineName, "job-a").
		Return([]*build.Build{{ID: 1, Status: build.Succeeded}}, nil)
	svc.EXPECT().ListJobBuilds(ctx, m.TeamCanonical, m.PipelineName, "job-b").
		Return([]*build.Build{{ID: 2, Status: build.Succeeded}}, nil)

	result := w.checkPassedConstraints(ctx, m, &b, j)
	assert.True(t, result)
}

func TestRunHooks(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "test-job",
	}

	pp := testPipeline()
	cwd := t.TempDir()

	b := build.Build{
		ID:     60,
		Status: build.Succeeded,
		Job:    []build.Step{},
	}

	hooks := []utils.RunnerCommand{
		{Runner: "exec", Params: map[string]string{"path": "echo", "args": "hook1"}},
		{Runner: "exec", Params: map[string]string{"path": "echo", "args": "hook2"}},
	}

	// 2 hooks = 2 UpdateJobBuild calls
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(60), gomock.Any()).
		Return(nil).Times(2)

	w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "task-name", hooks, "on_success")

	require.Len(t, b.Job, 2)
	assert.Equal(t, "task-name:0:on_success", b.Job[0].Name)
	assert.Equal(t, "task-name:1:on_success", b.Job[1].Name)
}

func TestRunHooks_SingleHook_NoIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "test-job",
	}

	pp := testPipeline()
	cwd := t.TempDir()

	b := build.Build{ID: 61, Job: []build.Step{}}

	hooks := []utils.RunnerCommand{
		{Runner: "exec", Params: map[string]string{"path": "echo", "args": "only"}},
	}

	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(61), gomock.Any()).
		Return(nil)

	w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "step", hooks, "ensure")

	require.Len(t, b.Job, 1)
	assert.Equal(t, "step:ensure", b.Job[0].Name)
}

func TestRunHooks_JobLevel_NoStepName(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "test-job",
	}

	pp := testPipeline()
	cwd := t.TempDir()

	b := build.Build{ID: 62, Job: []build.Step{}}

	hooks := []utils.RunnerCommand{
		{Runner: "exec", Params: map[string]string{"path": "echo", "args": "job-level"}},
	}

	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(62), gomock.Any()).
		Return(nil)

	w.runHooks(ctx, m, &b, &b.Job, cwd, pp, "", hooks, "on_failure")

	require.Len(t, b.Job, 1)
	assert.Equal(t, "on_failure", b.Job[0].Name)
}

func TestProcessMessage_JobDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "test-job",
	}
	cwd := t.TempDir()

	// GetPipeline returns empty pipeline → CreateJobBuild will be called
	svc.EXPECT().GetPipeline(ctx, m.TeamCanonical, m.PipelineName).
		Return(&pipeline.Pipeline{Name: "test-pipeline"}, nil)

	// CreateJobBuild
	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 1}, nil)

	// GetPipelineJob — no tasks, no gets
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&job.Job{Name: "test-job"}, nil)

	// Succeeded → UpdateJobBuild
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(1), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
			assert.Equal(t, build.Succeeded, b.Status)
			return nil
		})

	w.processMessage(ctx, m, cwd)
}

func TestBuildPullParams_WithVersionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical:     "main",
		PipelineName:      "test-pipeline",
		JobName:           "test-job",
		ResourceCanonical: "cron.my-cron",
		VersionID:         5,
	}
	b := build.Build{ID: 70}

	rt := restype.ResourceType{
		Pull: utils.RunnerCommand{
			Params: map[string]string{},
		},
		Params: []string{"url"},
	}
	r := resource.Resource{
		Canonical: "cron.my-cron",
		Params: resource.Params{
			Params: map[string]string{"url": "http://example.com"},
		},
	}
	g := job.GetStep{}

	svc.EXPECT().ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, "cron.my-cron").
		Return([]*resource.Version{
			{ID: 3, Version: map[string]interface{}{"ref": "abc"}},
			{ID: 5, Version: map[string]interface{}{"ref": "def"}},
		}, nil)

	params := w.buildPullParams(ctx, m, &b, rt, r, g)
	require.NotNil(t, params)
	assert.Equal(t, "def", params["version_ref"])
	assert.Equal(t, "http://example.com", params["param_url"])
}

func TestBuildPullParams_NoVersionID_UsesLatest(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "test-job",
	}
	b := build.Build{ID: 71}

	rt := restype.ResourceType{
		Pull: utils.RunnerCommand{
			Params: map[string]string{},
		},
	}
	r := resource.Resource{Canonical: "cron.my-cron"}
	g := job.GetStep{}

	// Returns versions ordered by ID desc — after Reverse, last becomes first
	svc.EXPECT().ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, "cron.my-cron").
		Return([]*resource.Version{
			{ID: 1, Version: map[string]interface{}{"ref": "old"}},
			{ID: 2, Version: map[string]interface{}{"ref": "latest"}},
		}, nil)

	params := w.buildPullParams(ctx, m, &b, rt, r, g)
	require.NotNil(t, params)
	assert.Equal(t, "latest", params["version_ref"])
}

func TestBuildPullParams_NoVersions_Fails(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "test-job",
	}
	b := build.Build{ID: 72}

	rt := restype.ResourceType{
		Pull: utils.RunnerCommand{Params: map[string]string{}},
	}
	r := resource.Resource{Canonical: "cron.my-cron"}
	g := job.GetStep{}

	svc.EXPECT().ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, "cron.my-cron").
		Return([]*resource.Version{}, nil)

	// Should call failBuild
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(72), gomock.Any()).
		Return(nil)

	params := w.buildPullParams(ctx, m, &b, rt, r, g)
	assert.Nil(t, params)
}

// Silence the unused import warnings
var _ = time.Now
