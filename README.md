# SaktiNotaris — Sistem Manajemen Kantor Notaris Indonesia
Backend 100% Go · MySQL 8.x · UI Tabler + HTMX · Modular monolith.

## Syarat
- Go ≥ 1.24, MySQL 8.x, (opsional) Tesseract OCR untuk ekstraksi KTP.

## Jalan cepat (dev)
```bash
cp .env.example .env          # isi DB_PASS
mysql -u root -p -e "CREATE DATABASE notaryos CHARACTER SET utf8mb4;"
make migrate                  # butuh akses mysql client
go run ./cmd/notaryos
# buka http://localhost:8080/login
```

## Perintah wajib
```bash
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

## Membuat admin pertama
```sql
INSERT INTO offices(name, notary_name, sk_number, address, phone, email)
VALUES ('Kantor Notaris Contoh','Nama Notaris, S.H.','SK.123','Jl. Contoh 1','021-000','office@contoh.id');
-- role super_admin id dari tabel roles; hash bcrypt via utilitas internal/auth
INSERT INTO users(office_id, role_id, name, email, password_hash)
VALUES (1, 1, 'Admin','admin@contoh.id','<bcrypt-hash>');
```

## Struktur
`cmd/notaryos` entrypoint · `internal/` 19 paket domain · `migrations/` skema MySQL ·
`internal/app/templates/` UI Tabler · `deploy/` nginx+systemd+backup · `docs/` PRD & audit.
