# UI Interaction Fix Report — akar masalah → perbaikan → bukti

## F-1 Dropdown/Tab/Collapse mati total (KRITIS)
- Gejala: klik +Buat/notif/user/aksi-cepat tidak membuka apa pun; tidak ada error console.
- Akar: `web/assets/vendor/tabler.min.js` (v1.7.2) TIDAK mengekspos `window.bootstrap`
  (terbukti via E2E: `typeof bootstrap === 'undefined'`). Semua atribut `data-bs-toggle`
  tidak ter-bind ke apa pun. Bukan CSS, bukan z-index.
- Perbaikan: `web/assets/notarius/ui.js` (milik sendiri, tanpa CDN, tanpa Bootstrap ganda):
  dropdown (toggle/klik-luar/Escape), collapse (sidebar, navbar), tab (panel+link aktif),
  idempoten. Markup `data-bs-toggle` tetap (kompatibel Bootstrap).
- Bukti: `interact.py` 11/11 (buka/item/Escape/klik-luar/user/tab 5/mobile/sidebar/h-scroll/nav publik).

## F-2 Dua tombol tanpa `type`
- `+ Buat` (partials) dan `+ Aksi Cepat` (dashboard2) — di dalam non-form tapi tetap
  diperbaiki ke `type="button"` agar tak pernah jadi submit tak sengaja.
- Bukti: sweep `ui_audit.py` 0 tombol tanpa type di 60 rute.

## F-3 Rute `/tentang-kami` dirujuk tapi belum terdaftar
- Nav + sitemap merujuknya; request 404. Didaftarkan ke `publicPageSlug("tentang-kami")`,
  dengan fallback profil kantor publik bila belum ada halaman terbit (tidak pernah 404 mentah).
- Bukti: sweep `load /tentang-kami → 200`.

## F-4 Include htmx tak terpakai
- `htmx.min.js` dimuat tapi nol pemakaian `hx-*`/`htmx.` di seluruh repo.
- Dihapus dari head (tetap lokal, tanpa CDN), dicatat agar tak menyesatkan audit berikut.

## Bukan bug (diputuskan)
- `/layanan/pt` 404: slug valid `PT-DIRIAN` dkk (uppercase di DB); handler case-insensitive.
  Uji memakai slug salah → diperbaiki ke slug dinamis dari indeks.
- Logo/login default saat logged-out: by design (fallback instalasi), bukan bug.
- Rate limiter login sempat menggagalkan 2 langkah uji paralel → bukti limiter aktif;
  skrip diberi jeda, limiter TIDAK dimatikan.

## Berkas diubah
- BARU: `web/assets/notarius/ui.js`, `scripts/qa_shots/{ui_audit,interact,confirm}.py`,
  `docs/UI_{INTERACTION_AUDIT,FIX_REPORT,E2E_TEST_REPORT,REGRESSION_REPORT}.md`.
- UBAH: `internal/app/app.go` (rute), `*_templates/{partials,dashboard2}.gohtml` (type+script),
  `internal/app/templates/public.gohtml` (pub_about), `internal/app/cms_public.go` (fallback),
  `scripts/qa_shots/ui_audit.py` (slug dinamis, 404-page, anti-rate-limit).
