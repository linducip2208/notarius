# Branding Guide — dokumen & portal (konsisten, tak ubah isi hukum)

## Prinsip
Branding (logo/nama/warna) menempel pada **bingkai** dokumen, bukan isi hukum.
Isi akta/minuta tidak pernah diubah otomatis oleh branding.

## Cakupan
- Kop kuitansi/invoice: `{{.Brand.AppName}}` + nomor + QR verifikasi (template `billing.gohtml`).
- Judul tab + favicon: nama aplikasi + favicon kantor (atau default).
- Login: logo + nama + tagline kantor instalasi (brand kantor pertama).
- Portal klien & halaman verifikasi publik: nama aplikasi kantor.
- Website publik: tema penuh (warna/font/logo/footer).

## Menyiapkan aset
1. Unggah logo PNG (kotak, ≥256px) + favicon PNG ke **Media Library**.
2. Salin `ref` (cth. `cms/2026/11/ab12cd34.png`) ke kolom Logo/Favicon di **Branding**.
3. Simpan → verifikasi di halaman Kuitansi, `/verify`, dan website.

## Fallback & validasi
- Ref kosong/salah → logo bawaan; warna invalid → default (tidak pernah layout rusak).
- Favicon non-gambar → browser mengabaikan (tidak error).
