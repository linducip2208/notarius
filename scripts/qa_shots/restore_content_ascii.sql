UPDATE website_pages SET
  title='SaktiNotaris - Kantor Notaris di Jakarta',
  excerpt='Jasa notaris untuk kebutuhan korporasi, perjanjian, dan legalitas dokumen. Standalone satu kantor, bukan PPAT.',
  content='<h2>Melayani kebutuhan notaris kantor Anda</h2><p>SaktiNotaris membantu kantor notaris mengelola permohonan, identitas penghadap, akta, repertorium, protokol, dan keuangan dalam satu sistem mandiri. Sistem mencatat proses; keputusan hukum tetap berada di tangan Notaris.</p><p>Ruang lingkup: akta korporasi, perjanjian, wasiat, legalisasi, dan asistensi administrasi badan hukum. Kami bukan PPAT, bukan firma hukum, dan tidak mengklaim integrasi resmi pemerintah.</p>',
  seo_title='Notaris Jakarta - SaktiNotaris',
  seo_desc='Kantor notaris untuk pendirian PT, perjanjian, wasiat, dan legalisasi. Konsultasi melalui formulir resmi.'
WHERE slug='beranda' AND office_id=1;

DELETE FROM website_page_sections WHERE page_id=2 AND office_id=1;
INSERT INTO website_page_sections(office_id,page_id,stype,title,subtitle,content,cta_label,cta_url,background,align,layout,visible,position) VALUES
(1,2,'hero','Notaris untuk Kebutuhan Korporasi','Satu kantor. Satu sistem. Proses tercatat, keputusan tetap manusia.','<p>Siapkan dokumen lebih rapi dengan alur permohonan dan checklist yang jelas.</p>','Konsultasi','/konsultasi','dark','center','standard',1,0),
(1,2,'services','Layanan Utama','Apa yang kami bantu','<p><strong>Pendirian dan perubahan PT/CV</strong> - modal, pengurus, dan pemegang saham.</p><p><strong>Akta perjanjian dan kuasa</strong> - disusun dari template tersetujui notaris.</p><p><strong>Wasiat dan legalisasi</strong> - dicatat dengan pengamanan akses terbatas.</p>','','','light','left','standard',1,1),
(1,2,'process','Alur Kerja','Dari konsultasi hingga akta selesai','<p>1. Konsultasi dan pengecekan syarat. 2. Verifikasi identitas oleh staf. 3. Penyusunan draf akta berversi. 4. Review dan pengesahan Notaris. 5. Pencatatan repertorium dan pengarsipan.</p>','','','white','left','standard',1,2),
(1,2,'cta','Butuh Layanan Notaris?','Konsultasi awal dapat disampaikan melalui formulir resmi.','<p>Kami akan menghubungi Anda untuk melengkapi persyaratan.</p>','Ajukan Konsultasi','/konsultasi','primary','center','standard',1,3);
