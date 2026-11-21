# CMS Installation — instalasi mandiri + upgrade

## Prasyarat
Go 1.26+, MySQL 8.x, Tesseract (opsional, untuk OCR), reverse proxy + TLS (produksi).

## Fresh install
1. Salin `.env.example` → `.env`, isi `DB_*`, `NOTARYOS_FIELD_KEY` (base64 32 byte),
   `SESSION_NAME`, opsional `NOTARYOS_SMTP_*`, `NOTARYOS_NOTIFY_WEBHOOK`, `NOTARYOS_S3_*`, `CLAMAV_ADDR`.
2. `make migrate` (001→015 otomatis; idempoten via `schema_migrations`).
3. `go run ./scripts/seed` (admin awal) lalu login.
4. Buka `/cms/setup`: isi identitas + kontak + tema + domain, lalu "Buat konten contoh"
   (tiga halaman DRAF berlabel [CONTOH]; terbitkan manual setelah disunting).

## Upgrade dari schema lama
`make migrate` menerapkan hanya migrasi yang belum tercatat (015 menambah
`whatsapp`/`website` secara idempoten + 11 tabel CMS). Backup dulu:
`deploy/backup/backup.sh` (butuh `GPG_RECIPIENT` + kredensial DB).

## Backup mencakup
Database (termasuk `office_branding`, `website_*`, `website_media` rows) +
direktori storage (berkas DMS + `cms/…` media + thumb). Restore: buat DB kosong,
impor dump, `GRANT` ke user app, pastikan storage dir ikut disalin lalu verifikasi
hitungan tabel (prosedur teruji lokal: `docs/qa/RESTORE_TEST.md`, 0 selisih).

## Uji instalasi bersih (terisolasi)
Gunakan DB kosong + storage dir sementara; migrasi harus `count>0` sekali lalu `0`
saat diulang; halaman publik tanpa konten terbit → landing lama (bukan error).
