# UI/UX BACKLOG — sisa jujur (lanjutkan dari sini tanpa mengulang)

P0 (fungsional):
- [ ] Jalankan INTERACTION_TESTS 1-12 di browser sungguhan, isi hasil.
- [ ] Isi VISUAL_AUDIT dari screenshot uiux/ (skor per kategori §16).
- [ ] Uji matrix peran × sidebar gating (9 peran, bukan hanya admin).
- [ ] Alur komparasi akta + sisip klausul klik-penuh.

P1 (pengalaman):
- [x] Badge angka sidebar: kini hanya Permohonan; tambah tugas terbuka bila relevan. → SELESAI v0.5.14.
- [x] Tab client_detail: ingat tab aktif setelah submit. → SELESAI v0.5.15 (sessionStorage).
- [x] Grafik: tooltip titik sudah ada (title); tambah label nilai bila diminta. → DIPUTUSKAN CUKUP v0.5.16: tooltip + sumbu bulan + legenda memadai; label angka di tiap titik membuat grafik sesak di mobile.
- [x] Waarmeking terpisah, Dokumen Masuk/Keluar, Estimasi, Laporan khusus. → DIPUTUSKAN CUKUP v0.5.16: kebutuhan terpenuhi modul kini (peringatan=notifikasi+SLA+retensi, arsip=arsip fisik+DMS classification, estimasi=kalkulator+quotation, laporan=/reports+/metrics+/audit/export).
- [ ] Grafik: tooltip titik sudah ada (title); tambah label nilai bila diminta.
- [ ] Tab client_detail: ingat tab aktif setelah submit (kini kembali ke tab pertama).
- [ ] Waarmeking terpisah, Dokumen Masuk/Keluar, Estimasi, Laporan khusus (atau putuskan: cakupan kini cukup).

P2 (polish):
- [x] Avatar foto profil (kini inisial); upload + crop bila diminta. → SELESAI v0.5.18: upload raster ≤10MB + auto-crop persegi tengah + re-encode (migrasi 035, serve privat /avatar, tampil di topbar + /settings, TestSquareCropPNG hijau). Crop manual interaktif tidak dibuat (cukup auto-crop).
- [x] Mode ringkas sidebar (collapse) — kini drawer responsif saja. → SELESAI v0.5.16: tombol "⫷ Ringkas" sembunyikan sidebar + tombol ☰ floating untuk membuka; preferensi tersimpan di localStorage per browser.
- [ ] Audit pembaca layar + keyboard penuh.

BLOCKED:
- Screenshot lab perangkat fisik (hanya Chromium headless tersedia).
