# PROJECT MASTER AUDIT — SaktiNotaris (2026-11-21)

Metode: baca seluruh repo (Go packages, routes, migrasi 001-011, templates, deploy, docs).
Fokus audit ini: kebenaran implementasi vs klaim; bug kritis; celah izin; integritas data.

## Temuan kritis (sudah diperbaiki sesi ini)
1. OCR tidak tersimpan (ai_results INSERT tanpa kolom fields) → DIPERBAIKI + diindeks FULLTEXT.
2. OCR context nil + tanpa timeout → DIPERBAIKI (context + 90 detik).
3. PDF terpotong 1 halaman → DIPERBAIKI (paginasi multipage + footer + test).
4. 14 handler tanpa audit (invoice, payment, legalisasi, gov.*, template, corp, janji, user.link, file.*) → DIPERBAIKI (semua tercatat).
5. 6 celah izin (govAdvance tanpa izin, templateEdit/deeds/reporte/settings terbuka, unduhan akta tanpa cek) → DIPERBAIKI (forbidden + isolasi kantor + portal-klinik).
6. Migrasi 011 gagal boot bila sudah teraplikasi manual → prosedur: tandai schema_migrations; file tetap idempoten-urutan.

## Arsitektur (valid, pertahankan)
Go chi + MySQL 8, Tabler lokal + htmx, sesi SQL, RBAC kode + cermin DB (role_permissions, migrasi 010, 79 baris),
storage privat lokal (bisa pindah S3 tanpa ubah bisnis), OCR Tesseract lokal, audit imutabel.

## Status modul (ringkas; detail: PROJECT_FEATURE_SCORECARD.md)
- Auth/session/MFA/CSRF/rate-limit: VERIFIED (test hijau).
- RBAC 9/19 + matriks UI /roles + isolasi kantor: VERIFIED (uji silang manual pending otomatis).
- DMS inti (library, detail, preview otorisasi, versi, workflow+SoD, OCR queue+retry, search FTS, share expiring, arsip, retensi+hold, duplikat, audit akses): IMPLEMENTED, uji integrasi + E2E browser berjalan.
- Keuangan (invoice multi-baris, anti-overpay, rekonsiliasi): VERIFIED (test).
- Repertorium auto + protokol + legalisasi + wasiat handover: VERIFIED (test + E2E).
- Deploy: systemd/nginx/backup-GPG + Dockerfile Go+Tesseract-ID: IMPLEMENTED, belum uji restore di VPS.
