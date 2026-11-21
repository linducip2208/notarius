# White Label Guide — branding per kantor (standalone)

Setiap instalasi milik satu kantor. Semua identitas/branding tersimpan di database
(`offices` + `office_branding`), bukan di kode. Ubah via **Website → Branding**
(butuh izin `website.branding.manage`, default: office_admin).

## Yang bisa diubah
- Nama aplikasi, singkatan, tagline; logo, favicon, latar login (via Media Library → salin `ref`, mis. `cms/2026/11/ab12.png`, tempel ke kolom logo).
- Warna utama/sekunder/aksen/sidebar/header (validasi HEX; input jahat dinormalisasi, bukan error misterius).
- Gaya tombol (kotak/membulat/pil), radius 0–24, font sistem/serif/mono (lokal, tanpa CDN).
- Teks footer & hak cipta, bahasa, zona waktu, format tanggal, mata uang.

## Di mana branding tampil
- Login (logo + nama + tagline), sidebar + nama aplikasi, judul tab browser, favicon.
- Portal klien, halaman verifikasi publik (`/verify`), kop kuitansi/invoice.
- Website publik: warna, font, nama, logo, footer.

## Pratinjau jujur
Halaman Branding menampilkan pratinjau langsung; **tidak ada yang berubah di produksi
sebelum tombol Simpan ditekan**. Setelah simpan, cache website dihapus otomatis.

## Fallback
Kolom kosong/tak valid → default SaktiNotaris + biru `#1f6feb`. Tidak pernah CSS rusak
(warna non-HEX ditolak-normalisasi; radius dijepit; font tak dikenal → sistem).

## Batas
- Login memakai brand kantor pertama instalasi (belum ada konteks kantor pra-login).
- 1 baris branding per kantor (`office_id` PRIMARY KEY).
