#!/usr/bin/env bash
# Restore: decrypt + gunzip + mysql. Usage: restore.sh <backup-file> [target-db]
set -euo pipefail
SRC="${1:?backup file required}"
TARGET="${2:-${DB_NAME:-notaryos}_restore}"
gpg --decrypt "$SRC" | gunzip | mysql -h "${DB_HOST:-127.0.0.1}" -u "${DB_USER:?}" -p"${DB_PASS:?}" "$TARGET"
echo "restored to $TARGET — verify, then promote."
