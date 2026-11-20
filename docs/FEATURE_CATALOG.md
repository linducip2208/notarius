# FEATURE_CATALOG — v0.3.1 (28 fitur, tanpa fitur artifisial)
Konvensi status: PLANNED | IN_PROGRESS | BLOCKED | IN_REVIEW | TESTING | COMPLETED | DEFERRED.
Detail implementasi dirujuk ke file kode yang tercantum — status COMPLETED berarti kode + test hijau.

## Modul A — Fondasi & Akses
- **F-001 Auth sesi aman** — login bcrypt cost-12, sesi server-side SQL, logout revoke, MFA 2 langkah.
  Status: COMPLETED (`internal/auth`, `internal/app`).
- **F-002 RBAC 9 peran / 19 izin** — penegakan server-side via middleware. Status: COMPLETED.
- **F-003 Manajemen kantor & pengguna** — CRUD kantor, daftar pengguna, peran, taut klien, seed idempoten. Status: COMPLETED.
- **F-004 Migrasi MySQL idempoten** — `schema_migrations`, 8 berkas, FK/indeks/DECIMAL. Status: COMPLETED.
- **F-005 Audit log imutabel** — login, klien, permohonan, akta, gov, wasiat, BO, protokol, legalisasi. Status: COMPLETED.

## Modul B — Klien
- **F-006 Klien perorangan & korporat** — validasi NIK/email/telepon/NPWP. Status: COMPLETED.
- **F-007 Deteksi duplikat** — telepon sama / nama mirip (SOUNDS LIKE). Status: COMPLETED.
- **F-008 Identitas + verifikasi manusia** — skema `identities` + `verified_by/at`; OCR tidak pernah auto-verify. Status: COMPLETED.
- **F-009 Perwakilan & beneficial owner** — representatives + beneficial_owners (TPPU, PEP, %). Status: COMPLETED (`internal/corp`, migrasi 008, test integrasi).

## Modul C — Permohonan & Korporat
- **F-010 Katalog layanan** — seed 22 layanan + price_items. Status: COMPLETED.
- **F-011 Permohonan + workflow status** — 9 status, transisi legal diuji. Status: COMPLETED.
- **F-012 Data korporat (PT/CV/Firma/Yayasan/Koperasi)** — companies, shareholders, officers, BO, modal + UI penuh. Status: COMPLETED.
- **F-013 Perjanjian (kerjasama, utang, sewa, kuasa, prenup)** — via jenis akta (Agreement/POA/Prenup/Other) + template + PDF/DOCX. Status: COMPLETED.
- **F-014 Wasiat & waris (akses terbatas)** — register + serah-terima notaris-only + audit. Status: COMPLETED.

## Modul D — Kenotariatan
- **F-015 Draf akta berversi + kunci final** — hanya dari template approved; Review→Approve→repertorium otomatis. Status: COMPLETED.
- **F-016 Repertorium** — nomor unik per kantor, otomatis saat pengesahan. Status: COMPLETED.
- **F-017 Protokol (minuta/salinan/grosse)** — bendel per tahun + form create + retensi 25 th. Status: COMPLETED.
- **F-018 Legalisasi/Waarmerking/Salinan/Kutipan** — register + biaya + COMPLETED→HANDED_OVER. Status: COMPLETED.
- **F-019 Template + klausul approved-notaris** — editor + preview + missing-vars + approval gate. Status: COMPLETED.

## Modul E — Dokumen, AI, Pemerintah
- **F-020 Doc engine {{var}}** — render, missing-vars, hash versi, watermark draft. Status: COMPLETED.
- **F-021 Ekspor PDF/DOCX** — PDF stdlib + DOCX stdlib (zip) + watermark DRAFT. Status: COMPLETED.
- **F-022 OCR lokal + provenance + verifikasi manusia** — Tesseract adapter + temp-resolve + Noop jujur. Status: COMPLETED.
- **F-023 Asistensi AHU/OSS** — checklist per-item toggle + bukti, tanpa endpoint liar. Status: COMPLETED.
- **F-024 Upload privat + otorisasi unduhan** — upload privat + GET /files/{id} office-scoped + OCR. Status: COMPLETED.

## Modul F — Keuangan, Portal, Operasi
- **F-025 Invoice + pembayaran + rekonsiliasi** — int64 rupiah, multi-baris, anti-overpay. Status: COMPLETED.
- **F-026 Portal klien terisolasi** — akun client hanya data sendiri (server-side). Status: COMPLETED.
- **F-027 Dashboard & laporan real-query + /metrics** — tanpa angka fabrikasi. Status: COMPLETED.
- **F-028 Deploy & backup** — systemd, nginx + HSTS/TLS, backup/restore GPG, monitoring. Status: COMPLETED.
