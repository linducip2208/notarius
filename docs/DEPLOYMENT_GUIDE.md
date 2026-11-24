# DEPLOYMENT_GUIDE (Linux VPS + Nginx + systemd)
1. Siapkan VPS Ubuntu 24.04 + MySQL 8: `mysql < deploy/mysql/init.sql`, ubah password.
2. Buat user: `useradd -r -m notaryos`; salin repo ke `/opt/notaryos`; `go build -o bin/notaryos ./cmd/notaryos`.
3. Env: `/etc/notaryos/env` (0600, root:notaryos) berisi DB_* (lihat `.env.example`).
4. Jalankan migrasi saat boot otomatis (main.go) — rollback: restore backup + catat versi `schema_migrations`.
5. `cp deploy/systemd/notaryos.service /etc/systemd/system/ && systemctl enable --now notaryos`.
6. `cp deploy/nginx/notaryos.conf /etc/nginx/sites-enabled/`; `certbot --nginx` untuk TLS; set `COOKIE_SECURE=true`.
7. Health: `/healthz` (liveness), `/readyz` (DB). Log: journald JSON (slog).
8. Backup: cron `deploy/backup/backup.sh` (full-dump mysqldump seluruh DB incl. ppat_register, office_flags, doc_checklist_*, ai_results; GPG via `GPG_RECIPIENT`, honours DB_HOST/DB_PORT/DB_USER/DB_PASS/DB_NAME dari `/etc/notaryos/env`); uji restore tiap bulan (`restore.sh`).
   Prasyarat GPG: impor public key penerima di mesin backup (`gpg --import pubkey.asc`) SEBELUM cron pertama — tanpa ini backup gagal di tahap enkripsi. Target drill yang valid dengan kredensial app adalah `notaryos_restore` (sesuai grant).
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
10. Pengingat otomatis harian (timer systemd, 07:00 WIB).
    Prinsip: TANPA perubahan kode Go. Endpoint `POST /reminders/run`
    (internal/app/verify.go `remindersRun`) tetap butuh sesi login + izin
    `office.manage` + token CSRF — timer memakainya lewat skrip login
    localhost, bukan jalur auth baru. Aman dari pemalsuan: skrip menolak
    BASE non-localhost, kredensial hanya di file 0600, tak pernah di argv/ps.
    a. Buat akun layanan khusus: peran `office_admin` (satu-satunya peran
       operasional dengan `office.manage`; staff/finance/doc_operator/
       auditor DITOLAK endpoint ini), TANPA MFA (login skrip tak bisa
       lewati TOTP), password panjang acak.
    b. Simpan kredensial: `install -m 0600 -o root -g notaryos /dev/null /etc/notaryos/reminder.env`,
       isi `REMINDER_USER=<email akun layanan>`, `REMINDER_PASS=<password>`.
       Opsional: `REMINDER_BASE_URL=http://127.0.0.1:8080` (default ini;
       wajib localhost), `SESSION_NAME=notaryos_sess` (bila diubah).
    c. Salin unit + skrip sudah di repo: `cp deploy/systemd/notaryos-reminder.service
       deploy/systemd/notaryos-reminder.timer /etc/systemd/system/`
       (pastikan `/opt/notaryos/deploy/reminder/run-reminder.sh` executable),
       lalu `timedatectl set-timezone Asia/Jakarta` (jadwal 07:00 mengikuti
       zona waktu sistem) + `systemctl daemon-reload && systemctl enable --now notaryos-reminder.timer`.
    d. Verifikasi: `systemctl list-timers notaryos-reminder.timer` (NEXT =
       besok 07:00), uji manual `systemctl start notaryos-reminder.service`
       lalu `journalctl -u notaryos-reminder.service` harus `reminder: ok`,
       dan `/audit` memuat `reminders.run` oleh akun layanan.
    e. Idempotensi: pemindaian anti-spam 24 jam (lihat docs/PROJECT_GAP_ANALYSIS),
       jadi timer 07:00 + ticker in-process 24 jam (internal/app/app.go `New`)
       + tombol manual "Jalankan Pengingat" boleh hidup berdampingan.
    f. Alternatif bila timer tak dipakai: cron root
       `0 7 * * * notaryos /opt/notaryos/deploy/reminder/run-reminder.sh`
       dengan env dari `/etc/notaryos/reminder.env`
       (`set -a; . /etc/notaryos/reminder.env; set +a` di wrapper).
