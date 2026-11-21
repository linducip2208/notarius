# PROJECT TEST MATRIX — status bukti

| Jenis | Perintah | Status |
|---|---|---|
| gofmt | `go fmt ./...` | HIJAU |
| vet | `go vet ./...` | HIJAU |
| build | `go build ./...` | HIJAU |
| unit | `go test ./internal/...` (+rename AV) | HIJAU (pdf baru PASS) |
| integrasi MySQL | `NOTARYOS_TEST_DSN=... go test ./tests/` | 17/17 PASS (dms_test 8 + inti 9; TestDeedVersionLock tangkap bug repertorium) |
| migrasi | boot otomatis 001-011 | HIJAU (count 0, siap) |
| E2E browser DMS | `scripts/qa_shots/dms_e2e.py` | 10/10 PASS (bukti docs/qa/DMS_E2E_LOG.md) |
| E2E Fase 1 jualan | `fase1a.py` 6/6 + `fase1b.py` 8/8 | PASS: kuitansi QR HMAC, /verify publik + tolak palsu, portal bayar + bukti, pengingat |
| E2E bisnis | smoke login→permohonan→akta→invoice | HIJAU (log audit) |
| screenshot | `scripts/qa_shots/*.py` | 15+ halaman OK |
| race detector | `go test -race` | BACKLOG (host Windows/AV) |
| restore VPS | — | BACKLOG (B-09) |
| pen-test | — | BACKLOG (butuh pihak ketiga) |
