# CMS Security — kontrol yang diuji, bukan klaim

## Otorisasi (server-side, diuji matriks)
- Semua rute `/cms/*` di belakang `RequireAuth` + `requireWeb(perm)` spesifik.
  Menu disembunyikan via `can`, TAPI keputusan selalu di handler (menyembunyikan
  menu bukan pengamanan — dibuktikan `matrix.py`: 403 untuk peran tanpa izin).
- Izin baru mengikuti konvensi `website.*` (14 izin). Tidak ada role baru.
- Publik (`/`, `/artikel`, `/media`, …) tanpa login, query HANYA `status='PUBLISHED'`
  (diuji `TestCMSLifecycle`: DRAF → 404 publik).

## CSRF & XSS
- Semua POST CMS di grup CSRF middleware (token per render).
- Konten halaman/artikel/section/FAQ: `cms.SanitizeHTML` saat SIMPAN + `cmsHTML`
  (sanitasi ulang) saat render. Allowlist: p/br/b/i/u/list/h2-h4/blockquote/a/img.
  Atribut/event handler dibuang; `javascript:`/`data:`/`vbscript:` ditolak;
  `</a>` tanpa pasangan dibuang (unit `TestSanitize`).
- Branding: warna HEX tervalidasi, font dari allowlist map (tanpa CDN), radius dijepit —
  tidak ada CSS injection (`TestBrandingValidation`).

## Upload (diuji)
- Allowlist ekstensi + sniff MIME + kecocokan ekstensi↔MIME + cap 10 MB.
- Raster di-decode + encode ulang (buang EXIF/metadata); SVG aktif ditolak.
- Nama acak (`cms/tahun/bulan/hex.ext`); traversal `..` ditolak; serve by-ID.
- Hapus ditolak bila dipakai konten (`website_media_usage`).
- Rate limit 20 unggahan/menit.

## Isolasi file
- Media CMS (publik) vs DMS privat (otorisasi per unduhan): store terpisah,
  tidak ada URL publik ke dokumen akta/klien. `/media/{id}` hanya membaca
  `website_media`; ID DMS tidak routable ke sana.

## IDOR & traversal
- Semua baca CMS admin memfilter `office_id`; ID lintas kantor → 404, bukan bocor.
- `parent_id` nav dan `page_id` section divalidasi sekantor.

## Host & redirect
- Tidak ada redirect terbuka dari input (CTA/nav hanya relatif + http(s) valid).
- Host aneh tidak menentukan canonical dan tidak meracuni cache.

## Audit & secret
- Audit: branding/site/domain save, page/article create/update/duplicate/delete,
  media upload/delete, setup — aktor + detail + IP (tanpa password/token/secret).
- Riwayat publikasi terpisah per aksi (publish/unpublish/duplicate/sample).
- Tidak ada kredensial di kode; kunci hanya via env (NOTARYOS_*).

## Sisa risiko (jujur)
- Pen-test independen belum dilakukan; screen-reader audit penuh belum.
- ClamAV opsional: tanpa daemon, unggahan CMS mengandalkan validasi di atas.
- TLS/pen-test produksi = gate eksternal (B-08, pen-test).
