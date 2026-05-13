# QID

QID is a CI/CD based on Queues, meaning that the ones running the Jobs are basically queue workers.

:warning: This is still a PoC and under heavily development which may cause breaking changes on the API or configuration :warning:

## Basic architecture

QID has a simple architecture, you have the main service which is the brain of operations and then you have the workers which
run the Jobs tasks and interact with the QID server to make any operations.

## How it works

Pipelines Resources are check with the [`check_interval`](#check_interval) to see if there is a new version to trigger a new Build for a
Job that depends on that Resource that changed. If there is then a new job/s will be queued for a Worker to do.
Then when it finishes it checks if another job depends on it and queues for that job/s to be triggered.

When a job/resource is executed in a worker it creates a `$WORKDIR` `$XDG_CACHE_HOME/qid/{UUID}/` in which everything is executed.

The execution of any action is done on a [`runner`](#runner)

## Authentication and Authorization

QID uses JWT-based authentication with a role-based access control (RBAC) model.

### Users

Users are created either via the `--users` server flag or through the API (by a global admin). Each user has a username, password (bcrypt hashed), and an optional global admin flag.

To generate the hashed password value for the `--users` flag, use the helper command:

```
qid user-password --username myuser --password mypassword
```

This outputs `myuser:$2a$14$...` which can be passed to `--users`.

### Teams

Everything in QID is scoped under a Team. A default `main` team is used when not specified. Teams have a canonical name (lowercase, URL-safe) derived from their display name.

### Roles

There are 3 authorization levels:

* **Global Admin**: Can do everything — create users, create/delete teams, manage all team members, and perform all operations on any team's pipelines
* **Team Admin**: Can manage the team (update name, add/remove members), and create/edit/delete pipelines within that team
* **Team Member**: Can view the team and its pipelines, trigger jobs and resources, and view builds and resource versions

### Login

On the web UI, users are presented with a login screen. After login, the JWT is stored in the browser's local storage.

On the CLI, use:

```
qid client login --url localhost:4000 --username myuser --password mypassword
```

The JWT is stored at `$XDG_CONFIG_HOME/qid/authentication` and automatically used for subsequent CLI commands.

## Server Configuration

* `port`: The port to expose for the web server and API
* `jwt-secret` (REQUIRED): The secret used to sign JWT tokens for authentication
* `users`: List of initial users to create on startup, format: `USERNAME:HASH-PASSWORD` (use `qid user-password` to generate)
* `db-system`: Which type of DB to use, each one has it's values
  * `mysql`: Uses a MySQL DB
    * `db-host`: Database Host
    * `db-port`: Database Port
    * `db-user`: Database User
    * `db-password`: Database Password
    * `db-name`: Database Name
  * `sqlite`: Uses SQLite. The file will be at `$XDG_DATA_HOME/qid/qid.db`
  * `mem`: Run the DB in memory, which basically uses the SQLite memory option
* `run-worker`: Specifies if you want to run the workers on the same server or not
* `concurrency`: How many parallel instances has the worker
* `pubsub-system`: Which DB system to use. Internally I use [google/go-cloud/pubsub](https://gocloud.dev/howto/runtimevar/#services) so any of those could be implemented but I only did it with `mem` and `nats`. For NATS you'll need to pass the `NATS_SERVER_URL`. For any other just open an issue.
* `log-level`: Sets the log level (`debug`, `info`, `warn`, `error`), by default is `info`
* `team-canonical`: Team canonical to scope the initial pipeline creation (default: `main`)
* `pipeline-name`: If defined it'll create a pipeline on the service start
* `pipeline-config`: Path to the Pipeline configuration
* `pipeline-vars`: Path to the Pipeline vars


## Worker Configuration

* `qid-url`: Used to make all the interactions with the DB through it
* `jwt-secret` (REQUIRED): Must match the server's JWT secret — used to generate a worker token that bypasses authorization
* `concurrency`: How many parallel instances has the worker
* `pubsub-system`: Which DB system to use. Internally I use [google/go-cloud/pubsub](https://gocloud.dev/howto/runtimevar/#services) so any of those could be implemented but I only did it with `mem` and `nats`. For NATS you'll need to pass the `NATS_SERVER_URL`. For any other just open an issue.
* `log-level`: Sets the log level (`debug`, `info`, `warn`, `error`), by default is `info`

## Pipeline

The Pipeline is configured using [HCL](https://github.com/hashicorp/hcl) which makes it so the pipeline is really easy to read and write compared to YAML/JSON.

The main blocks of the Pipeline are:
* `variables`: Allow to change Pipelines values on creation dynamically
* `resource_type`: Define a task that will check if an external resource has a new version (like a new git commit or a new s3 image)
* `resource`: An execution of a `resource_type` passing the needed parameters (like git_url or credentials)
* `job`: The main block that executes the task having as context the resource that it depends on and that can automatically trigger it

### HCL

A quick introduction to people not use to HCL.

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

Blocks can sometimes be defined multiple times, lie the root ones: `job`, `resource`, `variable` ... etc.

### Variables

You can define variables to the pipeline configuration for reusability purpose (same Pipeline for different usecases).

Another usecase for the Variables is to pass secrets to the Pipeline. On the future I may implement a dedicated element [`secret_type` and `secret`](https://github.com/xescugc/qid/issues/12) that would make it more simpler and declarative to use secrets.

When creating a Pipeline, one of the options is the `vars` which is a JSON withe the overwrite values of the Variables

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
  default = "https://github.com/xescugc/qid.git"
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
  get "git" "qid" {
    passed  = ["test"]
    trigger = true
  }
  task "build" {
    run "exec" {
      path = "make"
      args = "-C ${var.repo_name} release"
    }
  }
}
```

Will use `exec` when running the task `build`, and all the content of the `run`  will be passed to the `runner` as env variables,
in this case `$path` and `$args` will be available to be used on the `runner`.

The runner `exec` will be always available to be used (no need to define it) and it looks like:

```hcl
runner "exec" {
  run {
    path = "$path"
    args = ["$args"]
  }
}
```

So to use it you need to pass to the block a `path` and an `args`. If you want to pass  multyline or separate the arguments here is an example:

```hcl
job "build" {
  get "git" "qid" {
    passed  = ["test"]
    trigger = true
  }
  task "build" {
    run "exec" {
      path = "make"
      args = <<-EOT
          '-C'
          '${var.repo_name}'
          'release'
        EOT
    }
  }
}
```

The logic for separating the `args` (as this is just a 1 string with `\n`) is implemented using the [go-shellquote#Split](https://github.com/kballard/go-shellquote) for reference, so using in this case `'arg1' 'arg2'`

#### `run`

It defines in which context the task is going to run.

##### `path`

The name of or path to the executable to run.

##### `args`

Arguments to pass to the command.

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

Resource Type are the core of QID CI/CD as they are the ones that automate all the Jobs by listening to external resources changes.

For now you always have to defined the `resource_type` but ideally they are really reusable in between different pipelines so in the future this will [change](https://github.com/xescugc/qid/issues/11) so you can reuse it

#### Internals

This is the list of internal Resource Types:

##### `cron`

It'll always return a new value (internally does `date` so for now only available if `date` is present). When used in a [`resource`](#resource) with
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
      args = "'IN'"
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

Is called periodically (`@every 1m` but can be changed on the [`resource.check_interval`](#check_interval)) to see if there are new versions.
The **LAST LINE** (so do not pretty print JSON) of the output has to be a JSON containing an array of all the new versions. This new versions will the be available via ENV to the `pull` and `push` by flattening(nested values will be flatten with `_`) the JSON and prefixing it with `$version_`(JSON key).

For example on `git` you could do `git log -1 --pretty=format:"%H" | jq -Rsc "(. / \"\n\" | map(select(length>0) | { "ref": . }))"` which will then return 

```json
[{"ref":"7101df99a068495ccf23ec656db8d93d18fe30a2"}]
```

Then on the `pull` you'll have a `$version_ref` available, if there where more attributes those would also be available.

On the check you have also access to the **PREVIOUS** version that was detected so on the script you should check if there is anything new, if not
then just return `[]`, any element on the `[]` will be considered a new version (there is no uniqueness detection) so check to not return
the same version again (if that's not what's needed ofc).

#### `pull`

The `label` is the [`runner`](#runner)

When a Job has a `get` to a resource, when the Job is executed the `resource.pull` is ran beforehand and the `job.task` is ran on the same context so you can pull from GIT(if git is the resource) and the `job.task` will have access to the pulled repository locally. It will have version (with `$version_*`) which is the version to pull (same one returned on the `check`)

#### `push`

NOT YET IMPLEMENTED. But will be used to push new content to the Resource Type

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
    args = <<-EOT
        '-ec'
        'git clone --quiet $param_url $param_name
        cd $param_name
        if [[ -n $version_ref ]]; then
          git log $version_ref..HEAD --pretty=format:"%H" | jq -Rsc "(. / \"\n\" | map(select(length>0) | { "ref": . }))"
        else
          git log -1 --pretty=format:"%H" | jq -Rsc "(. / \"\n\" | map(select(length>0) | { "ref": . }))"
        fi'
      EOT
  }
  pull "exec" {
    path = "/bin/sh"
    args = <<-EOT
        '-ec'
        'git clone $param_url $param_name
        cd $param_name
        git checkout $version_ref'
      EOT
  }
  push "exec" { }
}
```

### Resource

Resources are the implementation of the [Resource Type](#resource-type) by initializing it with the params defined on the [`resource_type.params`](#params).

When defined a `resource` the first label is the `resource_type` and the 2nd is the name of the resource.

#### `params`

Is a block that contains all the `resource_type.params` defined that will be passed to it

#### `check_interval`

Interval in which to check the resource. By default the value is `@every 1m` and the syntax is the [CRON](https://github.com/netresearch/go-cron)

#### Example

```hcl
resource "git" "my_repo" {
  params {
    url = "https://github.com/xescugc/qid.git"
    name = "qid"
  }
  check_interval = "@every 3s"
}
```

### Job

Jobs are where the use actions happen. Jobs define a set of tasks that will be ran in groups, first all the `get` and then all the `task`.

#### `get`

Tasks are the ones marking which resources does the job depends on. The labels have to match the `resource`, it
has the following configuration:

##### `trigger`

Boolean marking if changes on that resource will automatically trigger the Job

##### `passed`

Array of job names that the Job depends on. Meaning that if that job finishes and has also
the resource and trigger it'll be automatically executed on success

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

3 jobs, `test` depends on `gen` and `gen` and `notify_slack` are triggered by the `git.my_repo` `resource`

```hcl
job "gen" {
  get "git" "my_repo" {
    trigger = true
  }
  task "gen" {
    run "exec" {
      path = "make"
      args = <<-EOT
          '-C'
          '${var.repo_name}'
          'gen'
        EOT
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
      args = <<-EOT
          '-C'
          '${var.repo_name}'
          'test'
        EOT
    }
  }
  ensure "exec" {
    path = "ls"
  }
}
```

## CLI

Not all the API is ported to the CLI [yet](https://github.com/xescugc/qid/issues/58), it exists on the `qid/transport/http/client` as a GO client if need be.

First, log in to get a JWT stored locally:

```
qid client login --url localhost:4000 --username myuser --password mypassword
```

Then you can interact with the API. All pipeline and job commands require `--team-canonical` (`-tc`) to scope the operation to a team (defaults to `main`):

```
qid client pipelines --tc main list
qid client pipelines --tc main create --name my-pipeline --config pipeline.hcl
qid client pipelines --tc main get --name my-pipeline
qid client pipelines --tc main update --name my-pipeline --config pipeline.hcl
qid client pipelines --tc main delete --name my-pipeline

qid client jobs --tc main --pn my-pipeline get --jn my-job
qid client jobs --tc main --pn my-pipeline trigger --jn my-job
```
