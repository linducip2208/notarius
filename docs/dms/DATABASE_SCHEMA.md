# DMS — Skema Database (MySQL)

Sumber kebenaran: `migrations/001_core.sql`, `migrations/004_ops.sql`.
Kolom di bawah disalin dari migrasi; bukan asumsi.

## 1. `files` (metadata berkas)

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | BIGINT UNSIGNED AI PK | identitas baris |
| `office_id` | BIGINT UNSIGNED | FK → `offices(id)`, `ON DELETE RESTRICT` |
| `owner_type` | VARCHAR(50) | polimorfik (mis. client), tanpa FK |
| `owner_id` | BIGINT UNSIGNED | id pemilik, tanpa FK |
| `ref` | VARCHAR(512) | referensi storage (`Store.Save/Open`), bukan URL publik |
| `original_name` | VARCHAR(255) | nama asli untuk tampilan |
| `mime` | VARCHAR(100) | hasil sniff server-side |
| `size_bytes` | BIGINT | ukuran aktual |
| `sha256` | CHAR(64) | hash integritas |
| `ocr_status` | ENUM `PENDING,DONE,FAILED,SKIPPED` | default `PENDING` |
| `created_by` | BIGINT UNSIGNED NULL | user pengunggah |
| `created_at` | TIMESTAMP | default now |

Index: `idx_files_owner (owner_type, owner_id)`.
Catatan: belum ada `UNIQUE(sha256, office_id)` — dedup di aplikasi bila perlu.

## 2. `ai_results` (hasil OCR/AI)

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | BIGINT UNSIGNED AI PK | — |
| `file_id` | BIGINT UNSIGNED NULL | FK → `files(id)` `ON DELETE SET NULL` |
| `provider` | VARCHAR(100) | mis. `local-tesseract`, `manual` |
| `provenance` | VARCHAR(255) | versi biner/model/konfigurasi |
| `doc_type` | VARCHAR(20) NULL | `KTP/KK/NPWP/OTHER` |
| `fields` | JSON NULL | hasil ekstraksi |
| `confidence` | DECIMAL(5,4) NULL | skor kepercayaan |
| `verified_by/at` | BIGINT/TIMESTAMP NULL | verifikasi manusia (wajib sebelum dipakai) |
| `created_at` | TIMESTAMP | — |

## 3. `retention_policies` (retensi arsip)

| Kolom | Tipe |
|---|---|
| `id` | BIGINT UNSIGNED AI PK |
| `entity` | VARCHAR(100) UNIQUE |
| `retain_years` | INT |
| `note` | VARCHAR(500) NULL |

Seed bawaan: `minuta=25`, `repertorium=25`, `legalisasi_register=10`, `keuangan=10`.

## 4. `audit_logs` (audit imutabel)

Didefinisikan di `001_core.sql`. Ditulis append-only oleh lapisan `internal/audit`.
Tidak ada UPDATE/DELETE dari aplikasi; retensi mengikuti kebijakan arsip.

## 5. Diagram relasi (teks)

`offices (1) ──< files (office_id, RESTRICT)`
`files (1) ──< ai_results (file_id, SET NULL)`
`users (1) ──< files.created_by (nullable, tanpa cascade tegas)`
`users (1) ──< ai_results.verified_by (nullable)`
