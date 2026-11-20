# FINAL_UI_UX_AUDIT — NOTARIUS (2026-11-21, pasca-implementasi)

## Verifikasi per acceptance
- [x] Navbar / hero / fitur / showcase / workflow / keamanan / harga / FAQ / CTA / footer / mobile / SEO / CTA fungsi.
- [x] Layout app: sidebar + header (search, notif, avatar, jabatan, dropdown) — topbar baru, bukan Tabler polos.
- [x] Statistik real (counts + outstanding + revenue), chart 6 bulan, agenda, notifikasi, recent, tasks.
- [x] Klien: breadcrumb, search, filter jenis, pagination+total, row aksi, ekspor CSV; kolom nama/jenis/identitas/kontak/status/dibuat.
- [x] Detail klien: umum, identitas, alamat, kontak, duplikat, perwakilan, dokumen, riwayat layanan, jejak audit.
- [x] Editor akta: sidebar dokumen (daftar akta), toolbar (review/sahkan/PDF/DOCX/cetak), canvas, panel klausul,
      riwayat versi, komparasi, approval panel. Textarea + sisip klausul (jujur: belum rich-text; lisensi editor ditunda).
- [x] Repertorium: filter tahun/bulan/nomor, detail baris, cetak, ekspor CSV; nomor dari backend (FOR UPDATE).
- [~] Metronic layout: struktur setara (shell/sidebar/header/cards/timeline/toast/empty/loading) dengan token sendiri;
      BLOCKED untuk label resmi.
- [x] Tidak ada tombol mati yang ditemukan pada rute baru (semua form punya handler; diuji 200).
- [x] Responsive: viewport + grid Tabler + tabel responsif + aturan 768px; belum uji perangkat fisik (catat).

## Sisa jujur
- Halaman Waarmerking terpisah belum ada (masih tab di Legalisasi — by design, bukan halaman kosong).
- Dokumen Masuk/Keluar terpisah belum ada (Document Center global sebagai gantinya).
- Estimasi Biaya + Laporan Akta/Keuangan khusus belum ada (Invoice + Reports + Tariffs sebagai cakupan kini).
- Role & Permission UI khusus belum ada (manajemen peran via seed + settings pengguna).
