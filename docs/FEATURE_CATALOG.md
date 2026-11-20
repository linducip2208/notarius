# FEATURE_CATALOG — v0.1 (28 fitur, tanpa fitur artifisial)
Konvensi status: PLANNED | IN_PROGRESS | BLOCKED | IN_REVIEW | TESTING | COMPLETED | DEFERRED.
Detail implementasi dirujuk ke file kode yang tercantum — status COMPLETED berarti kode + test hijau.

## Modul A — Fondasi & Akses
- **F-001 Auth sesi aman** — login bcrypt cost-12, sesi server-side, logout revoke, login monitoring.
  Peran: semua. Izin: —. Entitas: users, login_attempts. Test: `internal/auth/auth_test.go`. Status: COMPLETED (`internal/auth`, `internal/app` login/logout).
- **F-002 RBAC 9 peran / 19 izin** — penegakan server-side via middleware. Status: COMPLETED (`internal/rbac`, `internal/middleware`).
- **F-003 Manajemen kantor & pengguna** — CRUD kantor, daftar pengguna, peran. Status: COMPLETED (`/settings`, `internal/users`).
- **F-004 Migrasi MySQL idempoten** — `schema_migrations`, 4 berkas, FK/indeks/DECIMAL. Status: COMPLETED (`internal/database`, `migrations/`).
- **F-005 Audit log imutabel** — login, klien, permohonan, akta. Status: COMPLETED (`internal/audit`).

## Modul B — Klien
- **F-006 Klien perorangan & korporat** — validasi NIK/email/telepon/NPWP. Status: COMPLETED (`/clients`, `internal/clients`).
- **F-007 Deteksi duplikat** — telepon sama / nama mirip (SOUNDS LIKE). Status: COMPLETED (detail klien).
- **F-008 Identitas + verifikasi manusia** — skema `identities` + `verified_by/at`; OCR tidak pernah auto-verify. Status: COMPLETED (identitas + verifikasi manusia di detail klien).
- **F-009 Perwakilan & beneficial owner** — tabel representatives; kolom korporat menyusul. Status: IN_PROGRESS.

## Modul C — Permohonan & Korporat
- **F-010 Katalog layanan** — seed via `services`; persyaratan JSON per layanan. Status: COMPLETED (daftar + seed manual).
- **F-011 Permohonan + workflow status** — 9 status, transisi legal diuji. Status: COMPLETED (`internal/requests`, `/requests`).
- **F-012 Data korporat (PT/CV/Firma/Yayasan/Koperasi)** — companies, shareholders, officers, modal. Status: IN_PROGRESS (skema COMPLETED, UI input menyusul).
- **F-013 Perjanjian (kerjasama, utang, sewa, kuasa, prenup)** — via jenis akta + template. Status: IN_PROGRESS.
- **F-014 Wasiat & waris (akses terbatas)** — tabel `wills` + flag restricted. Status: COMPLETED (register wasiat akses notaris/auditor saja).

## Modul D — Kenotariatan
- **F-015 Draf akta berversi + kunci final** — hanya dari template approved; final terkunci. Status: COMPLETED (`internal/deeds`, `/deeds`).
- **F-016 Repertorium** — nomor unik per kantor. Status: COMPLETED (`/repertorium`).
- **F-017 Protokol (minuta/salinan/grosse)** — bendel per tahun. Status: COMPLETED (`/protocols`).
- **F-018 Legalisasi/Waarmerking/Salinan/Kutipan** — register + biaya. Status: COMPLETED (`/legalisasi`).
- **F-019 Template + klausul approved-notaris** — tabel + gate approval. Status: IN_PROGRESS (`/templates` list; editor menyusul).

## Modul E — Dokumen, AI, Pemerintah
- **F-020 Doc engine {{var}}** — render, missing-vars, hash versi, watermark draft. Status: COMPLETED (`internal/docengine` + test).
- **F-021 Ekspor PDF/DOCX** — Status: COMPLETED (ekspor PDF stdlib + watermark DRAFT; typesetting lanjutan menyusul).
- **F-022 OCR lokal + provenance + verifikasi manusia** — interface + Noop jujur. Status: BLOCKED (Tesseract belum terinstal; BLOCKERS.md).
- **F-023 Asistensi AHU/OSS** — checklist manual + bukti, tanpa endpoint liar. Status: COMPLETED (`internal/gov`, `/gov`).
- **F-024 Upload privat + otorisasi unduhan** — `internal/storage` + tabel files. Status: COMPLETED (upload privat + daftar berkas + OCR jujur).

## Modul F — Keuangan, Portal, Operasi
- **F-025 Invoice + pembayaran + rekonsiliasi** — int64 rupiah, anti-kelebihan bayar. Status: COMPLETED (`internal/finance`, `/finance`).
- **F-026 Portal klien terisolasi** — akun client hanya data sendiri (server-side). Status: COMPLETED (isolasi server-side via user_client_links).
- **F-027 Dashboard & laporan real-query** — tanpa angka fabrikasi. Status: COMPLETED (`/`, `/reports`).
- **F-028 Deploy & backup** — systemd, nginx, backup/restore GPG. Status: COMPLETED (`deploy/`, `docs/DEPLOYMENT_GUIDE.md`), belum diuji di VPS (TEST_RESULTS.md).

