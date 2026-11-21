# DMS — Spesifikasi Workflow Dokumen

## 1. Status berkas (`files.ocr_status`)

`PENDING → DONE | FAILED | SKIPPED`. Kolom ENUM dengan default `PENDING`
(`migrations/004_ops.sql`). Tidak ada status lain di DB.

## 2. Alur utama

1. **Upload** — staf (`doc_operator`/`staff`) unggah via `POST /clients/{id}/files`.
   Sistem: cek kantor, sniff MIME, hitung `sha256`, `Store.Save`, insert `files`
   (`ocr_status=PENDING`), tulis audit.
2. **Klasifikasi awal** — `ocr.Classify(nama)` memberi saran `KTP/KK/NPWP/OTHER`.
   Sifatnya advis; staf dapat mengoreksi saat verifikasi.
3. **OCR** — `POST /files/{id}/ocr` menjalankan `Provider.Extract`.
   Sukses → insert `ai_results` + `ocr_status=DONE`.
   Engine tak tersedia/gagal → `FAILED` (atau `SKIPPED` bila staf melewati manual).
4. **Verifikasi manusia (wajib)** — pemegang `identity.verify` memeriksa `fields`
   lawan dokumen asli, lalu mengisi `verified_by/at`. Tanpa ini hasil OCR
   tidak boleh mengalir ke `identities` atau draf akta.
5. **Penautan** — berkas terhubung via `(owner_type, owner_id)` ke klien/layanan;
   unduhan selalu otorisasi per-file.
6. **Retensi/arsip** — ikuti `retention_policies`; lihat ARCHIVE_MANAGEMENT.md.

## 3. Aturan transisi

- `PENDING → DONE`: hanya bila `ai_results` tertulis untuk `file_id` tersebut.
- `* → FAILED`: OCR error (engine hilang, file rusak/tak terbaca).
- `PENDING → SKIPPED`: keputusan staf eksplisit (mis. file non-teks); wajib audit.
- `DONE → DONE` (OCR ulang): diizinkan; hasil baru = row `ai_results` baru
  (riwayat tidak ditimpa), `verified_*` hasil lama tidak terbawa.

## 4. Kegagalan & fallback

- `NoopProvider` mengembalikan error deskriptif → UI menampilkan fallback
  entri manual, bukan berpura-pura OCR berhasil.
- Upload > 100 MB ditolak di `LocalStore.Save`.
- File korup: `FAILED` + pesan; berkas tetap tersimpan untuk investigasi.
