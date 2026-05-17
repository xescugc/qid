# Deployment

This guide covers deploying PikoCI on a single server. This setup uses in-memory pubsub and an embedded worker, which is suitable for demos and small teams.

For production scaling with multiple workers or high availability, use an external queue backend and run workers separately. See [Queue Backends](Queue) and [Running Workers Separately](Workers).

## Quick start

Download or build the binary and run it:

```bash
pikoci server \
  --jwt-secret your-secret \
  --db-system sqlite \
  --db-name pikoci.db \
  --run-worker
```

PikoCI is now running on port 8080. That's it for a minimal deploy.

## Production single-server setup

This setup runs PikoCI as a bare binary managed by systemd, with supporting services (Caddy, Prometheus, Grafana, Node Exporter) in Docker Compose. This avoids Docker-in-Docker while keeping infrastructure containerized.

All config files are in the [`deploy/`](https://github.com/xescugc/pikoci/tree/master/deploy) directory.

### 1. Install PikoCI

Copy the binary to the server:

```bash
# Build from source
GOOS=linux GOARCH=amd64 go build -o pikoci .
scp pikoci root@your-server:/usr/local/bin/pikoci

# Or download a release
curl -L https://github.com/xescugc/pikoci/releases/latest/download/linux-amd64 -o /usr/local/bin/pikoci
chmod +x /usr/local/bin/pikoci
```

### 2. Create the system user and directories

```bash
useradd --system --no-create-home pikoci
mkdir -p /var/lib/pikoci /etc/pikoci
chown pikoci:pikoci /var/lib/pikoci
```

### 3. Configure PikoCI

Copy the example env file and fill in your values:

```bash
cp deploy/pikoci.env.example deploy/pikoci.env
```

Edit `deploy/pikoci.env` with your secrets:

```bash
PIKOCI_SERVER_JWT_SECRET=your-secure-random-secret
PIKOCI_SERVER_DB_SYSTEM=sqlite
PIKOCI_SERVER_DB_NAME=/var/lib/pikoci/pikoci.db
PIKOCI_SERVER_PUBSUB_SYSTEM=mem
PIKOCI_SERVER_RUN_WORKER=true
GF_SECURITY_ADMIN_PASSWORD=your-grafana-password
```

Override the default admin password:

```bash
pikoci user-password -u admin -p your-password
# Add the output to deploy/pikoci.env as:
# PIKOCI_SERVER_USERS=admin:$2a$10$...
```

`deploy/pikoci.env` is gitignored — only the `.example` file is tracked.

See [Server Configuration](Server) for all available options.

### 4. Deploy

The deploy script copies everything to the server — binary, configs, and env files:

```bash
./deploy/deploy.sh root@your-server
```

Or deploy manually:

```bash
# Install systemd unit
cp deploy/pikoci.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now pikoci
```

Verify it's running:

```bash
systemctl status pikoci
curl http://localhost:8080/metrics
```

### 5. Supporting services

The deploy script also syncs Docker Compose configs and env files. To start them manually:

```bash
cd /opt/pikoci
docker compose up -d
```

This starts:

- **Caddy** — reverse proxy with automatic HTTPS (ports 80/443)
- **Prometheus** — scrapes PikoCI `/metrics` and Node Exporter
- **Grafana** — dashboards (accessible at `grafana.pikoci.com`)
- **Node Exporter** — host-level metrics

### 6. DNS

Point your domain's A record to the server IP:

- `pikoci.com` (or your domain) → server IP
- `grafana.pikoci.com` → server IP

Caddy handles TLS certificates automatically via Let's Encrypt.

## Secrets management

- Keep your secrets in `deploy/pikoci.env` locally (gitignored). The deploy script copies it to the server with correct permissions (`chmod 600`)
- The `pikoci.env.example` file is a template — never commit actual secrets
- Generate JWT secrets with: `openssl rand -hex 32`
- Generate password hashes with: `pikoci user-password -u <user> -p <password>`

## Monitoring

PikoCI exposes a `/metrics` endpoint in Prometheus format with:

- Go runtime metrics (goroutines, memory, GC)
- HTTP request counts by status code and method
- HTTP request duration histograms

The included `prometheus.yml` scrapes both PikoCI and Node Exporter. Add Prometheus as a data source in Grafana to build dashboards.

## Deploy script

The `deploy/deploy.sh` script automates the deploy process:

```bash
# Download latest release from GitHub and deploy
./deploy/deploy.sh root@your-server

# Or build from source instead
./deploy/deploy.sh --build root@your-server
```

It downloads the binary (or builds it with `--build`), copies it and all configs (including the env file) to the server, restarts the systemd service, and runs `docker compose up -d`.

## Self-deploy pipeline

PikoCI can deploy itself. A basic pipeline would:

1. Watch the git repo for new commits
2. Build the binary
3. Copy it to the server and restart the service

See [Pipeline Reference](Pipeline) for how to set up git resources and task steps.
