-- 027_workflow_engine.sql — Workflow Engine generik (fondasi data, TANPA ubah handler existing).
-- Gaya 016-026: CREATE TABLE IF NOT EXISTS, tanpa seed, tanpa DROP/ALTER tabel lama.
-- Aditif murni: workflowGate (internal/app/wf_rules.go) + service_workflow_rules tetap jalan;
-- tabel baru ini fondasi template/steps/instances generik untuk tahap berikutnya.
-- Office-scoped: templates + instances bawa office_id (FK offices CASCADE);
-- steps/states ikut kantor via rantai FK (template -> instance) dengan ON DELETE CASCADE.

CREATE TABLE IF NOT EXISTS workflow_templates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  service_code VARCHAR(50) NOT NULL,
  name VARCHAR(255) NOT NULL,
  version INT NOT NULL DEFAULT 1,
  active TINYINT(1) NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  UNIQUE KEY uq_wf_template (office_id, service_code, version),
  INDEX idx_wf_template_office_svc (office_id, service_code)
);

CREATE TABLE IF NOT EXISTS workflow_steps (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  template_id BIGINT UNSIGNED NOT NULL,
  step_key VARCHAR(100) NOT NULL,
  label VARCHAR(255) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  required_role VARCHAR(50) NOT NULL DEFAULT '',
  require_checklist TINYINT(1) NOT NULL DEFAULT 1,
  require_evidence VARCHAR(50) NULL,
  skippable TINYINT(1) NOT NULL DEFAULT 0,
  FOREIGN KEY (template_id) REFERENCES workflow_templates(id) ON DELETE CASCADE,
  UNIQUE KEY uq_wf_step (template_id, step_key),
  INDEX idx_wf_step_order (template_id, sort_order)
);

CREATE TABLE IF NOT EXISTS workflow_instances (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  office_id BIGINT UNSIGNED NOT NULL,
  request_id BIGINT UNSIGNED NOT NULL UNIQUE,
  template_id BIGINT UNSIGNED NOT NULL,
  current_step VARCHAR(100) NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (office_id) REFERENCES offices(id) ON DELETE CASCADE,
  FOREIGN KEY (request_id) REFERENCES service_requests(id) ON DELETE CASCADE,
  FOREIGN KEY (template_id) REFERENCES workflow_templates(id) ON DELETE CASCADE,
  INDEX idx_wf_inst_office (office_id),
  INDEX idx_wf_inst_template (template_id)
);

CREATE TABLE IF NOT EXISTS workflow_step_states (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  instance_id BIGINT UNSIGNED NOT NULL,
  step_key VARCHAR(100) NOT NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
  actor VARCHAR(100) NULL,
  note VARCHAR(500) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (instance_id) REFERENCES workflow_instances(id) ON DELETE CASCADE,
  UNIQUE KEY uq_wf_state (instance_id, step_key),
  INDEX idx_wf_state_status (instance_id, status)
);
