# Homepage Builder Guide — 11 section + validasi

Section ditambah dari halaman edit (`/cms/pages/{id}`): pilih tipe → isi → Simpan.
Urutan = kolom position (kecil dulu). Toggle tampil/sembunyi tanpa hapus.

## 11 tipe & perilakunya
| Tipe | Render |
|---|---|
| hero | headline besar + sub + CTA + konten |
| intro, text, info | teks standar (align/background/layout variatif) |
| profile | seperti intro (isi profil manual — JANGAN mengarang gelar) |
| services | daftar layanan info (link ke `/layanan-kami/{slug}` manual via CTA) |
| process | langkah alur (tulis manual sebagai list) |
| articles | heading + daftar artikel terbit otomatis |
| faq | daftar FAQ aktif otomatis |
| contact | info kontak kantor otomatis dari profil |
| map | placeholder peta (embed manual via konten; tanpa API key) |
| footer | teks footer |
| gallery | grid gambar (pakai ID media) |
| cta | panel ajakan + tombol |

## Background / align / layout
- background: light/white/dark/primary (di luar itu → light).
- align: left/center/right. layout: standard/wide/split.
- settings JSON opsional: `{"columns":1-3,"ratio":"16:9|4:3|1:1","theme":"auto|light|dark"}`
  — kunci lain dibuang diam-diam (bukan error).

## Gambar section
Isi **ID media** (angka dari Media Library). Render via `/media/{id}` (publik, cache).
Gambar rusak/hilang → section tetap tampil tanpa gambar (tidak 500).

## Keamanan
Konten section disanitasi server (allowlist p/b/i/list/heading/link/gambar;
`javascript:`/event handler dibuang). Iframe & script tidak didukung — sengaja.
