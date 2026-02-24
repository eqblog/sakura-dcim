#!/bin/bash
# Sakura DCIM — Automated PostgreSQL Backup Script
# Run via cron or Docker cron container.

set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/backups}"
PGHOST="${PGHOST:-postgres}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-sakura}"
PGPASSWORD="${PGPASSWORD:-sakura}"
PGDATABASE="${PGDATABASE:-sakura_dcim}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"

export PGPASSWORD

DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="sakura_dcim_${DATE}.sql.gz"
FILEPATH="${BACKUP_DIR}/${FILENAME}"

mkdir -p "${BACKUP_DIR}"

echo "[$(date)] Starting backup of ${PGDATABASE}..."

pg_dump -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" \
  --no-owner --no-acl --clean --if-exists \
  | gzip > "${FILEPATH}"

SIZE=$(du -h "${FILEPATH}" | cut -f1)
echo "[$(date)] Backup completed: ${FILENAME} (${SIZE})"

# Cleanup old backups
echo "[$(date)] Removing backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -name "sakura_dcim_*.sql.gz" -mtime "+${RETENTION_DAYS}" -delete

REMAINING=$(find "${BACKUP_DIR}" -name "sakura_dcim_*.sql.gz" | wc -l)
echo "[$(date)] Backup rotation complete. ${REMAINING} backup(s) retained."
