# BLOCKERS
| ID | Item | Dampak | Jalan keluar |
|----|------|--------|--------------|
| B-01 | Tesseract OCR belum terinstal di host | F-022 runtime perlu binary | SELESAI (kode): adapter `ocr.TesseractProvider` + fallback Noop jujur; instal Tesseract, set OCR_BINARY |
| B-02 | Kredensial/AI key eksternal tidak ada | AI eksternal nonaktif (by design) | Adapter hanya bila kunci disediakan via env + persetujuan; tanpa kunci = fitur AI mati jujur |
| B-03 | Tidak ada API resmi AHU/OSS yang diotorisasi | F-023 asistensi manual saja | SELESAI (kode): checklist + bukti manual; jangan hardcode endpoint |
| B-04 | AV host memblokir nama `validate.test.exe` | `go test` paket validate gagal default | Terbukti lolos via biner rename; catat; jalankan CI di Linux |
| B-05 | Rate-limit + CSRF + MFA + enkripsi NIK | SELESAI 0.3 (rate-limit teruji, CSRF aktif, MFA TOTP, NIK AES-GCM, +HSTS) | Sisa: pen-test eksternal + TLS di reverse-proxy |
| B-06 | Tinjauan notaris atas alur hukum belum ada | Belum READY FOR PRODUCTION | Wajib sebelum produksi; kode siap diuji notaris |
| B-07 | Sesi in-memory: restart menghapus login | SELESAI: SQLStore + cleanup/jam (migrasi 007) | Multi-instance siap; pantau tabel sessions |
