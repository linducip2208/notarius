-- 024_bundling.sql — paket bundel layanan (reference-only, tanpa data bisnis).
-- Gaya 016-022: ALTER polos + INSERT IGNORE.

-- Kategori 'ppat' sudah dilebarkan di 021; ALTER ini dipertahankan sebagai
-- pengaman idempoten (no-op bila sudah sama) agar file ini tetap valid
-- mandiri di MySQL strict mode.
ALTER TABLE services
  MODIFY COLUMN category ENUM('corporate','agreement','will','legalization','deed','other','ppat') NOT NULL;

INSERT IGNORE INTO services(code, category, name, base_price, requirements) VALUES
('BUNDEL-PT-OSS','corporate','Paket Pendirian PT + NIB + NPWP',7500000,'["KTP pendiri","NPWP","Susunan direksi","Modal","KBLI"]'),
('BUNDEL-AJB-BALIKNAMA','ppat','Paket AJB + Balik Nama',5000000,'["Sertifikat","KTP penjual & pembeli","PBB lunas","PPh & BPHTB"]'),
('BUNDEL-CV-NIB','corporate','Paket CV + NIB',3500000,'["KTP sekutu","Domisili","KBLI"]');
