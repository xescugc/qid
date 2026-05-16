package worker

import (
	"context"
	"encoding/json"
	"fmt"
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
		pikoci: svc,
		topic:  topic,
		logger: logger,
	}
	return w, svc, topic
}


func runnerHook(rc utils.RunnerCommand) job.HookStep {
	return job.HookStep{Type: job.StepTypeRunner, Runner: &rc}
}

func testPipeline() *pipeline.Pipeline {
	return &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "test-job",
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeGet,
						Get:  &job.GetStep{Type: "cron", Name: "my-cron", Trigger: true},
					},
					{
						Type: job.StepTypeTask,
						Task: &job.TaskStep{
							Name: "echo",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"hello"},
								Params: map[string]string{
									"path": "echo",
								},
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
				Params: &resource.Params{},
			},
		},
		ResourceTypes: []restype.ResourceType{
			{
				ID:     1,
				Name:   "cron",
				Params: []string{},
				Check: &utils.RunnerCommand{
					Runner: "exec",
					Args:   []string{"-ec", `echo "[{\"date\":\"now\"}]"`},
					Params: map[string]string{
						"path": "/bin/sh",
					},
				},
				Pull: &utils.RunnerCommand{
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
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeTask,
						Task: &job.TaskStep{
							Name: "echo",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"hello"},
								Params: map[string]string{
									"path": "echo",
								},
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
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeGet,
						Get:  &job.GetStep{Type: "cron", Name: "my-cron", Trigger: true},
					},
					{
						Type: job.StepTypeTask,
						Task: &job.TaskStep{
							Name: "echo",
							Run:  utils.RunnerCommand{Runner: "exec", Args: []string{"hello"}, Params: map[string]string{"path": "echo"}},
						},
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
				Pull: &utils.RunnerCommand{
					Runner: "exec",
					Args:   []string{"pulling"},
					Params: map[string]string{"path": "echo"},
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
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeGet,
						Get: &job.GetStep{
							Type:    "cron",
							Name:    "my-cron",
							Passed:  []string{"upstream-job"},
							Trigger: true,
						},
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
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeGet,
						Get: &job.GetStep{
							Type:    "cron",
							Name:    "my-cron",
							Passed:  []string{"upstream-job"},
							Trigger: true,
						},
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
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeTask,
						Task: &job.TaskStep{
							Name: "will-fail",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Params: map[string]string{
									"path": "false", // exits with code 1
								},
							},
						},
						OnFailure: []job.HookStep{
							runnerHook(utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"task failed"},
								Params: map[string]string{
									"path": "echo",
								},
							}),
						},
					},
				},
				OnFailure: []job.HookStep{
					runnerHook(utils.RunnerCommand{
						Runner: "exec",
						Args:   []string{"job failed"},
						Params: map[string]string{
							"path": "echo",
						},
					}),
				},
				Ensure: []job.HookStep{
					runnerHook(utils.RunnerCommand{
						Runner: "exec",
						Args:   []string{"always runs"},
						Params: map[string]string{
							"path": "echo",
						},
					}),
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
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeTask,
						Task: &job.TaskStep{
							Name: "echo",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"hello"},
								Params: map[string]string{
									"path": "echo",
								},
							},
						},
					},
				},
			},
			{
				ID:   2,
				Name: "downstream-job",
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeGet,
						Get: &job.GetStep{
							Type:    "cron",
							Name:    "my-cron",
							Passed:  []string{"upstream-job"},
							Trigger: true,
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
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeGet,
						Get:  &job.GetStep{Type: "cron", Name: "my-cron", Trigger: true},
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
				Check: &utils.RunnerCommand{
					Runner: "exec",
					Args:   []string{"-ec", `printf "[{\"date\":\"now\"}]\n"`},
					Params: map[string]string{
						"path": "/bin/sh",
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
		Plan: []job.PlanStep{
			{
				Type: job.StepTypeGet,
				Get: &job.GetStep{
					Type:   "cron",
					Name:   "my-cron",
					Passed: []string{"job-a", "job-b"},
				},
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

	hooks := []job.HookStep{
		runnerHook(utils.RunnerCommand{Runner: "exec", Args: []string{"hook1"}, Params: map[string]string{"path": "echo"}}),
		runnerHook(utils.RunnerCommand{Runner: "exec", Args: []string{"hook2"}, Params: map[string]string{"path": "echo"}}),
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

	hooks := []job.HookStep{
		runnerHook(utils.RunnerCommand{Runner: "exec", Args: []string{"only"}, Params: map[string]string{"path": "echo"}}),
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

	hooks := []job.HookStep{
		runnerHook(utils.RunnerCommand{Runner: "exec", Args: []string{"job-level"}, Params: map[string]string{"path": "echo"}}),
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

	// GetPipelineJob — no plan steps
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
		Pull: &utils.RunnerCommand{
			Params: map[string]string{},
		},
		Params: []string{"url"},
	}
	r := resource.Resource{
		Canonical: "cron.my-cron",
		Params: &resource.Params{
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
		Pull: &utils.RunnerCommand{
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
		Pull: &utils.RunnerCommand{Params: map[string]string{}},
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

func TestProcessJob_PutStep_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "deploy-job",
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "deploy-job",
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeTask,
						Task: &job.TaskStep{
							Name: "build",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"building"},
								Params: map[string]string{"path": "echo"},
							},
						},
					},
					{
						Type: job.StepTypePut,
						Put: &job.PutStep{
							Type:   "git",
							Name:   "repo",
							Params: map[string]string{"tag": "latest"},
						},
					},
				},
			},
		},
		Resources: []resource.Resource{
			{ID: 1, Name: "repo", Type: "git", Canonical: "git.repo", Params: &resource.Params{Params: map[string]string{"url": "http://example.com"}}},
		},
		ResourceTypes: []restype.ResourceType{
			{
				ID: 1, Name: "git",
				Params: []string{"url"},
				Push: &utils.RunnerCommand{
					Runner: "exec",
					Args:   []string{"pushing"},
					Params: map[string]string{"path": "echo"},
				},
			},
		},
		Runners: []runner.Runner{
			{Name: "exec", Run: utils.RunCommand{Path: "$path", Args: []string{"$args"}}},
		},
	}
	cwd := t.TempDir()

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 80}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// UpdateJobBuild: after task step + after put step + after marking succeeded
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(80), gomock.Any()).
		Return(nil).Times(3)

	w.processJob(ctx, m, cwd, pp)
}

func TestProcessJob_OrderedPlan_GetTaskPut(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "ordered-job",
		VersionID:     1,
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "ordered-job",
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeGet,
						Get:  &job.GetStep{Type: "cron", Name: "my-cron", Trigger: true},
					},
					{
						Type: job.StepTypeTask,
						Task: &job.TaskStep{
							Name: "build",
							Run:  utils.RunnerCommand{Runner: "exec", Args: []string{"building"}, Params: map[string]string{"path": "echo"}},
						},
					},
					{
						Type: job.StepTypePut,
						Put:  &job.PutStep{Type: "git", Name: "repo"},
					},
				},
			},
		},
		Resources: []resource.Resource{
			{ID: 1, Name: "my-cron", Type: "cron", Canonical: "cron.my-cron"},
			{ID: 2, Name: "repo", Type: "git", Canonical: "git.repo"},
		},
		ResourceTypes: []restype.ResourceType{
			{
				ID: 1, Name: "cron",
				Pull: &utils.RunnerCommand{Runner: "exec", Args: []string{"pulling"}, Params: map[string]string{"path": "echo"}},
			},
			{
				ID: 2, Name: "git",
				Push: &utils.RunnerCommand{Runner: "exec", Args: []string{"pushing"}, Params: map[string]string{"path": "echo"}},
			},
		},
		Runners: []runner.Runner{
			{Name: "exec", Run: utils.RunCommand{Path: "$path", Args: []string{"$args"}}},
		},
	}
	cwd := t.TempDir()

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 90}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)
	svc.EXPECT().ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, "cron.my-cron").
		Return([]*resource.Version{
			{ID: 1, Version: map[string]interface{}{"date": "now"}},
		}, nil)

	// UpdateJobBuild: after get + after task + after put + after success = 4
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(90), gomock.Any()).
		Return(nil).Times(4)

	w.processJob(ctx, m, cwd, pp)
}

