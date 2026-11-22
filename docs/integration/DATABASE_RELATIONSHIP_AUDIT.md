# Database Relationship Audit (MySQL 8 aktual)
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
