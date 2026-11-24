-- 033_workflow_sla.sql — SLA per langkah + jatuh tempo step (fondasi eskalasi, TANPA ubah handler existing).
-- Gaya 016-032: ALTER polos (runner menoleransi DDL duplikat
-- 1060/1061/1050/1068/1091 untuk rerun parsial).
-- workflow_steps.sla_days: target hari penyelesaian langkah (NULL = tanpa target).
-- workflow_step_states.due_at: tanggal jatuh tempo langkah (diisi CURDATE+sla_days
-- saat state PENDING dibuat). started_at: kapan langkah mulai berjalan.
ALTER TABLE workflow_steps
  ADD COLUMN sla_days INT NULL DEFAULT NULL;

ALTER TABLE workflow_step_states
  ADD COLUMN due_at DATE NULL,
  ADD COLUMN started_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP;
