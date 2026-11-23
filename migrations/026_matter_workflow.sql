-- 026_matter_workflow.sql — fondasi Matter & workflow per-layanan (aditif, tanpa ubah data lama).
-- Gaya 016-025: CREATE TABLE IF NOT EXISTS + INSERT IGNORE + ALTER aman.

-- 1) Multi-pihak per matter: satu pekerjaan boleh melibatkan banyak klien
-- dengan peran berbeda (penghadap, penjual, pembeli, kreditur, debitur,
-- pemberi/penerima kuasa, saksi, dst). Tabel clients tetap master identitas.
CREATE TABLE IF NOT EXISTS matter_parties (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  request_id BIGINT UNSIGNED NOT NULL,
  client_id BIGINT UNSIGNED NOT NULL,
  role VARCHAR(50) NOT NULL DEFAULT 'PIHAK',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (request_id) REFERENCES service_requests(id) ON DELETE CASCADE,
  FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
  UNIQUE KEY uq_matter_party (request_id, client_id, role)
);

-- 2) Aturan workflow per layanan (menggantikan if ad-hoc per handler).
-- require_checklist: semua item harus VERIFIED/RECEIVED sebelum approve.
-- require_bpn_receipt: wajib resi BPN asli (PPAT).
-- min_parties: jumlah pihak minimum sebelum bisa maju ke IN_REVIEW.
CREATE TABLE IF NOT EXISTS service_workflow_rules (
  service_code VARCHAR(50) NOT NULL PRIMARY KEY,
  require_checklist TINYINT(1) NOT NULL DEFAULT 1,
  require_verified_identity TINYINT(1) NOT NULL DEFAULT 1,
  require_deed_kind VARCHAR(20) NULL,
  require_bpn_receipt TINYINT(1) NOT NULL DEFAULT 0,
  min_parties INT NOT NULL DEFAULT 1
);

-- Seed aturan untuk layanan berisiko tinggi (lainnya ikut default aman).
INSERT IGNORE INTO service_workflow_rules(service_code, require_checklist, require_verified_identity, require_deed_kind, require_bpn_receipt, min_parties) VALUES
('PPAT-AJB', 1, 1, 'AJB', 1, 2),
('PPAT-HIBAH', 1, 1, 'Hibah', 1, 2),
('PPAT-TUKAR', 1, 1, 'TukarMenukar', 1, 2),
('PPAT-APHB', 1, 1, 'APHB', 1, 1),
('PPAT-INBRENG', 1, 1, 'Inbreng', 0, 1),
('PPAT-SKMHT', 1, 1, 'SKMHT', 0, 2),
('PPAT-APHT', 1, 1, 'APHT', 0, 2),
('BUNDEL-AJB-BALIKNAMA', 1, 1, 'AJB', 1, 2),
('WASIAT', 1, 1, 'Will', 0, 1),
('WASIAT-DAFTAR', 1, 1, NULL, 0, 1),
('KUASA', 1, 1, 'POA', 0, 2),
('PRENUP', 1, 1, 'Prenup', 0, 2),
('HUTANG', 1, 1, 'Agreement', 0, 2),
('PK-UMUM', 1, 1, 'Agreement', 0, 2),
('SEWA', 1, 1, 'Agreement', 0, 2),
('JUALBELI-NONPPAT', 1, 1, 'Agreement', 0, 2),
('PT-DIRIAN', 1, 1, 'PT', 0, 1),
('CV-DIRIAN', 1, 1, 'CV', 0, 1),
('FIRMA', 1, 1, 'Firma', 0, 1),
('YAYASAN-DIRIAN', 1, 1, 'Yayasan', 0, 1),
('KOPERASI', 1, 1, 'Koperasi', 0, 1);

-- 3) Nomor register wasiat unik per kantor (7 baris existing terverifikasi tanpa duplikat).
-- MySQL mengizinkan banyak NULL dalam UNIQUE KEY, jadi baris tanpa nomor tetap aman.
CREATE UNIQUE INDEX uq_will_register ON wills (office_id, register_no);
