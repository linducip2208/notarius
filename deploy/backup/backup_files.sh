#!/usr/bin/env bash
# Backup berkas upload storage LOKAL (KTP, minuta, bukti).
# Pasangan dari backup.sh (DB-only): tanpa skrip ini, restore DB saja
# tidak mengembalikan berkas bila disk/server hilang.
#
# Mode storage (internal/storage/s3.go FromEnv):
#   - Lokal (default): NOTARYOS_S3_ENDPOINT kosong -> berkas di STORAGE_DIR.
#     Skrip ini melakukan rsync snapshot per-stamp.
#   - S3: NOTARYOS_S3_ENDPOINT diset -> skrip SKIP (proteksi via
#     versioning + lifecycle bucket, lihat docs/DEPLOYMENT_GUIDE.md).
#
# Env (tanpa kredensial di dalam skrip):
#   STORAGE_DIR          sumber berkas (default: ./storage; Docker: /data)
#   FILES_BACKUP_DIR     tujuan snapshot (default: /var/backups/notaryos/files)
#   FILES_RETENTION_DAYS retensi snapshot dalam hari (default: 30)
#   NOTARYOS_S3_ENDPOINT bila diset -> mode S3, lewati rsync lokal
#
# Cron contoh (jalankan bersamaan dengan backup.sh harian):
#   0 2 * * *  notaryos  /opt/notaryos/deploy/backup/backup.sh
#   15 2 * * * notaryos  /opt/notaryos/deploy/backup/backup_files.sh
set -euo pipefail

SRC="${STORAGE_DIR:-./storage}"
DEST_BASE="${FILES_BACKUP_DIR:-/var/backups/notaryos/files}"
RETENTION="${FILES_RETENTION_DAYS:-30}"
STAMP=$(date -u +%Y%m%dT%H%M%SZ)

# Mode S3: berkas bukan di disk lokal -> jangan rsync, andalkan bucket policy.
if [ -n "${NOTARYOS_S3_ENDPOINT:-}" ]; then
  echo "backup_files: mode S3 terdeteksi (NOTARYOS_S3_ENDPOINT set) — lewati rsync lokal; pastikan versioning + lifecycle bucket aktif." >&2
  exit 0
fi

if [ ! -d "$SRC" ]; then
  echo "backup_files: storage dir tidak ditemukan: $SRC (set STORAGE_DIR)" >&2
  exit 1
fi
command -v rsync >/dev/null 2>&1 || { echo "backup_files: rsync wajib terpasang" >&2; exit 1; }

mkdir -p "$DEST_BASE"
DEST="$DEST_BASE/$STAMP"
mkdir -p "$DEST"

rsync -a --delete "$SRC"/ "$DEST"/

# Verifikasi: jumlah file + total ukuran sumber vs tujuan harus sama.
SRC_FILES=$(find "$SRC" -type f | wc -l)
DEST_FILES=$(find "$DEST" -type f | wc -l)
SRC_SIZE=$(du -sb "$SRC" | cut -f1)
DEST_SIZE=$(du -sb "$DEST" | cut -f1)
echo "backup_files: src_files=$SRC_FILES dest_files=$DEST_FILES src_bytes=$SRC_SIZE dest_bytes=$DEST_SIZE dest=$DEST"
if [ "$SRC_FILES" != "$DEST_FILES" ]; then
  echo "backup_files: VERIFIKASI GAGAL — jumlah file berbeda (src=$SRC_FILES dest=$DEST_FILES)" >&2
  exit 1
fi

# Penanda snapshot OK + symlink 'latest' untuk restore cepat.
date -u +%Y%m%dT%H%M%SZ > "$DEST/.snapshot_ok"
ln -sfn "$DEST" "$DEST_BASE/latest"

# Retensi: hapus snapshot direktori lebih tua dari RETENTION hari (jangan sentuh symlink).
find "$DEST_BASE" -maxdepth 1 -mindepth 1 -type d -mtime +"$RETENTION" -print -exec rm -rf {} +

echo "backup_files: OK $DEST (retensi ${RETENTION} hari)"
