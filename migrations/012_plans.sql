-- 012_plans.sql — paket layanan per kantor (LITE/PRO/ENTERPRISE).
ALTER TABLE offices ADD COLUMN plan ENUM('LITE','PRO','ENTERPRISE') NOT NULL DEFAULT 'PRO';
