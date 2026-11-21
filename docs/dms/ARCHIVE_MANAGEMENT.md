# DMS — Manajemen Arsip & Retensi

## 1. Kebijakan (seed `migrations/004_ops.sql`)

| Entitas | Tahun | Catatan |
|---|---|---|
| `minuta` | 25 | UUJN: protokol disimpan 25 tahun oleh notaris bersangkutan |
| `repertorium` | 25 | bagian protokol notaris |
| `legalisasi_register` | 10 | arsip administrasi kantor |
| `keuangan` | 10 | arsip perpajakan & audit |

`retention_policies(entity UNIQUE, retain_years, note)`.

## 2. Siklus hidup arsip

1. **Aktif** — berkas dipakai pada layanan/akta berjalan.
2. **Semi-aktif** — layanan selesai; berkas read-only secara de-facto
   (perubahan hanya via koreksi tercatat + audit).
3. **Inaktif/retensi** — melewati masa aktif tetapi wajib simpan
   sesuai tabel di atas; akses dibatasi (`protocol.view`, `reports.view`).
4. **Pemusnahan/serah** — hanya setelah masa retensi lampau + persetujuan
   notaris; minuta/protokol mengikuti UUJN (penyerahan, bukan hapus liar).
   Aplikasi tidak menyediakan hapus massal arsip.

## 3. Aturan keras

- `files.office_id` `ON DELETE RESTRICT` — kantor berarsip tidak bisa dihapus.
- `ai_results.file_id` `ON DELETE SET NULL` — hasil OCR bertahan sebagai jejak
  meski file dirujuk ulang; jangan hapus fisik tanpa menghapus/mengarsipkan metadata.
- Penghapusan fisik (bila diizinkan kebijakan) wajib: tiket persetujuan +
  baris audit + verifikasi `sha256` sebelum hapus.

## 4. Operasional

- Backup: direktori `LocalStore` + dump MySQL diambil berpasangan.
- Restore: validasi sampling `ref → sha256` sebelum serah-terima ke produksi.
- Serah-terima protokol notaris: di luar fitur aplikasi; catat manual
  (berita acara) dan lampirkan sebagai berkas `owner_type=protocol`.
