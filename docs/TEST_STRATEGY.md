# TEST_STRATEGY
- Unit: money (parse/PPN/format/overflow), docengine (render/missing/hash/watermark),
  rbac (matriks izin), requests (transisi), gov (alur asistensi), validate (NIK/NPWP), auth (bcrypt/sesi).
- Integrasi DB (MySQL asli, bukan mock): users.authenticate, clients CRUD+duplikat,
  requests move + events, deeds version lock, finance overpay — **Sprint 3** (butuh DB uji).
- E2E: login → klien → permohonan → akta → invoice → bayar via HTTP test — Sprint 3.
- Keamanan: RBAC handler test, XSS template (auto-escape), traversal storage — Sprint 3.
- Perintah: `go fmt ./... && go vet ./... && go test ./... && go test -race ./... && go build ./...`.
- Bukti: `docs/TEST_RESULTS.md` + `project/progress.json`. Mock bukan bukti integrasi MySQL.
