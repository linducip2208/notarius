-- 030_quotations.sql — penawaran resmi terikat permohonan (tanpa ubah invoice existing).
-- Quotations = dokumen penawaran (bukan janji tarif final); konversi ke invoice
-- memakai jalur anti-overpay yang sama (paid=0, status ISSUED, RecordPayment utuh).
CREATE TABLE IF NOT EXISTS quotations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  request_id BIGINT UNSIGNED NULL,
  number VARCHAR(50) NOT NULL,
  valid_until DATE NULL,
  status ENUM('DRAFT','SENT','ACCEPTED','REJECTED','EXPIRED') NOT NULL DEFAULT 'DRAFT',
  notes TEXT NULL,
  created_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE RESTRICT,
  FOREIGN KEY (request_id) REFERENCES service_requests(id) ON DELETE SET NULL,
  UNIQUE KEY uq_quot_office_number (office_id, number),
  INDEX idx_quot_status (status),
  INDEX idx_quot_request (request_id)
);
CREATE TABLE IF NOT EXISTS quotation_lines (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  quotation_id BIGINT UNSIGNED NOT NULL,
  description VARCHAR(255) NOT NULL,
  qty INT NOT NULL DEFAULT 1,
  unit_price DECIMAL(18,2) NOT NULL DEFAULT 0,
  line_total DECIMAL(18,2) NOT NULL DEFAULT 0,
  FOREIGN KEY (quotation_id) REFERENCES quotations(id) ON DELETE CASCADE
);
-- Tautan balik invoice -> penawaran (NULLABLE agar invoice manual existing tak tersentuh).
ALTER TABLE invoices ADD COLUMN quotation_id BIGINT UNSIGNED NULL;
ALTER TABLE invoices ADD INDEX idx_inv_quotation (quotation_id);
ALTER TABLE invoices ADD CONSTRAINT fk_inv_quotation FOREIGN KEY (quotation_id) REFERENCES quotations(id) ON DELETE SET NULL;
