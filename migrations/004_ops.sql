-- 004_ops.sql — finance, government assisted workflows, notifications, files, retention
CREATE TABLE IF NOT EXISTS price_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  service_id BIGINT UNSIGNED NULL,
  name VARCHAR(255) NOT NULL,
  amount DECIMAL(18,2) NOT NULL DEFAULT 0,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS invoices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  client_id BIGINT UNSIGNED NOT NULL,
  request_id BIGINT UNSIGNED NULL,
  number VARCHAR(50) NOT NULL,
  subtotal DECIMAL(18,2) NOT NULL DEFAULT 0,
  tax DECIMAL(18,2) NOT NULL DEFAULT 0,
  total DECIMAL(18,2) NOT NULL DEFAULT 0,
  paid DECIMAL(18,2) NOT NULL DEFAULT 0,
  status ENUM('DRAFT','ISSUED','PARTIAL','PAID','VOID') NOT NULL DEFAULT 'DRAFT',
  due_date DATE NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE RESTRICT,
  FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE RESTRICT,
  UNIQUE KEY uq_invoice_no (office_id, number),
  INDEX idx_inv_status (status)
);

CREATE TABLE IF NOT EXISTS invoice_lines (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  invoice_id BIGINT UNSIGNED NOT NULL,
  description VARCHAR(255) NOT NULL,
  qty INT NOT NULL DEFAULT 1,
  unit_price DECIMAL(18,2) NOT NULL DEFAULT 0,
  line_total DECIMAL(18,2) NOT NULL DEFAULT 0,
  FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS payments (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  invoice_id BIGINT UNSIGNED NOT NULL,
  amount DECIMAL(18,2) NOT NULL,
  method ENUM('CASH','TRANSFER','QRIS','OTHER') NOT NULL DEFAULT 'TRANSFER',
  reference VARCHAR(100) NULL,
  received_by BIGINT UNSIGNED NULL,
  paid_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS expenses (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  category VARCHAR(100) NOT NULL,
  description VARCHAR(255) NOT NULL,
  amount DECIMAL(18,2) NOT NULL,
  spent_at DATE NOT NULL,
  created_by BIGINT UNSIGNED NULL,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE RESTRICT,
  INDEX idx_exp_date (spent_at)
);

CREATE TABLE IF NOT EXISTS gov_submissions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  channel ENUM('AHU','OSS') NOT NULL,
  reference VARCHAR(255) NOT NULL,
  status ENUM('DRAFT','READY','SUBMITTED','ACCEPTED','REJECTED') NOT NULL DEFAULT 'DRAFT',
  checklist JSON NULL,
  receipt_no VARCHAR(100) NULL,
  receipt_ref VARCHAR(255) NULL,
  notes TEXT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE RESTRICT,
  INDEX idx_gov_status (status)
);

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(255) NOT NULL,
  body TEXT NULL,
  is_read TINYINT(1) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_notif_user (user_id, is_read)
);

CREATE TABLE IF NOT EXISTS files (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  owner_type VARCHAR(50) NOT NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  ref VARCHAR(512) NOT NULL,
  original_name VARCHAR(255) NOT NULL,
  mime VARCHAR(100) NOT NULL,
  size_bytes BIGINT NOT NULL,
  sha256 CHAR(64) NOT NULL,
  ocr_status ENUM('PENDING','DONE','FAILED','SKIPPED') NOT NULL DEFAULT 'PENDING',
  created_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE RESTRICT,
  INDEX idx_files_owner (owner_type, owner_id)
);

CREATE TABLE IF NOT EXISTS ai_results (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  file_id BIGINT UNSIGNED NULL,
  provider VARCHAR(100) NOT NULL,
  provenance VARCHAR(255) NOT NULL,
  doc_type VARCHAR(20) NULL,
  fields JSON NULL,
  confidence DECIMAL(5,4) NULL,
  verified_by BIGINT UNSIGNED NULL,
  verified_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS retention_policies (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  entity VARCHAR(100) NOT NULL UNIQUE,
  retain_years INT NOT NULL,
  note VARCHAR(500) NULL
);

INSERT IGNORE INTO retention_policies(entity, retain_years, note) VALUES
  ('minuta', 25, 'UUJN: protokol disimpan 25 tahun oleh notaris bersangkutan'),
  ('repertorium', 25, 'Bagian protokol notaris'),
  ('legalisasi_register', 10, 'Arsip administrasi kantor'),
  ('keuangan', 10, 'Arsip perpajakan & audit');

-- Seed roles & permissions (reference data only; no fake business records)
INSERT IGNORE INTO roles(name, description) VALUES
  ('super_admin','Super Administrator'),
  ('office_admin','Office Administrator'),
  ('notary','Notaris'),
  ('substitute_notary','Notaris Pengganti'),
  ('staff','Notary Staff'),
  ('finance','Finance Staff'),
  ('doc_operator','Document Operator'),
  ('client','Client'),
  ('auditor','Auditor / Read-only');

INSERT IGNORE INTO permissions(name) VALUES
  ('client.view'),('client.create'),('client.edit'),('identity.view'),('identity.verify'),
  ('deed.draft'),('deed.edit'),('deed.approve'),('protocol.view'),('repertorium.manage'),
  ('finance.view'),('finance.manage'),('template.manage'),('users.manage'),
  ('reports.view'),('office.manage'),('service.request'),('legalization.process'),('government.submit');
