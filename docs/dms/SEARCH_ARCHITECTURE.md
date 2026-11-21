# DMS — Arsitektur Pencarian

> Jujur: tidak ada mesin pencari khusus (Elasticsearch/dsb) di repo.
> Bagian ini mendeskripsikan kemampuan nyata hari ini + rencana.

## 1. Kemampuan hari ini (MySQL saja)

- Filter metadata `files`: `(owner_type, owner_id)` via `idx_files_owner`,
  `original_name` (LIKE), `mime`, rentang `created_at`, `ocr_status`.
- Filter hasil OCR: `ai_results.doc_type`, `provider`, status verifikasi
  (`verified_by IS [NOT] NULL`); `fields` JSON di-query via fungsi JSON MySQL.
- Isolasi kantor: setiap query wajib menyertakan `office_id`.
- Daftar klien: `GET /clients`, export: `GET /clients/export`.

## 2. Pola query baku

- Listing per pemilik: `WHERE office_id=? AND owner_type=? AND owner_id=? ORDER BY id DESC LIMIT ?`.
- Belum ada indeks full-text pada `original_name` maupun kolom hasil OCR —
  pencarian teks bebas berskala kecil (LIKE) dan tidak dijamin cepat
  untuk volume besar.

## 3. Batasan eksplisit

- Tidak ada pencarian isi biner (PDF/gambar) selain lewat `ai_results.fields`.
- Tidak ada ranking relevansi, faceting, atau highlight.
- JSON `fields` fleksibel tetapi tidak tervalidasi skema di DB.

## 4. Rencana bertahap (belum diimplementasikan)

1. Tambah indeks: `(office_id, created_at)`, prefix index `original_name(64)`.
2. Normalisasi kolom cari-umum (mis. `doc_type` terverifikasi) keluar dari JSON.
3. Bila volume membenarkan: FTS MySQL (`FULLTEXT`) atau indexer eksternal —
   putuskan via ADR; jangan klaim sebelum ada.
4. Audit tiap query export (`reports.view`) karena rawan exfiltrasi.
