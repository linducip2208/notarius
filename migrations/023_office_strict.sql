-- 023_office_strict.sql — flag mode ketat checklist + slot SLA per layanan (tanpa seed).
-- Gaya 016-022: CREATE TABLE IF NOT EXISTS, tanpa ubah data pengguna.
CREATE TABLE IF NOT EXISTS office_flags (
  office_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  strict_checklist TINYINT(1) NOT NULL DEFAULT 0,
  sla_days_json TEXT NULL,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE
);
