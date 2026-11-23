# DEPLOYMENT_GUIDE (Linux VPS + Nginx + systemd)
1. Siapkan VPS Ubuntu 24.04 + MySQL 8: `mysql < deploy/mysql/init.sql`, ubah password.
2. Buat user: `useradd -r -m notaryos`; salin repo ke `/opt/notaryos`; `go build -o bin/notaryos ./cmd/notaryos`.
3. Env: `/etc/notaryos/env` (0600, root:notaryos) berisi DB_* (lihat `.env.example`).
4. Jalankan migrasi saat boot otomatis (main.go) — rollback: restore backup + catat versi `schema_migrations`.
5. `cp deploy/systemd/notaryos.service /etc/systemd/system/ && systemctl enable --now notaryos`.
6. `cp deploy/nginx/notaryos.conf /etc/nginx/sites-enabled/`; `certbot --nginx` untuk TLS; set `COOKIE_SECURE=true`.
7. Health: `/healthz` (liveness), `/readyz` (DB). Log: journald JSON (slog).
8. Backup: cron `deploy/backup/backup.sh` (GPG, off-site); uji restore tiap bulan (`restore.sh`).
9. JANGAN deploy otomatis ke server tak dikenal; JANGAN seed data produksi palsu.
10. Go-Live v0.4.0 (setelah restart, migrasi 021–024 jalan otomatis — cek `schema_migrations`):
  a. Backup DB dulu: `deploy/backup/backup.sh` (wajib sebelum deploy biner).
  b. Deploy biner: `go build -o bin/notaryos ./cmd/notaryos`, lalu `systemctl restart notaryos`.
  c. Pastikan migrasi otomatis sukses: `journalctl -u notaryos | grep -i migrat` + `/readyz` 200.
  d. Login staf → buka `/dms/checklists` → klik ⚡ Pasang Standar (seed 23 paket checklist).
  e. Buka `/templates` → klik ⚡ Pasang 8 Konsep → Approve tiap template (DRAFT→APPROVED) sebelum dipakai produksi.
  f. Uji 1 permohonan end-to-end: buat permohonan → draf 1-klik → unggah berkas → verifikasi 2-kolom → invoice DP → portal timeline terlihat.
