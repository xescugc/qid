# Runners

Runners define how commands are executed. A runner wraps process execution so you can run jobs on the host, inside Docker containers, or in any custom environment.

## Defining a runner

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

| Field  | Required | Description                          |
|--------|----------|--------------------------------------|
| `name` | yes      | Label on the block                   |
| `run`  | yes      | Block with `path` and `args`         |
| `path` | no       | Executable path                      |
| `args` | no       | List of arguments                    |

### Variable expansion

Inside `path` and `args`, PikoCI expands:

| Variable    | Description                                 |
|-------------|---------------------------------------------|
| `$WORKDIR`  | Temporary working directory for the job     |
| `$<param>`  | Any parameter passed from a `run` block in a task |

For example, when a task uses `run "docker" { image = "golang:1.25" cmd = "make test" }`, the runner receives `$image` and `$cmd` as expandable variables.

## Using a runner

Reference a runner by name in `task`, `on_success`, `on_failure`, `ensure`, and resource type `check`/`pull`/`push` blocks:

```hcl
job "build" {
  get "git" "repo" {
    trigger = true
  }

  task "compile" {
    run "docker" {
      image = "golang:1.25"
      cmd   = "make build"
    }
  }
}
```

Parameters in the `run` block (like `image` and `cmd` above) are passed to the runner as environment variables.

## Built-in: exec

The `exec` runner is built in. It runs commands directly on the host machine:

```hcl
task "hello" {
  run "exec" {
    path = "echo"
    args = ["hello world"]
  }
}
```

The exec runner expands `$path` and `$args` from the `run` block:

```go
// Built-in exec runner definition
runner "exec" {
  run {
    path = "$path"
    args = ["$args"]
  }
}
```

You do not need to declare the `exec` runner in your pipeline. It is always available.

## Example: custom shell runner

```hcl
runner "bash" {
  run {
    path = "/bin/bash"
    args = ["-c", "$script"]
  }
}

job "example" {
  get "cron" "tick" { trigger = true }

  task "greet" {
    run "bash" {
      script = "echo hello from bash"
    }
  }
}
```
