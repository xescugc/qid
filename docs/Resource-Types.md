# Resource Types

A resource type defines how PikoCI interacts with an external system. It has three operations: **check** (detect new versions), **pull** (fetch a version), and **push** (publish to the resource).

## Defining a resource type

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
| `params` | yes      | List of parameter names the resource type accepts   |
| `check`  | yes      | Runner command to detect new versions               |
| `pull`   | yes      | Runner command to fetch a specific version           |
| `push`   | no       | Runner command to publish (used by `put` steps)     |

### Operations

**check** must output a JSON array of version objects to stdout. Each object becomes a version that PikoCI tracks. Example output:

```json
[{"ref": "abc123"}, {"ref": "def456"}]
```

**pull** fetches a specific version into the working directory. The version fields are available as `$version_<key>` environment variables.

**push** publishes to the resource. Used when a job has a `put` step.

### Environment variables

Inside `check`, `pull`, and `push` commands, PikoCI exposes:

| Variable            | Description                                  |
|---------------------|----------------------------------------------|
| `$param_<name>`     | Resource instance parameter value            |
| `$version_<key>`    | Version field from the last check (pull/push only) |
| `$put_<key>`        | Put step parameter value (push only)         |
| `$WORKDIR`          | Temporary working directory for the job      |
| `$path`, `$args`    | Runner `run` block values (for the exec runner) |

## Defining a resource

A resource is an instance of a resource type with concrete parameter values:

```hcl
resource "git" "my_repo" {
  params {
    url  = "https://github.com/xescugc/pikoci.git"
    name = "pikoci"
  }
}
```

| Field            | Required | Description                                           |
|------------------|----------|-------------------------------------------------------|
| `type`           | yes      | Must match a `resource_type` name                     |
| `name`           | yes      | Unique name for this resource instance                |
| `params`         | yes      | Key/value pairs matching the resource type's `params` |
| `check_interval` | no       | Schedule for automatic checks (cron syntax or `@every <duration>`) |

### Webhook triggers

Each resource gets a webhook token that can be used to trigger a check externally:

```
POST /webhooks/<webhook_token>
```

You can regenerate a webhook token via the API:

```
POST /teams/{team}/pipelines/{pipeline}/resources/{resource}/webhook_token
```

## Built-in: cron

The `cron` resource type is built in. You do not need to define it, just use it directly as a resource:

```hcl
resource "cron" "every_minute" {
  check_interval = "@every 1m"
  params {}
}
```

The cron check command outputs the current date as a version:

```json
[{"date": "Mon Jan 2 15:04:05 UTC 2006"}]
```

### Supported schedules

The `check_interval` field accepts:

- `@every <duration>`, e.g. `@every 10s`, `@every 5m`, `@every 1h`
- Standard cron expressions, e.g. `0 */5 * * *`
