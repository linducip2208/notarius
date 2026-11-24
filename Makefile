APP=notaryos

.PHONY: run build fmt vet test migrate seed-admin

run:
	go run ./cmd/notaryos

build:
	go build -o bin/notaryos ./cmd/notaryos

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

migrate:
	mysql -h $${DB_HOST:-127.0.0.1} -u $${DB_USER:-notaryos} -p$${DB_PASS} $${DB_NAME:-notaryos} -e "source migrations/001_core.sql; source migrations/002_clients.sql; source migrations/003_deeds.sql; source migrations/004_ops.sql; source migrations/005_seed.sql; source migrations/006_portal.sql; source migrations/007_sessions.sql; source migrations/008_bo_handover.sql; source migrations/009_tasks.sql; source migrations/010_role_permissions.sql; source migrations/011_dms.sql; source migrations/012_plans.sql; source migrations/013_tte.sql; source migrations/014_merge.sql; source migrations/015_wlabel_cms.sql; source migrations/016_client_crm.sql; source migrations/017_dms_enterprise.sql; source migrations/018_cms_growth.sql; source migrations/019_ops_growth.sql; source migrations/020_compliance_hardening.sql; source migrations/021_ppat.sql; source migrations/022_notaris_lengkap.sql; source migrations/023_office_strict.sql; source migrations/024_bundling.sql; source migrations/025_indeks.sql; source migrations/026_matter_workflow.sql; source migrations/027_workflow_engine.sql; source migrations/028_audit_office.sql; source migrations/029_checklist_template_unique.sql; source migrations/030_quotations.sql; source migrations/031_deed_handover.sql; source migrations/032_push_subscriptions.sql; source migrations/033_workflow_sla.sql; source migrations/034_ppat_bpn_date.sql; source migrations/035_user_avatar.sql;"
# Catatan: boot aplikasi juga menjalankan SEMUA migrasi otomatis via
# database.Migrate (schema_migrations). Target ini hanya untuk DB kosong/manual.

seed-admin:
	go run ./scripts/seed

restore-check:
	bash deploy/backup/restore.sh --check-only || true
