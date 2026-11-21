# MASTER IMPLEMENTATION AUDIT — baseline 2026-11-21 (sesi White Label + CMS)

Metode: baca langsung routes (`internal/app/app.go` 119 rute auth + 17 publik),
migrasi 001–014, store, template, middleware, tests. Skor/hasil dari run nyata.

## 1. Modul tersedia (berfungsi + terbukti)
- Auth: login rate-limit, sesi server-side, MFA TOTP, 9 peran/19 izin, matriks 120 sel PASS.
- Klien, permohonan 9 status, korporat + wizard, akta berversi + RTE sanitasi + diff + TTE,
  wasiat, repertorium, protokol, legalisasi, gov-asistensi, finance + QR + /verify, portal,
  template/klausul, DMS penuh (OCR, FTS, share, arsip, retensi, duplikat+merge, ClamAV on-demand),
  multi-kantor + tier + gate 402, SLA/eskalasi, webhook/SMTP handoff, S3 SigV4, backup/restore.
- Keamanan: CSRF, SecurityHeaders, bcrypt-12, NIK AES-GCM, audit imutabel.
- Uji: integrasi 22/22, unit 9 paket, E2E 10+6+8+8+14+120+14+5+5, beban baca 50/50 + tulis konkuren.

## 2. Modul belum tersedia (tugas perintah ini)
- White Label: identitas/branding/theme per kantor di DB (kini: profil kantor biasa + logo statis).
- Website CMS: pages/sections/articles/faq/nav/media/seo/domain/workflow (kini: landing statis + /layanan + /docs statis).
- Media library publik terpisah dari DMS privat.
- Cache terukur + invalidasi.
- Rate limit upload media; host/canonical validation.
- Setup wizard + konten contoh berlabel.

## 3. Setengah jadi / akan ditingkatkan
- SEO: sitemap/robots/llms statis → gabung konten CMS dinamis.
- Branding dokumen: receipt/invoice header masih merek statis.
- Aksesibilitas: landmark dasar ada; audit pembaca layar penuh belum.

## 4. Bug/duplikasi/risiko (dipantau saat implementasi)
- Tidak ada duplikasi modul; CMS dilarang duplikat (satu `internal/cms`).
- Risiko: template Go sensitif `</script>` hilang (pelajaran: E2E render check tiap halaman baru).
- Risiko: ENUM MySQL butuh migrasi eksplisit (014 preseden).
- Risiko: PowerShell `>` menulis UTF-16 (dump via cmd).
- Eksternal: tinjauan notaris, TLS produksi, restore VPS produksi, pen-test (tetap di BLOCKERS).

## 5. Aturan main sesi ini
- Backend Go, MySQL 8, Tabler, Modular Monolith, standalone per kantor. Larangan: hapus fitur,
  ganti framework, CDN inti, kredensial di kode, klaim AHU/OSS/TTE-PSrE, tombol/data palsu.
- Izin baru mengikuti konvensi `website.*`, dipetakan ke peran existing (tanpa role baru).
- JSON hanya untuk setting section tervalidasi; entitas CMS = tabel relasional.
- Setiap modul: migrasi → store+unit → handler → template → E2E → regresi.
