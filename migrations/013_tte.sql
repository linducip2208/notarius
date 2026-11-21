-- 013_tte.sql — tanda tangan elektronik internal (hash tersertifikasi sistem, BUKAN PSrE).
-- Jujur: hash SHA-256 atas isi+naskah+penandatangan+waktu, terverifikasi via /verify?code=DEED.
CREATE TABLE IF NOT EXISTS deed_signatures (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT NOT NULL,
  deed_id BIGINT NOT NULL,
  signer_user_id BIGINT NULL,
  signer_name VARCHAR(190) NOT NULL DEFAULT '',
  content_hash CHAR(64) NOT NULL DEFAULT '',
  seal_code VARCHAR(120) NOT NULL DEFAULT '',
  note VARCHAR(255) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_sig_deed (deed_id),
  INDEX idx_sig_office (office_id)
);
