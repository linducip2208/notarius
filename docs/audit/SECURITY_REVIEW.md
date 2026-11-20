# SECURITY_REVIEW — NOTARIUS (fakta kode, bukan klaim)

## Kekuatan (pertahankan)
- SQL 100% placeholder, interpolateParams=false; html/template auto-escape.
- bcrypt-12, sesi SQL 256-bit, expiry+cleanup, audit login; TOTP RFC6238 ±30s, pending 5 mnt HttpOnly Strict.
- AES-256-GCM NIK, key 32B env, tanpa key → last4 saja (fungsionalitas turun, jujur).
- Upload allowlist JPG/PNG/WebP/PDF + cap 100MB + sha256 + ref server-generated + LocalStore path Clean (tahan traversal); unduh office+RBAC.
- Header: nosniff, SAMEORIGIN, Referrer-Policy, Permissions-Policy, HSTS, CSP.

## Perbaikan sesi ini (P0/P1)
1. XFF spoof → hanya percaya bila peer = TRUSTED_PROXIES (default loopback); `SetTrustedProxies` dari env. Perlu uji: XFF acak tetap rate-limit.
2. CSRF logout → kini wajib token (route pindah ke grup auth).
3. OCR IDOR → cek office sebelum RunOCR.
4. Unduh header injection → sanitasi `" \r \n`, fallback "berkas".
5. MIME spoof → sniff magic bytes via DetectContentType; tolak bila header vs isi beda.
6. Approve race → SELECT FOR UPDATE dalam tx.
7. Gov advance/check → scoped office + PermGovSubmit.

## Sisa risiko (jangan klaim lolos audit)
- COOKIE_SECURE=false default + nginx :80 contoh → wajib TLS + Secure + HSTS; HSTS tanpa HTTPS percuma.
- PDF aktif polyglot, DOCX macro tidak relevan (generator sendiri), tapi viewer PDF tetap sandbox.
- Tanpa lockout akun per-user (hanya rate-limit IP), tanpa backup-code MFA, tanpa rotasi kunci NIK.
- CSP masih 'unsafe-inline' (HTMX/Tabler).
- Rahasia: pastikan /etc/notaryos/env chmod 600, .gitignore menutup .env, tanpa secret di repo.
- Backup GPG: simpan recipient + uji restore ke DB kosong; catat RTO/RPO.

## Test wajib sebelum prod
Rate-limit spoof, CSRF tanpa/salah token, cookie Secure/HttpOnly/SameSite, SQLi payload, XSS escaped, traversal `..%2f`, upload EXE sebagai PDF (harus ditolak + header CD aman), brute 11x → 429, TOTP replay gagal, decrypt round-trip + key 31B ditolak, backup decrypt+restore.
