# QID

QID is a CI/CD based on Queues, meaning that the ones running the Jobs are basically queue workers.

:warning: This is still a PoC and under heavily development which may cause breaking changes on the API or configuration :warning:

## Basic architecture

QID has a simple architecture, you have the main service which is the brain of operations and then you have the workers which
run the Jobs tasks and interact with the QID server to make any operations.

## How it works

The QID server checks periodically (every Xs) the Pipelines Resources to see if there is a new version to trigger a new Build for a
Job that depends on that Resource that changed. If there is then a new job/s will be queued for a Worker to do.
Then when it finishes it checks if another job depends on it and queues for that job/s to be triggered.

Workers execute everything on the host machine for now (either on the server if local or on the worker host), each time something is executed
it's done on a UUID folder so it never collisions with a previous run, will potentially [change](https://github.com/xescugc/qid/issues/57) on the future

## Server Configuration

* `port`: The port to expose fro the web server and API
* `db-system`: Which type of DB to use, each one has it's values
  * `mysql`: Uses a MySQL DB
    * `db-host`: Database Host
    * `db-port`: Database Port
    * `db-user`: Database User
    * `db-password`: Database Password
    * `db-name`: Database Name
  * `sqlite`: Uses SQLite
    * `db-file`: The file in which to store the DB (does not need to exist as it'll be created)
  * `mem`: Run the DB in memory, which basically uses the SQLite memory option
* `run-worker`: Specifies if you want to run the workers on the same server or not
* `concurrency`: How many parallel instances has the worker
* `pubsub-system`: Which DB system to use. Internally I use [google/go-cloud/pubsub](https://gocloud.dev/howto/runtimevar/#services) so any of those could be implemented but I only did it with `mem` and `nats`. For NATS you'll need to pass the `NATS_SERVER_URL`. For any other just open an issue.

## Worker Configuration

* `qid-url`: Used to make all the interactions with the DB through it
* `concurrency`: How many parallel instances has the worker
* `pubsub-system`: Which DB system to use. Internally I use [google/go-cloud/pubsub](https://gocloud.dev/howto/runtimevar/#services) so any of those could be implemented but I only did it with `mem` and `nats`. For NATS you'll need to pass the `NATS_SERVER_URL`. For any other just open an issue.

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

In case of QID if it has a label it means it can be defined multiple times in the context.
For example the Job Task can be defined multiple times to have multiple tasks.


### Variables

You can define variables to the pipeline configuration for reusability purpose (same Pipeline for different usecases).

Another usecase for the Variables is to pass secrets to the Pipeline. On the future I may implement a dedicated element [`secret_type` and `secret`](https://github.com/xescugc/qid/issues/12) that would make it more simpler and declarative to use secrets.

When creating a Pipeline, one of the options is the `vars` which is a JSON withe the overwrited values of the Variables

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

### Resource Type

Resource Type are the core of QID CI/CD as they are the ones that automate all the Jobs by listening to external resources changes.

For now you always have to defined the `resource_type` but ideally they are really reusable in between different pipelines so in the future this will [change](https://github.com/xescugc/qid/issues/11) so you can reuse it

#### `inputs`

List of keys that are required for a `resource` in order to implement this `resource_type` and will be passed to `check`, `pull` and `push` in ENV.

#### `check`

Is called periodically (every 1' on the future you'll be able to [change](https://github.com/xescugc/qid/issues/49) it) to see if there are new versions.
The output separated by `\n` is what it's considered the new versions (may [change](https://github.com/xescugc/qid/issues/9) on the future). It will have a special env
variable named `$LAST_VERSION_HASH` so the returned versions have to be NEW ones.

##### `path`

The name of or path to the executable to run.

##### `args`

Arguments to pass to the command.

#### `pull`

When a Job has a `get` to a resource, when the Job is executed the `resource.pull` is ran beforehand and the `job.task` is ran on the same context so you can pull from GIT and the `job.task` will have access to the pulled repository locally. It will have a special env variable named `$VERSION_HASH` which is the version to pull (same one returned on the `check`)

##### `path`

The name of or path to the executable to run.

##### `args`

Arguments to pass to the command.

#### `push`

NOT YET IMPLEMENTED. But will be used to push new content to the Resource Type

#### Example

This pulls the git repo and checks for the new Versions (commits in this case)

```hcl
resource_type "git" {
  inputs = [
    "url",
    "name",
  ]
  check {
    path = "/bin/bash"
    args = [
      "-ec",
      <<-EOT
        git clone --quiet $URL $NAME
        cd $NAME
        if [[ -n $LAST_VERSION_HASH ]]; then
          git log $LAST_VERSION_HASH..HEAD --pretty=format:"%H"
        else
          git log -1 --pretty=format:"%H"
        fi
      EOT
    ]
  }
  pull {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
        git clone $URL $NAME
        cd $NAME
        git checkout $VERSION_HASH
      EOT
    ]
  }
  push {
    path = "/bin/sh"
    args = [
      <<-EOT
        cd $NAME
        git push
      EOT
    ]
  }
}
```

### Resource

Resources are the implementation of the Resource Type by initializing it with the inputs defined on the `resource_type.inputs`.

When defined a `resource` the first label is the `resource_type` and the 2nd is the name of the resource.

#### Example

```hcl
resource "git" "my_repo" {
  url = "https://github.com/xescugc/qid.git"
  name = "qid"
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

#### `task`

Tasks are the ones that run the logic of the Job

##### `run`

###### `path`

The name of or path to the executable to run.

###### `args`

Arguments to pass to the command.

#### Example

3 jobs, `test` depends on `gen` and `gen` and `notify_slack` are triggered by the `git.my_repo` `resource`

```hcl
job "gen" {
  get "git" "my_repo" {
    trigger = true
  }
  task "gen" {
    run {
      path = "make"
      args = [ 
        "-C",
        "/qid",
        "gen"
      ]
    }
  }
}

job "notify_slack" {
  get "git" "my_repo" {
    trigger = true
  }
  task "notify" {
    run {
      path = "potato"
      args = [ 
        "slack",
      ]
    }
  }
}

job "test" {
  get "git" "my_repo" {
    passed  = ["gen"]
    trigger = true
  }
  task "test" {
    run {
      path = "make"
      args = [ 
        "-C",
        "/qid",
        "test"
      ]
    }
  }
}
```

## CLI

Not all the API is ported to the CLI [yet](https://github.com/xescugc/qid/issues/58), it exists on the `qid/transport/http/client` as a GO client if need be.

To have access to the client just run `qid client` and you can CRUD Pipelines under `qid client pipelines` and
you can also `jobs trigger`.
