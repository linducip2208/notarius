-- BACKFILL_AUDIT_OFFICE.sql — backfill aman audit_logs.office_id (kolom baru 028), SATU ENTITAS SAJA.
--
-- TUJUAN
--   Kolom audit_logs.office_id (migrasi 028_audit_office.sql) NULL untuk baris lama.
--   File ini mengisi office_id HANYA dari users.office_id milik actor, dan HANYA
--   untuk baris entity='user' yang actor-nya valid. Aman + idempoten: hanya baris
--   office_id IS NULL yang disentuh; baris yang sudah terisi TIDAK diubah.
--
-- RUANG LINGKUP (SENGAJA SEMPIT — 1 entitas)
--   DIKERJAKAN : entity='user' via JOIN users ON users.id = audit_logs.actor_id.
--   TIDAK DIKERJAKAN + ALASAN:
--     - entity='client'   : butuh join berantai audit_logs.entity_id -> clients.id
--                           -> clients.office_id; entity_id VARCHAR(100) campur format,
--                           risiko salah petakan kantor. Perlu pemetaan + review
--                           per-entitas tersendiri. TIDAK disentuh di sini.
--     - entity='deed' (deed_drafts/deeds) : sama — join berantai ke tabel akta +
--                           varian nama entity ('deed' vs 'deed_draft'), butuh audit
--                           nilai entity aktual dulu. TIDAK disentuh di sini.
--     - entity='document'/lainnya (identity, file, dsb.) : join berantai berbeda
--                           tiap entitas (identities->clients->offices, files->...),
--                           belum dipetakan. TIDAK disentuh di sini.
--     - Baris actor NULL / actor tanpa office (login gagal pre-auth, cron/sistem,
--       user yang office-nya NULL) : kantor tak diketahui — SENGAJA dibiarkan NULL
--       sesuai kontrak 028 (NULL = tak diketahui).
--
-- STATUS PENGUJIAN
--   BELUM TERUJI LIVE — sesuai instruksi, UPDATE ini TIDAK dijalankan ke DB
--   produksi/dev dari sesi ini. File ini hanya direview manual 2x (baca ulang
--   penuh: cek nama kolom/tabel vs 001_core.sql + 028_audit_office.sql dan
--   literal entity='user' vs internal/app/*.go). Wajib dry-run di staging +
--   backup sebelum eksekusi nyata. Sintaks MySQL 8, pernyataan polos (tanpa
--   IF NOT EXISTS / prosedur), mengikuti gaya migrasi 016-028.
--
-- CARA PAKAI (DBA, di staging dulu)
--   1) Backup: mysqldump ... --where="office_id IS NULL" atau snapshot penuh.
--   2) Jalankan blok VERIFIKASI SEBELUM, catat angkanya.
--   3) Jalankan blok PRATINJAU, pastikan kandidat masuk akal.
--   4) Jalankan blok UPDATE di dalam transaksi, cek ROW_COUNT, lalu COMMIT
--      (atau ROLLBACK bila aneh).
--   5) Jalankan blok VERIFIKASI SESUDAH. Sisa NULL pada entity='user' seharusnya
--      tinggal baris tanpa actor valid / actor tanpa office (lihat cek residual).
--
-- Skema acuan:
--   audit_logs(id, actor_id NULL, action, entity, entity_id, detail, created_at,
--              prev_hash, chain_hash, office_id NULL) — 001_core + 020 + 028.
--   users(id, office_id NULL, ...) — 001_core.

-- ============================================================================
-- 0) VERIFIKASI SEBELUM (read-only; jalankan dulu, simpan hasilnya)
-- ============================================================================

-- 0a) NULL per entity — gambaran sebelum backfill.
SELECT
  entity,
  COUNT(*) AS total,
  SUM(office_id IS NULL) AS null_office,
  SUM(office_id IS NOT NULL) AS filled_office
FROM audit_logs
GROUP BY entity
ORDER BY entity;

-- 0b) Target backfill ini: kandidat yang AKAN diisi (harus >= 0, semua office valid).
SELECT COUNT(*) AS user_backfill_candidates
FROM audit_logs a
JOIN users u ON u.id = a.actor_id
WHERE a.office_id IS NULL
  AND a.entity = 'user'
  AND a.actor_id IS NOT NULL
  AND u.office_id IS NOT NULL;

-- ============================================================================
-- 1) PRATINJAU (read-only; contoh 20 baris kandidat)
-- ============================================================================
SELECT a.id, a.actor_id, a.action, a.entity, a.entity_id,
       a.office_id AS audit_office_now, u.office_id AS actor_office
FROM audit_logs a
JOIN users u ON u.id = a.actor_id
WHERE a.office_id IS NULL
  AND a.entity = 'user'
  AND a.actor_id IS NOT NULL
  AND u.office_id IS NOT NULL
ORDER BY a.id
LIMIT 20;

-- ============================================================================
-- 2) UPDATE (satu-satunya pernyataan tulis; idempoten — aman diulang)
-- ============================================================================
START TRANSACTION;

UPDATE audit_logs a
JOIN users u ON u.id = a.actor_id
SET a.office_id = u.office_id
WHERE a.office_id IS NULL
  AND a.entity = 'user'
  AND a.actor_id IS NOT NULL
  AND u.office_id IS NOT NULL;

-- Cek ROW_COUNT() di sini; bila 0 padahal 0b > 0, ROLLBACK dan investigasi.
SELECT ROW_COUNT() AS rows_backfilled;

COMMIT;

-- ============================================================================
-- 3) VERIFIKASI SESUDAH (read-only; bandingkan dengan 0a)
-- ============================================================================

-- 3a) NULL per entity — sesudah. Yang boleh berubah hanya baris entity='user'.
SELECT
  entity,
  COUNT(*) AS total,
  SUM(office_id IS NULL) AS null_office,
  SUM(office_id IS NOT NULL) AS filled_office
FROM audit_logs
GROUP BY entity
ORDER BY entity;

-- 3b) Residual: entity='user' yang MASIH NULL — harusnya hanya baris tanpa
--     actor valid / actor tanpa office. Jika query ini mengembalikan baris
--     dengan actor_office NOT NULL, berarti backfill gagal/parsial.
SELECT a.id, a.actor_id, a.action, a.entity_id, a.created_at,
       u.office_id AS actor_office
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_id
WHERE a.office_id IS NULL
  AND a.entity = 'user'
ORDER BY a.id
LIMIT 50;

-- 3c) Ringkasan residual: 0 harapan untuk kategori 'actor_valid_beroffice'.
SELECT
  SUM(a.actor_id IS NULL) AS still_null_no_actor,
  SUM(a.actor_id IS NOT NULL AND u.id IS NULL) AS still_null_actor_hilang,
  SUM(a.actor_id IS NOT NULL AND u.id IS NOT NULL AND u.office_id IS NULL) AS still_null_actor_tanpa_office,
  SUM(a.actor_id IS NOT NULL AND u.office_id IS NOT NULL) AS masih_bisa_dibackfill_harusnya_nol
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_id
WHERE a.office_id IS NULL
  AND a.entity = 'user';
