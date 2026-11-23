# DATABASE AUDIT 4.0 — Fase 1B (READ-ONLY, migrations 001–029 + 030/031 terbaca)

Sumber: `migrations/*.sql` per 2026-11-24. Total ±108 tabel user (001:7, 002:10,
003:10, 004:10, 006–009:1+1+1+1, 011:16, 013:1, 015:12, 016:3, 017:4, 018:9,
019:3, 020:8, 021:1, 023:1, 026:2, 027:4, 030:2, 031:1).
005 seed data; 010/012/014/024/029 hanya ALTER/INSERT; 025/028 hanya INDEX/kolom.
Legenda: IMPLEMENTED_AND_VERIFIED | IMPLEMENTED_BUT_BROKEN | PARTIALLY_IMPLEMENTED
| UI_ONLY | API_ONLY | NOT_CONNECTED | NOT_IMPLEMENTED | NEEDS_LEGAL_REVIEW.

## 1. Inti & RBAC (001) — IMPLEMENTED_AND_VERIFIED
| Tabel | Kolom kunci | FK | Indeks penting |
|---|---|---|---|
| offices | plan(012), license/quota(017), whatsapp/website(015) | — | PK |
| roles/permissions/role_permissions | name UNIQUE | FK CASCADE dua sisi | PK komposit |
| users | office_id NULLABLE, role_id, mfa_secret, password_changed_at(019) | FK offices SET NULL, roles RESTRICT | idx_users_office, email UNIQUE |
| login_attempts | email, ip, success | TIDAK ADA (by design, pre-auth) | idx_login_email_time |
| audit_logs | actor, action, entity, office_id(028), prev/chain_hash(020) | FK actor SET NULL; office_id TANPA FK | idx_audit_entity/time/office |
Temuan: `audit_logs.office_id` NULL untuk baris lama/sistem — PARTIALLY_IMPLEMENTED
(rantai hash ada kolomnya, verifikasi rantai belum diaudit di sini).

## 2. Klien & Korporat (002+008+016) — IMPLEMENTED_AND_VERIFIED (parsial FK)
- clients: FK office RESTRICT; `reg_number` UNIQUE(office,reg_number)(016); status ENUM
  ACTIVE/INACTIVE/ARCHIVED; pic_user_id FK users. Indeks nama/phone.
- identities: FK client CASCADE, verified_by SET NULL; id_type ENUM KTP/KK/NPWP/
  PASSPORT/OTHER; nik_cipher (enkripsi app) + nik_last4; status(016) UNCHECKED/
  PENDING/VERIFIED/REJECTED/NEEDS_REVISION. INDEX client.
- representatives/companies/shareholders/officers: FK CASCADE ke induk.
  **FK HILANG**: `shareholders.client_id` tanpa FK; `company_officers` tanpa
  tautan ke users/client. Status: IMPLEMENTED_BUT_BROKEN (integritas).
- service_requests: status 9-state (DRAFT…CANCELLED); FK office/client/service
  RESTRICT, assignee SET NULL; idx_req_status/client + idx_req_office_status(025).
- request_events: FK request CASCADE; from/to VARCHAR bebas (bukan ENUM/FK
  → riwayat bisa diisi nilai liar) — PARTIALLY_IMPLEMENTED.
- appointments: FK office RESTRICT, client CASCADE; **`request_id` TANPA FK**.
- user_client_links (006): PK komposit, FK dua sisi — IMPLEMENTED_AND_VERIFIED.
- beneficial_owners (008): FK company CASCADE + edd_note(020).
- client_notes/merges/verifications (016): FK lengkap; merges winner/loser RESTRICT.

## 3. Akta & Register (003+013+020+021+022+026+031)
- deed_drafts: kind ENUM dimodifikasi 3x (003→021→022: +AJB/Hibah/…/Firma/
  Koperasi); status 5-state; `uq_deed_number(office,number)` tapi `number`
  NULLABLE → multi-NULL lolos unik = **risiko nomor ganda** — NEEDS_LEGAL_REVIEW.
  `request_id` dan `template_id` TANPA FK. witness/read/renvoi(020) NULLABLE.
