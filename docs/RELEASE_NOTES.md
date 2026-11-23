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
