# UI/UX BACKLOG — sisa jujur (lanjutkan dari sini tanpa mengulang)

P0 (fungsional):
- [ ] Jalankan INTERACTION_TESTS 1-12 di browser sungguhan, isi hasil.
- [ ] Isi VISUAL_AUDIT dari screenshot uiux/ (skor per kategori §16).
- [ ] Uji matrix peran × sidebar gating (9 peran, bukan hanya admin).
- [ ] Alur komparasi akta + sisip klausul klik-penuh.

P1 (pengalaman):
- [x] Badge angka sidebar: kini hanya Permohonan; tambah tugas terbuka bila relevan. → SELESAI v0.5.14: `PendingTasks` (COUNT OPEN) di `render()` + badge di Tugas (gated: halaman /tasks tetap PermViewClient).
- [x] Tab client_detail: ingat tab aktif setelah submit (kini kembali ke tab pertama). → SELESAI v0.5.15: `ui.js` simpan tab aktif per-path di sessionStorage + restore saat load (berlaku semua halaman bertab, tanpa ubah server).
- [ ] Grafik: tooltip titik sudah ada (title); tambah label nilai bila diminta.
- [ ] Tab client_detail: ingat tab aktif setelah submit (kini kembali ke tab pertama).
- [ ] Waarmeking terpisah, Dokumen Masuk/Keluar, Estimasi, Laporan khusus (atau putuskan: cakupan kini cukup).

P2 (polish):
- [ ] Avatar foto profil (kini inisial); upload + crop bila diminta.
- [ ] Mode ringkas sidebar (collapse) — kini drawer responsif saja.
- [ ] Audit pembaca layar + keyboard penuh.

BLOCKED:
- Screenshot lab perangkat fisik (hanya Chromium headless tersedia).
