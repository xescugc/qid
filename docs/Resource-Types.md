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
| `source` | no       | URL to fetch the definition from (mutually exclusive with inline commands) |
| `params` | no       | List of parameter names the resource type accepts   |
| `check`  | yes*     | Runner command to detect new versions               |
| `pull`   | yes*     | Runner command to fetch a specific version           |
| `push`   | no       | Runner command to publish (used by `put` steps)     |

\* Not required when `source` is set.

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

## Sourcing from URL

Instead of defining commands inline, you can point to an external HCL file:

```hcl
resource_type "my-git" {
  source = "pikoci://git"
}
```

Two URL formats are supported:

- **`pikoci://<name>`** resolves to the PikoCI registry. For shipped built-ins (`git`, `cron`), the embedded definition is used directly (no network call). For other names, fetches from `https://raw.githubusercontent.com/xescugc/pikoci/master/pikoci/builtin/resource_types/<name>.hcl`.
- **`https://...`** or **`http://...`** fetches HCL from any URL.

When `source` is set, you must not define inline `check`, `pull`, or `push` blocks. PikoCI will error if both are present.

## Overriding built-ins

All built-in resource types (`cron`, `git`) can be overridden by defining a `resource_type` block with the same name in your pipeline. Inline definitions always take precedence over built-ins.

This is useful when the built-in behavior doesn't match your needs. For example, the built-in `git` resource type uses `git ls-remote` or the GitHub/GitLab API to check for new commits. If you need a simpler check (no API, no token support) or want to add custom logic, define your own:

```hcl
resource_type "git" {
  params = ["url", "name"]

  check "exec" {
    path = "/bin/sh"
    args = ["-ec", "git ls-remote $param_url HEAD | awk '{print $1}' | jq -Rsc '[{\"ref\": .}]'"]
  }

  pull "exec" {
    path = "/bin/sh"
    args = ["-ec", "git clone $param_url $param_name && cd $param_name && git checkout $version_ref"]
  }

  push "exec" { }
}
```

This replaces the built-in `git` entirely for this pipeline. Resources using `resource "git" "..."` will use your definition instead.

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

The minimum `check_interval` is 10 seconds. Intervals shorter than 10s will be rejected on pipeline create/update.

Manual triggers (via the UI or API) and webhook triggers reset the check timer, so the next automatic check happens one full interval after the trigger.

## Built-in: git

The `git` resource type is built in with API-aware check support for GitHub and GitLab. You do not need to define it, just use it directly:

```hcl
resource "git" "my-repo" {
  params {
    url  = "https://github.com/xescugc/pikoci.git"
    name = "pikoci"
  }
}
```

### Params

| Param    | Required | Description                                      |
|----------|----------|--------------------------------------------------|
| `url`    | yes      | Repository URL (HTTPS)                           |
| `name`   | yes      | Directory name to clone into                     |
| `branch` | no       | Branch to track (defaults to HEAD)               |
| `token`  | no       | API/HTTPS auth token for private repos           |
| `pr`     | no       | Set to `"true"` to check for open pull requests instead of commits (requires `token`, GitHub/GitLab only) |

### Token setup

**GitHub**: Create a personal access token at **Settings > Developer settings > Personal access tokens > Fine-grained tokens**. The token needs **Contents** (read) permission for commit checks and cloning. For PR mode, it also needs **Pull requests** (read). For private repos, the token must have access to the repository.

**GitLab**: Create a project or personal access token at **Settings > Access Tokens**. The token needs the `read_repository` scope for commit checks and cloning. For PR mode (merge requests), it also needs `read_api`.

Pass the token via a pipeline variable to avoid hardcoding it:

```hcl
variable "git_token" {
  type = string
}

resource "git" "my-repo" {
  params {
    url   = "https://github.com/myorg/my-repo.git"
    name  = "my-repo"
    token = var.git_token
  }
}
```

Then provide the value in your vars file: `{"git_token": "ghp_..."}`.

### Check behavior

When `token` is provided and the URL matches a supported provider, the check uses the provider's API for efficiency:

- **GitHub** (`github.com`): Uses `GET /repos/{owner}/{repo}/commits?sha={branch}` with `Authorization: token` header
- **GitLab** (`gitlab.com`): Uses `GET /api/v4/projects/{id}/repository/commits?ref_name={branch}` with `PRIVATE-TOKEN` header
- **Other providers**: Falls back to `git ls-remote`

Without a token, all providers use `git ls-remote`.

### PR mode

When `pr = "true"` is set, the check command lists open pull requests (or merge requests on GitLab) instead of checking for commits. Each open PR becomes a version with its head SHA and PR number:

```json
[{"ref": "abc123", "pr": "42"}, {"ref": "def456", "pr": "43"}]
```

When a new PR is opened or an existing PR is updated (new commits pushed), PikoCI detects the change and triggers the job. The pull step fetches the PR's head ref so your CI runs against the PR code.

This requires a `token` and is supported on GitHub and GitLab.

### Pull behavior

Clones the repository with `git clone`, injecting the token into the HTTPS URL when provided. In PR mode, fetches the PR head ref. Otherwise, checks out the specific version ref.

### Push behavior

Pushes from the cloned directory, injecting the token into the remote URL when provided.

### Examples

Public repository:

```hcl
resource "git" "my-repo" {
  params {
    url  = "https://github.com/xescugc/pikoci.git"
    name = "pikoci"
  }
}
```

Private repository with token:

```hcl
resource "git" "private-repo" {
  params {
    url    = "https://github.com/myorg/private-repo.git"
    name   = "private-repo"
    branch = "main"
    token  = var.github_token
  }
}
```

CI on pull requests:

```hcl
resource "git" "prs" {
  params {
    url   = "https://github.com/myorg/my-repo.git"
    name  = "my-repo"
    token = var.github_token
    pr    = "true"
  }
}

job "ci" {
  get "git" "prs" {
    trigger = true
  }

  task "test" {
    run "docker" {
      image = "golang:1.23"
      cmd   = "cd my-repo && make test"
    }
  }
}
```
