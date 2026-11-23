-- 022_notaris_lengkap.sql — lengkapi katalog Notaris-PPAT (tanpa ubah data pengguna).
-- Gaya 016-021: CREATE TABLE IF NOT EXISTS + ALTER polos + INSERT IGNORE.

-- 1) Jenis akta Firma & Koperasi (sebelumnya jatuh ke Other).
ALTER TABLE deed_drafts
  MODIFY COLUMN kind ENUM('PT','CV','Firma','Koperasi','Yayasan','Agreement','Will','POA','Prenup','Other',
    'AJB','Hibah','TukarMenukar','APHB','Inbreng','SKMHT','APHT') NOT NULL DEFAULT 'Other';

-- 2) Layanan Inbreng (deed kind-nya sudah ada sejak 021).
INSERT IGNORE INTO services(code, category, name, base_price, requirements) VALUES
('PPAT-INBRENG','ppat','Inbreng Tanah ke PT',3000000,'["Sertifikat","KTP pemilik","Akta pendirian/perubahan PT","RUPS persetujuan inbreng"]');
