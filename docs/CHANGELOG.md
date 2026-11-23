# CHANGELOG
## [0.4.6] — 2026-11-23 — Aktivasi kantor + filter + audit scope + SEO responsif
- Aktivasi & Filter: kartu Aktivasi Kantor x/5 di /office (template checklist, template dokumen + approved, tarif/uji transaksi via invoice atau permohonan, uji 1 permohonan, mode ketat diketahui; fail-open 0 bila tabel belum termigrasi) + filter status permohonan /requests (pertahankan client + Reset) + filter status + jenis akta /deeds (kind difilter di Go cocok persis ENUM, aman injeksi) + badge jatuh tempo invoice LEWAT N hari + tgl Jt (overdue di atas, partisi stabil tanpa ubah ORDER BY store).
- Keamanan scope: 8 kebocoran office-scope ditutup — feed audit dashboard + /audit + /audit/export WHERE office (aktor NULL lolos), booking DISETUJUI/tolak UPDATE ... AND office_id, spesimen DMS AND office_id, deed detail props/template/timeline (versions JOIN deed_drafts, approval, signatures) AND office_id, invoice_lines JOIN invoices cek office; template yatim "documents" dihapus dari app2.gohtml.
- SEO/Responsif/Cetak (dipakai ulang v0.4.4/v0.4.5): robots (Allow /docs /verify, Disallow /portal /admin) + llms.txt live dari katalog, canonical + og:url absolut, FAQ JSON-LD fallback; responsif 5 halaman (table-responsive, flex-wrap, grid BO, deedview anti-luber); CSS print global + .no-print; audit CSV /audit/export (BOM + 1000 baris) + strict UX flash.
- Operasional: timer pengingat harian 07:00 WIB via deploy/reminder/run-reminder.sh (idempoten dengan ticker + tombol manual); backup DB --no-tablespaces + backup BERKAS backup_files.sh (drill lolos); MFA warning peran berhak wasiat tanpa blokir login; cookie sesi MaxAge 43200 + HttpOnly + SameSite + Secure konsisten.
- Docs: .env.example 49 kunci terdokumentasi (kanonis NOTARYOS_S3_* menang atas alias lawas, SEED_SK/ADDR/PHONE/EMAIL/ADMIN_NAME, DB_DSN, NOTARYOS_CHAIN_KEY, CLAMAV_ADDR, READ_ONLY, FILES_BACKUP_DIR/RETENTION, GPG_RECIPIENT, REMINDER_BASE_URL/USER/PASS); tanpa ubah skema/migrasi.
## [0.4.5] — 2026-11-23 — Timer pengingat + cetak rapi + MFA warning + cookie flags
- Pengingat otomatis: systemd timer+service harian 07:00 WIB via deploy/reminder/run-reminder.sh (login-CSRF localhost, tolak BASE non-localhost, kredensial di /etc/notaryos/reminder.env 0600, akun layanan office_admin tanpa MFA); idempoten berdampingan dengan ticker in-process + tombol manual; panduan langkah 10 baru di docs/DEPLOYMENT_GUIDE.md.
- Cetak rapi: CSS print global di partials (sembunyikan sidebar/form/tombol, anti-putus baris, margin 12mm) + .no-print di revisi/riwayat akta, breadcrumb kuitansi, halaman docs HowTo, dan konsep blanko PPAT.
- MFA warning: banner dashboard + halaman /mfa untuk peran berhak wasiat (notary/super_admin/substitute_notary/auditor) yang belum enroll TOTP (peringatan saja, tanpa blokir login); halaman MFA tampilkan status Enabled.
- Cookie flags: MaxAge 43200 sesi login/MFA konsisten + HttpOnly + SameSite + Secure di logout, cookie MFA, dan acting-office cookie.
- Drill backup DB+files lolos; audit SEO/responsif; sweep TODO (docs/dms/API_AND_ROUTES.md, docs/integration/CONTINUATION_STATE.md).
## [0.4.4] — 2026-11-23 — SEO dinamis + responsif + audit CSV + strict UX flash + cookie flags
- SEO dinamis: robots (Allow /docs /verify, Disallow /portal /admin) + llms.txt live dari katalog services (tak bisa drift dari sitemap/PSEO); canonical + og:url absolut (fallback notarius.example untuk localhost); FAQ JSON-LD fallback bila syarat kosong.
- Responsif 5 halaman: table-responsive info akta, flex-wrap form ubah-status/invoice, grid col-md form BO, deedview overflow anti-luber.
- Audit CSV: GET /audit/export (guard PermAuditView, filter q opsional, BOM + 1000 baris) + tombol Ekspor CSV di halaman audit.
- Strict UX flash: mode ketat checklist gagal → redirect 303 ke #checklist dengan flash (bukan 400 teks); inquiry→permohonan tolak 409 bila status bukan BARU + scope office_id saat update DIPROSES.
- Keamanan: cookie sesi MaxAge 43200 + flags konsisten di logout/MFA/acting-office; backup DB tambah --no-tablespaces.
- Infra rilis (dipakai ulang dari v0.4.3/v0.4.2): SLA override JSON per kantor + papan SLA + widget 5 permohonan mendesak; booking DISETUJUI → tugas persiapan otomatis; wasiat hardening (gate deed/portal/sidebar/feed); mapping 35 kode layanan + test; migrasi 025 indeks (MySQL-8 safe, idempoten); S3 env fix (kanonis NOTARYOS_S3_* menang); backup BERKAS backup_files.sh; USER_MANUAL + panduan uji.
## [0.4.2] — 2026-11-24 — Pengerasan + SLA + wasiat + infra rilis (di atas 0.4.0, tanpa ubah skema bisnis)
- Layanan & Katalog: mapping 35 kode layanan + test (deedTemplateKeyword/serviceDeedKind/corpEntityOf case-insensitive + trim, alias lama PT-PERUBAHAN/YAYASAN-PENDIRIAN/WASIAT-UMUM, 3 bundel BUNDEL-PT-OSS/CV-NIB/AJB-BALIKNAMA); repair kategori PPAT; papan kesiapan jual per layanan + template konsep (dari v0.4.1, dipakai ulang).
- OCR & DMS: tanpa ubah OCR enterprise 0.4.0 (dipakai ulang); DMS + widget dokumen overdue top-5 di dashboard (OverdueDocs, best-effort, gagal query = widget disembunyikan).
- Akta & Register: wasiat hardening — deedDetail gate kind 'Will' (hanya willReader/notaris + auditor baca), portal kecualikan Will (portalDeedID + daftar akta portal), sidebar /wills hanya deed.approve/auditor; wizard wasiat (+ AJB dipakai ulang); draf 1-klik tidak berubah.
- Keuangan & Laporan: rekap per kategori + top 5 layanan + papan SLA; override SLA per kantor via /office sla_days_json (JSON {"PPAT-AJB": 5, "default": 10}, rusak ditolak 400, kosong = bawaan); ServiceOverdue/ServiceSLAFor honours override (fail-open ke ServiceSLA); dashboard widget 5 permohonan paling mendesak.
- Portal & Notifikasi: booking DISETUJUI → tugas persiapan otomatis di /tasks (best-effort, tanpa ubah skema/status flow); WA reminder H-1 dipakai ulang (hanya bila gateway dikonfigurasi, tanpa klaim otomatis); portal sembunyikan wasiat (lihat Akta & Register).
- Keamanan: PII masking (maskNIKDisp/redactNIKDisp + test, hanya 4 digit terakhir tampil, NIK penuh tak pernah bocor); rate-limit upload 30/mnt + OCR 20/mnt per IP (staf batch aman; loginLimiter tak berubah); wasiat gate (lihat Akta & Register); tanpa ubah auth/RBAC.
- Migrasi DB 025_indeks.sql (idempoten, jalan otomatis saat boot): 5 CREATE INDEX tanpa IF NOT EXISTS (MySQL 8 tak mendukung — duplikat 1061 ditoleransi runner): idx_req_office_status, idx_deed_request, idx_deed_office_status, idx_checkrec_req_status, idx_ppat_office_status. Tanpa ubah kolom/tabel/seed/drop. S3 env fix: kanonis NOTARYOS_S3_* menang, alias S3_* lama tetap jalan. Backup BERKAS: backup_files.sh (rsync + verifikasi count + symlink latest + retensi, SKIP otomatis bila S3; proteksi via versioning + sync off-bucket).
- Go-Live: backup DB + backup BERKAS → deploy biner → restart (migrasi otomatis incl. 025) → cek schema_migrations + SHOW INDEX → uji 1 permohonan (lihat docs/DEPLOYMENT_GUIDE.md).
- Docs: USER_MANUAL baru (alur staf/notaris/keuangan/klien + aktivasi 10 langkah), panduan uji + release notes v0.4.0 + template konsep + kesiapan jual (dari v0.4.1).
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
