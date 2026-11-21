# PROJECT GAP ANALYSIS — sisa jujur (bukan janji)

P0 (fungsional, sebelum klaim produksi):
- Uji restore backup: TERBUKTI (`docs/qa/RESTORE_TEST.md` — dump+restore 0 selisih, RTO <60 dtk). Uji di VPS produksi tetap wajib (B-09).
- Matriks izin otomatis 9 peran × rute kunci: TERBUKTI (`scripts/qa_shots/matrix.py` → `docs/qa/PERMISSION_MATRIX.md`, 120 sel cocok). Menangkap 5 celah baca (deeds/repertorium/protocols/templates/tasks/tariffs) — DIPERBAIKI.
- Uji beban: asap konkuren 20×10 baca → 50/50 tanpa 500 (`LOAD2`); tulis konkuren TERBUKTI (`TestApproveConcurrent` 8 goroutine → tepat 1 menang + 1 repertorium). Belum: OCR queue + FTS data besar (butuh harness khusus).
- Klik-penuh browser: INTERACTION_TESTS 1-12 TERBUKTI (`scripts/qa_shots/interactions.py`) + a11y landmark (skip link, main, aria).

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
