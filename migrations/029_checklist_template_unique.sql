-- 029_checklist_template_unique.sql — cegah duplikat template per layanan ( Data sudah dibersihkan: 12x PT-DIRIAN -> 1).
-- Idempoten: gagal 1061 bila indeks sudah ada -> dilewati runner.
ALTER TABLE doc_checklist_templates ADD UNIQUE KEY uq_checklist_template (office_id, service_code);
