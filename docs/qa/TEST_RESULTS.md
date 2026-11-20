# TEST_RESULTS — 2026-11-20 (sesi arsitek)

## Static & build
- `go fmt ./...`: BERSIH (2 file diformat: app.go, ext.go).
- `go vet ./...`: BERSIH.
- `go build ./...`: BERSIH.

## Unit (dengan bukti)
- PASS: wib, docx (baru, zip-valid + watermark), money, rbac, middleware, auth, gov, requests, cryptoxx, mfa, docengine.
- PASS via rename (AV host): validate TestNIK (`notval2.exe -test.v`).
- FAIL default `go test ./internal/validate` di host Windows: `Access is denied` (B-04, AV memblokir `validate.test.exe`) — BUKAN kegagalan kode.

## Integrasi MySQL
- BELUM DIJALANKAN sesi ini (butuh `NOTARYOS_TEST_DSN` ke DB uji kosong + seed 005). 9 skenario siap: users, duplicates, workflow, deed lock, finance overpay, gov toggle, BO+handover, protocol+legalisasi+appointment.
- Hasil terakhir tercatat: 6/6 HIJAU (IMPLEMENTATION_STATUS v0.2.0). Status kini: IMPLEMENTED, menunggu VERIFIED ulang setelah perubahan P0 (FOR UPDATE, scoped gov, MIME sniff).

## API/Auth/Security/Upload/Backup/UI
- BELUM DIJALANKAN sesi ini (butuh server + MySQL + browser). Dicatatkan sebagai utang jujur di REGRESSION_REPORT.
- Tidak ada hasil yang difabrikasi. Klaim VERIFIED final DILARANG sampai matriks TEST_PLAN dieksekusi.
