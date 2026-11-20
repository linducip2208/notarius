# TEST_PLAN — NOTARIUS

## Scope
Unit (gofmt, vet, `go test`, race bila didukung), integrasi MySQL asli (`tests/`, butuh NOTARYOS_TEST_DSN), migrasi rerun, API/auth/authz, dokumen PDF/DOCX panjang, upload spoof/traversal, backup/restore, UI manual browser (desktop/tablet/mobile), E2E login→permohonan→akta→repertorium→invoice→bayar.

## Matriks
| Area | Jenis | Perintah | Kriteria lolos |
|------|-------|----------|----------------|
| Format | static | `go fmt ./...` bersih | tanpa diff |
| Vet | static | `go vet ./...` | BERSIH |
| Unit | auto | `go test ./internal/...` (+rename utk AV) | semua PASS |
| Race | auto | `go test -race ./internal/...` | PASS bila didukung host |
| Integrasi | DB asli | `NOTARYOS_TEST_DSN=... go test ./tests -v` | 9 skenario PASS |
| Migrasi | DB | migrate → rerun → 009 applied | idempoten |
| Auth | API | login salah/benar, brute 11x | 401 vs 302 vs 429 |
| CSRF | API | POST tanpa/salah token | 403 |
| RBAC | API | client akses /settings,/finance,/wills | 403 + halaman denied |
| Dokumen | file | PDF+DOCX panjang+tabel+TTD | terbuka valid, watermark DRAFT |
| Upload | file | EXE-as-PDF, `..%2f`, CD header | ditolak/aman |
| Backup | ops | backup.sh → restore.sh ke DB kosong | data kembali |
| UI | manual | semua menu+tombol | tanpa dead link/form rusak |

## Larangan
- Tanpa mock DB sebagai bukti MySQL. Tanpa klaim VERIFIED tanpa bukti uji.
