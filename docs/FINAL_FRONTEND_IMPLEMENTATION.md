# FINAL_FRONTEND_IMPLEMENTATION — NOTARIUS (2026-11-21)

## Selesai
- Aset lokal: `tabler.min.css/js` + `htmx.min.js` di `web/assets/vendor/`; nol referensi CDN di template
  (verifikasi: grep `cdn.jsdelivr|unpkg` di `internal/app/templates/` = kosong). CSP diperketat (tanpa host CDN).
- Token NOTARIUS (`web/assets/notarius/notarius.css`): navy/gold, sidebar navy, topbar, avatar, badge ID,
  doc canvas, workflow stepper, print CSS, mobile 768px. Dok: `docs/design/DESIGN_SYSTEM.md`.
- Landing `/` 11 seksi: navbar sticky+mobile, hero+3 gambar WAN lokal, trust 5 item, 8 kartu fitur,
  showcase 3 panel, workflow 6 langkah, keamanan aktual, harga konsultasi (tanpa angka palsu),
  5 FAQ, CTA, footer, SEO meta. CTA → `/login` (fungsi nyata).
- Dashboard `/app` (redirect `/dashboard` kompatibel): sapaan, 4 kartu real-query, grafik 6 bulan
  (permohonan+akta), agenda hari ini, notifikasi + tandai-dibaca, aktivitas terbaru.
- Topbar: search global `/search`, tugas, notifikasi, avatar+nama+peran+dropdown (MFA/kantor/keluar).
- Sidebar navy berkelompok + gating `can()` per 9 peran; link baru: Dokumen, Tugas, Audit, Kantor, Tarif.
- Klien: breadcrumb, filter jenis+q, pagination 20/halaman + total, ekspor CSV, detail + riwayat
  layanan + jejak audit. Validasi backend tetap (NIK/telp/email).
- Editor akta: pilih template terapproved saat buat draf (ditolak bila belum approved),
  pustaka klausul sisip-1-klik, cetak, komparasi vN vs vN-1 berdampingan, submit review + sahkan + PDF/DOCX.
- Repertorium: filter tahun/bulan/nomor, ekspor CSV (penomoran server-side), cetak.
- Brand: judul/template NOTARIUS; modul Go tetap `notaryos` (rename modul = churn, ditunda).

## Belum / jujur
- Metronic resmi: BLOCKED (tanpa aset). Klaim "Metronic" DILARANG sampai aset resmi + hapus Tabler.
- Screenshot browser: BLOCKED (tanpa otomasi browser). Bukti pengganti: 29/29 rute HTTP 200
  + pemeriksaan konten + `go vet/build/test` hijau (lihat FINAL_FRONTEND_TEST_REPORT.md).
- Grafik memakai div proporsional (tanpa lib chart lokal); export PDF laporan belum ada (baru CSV).

## File diubah
`web/assets/{vendor,notarius}/`, `internal/app/{app,ext}.go`, `internal/{middleware,clients}`,
`internal/app/templates/{partials,landing,dashboard2,clients,registers,app2,login}.gohtml`,
`docs/design/DESIGN_SYSTEM.md`, `docs/FINAL_*.md`.
