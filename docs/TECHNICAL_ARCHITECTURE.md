# TECHNICAL_ARCHITECTURE
Modular monolith Go: `cmd/notaryos` → `internal/app` (chi router + handler tipis +
html/template Tabler) → paket domain (`users, clients, requests, deeds, finance, ...`)
→ `database/sql` + MySQL 8.x. Aturan: logika bisnis di service/store, bukan handler;
tanpa import siklus; interface untuk OCR (`ocr.Provider`), storage (`storage.Store`),
submission pemerintah (assisted). Background: `workers.Pool` goroutine. Observabilitas:
slog JSON + `audit_logs`. UI: Tabler via CDN + HTMX (pencarian klien tanpa reload);
CSP membatasi sumber skrip/gaya. Keputusan besar dicatat di `docs/DECISIONS.md`.
