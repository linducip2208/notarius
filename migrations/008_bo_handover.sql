-- 008_bo_handover.sql — beneficial owners + will handover (F-009, F-014 serah-terima)
CREATE TABLE IF NOT EXISTS beneficial_owners (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  company_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  nik_last4 CHAR(4) NULL,
  ownership_pct DECIMAL(5,2) NOT NULL DEFAULT 0,
  is_pep TINYINT(1) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
  INDEX idx_bo_company (company_id)
);

ALTER TABLE wills
  ADD COLUMN handed_over TINYINT(1) NOT NULL DEFAULT 0,
  ADD COLUMN handed_to VARCHAR(255) NULL,
  ADD COLUMN handed_at TIMESTAMP NULL,
  ADD COLUMN handover_note VARCHAR(500) NULL;
