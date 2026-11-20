# TEST_RESULTS — 2026-11-20 (sprint 2)
- `go build ./...`: PASS. `go vet ./...`: PASS.
- Unit PASS (9 paket): auth, money, docengine, rbac, requests, gov, validate (via biner
  rename, B-04), mfa (TOTP roundtrip), cryptoxx (AES-GCM roundtrip), middleware (rate-limit 3→429).
- Integrasi MySQL asli (`tests/`, NOTARYOS_TEST_DSN, db `notaryos_test`): 6/6 PASS —
  auth, klien+duplikat+verifikasi identitas, workflow + counts, kunci versi akta,
  overpay + piutang, asistensi AHU/OSS.
- Smoke test HTTP end-to-end (server :18080, DB `notaryos`): login→dashboard→klien→
  cari→permohonan→move→detail→korporat→saham→pengurus→akta→PDF→invoice 3 baris
  (3.774.000)→bayar 2jt→overpay ditolak (sisa 3.550.000)→wasiat→rate-limit 429. SEMUA LOLOS.
  Bug yang ditemukan & diperbaiki: kolom invoice, DECIMAL scan, kunci APPROVED, limiter IP.
- `go test -race`: butuh cgo/gcc — wajib di CI Linux.
