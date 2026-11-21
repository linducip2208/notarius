# Website CMS Guide — kelola website publik dari Admin Panel

Masuk: **Website →** (butuh izin `website.view`; kelola butuh izin granular).

## Alur konten (wajib)
`DRAFT → REVIEW → PUBLISHED` (atau `SCHEDULED` + waktu masa depan → terbit otomatis
oleh pemindai harian). `ARCHIVED` = arsip/takedown. **Draf/review/arsip TIDAK PERNAH
tampil publik** — public renderer hanya query `status='PUBLISHED'`.

Halaman duplikat selalu lahir sebagai DRAF. Konten contoh setup berlabel `[CONTOH]`
dan DRAF — tidak tampil sebelum disunting + diterbitkan manual.

## Halaman (`/cms/pages`)
- Slug unik per kantor `[a-z0-9-]`; SEO title/desc, canonical, OG image, robots.
- Section builder: tambah/susun (position), tampil/sembunyi, hapus. Tipe tervalidasi
  (hero, intro, profile, services, process, info, articles, faq, contact, map,
  footer, text, gallery, cta); URL CTA relatif/`http(s)` saja (`javascript:` ditolak);
  settings JSON hanya whitelist (columns 1–3, ratio, theme).
- Beranda = halaman slug `beranda` yang TERBIT. Tanpa itu, `/` menampilkan landing lama.
- Pratinjau: tombol "Pratinjau publik" (halaman DRAF tidak bisa dipratinjau publik —
  sengaja; gunakan tampilan admin).

## Artikel (`/cms/articles`)
Kategori, tag, penulis, cover, jadwal, SEO. Terkait otomatis (kategori sama, maks 3).
Tidak ada klaim hukum otomatis — tulis manual dan review manusia.

## Layanan (`/cms/services`), FAQ (`/cms/faq`), Navigasi (`/cms/nav`)
- Layanan = info publik sesuai ruang Notaris (BUKAN layanan operasional `/layanan`).
- FAQ tampil hanya yang visible, dikelompokkan kategori.
- Menu header/footer + induk + urutan. URL relatif/`http(s)`; `javascript:`/`data:` ditolak server.

## Riwayat (`/cms/history`)
Setiap create/update/duplicate/publish dicatat: aktor, dari→ke, catatan, IP.

## Pengaturan (`/cms/site`) & Setup (`/cms/setup`)
- SEO global, base URL, domain utama, centang HTTPS (butuh TLS di reverse proxy —
  tidak diklaim otomatis), verifikasi domain (konfirmasi manual + host terakhir terlihat).
- Setup awal: identitas + kontak + tema + domain + contoh konten DRAF.

## Batas jujur
- Pratinjau DRAF hanya di admin (tidak ada token pratinjau publik).
- Tidak ada analytics pengunjung (jangan mengarang angka).
- `robots` per halaman didukung; sitemap hanya URL terbit.
