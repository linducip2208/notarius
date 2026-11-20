# CHANGELOG
## [0.2.0] — 2026-11-20 — Sprint 2 (continuous execution)
- Seed katalog: 22 layanan + migrasi 005/006; runner Go idempoten (count 0).
- Keamanan: rate-limit login per-IP (diuji 10→429), CSRF semua POST, MFA TOTP stdlib + alur login 2 langkah, enkripsi NIK AES-GCM.
- Domain: identitas + verifikasi manusia, perwakilan, korporat + saham + pengurus, janji temu, template editor + klausul + approval, upload privat + OCR jujur, portal terisolasi, wasiat akses-terbatas, invoice multi-baris, PDF ekspor DRAFT.
- Bug diperbaiki dari smoke test: kolom invoice, DECIMAL→int64, kunci akta APPROVED, limiter per-IP.
- Test: 6 integrasi MySQL asli PASS + 9 paket unit PASS. Fitur: 26/28 (93%).
## [0.1.0] — 2026-11-20 — Fondasi v1 (greenfield)
- Repo kosong diaudit (INITIAL_AUDIT); diputuskan stack Go + MySQL + Tabler.
- 4 migrasi MySQL (35 domain), seed roles/permissions, kebijakan retensi.
- Auth bcrypt + sesi, RBAC 9 peran/19 izin, middleware keamanan.
- Domain: users, clients (+duplikat), requests (workflow), deeds (+repertorium/protokol/legalisasi),
  finance (invoice/PPN/pembayaran), docengine, ocr interface, gov asistensi, storage privat, workers.
- 17 halaman Tabler + HTMX; dashboard/laporan real-query.
- Unit test 7 paket hijau; `go vet` + `go build` bersih.
- Deploy: nginx, systemd, mysql init, backup/restore GPG.
