# CLI Reference

PikoCI provides three top-level commands: `server`, `worker`, and `client`, plus a utility command `user-password`.

## Global structure

```
pikoci server   [flags]          # Start the server
pikoci worker   [flags]          # Start a standalone worker
pikoci client   [flags] <cmd>    # Interact with the API
pikoci user-password [flags]     # Generate hashed passwords
```

## client

Manage pipelines and jobs via the PikoCI API.

### Global flags

| Flag | Alias | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--url` | `-u` | `localhost:4000` | **yes** | PikoCI server URL |
| `--jwt` | | | no | JWT token (if not provided, reads from `$XDG_CONFIG_HOME/pikoci/authentication`) |

### login

Authenticate and store the JWT locally.

```bash
pikoci client -u localhost:8080 login -u admin -p admin123
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--username` | `-u` | **yes** | Username |
| `--password` | `-p` | **yes** | Password |

### pipelines

Pipeline management commands. All require `--team-canonical` (default: `main`).

| Flag | Alias | Default | Description |
|------|-------|---------|-------------|
| `--team-canonical` | `-tc` | `main` | Team scope |

#### pipelines create

```bash
pikoci client -u localhost:8080 pipelines create \
  -n my-pipeline -c pipeline.hcl -v vars.json
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--name` | `-n`, `-pn` | **yes** | Pipeline name |
| `--config` | `-c` | **yes** | Path to HCL config file |
| `--vars` | `-v` | no | Path to JSON vars file |

#### pipelines update

```bash
pikoci client -u localhost:8080 pipelines update \
  -n my-pipeline -c pipeline.hcl --public
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--name` | `-n`, `-pn` | **yes** | Pipeline name |
| `--config` | `-c` | **yes** | Path to HCL config file |
| `--vars` | `-v` | no | Path to JSON vars file |
| `--public` | | no | Make the pipeline publicly visible |

#### pipelines list

```bash
pikoci client -u localhost:8080 pipelines list
```

#### pipelines get

```bash
pikoci client -u localhost:8080 pipelines get -n my-pipeline
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--name` | `-n`, `-pn` | **yes** | Pipeline name |

#### pipelines graph

Export the pipeline as a DOT graph.

```bash
pikoci client -u localhost:8080 pipelines graph -n my-pipeline | dot -Tsvg > pipeline.svg
```

| Flag | Alias | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--name` | `-n`, `-pn` | | **yes** | Pipeline name |
| `--format` | `-f` | `dot` | no | Output format |

#### pipelines delete

```bash
pikoci client -u localhost:8080 pipelines delete -n my-pipeline
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--name` | `-n`, `-pn` | **yes** | Pipeline name |

### jobs

Job management commands. Require `--team-canonical` and `--pipeline-name`.

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--team-canonical` | `-tc` | **yes** | Team scope |
| `--pipeline-name` | `-pn` | **yes** | Pipeline name |

#### jobs get

```bash
pikoci client -u localhost:8080 jobs get -tc main -pn my-pipeline -n my-job
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--job-name` | `-n`, `-jn` | **yes** | Job name |

#### jobs trigger

Manually trigger a job.

```bash
pikoci client -u localhost:8080 jobs trigger -tc main -pn my-pipeline -n my-job
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--job-name` | `-n`, `-jn` | **yes** | Job name |

## user-password

Generate a `USERNAME:HASHED_PASSWORD` string for the server's `--users` flag.

```bash
pikoci user-password -u myuser -p mypassword
# Output: myuser:$2a$10$...
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--username` | `-u` | **yes** | Username |
| `--password` | `-p` | **yes** | Plain-text password |

## server

See [Server Configuration](Server).

## worker

See [Running Workers Separately](Workers).
