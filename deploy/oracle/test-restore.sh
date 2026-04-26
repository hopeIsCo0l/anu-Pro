#!/usr/bin/env bash
# Weekly restore test. Downloads latest backup, restores to temp DB,
# runs smoke queries, drops temp DB. Alerts on failure.
set -euo pipefail

source /opt/anu/.env

RESTORE_DB="anu_restore_test"
BACKUP_FILE="/tmp/anu-restore-test.dump"

alert() {
    local msg="$1"
    echo "$msg" >&2
    echo -e "Subject: [ANU PRO] Restore Test FAILED\n\n${msg}" \
        | msmtp "${ALERT_EMAIL:-ops@anupro.app}" 2>/dev/null || true
}

cleanup() {
    sudo -u postgres dropdb --if-exists "${RESTORE_DB}" 2>/dev/null || true
    rm -f "${BACKUP_FILE}"
}
trap cleanup EXIT

echo "==> Finding latest backup"
LATEST_KEY=$(aws s3 ls "s3://${R2_BUCKET:-anu-backups}/anu-backups/" \
    --endpoint-url "${R2_ENDPOINT:?R2_ENDPOINT required}" \
    --recursive \
    | grep '\.dump$' \
    | sort | tail -1 | awk '{print $4}')

if [[ -z "$LATEST_KEY" ]]; then
    alert "No backup found in R2 bucket"
    exit 1
fi

echo "==> Downloading ${LATEST_KEY}"
aws s3 cp "s3://${R2_BUCKET:-anu-backups}/${LATEST_KEY}" "${BACKUP_FILE}" \
    --endpoint-url "${R2_ENDPOINT}" \
    || { alert "Failed to download backup ${LATEST_KEY}"; exit 1; }

echo "==> Creating restore DB"
sudo -u postgres createdb "${RESTORE_DB}"

echo "==> Restoring"
pg_restore \
    --dbname="${RESTORE_DB}" \
    --no-acl \
    --no-owner \
    "${BACKUP_FILE}" \
    || { alert "pg_restore failed for ${LATEST_KEY}"; exit 1; }

echo "==> Running smoke queries"
TENANT_COUNT=$(sudo -u postgres psql -d "${RESTORE_DB}" -t -c "SELECT COUNT(*) FROM public.tenants" 2>/dev/null || echo "-1")
USER_COUNT=$(sudo -u postgres psql -d "${RESTORE_DB}" -t -c "SELECT COUNT(*) FROM public.users" 2>/dev/null || echo "-1")

if [[ "$TENANT_COUNT" -lt 0 ]] || [[ "$USER_COUNT" -lt 0 ]]; then
    alert "Smoke queries failed — tables missing in restore from ${LATEST_KEY}"
    exit 1
fi

echo "==> Restore test passed: tenants=${TENANT_COUNT}, users=${USER_COUNT}"
echo "==> Backup: ${LATEST_KEY}"
