# NOTARIUS Design System — 100% Tabler (keputusan pemilik)

## Keputusan
Design system final = **Tabler murni** (di-vendor lokal di `web/assets/vendor/`:
`tabler.min.css`, `tabler.min.js`, `htmx.min.js`). Tidak ada CDN, tidak ada
stylesheet kustom kedua, tidak ada Metronic. Skeleton Metronic yang lama
telah dihapus dari repo.

## Komponen (kelas Tabler bawaan)
- Shell: `page` + `navbar-vertical` (`data-bs-theme="dark"`) + `page-wrapper` +
  `header.navbar` (topbar: search, tombol tugas/notifikasi, `avatar` + dropdown).
- Navigasi: `navbar-nav`, `nav-link`, seksi berlabel, gating server-side via `can()`.
- Konten: `card`, `card-header/body/footer`, `table card-table table-vcenter`,
  `btn`/`btn-primary`/`btn-outline-primary`/`btn-warning`/`btn-ghost-secondary`,
  `badge bg-*-lt`, `alert`, `list-group`, `dropdown-menu`, `avatar`, `ribbon`,
  `breadcrumb`, `form-control/select`, `input-icon`, `display-*`, `page-title`.
- Utilitas: `d-flex/gap-*/rounded/shadow/border/bg-dark-light` + sedikit inline
  `style` untuk tinggi/object-fit gambar (bukan design system kedua).
- Print: kelas `d-print-none` Tabler (cetak repertorium/dokumen).

## Aturan pakai
1. Semua halaman WAJIB `head/page_open/page_close` — dilarang layout sendiri.
2. DILARANG menambah stylesheet kustom atau referensi CDN baru.
3. Status tampil Bahasa Indonesia, bukan enum mentah.
4. Tombol final/destruktif WAJIB `confirm()` + audit trail.
5. Setiap tabel WAJIB empty-state; setiap form WAJIB pesan error jujur.
