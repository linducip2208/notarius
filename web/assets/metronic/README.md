# Metronic Integration — STATUS: BLOCKED (aset resmi belum tersedia)

Repo ini BELUM berisi aset Metronic (CSS/JS/media berlisensi).
Jangan mengunduh salinan bajakan. Struktur di bawah disiapkan agar
migrasi Tabler → Metronic dapat dilakukan segera setelah lisensi resmi ada.

## Direktori yang diminta

```
web/assets/metronic/css/     ← style.bundle.css, plugins.bundle.css (resmi)
web/assets/metronic/js/      ← scripts.bundle.js, plugins.bundle.js (resmi)
web/assets/metronic/media/   ← logo, ilustrasi, ikon (resmi)
internal/app/templates/metronic/ ← layout *.gohtml Metronic (dibangun setelah aset ada)
```

## Cara menempatkan aset resmi

1. Beli/unduh Metronic dari Keenthemes dengan lisensi yang sah.
2. Salin file produksi (bukan source penuh) ke folder di atas.
3. Catat versi + hash di `web/assets/metronic/VERSIONS.md`.
4. Jalankan `go build ./...` dan uji visual semua halaman.

## Sementara itu (jujur)

- UI aktif tetap Tabler via CDN (lihat `internal/app/templates/partials.gohtml`).
- Tidak ada campuran Tabler+Metronic di satu halaman final.
- Migrasi TIDAK diklaim selesai sampai semua halaman memakai Metronic lokal
  dan referensi Tabler dihapus.

## Acceptance migrasi

- [ ] Aset lokal terverifikasi (versi + lisensi).
- [ ] 20 layout Metronic (shell, sidebar, header, dashboard, tabel, form, modal, timeline, login, error, empty, loading, denied, mobile).
- [ ] Tidak ada `cdn.jsdelivr.net/tabler` tersisa.
- [ ] Regression visual desktop/tablet/mobile lolos.
