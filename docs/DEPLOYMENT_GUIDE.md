# DEPLOYMENT_GUIDE (Linux VPS + Nginx + systemd)
1. Siapkan VPS Ubuntu 24.04 + MySQL 8: `mysql < deploy/mysql/init.sql`, ubah password.
2. Buat user: `useradd -r -m notaryos`; salin repo ke `/opt/notaryos`; `go build -o bin/notaryos ./cmd/notaryos`.
3. Env: `/etc/notaryos/env` (0600, root:notaryos) berisi DB_* (lihat `.env.example`).
4. Jalankan migrasi saat boot otomatis (main.go) — rollback: restore backup + catat versi `schema_migrations`.
5. `cp deploy/systemd/notaryos.service /etc/systemd/system/ && systemctl enable --now notaryos`.
6. `cp deploy/nginx/notaryos.conf /etc/nginx/sites-enabled/`; `certbot --nginx` untuk TLS; set `COOKIE_SECURE=true`.
7. Health: `/healthz` (liveness), `/readyz` (DB). Log: journald JSON (slog).
8. Backup: cron `deploy/backup/backup.sh` (full-dump mysqldump seluruh DB incl. ppat_register, office_flags, doc_checklist_*, ai_results; GPG via `GPG_RECIPIENT`, honours DB_HOST/DB_PORT/DB_USER/DB_PASS/DB_NAME dari `/etc/notaryos/env`); uji restore tiap bulan (`restore.sh`).
   Urutan restore aman: stop app → backup DB berjalan → `restore.sh <file> [db_restore]` (skrip buat DB target + verifikasi count tabel & schema_migrations otomatis) → promote (repoint app/rename DB) → start app → cek `/readyz` 200.
   8a. Backup BERKAS (wajib — `backup.sh` hanya DB; KTP/minuta/bukti ada di storage):
       Mode storage (`internal/storage/s3.go` `FromEnv`): lokal bila `NOTARYOS_S3_ENDPOINT` kosong
       (path dari `STORAGE_DIR`, default `./storage`; Docker `/data`) vs S3-compatible bila endpoint diset
       (`NOTARYOS_S3_ENDPOINT/REGION/BUCKET/ACCESS/SECRET`).
       - Lokal: cron `deploy/backup/backup_files.sh` (env: `STORAGE_DIR`, `FILES_BACKUP_DIR`
         default `/var/backups/notaryos/files`, `FILES_RETENTION_DAYS` default 30; `rsync -a --delete`
         per-stamp + verifikasi jumlah file/ukuran src vs dest + symlink `latest` + retensi otomatis;
         tanpa kredensial di skrip). Contoh cron: `15 2 * * * notaryos /opt/notaryos/deploy/backup/backup_files.sh`.
         Verifikasi restore berkas: `SRC=<storage-dir> DEST=$(readlink -f /var/backups/notaryos/files/latest); diff <(cd "$SRC" && find . -type f | sort) <(cd "$DEST" && find . -type f ! -name .snapshot_ok | sort) && echo FILES_RESTORE_OK` lalu spot-check 1 berkas (bandingkan checksum vs DB ref).
       - S3/versioning: `backup_files.sh` otomatis SKIP bila `NOTARYOS_S3_ENDPOINT` diset — proteksi via bucket:
         `aws s3api put-bucket-versioning --bucket "$NOTARYOS_S3_BUCKET" --versioning-configuration Status=Enabled`
         + lifecycle noncurrent-version expiry (±90 hari) & abort incomplete multipart;
         salinan off-bucket: `aws s3 sync s3://"$NOTARYOS_S3_BUCKET" s3://"${NOTARYOS_S3_BUCKET}-backup" --delete`
         (MinIO: `mc version enable SRC/BUCKET` + `mc mirror --overwrite SRC/BUCKET DST/BUCKET-backup`).
9. JANGAN deploy otomatis ke server tak dikenal; JANGAN seed data produksi palsu.
10. Go-Live v0.4.0 (setelah restart, migrasi 021–024 jalan otomatis — cek `schema_migrations`):
  a. Backup DB dulu: `deploy/backup/backup.sh` (wajib sebelum deploy biner).
  b. Deploy biner: `go build -o bin/notaryos ./cmd/notaryos`, lalu `systemctl restart notaryos`.
  c. Pastikan migrasi otomatis sukses: `journalctl -u notaryos | grep -i migrat` + `/readyz` 200.
   c1. Verifikasi migrasi 025: `mysql $DB -e "SELECT version FROM schema_migrations ORDER BY version"` harus memuat `025`; lalu verifikasi indeks: `SHOW INDEX FROM service_requests` (lihat `idx_req_office_status`), `SHOW INDEX FROM deed_drafts` (`idx_deed_request`, `idx_deed_office_status`), `SHOW INDEX FROM doc_checklist_records` (`idx_checkrec_req_status`), `SHOW INDEX FROM ppat_register` (`idx_ppat_office_status`).
  d. Login staf → buka `/dms/checklists` → klik ⚡ Pasang Standar (seed 23 paket checklist).
  e. Buka `/templates` → klik ⚡ Pasang 8 Konsep → Approve tiap template (DRAFT→APPROVED) sebelum dipakai produksi.
  f. Uji 1 permohonan end-to-end: buat permohonan → draf 1-klik → unggah berkas → verifikasi 2-kolom → invoice DP → portal timeline terlihat.
