# Secret Types

A secret type defines how PikoCI fetches secrets from an external system (e.g. Vault, a JSON file). It has one operation: **get** (fetch secret values). Connection config (address, token, etc.) is set on the secret_type block, and the path to fetch is provided inline at the step level.

## Defining a secret type

```hcl
secret_type "vault" {
  params  = ["path", "address", "token"]
  address = var.vault_address
  token   = var.vault_token

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
| `params` | no       | List of parameter names the get command accepts      |
| `get`    | yes*     | Runner command to fetch the secret values            |
| other    | no       | Config attributes (address, token, etc.) passed as `param_<key>` env vars to the get command |

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
| `$param_<name>`     | Config values from the secret_type block + `$param_path` from the step |
| `$WORKDIR`          | Temporary working directory for the job      |

## Sourcing from URL

Instead of defining the `get` command inline, you can point to an external HCL file:

```hcl
secret_type "my-vault" {
  source  = "pikoci://vault"
  address = var.vault_address
  token   = var.vault_token
}
```

Two URL formats are supported:

- **`pikoci://<name>`** resolves to the PikoCI registry. For shipped built-ins (`vault`, `file`), the embedded definition is used directly (no network call).
- **`https://...`** or **`http://...`** fetches HCL from any URL.

When `source` is set, you must not define an inline `get` block. PikoCI will error if both are present. Config attributes (like `address`, `token`) are still set on the block and merged with the resolved definition.

## Using secrets in steps

Reference a secret_type by name in `get`, `task`, or `put` steps, providing the path to fetch:

```hcl
job "deploy" {
  get "cron" "timer" {
    trigger = true
  }
  task "migrate" {
    secrets = {
      "vault" = "secret/data/db"
    }
    run "exec" {
      path = "make"
      args = ["migrate"]
      # $secret_username and $secret_password available as env vars
    }
  }
  task "cache-warmup" {
    secrets = {
      "vault" = "secret/data/redis"
    }
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "redis-cli -h $secret_host -a $secret_password PING"]
    }
  }
}
```

Each step can use a different path from the same secret_type. No need to pre-declare each secret path. Secrets are fetched once per step execution. If the fetch fails, the step fails immediately before the runner executes.

## Built-in: vault

The `vault` secret type is built in. It uses the [Vault CLI](https://developer.hashicorp.com/vault/docs/commands) to fetch secrets from HashiCorp Vault. Requires `vault` and `jq` to be installed on the worker.

```hcl
secret_type "my-vault" {
  source  = "pikoci://vault"
  address = var.vault_address
  token   = var.vault_token
}
```

### Config

| Attribute | Required | Description                                      |
|-----------|----------|--------------------------------------------------|
| `address` | yes      | Vault server address (e.g. `http://vault:8200`)  |
| `token`   | yes      | Vault authentication token                       |

### How it works

The built-in `get` command runs:

```sh
VAULT_ADDR=$param_address VAULT_TOKEN=$param_token vault kv get -format=json "$param_path" | jq -c '.data.data // .data'
```

This handles both KV v1 (`.data`) and KV v2 (`.data.data`) secret engines. The `path` comes from the step's `secrets` map.

### Vault authentication

There are three ways to provide Vault credentials:

1. **Pipeline variables** (recommended): pass `address` and `token` as pipeline variables, provided via a vars file at pipeline creation time. This keeps secrets out of the HCL.

2. **Worker environment variables**: if `VAULT_ADDR` and `VAULT_TOKEN` are set on the worker, you can define a custom secret_type that omits the address/token params.

3. **Vault agent/AppRole**: run Vault Agent on the worker with auto-auth configured.

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
  source  = "pikoci://vault"
  address = var.vault_address
  token   = var.vault_token
}

job "deploy" {
  get "cron" "timer" { trigger = true }
  task "migrate" {
    secrets = {
      "my-vault" = "secret/data/db"
    }
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "DATABASE_URL=postgres://$secret_username:$secret_password@localhost/app make migrate"]
    }
  }
}
```

Provide the token in your vars file: `{"vault_token": "hvs.CAESI..."}`.

## Built-in: file

The `file` secret type is built in. It reads a JSON file from disk and exposes its keys as secret environment variables. The path to the file is provided at the step level.

```hcl
secret_type "my-file" {
  source = "pikoci://file"
}
```

No config attributes needed. The file path comes from the step:

```hcl
task "migrate" {
  secrets = {
    "my-file" = "/run/secrets/db.json"
  }
  run "exec" { ... }
}
```

The file must contain a JSON object:

```json
{"username": "admin", "password": "s3cret", "host": "db.example.com"}
```

## Example: custom secret type

You can define any secret type with a custom `get` command. The only requirement is that the last line of stdout is a JSON object:

```hcl
secret_type "aws-ssm" {
  params = ["name", "region"]
  region = "us-east-1"

  get "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      VALUE=$(aws ssm get-parameter --name $param_path \
        --region $param_region --with-decryption \
        --query 'Parameter.Value' --output text)
      echo "{\"value\": \"$VALUE\"}"
      EOT
    ]
  }
}

job "deploy" {
  get "cron" "timer" { trigger = true }
  task "use-key" {
    secrets = {
      "aws-ssm" = "/prod/api-key"
    }
    run "exec" {
      path = "/bin/sh"
      args = ["-ec", "curl -H 'Authorization: $secret_value' https://api.example.com"]
    }
  }
}
```
