-- 020_compliance_hardening.sql — formalitas akta, mandat pengganti, rantai audit, izin.
-- Gaya 016-019: CREATE TABLE IF NOT EXISTS + ALTER polos.

-- Formalitas akta: saksi, pembacaan, renvoi (W-04). Wajib diisi saat pengesahan.
ALTER TABLE deed_drafts
  ADD COLUMN witness1_name VARCHAR(160) NULL,
  ADD COLUMN witness1_id VARCHAR(32) NULL,
  ADD COLUMN witness2_name VARCHAR(160) NULL,
  ADD COLUMN witness2_id VARCHAR(32) NULL,
  ADD COLUMN read_at TIMESTAMP NULL,
  ADD COLUMN read_place VARCHAR(200) NULL,
  ADD COLUMN renvoi_note TEXT NULL;

-- Salinan: taut ke minuta + penomoran per akta (W-05).
ALTER TABLE legalizations
  ADD COLUMN deed_id BIGINT UNSIGNED NULL,
  ADD COLUMN copy_no INT NULL;
ALTER TABLE legalizations
  MODIFY COLUMN kind ENUM('LEGALISASI','WAARMERKING','SALINAN','KUTIPAN','GROSSE') NOT NULL;

-- Protokol: lokasi + status + penanggung jawab (W-09).
ALTER TABLE protocols
  ADD COLUMN location_id BIGINT UNSIGNED NULL,
  ADD COLUMN status ENUM('IN_STORAGE','OUT','RETURNED') NOT NULL DEFAULT 'IN_STORAGE',
  ADD COLUMN holder VARCHAR(160) NULL;

-- Mandat notaris pengganti: SK, periode, ruang lingkup (W-12).
CREATE TABLE IF NOT EXISTS substitute_mandates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  sk_no VARCHAR(120) NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  scope VARCHAR(255) NOT NULL DEFAULT 'penuh',
  created_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_submandate (office_id, user_id)
);

-- Idempotensi pembayaran (F-08).
ALTER TABLE payments
  ADD COLUMN idem_key VARCHAR(64) NULL,
  ADD UNIQUE KEY uq_payment_idem (idem_key);

-- Catatan EDD untuk BO PEP (F-06).
ALTER TABLE beneficial_owners
  ADD COLUMN edd_note VARCHAR(500) NULL;

-- Rantai audit anti-tamper: hash berantai (SEC-25).
ALTER TABLE audit_logs
  ADD COLUMN prev_hash VARCHAR(64) NULL,
  ADD COLUMN chain_hash VARCHAR(64) NULL;

-- Insiden keamanan (P-03).
CREATE TABLE IF NOT EXISTS security_incidents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  kind VARCHAR(60) NOT NULL,
  title VARCHAR(255) NOT NULL,
  detail TEXT NOT NULL,
  status ENUM('OPEN','HANDLING','CLOSED') NOT NULL DEFAULT 'OPEN',
  reported_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_secinc (office_id, status)
);

-- Legal Compliance Center (modul 15): register, persyaratan, checklist, bukti, temuan, review, perubahan.
CREATE TABLE IF NOT EXISTS lc_regulations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  number VARCHAR(120) NOT NULL,
  year INT NOT NULL,
  issuer VARCHAR(160) NOT NULL DEFAULT '',
  status VARCHAR(60) NOT NULL DEFAULT 'berlaku',
  source_url VARCHAR(500) NOT NULL DEFAULT '',
  verified_at DATE NULL,
  note TEXT NOT NULL,
  UNIQUE KEY uq_lc_reg (office_id, number, year)
);

CREATE TABLE IF NOT EXISTS lc_requirements (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  regulation_id BIGINT UNSIGNED NULL,
  code VARCHAR(20) NOT NULL DEFAULT '',
  title VARCHAR(255) NOT NULL,
  obligation TEXT NOT NULL,
  module VARCHAR(80) NOT NULL DEFAULT '',
  risk ENUM('CRITICAL','HIGH','MEDIUM','LOW') NOT NULL DEFAULT 'MEDIUM',
  status ENUM('COMPLIANT_TECHNICAL_EVIDENCE','PARTIALLY_IMPLEMENTED','NON_COMPLIANT','NOT_APPLICABLE_WITH_REASON','HUMAN_REVIEW_REQUIRED','SOURCE_DOCUMENT_NOT_VERIFIED','NOT_TESTED') NOT NULL DEFAULT 'NOT_TESTED',
  evidence TEXT NOT NULL,
  owner_role VARCHAR(60) NOT NULL DEFAULT 'office_admin',
  UNIQUE KEY uq_lc_req (office_id, code)
);

CREATE TABLE IF NOT EXISTS lc_checks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  requirement_id BIGINT UNSIGNED NULL,
  title VARCHAR(255) NOT NULL,
  checked TINYINT(1) NOT NULL DEFAULT 0,
  checker VARCHAR(160) NOT NULL DEFAULT '',
  checked_at TIMESTAMP NULL,
  note VARCHAR(500) NOT NULL DEFAULT '',
  file_id BIGINT UNSIGNED NULL
);

CREATE TABLE IF NOT EXISTS lc_findings (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(255) NOT NULL,
  risk ENUM('CRITICAL','HIGH','MEDIUM','LOW') NOT NULL DEFAULT 'MEDIUM',
  cause TEXT NOT NULL,
  remediation TEXT NOT NULL,
  owner_role VARCHAR(60) NOT NULL DEFAULT 'office_admin',
  due_date DATE NULL,
  status ENUM('OPEN','IN_PROGRESS','RESOLVED','ACCEPTED') NOT NULL DEFAULT 'OPEN',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS lc_reviews (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  reviewer VARCHAR(160) NOT NULL,
  role VARCHAR(80) NOT NULL DEFAULT 'notaris',
  scope VARCHAR(255) NOT NULL DEFAULT '',
  decision ENUM('SETUJU','SETUJU_BERSYARAT','TOLAK') NOT NULL DEFAULT 'SETUJU',
  note TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS lc_changes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  regulation_id BIGINT UNSIGNED NULL,
  kind ENUM('BARU','DIUBAH','DICABUT') NOT NULL,
  summary VARCHAR(500) NOT NULL,
  impact VARCHAR(500) NOT NULL DEFAULT '',
  followup_task_id BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Cermin izin baru untuk UI /roles (penegakan tetap di kode rbac.go).
INSERT IGNORE INTO permissions(name) VALUES
  ('audit.view'),('lcc.view'),('lcc.manage');
INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON (
  (r.name IN ('super_admin'))
  OR (r.name='office_admin' AND p.name IN ('audit.view','lcc.view','lcc.manage'))
  OR (r.name IN ('notary','substitute_notary') AND p.name IN ('audit.view','lcc.view','lcc.manage'))
  OR (r.name='auditor' AND p.name IN ('audit.view','lcc.view'))
);