func TestProcessJob_TaskTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "timeout-job",
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "timeout-job",
				Plan: []job.PlanStep{
					{
						Type:    job.StepTypeTask,
						Timeout: 2 * time.Second,
						Task: &job.TaskStep{
							Name: "slow-task",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"10"},
								Params: map[string]string{
									"path": "sleep",
								},
							},
						},
						OnFailure: []job.HookStep{
							runnerHook(utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"task failed due to timeout"},
								Params: map[string]string{"path": "echo"},
							}),
						},
						Ensure: []job.HookStep{
							runnerHook(utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"ensure runs"},
								Params: map[string]string{"path": "echo"},
							}),
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
		Return(&build.Build{ID: 100}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// failBuild + on_failure hook + ensure hook = 3 updates
	var capturedBuild build.Build
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(100), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
			capturedBuild = b
			return nil
		}).Times(3)

	w.processJob(ctx, m, cwd, pp)

	assert.Equal(t, build.Failed, capturedBuild.Status)
	// The first step should contain the timeout message
	require.NotEmpty(t, capturedBuild.Steps)
	assert.Contains(t, capturedBuild.Steps[0].Logs, "timed out after 2s")
}

func TestProcessJob_GetTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "get-timeout-job",
		VersionID:     1,
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "get-timeout-job",
				Plan: []job.PlanStep{
					{
						Type:    job.StepTypeGet,
						Timeout: 1 * time.Second,
						Get:     &job.GetStep{Type: "cron", Name: "my-cron", Trigger: true},
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
				Pull: &utils.RunnerCommand{
					Runner: "exec",
					Args:   []string{"10"},
					Params: map[string]string{"path": "sleep"},
				},
			},
		},
		Runners: []runner.Runner{
			{Name: "exec", Run: utils.RunCommand{Path: "$path", Args: []string{"$args"}}},
		},
	}
	cwd := t.TempDir()

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 101}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)
	svc.EXPECT().ListResourceVersions(ctx, m.TeamCanonical, m.PipelineName, "cron.my-cron").
		Return([]*resource.Version{
			{ID: 1, Version: map[string]interface{}{"date": "now"}},
		}, nil)

	// failBuild = 1 update
	var capturedBuild build.Build
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(101), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
			capturedBuild = b
			return nil
		})

	w.processJob(ctx, m, cwd, pp)

	assert.Equal(t, build.Failed, capturedBuild.Status)
	require.NotEmpty(t, capturedBuild.Steps)
	assert.Contains(t, capturedBuild.Steps[0].Logs, "timed out after 1s")
}

