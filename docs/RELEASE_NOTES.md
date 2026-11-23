# RELEASE_NOTES — v0.4.0 (2026-11-23) — OCR ENTERPRISE + DMS LENGKAP + PPAT
Ringkasan: OCR enterprise (quality gate, preprocessing, voting multi-engine, parser KTP/KK/NPWP, cek konsistensi), DMS (preview universal, dipakai-di-mana, verifikasi 2-kolom, SLA + retensi, seed 23 paket checklist, mode ketat), Todo↔akta (preview + edit), deed preview semua draf + jenis Firma/Koperasi + 7 jenis PPAT, register PPAT + blanko + resi SLA, portal timeline, finance DP + tarif otomatis, laporan (rekap kategori, top 5, SLA), wizard korporat/AJB/wasiat, request form pintar + optgroup, draf 1-klik, publik (layanan detail + bundel 3 paket), PII masking, notifikasi WA manual, migrasi 021–024 (idempoten, jalan otomatis saat boot), docs (8 langkah + checklist cetak + panduan uji).
Go-Live: backup DB → deploy biner → restart (migrasi otomatis) → klik ⚡ Pasang Standar di /dms/checklists → ⚡ Pasang 8 Konsep di /templates → Approve template → uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.3.0 (2026-11-20) — FINAL INTERNAL, SIAP TINJAU NOTARIS
NOTARYOS 100%: 28/28 fitur COMPLETED. Go + MySQL + Tabler.
Yang baru di 0.3.0: deed Review/Sahkan + repertorium otomatis, beneficial owner TPPU, serah-terima wasiat, checklist AHU/OSS per-item, HSTS, session SQL + cleanup, seed idempoten, migrasi 001-008.
Cara coba: siapkan MySQL 8, `make migrate`, `make seed-admin` (env SEED_*), `go run ./cmd/notaryos`, buka /login.
Batasan produksi: wajib tinjauan notaris (B-06) + TLS publik + backup GPG teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius
