# PROJECT ARCHITECTURE — SaktiNotaris (ringkas, menunjuk detail)

- Runtime: Go 1.26 chi modular monolith (`cmd/notaryos`), MySQL 8, Tabler lokal + htmx,
  sesi SQL + cleanup/jam, RBAC kode + cermin DB, workers pool notifikasi.
- Config: `internal/config` (env + auto-`.env` dev); rahasia hanya env.
- Domain: users, clients (+identities/reps), requests (workflow mesin status), corp
  (+shareholders/officers/BO), deeds (+versions/wills), doctpl, docengine, pdf, docx,
  gov (asistensi manual), finance (int64 rupiah), notify, mfa, cryptoxx (AES-GCM NIK),
  files+storage (privat, MIME sniff, sha256), ocr (Tesseract adapter/Noop),
  dms (metadata, versi, workflow+SoD, OCR queue, FTS, share, arsip, retensi, audit).
- DMS storage: `storage.Store` interface (Save/Open/...) — LocalStore kini; S3 tanpa ubah bisnis.
- HTTP: grup auth+CSRF; rute publik hanya landing/layanan/docs/robots/sitemap/llms + /s/{token}.
- Migrasi: `internal/database` idempoten via schema_migrations (001-011).
- Observability: JSON slog, /healthz, /readyz, /metrics (RBAC), graceful shutdown 15 dtk.
- Detail: docs/audit/ARCHITECTURE_REVIEW.md, docs/dms/*.md, docs/design/*.md.
