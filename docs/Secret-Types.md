# Secret Types

A secret type defines how PikoCI fetches secrets from an external system (e.g. Vault, a JSON file). It has one operation: **get** (fetch secret values).

## Defining a secret type

```hcl
secret_type "vault" {
  params = ["path", "address", "token"]

  get "exec" {
    path = "/bin/sh"
    args = ["-ec", "VAULT_ADDR=$param_address VAULT_TOKEN=$param_token vault kv get -format=json $param_path | jq -c '.data.data // .data'"]
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
  path    = "secret/data/db"
  address = var.vault_address
  token   = var.vault_token
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
  path    = "secret/data/db"
  address = var.vault_address
  token   = var.vault_token
}
```

### Params

| Param     | Required | Description                                      |
|-----------|----------|--------------------------------------------------|
| `path`    | yes      | Vault KV path to read (e.g. `secret/data/db`)   |
| `address` | yes      | Vault server address (e.g. `http://vault:8200`)  |
| `token`   | yes      | Vault authentication token                       |

### Vault authentication

There are three ways to provide Vault credentials:

1. **Pipeline variables** (recommended): pass `address` and `token` as pipeline variables, provided via a vars file at pipeline creation time. This keeps secrets out of the HCL.

2. **Worker environment variables**: set `VAULT_ADDR` and `VAULT_TOKEN` on the worker process. In this case, pass empty strings for `address` and `token` in the secret block, or define a custom secret_type that omits those params.

3. **Vault agent/AppRole**: run Vault Agent on the worker with auto-auth configured. The agent manages token renewal and writes a token file that the Vault CLI reads automatically.

### How it works

The built-in `get` command runs:

```sh
VAULT_ADDR=$param_address VAULT_TOKEN=$param_token vault kv get -format=json "$param_path" | jq -c '.data.data // .data'
```

This handles both KV v1 (`.data`) and KV v2 (`.data.data`) secret engines.

### Example

```hcl
variable "vault_address" {
  type    = string
  default = "http://vault:8200"
}

variable "vault_token" {
  type = string
}

secret_type "my-vault" {
  source = "pikoci://vault"
}

secret "my-vault" "db-creds" {
  path    = "secret/data/db"
  address = var.vault_address
  token   = var.vault_token
}

job "deploy" {
  get "cron" "timer" { trigger = true }
  task "migrate" {
    secrets = ["my-vault.db-creds"]
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "DATABASE_URL=postgres://$secret_username:$secret_password@localhost/app make migrate"]
    }
  }
}
```

Provide the token in your vars file: `{"vault_token": "hvs.CAESI..."}`.

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
      <<-EOT
      VALUE=$(aws ssm get-parameter --name $param_name \
        --region $param_region --with-decryption \
        --query 'Parameter.Value' --output text)
      echo "{\"value\": \"$VALUE\"}"
      EOT
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
