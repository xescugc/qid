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
| `trigger` | no       | Auto-run the job on new versions (default `false`) |
| `passed`  | no       | List of job names that must have run with this version first |

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

### put

Pushes to a resource, running its `push` command.

```hcl
put "git" "my_repo" {
  params {
    name = "my-repo"
  }
}
```

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
