# MASTER_AUDIT — NOTARIUS (2026-11-20, v0.4.0-dev)

## 1. Struktur & stack (fakta)
- Backend 100% Go (`go.mod` module notaryos, go 1.26.2), chi router, MySQL 8.x (`internal/database`, migrasi 001-009).
- Paket: app, auth(+sqlstore), rbac, middleware(csrf/ratelimit/security), users, clients, requests, corp, deeds(+wills), doctpl, docengine, pdf, docx, files, storage, ocr(+tesseract), gov, finance, notify, mfa, cryptoxx, validate, workers, wib, config, audit.
- UI: Tabler via CDN (`partials.gohtml` head/sidebar/page_open) + htmx 1.9.12; 7 file gohtml, ~30 template define. Metronic BELUM ada (skeleton `web/assets/metronic/` + README, BLOCKED lisensi).
- Routes: 60+ (lihat `internal/app/app.go` Router). Grup auth+CSRF; login/mfa rate-limit 10/mnt; logout kini di dalam CSRF.

## 2. Status fitur (ringkas; detail FEATURE_GAP_ANALYSIS.md)
- VERIFIED (kode+uji): auth bcrypt-12, sesi SQL+cleanup/jam, MFA TOTP, RBAC 9 peran/19 izin, CRUD klien+duplikat, identitas+verifikasi manusia, reps, BO TPPU, permohonan workflow 9 status, korporat+modal, akta draft→review→approve→repertorium, repertorium, protokol create, legalisasi create+advance, wasiat register+handover, template editor+approve+klausul, docengine, PDF+DOCX stdlib, upload privat+unduh otorisasi+OCR temp-resolve, gov checklist toggle+advance scoped, invoice multi-baris+anti-overpay, pembayaran, expenses, tariff, tasks, audit log, office profile, documents center, search, metrics, dashboard/laporan real-query, backup/restore GPG, graceful shutdown.
- IMPLEMENTED (belum VERIFIED penuh): penomoran anti-race (FOR UPDATE baru; perlu uji konkuren), office-isolasi gov/files/deeds (baru; perlu uji cross-office), MIME sniff (baru; perlu uji spoof), XFF trusted-proxy (baru; perlu uji spoof), 403/404 (baru; perlu uji visual), deed detail+versions (baru; perlu uji browser), sidebar gating (baru; perlu uji peran).
- BLOCKED: migrasi Metronic (aset resmi tidak ada; dilarang bajakan), integrasi AHU/OSS resmi (tidak ada API resmi; asistensi manual by design), AI eksternal (tanpa kunci; mati jujur).

## 3. Risiko tertinggi
1. Tanpa tinjauan notaris atas alur hukum (B-06) — jangan produksi sebelum validasi praktisi.
2. Tanpa TLS publik + COOKIE_SECURE=false default — sesi/MFA sniffable bila HTTP.
3. Penomoran konkuren & isolasi antar-kantor — baru diperbaiki; wajib uji beban + cross-office sebelum klaim final.
4. Backup belum terbukti di-restore di VPS target.

## 4. Acceptance per fitur
Lihat FEATURE_GAP_ANALYSIS.md: tiap fitur punya lokasi kode, status, kekurangan, risiko, prioritas, AC, test yang dibutuhkan. IMPLEMENTED ≠ VERIFIED dicatat eksplisit.

## 5. Keputusan arsitektur
- Tetap Go+MySQL; tanpa Laravel/Rust. Tabler dipertahankan sampai Metronic resmi tersedia (anti dua design system: tidak ada campuran).
- Nomor/penomoran: tx + SELECT FOR UPDATE + UNIQUE backstop; generator server-side penuh ditunda (perlu counter table).
- File: privat lokal, unduh selalu otorisasi office+RBAC, MIME sniff, nama disanitasi.
