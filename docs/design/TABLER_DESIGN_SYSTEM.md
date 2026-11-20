# TABLER DESIGN SYSTEM — NOTARIUS (100% Tabler, keputusan pemilik)

## Sumber
- Tabler v1 lokal: `web/assets/vendor/tabler.{min.css,min.js}`, `htmx.min.js`.
- Token khusus: `web/assets/notarius/tokens.css` — HANYA variabel warna CSS,
  bukan komponen. Komponen selalu kelas Tabler.
- Verifikasi nol-CDN: `grep -rn "cdn.jsdelivr\|unpkg" internal/app/templates/ internal/middleware/` kosong;
  CSP `default-src 'self'`.

## Aturan
1. Layout: `page` > `navbar-vertical[data-bs-theme=dark]` + `page-wrapper` >
   `header.navbar` (topbar) + `page-header` + `page-body>.container-xl`.
2. Navigasi: `navbar-nav/nav-link`, grup via `nav-link disabled text-uppercase small`,
   aktif via JS location-match (emas) di `page_close`.
3. Data: `card`, `table card-table table-vcenter`, `badge bg-{blue,yellow,green,red,azure}-lt`,
   `avatar`, `list-group`, `dropdown-menu`, `pagination`, `breadcrumb`, `ribbon`.
4. Form: `form-control/select`, `input-icon`, `form-check`, label + required + error jujur.
5. DILARANG: stylesheet kedua, CDN baru, warna literal di template (pakai token/kelas),
   tombol tanpa handler, halaman kosong, angka fabrikasi.
