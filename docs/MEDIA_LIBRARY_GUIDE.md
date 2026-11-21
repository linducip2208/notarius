# Media Library Guide — publik & terpisah dari DMS

**Aturan utama:** media website PUBLIK (logo, foto, PDF brosur). Dokumen rahasia
TETAP di DMS privat (`/documents`), tidak pernah lewat sini. Storage terpisah:
`cms/{kantor}/{tahun}/{bulan}/{acak}.{ext}` vs berkas DMS.

## Upload (`/cms/media`, izin `website.media.manage`)
- Format: JPG/JPEG/PNG/GIF/WebP/SVG/PDF, 1 byte–10 MB.
- Validasi berlapis: ekstensi allowlist → sniff MIME → kecocokan ekstensi/MIME
  (PNG berlabel JPG ditolak) → SVG aktif (`<script`/`onload:`/`javascript:`) ditolak.
- Gambar raster (JPEG/PNG/GIF) di-**decode + encode ulang** (buang EXIF/metadata),
  dimensi dicatat. WebP/SVG/PDF lewat apa adanya (tervalidasi).
- Nama file acak aman; nama asli hanya label tampilan.
- Thumbnail ≤320px dibuat otomatis (`?thumb=1`), tanpa baris DB tambahan.

## Pakai & hapus
- Salin `ref` (cth. `cms/2026/11/ab12.png`) ke Branding / gambar section / cover / favicon.
- URL publik: `/media/{id}` (+ `?thumb=1`). Cache 1 hari. Tanpa login.
- Hapus **ditolak** bila media dipakai konten (tabel usage) — lepas dulu.
- Hapus menghapus byte + thumb (idempoten) + baris DB.

## Batas jujur
- Tanpa resize on-the-fly selain thumb 320px; responsive via CSS.
- Tanpa WebP otomatis (stdlib Go tak ada encoder; tanpa dep CDN).
- Rate limit unggah 20/menit/IP (anti-abuse).

## Keamanan terkait
Lihat `docs/CMS_SECURITY.md`: sniffing, spoofing, SVG, traversal (`..` ditolak),
otorisasi per kantor, publik-vs-privat.
