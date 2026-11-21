# PROJECT TEST MATRIX — status bukti

| Jenis | Perintah | Status |
|---|---|---|
| gofmt | `go fmt ./...` | HIJAU |
| vet | `go vet ./...` | HIJAU |
| build | `go build ./...` | HIJAU |
| unit | `go test ./internal/...` (+rename AV) | HIJAU (pdf, diff, verify, plan PASS) |
| integrasi MySQL | `NOTARYOS_TEST_DSN=... go test ./tests/` | 20/20 PASS (dms_test + inti; TestDeedVersionLock tangkap bug repertorium) |
| migrasi | boot otomatis 001-014 | HIJAU (count 0, siap) |
| E2E browser DMS | `scripts/qa_shots/dms_e2e.py` | 10/10 PASS (bukti docs/qa/DMS_E2E_LOG.md) |
| E2E Fase 1 jualan | `fase1a.py` 6/6 + `fase1b.py` 8/8 | PASS: kuitansi QR HMAC, /verify publik + tolak palsu, portal bayar + bukti, pengingat |
| E2E Fase 2 jualan | `fase2.py` 8/8 | PASS: template checklist→auto-create+progress, revisi editor kaya + sanitasi, diff highlight, label QR |
| E2E Fase 3 (TTE+multi-kantor+tier) | `fase3.py` 14/14 | PASS: offices ENTERPRISE+acting+HMAC, approve→segel DEED-*→/verify akta, sign hash-64, gate LITE→402→PRO |
| Matriks izin 9 peran | `matrix.py` 208 sel | PASS semua cocok (8 peran × 26 rute + admin /offices); bukti `docs/qa/PERMISSION_MATRIX.md` |
| CMS E2E | `cms.py` 25/25 | branding+SEO+domain+validasi, halaman+section+terbit+render+sanitasi, beranda ganti landing, artikel+FAQ+nav, media+re-encode+tolak, sitemap, histori, setup contoh DRAF |
| CMS unit+integrasi | `internal/cms` + `TestCMSLifecycle` + `TestCMSPerf` | validasi/section-schema/sanitasi/media; slug ganda+publish-guard+jadwal+isolasi publik; 100 hal/500 art/1000 media <500ms |
| INTERACTION 1-12 | `interactions.py` 14/14 | login±, sidebar, +Buat, notif, 5 tab, filter, CSV 134b, revisi RTE, komparasi, repertorium+cetak, a11y landmark, logout |
| Wizard korporat | `wizard.py` 5/5 | PT 5/5 langkah + peringatan modal; unit `corp.WizardProgress` PASS |
| Gabung duplikat | `merge.py` 5/5 | unggah 2x → kandidat → N berkas digabung ke kanonis + banner; migrasi 014 ENUM file |
| Portal akta | `portal_deed.py` | kartu "Akta Saya" + Unduh PDF klien tertaut |
| Pratinjau OOXML | `officex` unit | docx/xlsx/tolak PASS; iframe teks 50KB di detail dokumen |
| Malware | `malware` unit | PING/OK/FOUND/ERROR vs server palsu PASS; tombol Pindai on-demand |
| S3 backend | `storage` unit | SigV4 roundtrip vs httptest PASS; aktif via NOTARYOS_S3_* env |
| RTE sanitasi | `safehtml` unit | strip-script/atribut/malformed/totext PASS |
| Gateway notif | `deliver` unit | webhook HMAC + 500-error + env-kosong PASS |
| SLA eskalasi | `TestRemindersSLA` | PASS: macet >3 hari → notif; COMPLETED tidak dieskalasi; anti-spam |
| Restore backup | dump+restore nyata | 0 selisih (users/deeds/invoices/files/requests); RTO <60 dtk; bukti `docs/qa/RESTORE_TEST.md` |
| Beban baca | 20×10 konkuren | 50/50 tanpa 500 |
| Beban tulis | `TestApproveConcurrent` | 8 goroutine → tepat 1 menang + 1 repertorium |
| E2E bisnis | smoke login→permohonan→akta→invoice | HIJAU (log audit) |
| UI sweep | `ui_audit.py` | load+console+forms+buttons+onclick+dropdown+tab+sidebar+overlay+mobile, 60+ rute |
| UI interaksi | `interact.py` 11/11 + `confirm.py` 3/3 | dropdown/tab/sidebar/nav-mobile; confirm dismiss-vs-accept |
| screenshot | `scripts/qa_shots/*.py` | 15+ halaman OK |
| race detector | `go test -race` | BACKLOG (host Windows/AV) |
| Fase-8 CRM enterprise | `clients/crm.go` + routes | pencarian lanjutan, statistik, catatan, inkonsisten beralasan, gabung snapshot+audit, ekspor CSV |
| Fase-8 DMS enterprise | `dms/enterprise.go` + routes | field kustom tervalidasi, pencarian lanjutan + pagination + ekspor CSV, aksi massal, banding versi, matriks approval (gate APPROVED), preview watermark + audit, kuota (507 jujur), ekspor ZIP, kerahasiaan di preview/unduh/bagi, retensi-notify |
| Fase-8 E2E | `cms.py` 24/24, `matrix.py` 8 peran × 26 rute, `fase1a/b`, `fase2`, `fase3` 14/14, `wizard.py` 5/5, `merge.py`, `interact.py` 11/11, `confirm.py` 3/3, `dms_e2e.py` | SEMUA HIJAU 2026-11-22 |
| Fase-8 integrasi MySQL | `go test ./tests/` (biner AV-safe) | PASS penuh (termasuk TestClientsDuplicates + TestRemindersRun pasca-perbaikan zona waktu) |
| Bug nyata diperbaiki | jadwal UTC-vs-WIB; template CMS `.P.Robots`/`.A.Robots`; map-vs-struct di dasbor DMS; scan kolom pendWF | dengan tes sebagai bukti |
| restore VPS | — | BACKLOG (B-09) |
| pen-test | — | BACKLOG (butuh pihak ketiga) |
| CMS growth 14 ide | cms_growth.py SEMUA HIJAU | kalkulator, konsultasi, booking->janji temu, tim/galeri/testimoni terverifikasi, i18n, cari, arsip, preview HMAC, versi+rollback, kualitas, statistik, ekspor ZIP, banner, maintenance 503 |
