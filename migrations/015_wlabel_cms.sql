-- 015_wlabel_cms.sql — White Label + Website CMS (standalone per kantor).
-- Identitas inti tetap di `offices`; branding di `office_branding`; konten di `website_*`.

-- Idempotent contact columns (015 pernah gagal di tengah; aman diulang & di fresh install).
SET @have_whatsapp = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='offices' AND COLUMN_NAME='whatsapp');
SET @add_whatsapp = IF(@have_whatsapp=0,'ALTER TABLE offices ADD COLUMN whatsapp VARCHAR(50) NOT NULL DEFAULT ''''','SELECT 1');
PREPARE stmt_whatsapp FROM @add_whatsapp;
EXECUTE stmt_whatsapp;
DEALLOCATE PREPARE stmt_whatsapp;
SET @have_website = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='offices' AND COLUMN_NAME='website');
SET @add_website = IF(@have_website=0,'ALTER TABLE offices ADD COLUMN website VARCHAR(255) NOT NULL DEFAULT ''''','SELECT 1');
PREPARE stmt_website FROM @add_website;
EXECUTE stmt_website;
DEALLOCATE PREPARE stmt_website;

CREATE TABLE IF NOT EXISTS office_branding (
  office_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  app_name VARCHAR(120) NOT NULL DEFAULT 'SaktiNotaris',
  short_name VARCHAR(40) NOT NULL DEFAULT 'SN',
  tagline VARCHAR(255) NOT NULL DEFAULT '',
  primary_color CHAR(7) NOT NULL DEFAULT '#1f6feb',
  secondary_color CHAR(7) NOT NULL DEFAULT '#101f3c',
  accent_color CHAR(7) NOT NULL DEFAULT '#c9a844',
  sidebar_bg CHAR(7) NOT NULL DEFAULT '#101f3c',
  header_bg CHAR(7) NOT NULL DEFAULT '#ffffff',
  button_style ENUM('square','rounded','pill') NOT NULL DEFAULT 'rounded',
  border_radius TINYINT UNSIGNED NOT NULL DEFAULT 8,
  font_family VARCHAR(80) NOT NULL DEFAULT 'system',
  logo_ref VARCHAR(255) NOT NULL DEFAULT '',
  favicon_ref VARCHAR(255) NOT NULL DEFAULT '',
  login_bg VARCHAR(255) NOT NULL DEFAULT '',
  footer_text VARCHAR(255) NOT NULL DEFAULT '',
  copyright_text VARCHAR(255) NOT NULL DEFAULT '',
  default_lang CHAR(5) NOT NULL DEFAULT 'id',
  timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Jakarta',
  date_format VARCHAR(32) NOT NULL DEFAULT '02 Jan 2006',
  currency VARCHAR(8) NOT NULL DEFAULT 'IDR',
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS website_settings (
  office_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  site_title VARCHAR(160) NOT NULL DEFAULT '',
  site_desc VARCHAR(255) NOT NULL DEFAULT '',
  default_og VARCHAR(255) NOT NULL DEFAULT '',
  base_url VARCHAR(255) NOT NULL DEFAULT '',
  primary_domain VARCHAR(253) NOT NULL DEFAULT '',
  enforce_https TINYINT(1) NOT NULL DEFAULT 0,
  domain_verified TINYINT(1) NOT NULL DEFAULT 0,
  last_seen_host VARCHAR(253) NOT NULL DEFAULT '',
  setup_done TINYINT(1) NOT NULL DEFAULT 0,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS website_pages (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(200) NOT NULL,
  slug VARCHAR(200) NOT NULL,
  excerpt VARCHAR(255) NOT NULL DEFAULT '',
  content MEDIUMTEXT NOT NULL,
  featured_media_id BIGINT UNSIGNED NULL,
  template VARCHAR(40) NOT NULL DEFAULT 'page',
  status ENUM('DRAFT','REVIEW','SCHEDULED','PUBLISHED','ARCHIVED') NOT NULL DEFAULT 'DRAFT',
  scheduled_at TIMESTAMP NULL,
  published_at TIMESTAMP NULL,
  seo_title VARCHAR(200) NOT NULL DEFAULT '',
  seo_desc VARCHAR(255) NOT NULL DEFAULT '',
  canonical_url VARCHAR(255) NOT NULL DEFAULT '',
  og_image VARCHAR(255) NOT NULL DEFAULT '',
  robots VARCHAR(80) NOT NULL DEFAULT 'index,follow',
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP NULL,
  UNIQUE KEY uq_wpage (office_id, slug),
  INDEX idx_wpage_status (office_id, status),
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS website_page_sections (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  page_id BIGINT UNSIGNED NOT NULL,
  office_id BIGINT UNSIGNED NOT NULL,
  stype VARCHAR(40) NOT NULL,
  title VARCHAR(200) NOT NULL DEFAULT '',
  subtitle VARCHAR(255) NOT NULL DEFAULT '',
  content MEDIUMTEXT NOT NULL,
  image_media_id BIGINT UNSIGNED NULL,
  cta_label VARCHAR(120) NOT NULL DEFAULT '',
  cta_url VARCHAR(500) NOT NULL DEFAULT '',
  background VARCHAR(40) NOT NULL DEFAULT 'light',
  align ENUM('left','center','right') NOT NULL DEFAULT 'left',
  layout VARCHAR(40) NOT NULL DEFAULT 'standard',
  visible TINYINT(1) NOT NULL DEFAULT 1,
  position INT NOT NULL DEFAULT 0,
  settings JSON NULL,
  FOREIGN KEY (page_id) REFERENCES website_pages(id) ON DELETE CASCADE,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  INDEX idx_wpsec (page_id, position)
);

CREATE TABLE IF NOT EXISTS website_categories (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(120) NOT NULL,
  slug VARCHAR(140) NOT NULL,
  UNIQUE KEY uq_wcat (office_id, slug),
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS website_articles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  category_id BIGINT UNSIGNED NULL,
  title VARCHAR(200) NOT NULL,
  slug VARCHAR(200) NOT NULL,
  excerpt VARCHAR(255) NOT NULL DEFAULT '',
  content MEDIUMTEXT NOT NULL,
  cover_media_id BIGINT UNSIGNED NULL,
  tags VARCHAR(255) NOT NULL DEFAULT '',
  author VARCHAR(120) NOT NULL DEFAULT '',
  status ENUM('DRAFT','REVIEW','SCHEDULED','PUBLISHED','ARCHIVED') NOT NULL DEFAULT 'DRAFT',
  scheduled_at TIMESTAMP NULL,
  published_at TIMESTAMP NULL,
  seo_title VARCHAR(200) NOT NULL DEFAULT '',
  seo_desc VARCHAR(255) NOT NULL DEFAULT '',
  canonical_url VARCHAR(255) NOT NULL DEFAULT '',
  og_image VARCHAR(255) NOT NULL DEFAULT '',
  robots VARCHAR(80) NOT NULL DEFAULT 'index,follow',
  created_by BIGINT UNSIGNED NULL,
  updated_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP NULL,
  UNIQUE KEY uq_wart (office_id, slug),
  INDEX idx_wart_status (office_id, status, published_at),
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  FOREIGN KEY (category_id) REFERENCES website_categories(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS website_services (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(160) NOT NULL,
  slug VARCHAR(180) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  content MEDIUMTEXT NOT NULL,
  image_media_id BIGINT UNSIGNED NULL,
  position INT NOT NULL DEFAULT 0,
  visible TINYINT(1) NOT NULL DEFAULT 1,
  seo_title VARCHAR(200) NOT NULL DEFAULT '',
  seo_desc VARCHAR(255) NOT NULL DEFAULT '',
  UNIQUE KEY uq_wsvc (office_id, slug),
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS website_faqs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  category VARCHAR(120) NOT NULL DEFAULT 'Umum',
  question VARCHAR(255) NOT NULL,
  answer TEXT NOT NULL,
  position INT NOT NULL DEFAULT 0,
  visible TINYINT(1) NOT NULL DEFAULT 1,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  INDEX idx_wfaq (office_id, visible, position)
);

CREATE TABLE IF NOT EXISTS website_nav_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  area ENUM('header','footer') NOT NULL DEFAULT 'header',
  label VARCHAR(120) NOT NULL,
  url VARCHAR(500) NOT NULL,
  parent_id BIGINT UNSIGNED NULL,
  position INT NOT NULL DEFAULT 0,
  visible TINYINT(1) NOT NULL DEFAULT 1,
  new_tab TINYINT(1) NOT NULL DEFAULT 0,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_id) REFERENCES website_nav_items(id) ON DELETE CASCADE,
  INDEX idx_wnav (office_id, area, position)
);

CREATE TABLE IF NOT EXISTS website_media (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  ref VARCHAR(255) NOT NULL,
  original_name VARCHAR(255) NOT NULL,
  mime VARCHAR(120) NOT NULL,
  size_bytes BIGINT UNSIGNED NOT NULL DEFAULT 0,
  width INT NOT NULL DEFAULT 0,
  height INT NOT NULL DEFAULT 0,
  alt VARCHAR(200) NOT NULL DEFAULT '',
  caption VARCHAR(255) NOT NULL DEFAULT '',
  folder VARCHAR(80) NOT NULL DEFAULT 'umum',
  created_by BIGINT UNSIGNED NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  INDEX idx_wmedia (office_id, folder, created_at)
);

CREATE TABLE IF NOT EXISTS website_media_usage (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  media_id BIGINT UNSIGNED NOT NULL,
  used_in VARCHAR(40) NOT NULL,
  used_id BIGINT UNSIGNED NOT NULL,
  FOREIGN KEY (media_id) REFERENCES website_media(id) ON DELETE CASCADE,
  INDEX idx_wmuse (media_id)
);

CREATE TABLE IF NOT EXISTS website_publication_history (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('page','article') NOT NULL,
  ref_id BIGINT UNSIGNED NOT NULL,
  action VARCHAR(40) NOT NULL,
  from_status VARCHAR(20) NOT NULL DEFAULT '',
  to_status VARCHAR(20) NOT NULL DEFAULT '',
  actor_id BIGINT UNSIGNED NULL,
  note VARCHAR(255) NOT NULL DEFAULT '',
  ip VARCHAR(64) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  INDEX idx_wpub (office_id, kind, ref_id)
);
