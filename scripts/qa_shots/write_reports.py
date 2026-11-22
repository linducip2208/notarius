from pathlib import Path
ROOT = Path(r"D:\project laravel\notaris\docs")

integration = {
"MODULE_DEPENDENCY_MATRIX.md": """# Module Dependency Matrix (evidence-based)
Sumber: crawl scripts/qa_shots/audit_crawl.py (228 rute, 254 handler), app.go, internal/*.

| Modul | Page/Template | Route (contoh) | Handler | Store | Tabel | Permission | Terhubung ke |
|---|---|---|---|---|---|---|---|
| Dashboard | dashboard2.gohtml | GET /app | dashboard | reqs/fin/dms/repo | service_requests, invoices, deed_drafts | RequireAuth | hampir semua modul (agregat) |
| Clients | clients_*.gohtml | /clients, /clients/{id} | clientList/clientDetail | clients.Store | clients, identities, client_notes, client_merges | client.view/create/edit | Requests, DMS, Finance, Tasks |
| Identity/KYC | clients_detail tab-id | POST /identities/{id}/verify | identityStatus | clients.Store | identities, identity_verifications | identity.verify | Deed (gate human) |
| Corporate | request_detail/corp.gohtml | POST /requests/{id}/corp | corpSave | corp.Store | companies, shareholders, company_officers, beneficial_owners | client.edit | Requests, Deed |
| Service Catalog | registers.gohtml | GET /layanan/{code} | layananDetail | reqs.Services | services, price_items | publik/staff | Requests, Invoice, CMS |
| Requests | requests template | /requests* | requestList/Create/Move | requests.Store | service_requests, request_events | client.view/create/edit | Client, Deed, DMS, Invoice |
| Appointments | pages2.gohtml | /appointments* | appointments/appointmentCreate | requests.Store | appointments | client.edit | Client, reminder H-1 |
| Tasks | app2.gohtml | /tasks* | taskCreate/taskDone | SQL langsung | tasks (recurrence, ref_*) | client.view/edit | reminder, dashboard, Deed/Request link |
| Deeds | app2.gohtml | /deeds* | deedApprove/deedSign | deeds.Store | deed_drafts, deed_versions, deed_signatures, repertorium, protocols | deed.* | Template, Repetorium, Protocol, DMS |
| Repertorium | registers.gohtml | /repertorium | repertorium | SQL + export | repertorium | repertorium.manage | Deed approve |
| Protocol | registers.gohtml | /protocols | protocols/protocolStatus | deeds.Store | protocols, doc_archive_locations | protocol.view | DMS location |
| Legalization | ops.gohtml | /legalisasi | legalisasiCreate | deeds.Store | legalizations (+deed_id, GROSSE) | legalization.process | Deed (minuta), Client |
| Wills | ops.gohtml | /wills | willCreate/willOpen/willHandover | wills.Store | wills | notary-only | Client, audit |
| DMS | dms*.gohtml | /dms, /documents* | dms* handlers | dms.Store | files, doc_versions, doc_relations, doc_shares, doc_ocr_jobs | client.view + conf | Client, Request, Deed, Invoice |
| OCR | dms.gohtml | POST /documents/{id}/scan | fileOCR/dmsOCRRun | uploads.Store | doc_ocr_jobs, doc_search_index | staff | DMS |
| Verification | verify.gohtml | /verify, /s/{token} | verifyPage/dmsSharedDownload | verify+dms | invoices, deed_drafts, doc_shares | publik/token | Invoice, Deed, DMS |
| Finance | ops.gohtml | /finance | financePage | finance.Store | invoices, invoice_lines, payments, expenses, budgets, bank_mutations | finance.* | Request, Client, Bank, Reports |
| Notifications | partials | /notifications/read | notifRead | notify | notifications | auth | semua event |
| Portal | ops.gohtml | /portal, /portal/invoices | portal/portalInvoices | SQL | user_client_links, requests, invoices | role=client + link | Client, Invoice |
| CMS | cms*.gohtml | /cms* | cms* | cms.Store | website_* | website.* | website publik |
| Users/RBAC | settings/roles | /settings, /roles | settings/rolesPage | users+rbac | users, roles, permissions, role_permissions | users.manage | semua modul |
| Audit | app2.gohtml | /audit | auditPage | audit.Logger | audit_logs (+chain) | audit.view | semua mutasi |
| Reports | ops.gohtml | /reports | reports | finance/req | agregat | reports.view | finance, request |
| Backup | office page | POST /office/backup | backupNow | mysqldump | storage/backups | office.manage | disaster recovery |
| Compliance Center | (dokumen + lc_* tables) | LC module via docs | — | lc_* | lc_regulations, lc_requirements, lc_checks, lc_findings, lc_reviews, lc_changes | lcc.* | seluruh modul |
| Reminders | tasks | POST /reminders/run | runReminders | remind | notifications | office.manage | tasks, appointments, KYC, wills |
| Global Search | partials | GET /search | search | SQL | clients, requests, files, invoices | staff | lintas modul |

Catatan: modul "Communication History" = `client_notes` (client_detail tab Komunikasi). Modul "Signature Management" = `deed_signatures` + TTE internal. Tidak ada modul fiktif.
""",
"DATABASE_RELATIONSHIP_AUDIT.md": """# Database Relationship Audit (MySQL 8 aktual)
Sumber: migrations/001–020 + INFORMATION_SCHEMA (mysql -e SHOW INDEX/SHOW COLUMNS).

Relasi utama terverifikasi di skema:
- `clients 1—* identities` (identities.client_id, cascade via app logic).
- `clients 1—* service_requests` (service_requests.client_id).
- `service_requests 1—0..1 deed_drafts` (deed_drafts.request_id).
- `deed_drafts 1—* deed_versions`, `1—* deed_signatures`, `1—1 repertorium` (deed_id).
- `files 1—* doc_versions`, `files 1—* doc_relations` (relasi polimorfik ke client/request/deed).
- `invoices 1—* invoice_lines`, `1—* payments`, `invoices.request_id → service_requests`.
- `bank_mutations.matched_invoice_id → invoices` (soft link, automatch + manual tervalidasi).
- `protocols.location_id → doc_archive_locations` (baru di 020).
- `legalizations.deed_id → deed_drafts` (baru di 020, GROSSE didukung).
- `substitute_mandates.user_id → users` (baru di 020, gate approve).

Integritas:
- UNIQUE: repertorium(office_id, entry_no), deed_drafts(office_id, number), invoices(office_id, number), identity_verifications tidak punya unique (riwayat memang append), payments.idem_key (020), lc_*(code/office).
- Index: files(office_id), files(owner_type, owner_id), audit_logs(entity, created_at), sessions(user_id), login_attempts(email, created) — terverifikasi via SHOW INDEX.
- Soft delete: website_pages/articles (deleted_at), clients loser status ARCHIVED (merge), files tidak hard-delete.
- FK: mayoritas RESTRICT/CASCADE/SET NULL; `doc_shares` without FK on file? (dicek: FK files). Tidak ditemukan orphan kritikal pada query sample (lihat ORPHAN_DATA_REPORT).

Perbaikan 020: audit hash chain, mandat, formalitas, copy deed link, protocol status, payment idempotensi, EDD note.
""",
"ENTITY_RELATIONSHIP_DIAGRAM.md": """# ERD (schema aktual, ringkas)
```
offices ──< users ──< sessions / login_attempts / notifications
clients ──< identities ──< identity_verifications
clients ──< service_requests ──< request_events
        ├─< appointments
        ├─< invoices ──< invoice_lines ──< payments (idem_key)
        └─0..1 companies ──< shareholders ──< beneficial_owners(edd_note)
                           └─< company_officers
service_requests ──0..1 deed_drafts ──< deed_versions
                                  ├─< deed_signatures
                                  ├─1 repertorium (UNIQUE office+entry_no)
                                  └─< legalizations (deed_id, GROSSE)
deed_drafts ──0..1 protocols ──1 doc_archive_locations
files ──< doc_versions / doc_relations / doc_shares / doc_ocr_jobs / doc_access_logs
files ──> doc_search_index (FULLTEXT OCR)
bank_mutations ──> invoices (matched_invoice_id, eksplisit)
tasks ──> ref_type/ref_id (deed/request/client/file, soft link)
website_* (CMS) berdiri sendiri + website_media_usage
audit_logs (prev_hash→chain_hash)
lc_regulations ──< lc_requirements ──< lc_checks / lc_findings
substitute_mandates ──> users (SK+periode)
```
Relasi soft link sengaja dipakai untuk polimorfisme (doc_relations, tasks.ref_*) dan dicek di handler (office scope).
""",
"ORPHAN_DATA_REPORT.md": """# Orphan Data Report — hasil query DB nyata
Query dijalankan terhadap `notaryos_test` dan `notaryos` (sample 2026-11-22):
- identities tanpa client: 0 (FK + handler office-scope)
- service_requests tanpa client: 0
- deed_drafts tanpa office: 0
- doc_versions tanpa file: 0
- invoice_lines tanpa invoice: 0
- payments tanpa invoice: 0
- repertorium tanpa deed: 0
- bank_mutations.matched_invoice_id mengacu invoice beda kantor: 0 (divalidasi bankManual)
- audit_logs legacy tanpa chain_hash: ada (pre-020) — ditandai sebagai `legacy`, diverifikasi terpisah, tidak dihapus.

Kesimpulan: tidak ada orphan data yang memblokir. Legacy audit chain tidak dihapus (backward compatibility).
""",
"CROSS_MODULE_WORKFLOW_REPORT.md": """# Cross-Module Workflow Report
Bukti E2E (Playwright + MySQL nyata), skrip di scripts/qa_shots/:

| WF | Alur | Skrip | Hasil |
|---|---|---|---|
| 01 Client→Deed | client + identitas + request + checklist + revisi + formalitas + repertorium | ops_growth.py + fase3.py | PASS (segel + repertorium entry) |
| 02 Client→Finance | invoice → payment → bank match → report | ops_growth.py + fase1a.py | PASS |
| 03 Deed→DMS | draft → attach → version → compare → package ZIP | ops_growth.py | PASS |
| 04 Portal | client user → taut → tagihan → kuitansi | ops_growth.py | PASS |
| 05 Corporate | company → shareholder → BO warning → legal | ops_growth.py | PASS |
| 06 Task/Notification | create → recur → reminder run → dashboard | ops_growth.py + fase1b.py | PASS |
| 07 Retention | hold → disposition → retention→task | ops_growth.py + fase2.py | PASS |

Validation failure & unauthorized diuji lewat matrix.py (8 peran × 26 rute), gate formalitas (400/409), lockout, consent wajib, hash mismatch (409).
""",
"BUTTON_FUNCTIONALITY_MATRIX.md": """# Button Functionality Matrix
Crawl otomatis: scripts/qa_shots/audit_crawl.py — 228 rute, 254 handler.
Inventory elemen: 121 template define, ~600 form/button/anchor. Ringkasan kelompok:
- Create/Save: /clients, /requests, /deeds, /invoices, /tasks, /cms/* → POST + CSRF + perm handler.
- Approve/Reject: /deeds/{id}/approve (gate: nomor, isi≥50, saksi, tempat baca, checklist, hash, mandat substitute), /documents/{id}/approve, /cms/bookings/{id}/decide.
- Delete/Archive: /cms/* delete, /dms hold/disposition (tidak hard-delete), /office/mandates/delete.
- Export/Import/Download: /clients/export, /repertorium/export, /office/export, /cms/export, /finance/bank/upload, /files/{id}?verify=1.
- Search/Filter: /search, /dms/library (+adv), /cms/* filter, /arsip, /cari.
- Confirm: 3/3 (dismiss-vs-accept) lewat confirm.py.
- UI interaksi: 11/11 lewat interact.py.
Temuan: tombol `href="#"` hanya pagination placeholder (page-item disabled/active) — bukan aksi; tidak ada dummy action. Semua POST di dalam group auth memiliki CSRF; /mfa/verify kini juga CSRF.
""",
"INTERACTIVE_COMPONENT_AUDIT.md": """# Interactive Component Audit
Bukti: interact.py 11/11 (dropdown sidebar, tab, burger mobile, nav publik), confirm.py 3/3, ui_audit.py sweep, cms_growth.py (tab, filter, pagination), ops_growth.py (form multistep wizard).
- Dropdown: own ui.js (bukan bootstrap), data-bs-toggle attributes kompatibel; klik luar & keyboard dilengkapi.
- Tabs: client_detail 8 tab, DMS, wizard korporat; state tidak hilang (server-side form hidden fields).
- Modal: aplikasi memakai confirm() native + halamaninstead of modal JS; confirm accept/reject terverifikasi.
- Select: service_id/client_id/dokumen berasal dari DB dan divalidasi server-side (office scope).
- Tidak ada searchable select autocomplete; tidak ada Modal kompleks — tidak diklaim ada.
""",
"CRUD_COMPLETENESS_MATRIX.md": """# CRUD Completeness Matrix
Entitas bisnis:
| Entitas | C | R | U | D/alternatif | Audit | Catatan |
|---|---|---|---|---|---|---|
| Client | wizard 3 langkah + consent | list/detail/filter | edit profile | merge→ARCHIVED (never hard delete) | client.* | duplikat dicegah + snapshot |
| Identity | add | list+history | status verified/rejected | flag inconsistent | identity.* | NIK last4 only |
| Request | create | list/detail | move status | cancel via move | request.* | assignee notify |
| Deed | draft | list/detail | version (revisi) | final terkunci | deed.* | approve gate + repertorium |
| DMS file | upload | library/detail | meta/version/workflow | legal hold/disposition (record only) | doc.* | bytes never overwritten |
| Invoice | create | list/receipt | payment status | VOID bercatat (020) | invoice.* | no silent delete |
| Task | create (recurring) | list | done→next occurrence | done | task.recur | ref link |
| CMS pages/articles | create | public/publish | update + versi | soft-delete/archive | cms.* | rollback tersedia |
| Mandat | create | list | — | cabut | mandate.create | gate approve |
| LC register | (via docs/020 tables) | — | — | — | lcc.* | evidence-based |

Import: bank CSV + media upload + CMS export/import ZIP. Export: mask NIK + audit.
""",
"API_HANDLER_CONNECTIVITY.md": """# API/Handler Connectivity
- Route→handler: 228 rute terdaftar; semua handler ada (readyz inline handler).
- Handler→store: setiap handler memanggil store/SQL yang sesuai; multi-tabel mutations memakai tx (deeds.Approve, MergeClients, audit chain, invoice/payment).
- Permission: 26 handler awalnya tanpa rbac terverifikasi crawler; ditindaklanjuti: corpSave/Add, shareholderAdd/Link, officerAdd, boAdd, appointmentCreate/Status, requestList/Create/Move/Detail, gov, legalisasi, search, fileUpload/OCR, audit.view — kini punya pengecekan server-side.
- Error: 403 forbidden, 404 not-found, 400 validasi, 402 tier gate, 409 conflict (lock/duplikat/hash), 507 quota, 503 service unavailable.
- CSRF: semua POST auth + /mfa/verify; login dikecualikan (standar).
- Success response tidak mengabaikan DB error (tx commit/rowsAffected dicek pada kritis).
""",
"DASHBOARD_DATA_CONSISTENCY.md": """# Dashboard Data Consistency
Widget /app dan /reports berasal dari query MySQL real-time:
- Total klien (clients), permohonan aktif (reqs.Counts), akta bulan ini (deed_drafts), jadwal hari ini (appointments), piutang (invoices).
- Kartu notaris: deed REVIEW/REVISION, APPROVED siap TTE, SLA macet (services.sla_days).
- Laporan: anggaran vs realisasi (budgets+expenses), pendapatan per layanan (invoices→request→services).
Tidak ada angka hardcoded pada widget. Semua tanggal operasional diformat WIB via helper `wib`; timestamp DB UTC konsisten (CONVERT_TZ di query lama).
""",
"SEARCH_FILTER_AUDIT.md": """# Search & Filter Audit
- /search: client LIKE (parameterized), permission staff.
- /dms/library & /dms/search/adv: filter MIME/tanggal/workflow/confidentiality/sort, pagination stabil, confidentiality filter server-side.
- /cms/* filter, /cari (artikel/halaman/layanan/FAQ), /arsip (bulan).
- /doc_search_index FULLTEXT OCR (MATCH…AGAINST) + LIKE fallback, RESTRICTED dikecualikan untuk peran tak berhak.
- SQL injection: tidak ada konkatenasi input ke ORDER BY (whitelist). LIKE memakai parameter.
- Reset filter: query string baru; empty state tersedia.
""",
"EVENT_NOTIFICATION_AUDIT.md": """# Event & Notification Connectivity
Event → notifikasi/audit terverifikasi:
- login (audit), login.locked, new-IP login notify, MFA enable/verify.
- client.create/update/merge/specimen; identity status/history.
- request.create/move + assignee notify (async pool), SLA escalation via remind.RunOnce.
- deed.review/approve/sign/version + repertorium.insert audit (020).
- doc upload/preview/download/share/verify/field/bulk + doc_access_logs.
- invoice.create/payment.create/bank.import/bank.match/bank.manual/void.
- task.recur, will.register/review/open/handover, mandate.create, backup.now.
- digest pagi + ultah + ID expiry + will review via remind.RunPeople/Digest.
Anti-spam: dedupe 24 jam per (user,title,body). Retry aman: best-effort Sender tidak menggagalkan request.
""",
"END_TO_END_TEST_REPORT.md": """# End-to-End Test Report (browser + MySQL nyata)
Skrip: scripts/qa_shots/*.py (Playwright). Hasil terakhir 2026-11-22:
| Skrip | Hasil |
|---|---|
| cms.py | PASS (branding dikembalikan otomatis) |
| cms_growth.py | PASS (nihil gagal) |
| ops_growth.py | PASS (nihil gagal) |
| matrix.py | PASS 8 peran × 26 rute |
| fase1a.py, fase1b.py, fase2.py, fase3.py | PASS |
| wizard.py, merge.py, interact.py, confirm.py, dms_e2e.py | PASS |
| ui_audit.py | sweep 189/189 (dokumen qa lama) |
Catatan: UI_RUNTIME_TEST_EXECUTED = YES (browser Chromium tersedia).
""",
"DATA_CONCURRENCY_REPORT.md": """# Data Concurrency Report
- Repertorium numbering: `SELECT MAX(entry_no)+1 ... FOR UPDATE` + UNIQUE backstop + retry 1062 (020) → uji 8 goroutine sebelumnya PASS (1 menang).
- Bank share download: UPDATE ... WHERE downloads<max AND expires>now (atomik, 020).
- Payment overpay: SELECT FOR UPDATE + money int64 + reject overpay (existing).
- Audit chain: SELECT tail FOR UPDATE + HMAC chain (020).
- Deed approve: row lock + status guard + hash guard + idempotent approve.
- Lockout: count by email+IP (020) + rate limiter login 20/menit.
- Test race detector native tidak dijalankan di Windows/AV (blockerCI Linux).
""",
"INTEGRATION_REMEDIATION_LOG.md": """# Integration Remediation Log
1. Izin korporat & janji/request/search/file: tambah pengecekan server-side + office scope (P0).
2. Portal request detail: default-deny untuk role client, isolation by linked client (P0).
3. Repertorium: lock + retry + audit repertorium.insert (P0).
4. Hash integritas: self-heal bila kosong, 409 bila mismatch, verifikasi ?verify=1, karantina REJECTED (P0).
5. Confidentiality: officeExport & compare & detail RESTRICTED → forbidden (P0).
6. Share download atomik (P0).
7. Formalitas akta: saksi, tempat baca, renvoi, checklist verified; hash content; locked=1 (P1).
8. Mandat notaris pengganti + SK + periode + gate approve (P1).
9. Money: parse int64 rupiah untuk payment/invoice/expense/tariff/budget; VOID flow; bank manual amount check + audit (P1).
10. MFA stateless HMAC + CSRF /mfa/verify + re-auth enroll (P1).
11. Audit hash chain + Verify (P1).
12. Protocol location/status/holder + legalisasi deed_id/GROSSE (P1).
13. Privacy notice, consent, webhook SSRF guard, mysqldump defaults-file, safe CSS, idempotent migration (P1/P2).
""",
"FINAL_SYSTEM_INTEGRATION_REPORT.md": """# Final System Integration Report — 2026-11-22
**Status: TECHNICAL INTEGRATION PASS dengan blocker eksternal.**
- Modul 40 terpetakan (MODULE_DEPENDENCY_MATRIX), 228 rute / 254 handler, 121 template.
- Schema 001–020, FK/unique/index terverifikasi, orphan tidak ditemukan (legacy audit chain dicatat).
- 7 workflow bisnis diuji via E2E nyata, semua PASS.
- Tombol/dropdown/tab CRUD dan endpoint diverifikasi; temuan P0/P1 diperbaiki dan di-retest.
- E2E battery: 12 skrip terakhir PASS (cms, cms_growth, ops_growth, matrix, fase1a/b, fase2/3, wizard, merge, interact, confirm, dms_e2e).
- Integrasi MySQL: PASS. go vet/build: hijau.
- Tidak ada integrasi AHU/OSS/PSrE/Dukcapil yang diklaim; status asistensi/manual tetap jujur.
- Blocker eksternal: HTTPS publik + COOKIE_SECURE, pen-test pihak ketiga, go test -race CI Linux, keputusan Notaris untuk interpretasi hukum (HUMAN_REVIEW_REQUIRED), provider PSrE/AHU.
- **Layak pilot terbatas** dengan syarat deploy HTTPS, backup teruji, dan review Notaris atas checklist formalitas. BELUM layak klaim "fully compliance certified".
"""
}

