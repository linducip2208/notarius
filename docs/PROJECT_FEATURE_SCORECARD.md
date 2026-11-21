# PROJECT FEATURE SCORECARD — bukti, bukan klaim (2026-11-21)

Bobot: fungsional 30% · benar+workflow 20% · security/izin 20% · reliabilitas 10% · UI/UX 10% · test+dok 10%.

| Fitur | Skor | Bukti |
|---|---|---|
| Auth & session (+MFA/CSRF/rate-limit) | 97 | unit hijau, E2E login, audit login |
| RBAC 9/19 + matriks + isolasi kantor | 96 | 79 baris cermin, /roles, uji cross-office manual (matriks otomatis: backlog) |
| Klien/duplikat/identitas/BO | 96 | TestClientsDuplicates, verifikasi manusia UI |
| Permohonan workflow + checklist | 95 | TestRequestWorkflow, checklist per layanan |
| Korporat | 94 | integrasi BO+pengurus+modal; wizard per-jenis: backlog |
| Akta draf→review→sahkan→repertorium | 97 | TestDeedVersionLock, E2E approve, FOR UPDATE |
| Repertorium/protokol/legalisasi/wasiat | 96 | CRUD+advance+handover teruji |
| Template/klausul + engine + PDF/DOCX | 94 | PDF multipage test; DOCX valid; rich-text: backlog |
| OCR (Tesseract ind, queue, retry, FTS) | 93 | provider ready 5.4.0, RunOCR simpan+indeks; E2E OCR berjalan |
| AHU/OSS asistensi | 90 | checklist+bukti manual; tanpa API resmi (by design) |
| File privat + unduh otorisasi + MIME sniff | 95 | E2E upload/download, IDOR test |
| DMS (library/detail/preview/versi/workflow/share/arsip/retensi/duplikat/audit) | 94 | 11 halaman 200, E2E 10/10, integrasi 8/8; uji beban: backlog |
| Keuangan | 97 | multi-baris + anti-overpay (test) + kuitansi QR HMAC + verifikasi publik + portal bayar (E2E 14/14) |
| Portal klien terisolasi | 93 | server-side links; unduh akta portal: parsial |
| Dashboard/laporan | 94 | real-query; grafik+donat+feed |
| Deploy/backup/Docker | 88 | file lengkap; uji restore VPS: BELUM |
| UI/UX Tabler | 93 | screenshot Chromium 13 halaman; klik-penuh: parsial |
| QA otomatis | 90 | unit+integrasi+rename-AV; race/browser-penuh: backlog |

Rata-rata tertimbang: ±93. Target 98: butuh uji restore VPS, matriks izin otomatis, uji beban, klik-penuh browser.
