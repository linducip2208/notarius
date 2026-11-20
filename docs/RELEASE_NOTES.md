# RELEASE_NOTES — v0.3.0 (2026-11-20) — FINAL INTERNAL, SIAP TINJAU NOTARIS
NOTARYOS 100%: 28/28 fitur COMPLETED. Go + MySQL + Tabler.
Yang baru di 0.3.0: deed Review/Sahkan + repertorium otomatis, beneficial owner TPPU, serah-terima wasiat, checklist AHU/OSS per-item, HSTS, session SQL + cleanup, seed idempoten, migrasi 001-008.
Cara coba: siapkan MySQL 8, `make migrate`, `make seed-admin` (env SEED_*), `go run ./cmd/notaryos`, buka /login.
Batasan produksi: wajib tinjauan notaris (B-06) + TLS publik + backup GPG teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius
