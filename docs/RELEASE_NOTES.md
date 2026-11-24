# RELEASE_NOTES — v0.5.17 (2026-11-24) — DRILL RESTORE DIPERBAIKI
Ringkasan: grant DB restore + restore.sh toleran + .gitattributes LF + metadata 0.5.17.
Go-Live: seperti v0.5.16 + drill restore ke notaryos_restore. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.16 (2026-11-24) — SIDEBAR RINGKAS + PUTUSAN P1
Ringkasan: mode ringkas sidebar persisten + 2 putusan P1 terdokumentasi + metadata 0.5.16.
Go-Live: seperti v0.5.15 + cek preferensi sidebar. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.15 (2026-11-24) — INGAT TAB + UPDATE ATOMIK
Ringkasan: tab aktif diingat per halaman (P1) + 7 predikat office atomik pada tulis sensitif + metadata 0.5.15.
Go-Live: seperti v0.5.14 + cek tab bertahan + uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.14 (2026-11-24) — BADGE TUGAS SIDEBAR
Ringkasan: badge jumlah tugas OPEN di menu Tugas sidebar (1 query fail-open; guard halaman tak berubah) + metadata 0.5.14.
Go-Live: seperti v0.5.13 + cek badge Tugas. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.13 (2026-11-24) — KUNCI GUARD TUGAS + AUTO-SCAN CLAMAV
Ringkasan: taskCreate/taskDone ke requireWrite (auditor 403, GET tetap 200) + auto-scan ClamAV best-effort saat unggah (fail-open, tanpa hapus otomatis) + SECURITY_AUDIT temuan 1–4 TERTUTUP + metadata 0.5.13.
Go-Live: seperti v0.5.12 + verifikasi tolak auditor POST tugas + unggah dengan CLAMAV_ADDR, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.12 (2026-11-24) — DASHBOARD SEVEN + KUNCI GUARD TULIS
Ringkasan: dashboard /app gaya referensi Seven (clean-room, tanpa aset berbayar; semua angka query nyata) + guard tulis auditor dikunci di 5 endpoint + bulk-verify wajib PermVerifyIdentity + TestReadOnlyRoles hijau.
Go-Live: seperti v0.5.11 + verifikasi /app (strip minggu, Gantt Hari/Minggu/Bulan, kartu tier) + tolak auditor di endpoint tulis (403), lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.11 (2026-11-24) — TAUTAN SEED ALUR KERJA + DOCS DEPLOY PWA/WORKFLOW
Ringkasan: sejak v0.5.10 ada 2 commit (9a979ef, 4dc11f1; 2026-11-24): (1) empty workflow kini tombol Pasang paket standar → ke /dms/checklists (satu baris pages2.gohtml, tanpa ubah logika/RBAC); (2) DEPLOYMENT_GUIDE +9 — PWA produksi (HTTPS, verifikasi manifest/MIME/ikon, DevTools hijau + Install, SW no-cache + versioning cache) + workflow operasional 027 (seed idempoten, makna ACTIVE/COMPLETED/PENDING→DONE, backup tanpa cron); (3) hapus artefak coverage `wf` nyasar dari v0.5.10.
Go-Live: seperti v0.5.10 + verifikasi tombol seed dari kartu Alur Kerja kosong, cek poin deploy 12–13, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.10 (2026-11-24) — VERIFIKASI TYPO STATUS + MATRIKS LEGAL + TEST WORKFLOW
Ringkasan: NEEDS_DOCS→NEED_DOCS (counter benar); normalisasi target requestMove; klaim rekap SIGNED/ARCHIVED ditolak setelah verifikasi (sudah benar); 4 file docs/legal (tanpa fabrikasi pasal, semua NEEDS_LEGAL_REVIEW); 6 test workflow baru (coverage workflow 95,9%).
Go-Live: seperti v0.5.9 + uji 1 permohonan IN_REVIEW lengkap. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.9 (2026-11-24) — NOTARIUS 4.0 TAHAP 1-2: AUDIT + BPR + GUARD TUGAS
Ringkasan: sejak v0.5.8 ada 1 commit (cc8ef97, 2026-11-24): (1) 7 file audit baca-saja (repo, fitur, DB, API, UI/UX, security, proses bisnis); (2) BPR design — stack dikunci Go+MySQL+Tabler+HTMX, pertahankan fondasi (service_requests/matter_parties/clients/workflowGate/PPAT 2-buku), tanpa Metronic/REST/PPAT-baru; (3) guard requireStaff taskCreate/taskDone (tutup tulis tugas oleh client); (4) empty-state CTA klien + DMS fields/chain; (5) koreksi false alarm (tak ada halaman yatim/tombol buntu/filter mati; office-scope valid); tanpa migrasi baru.
Go-Live: seperti v0.5.8 + verifikasi client ditolak POST /tasks, cek CTA empty-state klien/DMS, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.8 (2026-11-24) — PWA INSTALLABLE + RESPONSIF MOBILE
Ringkasan: sejak v0.5.7 ada 1 commit (b6341c5, 2026-11-24): (1) PWA installable — manifest (start_url /app, standalone, ikon 192/512/maskable + apple-touch) via /manifest.webmanifest + service worker via /sw.js (scope root, no-cache) + head PWA di template app + publik, tanpa ubah auth/RBAC; (2) offline jujur — /offline menjelaskan data di server kantor + SW network-first navigasi (fallback cache lalu /offline), cache-first /assets/*, halaman privat tak di-cache, precache + bersih cache lama; (3) responsif mobile — CSS ≤576px + flex-wrap form aksi permohonan.
Go-Live: seperti v0.5.7 + uji install (Android/iOS), /offline langsung, navigasi saat koneksi mati (harus /offline), pastikan halaman privat tak tersaji dari cache, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.7 (2026-11-24) — BUG KONVERSI KEDALUWARSA TERBALIK + SUITE 24/24 LIVE
Ringkasan: sejak v0.5.6 ada 1 commit (2116fd2, 2026-11-24): (1) bug konversi kedaluwarsa terbalik — convertOnce memakai `? > CURDATE()` sehingga penawaran kedaluwarsa lolos konversi dan penawaran valid justru ditolak; diperbaiki satu baris menjadi `? < CURDATE()`; tertangkap live test (FAIL → PASS), bukan review — pelajaran proses: eksekusi melawan DB live yang membuktikan kebenaran, bukan baca kode; (2) smoke 24/24 rute baru terdaftar tanpa login (tanpa 404/500) + suite integrasi 24/24 PASS 0 FAIL live (naik dari 23/23); (3) test quotation+serah baru 157 baris (expire DRAFT→SENT→EXPIRED, convert→invoice ISSUED/ACCEPTED tertaut, handover mentah + cleanup FK terbalik); (4) docs v0.5.6.
Go-Live: seperti v0.5.6 + uji konversi penawaran masa depan (harus ISSUED) dan lewat tanggal (harus EXPIRED + tolak konversi), lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.6 (2026-11-24) — PENAWARAN KEDALUWARSA OTOMATIS + TANDA TERIMA SERAH CETAK
Ringkasan: sejak v0.5.5 ada 1 commit (b07dc16, 2026-11-24): (1) penawaran kedaluwarsa otomatis — lazy-expiry SENT→EXPIRED office-scoped di titik list /quotations (IsExpired + ExpireOverdueQuotations, DRAFT tak tersentuh, audit agregat quotation.expire satu baris) + unit test 10 kasus; (2) tanda terima serah cetak per baris GET /deeds/{id}/handover/{hid} (guard canViewDeed + office-scope, audit deed.handover.receipt, kode verifikasi teks 16 heks SHA-256, tanpa klaim QR — belum dapat dicek via /verify, cocok manual dengan riwayat serah) + tombol Cetak per baris; (3) docs v0.5.5.
Go-Live: seperti v0.5.5 + buka /quotations untuk picu lazy-expiry (verifikasi SENT lewat tanggal → EXPIRED), cetak 1 tanda terima via tombol Cetak di detail akta, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.5 (2026-11-24) — TUTUP LIFECYCLE AKTA + UJI INTEGRASI LIVE 23/23
Ringkasan: sejak v0.5.4 ada 1 commit (a325833, 2026-11-24): (1) arsip minuta final POST /deeds/{id}/archive SIGNED→ARCHIVED (guard notaris, anti-race, audit deed.archive, kartu Arsip Minuta Final hanya untuk SIGNED, tanpa hapus data/ubah minuta) — menutup lifecycle IN_REVIEW→APPROVED→IN_PROGRESS→SIGNED→ARCHIVED di atas advance auto-step/rantai status, serah terima, penawaran→invoice, export/import, jadwal baca, dan widget serah dari v0.5.3–v0.5.4; (2) migrasi 026+029+030+031 terverifikasi di test DB + suite integrasi 23 PASS 0 FAIL live; (3) docs v0.5.4.
Go-Live: seperti v0.5.4 + arsipkan 1 akta SIGNED → ARCHIVED, verifikasi tolak arsip non-SIGNED (400) + arsip ganda (409), lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.4 (2026-11-24) — RANTAI STATUS + SERAH TERIMA + PENAWARAN→INVOICE
Ringkasan: sejak v0.5.3 ada 3 commit (cabdf99, 670850d, d4bbf98; 2026-11-24): (1) kunci unik template 029 (12 PT-DIRIAN ganda → 1, refresh 8 item + isi ulang 13 permohonan); (2) status otomatis + serah terima 031 (akta sah → IN_PROGRESS, serah → COMPLETED) + penawaran resmi 030 (→ invoice ISSUED PPN 11%, kartu di permohonan); (3) rantai IN_REVIEW→APPROVED→IN_PROGRESS saat akta disahkan + jadwal pembacaan terikat akta (lunak) + widget serah dashboard + rekap bulan di laporan + verifikasi alur docs/ops/ALUR_UJI_REALITA.md (352 baris).
Go-Live: seperti v0.5.3 + pastikan migrasi 029–031 jalan otomatis (cek schema_migrations), verifikasi rantai status + serah 1 akta + penawaran → invoice, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.3 (2026-11-24) — AUTO-STEP WORKFLOW INSTANCES + EXPORT/IMPORT TEMPLATE
Ringkasan: sejak v0.5.2 ada 1 commit (7a0923f, 2026-11-24): (1) auto-step DONE dari kejadian nyata — draf → "draf", sahkan → "sah", checklist lengkap → "syarat", PPAT AKTA_JADI/LAPOR_BPN → "akta"/"lapor", tiap penandaan diaudit workflow.step (idempoten, fail-open); (2) export/import JSON checklist+workflow antar-kantor (idempoten, batas 2MB/100 paket/200 item, kartu Portabilitas + audit + strict flash 303); (3) docs v0.5.2 + TEST_STRATEGY live 23/23 + backfill audit 532 baris.
Go-Live: seperti v0.5.2 + verifikasi auto-step per permohonan (draf/sah/syarat) dan uji export/import JSON antar-kantor, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.2 (2026-11-24) — WORKFLOW INSTANCES LIVE + DOCS RILIS
Ringkasan: sejak v0.5.1 ada 1 commit (29058ed, 2026-11-24): (1) workflow instances live — seed 3 template (PT/AJB/Kuasa) via tombol seed checklist, auto-instance saat permohonan dibuat + kartu Alur Kerja + tombol maju, RBAC per langkah + audit step (fail-open aman); (2) test integrasi workflow 23/23 live (unit baru + tests/workflow_integration_test.go, 23/23 PASS 0 FAIL cleanup bersih); (3) docs v0.5.1 + TEST_STRATEGY live + BACKFILL_AUDIT panduan DBA (138 baris).
Go-Live: seperti v0.5.1 + klik seed checklist untuk pasang 3 template workflow, verifikasi kartu Alur Kerja + tombol maju per permohonan (RBAC per langkah), lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.1 (2026-11-24) — NOTARIUS 3.0 TAHAP 1-4 + WORKFLOW ENGINE + AUDIT OFFICE_ID + CI
Ringkasan: sejak v0.5.0 ada 3 commit (75b89f0, dc8ee0d, 68af938; 2026-11-24): (1) fondasi matter + workflowGate generik — migrasi 026 (matter_parties, service_workflow_rules + seed, wills.register_no unik), multi-pihak di akta, 9 bug audit, SIGNED + bedakan 1062, seed HUTANG/PK/SEWA + Wasiat; (2) BPR + finalisasi — audit 38 paket/277 route/127 template/98 tabel/11 state machine, 8 docs NOTARIUS_3_0_* + label 27 paket/9 konsep, E2E lulus; (3) lanjutan — migrasi 027 (workflow_templates/steps/instances/states + package workflow teruji), migrasi 028 (audit_logs.office_id + index, 12 jalur kritis), CI build/vet/unit + integrasi MySQL non-blocking.
Go-Live: seperti v0.5.0 + pastikan migrasi 026–028 jalan otomatis saat boot (cek schema_migrations), verifikasi gate workflow per layanan + multi-pihak di akta, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.5.0 (2026-11-23) — PRODUKSI PERDANA, 5 PENAHAN GO-LIVE TERTUTUP
Ringkasan: RILIS PRODUKSI PERDANA — 5 penahan go-live tertutup: (1) backup GPG streaming tanpa plaintext + purge file lama + kredensial via env; (2) PPAT gate checklist strict + larang resi fiktif; (3) payment office-scope + test isolasi; (4) race booking serial + versi terkunci + approve bedakan 1062; (5) test jujur — fixture dummy KTP palsu, 22 uji nyata, skip eksplisit + docs CI-vs-lokal.
Go-Live: seperti v0.4.8 + pastikan backup .sql.gz.gpg terbaru bisa di-restore (gpg --decrypt | gunzip | mysql) lalu purge plaintext lama via tombol /office, verifikasi gate PPAT strict + payment scope, lalu uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.4.8 (2026-11-23) — SETUP WIZARD + AUDIT LOGIN + LABEL FORM
Ringkasan: wizard /setup sekali-pakai (kantor + super_admin pertama, aktif hanya bila users kosong, redirect /login selebihnya, rate-limit + transaksi anti-race, fail-closed); audit login gagal / MFA gagal / logout / MFA-sukses + lockout terverifikasi; 12 label form + autocomplete; strict redirect + anchor checklist.
Go-Live: seperti v0.4.6 + untuk instalasi baru buka /setup sekali untuk buat kantor + admin pertama (setelah ada ≥1 user endpoint terkunci otomatis ke /login). Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.4.7 (2026-11-23) — PPAT TERKAIT + SEARCH GLOBAL + EMPTY CTA
Ringkasan: kartu register PPAT di detail permohonan (JOIN deed_drafts + scope kantor); search global tambah hasil PPAT (akta/jenis/para pihak/sertifikat) + tugas; 6 empty-state dapat CTA (seed 1-klik, anchor form); docs CHANGELOG + RELEASE_NOTES v0.4.6.
Go-Live: seperti v0.4.6 + verifikasi kartu PPAT terkait di detail permohonan dan search PPAT/tugas. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.4.6 (2026-11-23) — AKTIVASI KANTOR + FILTER + AUDIT SCOPE
Ringkasan: kartu Aktivasi Kantor x/5 di /office (checklist, template + approved, tarif/uji transaksi, 1 permohonan, mode ketat), filter status permohonan + filter status/jenis akta, badge jatuh tempo invoice LEWAT N hari; 8 kebocoran office-scope ditutup (audit feed/ekspor, booking, spesimen DMS, deed detail/timeline, invoice_lines) + template yatim dihapus; dipakai ulang: SEO dinamis + responsif + audit CSV + strict UX flash, timer pengingat 07:00 WIB, CSS print, MFA warning, cookie flags, backup DB+files; .env.example 49 kunci.
Go-Live: seperti v0.4.5 + verifikasi kartu aktivasi di /office mencapai 5/5 setelah ⚡ Pasang Standar di /dms/checklists → ⚡ Pasang 8 Konsep di /templates → Approve template → uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.4.5 (2026-11-23) — TIMER PENGINGAT + CETAK RAPI + MFA WARNING
Ringkasan: pengingat otomatis harian 07:00 WIB (systemd timer+service via skrip login-CSRF localhost, akun layanan office_admin tanpa MFA, idempoten dengan ticker + tombol manual), CSS print global + .no-print (akta, kuitansi, docs, blanko PPAT), banner MFA untuk peran berhak wasiat yang belum enroll, cookie flags konsisten (MaxAge + HttpOnly + SameSite + Secure), drill backup DB+files lolos, audit SEO/responsif, sweep TODO.
Go-Live: seperti v0.4.4 + pasang timer pengingat (langkah 10 docs/DEPLOYMENT_GUIDE.md: akun layanan, /etc/notaryos/reminder.env 0600, enable timer, verifikasi journalctl + /audit memuat reminders.run). Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.4.4 (2026-11-23) — SEO DINAMIS + RESPONSIF + AUDIT CSV + STRICT UX FLASH
Ringkasan: SEO dinamis (robots + llms.txt live dari katalog, canonical/og absolut, FAQ fallback), responsif 5 halaman (table-responsive, flex-wrap, grid BO, deedview anti-luber), audit CSV (/audit/export + tombol Ekspor, guard PermAuditView), strict UX flash (mode ketat redirect ke #checklist, inquiry guard 409 + scope kantor), cookie flags konsisten, backup DB --no-tablespaces; infra rilis dipakai ulang (SLA override + papan + widget, booking → tugas, wasiat hardening, mapping 35 kode, migrasi 025, S3 env fix, backup berkas, USER_MANUAL + panduan uji).
Go-Live: backup DB + backup BERKAS → deploy biner → restart (migrasi otomatis incl. 025) → cek schema_migrations + SHOW INDEX → ⚡ Pasang Standar di /dms/checklists → ⚡ Pasang 8 Konsep di /templates → Approve template → uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.4.2 (2026-11-24) — PENGERASAN + SLA + WASIAT + INFRA RILIS
Ringkasan: di atas 0.4.0 (tanpa ubah skema bisnis): mapping 35 kode layanan + test (case-insensitive + alias lama + 3 bundel), dashboard + laporan (top 5, rekap kategori, papan SLA + override JSON per kantor), wasiat hardening (gate deed/portal/sidebar, wizard wasiat), booking → tugas persiapan otomatis, WA reminder H-1 (bila gateway ada), PII masking + rate-limit upload/OCR, migrasi 025 indeks (MySQL-8 safe, idempoten), S3 env fix, backup berkas, repair kategori PPAT, USER_MANUAL baru + panduan uji + template konsep + kesiapan jual.
Go-Live: backup DB + backup BERKAS → deploy biner → restart (migrasi otomatis incl. 025) → cek schema_migrations + SHOW INDEX → ⚡ Pasang Standar di /dms/checklists → ⚡ Pasang 8 Konsep di /templates → Approve template → uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG + backup berkas teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.4.0 (2026-11-23) — OCR ENTERPRISE + DMS LENGKAP + PPAT
Ringkasan: OCR enterprise (quality gate, preprocessing, voting multi-engine, parser KTP/KK/NPWP, cek konsistensi), DMS (preview universal, dipakai-di-mana, verifikasi 2-kolom, SLA + retensi, seed 23 paket checklist, mode ketat), Todo↔akta (preview + edit), deed preview semua draf + jenis Firma/Koperasi + 7 jenis PPAT, register PPAT + blanko + resi SLA, portal timeline, finance DP + tarif otomatis, laporan (rekap kategori, top 5, SLA), wizard korporat/AJB/wasiat, request form pintar + optgroup, draf 1-klik, publik (layanan detail + bundel 3 paket), PII masking, notifikasi WA manual, migrasi 021–024 (idempoten, jalan otomatis saat boot), docs (8 langkah + checklist cetak + panduan uji).
Go-Live: backup DB → deploy biner → restart (migrasi otomatis) → klik ⚡ Pasang Standar di /dms/checklists → ⚡ Pasang 8 Konsep di /templates → Approve template → uji 1 permohonan. Detail: docs/DEPLOYMENT_GUIDE.md.
Batasan produksi tetap: wajib tinjauan notaris (B-06) + TLS publik + backup GPG teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius

# RELEASE_NOTES — v0.3.0 (2026-11-20) — FINAL INTERNAL, SIAP TINJAU NOTARIS
NOTARYOS 100%: 28/28 fitur COMPLETED. Go + MySQL + Tabler.
Yang baru di 0.3.0: deed Review/Sahkan + repertorium otomatis, beneficial owner TPPU, serah-terima wasiat, checklist AHU/OSS per-item, HSTS, session SQL + cleanup, seed idempoten, migrasi 001-008.
Cara coba: siapkan MySQL 8, `make migrate`, `make seed-admin` (env SEED_*), `go run ./cmd/notaryos`, buka /login.
Batasan produksi: wajib tinjauan notaris (B-06) + TLS publik + backup GPG teruji + pen-test sebelum data sungguhan.
Repo: https://github.com/linducip2208/notarius
