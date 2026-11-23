#!/usr/bin/env bash
# Restore: decrypt + gunzip + mysql. Usage: restore.sh <backup-file> [target-db]
# Urutan aman: 1) stop app, 2) backup DB berjalan, 3) restore ke DB _restore,
# 4) verifikasi (lihat bawah), 5) promote (rename/repoint), 6) start app + cek /readyz.
set -euo pipefail
SRC="${1:?backup file required}"
TARGET="${2:-${DB_NAME:-notaryos}_restore}"
mysql -h "${DB_HOST:-127.0.0.1}" -P "${DB_PORT:-3306}" -u "${DB_USER:?}" -p"${DB_PASS:?}" \
  -e "CREATE DATABASE IF NOT EXISTS \`$TARGET\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
gpg --decrypt "$SRC" | gunzip | mysql -h "${DB_HOST:-127.0.0.1}" -P "${DB_PORT:-3306}" -u "${DB_USER:?}" -p"${DB_PASS:?}" "$TARGET"
echo "restored to $TARGET — verifying…"
mysql -h "${DB_HOST:-127.0.0.1}" -P "${DB_PORT:-3306}" -u "${DB_USER:?}" -p"${DB_PASS:?}" "$TARGET" \
  -e "SELECT COUNT(*) AS tables_total FROM information_schema.tables WHERE table_schema=DATABASE(); SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 5; SELECT COUNT(*) AS ppat_register FROM ppat_register; SELECT COUNT(*) AS office_flags FROM office_flags;"
echo "restored to $TARGET — verify counts above, then promote (repoint app / rename DB), restart app, and check /readyz."
