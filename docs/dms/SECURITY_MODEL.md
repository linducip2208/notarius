# DMS — Model Keamanan

## 1. Prinsip

- Deny-by-default: tanpa sesi + permission, tidak ada akses berkas.
- Isolasi kantor (`office_id`): user hanya menyentuh `files` kantornya.
- Tidak ada URL publik; unduhan selalu via `GET /files/{id}` yang diautorisasi.
- Audit imutabel: `audit_logs` hanya append.

## 2. RBAC

- Role bawaan (seed `004_ops.sql`): `super_admin, office_admin, notary,
  substitute_notary, staff, finance, doc_operator, client, auditor`.
- Permission relevan: `client.view/create/edit`, `identity.view/verify`,
  `deed.draft/edit/approve`, `protocol.view`, `repertorium.manage`,
  `reports.view`, `office.manage`.
- Mapping role→permission di `010_role_permissions.sql`; enforcement di
  middleware + handler (`internal/rbac`, `internal/middleware`).

## 3. Kontrol per aksi DMS

| Aksi | Kontrol |
|---|---|
| Upload `POST /clients/{id}/files` | sesi + permission client/file; cek `office_id` klien; sniff MIME server-side; batas 100 MB; nama disanitasi |
| Unduh `GET /files/{id}` | sesi; cek `office_id` file vs user; 403 bila beda kantor (bukan 404-bocor bila kebijakan memilih 404) |
| OCR `POST /files/{id}/ocr` | sama dengan unduh + permission operator; tidak mengirim data sensitif ke AI eksternal tanpa otorisasi |
| Verifikasi hasil | permission `identity.verify`; catat `verified_by/at` + audit |

## 4. Integritas & anti-spoof

- `sha256` disimpan saat upload; unduhan/laporan dapat memverifikasi ulang.
- MIME di-sniff server, bukan dari header `Content-Type` klien.
- `ref` dibersihkan via `filepath.Clean` + validasi panjang (anti traversal).
- Session di `sessions` (`007_sessions.sql`); login rate-limit; MFA tersedia (`internal/mfa`).

## 5. Yang belum diklaim

- Tidak ada enkripsi-at-rest per-file di `LocalStore`; kerahasiaan mengandalkan
  permission FS + isolasi host. Bila butuh, tambahkan sebagai ADR, bukan klaim.
- Belum ada `UNIQUE(sha256, office_id)` dan FK polimorfik penuh — lihat ARCHITECTURE §5.
