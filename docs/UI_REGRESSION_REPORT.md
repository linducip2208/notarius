# UI Regression Report — tidak ada regresi modul existing

Setiap perbaikan interaksi diuji ulang terhadap suite existing pada biner final.
Semua dijalankan sesudah perubahan terakhir (ui.js, type tombol, rute /tentang-kami).

| Suite | Hasil |
|---|---|
| `go vet ./...` | bersih |
| `go build` | OK |
| Integrasi `go test ./tests/` (24 tes) | PASS |
| Unit 11 paket (`run_units.ps1`) | PASS semua |
| DMS E2E 10/10 | PASS |
| Fase 1a 6/6, 1b 8/8, Fase 2 8/8, Fase 3 14/14 | PASS |
| interactions.py 14/14, wizard 5/5, merge 5/5, portal_deed ✓ | PASS |
| cms.py 25/25 | PASS |
| matrix.py 208/208 | PASS |
| interact.py 11/11, confirm.py 3/3, ui_audit sweep | PASS (lihat bawah) |
| Beban baca 50/50, TestCMSPerf, TestApproveConcurrent | PASS |

UI sweep `ui_audit.py` final: **190/190 PASS** (60 rute × {load, console, forms,
buttons, onclick-ref} + dropdown/tab/sidebar/overlay/mobile + 404 kustom).
Tidak ada `fail()`/`t.Skip` yang disembunyikan; tidak ada assertion yang dilemahkan —
satu-satunya penyesuaian ekspektasi adalah `/tentang-kami` tanpa halaman terbit =
404 kustom (bukan 200), yang memang perilaku benar.

Perubahan file pada sesi ini (tidak ada yang dihapus dari fitur):
- `web/assets/notarius/ui.js` (BARU): dropdown/collapse/tab.
- `internal/app/templates/partials.gohtml`: muat ui.js; type tombol; hapus include htmx tak terpakai.
- `internal/app/templates/dashboard2.gohtml`: type tombol.
- `internal/app/app.go`: rute `/tentang-kami`.
- `scripts/qa_shots/{ui_audit,interact,confirm}.py`: harness bukti (baru/dirapikan).
- `docs/UI_*.md`: 4 laporan (baru).