func TestProcessJob_NoTimeout_Succeeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "no-timeout-job",
	}
	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "no-timeout-job",
				Plan: []job.PlanStep{
					{
						Type: job.StepTypeTask,
						// Timeout is zero (not set)
						Task: &job.TaskStep{
							Name: "echo",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"hello"},
								Params: map[string]string{"path": "echo"},
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
		Return(&build.Build{ID: 102}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// UpdateJobBuild: after task step + after marking succeeded
	var lastBuild build.Build
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(102), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
			lastBuild = b
			return nil
		}).Times(2)

	w.processJob(ctx, m, cwd, pp)

	assert.Equal(t, build.Succeeded, lastBuild.Status)
}

func TestProcessJob_TaskRetry_SucceedsOnSecondAttempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	cwd := t.TempDir()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "retry-job",
	}

	// Script that fails on first run (no marker file) and succeeds on second (marker exists).
	script := fmt.Sprintf(`#!/bin/sh
if [ -f "%s/marker" ]; then
  echo "success"
  exit 0
else
  touch "%s/marker"
  echo "fail"
  exit 1
fi
`, cwd, cwd)
	scriptPath := cwd + "/retry.sh"
	os.WriteFile(scriptPath, []byte(script), 0755)

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "retry-job",
				Plan: []job.PlanStep{
					{
						Type:     job.StepTypeTask,
						Attempts: 2,
						Task: &job.TaskStep{
							Name: "flaky",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Params: map[string]string{"path": scriptPath},
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

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 200}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// UpdateJobBuild: after task step (success) + after marking succeeded
	var capturedBuild build.Build
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(200), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
			capturedBuild = b
			return nil
		}).Times(2)

	w.processJob(ctx, m, cwd, pp)

	assert.Equal(t, build.Succeeded, capturedBuild.Status)
	require.NotEmpty(t, capturedBuild.Steps)
	assert.Contains(t, capturedBuild.Steps[0].Logs, "attempt 2/2")
	assert.Contains(t, capturedBuild.Steps[0].Logs, "success")
}

