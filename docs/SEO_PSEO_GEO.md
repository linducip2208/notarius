# SEO / PSEO / GEO STRATEGY — NOTARIUS (jualan aplikasi, tetap jujur)

## Prinsip dagang yang tidak dilanggar
- Tanpa angka pelanggan palsu, tanpa testimoni palsu, tanpa klaim sertifikasi,
  tanpa harga mengada-ada (harga = konsultasi/penawaran resmi).
- Tanpa klaim integrasi pemerintah. Tanpa halaman kota fiktif (tidak ada kantor cabang palsu).

## Teknis (sudah live)
- `robots.txt`: crawl hanya rute publik (`/`, `/layanan`, `/assets`); rute app di-Disallow.
- `sitemap.xml`: dinamis dari katalog `services` (landing + indeks + 22 URL layanan + anchor).
- Canonical per halaman; OG + JSON-LD `SoftwareApplication`; `llms.txt` ringkas faktual.

## PSEO (programmatic, dari data nyata)
- Sumbu: kode layanan katalog (`/layanan/{code}`) — 22 halaman unik: syarat (JSON requirements),
  tahapan, FAQ schema (`Service` + `FAQPage`), breadcrumb, internal-link silang, CTA penawaran.
- DILARANG: halaman per kota (tidak ada kehadiran fisik), halaman tag tipis, doorway pages.

## GEO (dikutip AI dengan benar)
- Fakta atomik per halaman (syarat list, langkah bernomor, batasan eksplisit).
- `llms.txt` + FAQ schema + jawaban "tanpa X" yang eksplisit (bukan PPAT, tanpa API resmi).
- Konsistensi klaim di semua halaman (satu sumber: katalog + docs ini).

## Berikutnya (backlog jualan)
- Artikel panduan (akta PT/CV/wasiat) bila ada penulis hukum; video demo; changelog publik.
- Search Console + ukur: impresi `/layanan/*`, CTR, kutipan AI (pantau manual).
