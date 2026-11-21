# PROJECT TEST MATRIX — status bukti

| Jenis | Perintah | Status |
|---|---|---|
| gofmt | `go fmt ./...` | HIJAU |
| vet | `go vet ./...` | HIJAU |
| build | `go build ./...` | HIJAU |
| unit | `go test ./internal/...` (+rename AV) | HIJAU (pdf, diff, verify, plan PASS) |
| integrasi MySQL | `NOTARYOS_TEST_DSN=... go test ./tests/` | 20/20 PASS (dms_test + inti; TestDeedVersionLock tangkap bug repertorium) |
| migrasi | boot otomatis 001-013 | HIJAU (count 0, siap) |
| E2E browser DMS | `scripts/qa_shots/dms_e2e.py` | 10/10 PASS (bukti docs/qa/DMS_E2E_LOG.md) |
| E2E Fase 1 jualan | `fase1a.py` 6/6 + `fase1b.py` 8/8 | PASS: kuitansi QR HMAC, /verify publik + tolak palsu, portal bayar + bukti, pengingat |
| E2E Fase 2 jualan | `fase2.py` 7/7 | PASS: template checklist→auto-create+progress, diff highlight, label QR |
| E2E Fase 3 (TTE+multi-kantor+tier) | `fase3.py` 14/14 | PASS: offices ENTERPRISE+acting+HMAC, approve→segel DEED-*→/verify akta, sign hash-64, gate LITE→402→PRO |
| E2E bisnis | smoke login→permohonan→akta→invoice | HIJAU (log audit) |
| screenshot | `scripts/qa_shots/*.py` | 15+ halaman OK |
| race detector | `go test -race` | BACKLOG (host Windows/AV) |
| restore VPS | — | BACKLOG (B-09) |
| pen-test | — | BACKLOG (butuh pihak ketiga) |
