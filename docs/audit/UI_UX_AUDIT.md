# UI/UX Audit 4.0 — Template `*.gohtml` (27 file, 130 `define`)

Metode: BACA SAJA. Semua `action=` form dipetakan ke route `Router()` (chi),
semua `href="/..."` dipetakan ke route GET, semua `define` dipetakan ke
`a.render(...)` / `renderPub`. Status memakai skala FASE 1C.

## 1. Halaman yatim / tanpa route — IMPLEMENTED_AND_VERIFIED

- 130 `define` terpetakan: 85 nama dipakai `a.render(w,r,"...")`, sisanya
  partial layout (`head/page_open/page_close/navbar/sidebar/flash/...`) atau
  view publik via `renderPub` (`pub_home/pub_page/pub_article/...`,
  `landing/layanan_index/layanan_detail/docs_index/docs_step/verify`).
- `RENDERED_NOT_DEFINED`: kosong. Tidak ada halaman yatim.
- Catatan: `clients.gohtml` (194 byte) hanya komentar penanda migrasi ke
  `clients_list/detail/edit/wizard.gohtml` — aman, tinggal hapus (LOW).

## 2. Tombol POST tanpa handler — IMPLEMENTED_AND_VERIFIED

- 150+ `action=` diperiksa: semua `POST` cocok dengan route `p.Post(...)`
  (`/deeds/{id}/approve`, `/dms/bulk`, `/cms/versions/{id}/rollback`, ...).
- Tidak ada tombol buntu. Kasus khusus terverifikasi: `action=/dms/library`
  dan `action=/dms/search` memakai `method=get` sesuai route `p.Get`
  (bukan POST), jadi valid.

## 3. Filter yang diabaikan handler — IMPLEMENTED_AND_VERIFIED

- `clients_list` (12 param: q/kind/status/docs/sort/service/province/city/
  from/to/per_page) → `clientList` membangun `clients.ListFilter` lengkap
  dan memanggil `ListAdvanced(office, f)` (`app.go:1488`). `per_page`
  di-whitelist 10/20/50/100.
- `dms/library`, `cms/*`, `repertorium`, `reports` memakai pola sama
  (query → struct filter → query office-scoped). Tidak ada filter mati.

## 4. Form tanpa validasi client — IMPLEMENTED_AND_VERIFIED

- Semua form POST authed menyertakan token CSRF (`{{.CSRF}}`), tanpa
  `novalidate`, `required`/`min`/`maxlength` terpasang pada input wajib
  (12 di `app2`, 14 di `cms`, 22 di `pages2`).
- Pengecualian by-design: `POST /login` (CSRF middleware exempt),
  `POST /konsultasi` + `POST /booking` publik (rate-limit 10/mnt, tanpa
  CSRF — wajar untuk form anonim). Server-side tetap validasi ulang
  (contoh: `bankUpload` cek format tanggal+nominal per baris).

## 5. Tabel tanpa `.table-responsive` — IMPLEMENTED_AND_VERIFIED

- 66 `<table` vs 66 `table-responsive` di 27 file — rasio 1:1, tidak ada
  tabel telanjang. Mobile aman.

## 6. Link buntu — IMPLEMENTED_AND_VERIFIED

- 181 `href="/..."` internal: semua cocok ke route GET (`/qr`, `/offline`,
  `/manifest.webmanifest`, `/media/by-ref`, `/s/{token}`, `/p/preview`...).
- Dualitas `/documents/{id}` (detail) vs `/files/{id}` (download biner)
  keduanya ada route-nya — bukan buntu, tapi perlu label UI yang jelas
  (lihat §8).

## 7. Empty-state tanpa CTA — PARTIALLY_IMPLEMENTED

| Lokasi | Kondisi | Status |
|---|---|---|
| `clients_list` "Tidak ada klien." | tanpa tautan `/clients/new` | NOT_CONNECTED |
| `dms_enterprise` "Belum ada field kustom." | tanpa CTA tambah field | NOT_CONNECTED |
| `dashboard2` panel kosong | ghost-button non-aksi | UI_ONLY |
| `cms_growth/slots`, `ops_growth/mandates` | teks saja, CTA ada di header lain | PARTIALLY_IMPLEMENTED |

Sisanya (18 empty-state di `dms.gohtml`, 12 di `app2`) sudah ber-CTA.

## 8. Inkonsistensi label/bahasa — PARTIALLY_IMPLEMENTED

- Campuran ID/EN: "Dashboard", "Status", "Export", "Delete", "Create",
  "Search", "Save", "Submit" di tengah kalimat Indonesia.
- Format tanggal ganda: `%d %b %Y` vs `%d %B %Y` (registers vs ppat_blanko).
- Penamaan ganda: "Wasiat" vs route `/wills`, "Legalisasi" vs "LCC",
  `/documents` vs `/dms/library` untuk konsep yang sama.

## 9. Prioritas perbaikan (P0 tertinggi)

1. P1 — CTA pada 2 empty-state (§7) + hapus stub `clients.gohtml`.
2. P2 — Glosarium label ID tunggal; samakan format tanggal
   (`%d %b %Y`); bedakan label "Detail dokumen" vs "Unduh berkas".
3. P3 — Ujicoba mobile 360px pada `dms.gohtml` (29 form, 14 tabel —
   terberat) dan `ops.gohtml` (24 form).
