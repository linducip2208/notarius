-- 019_ops_growth.sql — semua-ide-operasional: spesimen, bank, anggaran, tugas berulang,
-- review wasiat, password/device, SLA per layanan, gov follow-up.
-- Gaya 016-018: CREATE TABLE IF NOT EXISTS + ALTER polos.

-- Spesimen tanda tangan / cap per klien (pembanding visual).
CREATE TABLE IF NOT EXISTS client_specimens (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  client_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('TTD','CAP','STEMPEL','LAIN') NOT NULL DEFAULT 'TTD',
  file_id BIGINT UNSIGNED NOT NULL,
  note VARCHAR(255) NOT NULL DEFAULT '',
  created_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_spec_client (client_id)
);

-- Mutasi bank untuk rekonsiliasi jujur (cocok/selisih eksplisit).
CREATE TABLE IF NOT EXISTS bank_mutations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  stmt_date DATE NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  amount DECIMAL(18,2) NOT NULL DEFAULT 0,
  matched_invoice_id BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_bank_office (office_id, stmt_date)
);

-- Anggaran per kategori per bulan.
CREATE TABLE IF NOT EXISTS budgets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  category VARCHAR(120) NOT NULL,
  month DATE NOT NULL,
  amount DECIMAL(18,2) NOT NULL DEFAULT 0,
  UNIQUE KEY uq_budget (office_id, category, month)
);

-- Tugas: pengulangan + tautan konteks.
ALTER TABLE tasks
  ADD COLUMN recurrence ENUM('NONE','DAILY','WEEKLY','MONTHLY') NOT NULL DEFAULT 'NONE',
  ADD COLUMN ref_type ENUM('','deed','request','client','file') NOT NULL DEFAULT '',
  ADD COLUMN ref_id BIGINT UNSIGNED NULL;

-- Wasiat: review tahunan + pembukaan bersaksi ganda.
ALTER TABLE wills
  ADD COLUMN review_date DATE NULL,
  ADD COLUMN opened_by BIGINT UNSIGNED NULL,
  ADD COLUMN opened_witness VARCHAR(160) NULL,
  ADD COLUMN opened_at TIMESTAMP NULL,
  ADD COLUMN open_note VARCHAR(500) NULL;

-- Kata sandi: jejak pergantian untuk kebijakan berkala.
ALTER TABLE users
  ADD COLUMN password_changed_at TIMESTAMP NULL;

-- Sesi: catat IP + UA untuk notifikasi perangkat baru.
ALTER TABLE sessions
  ADD COLUMN ip VARCHAR(64) NULL,
  ADD COLUMN user_agent VARCHAR(255) NULL;

-- SLA per layanan (hari; default 3 = perilaku lama).
ALTER TABLE services
  ADD COLUMN sla_days INT NOT NULL DEFAULT 3;

-- Gov: tanggal tindak lanjut resi manual.
ALTER TABLE gov_submissions
  ADD COLUMN follow_up DATE NULL;