- deed_versions: uq_deed_ver(deed,version) — IMPLEMENTED_AND_VERIFIED.
- signatures (003, FK deed CASCADE) vs deed_signatures (013, **TANPA FK**,
  kolom office_id/deed_id polos) = **tabel ganda makna sama** —
  IMPLEMENTED_BUT_BROKEN (satu harus jadi view/deprecated).
- repertorium: uq_repert(office,entry_no); entry_no manual + ada `repertoriumGaps`
  di kode → celah nomor terdeteksi tapi tidak dikunci transaksi —
  PARTIALLY_IMPLEMENTED, NEEDS_LEGAL_REVIEW.
- protocols: uq_protocol(office,bundle,year) + location/status/holder(020).
- legalizations: uq_leg(office,kind,register); kind +GROSSE(020); deed_id/copy_no
  (020) TANPA FK ke deed_drafts.
- wills: handed_over/to/at(008), review/opening(019), uq_will_register(026,
  NULL-tolerant). restricted=1 default — IMPLEMENTED_AND_VERIFIED.
- templates/clauses/approvals: approvals polimorfik (entity/entity_id) TANPA FK
  (by design); decided/requested TANPA FK ke users.
- ppat_register (021): FK office RESTRICT + deed SET NULL; uq_ppat(office,akta_no);
  jenis 7 + status 4 — IMPLEMENTED_AND_VERIFIED.
- deed_handovers (031): riwayat serah, FK office+deed — IMPLEMENTED_AND_VERIFIED.
- matter_parties (026): uq_matter_party(request,client,role) — IMPLEMENTED_AND_VERIFIED.

## 4. Keuangan (004+019+020+030) — IMPLEMENTED_AND_VERIFIED (parsial)
- services: category ENUM dimodifikasi 3x (002→021→022/024); sla_days(019).
- invoices: uq_invoice_no(office,number); status 5-state; quotation_id(030) FK.
- invoice_lines/payments/expenses: standar; payments.idem_key UNIQUE(020) anti
  double-POST — IMPLEMENTED_AND_VERIFIED; `payments.received_by`,
  `expenses.created_by` TANPA FK.
- quotations/lines (030): uq_quot(office,number); status 5-state.
- bank_mutations/budgets/client_specimens (019): **ketiganya TANPA FK sama sekali**
  (office/client/file hanya BIGINT polos) — IMPLEMENTED_BUT_BROKEN.
- gov_submissions: channel AHU/OSS, status 5-state; follow_up(019); TANPA FK
  ke request — NOT_CONNECTED (subsistem asistensi lepas dari matter).
- tasks: status OPEN/DONE + recurrence/ref(019, ref_type default `''`) —
  ENUM dengan default string kosong = IMPLEMENTED_BUT_BROKEN (validasi).

## 5. DMS (011+014+017) — IMPLEMENTED_AND_VERIFIED (paling rapi FK)
- files: +category/doc_number/verification/workflow/confidentiality(011);
  fk category SET NULL; owner polimorfik (by design); idx_files_owner.
- doc_versions/relations/events/ocr_jobs/search/favorites/shares/archive
  _locations/movements/legal_holds/dispositions/access_logs: SEMUA FK files
  CASCADE + indeks relevan — IMPLEMENTED_AND_VERIFIED.
- doc_relations.related_type diperluas 014 (+`file`) — ENUM versi ganda antar
  file migrasi (011 vs 014) membingungkan audit — PARTIALLY_IMPLEMENTED.
- checklist templates/items/records + uq_checklist_template(029, pasca-cleanup
  12 duplikat PT-DIRIAN) — IMPLEMENTED_AND_VERIFIED.
