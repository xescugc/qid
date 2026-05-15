# Pipeline Reference

Pipelines are defined in [HCL](https://github.com/hashicorp/hcl). A pipeline file contains `variable`, `resource_type`, `resource`, `runner`, and `job` blocks.

## variable

Declares a pipeline variable. Variables can be referenced as `var.<name>` or `${var.<name>}` anywhere in the pipeline.

```hcl
variable "repo_url" {
  type    = string
  default = "https://github.com/xescugc/pikoci.git"
}

variable "repo_name" {
  type = string
}
```

| Field     | Required | Description                        |
|-----------|----------|------------------------------------|
| `name`    | yes      | Label on the block                 |
| `type`    | yes      | `string`                           |
| `default` | no       | Default value if not set via vars file |

Variables without a default must be provided via a JSON vars file (`--vars` / `--pipeline-vars`).

## resource_type

Defines how to check, pull, and push a resource. See [Resource Types](Resource-Types).

```hcl
resource_type "git" {
  params = ["url", "name"]

  check "exec" {
    path = "/bin/sh"
    args = ["-ec", "git ls-remote $param_url HEAD | awk '{print $1}'"]
  }

  pull "exec" {
    path = "/bin/sh"
    args = ["-ec", "git clone $param_url $param_name && git checkout $version_ref"]
  }

  push "exec" {
    path = "/bin/sh"
    args = ["-ec", "cd $param_name && git push"]
  }
}
```

| Field    | Required | Description                                         |
|----------|----------|-----------------------------------------------------|
| `name`   | yes      | Label on the block                                  |
| `source` | no       | URL to fetch definition (e.g. `pikoci://git`)       |
| `params` | no       | List of parameter names                             |

When `source` is set, inline commands are not needed.

## resource

An instance of a resource type. See [Resource Types](Resource-Types).

```hcl
resource "git" "my_repo" {
  params {
    url  = var.repo_url
    name = "my-repo"
  }
}

resource "cron" "every_10s" {
  check_interval = "@every 10s"
  params {}
}
```

| Field            | Required | Description                                      |
|------------------|----------|--------------------------------------------------|
| `type`           | yes      | Label, must match a `resource_type` name         |
| `name`           | yes      | Label, unique name for this resource              |
| `params`         | yes      | Block with key/value pairs passed to the resource type |
| `check_interval` | no       | Cron expression or `@every <duration>` for automatic checks |

## runner

Defines a reusable execution environment. See [Runners](Runners).

```hcl
runner "docker" {
  run {
    path = "docker"
    args = [
      "run", "--rm",
      "-v", "$WORKDIR:/workdir",
      "-w", "/workdir",
      "$image",
      "$cmd",
    ]
  }
}
```

| Field    | Required | Description                                    |
|----------|----------|------------------------------------------------|
| `name`   | yes      | Label on the block                             |
| `source` | no       | URL to fetch definition (e.g. `pikoci://docker`) |

When `source` is set, inline `run` block is not needed.

## job

Jobs contain a plan of steps executed in order. Each step is one of `get`, `task`, or `put`.

```hcl
job "build" {
  get "git" "my_repo" {
    trigger = true
  }

  task "compile" {
    run "exec" {
      path = "make"
      args = ["build"]
    }
  }

  put "git" "my_repo" {
    params {
      name = "my-repo"
    }
  }

  on_success "exec" {
    path = "echo"
    args = ["build succeeded"]
  }

  on_failure "exec" {
    path = "echo"
    args = ["build failed"]
  }

  ensure "exec" {
    path = "echo"
    args = ["cleanup"]
  }
}
```

### get

Fetches a resource version. If `trigger = true`, the job runs automatically when a new version is detected.

```hcl
get "git" "my_repo" {
  trigger = true
  passed  = ["test"]
}
```

| Field     | Required | Description                                    |
|-----------|----------|------------------------------------------------|
| `type`    | yes      | Label, resource type name                      |
| `name`    | yes      | Label, resource name                           |
| `trigger`  | no       | Auto-run the job on new versions (default `false`) |
| `passed`   | no       | List of job names that must have run with this version first |
| `timeout`  | no       | Maximum duration for the step (e.g. `"2m"`, `"30s"`) |
| `attempts` | no       | Maximum number of times to try the step (default `1`, no retry) |

### task

Runs a command via a runner.

```hcl
task "test" {
  run "exec" {
    path = "make"
    args = ["test"]
  }
}
```

| Field     | Required | Description                                    |
|-----------|----------|------------------------------------------------|
| `name`     | yes      | Label on the block                             |
| `timeout`  | no       | Maximum duration for the step (e.g. `"10m"`, `"1h"`) |
| `attempts` | no       | Maximum number of times to try the step (default `1`, no retry) |

### put

Pushes to a resource, running its `push` command.

```hcl
put "git" "my_repo" {
  params {
    name = "my-repo"
  }
}
```

| Field     | Required | Description                                    |
|-----------|----------|------------------------------------------------|
| `type`     | yes      | Label, resource type name                      |
| `name`     | yes      | Label, resource name                           |
| `timeout`  | no       | Maximum duration for the step (e.g. `"5m"`, `"30s"`) |
| `attempts` | no       | Maximum number of times to try the step (default `1`, no retry) |

### Step hooks

Each step (and the job itself) can have `on_success`, `on_failure`, and `ensure` blocks:

- `on_success` runs after the step succeeds
- `on_failure` runs after the step fails
- `ensure` always runs, regardless of success or failure

```hcl
task "deploy" {
  run "exec" {
    path = "make"
    args = ["deploy"]
  }
  on_failure "exec" {
    path = "echo"
    args = ["deploy failed"]
  }
}
```

### Step timeout

Any step can set a `timeout` to limit how long its runner execution takes. The value is a Go duration string (e.g. `"30s"`, `"5m"`, `"1h30m"`). If the step exceeds the timeout, the process is killed, the step is marked as failed with a "step timed out after ..." message in the logs, and `on_failure`/`ensure` hooks still run normally. If no timeout is set, the step runs with no time limit.

```hcl
task "long-build" {
  timeout = "10m"
  run "exec" {
    path = "make"
    args = ["build"]
  }
  on_failure "exec" {
    path = "echo"
    args = ["build timed out or failed"]
  }
}
```

### Step retry

Any step can set `attempts` to retry on failure. The value is the maximum number of times the step will be tried (default `1`, no retry). If the step fails and attempts remain, the runner is re-invoked. Hooks (`on_failure`, `on_success`, `ensure`) only run after the final attempt. When combined with `timeout`, each attempt gets a fresh timeout. Attempt markers (e.g. `--- attempt 2/3 ---`) appear in the build logs starting from the second attempt onward.

```hcl
task "flaky-test" {
  timeout  = "5m"
  attempts = 3
  run "exec" {
    path = "make"
    args = ["test"]
  }
  on_failure "exec" {
    path = "echo"
    args = ["tests failed after 3 attempts"]
  }
}
```

## Full example

Using built-in `git` and `docker` (no inline resource_type or runner blocks needed):

```hcl
variable "repo_url" {
  type    = string
  default = "https://github.com/xescugc/pikoci.git"
}

variable "repo_name" {
  type    = string
  default = "pikoci"
}

resource "git" "pikoci" {
  params {
    url  = var.repo_url
    name = var.repo_name
  }
}

resource "cron" "schedule" {
  check_interval = "@every 10s"
  params {}
}

job "test" {
  get "git" "pikoci" {
    trigger = true
  }

  task "run-tests" {
    run "docker" {
      image = "golang:1.23"
      cmd   = "cd ${var.repo_name} && make test"
    }
  }
}

job "deploy" {
  get "git" "pikoci" {
    passed  = ["test"]
    trigger = true
  }

  task "deploy" {
    run "exec" {
      path = "echo"
      args = ["deploying..."]
    }
  }

  on_success "exec" {
    path = "echo"
    args = ["deployed"]
  }
}
```
