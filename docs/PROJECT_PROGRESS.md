# PROJECT PROGRESS — 2026-11-21 (continuous execution)

Selesai:
- Audit repo penuh + 4 laporan subagent (backend, DB, security, UI).
- Bug kritis DIPERBAIKI + terbukti via test: OCR persist+timeout, PDF multipage+test, CSRF multipart (upload 100% gagal — TERTANGKAP E2E), DECIMAL scan (daftar layanan kosong diam-diam + kuitansi 404 + 3 halaman), repertorium 5-vs-6 kolom (Approve tak pernah sukses), hapus-file Windows (handle terbuka), v1 tak tersnapshot.
- 14 audit trail + 6 celah izin ditutup (isolasi kantor, SoD, portal terisolasi).
- DMS enterprise: migrasi 011 (16 tabel + kolom), store, 25 rute, 13 template, auto-enqueue OCR + auto-relasi, FTS+fallback, share expiring publik, arsip+QR label cetak, retensi+hold blokir-hancur, duplikat SHA-256, checklist template+auto+progres+notif.
- Fase 1 jualan: kuitansi QR HMAC + /verify publik + tolak palsu (6/6), portal bayar + bukti (8/8), pengingat anti-spam.
- Fase 2: diff LCS+test, checklist cerdas, label QR arsip (7/7).
- Fase 3: TTE internal (approve→segel HMAC→/verify akta→sign hash-64, 13/13), multi-kantor ENTERPRISE + acting HMAC-audit, tier LITE/PRO/ENTERPRISE + gate 402 jujur, harga konsultasi tanpa angka palsu.
- E2E hardening: confirm() JS auto-dismiss Playwright menutupi submit gagal → semua skrip wajib dialog-accept + asersi flash-URL/DB (bukan teks statis); tangkap duplikat nomor akta via UNIQUE nyata.
- Test: integrasi 24/24 (+SLA +approve-konkuren +CMS lifecycle +CMS perf 100/500/1000), unit hijau 11 paket (rename-AV), E2E 10+6+8+8+14+14+5+5+25+208sel.
- Fase 4: matriks izin 8-9 peran × 26 rute OTOMATIS (208 sel cocok; 9 celah baca DITUTUP), SLA eskalasi + test, portal "Akta Saya" + Unduh PDF (terbukti), INTERACTION 1-12 OTOMATIS 14/14, restore backup TERBUKTI 0 selisih (RTO <60 dtk), beban baca 50/50 + tulis konkuren.
- Fase 5 (semua sisa P1/P2): wizard korporat 5 langkah + unit (PT/CV/warning), gateway notifikasi webhook-HMAC + SMTP + FromEnv + unit, pratinjau teks DOCX/XLSX + unit, gabung duplikat + migrasi 014 + E2E 5/5, pemindai ClamAV on-demand + unit, backend S3 SigV4 + unit, RTE tersanitasi (write+render) + unit + E2E, uji tulis konkuren, a11y landmark (skip link/main/aria).
- Fase 6 (perintah White Label + CMS, 20 sub-fase): audit MASTER_IMPLEMENTATION_AUDIT; migrasi 015 (office_branding + 11 tabel website_*, idempoten); branding DB + pratinjau + aplikasi global (admin/portal/kuitansi/web); CMS CRUD + section builder + artikel/kategori + layanan + FAQ + nav; media library aman + thumb; SEO + domain jujur; workflow + histori + audit; setup wizard + contoh DRAF; renderer publik 8 rute non-Tabler + sitemap dinamis; cache + invalidasi; rate limit upload; 14 izin website.* + matriks 208; 12 dokumen; E2E CMS 25/25; perf realistis di DB uji.
- Fase 7 (perintah UI Interaction Audit & Repair): inventaris 136 endpoint + 12 toggle + 13 form-file + 9 tabel-file; 4 bug nyata DIPERBAIKI (dropdown/tab/collapse mati — bundle Tabler tanpa bootstrap global → ui.js; 2 tombol tanpa type; rute /tentang-kami hilang; include htmx mati dibuang); sweep 189/189 PASS; interact 11/11; confirm dismiss-vs-accept PASS; regresi penuh hijau.
- Scorecard bukti 88-97 (rata-rata ±94).

Blokir eksternal: tinjauan notaris, TLS produksi, uji restore VPS, pen-test, lab perangkat fisik.
