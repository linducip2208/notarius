# CMS Growth — 14 Fitur (bukti: `scripts/qa_shots/cms_growth.py` SEMUA HIJAU)

## Publik (tanpa login)
| Rute | Fungsi | Kejujuran |
|---|---|---|
| `/kalkulator?code=` | Estimasi dari `services.base_price` + `price_items` aktif | Label "bukan harga final" selalu tampil |
| `/konsultasi` | Formulir → `website_inquiries` (rate-limit) | Masuk ke meja staf, bukan email gelap |
| `/booking` | Slot mingguan berkuota → `website_bookings` | Kuota per jam; maks 2 bulan; HP wajib valid |
| `/tim`, `/galeri`, `/testimoni` | Profil, foto, kata klien | Terbit testimoni wajib `verify_note` |
| `/cari?q=` | Cari halaman/artikel/layanan/FAQ | — |
| `/arsip` | Arsip artikel per bulan | — |
| `?lang=en` | Versi Inggris bila ada, fallback ID | Tidak pernah kosong bila ID ada |
| `/p/preview?kind=&id=&token=` | Pratinjau draf (staf login + HMAC) | Tidak terindeks publik |

## Operasional
- **Persetujuan booking** membuat klien tamu + janji temu resmi (pengingat H-1 ikut jalan).
- **Versi konten**: snapshot tiap update (20 terakhir), rollback satu klik; pra-rollback ikut tersimpan.
- **Kualitas**: skor keterbacaan heuristik + daftar masalah SEO per konten terbit.
- **Statistik**: pageview agregat harian dihitung middleware (tanpa tracker pihak ketiga).
- **Ekspor**: ZIP markdown halaman+artikel.
- **Banner & maintenance**: pengumuman situs + mode 503 anonim (staf login tetap bisa kerja).
- **JSON-LD Notary** + hreflang-ready (`?lang=`), sitemap mencakup rute baru.

## Batasan jujur (sengaja)
- Kalkulator = estimasi, bukan penawaran. Final setelah verifikasi berkas.
- Testimoni tanpa bukti verifikasi tidak bisa terbit (ditolak server, bukan sekadar UI).
- Kuota booking per jam mulai (bukan per rentang tumpang-tindih) — cukup untuk praktik kantor.
- i18n hanya ID/EN; navigasi & banner belum diterjemahkan (fallback ID).
