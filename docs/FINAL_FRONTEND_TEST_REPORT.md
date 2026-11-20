# FINAL_FRONTEND_TEST_REPORT — 2026-11-21

## Build & statik
- `go fmt ./...`: BERSIH. `go vet ./...`: BERSIH. `go build ./...`: BERSIH.
- Unit: wib, docx, money, rbac, middleware, auth, gov, requests, cryptoxx, mfa, docengine — PASS.
  validate: PASS via biner rename (AV host, B-04).

## HTTP (server lokal :8080, login admin@contoh.id)
- 29/29 rute → 200: `/`, `/app`, `/dashboard`(redirect), `/login`, `/clients`, `/clients/export`,
  `/requests`, `/deeds`, `/repertorium`, `/repertorium/export`, `/protocols`, `/legalisasi`,
  `/gov`, `/finance`, `/expenses`, `/tariffs`, `/templates`, `/reports`, `/audit`, `/office`,
  `/tasks`, `/documents`, `/search`, `/portal`, `/settings`, `/metrics`,
  vendor CSS/JS/htmx lokal, 3 PNG landing.
- Konten: dashboard memuat Agenda/Aktivitas 6 Bulan/Notifikasi/Menunggu/Piutang; `/` memuat
  landing baru (bukan dashboard lama).

## Check nyata
- Nol `cdn.jsdelivr|unpkg` di template (grep kosong). CSP tanpa host CDN.
- Sidebar gating: tamu → redirect login; admin melihat semua grup (terverifikasi manual 1 peran;
  matrix 9 peran BELUM diuji sistematis — utang QA).
- Komparasi akta, sisip klausul, filter repertorium, ekspor CSV: render OK; alur klik penuh browser
  BELUM diuji (tanpa otomasi) — wajib sebelum rilis.
- Mobile 375–430px: BELUM uji perangkat; hanya aturan CSS + layout responsif.
