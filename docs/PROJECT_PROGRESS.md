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
- Test: integrasi 21/21 (+SLA), unit hijau (rename-AV), E2E 10+14+7+14+120sel+13, screenshot Chromium 20+ halaman.
- Fase 4: matriks izin 9 peran × 15 rute OTOMATIS (120 sel cocok; 6 celah baca DITUTUP: deeds/repertorium/protocols/templates/tasks/tariffs), SLA eskalasi macet >3 hari + test, portal "Akta Saya" + Unduh PDF (terbukti), INTERACTION 1-12 OTOMATIS 13/13, restore backup TERBUKTI 0 selisih (RTO <60 dtk), beban baca 20×10 = 50/50.
- Scorecard bukti 88-97 (rata-rata ±94).

Blokir eksternal: tinjauan notaris, TLS produksi, uji restore VPS, pen-test, lab perangkat fisik.
