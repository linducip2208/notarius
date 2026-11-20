-- 005_seed.sql — reference data only (service catalog + price items).
-- No fake business records: offices, clients, deeds are created by users.
INSERT IGNORE INTO services(code, category, name, base_price, requirements) VALUES
('PT-DIRIAN','corporate','Pendirian PT',5000000,'["KTP pendiri","NPWP","Susunan direksi & komisaris","Modal dasar/disetor"]'),
('PT-UBAH','corporate','Perubahan PT (AD/komposisi)',3500000,'["Akta terakhir","Data perubahan","Persetujuan pemegang saham"]'),
('PT-BUBAR','corporate','Pembubaran PT',4000000,'["Akta terakhir","Keputusan RUPS","Likuidator"]'),
('CV-DIRIAN','corporate','Pendirian CV',2500000,'["KTP sekutu","Domisili usaha"]'),
('CV-UBAH','corporate','Perubahan CV',1500000,'["Akta terakhir","Data perubahan"]'),
('FIRMA','corporate','Pendirian Firma',2500000,'["KTP sekutu","Domisili usaha"]'),
('YAYASAN-DIRIAN','corporate','Pendirian Yayasan',4000000,'["KTP pendiri","Struktur organ","Maksud & tujuan"]'),
('KOPERASI','corporate','Akta Koperasi',3000000,'["KTP pendiri","AD/ART konsep"]'),
('PK-UMUM','agreement','Perjanjian Kerjasama',2000000,'["Identitas para pihak","Objek perjanjian"]'),
('HUTANG','agreement','Perjanjian Utang-Piutang',2000000,'["Identitas para pihak","Nilai & tenor"]'),
('SEWA','agreement','Perjanjian Sewa-Menyewa',1500000,'["Identitas para pihak","Objek sewa","Jangka waktu"]'),
('JUALBELI-NONPPAT','agreement','Perjanjian Jual-Beli (non-PPAT)',2500000,'["Identitas para pihak","Objek & harga"]'),
('KUASA','agreement','Surat Kuasa Notariil',750000,'["Identitas pemberi & penerima kuasa","Ruang lingkup"]'),
('PRENUP','agreement','Perjanjian Kawin (Prenup/Postnup)',3000000,'["KTP calon/pasangan","Daftar harta"]'),
('WASIAT','will','Akta Wasiat (Olografis/Umum)',2500000,'["KTP pewasiat","Daftar ahli waris","Objek wasiat"]'),
('WASIAT-DAFTAR','will','Pendaftaran Wasiat',500000,'["Salinan akta wasiat"]'),
('LEGALISASI','legalization','Legalisasi Tanda Tangan',150000,'["Dokumen asli","KTP penghadap"]'),
('WAARMERKING','legalization','Waarmerking (pembukuan surat)',100000,'["Dokumen asli"]'),
('SALINAN','legalization','Salinan Akta',500000,'["Nomor akta","Keperluan"]'),
('KUTIPAN','legalization','Kutipan Akta',300000,'["Nomor akta","Keperluan"]'),
('AHU-BANTU','other','Asistensi Pengajuan AHU',1500000,'["Data perseroan lengkap"]'),
('OSS-BANTU','other','Asistensi Perizinan OSS',1000000,'["Data perusahaan & KBLI"]');

INSERT IGNORE INTO price_items(service_id, name, amount) VALUES
(NULL,'PNBP / biaya resmi (diteruskan)',50000),
(NULL,'Biaya materai & administrasi',20000);
