# NOTARIUS 3.0 — Panduan Deploy (delta; prosedur penuh tetap di `docs/DEPLOYMENT_GUIDE.md`)

Dokumen ini hanya merangkum + menambahkan temuan baru. Langkah VPS/MySQL/Nginx/
systemd/TLS/HEALTH jangan diulang di sini — ikut DEPLOYMENT_GUIDE langkah 1–7.

## 1. Migrasi 001–026 (otomatis saat boot; rollback = restore backup + catat `schema_migrations`)

- 001_core, 002_clients, 003_deeds, 004_ops, 005_seed (katalog 22 layanan),
006_portal, 007_sessions, 008_bo_handover, 009_tasks, 010_role_permissions.
- 011_dms, 012_plans, 013_tte, 014_merge, 015_wlabel_cms, 016_client_crm,
017_dms_enterprise, 018_cms_growth, 019_ops_growth, 020_compliance_hardening.
- 021_ppat (register PPAT + 6 layanan PPAT), 022_notaris_lengkap (PPAT-INBRENG),
023_office_strict (flag checklist ketat), 024_bundling (3 bundel),
025_indeks (5 indeks, tanpa IF NOT EXISTS — duplikat 1061 ditoleransi runner),
026_matter_workflow (`matter_parties` + `service_workflow_rules` + indeks unik wasiat).
- Urutan deploy: backup DB dulu → build biner → restart → cek log migrasi + `/readyz` 200.

## 2. Seed wajib (setelah restart, login staf)

1. `/dms/checklists` → ⚡ Pasang Standar (`POST /dms/checklists/seed`).
TEMUAN: kode berisi 27 paket (`seed_catalog.go`), bukan 20/23 seperti label lama —
5 layanan TANPA seed (WASIAT-DAFTAR, LEGALISASI, WAARMERKING, SALINAN, KUTIPAN),
tentukan manual. Tanpa seed ini gate checklist menahan SEMUA permohonan.
2. `/templates` → ⚡ Pasang Konsep (`POST /templates/seed`) → Approve SATU PER SATU
(`POST /templates/{id}/approve`, hanya notaris). TEMUAN: kode berisi 9 konsep,
label UI "8 Konsep" belum sinkron. Draf 1-klik GAGAL (fallback template pertama /
404 logis) bila belum ada template APPROVED.
3. Isi `/tariffs` + uji 1 permohonan end-to-end (langkah 4e–4f DEPLOYMENT_GUIDE).

## 3. Timer pengingat (ikut DEPLOYMENT_GUIDE langkah 10, ringkas)

- `notaryos-reminder.timer` 07:00 WIB (zona Asia/Jakarta) → `POST /reminders/run`
via skrip localhost + akun layanan `office_admin` TANPA MFA (kredensial 0600).
- Verifikasi: `systemctl list-timers`, uji `systemctl start`, cek `journalctl`
(`reminder: ok`) + jejak `/audit`. Idempoten 24 jam — aman jalan bareng ticker.

## 4. Backup: DB + BERKAS + GPG (ikut DEPLOYMENT_GUIDE langkah 8/8a, ringkas)

- DB: cron `deploy/backup/backup.sh` (full mysqldump incl. `ppat_register`,
`office_flags`, `doc_checklist_*`, `matter_parties`, `service_workflow_rules`).
- BERKAS (wajib — `backup.sh` hanya DB): cron `deploy/backup/backup_files.sh`
(`rsync -a --delete` + verifikasi count + symlink `latest` + retensi; SKIP bila S3,
proteksi via versioning + sync off-bucket).
- GPG: impor public key (`GPG_RECIPIENT`) SEBELUM cron pertama; uji restore tiap
bulan (`restore.sh`, target drill `notaryos_restore`, urutan stop→restore→promote→start).

## 5. Verifikasi pasca-deploy (wajib lulus semua)

1. `SELECT version FROM schema_migrations ORDER BY version` — HARUS memuat `026`.
2. Indeks 025: `SHOW INDEX FROM service_requests` (`idx_req_office_status`),
`SHOW INDEX FROM deed_drafts` (`idx_deed_request`, `idx_deed_office_status`),
`SHOW INDEX FROM doc_checklist_records` (`idx_checkrec_req_status`),
`SHOW INDEX FROM ppat_register` (`idx_ppat_office_status`).
3. `/readyz` 200 + `/healthz` OK; aktivasi kantor ≥ target di `/office`.
4. Seed terpasang (cek jumlah paket checklist + template APPROVED > 0) + uji 1
permohonan end-to-end + cek `/portal` terlihat.
5. JANGAN deploy ke server tak dikenal; JANGAN seed data produksi palsu.
