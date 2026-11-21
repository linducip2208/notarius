# UI Interaction Audit — NOTARIUS (otomatis + manual)

Metode: `scripts/qa_shots/ui_audit.py` (Playwright: console/network per halaman,
klik dropdown/tab/sidebar asli, audit semua form/button/onclick/overlay/mobile)
+ inspeksi statis template. Bukti mentah: `docs/qa/UI_AUDIT_RAW.md`.

## Inventaris stack
- Tabler v1.7.5 (bundle Bootstrap 5 di `tabler.min.js` — SATU lib, tanpa Bootstrap ganda).
- HTMX 1.x lokal (`htmx.min.js`) — TERDAFTAR 0 pemakaian `hx-*`/`htmx.` (keputusan: hapus include agar tak menyesatkan).
- Komponen Bootstrap dipakai: collapse ×2 (navbar landing, sidebar), dropdown ×4 (+Buat, notif, user, Aksi Cepat), tab ×5 (detail klien).
- TANPA modal/tooltip/popover di template mana pun (konfirmasi = `data-confirm` + native `confirm()`, di-dismiss otomatis oleh Playwright → semua submit E2E memakai dialog-accept).
- 13 file template berisi `<form>`, 9 berisi `<table>`. Semua POST di grup CSRF.

## Cakupan sweep
60 rute (15 publik + 45 login) × {load, console, forms, buttons, onclick-ref} +
dropdown/tab/sidebar/overlay/mobile khusus (lihat RAW).

## Hasil ringkas (biner final, 2026-11-22)
Sweep `ui_audit.py`: **189 pass, 0 fail** — 61 rute × {load, console, forms, buttons,
onclick-ref} + dropdown/tab/sidebar/overlay/mobile + 404 kustom.
Bukti mentah: `docs/qa/UI_AUDIT_RAW.md` (192 baris, nol baris FAIL).
