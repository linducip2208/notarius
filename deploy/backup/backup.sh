#!/usr/bin/env bash
# Encrypted MySQL backup. Requires: DB_* env, GPG_RECIPIENT.
set -euo pipefail
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
OUT="/var/backups/notaryos/notaryos-${STAMP}.sql.gz.gpg"
mysqldump -h "${DB_HOST:-127.0.0.1}" -u "${DB_USER:?}" -p"${DB_PASS:?}" \
  --single-transaction --routines --events "${DB_NAME:-notaryos}" \
  | gzip | gpg --encrypt --recipient "${GPG_RECIPIENT:?}" -o "$OUT"
echo "backup: $OUT"
