package backends_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/pikoci/pikoci"
	"github.com/xescugc/pikoci/pikoci/build"
	"github.com/xescugc/pikoci/pikoci/mysql"
	"github.com/xescugc/pikoci/pikoci/mysql/migrate"
	"github.com/xescugc/pikoci/pikoci/queue"
	"github.com/xescugc/pikoci/pikoci/unitwork"
	"github.com/xescugc/pikoci/pikoci/user"
	"github.com/xescugc/pikoci/worker"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/mempubsub"
)

func TestSecretsE2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})).With("service", "test")

	db, err := mysql.New("", 0, "", "", mysql.Options{
		MultiStatements: true,
		ClientFoundRows: true,
		System:          mysql.Mem,
	})
	require.NoError(t, err)

	err = migrate.Migrate(db, mysql.Mem)
	require.NoError(t, err)

	topic, err := pubsub.OpenTopic(ctx, fmt.Sprintf("%s://secrets-test", mempubsub.Scheme))
	require.NoError(t, err)
	defer topic.Shutdown(ctx)

	ur := mysql.NewUserRepository(db)
	tr := mysql.NewTeamRepository(db)
	ppr := mysql.NewPipelineRepository(db)
	jr := mysql.NewJobRepository(db)
	rr := mysql.NewResourceRepository(db, mysql.Mem)
	rt := mysql.NewResourceTypeRepository(db)
	br := mysql.NewBuildRepository(db)
	rur := mysql.NewRunnerRepository(db)
	str := mysql.NewSecretTypeRepository(db)
	suow := unitwork.NewStartUnitOfWork(db, mysql.Mem)

	jwtSecret := []byte("test-secret")
	svc := pikoci.New(ctx, topic, ur, tr, ppr, jr, rr, rt, br, rur, str, suow, jwtSecret, logger)
	svc.StartScheduler(ctx)

	// Migration already creates admin user and "main" team.
	// Create a test user if admin doesn't exist yet (ignore duplicate error).
	_, _ = svc.CreateUser(ctx, user.User{
		FullName: "admin",
		Username: "admin",
		Password: "$2a$14$rwQk8Qvc2rij7qhFO4P1W.OiSF6AkgVU1RCrLaY2wawJcpkPEKwbm",
	}, true)

	// Start worker
	subscription, err := pubsub.OpenSubscription(ctx, fmt.Sprintf("%s://secrets-test", mempubsub.Scheme))
	require.NoError(t, err)
	defer subscription.Shutdown(ctx)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w := worker.New(svc, topic, subscription, logger.With("component", "worker"))
		w.Run(ctx)
	}()

	t.Run("TaskWithInlineSecretType", func(t *testing.T) {
		// This pipeline has an inline secret_type that echoes JSON,
		// and a task that prints the secret env vars to stdout.
		hclConfig := []byte(`
secret_type "mock-vault" {
  params = ["path"]
  get "exec" {
    path = "/bin/sh"
    args = ["-ec", "echo '{\"username\":\"admin\",\"password\":\"s3cret\"}'"]
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
  task "use-secrets" {
    secrets = {
      "mock-vault" = "secret/data/db"
    }
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "echo secret_username=$secret_username secret_password=$secret_password"]
    }
  }
}
`)
		pp, err := svc.CreatePipeline(ctx, "main", "secrets-e2e", hclConfig, nil)
		require.NoError(t, err)
		require.NotNil(t, pp)
		assert.Len(t, pp.SecretTypes, 1)
		assert.Equal(t, "mock-vault", pp.SecretTypes[0].Name)
		// Trigger resource check to create a version (required by the get step)
		err = svc.TriggerPipelineResource(ctx, "main", "secrets-e2e", "cron.timer")
		require.NoError(t, err)

		// Wait for the resource version to be created
		require.Eventually(t, func() bool {
			vers, err := svc.ListResourceVersions(ctx, "main", "secrets-e2e", "cron.timer")
			return err == nil && len(vers) > 0
		}, 10*time.Second, 200*time.Millisecond)

		// Wait for the build triggered by the resource to finish
		var builds []*build.Build
		require.Eventually(t, func() bool {
			builds, err = svc.ListJobBuilds(ctx, "main", "secrets-e2e", "deploy")
			if err != nil || len(builds) == 0 {
				return false
			}
			return builds[0].Status != build.Started
		}, 15*time.Second, 200*time.Millisecond)

		require.NotEmpty(t, builds)
		b := builds[0]
		assert.Equal(t, build.Succeeded, b.Status, "build should succeed, error: %s", b.Error)

		// Verify a secret step exists (emitted before the task)
		var secretStep *build.Step
		var taskStep *build.Step
		for i, s := range b.Steps {
			if s.Type == "secret" && s.Name == "use-secrets" {
				secretStep = &b.Steps[i]
			}
			if s.Type == "task" && s.Name == "use-secrets" {
				taskStep = &b.Steps[i]
			}
		}
		require.NotNil(t, secretStep, "secret step should exist in build steps")
		assert.Empty(t, secretStep.Logs, "secret step logs should be empty on success")

		require.NotNil(t, taskStep, "task step 'use-secrets' should exist in build steps")
		assert.True(t, strings.Contains(taskStep.Logs, "secret_username=admin"), "logs should contain secret_username=admin, got: %s", taskStep.Logs)
		assert.True(t, strings.Contains(taskStep.Logs, "secret_password=s3cret"), "logs should contain secret_password=s3cret, got: %s", taskStep.Logs)
	})

	t.Run("TaskWithSourceSecretType", func(t *testing.T) {
		// Uses the built-in "file" secret_type via source
		// Create a temp file with JSON secrets
		tmpDir := t.TempDir()
		secretFile := tmpDir + "/secret.json"
		err := os.WriteFile(secretFile, []byte(`{"api_key":"abc123","api_secret":"xyz789"}`), 0644)
		require.NoError(t, err)

		hclConfig := []byte(fmt.Sprintf(`
secret_type "my-file" {
  source = "pikoci://file"
}


resource "cron" "timer" {
  check_interval = "@every 1h"
  params {}
}

job "deploy" {
  get "cron" "timer" {
    trigger = true
  }
  task "use-file-secrets" {
    secrets = {
      "my-file" = "%s"
    }
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "echo api_key=$secret_api_key api_secret=$secret_api_secret"]
    }
  }
}
`, secretFile))

		pp, err := svc.CreatePipeline(ctx, "main", "secrets-file-e2e", hclConfig, nil)
		require.NoError(t, err)
		require.NotNil(t, pp)
		assert.Len(t, pp.SecretTypes, 1)
		assert.Equal(t, "my-file", pp.SecretTypes[0].Name)
		assert.Equal(t, "pikoci://file", pp.SecretTypes[0].Source)

		// Trigger resource check to create a version
		err = svc.TriggerPipelineResource(ctx, "main", "secrets-file-e2e", "cron.timer")
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			vers, err := svc.ListResourceVersions(ctx, "main", "secrets-file-e2e", "cron.timer")
			return err == nil && len(vers) > 0
		}, 10*time.Second, 200*time.Millisecond)

		// Wait for the build triggered by the resource to finish
		var builds []*build.Build
		require.Eventually(t, func() bool {
			builds, err = svc.ListJobBuilds(ctx, "main", "secrets-file-e2e", "deploy")
			if err != nil || len(builds) == 0 {
				return false
			}
			return builds[0].Status != build.Started
		}, 15*time.Second, 200*time.Millisecond)

		require.NotEmpty(t, builds)
		b := builds[0]
		assert.Equal(t, build.Succeeded, b.Status, "build should succeed, error: %s", b.Error)

		var taskStep *build.Step
		for i, s := range b.Steps {
			if s.Type == "task" && s.Name == "use-file-secrets" {
				taskStep = &b.Steps[i]
				break
			}
		}
		require.NotNil(t, taskStep, "task step 'use-file-secrets' should exist in build steps")
		assert.True(t, strings.Contains(taskStep.Logs, "api_key=abc123"), "logs should contain api_key=abc123, got: %s", taskStep.Logs)
		assert.True(t, strings.Contains(taskStep.Logs, "api_secret=xyz789"), "logs should contain api_secret=xyz789, got: %s", taskStep.Logs)
	})

	t.Run("SecretFetchFailure", func(t *testing.T) {
		// The secret_type get command fails, so the step should fail
		hclConfig := []byte(`
secret_type "broken" {
  params = ["path"]
  get "exec" {
    path = "/bin/sh"
    args = ["-ec", "exit 1"]
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
  task "will-fail" {
    secrets = {
      "broken" = "does/not/matter"
    }
    run "exec" {
      path = "echo"
      args = ["should not reach here"]
    }
  }
}
`)

		_, err := svc.CreatePipeline(ctx, "main", "secrets-fail-e2e", hclConfig, nil)
		require.NoError(t, err)

		// Trigger resource check to create a version
		err = svc.TriggerPipelineResource(ctx, "main", "secrets-fail-e2e", "cron.timer")
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			vers, err := svc.ListResourceVersions(ctx, "main", "secrets-fail-e2e", "cron.timer")
			return err == nil && len(vers) > 0
		}, 10*time.Second, 200*time.Millisecond)

		var builds []*build.Build
		require.Eventually(t, func() bool {
			builds, err = svc.ListJobBuilds(ctx, "main", "secrets-fail-e2e", "deploy")
			if err != nil || len(builds) == 0 {
				return false
			}
			return builds[0].Status != build.Started
		}, 15*time.Second, 200*time.Millisecond)

		require.NotEmpty(t, builds)
		b := builds[0]
		assert.Equal(t, build.Failed, b.Status, "build should fail when secret fetch fails")

		// The secret step should have error about secret fetch failure
		var secretStep *build.Step
		for i, s := range b.Steps {
			if s.Type == "secret" && s.Name == "will-fail" {
				secretStep = &b.Steps[i]
				break
			}
		}
		require.NotNil(t, secretStep, "secret step should exist")
		assert.Contains(t, secretStep.Logs, "failed to fetch secret", "logs should mention secret fetch failure, got: %s", secretStep.Logs)
	})

	cancel()
	wg.Wait()
}

func startWorker(ctx context.Context, sy string, t queue.Topic, s pikoci.Service, l *slog.Logger) {
	subscription, err := pubsub.OpenSubscription(ctx, fmt.Sprintf("%s://secrets-test", sy))
	if err != nil {
		l.Error("failed to open subscription", "error", err)
		return
	}
	defer subscription.Shutdown(ctx)

	w := worker.New(s, t, subscription, l)
	w.Run(ctx)
}
