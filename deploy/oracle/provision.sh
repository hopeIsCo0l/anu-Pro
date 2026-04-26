#!/usr/bin/env bash
# Provision Oracle Cloud Always Free VM (Ubuntu 22.04, ARM).
# Idempotent — safe to re-run.
set -euo pipefail

### ── Variables ──────────────────────────────────────────────────────────────
DOMAIN="${DOMAIN:-api.anupro.app}"
APP_DIR="/opt/anu"
DATA_DIR="/opt/anu/data"

### ── System packages ────────────────────────────────────────────────────────
apt-get update -qq
apt-get install -y -qq \
    curl wget gnupg ca-certificates \
    ufw fail2ban \
    postgresql-16 postgresql-client-16 \
    awscli jq msmtp

### ── Docker ─────────────────────────────────────────────────────────────────
if ! command -v docker &>/dev/null; then
    curl -fsSL https://get.docker.com | sh
    usermod -aG docker ubuntu
fi

### ── Caddy ──────────────────────────────────────────────────────────────────
if ! command -v caddy &>/dev/null; then
    apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
    curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/gpg.key \
        | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt \
        | tee /etc/apt/sources.list.d/caddy-stable.list
    apt-get update -qq
    apt-get install -y -qq caddy
fi

### ── Firewall ───────────────────────────────────────────────────────────────
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp   comment 'SSH'
ufw allow 80/tcp   comment 'HTTP (Caddy ACME challenge)'
ufw allow 443/tcp  comment 'HTTPS'
ufw --force enable

### ── Postgres ───────────────────────────────────────────────────────────────
systemctl enable postgresql
systemctl start postgresql

# Create DB and user if not exists
sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='anu'" \
    | grep -q 1 || sudo -u postgres createdb anu

sudo -u postgres psql -tc "SELECT 1 FROM pg_roles WHERE rolname='anu'" \
    | grep -q 1 || sudo -u postgres psql -c "CREATE USER anu WITH PASSWORD '${DB_PASSWORD:?DB_PASSWORD is required}'"

sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE anu TO anu"

# WAL archiving (required for point-in-time recovery)
PG_CONF="/etc/postgresql/16/main/postgresql.conf"
grep -q "^wal_level = replica" "$PG_CONF" || echo "wal_level = replica" >> "$PG_CONF"
grep -q "^archive_mode = on"   "$PG_CONF" || echo "archive_mode = on"   >> "$PG_CONF"
systemctl reload postgresql

### ── App directory ─────────────────────────────────────────────────────────
mkdir -p "$APP_DIR" "$DATA_DIR"
chown -R ubuntu:ubuntu "$APP_DIR"

### ── Caddy config ───────────────────────────────────────────────────────────
cp "$(dirname "$0")/Caddyfile" /etc/caddy/Caddyfile
systemctl enable caddy
systemctl reload caddy || systemctl start caddy

### ── Systemd units ──────────────────────────────────────────────────────────
cp "$(dirname "$0")/anu-api.service"    /etc/systemd/system/
cp "$(dirname "$0")/anu-worker.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable anu-api anu-worker

### ── Backup timers ──────────────────────────────────────────────────────────
cp "$(dirname "$0")/backup.service"       /etc/systemd/system/
cp "$(dirname "$0")/backup.timer"         /etc/systemd/system/
cp "$(dirname "$0")/restore-test.service" /etc/systemd/system/
cp "$(dirname "$0")/restore-test.timer"   /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now backup.timer restore-test.timer

echo "Provisioning complete. Set DB_PASSWORD, deploy .env to $APP_DIR, then:"
echo "  systemctl start anu-api anu-worker"
