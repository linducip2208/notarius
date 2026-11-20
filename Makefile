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
	mysql -h $${DB_HOST:-127.0.0.1} -u $${DB_USER:-notaryos} -p$${DB_PASS} $${DB_NAME:-notaryos} -e "source migrations/001_core.sql; source migrations/002_clients.sql; source migrations/003_deeds.sql; source migrations/004_ops.sql; source migrations/005_seed.sql; source migrations/006_portal.sql; source migrations/007_sessions.sql; source migrations/008_bo_handover.sql;"

seed-admin:
	go run ./scripts/seed

restore-check:
	bash deploy/backup/restore.sh --check-only || true
