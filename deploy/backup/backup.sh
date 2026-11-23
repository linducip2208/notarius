#!/usr/bin/env bash
# Encrypted MySQL backup. Requires: DB_* env, GPG_RECIPIENT.
set -euo pipefail
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
OUT="/var/backups/notaryos/notaryos-${STAMP}.sql.gz.gpg"
mkdir -p "$(dirname "$OUT")"
mysqldump -h "${DB_HOST:-127.0.0.1}" -P "${DB_PORT:-3306}" -u "${DB_USER:?}" -p"${DB_PASS:?}" \
  --single-transaction --routines --events --no-tablespaces "${DB_NAME:-notaryos}" \
  | gzip | gpg --encrypt --recipient "${GPG_RECIPIENT:?}" -o "$OUT"
echo "backup: $OUT"
# Full-dump mencakup SEMUA tabel (termasuk ppat_register [021], office_flags [023],
# doc_checklist_* [011], ai_results [004]) — tak perlu daftar per-tabel.
