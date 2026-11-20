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
