#!/usr/bin/env bash
# Daily Postgres backup to Cloudflare R2.
# Triggered by backup.timer at 02:00 UTC.
set -euo pipefail

source /opt/anu/.env

TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
YEAR=$(date -u +%Y)
MONTH=$(date -u +%m)
DAY=$(date -u +%d)
BACKUP_FILE="/tmp/anu-${TIMESTAMP}.dump"
R2_KEY="anu-backups/${YEAR}/${MONTH}/${DAY}/anu-${TIMESTAMP}.dump"

alert() {
    local msg="$1"
    echo "$msg" >&2
    # Send alert via msmtp (configure /etc/msmtprc for your email provider)
    echo -e "Subject: [ANU PRO] Backup FAILED\n\n${msg}" \
        | msmtp "${ALERT_EMAIL:-ops@anupro.app}" 2>/dev/null || true
}

echo "==> Dumping database"
pg_dump \
    --format=custom \
    --no-acl \
    --no-owner \
    "${DATABASE_URL}" \
    -f "${BACKUP_FILE}" || { alert "pg_dump failed at ${TIMESTAMP}"; exit 1; }

CHECKSUM=$(sha256sum "${BACKUP_FILE}" | awk '{print $1}')
SIZE=$(stat -c%s "${BACKUP_FILE}")

echo "==> Uploading to R2 (${SIZE} bytes)"
aws s3 cp "${BACKUP_FILE}" "s3://${R2_BUCKET:-anu-backups}/${R2_KEY}" \
    --endpoint-url "${R2_ENDPOINT:?R2_ENDPOINT required}" \
    --checksum-sha256 "${CHECKSUM}" \
    || { alert "S3 upload failed at ${TIMESTAMP}"; exit 1; }

echo "==> Writing manifest"
cat > /tmp/manifest.json <<EOF
{
  "timestamp": "${TIMESTAMP}",
  "file": "${R2_KEY}",
  "size_bytes": ${SIZE},
  "sha256": "${CHECKSUM}"
}
EOF
aws s3 cp /tmp/manifest.json \
    "s3://${R2_BUCKET:-anu-backups}/anu-backups/${YEAR}/${MONTH}/${DAY}/manifest.json" \
    --endpoint-url "${R2_ENDPOINT}"

echo "==> Pruning backups older than 30 days"
CUTOFF=$(date -u -d "30 days ago" +%Y%m%d)
aws s3 ls "s3://${R2_BUCKET:-anu-backups}/anu-backups/" \
    --endpoint-url "${R2_ENDPOINT}" \
    --recursive \
    | awk '{print $4}' \
    | while read -r key; do
        keydate=$(echo "$key" | grep -oP '\d{8}(?=T)' || true)
        if [[ -n "$keydate" && "$keydate" < "$CUTOFF" ]]; then
            echo "  Deleting old backup: $key"
            aws s3 rm "s3://${R2_BUCKET:-anu-backups}/${key}" \
                --endpoint-url "${R2_ENDPOINT}" || true
        fi
    done

# Cleanup
rm -f "${BACKUP_FILE}" /tmp/manifest.json

echo "==> Backup complete: ${R2_KEY}"
