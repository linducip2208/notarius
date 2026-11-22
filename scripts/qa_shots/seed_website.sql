INSERT INTO website_services(office_id,name,slug,description,content,image_media_id,position,visible,locale,seo_title,seo_desc)
SELECT 1,'Pendirian PT/CV','pendirian-pt-cv','Pendirian badan usaha lengkap dengan modal, pengurus, dan pemegang saham.','<p>Dokumen disiapkan sesuai bentuk badan usaha.</p>',2,0,1,'id','Pendirian PT/CV','Layanan pendirian PT dan CV'
WHERE NOT EXISTS(SELECT 1 FROM website_services WHERE office_id=1 AND slug='pendirian-pt-cv');
INSERT INTO website_services(office_id,name,slug,description,content,image_media_id,position,visible,locale,seo_title,seo_desc)
SELECT 1,'Akta Perjanjian','akta-perjanjian','Perjanjian, kuasa, dan kesepakatan tertulis lainnya.','<p>Draf disusun dari template yang telah disetujui notaris.</p>',3,1,1,'id','Akta Perjanjian','Layanan akta perjanjian'
WHERE NOT EXISTS(SELECT 1 FROM website_services WHERE office_id=1 AND slug='akta-perjanjian');
INSERT INTO website_services(office_id,name,slug,description,content,image_media_id,position,visible,locale,seo_title,seo_desc)
SELECT 1,'Wasiat','wasiat','Pencatatan dan penyimpanan wasiat dengan akses terbatas.','<p>Data pewasiat dan ahli waris dijaga kerahasiaannya.</p>',4,2,1,'id','Wasiat','Layanan wasiat'
WHERE NOT EXISTS(SELECT 1 FROM website_services WHERE office_id=1 AND slug='wasiat');
INSERT INTO website_services(office_id,name,slug,description,content,image_media_id,position,visible,locale,seo_title,seo_desc)
SELECT 1,'Legalisasi & Salinan','legalisasi-salinan','Legalisasi, waarmerking, salinan, dan kutipan dokumen.','<p>Setiap penerbitan salinan tercatat dan terhubung ke minuta.</p>',5,3,1,'id','Legalisasi & Salinan','Layanan legalisasi dokumen'
WHERE NOT EXISTS(SELECT 1 FROM website_services WHERE office_id=1 AND slug='legalisasi-salinan');

UPDATE website_page_sections SET image_media_id=1 WHERE office_id=1 AND page_id=2 AND stype='hero';
UPDATE website_page_sections SET image_media_id=2 WHERE office_id=1 AND page_id=2 AND stype='services';
UPDATE website_page_sections SET image_media_id=3 WHERE office_id=1 AND page_id=2 AND stype='process';

SELECT id,name,slug,image_media_id FROM website_services ORDER BY position;
