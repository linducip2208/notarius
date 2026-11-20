# RESPONSIVE GUIDELINES

- Navigasi: `navbar-expand-lg` — sidebar jadi drawer via toggler di <992px.
- Grid: `col-sm-6 col-lg*` — kartu 2-per-baris di tablet, penuh di desktop.
- Tabel: selalu dalam `table-responsive` (scroll horizontal, bukan layout hancur).
- Form: `col-md-*` runtuh ke 1 kolom di mobile; grup tombol `flex-wrap`.
- Kanvas dokumen & gambar: `max-width` + `overflow:auto` pada `pre`.
- Target uji: 1920/1600/1440/1366/1280, 1024/768, 430/390/375 (lihat qa/RESPONSIVE_TESTS.md).
- Status: verifikasi browser fisik/perangkat BELUM dilakukan (tanpa lab perangkat) —
  bukti kini: screenshot Chromium headless 1440 + 390 (docs/screenshots/uiux/).
