# TEST_STRATEGY
- Unit: money (parse/PPN/format/overflow), docengine (render/missing/hash/watermark),
  rbac (matriks izin), requests (transisi), gov (alur asistensi), validate (NIK/NPWP), auth (bcrypt/sesi).
- Integrasi DB (MySQL asli, bukan mock): users.authenticate, clients CRUD+duplikat,
  requests move + events, deeds version lock, finance overpay — **Sprint 3** (butuh DB uji).
- E2E: login → klien → permohonan → akta → invoice → bayar via HTTP test — Sprint 3.
- Keamanan: RBAC handler test, XSS template (auto-escape), traversal storage — Sprint 3.
- Perintah: `go fmt ./... && go vet ./... && go test ./... && go test -race ./... && go build ./...`.
- Bukti: `docs/TEST_RESULTS.md` + `project/progress.json`. Mock bukan bukti integrasi MySQL.

## Status jujur CI-vs-lokal (PR #5, 2026-11-23)
Repo ini TIDAK punya `.github/workflows` (tidak ada CI MySQL service), jadi
aturan jujurnya: tidak ada workflow baru — yang jalan di CI hanyalah suite
tanpa dependensi eksternal. Klaim "hijau" hanya sah untuk kolom CI.

| Suite | Perintah | CI (tanpa Tesseract/DB) | Lokal (dengan deps) | Bila deps absen |
|---|---|---|---|---|
| `internal/ocr` murni (fixture `testdata/dummy_ktp.png` KTP PALSU "CONTOH SAJA"/NIK nol + konstanta teks ParseKTP/KK/NPWP + grade bounds + Classify/Noop) | `go test ./internal/ocr/ -count=1` | ✅ WAJIB lolos | ✅ lolos | N/A (tanpa deps) |
| `internal/ocr` OCR ujung-ke-ujung (`TestTesseractExtract_Fixture`) | `NOTARYOS_TEST_TESSERACT=1 OCR_BINARY=<path> go test ./internal/ocr/ -run TestTesseract -v` | ⏭️ SKIP eksplisit ("butuh OCR asli...") — BUKAN bukti OCR jalan | ✅ lolos bila `tesseract --list-langs` memuat `ind` | `t.Skipf` berpesan, exit 0 |
| `tests/` integrasi MySQL asli | `NOTARYOS_TEST_DSN="user:pass@tcp(127.0.0.1:3306)/notaryos_test?parseTime=true&loc=UTC" go test ./tests/ -count=1` | ⏭️ SKIP eksplisit via `TestMain` ("BUKAN bukti hijau...") — exit 0 | ✅ lolos bila DB uji + migrasi tersedia | pesan SKIP + exit 0, bukan FAIL |

Catatan:
- `TestAssessImage_Sample`/`TestBuildVariants` lama memakai path absolut mesin
  pribadi (`D:\folder selamat lindu\...`) sehingga SKIP diam-diam di CI —
  SUDAH DIHAPUS, diganti fixture repo `testdata/dummy_ktp.png` (regenerasi:
  `go run ./internal/ocr/testdata/generate_dummy.go` dari dir testdata/).
- `tests/integration_test.go` exit 0 tanpa DSN adalah SKIP yang disengaja agar
  CI tanpa DB tidak merah — JANGAN dibaca sebagai "integrasi lolos".
- Perintah lokal lengkap: `go test ./internal/ocr/ -count=1 -v` (selalu),
  lalu sesuai kebutuhan tambah `NOTARYOS_TEST_TESSERACT=1` dan/atau
  `NOTARYOS_TEST_DSN=...`.
