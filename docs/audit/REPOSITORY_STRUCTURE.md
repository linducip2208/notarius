# REPOSITORY_STRUCTURE — v0.1
```
cmd/notaryos/main.go            entrypoint (migrate + serve)
internal/
  app/          router, handler, template Tabler (embed)
  config/       env-only config
  database/     open + migrasi Go
  middleware/   security headers, sesi, RBAC
  auth/         bcrypt + sesi
  users/ offices→(users)        akun & peran
  clients/      klien + duplikat
  requests/     katalog layanan, permohonan, workflow status
  deeds/        draf akta, repertorium, protokol, legalisasi
  finance/      invoice, pembayaran, beban (int64 rupiah)
  audit/        audit log imutabel
  docengine/    render {{var}}, versi, watermark
  ocr/          interface provider + Noop jujur
  gov/          asistensi AHU/OSS (tanpa endpoint liar)
  storage/      file privat + adapter S3
  workers/      background pool
  money/ validate/ rbac/
migrations/ 001_core, 002_clients, 003_deeds, 004_ops
deploy/ nginx, systemd, mysql, backup
docs/ PRD + audit + progres
project/ progress.json, features.json, roadmap.json
web/ (disiapkan untuk aset Tabler bila di-vendor-kan; saat ini via CDN)
```
