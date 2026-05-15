# PikoCI

PikoCI is a CI/CD based on Queues, meaning that the ones running the Jobs are basically queue workers.

:warning: This is still a PoC and under heavy development which may cause breaking changes on the API or configuration :warning:

## Basic architecture

PikoCI has a simple architecture, you have the main service which is the brain of operations and then you have the workers which
run the Jobs tasks and interact with the PikoCI server to make any operations.

## How it works

Pipeline Resources are checked with the [`check_interval`](#check_interval) to see if there is a new version to trigger a new Build for a
Job that depends on that Resource that changed. If there is, then the new job(s) will be queued for a Worker to do.
Then when it finishes it checks if another job depends on it and queues that job(s) to be triggered.

Resources can also be triggered immediately via [webhooks](#webhooks) instead of waiting for the next poll interval.

When a job/resource is executed in a worker it creates a `$WORKDIR` `$XDG_CACHE_HOME/pikoci/{UUID}/` in which everything is executed.

The execution of any action is done on a [`runner`](#runner)

## Authentication and Authorization

PikoCI uses JWT-based authentication with a role-based access control (RBAC) model.

### Users

Users are created either via the `--users` server flag or through the API (by a global admin). Each user has a username, password (bcrypt hashed), and an optional global admin flag.

To generate the hashed password value for the `--users` flag, use the helper command:

```
pikoci user-password --username myuser --password mypassword
```

This outputs `myuser:$2a$14$...` which can be passed to `--users`.

### Teams

Everything in PikoCI is scoped under a Team. A default `main` team is used when not specified. Teams have a canonical name (lowercase, URL-safe) derived from their display name.

### Roles

There are 3 authorization levels:

* **Global Admin**: Can do everything — create users, create/delete teams, manage all team members, and perform all operations on any team's pipelines
* **Team Admin**: Can manage the team (update name, add/remove members), and create/edit/delete pipelines within that team
* **Team Member**: Can view the team and its pipelines, trigger jobs and resources, and view builds and resource versions

### Login

On the web UI, users are presented with a login screen. After login, the JWT is stored in the browser's local storage.

On the CLI, use:

```
pikoci client login --url localhost:4000 --username myuser --password mypassword
```

The JWT is stored at `$XDG_CONFIG_HOME/pikoci/authentication` and automatically used for subsequent CLI commands.

## Server Configuration

* `port`: The port to expose for the web server and API
* `jwt-secret` (REQUIRED): The secret used to sign JWT tokens for authentication
* `users`: List of initial users to create on startup, format: `USERNAME:HASH-PASSWORD` (use `pikoci user-password` to generate)
* `db-system`: Which type of DB to use. Each system has its own required flags:
  * `mem`: Run the DB in memory (default). Uses SQLite's in-memory mode
  * `sqlite`: Uses SQLite. The file will be at `$XDG_DATA_HOME/pikoci/pikoci.db`
  * `mysql`: Uses a MySQL/MariaDB DB. Requires: `db-host`, `db-port`, `db-user`, `db-password`, `db-name`
  * `postgresql`: Uses a PostgreSQL DB (also compatible with CockroachDB). Requires: `db-host`, `db-port`, `db-user`, `db-password`, `db-name`
* `db-host`: Database host
* `db-port`: Database port
* `db-user`: Database user
* `db-password`: Database password
* `db-name`: Database name
* `run-worker`: Specifies if you want to run the workers on the same server or not
* `concurrency`: How many parallel instances has the worker
* `pubsub-system`: Which PubSub system to use. Supported values:
  * `mem`: In-memory (default, for development/testing)
  * `nats`: NATS messaging. Requires env var `NATS_SERVER_URL`
  * `rabbit`: RabbitMQ. Requires env var `RABBIT_SERVER_URL` (AMQP URL format, e.g. `amqp://guest:guest@localhost:5672/`)
  * `kafka`: Apache Kafka. Requires env var `KAFKA_BROKERS` (comma-separated broker list, e.g. `localhost:9092`)
* `log-level`: Sets the log level (`debug`, `info`, `warn`, `error`), by default is `info`
* `team-canonical`: Team canonical to scope the initial pipeline creation (default: `main`)
* `pipeline-name`: If defined it'll create a pipeline on the service start
* `pipeline-config`: Path to the Pipeline configuration
* `pipeline-vars`: Path to the Pipeline vars


## Worker Configuration

* `pikoci-url`: Used to make all the interactions with the DB through it
* `jwt-secret` (REQUIRED): Must match the server's JWT secret — used to generate a worker token that bypasses authorization
* `concurrency`: How many parallel instances has the worker
* `pubsub-system`: Which PubSub system to use (`mem`, `nats`, `rabbit`, `kafka`). See Server Configuration for env var details per system
* `log-level`: Sets the log level (`debug`, `info`, `warn`, `error`), by default is `info`

## Supported Backends

### Database Backends

| Name | `--db-system` value | Driver | Tested with (Docker image) | Notes |
|---|---|---|---|---|
| In-Memory | `mem` | SQLite (`:memory:`) | — | Default, for dev/testing |
| SQLite | `sqlite` | SQLite | — | File-based, single-node |
| MySQL/MariaDB | `mysql` | `go-sql-driver/mysql` | `mariadb:11.4.2` | Production-ready |
| PostgreSQL | `postgresql` | `lib/pq` | `postgres:17` | Production-ready |
| CockroachDB | — | `lib/pq` (PG-compatible) | `cockroachdb/cockroach` | Use `postgresql` as db-system |
| TiDB | — | `go-sql-driver/mysql` (MySQL-compatible) | `pingcap/tidb` | Use `mysql` as db-system |

CockroachDB and TiDB work as PostgreSQL-compatible and MySQL-compatible databases respectively. They are not first-class `--db-system` flag values; instead, use `postgresql` or `mysql` and point the connection to the CockroachDB/TiDB host.

### PubSub Backends

| Name | `--pubsub-system` value | Env Var | Docker Image |
|---|---|---|---|
| In-Memory | `mem` | — | — |
| NATS | `nats` | `NATS_SERVER_URL` | `nats:2.12.0` |
| RabbitMQ | `rabbit` | `RABBIT_SERVER_URL` | `rabbitmq:3-management` |
| Kafka | `kafka` | `KAFKA_BROKERS` | `bitnami/kafka` |

## Pipeline

The Pipeline is configured using [HCL](https://github.com/hashicorp/hcl) which makes it so the pipeline is really easy to read and write compared to YAML/JSON.

The main blocks of the Pipeline are:
* `variables`: Allow to change Pipelines values on creation dynamically
* `resource_type`: Define a task that will check if an external resource has a new version (like a new git commit or a new s3 image)
* `resource`: An execution of a `resource_type` passing the needed parameters (like git_url or credentials)
* `job`: The main block that executes the task having as context the resource that it depends on and that can automatically trigger it

### HCL

A quick introduction for people not used to HCL.

HCL has 2 main structures

#### Attribute

A simple assignation of a value to a key

```
key = "value"
```

#### Block

A block is defined for a key and N labels and then `{}`

```
my_block "label1" "label2" {}
```

Blocks can sometimes be defined multiple times, like the root ones: `job`, `resource`, `variable`, etc.

### Variables

You can define variables to the pipeline configuration for reusability purpose (same Pipeline for different usecases).

Another usecase for the Variables is to pass secrets to the Pipeline. On the future I may implement a dedicated element [`secret_type` and `secret`](https://github.com/xescugc/pikoci/issues/12) that would make it more simpler and declarative to use secrets.

When creating a Pipeline, one of the options is `vars`, which is a JSON with the overridden values of the Variables

Variables are used like:
* `attribute = var.my_var`
* `attribute = "my_${var.my_var}_var"`

#### `type`

REQUIRED and expects one of the following: `string`, `number` or `boolean`

#### `default`

The default value of the variable

#### Example

```hcl
variable "repo_url" {
  type = string
  default = "https://github.com/xescugc/pikoci.git"
}

variable "repo_name" {
  type = string
}
```

### Runner

Runners are the execution part of anything of the actions (job, resource etc). With that you can define which is the context in which to execute the task.
There is a default one defined, `exec` that allows to execute everything on the host machine of the workers.

To define which `runner` to use, the blocks that have that will declare it, for example a job like:

```hcl
job "build" {
  get "git" "pikoci" {
    passed  = ["test"]
    trigger = true
  }
  task "build" {
    run "exec" {
      path = "make"
      args = [
        "-C",
        "${var.repo_name}",
        "release",
      ]
    }
  }
}
```

Will use `exec` when running the task `build`, and all the content of the `run`  will be passed to the `runner` as env variables,
in this case `$path` will be available to be used on the `runner`. The `$args` placeholder in the runner definition is replaced by the `args` list from the `run` block.

The runner `exec` will be always available to be used (no need to define it) and it looks like:

```hcl
runner "exec" {
  run {
    path = "$path"
    args = ["$args"]
  }
}
```

So to use it you need to pass a `path` and `args` to the block. Each element of the `args` list is passed as a single argument:

```hcl
job "build" {
  get "git" "pikoci" {
    passed  = ["test"]
    trigger = true
  }
  task "build" {
    run "exec" {
      path = "make"
      args = [
        "-C",
        "${var.repo_name}",
        "release",
      ]
    }
  }
}
```

#### `run`

It defines in which context the task is going to run.

##### `path`

The name of or path to the executable to run.

##### `args`

A list of arguments to pass to the command. Each element is passed as a single argument (no shell-style splitting).

#### Example

```hcl
runner "docker" {
  run {
    path = "docker"
    args = [
      "run", "--rm",
      "-v","$WORKDIR:/workdir",
      "-w","/workdir",
      "$image",
      "$cmd"
    ]
  }
}
```

### Resource Type

Resource Types are the core of PikoCI CI/CD as they are the ones that automate all the Jobs by listening to external resource changes.

For now you always have to define the `resource_type`, but ideally they are reusable between different pipelines so in the future this will [change](https://github.com/xescugc/pikoci/issues/11) so you can reuse them

#### Internals

This is the list of internal Resource Types:

##### `cron`

It always returns a new value (internally runs `date`, so for now only available if `date` is present). When used in a [`resource`](#resource) with
[`check_interval`](#check_interval) you effectively created a way to execute a Job periodically.

###### Example

Every `5s` it'll run the Job `my_job` as it's registered with `get "cron" "my_cron"`

```hcl
resource "cron" "my_cron" {
  check_interval = "@every 5s"
  params {}
}

job "my_job" {
  get "cron" "my_cron" {
    trigger = true
  }
  task "echo" {
    run "exec" {
      path = "echo"
      args = [
        "IN",
      ]
    }
  }
}
```

#### `params`

List of keys that are required for a `resource` in order to implement this `resource_type` and will be passed to `check`, `pull` and `push` in ENV as `$param_`+(param name). So if you have something like:

```hcl
resource_type "git" {
    params = [ "url", "name" ]
    // more things ...
}
```

It'll be passed to the `check`, `pull` and `push` as `$param_url` and `$param_name`

#### `check`

The `label` is the [`runner`](#runner)

Called periodically (`@every 1m` by default, configurable via [`resource.check_interval`](#check_interval)) to see if there are new versions.
The **LAST LINE** (do not pretty-print JSON) of the output must be a JSON array of all the new versions. These new versions will be available via ENV to `pull` and `push` by flattening (nested values are flattened with `_`) the JSON and prefixing it with `$version_` (JSON key).

For example on `git` you could do `git log -1 --pretty=format:"%H" | jq -Rsc "(. / \"\n\" | map(select(length>0) | { "ref": . }))"` which will then return 

```json
[{"ref":"7101df99a068495ccf23ec656db8d93d18fe30a2"}]
```

Then on the `pull` you'll have a `$version_ref` available, if there were more attributes those would also be available.

On the check you also have access to the **PREVIOUS** version that was detected, so in the script you should check if there is anything new. If not,
return `[]`. Any element in the `[]` will be considered a new version (there is no uniqueness detection), so make sure not to return
the same version again (unless that's what you need).

#### `pull`

The `label` is the [`runner`](#runner)

When a Job has a `get` to a resource, the `resource.pull` is run before the `job.task` in the same context. For example, you can pull from Git (if git is the resource) and the `job.task` will have access to the pulled repository locally. The version is available via `$version_*` env vars (the same values returned by `check`).

#### `push`

The `label` is the [`runner`](#runner)

When a Job has a `put` step for a resource, the `resource_type.push` is run. Resource-level params are available with the `$param_*` prefix (same as `check` and `pull`). Put-step params use the `$put_*` prefix (e.g. `$put_tag`).

#### Example

This pulls the git repo and checks for the new Versions (commits in this case)

```hcl
resource_type "git" {
  params = [
    "url",
    "name",
  ]
  check "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      git clone --quiet $param_url $param_name
      cd $param_name
      if [[ -n $version_ref ]]; then
        git log $version_ref..HEAD --pretty=format:"%H" | jq -Rsc "(. / \"\n\" | map(select(length>0) | { \"ref\": . }))"
      else
        git log -1 --pretty=format:"%H" | jq -Rsc "(. / \"\n\" | map(select(length>0) | { \"ref\": . }))"
      fi
      EOT
    ]
  }
  pull "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      git clone $param_url $param_name
      cd $param_name
      git checkout $version_ref
      EOT
    ]
  }
  push "exec" { }
}
```

### Resource

Resources are the implementation of the [Resource Type](#resource-type) by initializing it with the params defined on the [`resource_type.params`](#params).

When defining a `resource`, the first label is the `resource_type` and the second is the name of the resource.

#### `params`

Is a block that contains all the `resource_type.params` defined that will be passed to it

#### `check_interval`

Interval in which to check the resource. By default the value is `@every 1m` and the syntax is the [CRON](https://github.com/netresearch/go-cron)

#### Example

```hcl
resource "git" "my_repo" {
  params {
    url = "https://github.com/xescugc/pikoci.git"
    name = "pikoci"
  }
  check_interval = "@every 3s"
}
```

### Webhooks

By default, PikoCI discovers new resource versions by polling at the `check_interval` (default: every 1 minute). Webhooks let external services (GitHub, GitLab, etc.) push notifications to PikoCI, triggering a resource check immediately.

Each resource is automatically assigned a webhook token (UUID) on creation. The webhook endpoint is:

```
POST /webhooks/{webhook_token}
```

This endpoint is **public** (no JWT required) — the token itself acts as authentication. It accepts any POST request and ignores the body, making it compatible with any service that can send HTTP POST requests (GitHub webhooks, GitLab webhooks, curl scripts, etc.).

#### Viewing the Webhook URL

Team admins can view the webhook URL from the resource detail page by clicking the dropdown arrow next to the "Trigger Resource" button and selecting "Webhook URL". The panel shows the full URL and provides a "Copy" button.

Non-admin users cannot see the webhook token — it is stripped from API responses.

#### Regenerating the Token

Admins can regenerate the webhook token from the same panel by clicking "Regenerate Token". This invalidates the previous URL immediately. The API endpoint for programmatic regeneration is:

```
POST /teams/{team_canonical}/pipelines/{pipeline_name}/resources/{resource_canonical}/webhook_token
```

This endpoint requires admin authorization.

#### Example: GitHub Webhook

1. Navigate to the resource detail page in PikoCI
2. Click the dropdown arrow → "Webhook URL"
3. Copy the URL
4. In your GitHub repository, go to Settings → Webhooks → Add webhook
5. Paste the PikoCI webhook URL
6. Set Content type to `application/json`
7. Select "Just the push event"
8. Save

Now every push to the repository will immediately trigger a resource check in PikoCI.

### Job

Jobs are where the user actions happen. Jobs define a **plan** — an ordered sequence of `get`, `task`, and `put` steps that are executed in the order they appear in the HCL configuration. This allows interleaving steps for CD workflows (e.g., build → push → deploy).

#### `get`

Get steps declare which resources the job depends on. The labels must match a `resource` definition. Each has the following configuration:

##### `trigger`

Boolean marking if changes on that resource will automatically trigger the Job

##### `passed`

Array of job names that the Job depends on. If a listed job finishes successfully and also
has the resource with trigger enabled, this job will be automatically executed

##### `on_success`

The `label` is the [`runner`](#runner)

Runs when the Job succeeds

##### `on_failure`

The `label` is the [`runner`](#runner)

Runs when the Job fails

##### `ensure`

The `label` is the [`runner`](#runner)

Runs always

#### `task`

Tasks are the ones that run the logic of the Job

##### `run`

The `label` is the [`runner`](#runner)

The actual logic for the Job to run

##### `on_success`

The `label` is the [`runner`](#runner)

Runs when the Job succeeds

##### `on_failure`

The `label` is the [`runner`](#runner)

Runs when the Job fails

##### `ensure`

The `label` is the [`runner`](#runner)

Runs always

#### `put`

Put steps push content to a resource. The labels must match a `resource` definition: the first label is the resource type and the second is the resource name. Any additional attributes inside the `put` block are passed as parameters to the `resource_type.push` runner.

##### `on_success`

The `label` is the [`runner`](#runner)

Runs when the put step succeeds

##### `on_failure`

The `label` is the [`runner`](#runner)

Runs when the put step fails

##### `ensure`

The `label` is the [`runner`](#runner)

Runs always

#### `on_success`

The `label` is the [`runner`](#runner)

Runs when the Job succeeds

#### `on_failure`

The `label` is the [`runner`](#runner)

Runs when the Job fails

#### `ensure`

The `label` is the [`runner`](#runner)

Runs always

#### Example

2 jobs: `test` depends on `gen`, and `gen` is triggered by the `git.my_repo` resource

```hcl
job "gen" {
  get "git" "my_repo" {
    trigger = true
  }
  task "gen" {
    run "exec" {
      path = "make"
      args = [
        "-C",
        "${var.repo_name}",
        "gen",
      ]
    }
    on_success "exec" {
      path = "ls"
    }
  }
}

job "test" {
  get "git" "my_repo" {
    passed  = ["gen"]
    trigger = true
  }
  task "test" {
    run "exec" {
      path = "make"
      args = [
        "-C",
        "${var.repo_name}",
        "test",
      ]
    }
  }
  ensure "exec" {
    path = "ls"
  }
}
```

A CD pipeline with interleaved get/task/put steps:

```hcl
job "deploy" {
  get "git" "my_repo" {
    trigger = true
  }
  task "build" {
    run "docker" {
      image = "golang:1.25"
      cmd = "make build"
    }
  }
  put "git" "my_repo" {
    tag = "latest"
  }
}
```

## CLI

Not all the API is ported to the CLI [yet](https://github.com/xescugc/pikoci/issues/58). A Go client is available at `pikoci/transport/http/client` if needed.

First, log in to get a JWT stored locally:

```
pikoci client login --url localhost:4000 --username myuser --password mypassword
```

Then you can interact with the API. All pipeline and job commands require `--team-canonical` (`-tc`) to scope the operation to a team (defaults to `main`):

```
pikoci client pipelines --tc main list
pikoci client pipelines --tc main create --name my-pipeline --config pipeline.hcl
pikoci client pipelines --tc main get --name my-pipeline
pikoci client pipelines --tc main update --name my-pipeline --config pipeline.hcl
pikoci client pipelines --tc main delete --name my-pipeline

pikoci client jobs --tc main --pn my-pipeline get --jn my-job
pikoci client jobs --tc main --pn my-pipeline trigger --jn my-job
```