func TestProcessJob_TaskRetry_ExhaustsAttempts(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	cwd := t.TempDir()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "exhaust-job",
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "exhaust-job",
				Plan: []job.PlanStep{
					{
						Type:     job.StepTypeTask,
						Attempts: 2,
						Task: &job.TaskStep{
							Name: "always-fail",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Params: map[string]string{"path": "false"},
							},
						},
						OnFailure: []job.HookStep{
							runnerHook(utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"failed after retries"},
								Params: map[string]string{"path": "echo"},
							}),
						},
					},
				},
			},
		},
		Runners: []runner.Runner{
			{Name: "exec", Run: utils.RunCommand{Path: "$path", Args: []string{"$args"}}},
		},
	}

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 201}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// failBuild + on_failure hook = 2 updates
	var capturedBuild build.Build
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(201), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
			capturedBuild = b
			return nil
		}).Times(2)

	w.processJob(ctx, m, cwd, pp)

	assert.Equal(t, build.Failed, capturedBuild.Status)
	require.NotEmpty(t, capturedBuild.Steps)
	assert.Contains(t, capturedBuild.Steps[0].Logs, "attempt 2/2")
}

func TestProcessJob_TaskRetry_WithTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	w, svc, _ := newTestWorker(ctrl)

	ctx := context.Background()
	cwd := t.TempDir()
	m := queue.Body{
		TeamCanonical: "main",
		PipelineName:  "test-pipeline",
		JobName:       "timeout-retry-job",
	}

	pp := &pipeline.Pipeline{
		ID:   1,
		Name: "test-pipeline",
		Jobs: []job.Job{
			{
				ID:   1,
				Name: "timeout-retry-job",
				Plan: []job.PlanStep{
					{
						Type:     job.StepTypeTask,
						Attempts: 2,
						Timeout:  1 * time.Second,
						Task: &job.TaskStep{
							Name: "slow-task",
							Run: utils.RunnerCommand{
								Runner: "exec",
								Args:   []string{"10"},
								Params: map[string]string{"path": "sleep"},
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

	svc.EXPECT().CreateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, gomock.Any()).
		Return(&build.Build{ID: 202}, nil)
	svc.EXPECT().GetPipelineJob(ctx, m.TeamCanonical, m.PipelineName, m.JobName).
		Return(&pp.Jobs[0], nil)

	// failBuild = 1 update
	var capturedBuild build.Build
	svc.EXPECT().UpdateJobBuild(ctx, m.TeamCanonical, m.PipelineName, m.JobName, uint32(202), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
			capturedBuild = b
			return nil
		})

	w.processJob(ctx, m, cwd, pp)

	assert.Equal(t, build.Failed, capturedBuild.Status)
	require.NotEmpty(t, capturedBuild.Steps)
	logs := capturedBuild.Steps[0].Logs
	assert.Contains(t, logs, "attempt 2/2")
	assert.Contains(t, logs, "timed out after 1s")
}

// Silence the unused import warnings
var _ = time.Now
