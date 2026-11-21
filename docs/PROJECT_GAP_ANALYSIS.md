# PROJECT GAP ANALYSIS — sisa jujur (bukan janji)

P0 (fungsional, sebelum klaim produksi):
- Uji restore backup: TERBUKTI (`docs/qa/RESTORE_TEST.md` — dump+restore 0 selisih, RTO <60 dtk). Uji di VPS produksi tetap wajib (B-09).
- Matriks izin otomatis 9 peran × rute kunci: TERBUKTI 208 sel (`matrix.py` → `docs/qa/PERMISSION_MATRIX.md`, termasuk 13 rute CMS). Menangkap 6 celah baca + 3 sel CMS — DIPERBAIKI.
- Uji beban: baca 50/50 + tulis konkuren TERBUKTI (`TestApproveConcurrent`); CMS realistis TERBUKTI (`TestCMSPerf`: 100 hal + 500 art + 1000 media, baca <500ms). Belum: OCR queue + FTS data besar.
- Klik-penuh browser: INTERACTION_TESTS 1-12 TERBUKTI 14/14 (`interactions.py`) + a11y landmark.
- Website CMS E2E 25/25 TERBUKTI (`scripts/qa_shots/cms.py`): branding, SEO/domain+validasi, halaman+section+terbit+render publik+sanitasi, beranda ganti landing, artikel, FAQ, nav, media+re-encode+tolak, sitemap CMS, riwayat, setup contoh DRAF.

P1 (produk):
- Notifikasi SLA/eskalasi workflow: TERBUKTI (`remind.RunOnce` → `TestRemindersSLA`; permohonan SUBMITTED/IN_REVIEW/NEEDS_DOCS macet >3 hari dieskalasi, anti-spam 24 jam).
- Unduh akta langsung dari portal klien: TERBUKTI (kartu "Akta Saya" + Unduh PDF, otorisasi via canViewDeed server-side).
- Rich-text editor akta: TERBUKTI (`internal/safehtml` allowlist + unit; toolbar B/I/U/list/H + sanitasi server write+render; PDF/DOCX/diff pakai teks bersih; E2E fase2+interactions revisi via contenteditable).
- Pratinjau DOCX/XLSX: TERBUKTI (`internal/officex` ekstraksi teks 50KB + unit docx/xlsx/tolak; iframe pratinjau; file >8MB/terenkripsi ditolak jujur).
- Penggabungan metadata duplikat: TERBUKTI (`MergeDuplicates` + migrasi 014 ENUM file + E2E merge.py 8 berkas → kanonis; tanpa hapus byte + banner DIGABUNG_KE).
- Wizard korporat per-jenis: TERBUKTI (`corp.WizardProgress` murni + unit PT/CV/warning; halaman `/requests/{id}/wizard` 5 langkah + progress + peringatan 25%/issued>dasar; E2E wizard.py PT 5/5).
- Gateway notifikasi nyata: TERBUKTI (`internal/deliver`: Webhook HMAC + SMTP env + FromEnv; runReminders handoff best-effort; unit lawan httptest; tanpa env = in-app only).

P2 (skala):
- Engine search khusus (kini: MySQL FULLTEXT + LIKE fallback — cukup untuk skala kantor).
- S3 backend: TERBUKTI (`internal/storage/s3.go` SigV4 PUT/GET + unit lawan httptest in-memory; aktif via NOTARYOS_S3_* env, default local).
- Malware scanner: TERBUKTI (`internal/malware` klien clamd INSTREAM + unit lawan server palsu; tombol Pindai on-demand di detail dokumen + audit; tanpa daemon dilaporkan tidak-tersedia, bukan bersih).

BLOCKED eksternal:
- Tinjauan notaris atas alur hukum (B-06) — wajib sebelum data sungguhan.
- TLS publik + COOKIE_SECURE=true di produksi.
- AV host memblokir *.test.exe (B-04) — workaround rename; CI Linux disarankan.
- `go test -race` butuh GCC (tak ada di host ini) — pindah ke CI Linux.

SELESAI SESI INI (White Label + CMS, standalone per kantor):
- Branding DB + tema + pratinjau + aplikasi global (admin/portal/kuitansi/web).
- CMS: halaman+section builder, artikel+kategori, layanan info, FAQ, nav, media library aman, SEO+domain, workflow+histori+audit, setup wizard + contoh DRAF.
- Publik: /, /tentang-kami, /kontak, /p/{slug}, /artikel(+slug), /faq, /layanan-kami(+slug), /media, sitemap dinamis — hanya PUBLISHED.
