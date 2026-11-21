-- 017_dms_enterprise.sql — custom fields, approval matrix, kuota, lisensi.
-- Idempoten: semua CREATE IF NOT EXISTS; ALTER via pola aman.

CREATE TABLE IF NOT EXISTS doc_custom_fields (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(120) NOT NULL,
  ftype ENUM('text','number','date','select','bool') NOT NULL DEFAULT 'text',
  options VARCHAR(500) NOT NULL DEFAULT '',
  required TINYINT(1) NOT NULL DEFAULT 0,
  position INT NOT NULL DEFAULT 0,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  UNIQUE KEY uq_dcfield (office_id, name)
);

CREATE TABLE IF NOT EXISTS doc_field_values (
  file_id BIGINT UNSIGNED NOT NULL,
  field_id BIGINT UNSIGNED NOT NULL,
  value VARCHAR(500) NOT NULL DEFAULT '',
  PRIMARY KEY (file_id, field_id),
  FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
  FOREIGN KEY (field_id) REFERENCES doc_custom_fields(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS doc_approval_chains (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  step_order INT NOT NULL DEFAULT 0,
  role_required VARCHAR(40) NOT NULL,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  UNIQUE KEY uq_dchain (office_id, step_order)
);

CREATE TABLE IF NOT EXISTS doc_approvals (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  file_id BIGINT UNSIGNED NOT NULL,
  step_order INT NOT NULL DEFAULT 0,
  role_required VARCHAR(40) NOT NULL DEFAULT '',
  approver_id BIGINT UNSIGNED NULL,
  status ENUM('PENDING','APPROVED','REJECTED') NOT NULL DEFAULT 'PENDING',
  note VARCHAR(500) NULL,
  decided_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE RESTRICT,
  FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
  INDEX idx_dappr (file_id, step_order)
);

SET @have_quota = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='offices' AND COLUMN_NAME='storage_quota_bytes');
SET @add_quota = IF(@have_quota=0,'ALTER TABLE offices ADD COLUMN storage_quota_bytes BIGINT UNSIGNED NOT NULL DEFAULT 10737418240','SELECT 1');
PREPARE stmt_quota FROM @add_quota;
EXECUTE stmt_quota;
DEALLOCATE PREPARE stmt_quota;
SET @have_lic = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='offices' AND COLUMN_NAME='license_key');
SET @add_lic = IF(@have_lic=0,'ALTER TABLE offices ADD COLUMN license_key VARCHAR(255) NOT NULL DEFAULT ''''','SELECT 1');
PREPARE stmt_lic FROM @add_lic;
EXECUTE stmt_lic;
DEALLOCATE PREPARE stmt_lic;
SET @have_licexp = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='offices' AND COLUMN_NAME='license_expires');
SET @add_licexp = IF(@have_licexp=0,'ALTER TABLE offices ADD COLUMN license_expires DATE NULL','SELECT 1');
PREPARE stmt_licexp FROM @add_licexp;
EXECUTE stmt_licexp;
DEALLOCATE PREPARE stmt_licexp;
