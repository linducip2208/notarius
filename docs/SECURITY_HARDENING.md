# Hardening Keamanan (bukti: kode + E2E + tes)

## Lockout akun (baru)
- 5x login gagal dalam 15 menit → akun terkunci 15 menit (per email, lintas IP).
- Upaya gagal/berhasil dicatat di `login_attempts` (IP + waktu); sukses menghapus riwayat gagal.
- Kunci + respons memakai pesan generik (tidak membocorkan email terdaftar vs kata sandi salah).
- Bukti: curl 6x kata sandi salah → upaya ke-6 menampilkan "Akun terkunci sementara…".

## Cookie & token
- Cookie CSRF kini `HttpOnly` (token disuntik server-side ke form; tidak butuh dibaca JS).
- Token pratinjau draf HMAC kini kedaluwarsa 24 jam (dulu abadi); tetap butuh sesi staf.

## Lapisan yang sudah ada (ringkas)
bcrypt-12 · CSRF double-submit · rate-limit login/IP · sesi DB + cleanup · SameSite/Secure/HSTS/nosniff/SAMEORIGIN/CSP ·
RBAC server-side · NIK AES-GCM · MFA TOTP · sanitasi HTML · validasi upload + re-encode · audit imutabel ·
kuota/kerahasiaan DMS · tautan berbagi bertoken kedaluwarsa.

## Sisa jujur (butuh aksi deploy/pihak ketiga)
- HTTPS + `COOKIE_SECURE=true` di VPS (wajib sebelum produksi).
- Pen-test independen, audit screen-reader, `go test -race` di Linux CI.
- CSP masih `unsafe-inline` (kebutuhan Tabler); AV mengandalkan pindai on-demand.
