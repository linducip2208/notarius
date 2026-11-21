# CMS Test Report — bukti, bukan klaim

Tanggal: 2026-11-21. Lingkungan: Windows dev, MySQL 8.4 lokal, Tesseract aktif.

## Unit (`internal/cms`, AV-safe via `go test -c`)
- TestBrandingValidation, TestNavURL, TestSlug, TestSectionSchema,
  TestSanitize, TestMediaValidate — **PASS**.

## Integrasi (`TestCMSLifecycle`, DB `notaryos_test`)
Branding default + tolak nama kosong; slug ganda ditolak; terbit tanpa isi ditolak;
section jahat (tipe/CTA) ditolak; publish→publik tampil; jadwal masa depan tidak
terbit, lewat → terbit; riwayat tercatat; DRAF → 404 publik — **PASS** (23/23 suite).

## E2E browser (`scripts/qa_shots/cms.py`, 25/25 PASS)
Dashboard, branding simpan+tampil (warna `#0a7d44` tampil di admin), SEO, domain
(valid + tolak `javascript:`), halaman terbit + section CTA, render publik +
`<script>` hilang, daftar, beranda CMS ganti landing (konteks anonim), artikel
terbit+tampil, FAQ, nav, media PNG (re-encode; ref tampil; `.py` ditolak),
sitemap berisi URL CMS, riwayat, setup + contoh = DRAF.

## Keamanan E2E
- XSS `<script>` pada konten halaman → dibuang saat render publik.
- `javascript:` pada domain & CTA → ditolak server (400/validasi).
- Draf/review/scheduled/archived → tidak di sitemap, tidak dirender publik.
- Upload `.py` → ditolak (ekstensi).

## Belum diuji (jujur)
- Beban: 100 halaman/500 artikel/1000 media (butuh generator dataset).
- `go test -race` (tanpa GCC di host ini; sarankan CI Linux).
- Screen-reader penuh; pen-test independen.
