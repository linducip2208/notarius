# SECURITY_REQUIREMENTS
- Hash: bcrypt cost 12; kebijakan min 8 char (perketat + TOTP sprint 3, TD-03).
- Sesi: token 256-bit server-side, HttpOnly, SameSite=Lax, Secure bila TLS, TTL 12 jam.
- RBAC server-side di semua handler; uji di `internal/rbac/rbac_test.go`.
- Header: CSP (Tabler+HTMX CDN allowlist), nosniff, SAMEORIGIN, Referrer strict.
- SQL: parameterized `database/sql` di semua query; upload: batas 100MB, path traversal guard, tidak ada file publik.
- Secret: hanya env; tidak ada kredensial di repo (audit: nihil).
- Data: UTC di DB; NIK terenkripsi field-level sebelum produksi (TD-02); consent dicatat.
- Audit: login, klien, permohonan, akta — imutabel (append-only secara kebijakan).
- Rate limiting login + CSRF token per-form: WAJIB sebelum rilis publik (dicatat di BLOCKERS.md).
