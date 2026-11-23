-- 031_deed_handover.sql — riwayat serah dokumen akta ke klien (tanda terima serah dokumen).
-- Akta (deed_drafts) sebelumnya TIDAK punya jejak serah ke klien (wasiat punya
-- willHandover, legalisasi punya HANDED_OVER). Satu akta dapat diserahkan
-- berulang (minuta/salinan/grosse susulan) sehingga dipakai tabel riwayat,
-- bukan kolom tunggal.
-- Idempoten: CREATE TABLE IF NOT EXISTS (tabel ada = pernyataan dilewati) +
-- runner menoleransi DDL duplikat (1060/1061/1050/1068/1091) untuk rerun parsial.
CREATE TABLE IF NOT EXISTS deed_handovers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  deed_id BIGINT UNSIGNED NOT NULL,
  handed_to VARCHAR(255) NOT NULL,
  handed_at TIMESTAMP NULL,
  note TEXT NULL,
  created_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE RESTRICT,
  FOREIGN KEY (deed_id) REFERENCES deed_drafts(id) ON DELETE CASCADE,
  INDEX idx_deed_handover_office_deed (office_id, deed_id)
);
