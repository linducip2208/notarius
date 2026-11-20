# TECHNICAL_DEBT — register berjalan
| ID | Tanggal | Item | Dampak | Rencana |
|----|---------|------|--------|---------|
| TD-01 | 2026-11-20 | Session store in-memory (hilang saat restart) | Single-instance OK; multi-instance butuh store DB/Redis | Ganti `internal/auth.Store` dengan implementasi SQL tanpa ubah handler |
| TD-02 | 2026-11-20 | NIK cipher belum dienkripsi field-level (kolom `nik_cipher` siap, enkripsi AES-GCM belum di-wire) | Kepatuhan PDP | Wire AES-GCM sebelum seed data produksi |
| TD-03 | 2026-11-20 | MFA kolom tersedia (`mfa_secret`) tapi alur TOTP belum diimplementasi | Keamanan | Sprint 3 |
| TD-04 | 2026-11-20 | OCR NoopProvider (Tesseract belum terinstal di host ini) | Fitur OCR nonaktif jujur | Instal Tesseract + adapter, lihat BLOCKERS.md |
| TD-05 | 2026-11-20 | PDF/DOCX export belum ada (render template teks saja) | Output dokumen | Tambah generator PDF di `docengine` sprint 3 |
