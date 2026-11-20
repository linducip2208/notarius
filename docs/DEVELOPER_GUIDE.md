# DEVELOPER_GUIDE
- Bahasa: Go saja untuk logika. HTML/CSS/JS Tabler hanya presentasi.
- Tambah domain: buat paket `internal/<domain>` berisi model+store+service;
  daftarkan store di `app.New`, route di `app.Router`, template di `templates/`,
  izin di `rbac.go`, tabel di `migrations/00N_*.sql`, uji di `<domain>_test.go`.
- Aturan: handler tipis; validasi di store/service; otorisasi via `RequirePerm`/cek `rbac.Has`;
  uang int64; waktu UTC; error Bahasa Indonesia yang jelas.
- Template: tambah `{{define "nama"}}` + render di handler; daftarkan link sidebar di `partials.gohtml`.
- Migrasi: file baru bernomor urut, idempoten (`IF NOT EXISTS`), satu tujuan per file.
- Sebelum commit: `go fmt ./... && go vet ./... && go test ./...`; update
  `docs/IMPLEMENTATION_STATUS.md`, `FEATURE_CATALOG.md`, `CHANGELOG.md`, `project/*.json`.
