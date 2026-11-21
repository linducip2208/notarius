# CHANGELOG
## [Hardening] — 2026-11-22 — lockout + token expiry (dibuktikan)
- Lockout: 5x gagal/15 mnt → kunci 15 mnt per email lintas IP; `login_attempts` aktif dipakai; sukses membersihkan riwayat; pesan generik; audit `login.locked`.
- Cookie CSRF → `HttpOnly`; token pratinjau draf HMAC → kedaluwarsa 24 jam.
- Bukti: curl 6x salah → kunci tampil; integrasi MySQL PASS; E2E growth/cms/matrix/fase1a-b/fase2-3/wizard/merge/interact/confirm SEMUA HIJAU.
- Lihat `docs/SECURITY_HARDENING.md`; sisa deploy: HTTPS + COOKIE_SECURE, pen-test, race-CI.
- Publik: kalkulator estimasi transparan (DB services+price_items, bukan harga final), konsultasi masuk → staf, booking slot berkuota → janji temu + klien tamu, tim/galeri/testimoni terverifikasi, pencarian situs, arsip artikel, i18n ID/EN (fallback ID), JSON-LD Notary, banner pengumuman, mode pemeliharaan (503 anonim, staf bypass), pageview internal tanpa tracker.
- Admin: inbox konsultasi + status, slot + keputusan booking, tim/galeri/testimoni (terbit wajib bukti verifikasi), versi konten + rollback, dasbor kualitas (keterbacaan + SEO), statistik kunjungan, ekspor ZIP markdown, pratinjau draf bertoken HMAC.
- Integrasi jujur: CTA layanan lama → /konsultasi + /kalkulator (ganti /login); migrasi 018; izin baru website.inquiries/engagement.manage.
- Perbaikan: duplikat rute /layanan dihapus (pakai hub pSEO yang sudah ada + syarat).
## [Fase-8] — 2026-11-22 — Enterprise CRM + DMS lanjutan (uncommitted → verifikasi)
- Migrasi 016_client_crm + 017_dms_enterprise (idempoten) teraplikasi di DB prod & test.
- CRM: pencarian lanjutan, statistik, catatan, tandai inkonsisten (alasan wajib), pratinjau/penggabungan duplikat, ekspor CSV.
- Korporat: referensi legal manual (no AHU/OSS), taut pemegang saham→klien.
- DMS: field kustom tervalidasi, pencarian lanjutan + pagination + ekspor CSV, aksi massal, banding versi, matriks persetujuan bertingkat (gate APPROVED), pratinjau watermark + audit, analitik + kuota (blok 507 + pesan jujur), ekspor ZIP per kantor, penegakan kerahasiaan di preview/unduh/bagi, retensi-notify tanpa hapus otomatis.
- Template: library bulk/pagination/filter tanggal-MIME-urut, dashboard kuota, lisensi informasional, linimasa akta.
- Bug diperbaiki (dengan tes): Schedule menyimpan UTC sementara pembanding WIB → simpan waktu dinding lokal (TestRemindersRun flake zona waktu).
- Smoke: 11/11 rute baru 200; go vet bersih; integrasi MySQL penuh (lihat bawah).
- Build fix: 3 handler hilang (deedApprove, deedToReview, govCheck) diimplementasi + route + audit; `go vet/build` hijau.
- F-009 COMPLETED: beneficial_owners (migrasi 008) + UI + validasi TPPU + PEP flag + test integrasi.
- F-014 COMPLETED: serah-terima wasiat (handover notaris-only + audit + UI + test ganda-gagal).
- F-015/F-023 COMPLETED: aksi Review/Sahkan di daftar akta, toggle checklist AHU/OSS per-item.
- Hardening: HSTS + Permissions-Policy, session SQL + cleanup per-jam, seed idempoten, Makefile migrate 001-008.
- F-022 COMPLETED (honest): adapter Tesseract lokal + provenance + fallback Noop jujur + docs instalasi; tanpa kunci AI eksternal (by design).
- Test: 10 paket unit PASS + validate via rename (AV) PASS + 8 integrasi MySQL (2 baru) siap; fitur 28/28 = 100%.
## [0.2.0] — 2026-11-20 — Sprint 2 (continuous execution)
- Seed katalog: 22 layanan + migrasi 005/006; runner Go idempoten (count 0).
- Keamanan: rate-limit login per-IP (diuji 10→429), CSRF semua POST, MFA TOTP stdlib + alur login 2 langkah, enkripsi NIK AES-GCM.
- Domain: identitas + verifikasi manusia, perwakilan, korporat + saham + pengurus, janji temu, template editor + klausul + approval, upload privat + OCR jujur, portal terisolasi, wasiat akses-terbatas, invoice multi-baris, PDF ekspor DRAFT.
- Bug diperbaiki dari smoke test: kolom invoice, DECIMAL→int64, kunci akta APPROVED, limiter per-IP.
- Test: 6 integrasi MySQL asli PASS + 9 paket unit PASS. Fitur: 26/28 (93%).
## [0.1.0] — 2026-11-20 — Fondasi v1 (greenfield)
- Repo kosong diaudit (INITIAL_AUDIT); diputuskan stack Go + MySQL + Tabler.
- 4 migrasi MySQL (35 domain), seed roles/permissions, kebijakan retensi.
- Auth bcrypt + sesi, RBAC 9 peran/19 izin, middleware keamanan.
- Domain: users, clients (+duplikat), requests (workflow), deeds (+repertorium/protokol/legalisasi),
  finance (invoice/PPN/pembayaran), docengine, ocr interface, gov asistensi, storage privat, workers.
- 17 halaman Tabler + HTMX; dashboard/laporan real-query.
- Unit test 7 paket hijau; `go vet` + `go build` bersih.
- Deploy: nginx, systemd, mysql init, backup/restore GPG.
