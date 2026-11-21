# UI E2E Test Report — interaksi (Playwright, Chromium)

Tanggal: 2026-11-22. Tooling repo: `scripts/qa_shots/*.py` (tidak tambah dependensi browser).

| Skrip | Cakupan | Hasil |
|---|---|---|
| `interact.py` (11 cek) | login, 3 dropdown (buka/item/Escape/klik-luar), user dropdown, 5 tab + link aktif, sidebar mobile, h-scroll 390px, nav publik mobile | 11/11 PASS |
| `confirm.py` (3 cek) | dismiss = tanpa aksi; create + accept = terhapus + flash | 3/3 PASS |
| `ui_audit.py` (sweep 60+ rute) | load, console, network, forms, buttons, onclick-ref, dropdown, tab, sidebar, overlay, mobile | 190/190 PASS |
| `cms.py` (25 cek, termasuk pratinjau/branding) | — | 25/25 PASS |
| `interactions.py` (14 cek, forms/revisi/tab/filter/CSV/logout) | — | 14/14 PASS |

Aturan main: dialog `confirm()` di-accept/dismiss eksplisit per kasus uji
(`pg.on("dialog")` / `pg.once("dialog")`) — tidak pernah mengandalkan default.
Rate limiter login (10/mnt) kadang menjegal rangkaian uji paralel; solusinya
jeda antar skrip, BUKAN mematikan limiter — limiter terbukti aktif (= PASS keamanan).

Kegagalan berbasis browser yang ditemukan & diperbaiki via E2E:
1. Dropdown Bootstrap mati total (bundle Tabler tak sediakan `window.bootstrap`) → `web/assets/notarius/ui.js` + E2E `interact.py`.
2. `type` hilang pada 2 tombol dropdown (+Buat, +Aksi Cepat) → ditambah.
3. Rute `/tentang-kami` dirujuk nav+sitemap tapi belum terdaftar → didaftarkan; halaman absen → 404 kustom (diuji).
4. `BERANDA CMS` — verifikasi render konten + font tanpa ZgotmplZ.