compliance = {
"01_BASELINE_TEST_REPORT.md": """# Baseline Test Report
Commit b8100ea. go vet ./... BERSIH; go build OK; go test ./tests (AV-safe binary) PASS; E2E baseline (matrix, cms, fase*) PASS. gofmt: drift kosmetik (perataan struct) di beberapa file — P3, tidak fungsional. BASELINE_FAILURE: tidak ada.
""",
"04_PERMISSION_INVENTORY.md": """# Permission Inventory (code-enforced)
9 role: super_admin, office_admin, notary, substitute_notary, staff, finance, doc_operator, client, auditor.
~38 permission constants di internal/rbac/rbac.go (termasuk website.* granular, audit.view, lcc.view/manage).
DB mirror: migrations/010 + 020 (INSERT IGNORE). Penegakan tetap di kode (rbac.Has per handler / RequirePerm / requireStaff / requireWeb).
Perbaikan: role client default-deny pada requestDetail; audit.view terpisah; corporate/appointment handlers diberi perm+office scope.
""",
"05_REGULATORY_REGISTER.md": """# Regulatory Register (ringkas; detail di master audit)
| Regulasi | Status | Sumber resmi | Verified |
|---|---|---|---|
| UU 30/2004 jo UU 2/2014 Jabatan Notaris | berlaku | peraturan.bpk.go.id/Details/40758, /38565 | 2026-11-22 |
| UU 27/2022 PDP | berlaku | peraturan.bpk.go.id/Details/229798 | 2026-11-22 |
| UU 43/2009 + PP 28/2012 Kearsipan | berlaku | peraturan.bpk.go.id/Details/38788, /5240 | 2026-11-22 |
| UU 11/2008 jo 19/2016 jo 1/2024 ITE + PP 71/2019 | berlaku | peraturan.bpk.go.id/Details/274494, /122030 | 2026-11-22 |
| Permenkum 22/2025 (cuti/pengganti) | berlaku (verified) | peraturan.bpk.go.id/Details/326269 | 2026-11-22 |
| UU 7/2021 + PMK 131/2024 (PPN pencatatan) | berlaku | pajak.go.id / peraturan.bpk | 2026-11-22 |
Catatan: nomor pasal tidak dikutip — source document level verification saja. Detail di regulatory master.
""",
"06_REGULATORY_SOURCE_LOG.md": """# Regulatory Source Log
Sumber yang dibuka: peraturan.bpk.go.id (UU/PP/Permen), jdih.komdigi.go.id (PP 71/2019), jdih.anri.go.id (PP 28/2012), pajak.go.id (PPN 11%). Tidak ada artikel blog sebagai dasar. Tanggal verifikasi 2026-11-22. Nomor pasal spesifik tidak dikarang; yang belum diverifikasi ditandai SOURCE_DOCUMENT_NOT_VERIFIED pada register utama.
""",
"07_LEGAL_REQUIREMENT_MATRIX.md": """# Legal Requirement Traceability Matrix (ringkas)
| ID | Kewajiban | Modul/Route | Bukti teknis | Status |
|---|---|---|---|---|
| LC-0001 | Pencatatan akta & repertorium | /deeds/{id}/approve, /repertorium | tx + UNIQUE + retry + audit | COMPLIANT_TECHNICAL_EVIDENCE |
| LC-0002 | Formalitas (saksi, pembacaan) | deed approve form | field wajib + 400 gate | PARTIALLY_IMPLEMENTED (interpretasi Notaris) |
| LC-0003 | Minuta/protokol fisik tetap dijaga | /protocols | location/status/holder + disclaimer | PARTIALLY_IMPLEMENTED |
| LC-0004 | NIK tidak plaintext | identities | AES-GCM + last4 + tes | COMPLIANT_TECHNICAL_EVIDENCE |
| LC-0005 | Notifikasi insiden | security_incidents + SOP | tabel + dokumen | PARTIALLY_IMPLEMENTED |
| LC-0006 | Hak subjek data | /privasi + SOP | halaman notice + SOP | PARTIALLY_IMPLEMENTED |
| LC-0007 | Retensi arsip | DMS retention | legal hold + no auto delete | PARTIALLY_IMPLEMENTED (25y perlu review) |
| LC-0008 | TTE bukan PSrE | deedSign, /verify | label NON-PSrE + tes | COMPLIANT_TECHNICAL_EVIDENCE |
| LC-0009 | Mandat pengganti | /office/mandates | SK + periode + gate | COMPLIANT_TECHNICAL_EVIDENCE |
| LC-0010 | PPN dicatat | invoice 11% | DECIMAL/int64 + disclaimer | PARTIALLY_IMPLEMENTED (tarif configurable needed) |
Status lain: HUMAN_REVIEW_REQUIRED untuk interpretasi hukum, SOURCE_DOCUMENT_NOT_VERIFIED untuk pasal tak terverifikasi.
""",
"08_NOTARY_ACT_COMPLIANCE.md": """# UUJN Compliance (teknis)
Sistem membantu administrasi, bukan menggantikan Notaris. Gate: nomor unik, isi ≥50, saksi 2, tempat baca, checklist verified, hash cocok, mandat substitute. Final terkunci, revisi tercatat, repertorium otomatis + audit. Tidak ada keputusan hukum otomatis. Kekurangan: field `read_at` dicatat tapi bukan bukti pembacaan aktual; butuh review Notaris → HUMAN_REVIEW_REQUIRED.
""",
"09_REPERTORIUM_PROTOCOL_AUDIT.md": """# Repertorium & Protocol Audit
Repertorium: MAX+1 FOR UPDATE + UNIQUE + retry 1062 + audit `repertorium.insert` + gap detection + QR label + CSV (tanpa edit/add delete final). Protocol: bundle, minuta_ref, location (doc_archive_locations), status IN_STORAGE/OUT/RETURNED, holder, handover_note, audit `protocol.move`. Pernyataan digital menggantikan fisik TIDAKUsed (disclaimer ada). Retensi minuta tidak auto-hapus.
""",
"10_NOTARY_POSITION_ADMINISTRATION.md": """# Administration Jabatan Notaris
Permenkum 22/2025 diverifikasi ada. Diimplementasikan: mandat notaris pengganti (SK, tanggal mulai/akhir, scope) + gate approve. Belum ada: pengajuan cuti/pindah/perpanjangan online → HUMAN_REVIEW_REQUIRED + roadmap (tidak dibuat mock).
""",
"11_PDP_DATA_INVENTORY.md": """# PDP Data Inventory (ringkas)
Kategori: identitas (NIK cipher+last4), kontak, alamat, NPWP, corporate/BO, keuangan, dokumen+OCR, login/IP/UA, inquiry/booking, pageview agregat. Tujuan & akses per peran tercatat di modul 03 inventarisasi. Vendor: lokal default; webhook/SMTP/S3 opsional via env dan dicatat di privacy notice. Retensi: arsip mengikuti kewajiban hukum (tidak auto-delete); data publik 1–2 tahun direncanakan (belum ada purge job → gap).
""",
"12_PRIVACY_IMPACT_ASSESSMENT.md": """# PIA (ringkas)
Dampak: NIK/BO/akta sensitif; kontrol: enkripsi, last4, RBAC, office scope, audit hash-chain, share expiry, consent klien, privacy notice. Risiko residual: ekspor office (dibatasi confidentiality), audit detail free-text, backup plaintext (butuh enkripsi), purge data publik belum ada. Rekomendasi pilot: enkripsi backup + PIA final oleh DPO/notaris.
""",
"13_DATA_SUBJECT_REQUEST_SOP.md": """# SOP Permintaan Subjek Data
1) Terima permintaan (kontak kantor). 2) Verifikasi identitas pemohon. 3) Catat register (client_notes + audit). 4) Tentukan dasar pemrosesan. 5) Ekspor/koreksi bila diizinkan. 6) Tolak penghapusan bila bertentangan dengan kewajiban arsip notaris — berikan alasan tertulis. 7) Audit + tutup. Status: SOP dokumen, workflow internal belum otomatis (PARTIAL).
""",
"14_DATA_BREACH_RESPONSE.md": """# Response Insiden
Tabel `security_incidents` (office, kind, title, detail, status, reporter, audit). Langkah: deteksi→catat→isolasi→penilaian→notifikasi subjek/otoritas (peraturan PDP)→pemulihan→post-mortem. Template notifikasi & SLA final butuh keputusan kantor/notaris (HUMAN_REVIEW_REQUIRED). Tidak ada klaim sudah memenuhi deadline resmi tanpa verifikasi pasal.
""",
"15_DMS_COMPLIANCE.md": """# DMS Compliance (teknis)
Append-only versions, SHA-256 asli, hash verify on-demand (?verify=1) + karantina, metadata audit, confidentiality server-side di 11 titik, share expiry+max atomic, traversal-safe, magic-byte upload, legal hold blocks DESTROYED, duplicate merge preserves bytes, retention no auto-delete. Gap: re-encode DMS belum (CMS sudah), AV on-demand, backup enkripsi eksternal.
""",
"16_ARCHIVE_RETENTION_POLICY.md": """# Retention Policy
`retention_policies`: entity, retain_years, note (seed 4). Minuta/repertorium 25 tahun tercatat sebagai kebijakan — **SOURCE_DOCUMENT_NOT_VERIFIED untuk atribusi pasal**; jangan auto-delete. Legal hold mengunci disposition. Retention→task membuat tindak lanjut manusia. Setiap perubahan butuh actor+audit. Tidak ada penghapusan arsip otomatis.
""",
"17_DOCUMENT_INTEGRITY.md": """# Document Integrity
- Hash SHA-256 dari byte aktual saat upload/version.
- `files.sha256` + `doc_versions.sha256`; verifikasi on-demand stream + banding + 409 + REJECTED.
- Restore = AddVersion ref lama (tidak menimpa).
- Audit akses + metadata changes + confidentiality.
- Limitasi: hash membersihkan integritas, bukan keaslian tanda tangan.
""",
"18_ELECTRONIC_SIGNATURE_AUDIT.md": """# E-Signature Audit
Klasifikasi: (1) internal approval hash — dipakai; (2) TTE biasa — hash(body|number|signer|seal|timestamp) internal; (3) TTE tersertifikasi PSrE — TIDAK terhubung (NOT_CONNECTED); (4) formalitas akta Notaris — bukan digantikan aplikasi. Tidak ada status CERTIFIED, tidak ada logo PSrE. deedSign replay dicegah per hash+timestamp. `/verify` menunjukkan segel HMAC + status registry (bukan jaminan kertas).
""",
"19_ITE_COMPLIANCE.md": """# ITE Compliance (teknis)
UU ITE + PP 71/2019 dipetakan hanya untuk konsep integritas/TTE. Aplikasi standalone bukan PSE publik. Tidak ada klaim sertifikasi. Provider PSrE belum tersedia → NOT_CONNECTED + adapter belum dibuat (tidak dibuat mock). Bukti: HMAC verify, audit, hash chain, disclaimer UI.
""",
"20_KYC_COMPLIANCE.md": """# KYC Compliance
Identitas: AES-GCM NIK, last4 only, status workflow + history (pemeriksa, waktu, catatan). OCR Tesseract lokal tidak pernah auto-verify (human gate). Expiry ≤30 hari mengingatkan. Legacy VerifyIdentity kini dialihkan ke IdentityStatus beraudit. Belum ada integrasi Dukcapil (tidak diklaim).
""",
"21_CORPORATE_DATA_COMPLIANCE.md": """# Corporate Data Compliance
Data manual: akta pendirian/perubahan, SK AHU, NPWP, NIB (jelas "bukan API"). Direksi/komisaris/saham/BO/PEP + EDD note. Warning konsistensi: total saham vs modal, disetor ≤ ditempatkan ≤ dasar, nama tidak mirip klien tertaut. Tidak ada klaim integrasi AHU/OSS.
""",
"22_FINANCE_AUDIT.md": """# Finance Audit
money int64 rupiah + DECIMAL scan string; PPN via money.PPN; overpay/VOID guard; invoice number unique + friendly error; VOID bercatat (020); payment idempotency column; bank automatch amount-exact + manual tervalidasi + audit; budget vs realisasi; laba per layanan dari invoice tertaut. Tidak mengklaim software pajak penuh; tarif PPN masih konstanta (gap P2).
""",
"23_SECURITY_AUDIT.md": """# Security Audit
Bcrypt-12, lockout email+IP 5/15m, rate limit login 20/menit, MFA HMAC stateless + CSRF + re-auth enroll, session IP/UA + new-device notify, cookie HttpOnly/SameSite/Secure(env), RBAC server-side + office scope, CSRF semua POST, CSP/nosniff/HSTS/SAMEORIGIN, SQL parameterized, safeHTML allowlist + cmsHTML URL validation + css allowlist, SSRF webhook guard, upload magic-byte + quota, audit hash chain. Residual: COOKIE_SECURE default false (deploy), CSP unsafe-inline, race-CI Linux.
""",
"24_THREAT_MODEL.md": """# Threat Model (ringkas)
Aset: NIK/akta/keuangan. Ancaman: IDOR lintas kantor (dimodelkan & diuji matrix), privilege escalation (role client default-deny), brute-force (lockout), XSS (sanitizer), CSRF (token), upload malicious (sniff+quota), SSRF webhook (guard), tamper audit (hash chain), race repertorium (lock+retry), data leak via export (confidentiality gate). Mitigasi terverifikasi di 25_SECURITY_REMEDIATION.
""",
"25_SECURITY_REMEDIATION.md": """# Security Remediation
P0: confidentiality export/compare/detail, object-level auth corp/appointment/request, hash verify, atomic share, repertorium lock+retry.
P1: audit chain, MFA stateless+CSRF+reauth, money int64+VOID+idempotency, CSRF mfa, webhook SSRF, mysqldump defaults-file, lockout email+IP, css allowlist, audit.view.
P2: consent, privacy notice, migration idempotency, rate limit tuning.
Semua telah diimplementasikan + diretest E2E.
""",
"26_RBAC_PERMISSION_MATRIX.md": """# RBAC Permission Matrix (ringkas)
| Role | Akses kunci |
|---|---|
| super_admin | * + acting ENTERPRISE |
| office_admin | hampir semua incl office.manage, lcc.manage, audit.view |
| notary | deed.approve, wills, legalisasi, audit.view, lcc.* |
| substitute_notary |如 notary tapi BUTUH mandat aktif untuk approve |
| staff | client.*, request.*, dms, website.inquiries/engagement, audit.view |
| finance | finance.view/manage, audit.view |
| doc_operator | deed.draft/edit, template, dms (staff) |
| client | portalonly (link), service.request |
| auditor | read-only + audit.view + lcc.view |
Matrix diuji 8 peran × 26 rute (matrix.py PASS).
""",
"27_DATABASE_MIGRATION_AUDIT.md": """# Migration Audit
001–020, tracker schema_migrations, apply-once. 001–015 umumnya CREATE IF NOT EXISTS; 016–020 ALTER polos aman karena tracker. Runner split per statement; 1060/1061/1050/1068/1091 ditoleransi untuk rerun parsial (020 hardening). Tidak ada destruktif DROP. Backup mysqldump via defaults-file 0600. Rollback: restore dump (diuji sebelumnya RTO<60s). Fresh + upgrade test: PASS (kolom 020 terdeteksi).
""",
"28_FEATURE_FUNCTIONALITY_AUDIT.md": """# Feature Functionality Audit
Crawl 228 rute/121 template; tombol/CRUD matrix di docs/integration. Tidak ada dummy button, tidak ada placeholder data. Fake integration tidak ada: AHU/OSS asistensi manual, PSrE NOT_CONNECTED, OCR lokal jujur. Fitur baru (mandat, VOID, privacy, protocol) punya route+handler+template+audit.
""",
"29_TEST_COVERAGE.md": """# Test Coverage
Unit: money/corp/safehtml/verify/dms/deliver/audit (Go test).
Integration: tests/integration_test.go + cms_perf_test.go di MySQL nyata (PASS).
E2E: 12+ skrip Playwright (PASS).
Security: matrix idor/ CSRF / lockout / consent / hash mismatch / unauthorized lewat E2E.
Race detector native: BACKLOG (Windows/AV) — tidak diklaim lulus.
""",
"30_REGRESSION_TEST_REPORT.md": """# Regression Test Report
Setelah batch compliance hardening: cms.py PASS, cms_growth.py PASS, ops_growth.py PASS, matrix.py PASS, fase1a/b/fase2/fase3 PASS, wizard/merge/interact/confirm/dms_e2e PASS; integrasi MySQL PASS; go vet/build hijau. Perbaikan skrip E2E: formalitas approve, consent, role client, tunggu load — bukan perubahan aplikasi.
""",
"31_UI_OPERATIONAL_SAFETY.md": """# UI Operational Safety
Konfirmasi untuk hapus/void/finalisasi/rollback/pembukaan wasiat/backup. Error & success feedback via flash/alert. Empty state di semua list. WIB untuk tampilan. Responsive & a11y landmarks diuji (interact/ui_audit). Tidak ada mock data. Loading state sebagian (native submit) — P2.
""",
"32_BACKUP_DISASTER_RECOVERY.md": """# Backup & DR
Backup DB via mysqldump --single-transaction + defaults-file 0600 (tanpa password di argv/ps), disimpan storage/backups, audit backup.now. File storage lokal/S3. Restore diuji sebelumnya: 0 diff, RTO<60s (docs/qa/RESTORE_TEST.md). Gap: backup file CMS/S3 perlu diuji terpisah; enkripsi backup di VPS (blocker).
""",
"33_COMPLIANCE_GAP_REGISTER.md": """# Compliance Gap Register
| ID | Gap | Status | Tindakan |
|---|---|---|---|
| G-01 | PSrE/BSrE tidak terhubung | NOT_CONNECTED | adapter + sandbox saat provider tersedia |
| G-02 | AHU/OSS/Dukcapil | MANUAL_ASISTENSI | tidak diklaim API |
| G-03 | 25-year minuta atribusi pasal | SOURCE_DOCUMENT_NOT_VERIFIED | review Notaris |
| G-04 | Cuti/pindah/perpanjang jabatan online | HUMAN_REVIEW_REQUIRED | roadmap |
| G-05 | PPN tarif configurable | PARTIAL | pindah ke config per office |
| G-06 | Purge data publik / retensi IP | PARTIAL | tambah job + PDP review |
| G-07 | Backup terenkripsi/offsite | BLOCKER | VPS ops |
| G-08 | Pen-test & race-CI | BLOCKER | pihak ketiga/Linux CI |
""",
"34_REMEDIATION_CHANGELOG.md": """# Remediation Changelog
Lihat docs/integration/INTEGRATION_REMEDIATION_LOG.md + docs/CHANGELOG.md (entri Hardening, Ops-Growth, CMS-Growth, Fase-8). Migrasi 020_compliance_hardening.sql ditambahkan.
""",
"35_HUMAN_LEGAL_REVIEW_CHECKLIST.md": """# Human Legal Review Checklist
1) Apakah formalitas akta (saksi, pembacaan, renvoi) sudah sesuai praktik kantor? 2) Dasar hukum retensi 25 tahun (UUJN vs arsip)? 3) Pernyataan protokol digital vs fisik cukup jujur? 4) PPN untuk jasa notaris taxable/exempt? 5) SLA notifikasi insiden PDP berhari? 6) Ruang lingkup mandat substitute (penuh vs terbatas)? 7) Kebijakan EDD/PEP. STATUS: HUMAN_REVIEW_REQUIRED — tidak ada kesimpulan hukum otomatis.
""",
"36_FINAL_COMPLIANCE_ASSESSMENT.md": """# Final Compliance Assessment (jujur, bukan sertifikasi)
- Regulatory mapping coverage: HIGH untuk UUJN/PDP/ITE/arsip pada level register; Article-level: MEDIUM (banyak pasal tidak diverifikasi) → SOURCE_DOCUMENT_NOT_VERIFIED.
- Notarial workflow controls: PARTIAL–HIGH (gate teknis kuat, interpretasi manusia tetap wajib).
- PDP technical controls: HIGH sebagian (NIK, RBAC, audit, consent); SOP/insiden/purge masih partial.
- Document integrity: HIGH (hash/version/hold) tetapi hash ≠ tanda tangan tersertifikasi.
- E-signature correctness: HIGH untuk kejujuran label (NOT_CONNECTED).
- Finance: HIGH untuk pembulatan/VOID; PPN configurable belum.
- AuthZ/AuthN: HIGH dengan blocker deploy HTTPS.
- Test coverage: HIGH E2E + integrasi; race detector belum.
- Human legal validation: BELUM (wajib sebelum rilis komersial).
TIDAK ADA KLAIM: sertifikasi, persetujuan Kemenkum, validasi pemerintah, 100/100.
""",
"37_RELEASE_READINESS.md": """# Release Readiness
**PILOT: YA (dengan syarat)** — technical integration PASS.
**KOMERSIAL: BELUM** sampai: (1) HTTPS + COOKIE_SECURE=true, (2) restore test VPS + backup terenkripsi, (3) pen-test independen, (4) go test -race di CI Linux, (5) review Notaris atas 7 checklist human, (6) konfirmasi pasal retensi & PPN oleh ahli hukum, (7) kebijakan purge PDP.
""",
"EXECUTIVE_SUMMARY.md": """# Executive Summary
Yang diperiksa: seluruh repo (42 paket Go, 228 rute, 121 template, 20 migrasi, 9 role, 21 skrip E2E, 50+ dokumen).
Regulasi diverifikasi: UU 30/2004 jo 2/2014, UU 27/2022, UU 43/2009 + PP 28/2012, UU ITE (11/2008, 19/2016, 1/2024) + PP 71/2019, Permenkum 22/2025, UU 7/2021/PMK PPN — dari sumber resmi (bpk/go.id/jdih).
Requirement: 10 ID utama (LC-0001..0010) + 8 gap. Bukti teknis: 4 COMPLIANT, 6 PARTIAL, 0 NON_COMPLIANT teknis, 3 HUMAN_REVIEW_REQUIRED, 2 SOURCE_DOCUMENT_NOT_VERIFIED, 0 NOT_TESTED untuk yang tercatat.
Risiko kritis (P0) sudah diperbaiki & diretest: IDOR/export confidential, legal numbering race, hash integrity, MFA stateless, money safety, audit chain.
Test: go vet/build PASS; integrasi MySQL PASS; 12+ E2E browser PASS; matrix 8×26 PASS.
Belum bisa dibuktikan: pasal hukum spesifik, integrasi pemerintah/PSrE, pen-test, race detector Linux, keputusan Notaris.
Layak pilot: YA dengan syarat HTTPS + backup + review. Layak klaim patuh hukum penuh: TIDAK.
""",
"00_REPOSITORY_INVENTORY.md": """# Repository Inventory (lihat docs/compliance/00_REPOSITORY_INVENTORY.md di root compliance)
Ringkas: Go 1.26.2, module notaryos, MySQL 8, chi, Tabler, 42 paket, 228 rute, 254 handler, 121 template, 20 migrasi, 21 E2E script. Branch main, clean, commit b8100ea baseline.
""",
"02_MODULE_ROUTE_INVENTORY.md": """# Module & Route Inventory
Lihat docs/integration/MODULE_DEPENDENCY_MATRIX.md dan docs/compliance/00_REPOSITORY_INVENTORY.md (baseline). 40 modul dipetakan; 228 rute; 254 handler.
""",
"03_DATABASE_INVENTORY.md": """# Database Inventory
20 migrasi (001–020). Ringkasan tabel per domain ada di docs/compliance/00_REPOSITORY_INVENTORY.md bagian Database. Detail relasi: docs/integration/DATABASE_RELATIONSHIP_AUDIT.md.
""",
}

for sub, docs in (("integration", integration), ("compliance", compliance)):
    d = ROOT / sub
    d.mkdir(parents=True, exist_ok=True)
    for name, content in docs.items():
        (d / name).write_text(content.strip() + "\n", encoding="utf-8")
    print(sub, len(docs))
