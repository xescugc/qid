# Services

A service is an ephemeral process that runs alongside a job's tasks. Services are started before tasks and stopped unconditionally after, regardless of whether the tasks succeed or fail. Common use cases include databases, caches, message brokers, or any dependency that needs to be running while tasks execute.

## Defining a service

Services are runner-agnostic. You can start any process: a local daemon, a container, a background script, etc.

```hcl
service "postgres" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "pg_ctl -D $WORKDIR/pgdata -l $WORKDIR/pg.log start"]
  }

  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "pg_isready"]
    interval = "1s"
    timeout  = "30s"
  }

  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "pg_ctl -D $WORKDIR/pgdata stop"]
  }
}
```

| Field         | Required | Description                                                      |
|---------------|----------|------------------------------------------------------------------|
| `name`        | yes      | Label on the block                                               |
| `source`      | no       | URL to fetch the definition from (mutually exclusive with inline `start`/`stop`) |
| `params`      | no       | List of parameter names for per-job customization                |
| `start`       | yes*     | Runner command to start the service                              |
| `ready_check` | no       | Runner command to verify the service is ready (polled)           |
| `stop`        | yes*     | Runner command to stop the service (always runs)                 |

\* Not required when `source` is set.

### start

The `start` block runs once when the job begins, before any `get`, `task`, or `put` steps. If start fails, the job fails immediately and `stop` is still called for any already-started services.

### ready_check

The `ready_check` block is optional. When present, PikoCI polls the command at the specified interval until it exits with code 0 (ready) or the timeout is exceeded (fail). If no `ready_check` is defined, the job proceeds immediately after `start` completes.

| Field      | Default | Description                            |
|------------|---------|----------------------------------------|
| `interval` | `"1s"`  | Time between ready check attempts      |
| `timeout`  | `"60s"` | Maximum time to wait for readiness     |

### stop

The `stop` block runs unconditionally after all tasks complete, whether they succeeded or failed. Stop failures are logged but do not change the job's status.

## Sourcing from URL

Instead of defining `start`/`stop` commands inline, you can point to an external HCL file:

```hcl
service "postgres" {
  source = "https://example.com/services/postgres.hcl"
  params = ["version"]
}
```

Two URL formats are supported:

- **`pikoci://<name>`** resolves to the PikoCI registry (no built-in services are shipped yet, but this is reserved for future additions).
- **`https://...`** or **`http://...`** fetches HCL from any URL.

When `source` is set, you must not define inline `start`, `stop`, or `ready_check` blocks. PikoCI will error if both are present.

## Referencing services in jobs

Reference a top-level service by name in a job:

```hcl
job "test" {
  service "postgres" {}

  get "cron" "timer" { trigger = true }
  task "run-tests" {
    run "exec" {
      path = "make"
      args = ["test"]
    }
  }
}
```

### Param overrides

Pass parameters to customize a service per job:

```hcl
job "test-pg15" {
  service "postgres" {
    version = "15"
  }
  ...
}

job "test-pg16" {
  service "postgres" {
    version = "16"
  }
  ...
}
```

Parameters are available in the service's start, ready_check, and stop commands as `$param_<name>`.

## Environment variables

Inside service commands (`start`, `ready_check`, `stop`), PikoCI exposes:

| Variable               | Description                                    |
|------------------------|------------------------------------------------|
| `$BUILD_ID`            | Unique ID of the current build                 |
| `$BUILD_JOB_NAME`      | Name of the job                                |
| `$BUILD_PIPELINE_NAME` | Name of the pipeline                           |
| `$WORKDIR`             | Temporary working directory for the job        |
| `$param_<name>`        | Per-job parameter overrides                    |

## Lifecycle

1. All `service` steps are collected from the job's plan
2. Services are started in order (each `start` command runs sequentially)
3. Ready checks run concurrently for all services that define one
4. If all services are ready, `get`, `task`, and `put` steps execute normally
5. After all steps complete (or if any step fails), `stop` runs for every started service

Services appear in the build as steps with type "service" and names like "postgres:start", "postgres:ready", "postgres:stop".

## Examples

### Local process

Start PostgreSQL directly on the worker. No containers needed:

```hcl
service "postgres" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "initdb -D $WORKDIR/pgdata && pg_ctl -D $WORKDIR/pgdata -l $WORKDIR/pg.log start"]
  }

  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "pg_isready"]
    interval = "1s"
    timeout  = "30s"
  }

  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "pg_ctl -D $WORKDIR/pgdata stop"]
  }
}
```

### Podman (rootless containers)

Use podman for rootless, daemonless containers:

```hcl
service "postgres" {
  params = ["version"]

  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "podman run -d --name pg-$BUILD_ID -e POSTGRES_PASSWORD=test postgres:$param_version"]
  }

  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "podman exec pg-$BUILD_ID pg_isready"]
    interval = "2s"
    timeout  = "30s"
  }

  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "podman rm -f pg-$BUILD_ID"]
  }
}
```

### Redis

A simple Redis instance with no ready check (starts fast enough):

```hcl
service "redis" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "redis-server --daemonize yes --dir $WORKDIR"]
  }

  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "redis-cli shutdown"]
  }
}
```
