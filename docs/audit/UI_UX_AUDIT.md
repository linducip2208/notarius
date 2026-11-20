# UI_UX_AUDIT — Tabler kini vs Metronic target

## Temuan P0 (diperbaiki sebagian)
1. Dashboard CSRF hilang → DIPERBAIKI (csrf ditambah).
2. Sidebar tanpa grup/izin → DIPERBAIKI SEBAGIAN (grup Workspace/Klien/Layanan/Register/Keuangan + `can` gating; ikon+active-state + badge count menyusul Metronic).
3. Tombol tanpa konfirmasi (Sahkan/Verifikasi/Serahkan/Terbitkan) → DIPERBAIKI SEBAGIAN (confirm pada Sahkan akta; sisanya menyusul).
4. Tanpa halaman detail akta → DIPERBAIKI (deed_detail: body+versi+review/approve+PDF/DOCX).
5. Tanpa Document Center → DIPERBAIKI (documents: search+unduh otorisasi).
6. Tanpa 403/404 → DIPERBAIKI (forbidden/notfound + pesan minta akses).
7. Tabel 9-kolom + form 5-kolom hancur di mobile → BELUM (butuh drawer/detail + kolom prioritas; tunggu Metronic).
8. CDN tanpa SRI/fallback → BELUM (vendor lokal menunggu keputusan; intranet berisiko).

## Standar Metronic yang diminta (belum mulai — BLOCKED aset)
App shell, grouped aside+ikon, header+user-menu+notif+search, breadcrumb+actions, cards, stat widgets, DataTables, dropdown/modal/tabs/accordion/stepper/timeline/toast, validasi inline, datepicker, pagination, login/forgot/reset, error/empty/loading/denied, mobile responsif, varian layout per kebutuhan (dashboard, detail klien, editor akta, register, keuangan, laporan).

## Keputusan jujur
- Tidak ada campuran Tabler+Metronic. Selama aset resmi tidak ada, UI tetap Tabler penuh.
- `web/assets/metronic/README.md` menjelaskan penempatan aset + AC migrasi.
- Klaim migrasi selesai DILARANG sampai semua halaman Metronic lokal + Tabler dihapus + regression visual lolos.
