# SECURITY_AUDIT — DRAF v0.1 (2026-11-20)
Kuat: bcrypt-12, sesi 256-bit server-side + revoke, RBAC granular server-side (diuji),
parameterized query menyeluruh, CSP allowlist CDN, audit append-only, tanpa secret di repo,
file privat non-publik + guard traversal.
Kurang (sebelum publik): rate limiting login, CSRF token per-form, TOTP MFA, enkripsi
field NIK (kolom siap), security headers HSTS (butuh TLS), uji penetrasi, review dependensi
(`go list -m -versions`, Dependabot/Snyk). Tidak ada kerentanan kritis yang diketahui
pada kode sendiri; verifikasi ulang wajib tiap sprint.
