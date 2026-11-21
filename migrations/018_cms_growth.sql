-- 018_cms_growth.sql — CMS growth: inquiries, booking, engagement, i18n, versions, stats.
-- Gaya sama seperti 016/017: CREATE TABLE IF NOT EXISTS + ALTER polos (kolom baru,
-- aman karena belum ada di DB mana pun). Versi dicatat di schema_migrations.

-- 1. Konsultasi masuk (formulir publik).
CREATE TABLE IF NOT EXISTS website_inquiries (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(160) NOT NULL,
  contact VARCHAR(160) NOT NULL,
  service VARCHAR(160) NOT NULL DEFAULT '',
  message TEXT NOT NULL,
  status ENUM('BARU','DIPROSES','SELESAI','SPAM') NOT NULL DEFAULT 'BARU',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_inq_office (office_id, status)
);

-- 2. Slot ketersediaan + booking publik.
CREATE TABLE IF NOT EXISTS website_slots (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  weekday TINYINT NOT NULL DEFAULT 1,
  start_time TIME NOT NULL DEFAULT '09:00:00',
  end_time TIME NOT NULL DEFAULT '12:00:00',
  quota INT NOT NULL DEFAULT 4,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  INDEX idx_slot_office (office_id, weekday)
);

CREATE TABLE IF NOT EXISTS website_bookings (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  book_date DATE NOT NULL,
  start_time TIME NOT NULL,
  name VARCHAR(160) NOT NULL,
  contact VARCHAR(160) NOT NULL,
  purpose VARCHAR(255) NOT NULL DEFAULT '',
  status ENUM('MENUNGGU','DISETUJUI','DITOLAK','BATAL') NOT NULL DEFAULT 'MENUNGGU',
  appointment_id BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_book_office (office_id, status, book_date)
);

-- 3. Tim, galeri, testimoni (terverifikasi, tanpa rating palsu).
CREATE TABLE IF NOT EXISTS website_team (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(160) NOT NULL,
  title VARCHAR(160) NOT NULL DEFAULT '',
  photo_media_id BIGINT UNSIGNED NULL,
  position INT NOT NULL DEFAULT 0,
  visible TINYINT(1) NOT NULL DEFAULT 1,
  INDEX idx_team_office (office_id, visible)
);

CREATE TABLE IF NOT EXISTS website_gallery (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  media_id BIGINT UNSIGNED NOT NULL,
  caption VARCHAR(255) NOT NULL DEFAULT '',
  position INT NOT NULL DEFAULT 0,
  INDEX idx_gal_office (office_id)
);

CREATE TABLE IF NOT EXISTS website_testimonials (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(160) NOT NULL,
  content TEXT NOT NULL,
  status ENUM('DRAFT','PUBLISHED') NOT NULL DEFAULT 'DRAFT',
  verify_note VARCHAR(500) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_testi_office (office_id, status)
);

-- 4. Taut layanan CMS -> katalog services (syarat + harga jujur dari DB).
ALTER TABLE website_services
  ADD COLUMN service_code VARCHAR(50) NULL AFTER slug;

-- 5. i18n: locale ID/EN untuk konten publik (default 'id', EN fallback ke ID).
ALTER TABLE website_pages
  ADD COLUMN locale VARCHAR(5) NOT NULL DEFAULT 'id';
ALTER TABLE website_articles
  ADD COLUMN locale VARCHAR(5) NOT NULL DEFAULT 'id';
ALTER TABLE website_services
  ADD COLUMN locale VARCHAR(5) NOT NULL DEFAULT 'id';
ALTER TABLE website_faqs
  ADD COLUMN locale VARCHAR(5) NOT NULL DEFAULT 'id';

-- 6. Versi konten (rollback jujur, maks dipangkas di aplikasi).
CREATE TABLE IF NOT EXISTS website_content_versions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('page','article') NOT NULL,
  ref_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(200) NOT NULL DEFAULT '',
  content MEDIUMTEXT NOT NULL,
  actor_id BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_ver_ref (office_id, kind, ref_id)
);

-- 7. Statistik kunjungan internal (agregat harian, tanpa tracker pihak ketiga).
CREATE TABLE IF NOT EXISTS website_pageviews (
  office_id BIGINT UNSIGNED NOT NULL,
  path VARCHAR(255) NOT NULL,
  day DATE NOT NULL,
  views INT NOT NULL DEFAULT 0,
  PRIMARY KEY (office_id, path, day)
);

-- 8. Mode pemeliharaan + banner pengumuman.
ALTER TABLE website_settings
  ADD COLUMN maintenance TINYINT(1) NOT NULL DEFAULT 0,
  ADD COLUMN announcement VARCHAR(500) NOT NULL DEFAULT '';

-- 9. Cermin izin baru untuk UI /roles (penegakan tetap di kode rbac.go).
INSERT IGNORE INTO permissions(name) VALUES
  ('website.inquiries.manage'),('website.engagement.manage');
INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON (
  (r.name IN ('super_admin'))
  OR (r.name='office_admin' AND p.name IN ('website.inquiries.manage','website.engagement.manage'))
  OR (r.name IN ('notary','substitute_notary','staff') AND p.name IN ('website.inquiries.manage','website.engagement.manage'))
);
