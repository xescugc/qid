# Variables and Secrets

Pipeline variables let you parameterize your pipeline configuration.

## Declaring variables

```hcl
variable "repo_url" {
  type    = string
  default = "https://github.com/xescugc/pikoci.git"
}

variable "repo_name" {
  type = string
}
```

| Field     | Required | Description                            |
|-----------|----------|----------------------------------------|
| `type`    | yes      | Variable type (`string`)               |
| `default` | no       | Default value if not provided          |

## Providing values

Variables without a default must be set via a JSON vars file:

```json
{
  "repo_name": "pikoci"
}
```

Pass the file when creating or updating a pipeline:

```bash
# Via CLI
pikoci client pipelines create -n my-pipeline -c pipeline.hcl -v vars.json

# At server startup
pikoci server --pipeline-name my-pipeline --pipeline-config pipeline.hcl --pipeline-vars vars.json
```

## Using variables

Reference variables with `var.<name>` or string interpolation `${var.<name>}`:

```hcl
resource "git" "repo" {
  params {
    url  = var.repo_url
    name = "${var.repo_name}"
  }
}

task "build" {
  run "exec" {
    path = "/bin/sh"
    args = ["-ec", "cd ${var.repo_name} && make build"]
  }
}
```

## Secrets

Currently, sensitive values (API keys, tokens, etc.) are passed as variables. Keep your vars file out of version control:

```json
{
  "deploy_token": "secret-value"
}
```

```hcl
variable "deploy_token" {
  type = string
}

task "deploy" {
  run "exec" {
    path = "/bin/sh"
    args = ["-ec", "curl -H 'Authorization: ${var.deploy_token}' ..."]
  }
}
```

A dedicated `secret_type` / `secret` backend system is planned for future releases.
