#!/usr/bin/env bash
set -euo pipefail

# Deploy PikoCI to a remote server.
#
# Usage: ./deploy.sh [--build] <ssh-host>
#
# By default, downloads the latest release from GitHub.
# Pass --build to build from source instead.
#
# Detects the server architecture automatically (amd64/arm64).
# Installs Docker on the server if not already present.
#
# Examples:
#   ./deploy.sh root@pikoci.com
#   ./deploy.sh --build root@pikoci.com

BUILD=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --build)
            BUILD=true
            shift
            ;;
        -*)
            echo "Unknown option: $1"
            exit 1
            ;;
        *)
            SSH_HOST="$1"
            shift
            ;;
    esac
done

if [ -z "${SSH_HOST:-}" ]; then
    echo "Usage: $0 [--build] <ssh-host>"
    echo "Example: $0 root@pikoci.com"
    exit 1
fi

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEPLOY_DIR="$REPO_ROOT/deploy"

if [ ! -f "$DEPLOY_DIR/pikoci.env" ]; then
    echo "Error: deploy/pikoci.env not found."
    echo "Create it from the template: cp deploy/pikoci.env.example deploy/pikoci.env"
    exit 1
fi

# Detect server architecture
SERVER_ARCH=$(ssh "$SSH_HOST" 'uname -m')
case "$SERVER_ARCH" in
    x86_64)  GOARCH=amd64 ;;
    aarch64) GOARCH=arm64 ;;
    *)
        echo "Error: unsupported server architecture: $SERVER_ARCH"
        exit 1
        ;;
esac
BINARY="$REPO_ROOT/builds/pikoci-linux-$GOARCH"

if [ "$BUILD" = true ]; then
    echo "==> Building PikoCI binary (linux/$GOARCH)..."
    cd "$REPO_ROOT"
    GOOS=linux GOARCH="$GOARCH" go build -o "$BINARY" .
else
    echo "==> Downloading latest PikoCI release (linux/$GOARCH)..."
    mkdir -p "$REPO_ROOT/builds"
    curl -fSL "https://github.com/xescugc/pikoci/releases/latest/download/linux-$GOARCH" -o "$BINARY"
    chmod +x "$BINARY"
fi

echo "==> Stopping PikoCI service..."
ssh "$SSH_HOST" 'systemctl stop pikoci 2>/dev/null || true'

echo "==> Copying binary to $SSH_HOST..."
scp "$BINARY" "$SSH_HOST":/usr/local/bin/pikoci

echo "==> Syncing deploy configs..."
ssh "$SSH_HOST" 'mkdir -p /opt/pikoci /etc/pikoci /var/lib/pikoci && id -u pikoci &>/dev/null || useradd --system --no-create-home pikoci && chown pikoci:pikoci /var/lib/pikoci'
scp "$DEPLOY_DIR/pikoci.service" "$SSH_HOST":/etc/systemd/system/pikoci.service
scp "$DEPLOY_DIR/docker-compose.yml" "$SSH_HOST":/opt/pikoci/docker-compose.yml
scp "$DEPLOY_DIR/Caddyfile" "$SSH_HOST":/opt/pikoci/Caddyfile
scp "$DEPLOY_DIR/prometheus.yml" "$SSH_HOST":/opt/pikoci/prometheus.yml

echo "==> Syncing env file..."
scp "$DEPLOY_DIR/pikoci.env" "$SSH_HOST":/etc/pikoci/pikoci.env
ssh "$SSH_HOST" 'chmod 600 /etc/pikoci/pikoci.env'
scp "$DEPLOY_DIR/pikoci.env" "$SSH_HOST":/opt/pikoci/pikoci.env
ssh "$SSH_HOST" 'chmod 600 /opt/pikoci/pikoci.env'
ssh "$SSH_HOST" 'rm -f /opt/pikoci/.env && grep -E "^(PIKOCI_DOMAIN|PIKOCI_GRAFANA_DOMAIN|GF_)" /opt/pikoci/pikoci.env > /opt/pikoci/.env'

# Install Docker if not present
if ! ssh "$SSH_HOST" 'command -v docker &>/dev/null'; then
    echo "==> Installing Docker on $SSH_HOST..."
    ssh "$SSH_HOST" 'curl -fsSL https://get.docker.com | sh'
fi

echo "==> Restarting PikoCI service..."
ssh "$SSH_HOST" 'systemctl daemon-reload && systemctl restart pikoci'

echo "==> Updating Docker Compose services..."
ssh "$SSH_HOST" 'cd /opt/pikoci && docker compose up -d'

echo "==> Deploy complete."
