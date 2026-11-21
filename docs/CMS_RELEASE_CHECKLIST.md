# CMS Release Checklist — sebelum klaim selesai

## Fungsional (E2E `scripts/qa_shots/cms.py`, 25/25)
- [x] Dashboard CMS (count, missing SEO, riwayat)
- [x] Branding tersimpan + tampil di admin
- [x] SEO global + domain (valid + tolak `javascript:`)
- [x] Halaman terbit + section + render publik + sanitasi script
- [x] Beranda CMS menggantikan landing
- [x] Artikel terbit + tampil publik + terkait
- [x] FAQ tersimpan + tampil publik
- [x] Nav tersimpan (header)
- [x] Media unggah + re-encode + ref tampil + tolak .py
- [x] Sitemap memuat URL CMS (hanya terbit)
- [x] Riwayat publikasi tercatat
- [x] Setup + konten contoh = DRAF

## Keamanan (unit + E2E)
- [x] Unit: branding validation, nav URL, slug, section schema, sanitizer, media validate
- [x] Integrasi: lifecycle + slug ganda + publish-guard + scheduled + isolasi publik (`TestCMSLifecycle`)
- [x] Direktori privat tak terekspouse; CSRF semua POST; rate limit upload
- [ ] Pen-test independen (eksternal)

## Regresi
- [x] `go vet ./...` bersih; `go build` OK
- [x] Integrasi 23/23 (22 lama + CMS)
- [x] Matriks izin 28 rute (tambah rute CMS)
- [ ] Uji beban halaman publik 100 halaman/500 artikel (dataset realistis — backlog)

## Rilis
- [x] Migrasi 015 di prod + test DB
- [x] Dokumen 12 berkas sinkron implementasi
- [ ] Restore di VPS produksi (B-09); TLS produksi (B-08); tinjauan notaris (B-06)
