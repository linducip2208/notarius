# NOTARIUS Design System v1

## Status jujur
- Token + komponen di bawah adalah **lapisan identitas NOTARIUS** (`web/assets/notarius/notarius.css`)
  di atas utilitas Tabler yang di-vendor lokal (`web/assets/vendor/`).
- Ini **BUKAN aset Metronic resmi** — migrasi Metronic tetap BLOCKED sampai lisensi resmi tersedia
  (lihat `web/assets/metronic/README.md`). Tidak ada campuran dua design system:
  satu-satunya framework tampilan adalah Tabler-lokal + token NOTARIUS.

## Token
| Nama | Nilai | Pakai |
|------|-------|-------|
| `--nt-navy` | #14264B | sidebar, heading, tombol primer teks |
| `--nt-navy-2` | #1C3159 | varian navy |
| `--nt-gold` | #D6A844 | CTA, aksen, badge |
| `--nt-gold-lt` | #F3D58A | latar ikon |
| `--nt-blue` | #3978E8 | link, info |
| `--nt-bg` | #F5F7FB | latar aplikasi |
| `--nt-surface` | #FFFFFF | kartu |
| `--nt-text` / `--nt-text-2` | #17233C / #69758B | teks |
| `--nt-success` / `--nt-warning` / `--nt-danger` | #16A37B / #E9A23B / #DC5252 | status |
| `--nt-border` | #E7EBF2 | garis |
| radius | 12px / 8px | kartu / tombol-input |
| shadow | 0 10px 30px -12px rgba(20,38,75,.18) | kartu terangkat |

## Komponen (`notarius.css` + partials Tabler)
- `.btn-gold`, `.stat-card` + `.stat-ico.{blue,gold,green,navy}`, `.topbar` sticky,
  `.avatar-circle`, `.doc-canvas` (kertas A4-ish), `.wf-dot/.wf-line` (workflow),
  badge status Bahasa Indonesia, tabel (header uppercase kecil), print CSS
  (sembunyikan nav saat cetak repertorium/dokumen), breakpoint mobile 768px.
- Layout: `page_open` (topbar search + tugas + notifikasi + avatar dropdown) +
  `sidebar` navy berkelompok + gating `can(role,perm)` + `page_close`.
- Tidak ada gradient berlebihan, tidak ada dark-mode default, whitespace lega.

## Aturan pakai
1. Semua halaman baru WAJIB pakai `head/page_open/page_close` — dilarang layout sendiri.
2. Warna literal (hex) di template DILARANG; pakai class token.
3. Status tampil Bahasa Indonesia (Aktif, Selesai, dll.), bukan enum mentah.
4. Tombol destruktif/final WAJIB `confirm()` + audit trail.
5. Setiap tabel WAJIB empty-state; setiap form WAJIB pesan error jujur.
