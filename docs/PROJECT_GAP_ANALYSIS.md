# PROJECT GAP ANALYSIS — sisa jujur (bukan janji)

P0 (fungsional, sebelum klaim produksi):
- Uji restore backup: TERBUKTI (`docs/qa/RESTORE_TEST.md` — dump+restore 0 selisih, RTO <60 dtk). Uji di VPS produksi tetap wajib (B-09).
- Matriks izin otomatis 9 peran × rute kunci: TERBUKTI (`scripts/qa_shots/matrix.py` → `docs/qa/PERMISSION_MATRIX.md`, 120 sel cocok). Menangkap 5 celah baca (deeds/repertorium/protocols/templates/tasks/tariffs) — DIPERBAIKI.
- Uji beban: asap konkuren 20×10 baca → 50/50 tanpa 500 (`LOAD2`). Belum: Approve konkuren + OCR queue + FTS data besar (butuh harness khusus).
- Klik-penuh browser: INTERACTION_TESTS 1-12 TERBUKTI (`scripts/qa_shots/interactions.py`).

P1 (produk):
- Notifikasi SLA/eskalasi workflow: TERBUKTI (`remind.RunOnce` → `TestRemindersSLA`; permohonan SUBMITTED/IN_REVIEW/NEEDS_DOCS macet >3 hari dieskalasi, anti-spam 24 jam).
- Unduh akta langsung dari portal klien: TERBUKTI (kartu "Akta Saya" + Unduh PDF, otorisasi via canViewDeed server-side).
- Rich-text editor akta (kini: textarea + sisip klausul — jujur, bukan klaim editor).
- Pratinjau DOCX/XLSX (kini: tolak jujur + sarankan unduh).
- Penggabungan metadata duplikat (kini: tautan relasi manual).

P2 (skala):
- Engine search khusus (kini: MySQL FULLTEXT + LIKE fallback — cukup untuk skala kantor).
- S3 backend (interface siap; kini: local only).
- Malware scanner hook (kini: MIME+magic-byte sniff; interface adapter belum).

BLOCKED eksternal:
- Tinjauan notaris atas alur hukum (B-06) — wajib sebelum data sungguhan.
- TLS publik + COOKIE_SECURE=true di produksi.
- AV host memblokir *.test.exe (B-04) — workaround rename; CI Linux disarankan.