- approval chains/approvals, custom_fields/values (017): FK lengkap.
- doc_shares token CHAR(32) UNIQUE; doc_access_logs VIEW/DOWNLOAD/SHARE/PRINT.

## 6. CMS/White-label (015+018) — IMPLEMENTED_AND_VERIFIED
- office_branding/website_settings/pages/sections/categories/articles/services/
  faqs/nav/media/usage/publication_history: FK office CASCADE, uq slug per
  office, idx status — rapi. Locale ditambah belakangan (018).
- inquiries (BARU/DIPROSES/SELESAI/SPAM) vs bookings (MENUNGGU/DISETUJUI/
  DITOLAK/BATAL) vs testimonials (DRAFT/PUBLISHED): **3 ENUM status Indonesia
  yang tak selaras makna** — IMPLEMENTED_BUT_BROKEN (unifikasi/ mapping).
- slots/team/gallery/pageviews/versions: TANPA FK eksplisit di potongan terbaca
  (kolom office_id polos) — perlu verifikasi file penuh; sementara
  PARTIALLY_IMPLEMENTED.

## 7. Kepatuhan & Workflow (020+027+028)
- security_incidents/lc_regulations/requirements/checks/findings/reviews/changes:
  SEMUA kolom office_id TANPA FK; lc_checks.requirement/file TANPA FK;
  **0 referensi di `internal/app/*.go`** untuk security_incidents (grep=0) =
  tabel tulis-tanpa-pembaca — NOT_CONNECTED (atau UI_ONLY bila ada UI lain).
- substitute_mandates (4 ref): PARTIALLY_IMPLEMENTED. office_flags (11 ref):
  IMPLEMENTED_AND_VERIFIED. service_workflow_rules (5 ref, seed 21 aturan):
  IMPLEMENTED_AND_VERIFIED. workflow_templates/steps/instances/states (027):
  FK CASCADE penuh + uq — IMPLEMENTED_AND_VERIFIED (engine baru, adopsi
  handler lama belum diaudit → NEEDS_LEGAL_REVIEW sebelum jadi gate akta).
- sessions (007): PK token, FK users CASCADE, idx exp + ip/UA(019) —
  IMPLEMENTED_AND_VERIFIED.

## 8. Prioritas perbaikan (urut)
1. P0 — FK hilang: shareholders.client_id, appointments.request_id,
   deed_drafts.request_id/template_id, legalizations.deed_id, payments.
   received_by, expenses.created_by, bank_mutations/budgets/specimens,
   lc_*, deed_signatures(013). Tambah FK/indeks + backfill.
2. P0 — Unifikasi tabel tanda tangan (003 signatures vs 013 deed_signatures);
   jadikan satu + migrasi data. NEEDS_LEGAL_REVIEW (kekuatan pembuktian).
3. P0 — Penomoran legal: deed number NULLABLE, repertorium entry manual,
   will register NULL-tolerant → kunci transaksi/sequence per office.
   NEEDS_LEGAL_REVIEW (UUJN: repertorium & minuta 25 thn).
4. P1 — ENUM duplikat makna: status request/deed/wills/ppat/legalisasi/files/
   checklist/invoice/gov + 3 ENUM Indonesia CMS + ref_type `''` + related_type
   dua versi. Buat tabel status kanonis/mapping.
5. P1 — Sambungkan yang NOT_CONNECTED: security_incidents (tulis handler +
   UI/retensi) atau tandai deprecated; gov_submissions ↔ request.
6. P2 — Kolom tanpa penulis/pembaca terverifikasi: budgets, bank_mutations
   (rekonsiliasi?), specimens, lc_checks/file_id, office_flags.sla_days_json.
   Audit grep per-kolom sebelum hapus.
7. P2 — audit_logs.actor TANPA FK untuk aktor non-user + office_id TANPA FK:
   dokumentasikan sebagai keputusan desain (alat forensik, bukan relasi).
