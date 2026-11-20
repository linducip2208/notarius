# BLOCKERS
| ID | Item | Dampak | Jalan keluar |
|----|------|--------|--------------|
| B-01 | Tesseract OCR belum terinstal di host | F-022 BLOCKED | Instal Tesseract, set OCR_BINARY, implementasi adapter `ocr.Provider`; manual-entry tetap tersedia |
| B-02 | Kredensial/AI key eksternal tidak ada | AI eksternal nonaktif (by design) | Adapter hanya bila kunci disediakan via env + persetujuan; tanpa kunci = fitur AI mati jujur |
| B-03 | Tidak ada API resmi AHU/OSS yang diotorisasi | F-023 asistensi manual saja | Jangan hardcode endpoint; dokumentasikan kredensial resmi bila kantor memilikinya |
| B-04 | AV host memblokir nama `validate.test.exe` | `go test` paket validate gagal default | Terbukti lolos via biner rename; catat; jalankan CI di Linux |
| B-05 | Rate-limit + CSRF + MFA + enkripsi NIK | SEBAGIAN SELESAI 0.2 (rate-limit teruji, CSRF aktif, MFA TOTP, NIK AES-GCM) — sisa: HSTS/TLS + pen-test |
| B-06 | Tinjauan notaris atas alur hukum belum ada | Belum READY FOR PRODUCTION | Wajib sebelum produksi (§W) |
| B-07 | Sesi in-memory: restart menghapus login | Single-instance; teramati saat restart smoke test | Ganti session store SQL di Sprint 3 |