11. Go-Live v0.4.0 (setelah restart, migrasi 021–024 jalan otomatis — cek `schema_migrations`):
  a. Backup DB dulu: `deploy/backup/backup.sh` (wajib sebelum deploy biner).
  b. Deploy biner: `go build -o bin/notaryos ./cmd/notaryos`, lalu `systemctl restart notaryos`.
  c. Pastikan migrasi otomatis sukses: `journalctl -u notaryos | grep -i migrat` + `/readyz` 200.
   c1. Verifikasi migrasi 025: `mysql $DB -e "SELECT version FROM schema_migrations ORDER BY version"` harus memuat `025`; verifikasi indeks: `SHOW INDEX FROM service_requests` (lihat `idx_req_office_status`), `SHOW INDEX FROM deed_drafts` (`idx_deed_request`, `idx_deed_office_status`), `SHOW INDEX FROM doc_checklist_records` (`idx_checkrec_req_status`), `SHOW INDEX FROM ppat_register` (`idx_ppat_office_status`).
   c2. Verifikasi migrasi 026-035 (otomatis saat boot): `SELECT version FROM schema_migrations ORDER BY version` harus memuat `026` s/d `035` — 026 matter/workflow rules, 027 workflow engine, 028 audit office_id, 029 template unik, 030 quotations, 031 deed handover, 032 push subscriptions, 033 workflow SLA, 034 PPAT BPN date, 035 avatar pengguna. Bila ada yang hilang, cek `journalctl -u notaryos | grep -i migrat`.
  d. Login staf → buka `/dms/checklists` → klik ⚡ Pasang Standar (seed 23 paket checklist).
  e. Buka `/templates` → klik ⚡ Pasang 8 Konsep → Approve tiap template (DRAFT→APPROVED) sebelum dipakai produksi.
  f. Uji 1 permohonan end-to-end: buat permohonan → draf 1-klik → unggah berkas → verifikasi 2-kolom → invoice DP → portal timeline terlihat.
12. PWA produksi (installable, aman).
    a. Wajib HTTPS di produksi: service worker hanya jalan di secure context; `http://localhost` dikecualikan untuk dev.
    b. Verifikasi instalabilitas: `curl -sI https://DOMAIN/manifest.webmanifest` harus 200 + MIME `application/manifest+json`; ikon 192/512 + maskable ada; `start_url` `/app`, `scope` `/`.
    c. Di Chrome: DevTools → Application → Manifest/Service Workers hijau + tombol Install muncul; bila tidak, periksa field (b) di atas.
    d. Update SW: `/sw.js` diserve `Cache-Control: no-cache` (sudah di `internal/app/app.go` `pwaSW`); versioning via nama cache `saktinotaris-vN` di `web/assets/sw.js` (naikkan tiap rilis aset agar cache lama dibersihkan saat activate).
13. Workflow engine operasional (migrasi 027).
    a. Pasang template: login staf → `/dms/checklists` → ⚡ Pasang Standar (sekaligus seed 3 template PT-DIRIAN/PPAT-AJB/KUASA, idempoten per office+service_code+version=1); instance dibuat otomatis saat permohonan dibuat.
    b. Status instance: `ACTIVE` = berjalan (`current_step` = langkah aktif); `COMPLETED` = semua langkah DONE (`current_step` NULL). Status step: `PENDING` → `DONE` (tombol maju / auto-step kejadian nyata, diaudit `workflow.step`).
    c. Backup: full-dump `deploy/backup/backup.sh` sudah mencakup `workflow_templates`, `workflow_steps`, `workflow_instances`, `workflow_step_states` (+ `service_workflow_rules`); tanpa cron tambahan.
