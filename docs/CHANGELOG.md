# CHANGELOG
## [0.4.0] — 2026-11-23 — OCR enterprise + DMS lengkap + PPAT + bundel publik
- Layanan & Katalog: request form pintar + optgroup kategori, draf 1-klik dari permohonan, publik layanan detail + 3 paket bundel (PT+OSS, AJB+Balik Nama, CV+NIB), wizard korporat/AJB/wasiat, docs 8 langkah + checklist cetak + panduan uji.
- OCR & DMS: OCR enterprise (quality gate + preprocessing + voting multi-engine + parser KTP/KK/NPWP + cek konsistensi antar-dokumen); DMS preview universal (PDF/gambar/teks), dipakai-di-mana per berkas, verifikasi 2-kolom (checklist vs berkas), SLA + retensi per dokumen, seed 23 paket checklist, mode ketat per kantor.
- Akta & Register: Todo↔akta (preview + edit dari daftar tugas), preview semua draf di daftar deed, jenis baru Firma/Koperasi + 7 jenis PPAT (AJB, Hibah, TukarMenukar, APHB, Inbreng, SKMHT, APHT), register PPAT terpisah (2 buku) + blanko bernomor + resi SLA BPN.
- Keuangan & Laporan: uang muka/DP per invoice, tarif otomatis dari katalog saat buat invoice, laporan rekap per kategori + top 5 layanan + kepatuhan SLA.
- Portal & Notifikasi: timeline status permohonan di portal klien, notifikasi WA manual per permohonan (template + audit, tanpa klaim otomatis).
- Keamanan: PII masking (NIK/NPWP/tanggal lahir) di daftar + preview + log sesuai peran; tanpa perubahan auth/RBAC.
- Migrasi DB 021–024 (idempoten, jalan otomatis saat boot via tracker schema_migrations): 021 register PPAT + 6 layanan PPAT; 022 jenis Firma/Koperasi + layanan Inbreng; 023 flag mode ketat + slot SLA per kantor; 024 kategori 'ppat' + 3 layanan bundel. Tanpa DROP/ubah data pengguna (INSERT IGNORE + CREATE TABLE IF NOT EXISTS).
- Go-Live: backup DB → deploy biner → restart (migrasi otomatis) → ⚡ Pasang Standar di /dms/checklists → ⚡ Pasang 8 Konsep di /templates → Approve template → uji 1 permohonan (lihat docs/DEPLOYMENT_GUIDE.md).
## [Ops-Growth] — 2026-11-22 — semua ide fitur non-SaaS (dibuktikan E2E + integrasi)
- CRM: skor kelengkapan profil, timeline 360 (tagihan/janji/berkas/verifikasi), spesimen TTD pembanding visual.
- Keuangan: rekonsiliasi bank CSV + automatch eksplisit, anggaran vs realisasi, laba per layanan, invoice tertaut permohonan.
- Operasi: inquiry→permohonan 1 klik, peringatan konsistensi korporat + org chart cetak, gate pra-pengesahan akta, paket ZIP akta, berkas tertaut akta, 3-way template, antrean legalisasi + estimasi, tindak lanjut gov, review + pembukaan wasiat bersaksi, tugas berulang + tautan konteks + refLink, retensi→tugas, backup mysqldump 1 klik, mode darurat baca-saja (READ_ONLY=1), kalender 14 hari, kartu notaris + banner kata sandi di dasbor, celah nomor repertorium + label QR, kuitansi portal + notifikasi berkas baru.
- Pengingat: ultah, identitas kedaluwarsa, review wasiat tahunan, ringkasan pagi, SLA per layanan.
- Keamanan: sesi catat IP/UA + notifikasi perangkat baru (login biasa + MFA).
- Migrasi 019. Bukti: ops_growth.py nihil gagal + integrasi MySQL + regresi penuh.
- Batasan jujur: H-1 WA aktif hanya bila gateway dikonfigurasi; OCR search sudah ada (dipakai ulang); kata sandi belum ada alur ganti mandiri (banner + admin reset).
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
