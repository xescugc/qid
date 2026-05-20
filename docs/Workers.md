# Running Workers Separately

By default, PikoCI runs an embedded worker inside the server process. For production setups, you can run workers as separate processes on different machines.

## Architecture

```
                    ┌─────────┐
                    │  Server  │  --run-worker=false
                    └────┬────┘
                         │ pub/sub queue
              ┌──────────┼──────────┐
              │          │          │
         ┌────┴───┐ ┌───┴────┐ ┌──┴─────┐
         │Worker 1│ │Worker 2│ │Worker 3│
         └────────┘ └────────┘ └────────┘
```

The server publishes jobs to a queue. Workers subscribe, execute the jobs, and report results back via the PikoCI API.

## Requirements

- A non-memory queue backend (`nats`, `rabbit`, or `kafka`). The `mem` backend only works within a single process.
- Workers must be able to reach the server URL and the queue backend.
- Workers must use the same `--jwt-secret` as the server.

## Server setup

Disable the embedded worker:

```bash
pikoci server \
  --jwt-secret my-secret \
  --db-system postgresql \
  --db-host db.example.com \
  --db-port 5432 \
  --db-user pikoci \
  --db-password secret \
  --db-name pikoci \
  --pubsub-system nats \
  --run-worker=false
```

## Worker setup

```bash
pikoci worker \
  --pikoci-url http://server:8080 \
  --pubsub-system nats \
  --jwt-secret my-secret \
  --concurrency 4
```

## Worker flags

| Flag | Alias | Default | Required | Description |
|------|-------|---------|----------|-------------|
| `--pikoci-url` | `-u` | `localhost:8080` | no | PikoCI server URL |
| `--pubsub-system` | | `mem` | no | Queue backend (must match server) |
| `--concurrency` | | `1` | no | Number of parallel job goroutines |
| `--log-level` | | `info` | no | Log level: `debug`, `info`, `warn`, `error` |
| `--jwt-secret` | | | **yes** | Must match the server's `--jwt-secret` |
| `--config` | `-c` | | no | Path to a config file |

## Environment variables

Worker flags can be set via environment variables:

```bash
export PIKOCI_URL=http://server:8080
export PUBSUB_SYSTEM=nats
export JWT_SECRET=my-secret
export CONCURRENCY=4
```

## Scaling

Run multiple worker instances to increase throughput. Each worker independently subscribes to the queue:

```bash
# Machine A
pikoci worker --pikoci-url http://server:8080 --pubsub-system nats --jwt-secret my-secret --concurrency 2

# Machine B
pikoci worker --pikoci-url http://server:8080 --pubsub-system nats --jwt-secret my-secret --concurrency 4
```

## Signal handling

Standalone workers support the same two shutdown modes as the server:

| Signal | Behavior |
|--------|----------|
| `SIGQUIT` | Stop accepting new jobs, wait for in-flight jobs to finish (up to 10 minutes), then exit. |
| `SIGTERM` / `SIGINT` | Cancel running jobs and exit immediately. |

```bash
# Graceful shutdown
kill -QUIT $(pidof pikoci)

# Immediate shutdown
kill -TERM $(pidof pikoci)
```
