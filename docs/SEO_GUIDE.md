# SEO Guide — metadata, sitemap, robots (tanpa klaim palsu)

## Per halaman/artikel
Isi di form: SEO title, SEO desc, canonical, OG image, robots
(`index,follow` default; `noindex` untuk draf/preview internal).
Dashboard CMS menandai konten TERBIT tanpa SEO title/desc (daftar "Butuh SEO").

## Global (`/cms/site`)
Judul situs, deskripsi situs, OG image default, base URL (untuk canonical absolut).
Kosong = fallback aman (nama kantor + host aktif).

## Sitemap (`/sitemap.xml`)
Hanya URL publik: `/`, layanan operasional, tutorial, halaman CMS terbit,
artikel terbit, layanan info, halaman statis (`tentang-kami`, `kontak`, `artikel`,
`faq`, `layanan-kami`). DRAF/REVIEW/SCHEDULED/ARCHIVED **tidak pernah** masuk.
Uji: `cms.py` menegaskan `/artikel/` dan `/p/` muncul setelah terbit.

## Robots (`/robots.txt`)
Allow `/`, `/layanan`, `/artikel`, `/faq`, aset. Disallow total: `/app`,
`/clients`, `/requests`, `/deeds`, `/finance`, `/settings`, `/login`, `/cms`.

## Larangan
- Tidak ada rating, testimoni, penghargaan, kredensial Notaris palsu.
- Tidak ada metadata duplikat: canonical per dokumen; sitemap tanpa duplikat.
- OG image default bila artikel tanpa cover (bukan gambar rusak).
