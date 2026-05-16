# Secret Types

A secret type defines how PikoCI fetches secrets from an external system (e.g. Vault, a JSON file). It has one operation: **get** (fetch secret values).

## Defining a secret type

```hcl
secret_type "vault" {
  params = ["path"]

  get "exec" {
    path = "/bin/sh"
    args = ["-ec", "vault kv get -format=json $param_path | jq -c '.data.data // .data'"]
  }
}
```

| Field    | Required | Description                                         |
|----------|----------|-----------------------------------------------------|
| `name`   | yes      | Label on the block                                  |
| `source` | no       | URL to fetch the definition from (mutually exclusive with inline `get`) |
| `params` | no       | List of parameter names the secret type accepts      |
| `get`    | yes*     | Runner command to fetch the secret values            |

\* Not required when `source` is set.

### The get operation

**get** must output a JSON object on its last stdout line. Each key-value pair becomes a `secret_<key>` environment variable available to the step. Example output:

```json
{"username": "admin", "password": "s3cret"}
```

This produces `secret_username=admin` and `secret_password=s3cret` in the step's environment.

### Environment variables

Inside the `get` command, PikoCI exposes:

| Variable            | Description                                  |
|---------------------|----------------------------------------------|
| `$param_<name>`     | Secret instance parameter value              |
| `$WORKDIR`          | Temporary working directory for the job      |
| `$path`, `$args`    | Runner `run` block values (for the exec runner) |

## Sourcing from URL

Instead of defining the `get` command inline, you can point to an external HCL file:

```hcl
secret_type "my-vault" {
  source = "pikoci://vault"
}
```

Two URL formats are supported:

- **`pikoci://<name>`** resolves to the PikoCI registry. For shipped built-ins (`vault`, `file`), the embedded definition is used directly (no network call). For other names, fetches from `https://raw.githubusercontent.com/xescugc/pikoci/master/pikoci/builtin/secret_types/<name>.hcl`.
- **`https://...`** or **`http://...`** fetches HCL from any URL.

When `source` is set, you must not define an inline `get` block. PikoCI will error if both are present.

## Defining a secret

A secret is an instance of a secret type with concrete parameter values:

```hcl
secret "vault" "db-creds" {
  path = "secret/data/db"
}
```

| Field  | Required | Description                                      |
|--------|----------|--------------------------------------------------|
| `type` | yes      | Label, must match a `secret_type` name           |
| `name` | yes      | Label, unique name for this secret               |
| other  | no       | Remaining attributes are passed as params filtered by `secret_type.params` |

The secret canonical is `<type>.<name>` (e.g. `vault.db-creds`).

## Using secrets in steps

Reference secrets by canonical name in `get`, `task`, or `put` steps:

```hcl
job "deploy" {
  get "cron" "timer" {
    trigger = true
  }
  task "migrate" {
    secrets = ["vault.db-creds"]
    run "exec" {
      path = "make"
      args = ["migrate"]
      # $secret_username and $secret_password available as env vars
    }
  }
}
```

Secrets are fetched once per step execution. If the fetch fails, the step fails immediately before the runner executes.

## Built-in: vault

The `vault` secret type is built in. It uses the [Vault CLI](https://developer.hashicorp.com/vault/docs/commands) to fetch secrets from HashiCorp Vault. Requires `vault` and `jq` to be installed on the worker.

```hcl
secret_type "my-vault" {
  source = "pikoci://vault"
}

secret "my-vault" "db-creds" {
  path = "secret/data/db"
}
```

### Params

| Param  | Required | Description                                      |
|--------|----------|--------------------------------------------------|
| `path` | yes      | Vault KV path to read (e.g. `secret/data/db`)   |

### Vault configuration

The worker must have the `VAULT_ADDR` and `VAULT_TOKEN` (or another auth method) environment variables set. The built-in `get` command runs:

```sh
vault kv get -format=json "$param_path" | jq -c '.data.data // .data'
```

This handles both KV v1 (`.data`) and KV v2 (`.data.data`) secret engines.

### Example

```hcl
variable "vault_path" {
  type    = string
  default = "secret/data/app"
}

secret_type "my-vault" {
  source = "pikoci://vault"
}

secret "my-vault" "app-secrets" {
  path = var.vault_path
}

job "deploy" {
  get "cron" "timer" { trigger = true }
  task "deploy" {
    secrets = ["my-vault.app-secrets"]
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "echo Deploying with user=$secret_username"]
    }
  }
}
```

## Built-in: file

The `file` secret type is built in. It reads a JSON file from disk and exposes its keys as secret environment variables.

```hcl
secret_type "my-file" {
  source = "pikoci://file"
}

secret "my-file" "creds" {
  path = "/etc/pikoci/secrets/db.json"
}
```

### Params

| Param  | Required | Description                                      |
|--------|----------|--------------------------------------------------|
| `path` | yes      | Absolute path to a JSON file on the worker       |

The file must contain a JSON object:

```json
{"username": "admin", "password": "s3cret", "host": "db.example.com"}
```

### Example

```hcl
secret_type "my-file" {
  source = "pikoci://file"
}

secret "my-file" "db" {
  path = "/run/secrets/db.json"
}

job "deploy" {
  get "cron" "timer" { trigger = true }
  task "migrate" {
    secrets = ["my-file.db"]
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "DATABASE_URL=postgres://$secret_username:$secret_password@$secret_host/app make migrate"]
    }
  }
}
```

## Example: custom secret type

You can define any secret type with a custom `get` command. The only requirement is that the last line of stdout is a JSON object:

```hcl
secret_type "aws-ssm" {
  params = ["name", "region"]

  get "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      "aws ssm get-parameter --name $param_name --region $param_region --with-decryption --query 'Parameter.Value' --output text"
    ]
  }
}

secret "aws-ssm" "api-key" {
  name   = "/prod/api-key"
  region = "us-east-1"
}

job "deploy" {
  get "cron" "timer" { trigger = true }
  task "use-key" {
    secrets = ["aws-ssm.api-key"]
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "curl -H 'Authorization: $secret_value' https://api.example.com"]
    }
  }
}
```
