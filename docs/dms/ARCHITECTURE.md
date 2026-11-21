# DMS — Arsitektur (NotaryOS)

> Bahasa Indonesia, ringkas-teknis. Hanya mendeskripsikan yang terverifikasi di kode.

## 1. Gambaran umum

DMS (Document Management System) adalah subsistem pengelolaan berkas kantor notaris:
Go (HTTP + template server-side) + MySQL untuk metadata, file fisik di
private local storage. Tidak ada object storage eksternal yang aktif.

## 2. Komponen

| Lapisan | Implementasi | Lokasi |
|---|---|---|
| HTTP routing | `chi` router | `internal/app/app.go` |
| Upload/unduh/OCR | handler `fileUpload`, `fileDownload`, `fileOCR` | `internal/app/` + `internal/files/` |
| Abstraksi storage | interface `storage.Store`, impl `LocalStore` | `internal/storage/storage.go` |
| OCR | interface `ocr.Provider`, `Tesseract`, `NoopProvider` | `internal/ocr/` |
| Metadata | tabel `files`, `ai_results` (MySQL) | `migrations/004_ops.sql` |
| Audit | tabel `audit_logs` (imutable, append-only) | `migrations/001_core.sql` |
| UI | Tabler 100% + Go templates (server-side, tanpa SPA) | `web/` |

## 3. Aliran data

1. Upload (`POST /clients/{id}/files`) → validasi → `Store.Save(ref, ...)` → row `files`.
2. Unduh (`GET /files/{id}`) → otorisasi → `Store.Open(ref)` → stream ke klien.
3. OCR (`POST /files/{id}/ocr`) → `Provider.Extract` → row `ai_results`, `files.ocr_status` diperbarui.
4. Verifikasi manusia → `ai_results.verified_by/at` diisi (wajib sebelum dipakai ke identitas/akta).

## 4. Keputusan kunci

- **Metadata vs biner dipisah**: MySQL tidak menyimpan blob; hanya `ref`, `sha256`, `mime`, `size_bytes`.
- **Storage terabstraksi**: bisnis hanya memanggil `Save`/`Open`; pindah ke S3 cukup ganti implementasi `Store`.
- **OCR lokal-first**: Tesseract `ind`; AI eksternal opsional dan tidak boleh menerima data sensitif tanpa otorisasi.
- **Audit imutabel**: setiap aksi DMS dicatat di `audit_logs`; tidak ada endpoint hapus/ubah audit.

## 5. Batasan yang diketahui

- `files.ref` belum punya constraint `UNIQUE(sha256, office)` (temuan `docs/audit/`).
- Relasi `owner_type/owner_id` polimorfik tanpa FK penuh — integritas dijaga di lapisan aplikasi.
- Klasifikasi `ocr.Classify` hanya heuristik nama file; staf wajib konfirmasi.
