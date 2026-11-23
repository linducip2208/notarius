-- 021_ppat.sql — Notaris-PPAT: jenis akta tanah, register PPAT, layanan PPAT.
-- Gaya 016-020: CREATE TABLE IF NOT EXISTS + ALTER polos, seed reference-only.

-- 1) Jenis akta PPAT masuk ke deed_drafts (tetap 1 tabel draf, register terpisah).
ALTER TABLE deed_drafts
  MODIFY COLUMN kind ENUM('PT','CV','Yayasan','Agreement','Will','POA','Prenup','Other',
    'AJB','Hibah','TukarMenukar','APHB','Inbreng','SKMHT','APHT') NOT NULL DEFAULT 'Other';

-- 2) Register PPAT terpisah dari Repertorium Notaris (wajib: 2 buku).
CREATE TABLE IF NOT EXISTS ppat_register (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  deed_id BIGINT UNSIGNED NULL,
  akta_no VARCHAR(50) NOT NULL,
  akta_date DATE NULL,
  jenis ENUM('AJB','Hibah','TukarMenukar','APHB','Inbreng','SKMHT','APHT') NOT NULL DEFAULT 'AJB',
  penjual VARCHAR(255) NULL,
  pembeli VARCHAR(255) NULL,
  objek_sertifikat VARCHAR(100) NULL,
  objek_luas VARCHAR(50) NULL,
  objek_lokasi VARCHAR(255) NULL,
  status ENUM('DRAFT_AKTA','AKTA_JADI','LAPOR_BPN','SERTIFIKAT_TERBIT') NOT NULL DEFAULT 'DRAFT_AKTA',
  bpn_receipt VARCHAR(100) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE RESTRICT,
  FOREIGN KEY (deed_id) REFERENCES deed_drafts(id) ON DELETE SET NULL,
  UNIQUE KEY uq_ppat (office_id, akta_no)
);

-- 3) Kategori 'ppat' belum ada di ENUM services (002); lebarkan DULU agar
-- insert di bawah valid di MySQL strict mode (tanpa ini, INSERT IGNORE
-- menyimpan '' diam-diam / corrupt kategori).
ALTER TABLE services
  MODIFY COLUMN category ENUM('corporate','agreement','will','legalization','deed','other','ppat') NOT NULL;

-- 4) Katalog layanan PPAT (reference-only, tanpa data bisnis).
INSERT IGNORE INTO services(code, category, name, base_price, requirements) VALUES
('PPAT-AJB','ppat','AJB (Akta Jual Beli Tanah)',3500000,'["Sertifikat asli","KTP/KK/NPWP penjual & pembeli + pasangan","PBB lunas","PPh & BPHTB"]'),
('PPAT-HIBAH','ppat','Akta Hibah Tanah',3000000,'["Sertifikat asli","KTP pemberi & penerima hibah","PBB lunas","BPHTB"]'),
('PPAT-TUKAR','ppat','Akta Tukar Menukar',3500000,'["Dua sertifikat asli","KTP kedua belah pihak","PBB lunas kedua objek"]'),
('PPAT-APHB','ppat','APHB (Akta Pembagian Hak Bersama)',3000000,'["Sertifikat waris","SKW + KTP semua ahli waris","PBB lunas"]'),
('PPAT-SKMHT','ppat','SKMHT (Kuasa Membebankan HT)',1500000,'["Sertifikat","KTP pemberi & penerima kuasa","Data kredit bank"]'),
('PPAT-APHT','ppat','APHT (Pembebanan Hak Tanggungan)',2500000,'["Sertifikat","SKMHT bila dikuasakan","Data kredit & peringkat"]');
