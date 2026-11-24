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
# buka http://localhost:8080/setup untuk admin pertama (sekali-pakai),
# selanjutnya http://localhost:8080/login
```

## Perintah wajib
```bash
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

## Membuat admin pertama
Cara utama: buka `/setup` (aktif hanya bila tabel users masih kosong;
begitu ada 1 user dialihkan ke `/login`). Fallback SQL bila perlu:
```sql
INSERT INTO offices(name, notary_name, sk_number, address, phone, email)
VALUES ('Kantor Notaris Contoh','Nama Notaris, S.H.','SK.123','Jl. Contoh 1','021-000','office@contoh.id');
-- role super_admin id dari tabel roles; hash bcrypt via utilitas internal/auth
INSERT INTO users(office_id, role_id, name, email, password_hash)
VALUES (1, 1, 'Admin','admin@contoh.id','<bcrypt-hash>');
```

## Struktur
`cmd/notaryos` entrypoint · `internal/` 38 paket · `migrations/` 001–034 (021 PPAT, 023 office ketat, 025 indeks, 026 matter/workflow, 027 engine, 028 audit office, 029 template unik, 030 quotation, 031 serah, 032 push, 033 SLA, 034 PPAT-BPN) ·
`internal/app/templates/` UI Tabler · `deploy/` nginx+systemd (timer pengingat)+backup DB & berkas ·
`docs/` USER_MANUAL + panduan (PRD, audit, DEPLOYMENT_GUIDE).

## Rilis
Lihat `docs/RELEASE_NOTES.md` untuk catatan rilis dan `docs/DEPLOYMENT_GUIDE.md` untuk go-live.
