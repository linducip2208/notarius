# DMS — Panduan Admin

## 1. Prasyarat host

- Go toolchain sesuai `go.mod`, MySQL, migrasi (`migrations/`) teraplikasi.
- Tesseract + data bahasa `ind` bila OCR lokal diinginkan; tanpa itu
  aplikasi tetap jalan (fallback `NoopProvider` + entri manual).
- Direktori storage privat writable, permission `0750/0640`.

## 2. Konfigurasi

- Salin `.env.example → .env`; isi DSN MySQL, basis storage, secret sesi.
- Wiring storage di `internal/app`: default `storage.NewLocal(base)`.
  Pindah S3 = ganti konstruktor + config; ref tidak berubah.

## 3. Pengguna & peran

- Role seed: `super_admin … auditor` (`004_ops.sql`); mapping di
  `010_role_permissions.sql`.
- Beri `identity.verify` hanya ke peran yang boleh mengesahkan data.
- `auditor` read-only; `doc_operator` untuk upload/OCR harian.

## 4. Operasional rutin

- Backup berpasangan: direktori storage + dump MySQL.
- Monitor: ukuran storage, antrian `files.ocr_status=PENDING/FAILED`,
  pertumbuhan `audit_logs`.
- Restore: validasi sampling `ref → sha256` sebelum buka akses.

## 5. Insiden

- Dugaan traversal/exfiltrasi: bekukan akun, periksa `audit_logs`
  untuk `GET /files/{id}` anomali, putar secret sesi.
- Korupsi storage: identifikasi `ref` gagal `Open`, restore dari backup
  pasangan (bukan campur backup beda waktu).
- OCR massal gagal: cek biner Tesseract/`ind`, bukan error aplikasi dulu.
