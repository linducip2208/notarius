# NOTARIUS 3.0 — ERD Ringkas (hasil audit migrasi 001–026)

Tanpa diagram raksasa: daftar per domain + relasi kunci. Total ±90
CREATE TABLE unik (klaim 98 Kelebihan hitung akibat ALTER/CREATE ganda —
angka jujur dari `grep CREATE TABLE migrations/*.sql | sort -u`).
Konvensi: `office_id` = skop kantor di hampir semua tabel operasional.

## 1. Inti Matter (026 yang baru ★)

| Tabel | Kunci/relasi |
|---|---|
| `service_requests` (Matter) | FK `client_id→clients`, `service_id→services`, `assignee_id→users`; status workflow |
| ★ `matter_parties` (026) | FK `request_id→service_requests` CASCADE, FK `client_id→clients` CASCADE; unik (request,client,role) |
| ★ `service_workflow_rules` (026) | PK `service_code` (string, TANPA FK ke services — disengaja, longgar); 21 seed |
| `request_events` | jurnal (request_id, from→to, actor) — TANPA FK formal |
| `services` | katalog layanan (code unik: PPAT-AJB, WASIAT, PT-DIRIAN…) |

## 2. Pihak & identitas

`clients` (master) ← `identities`, `identity_verifications`, `client_notes`,
`client_representatives`, `client_specimens`, `user_client_links`,
`shareholders` (+link client), `company_officers`, `beneficial_owners`,
`companies`, `client_merges`. `matter_parties.client_id→clients`.

## 3. Akta & register notaris

`deed_drafts` (FK request_id, office) ← `deed_versions`, `deed_signatures`
(hash+seal; TANPA FK formal). Register: `repertorium` (entry_no/kantor),
`legalizations`, `protocols`, `wills` (★ unik (office_id, register_no), 026),
`signatures`. PPAT terpisah: `ppat_register` (deed_id opsional → deed_drafts).

## 4. Pemerintah (asistensi)

`gov_submissions` (office, status, receipt_no/ref) — tanpa API instansi;
murni pencatatan asistensi AHU/OSS.

## 5. Keuangan

`invoices` ← `invoice_lines`; `payments` (→invoices, TANPA FK formal);
`bank_mutations`, `budgets`, `expenses`, `price_items`, `tariffs` implisit.
Kwintansi via query, bukan tabel sendiri.

## 6. DMS & arsip

`documents/files` ← `doc_versions`, `doc_relations`, `doc_shares`,
`doc_favorites`, `doc_access_logs`, `doc_approvals`←`doc_approval_chains`,
`doc_workflow_events`, `doc_checklist_templates`←`doc_checklist_items`,
`doc_checklist_records`, `doc_categories`, `doc_templates`, `doc_field_values`
←`doc_custom_fields`, `doc_ocr_jobs`, `doc_search_index`, `doc_archive_locations`,
`doc_archive_movements`, `doc_legal_holds`, `doc_dispositions`,
`retention_policies`, `ai_results`, `lc_*` (lc_reviews/requirements/regulations/checks/changes/findings).

## 7. Operasional kantor

`offices`, `office_branding`, `office_flags`, `users`, `roles`,
`permissions`, `role_permissions`, `sessions`, `login_attempts`,
`substitute_mandates`, `appointments`, `tasks`, `notifications`,
`audit_logs` (append-only, TANPA FK — disengaja agar tahan hapus),
`security_incidents`, `reminders` implisit.

## 8. Website/CMS publik

`website_pages`←`website_page_sections`, `website_articles`,
`website_categories`, `website_services`, `website_faqs`, `website_nav_items`,
`website_media`+`website_media_usage`, `website_settings`, `website_team`,
`website_gallery`, `website_testimonials`, `website_slots`, `website_bookings`,
`website_inquiries`, `website_content_versions`,
`website_publication_history`, `website_pageviews`.

## 9. Catatan FK yang hilang (jujur)

- SENGAJA tanpa FK: `audit_logs`, `request_events`, `deed_signatures`,
  `payments`, `service_workflow_rules` (tahan terhadap hapus/arsip; integritas
  di aplikasi, bukan DB).
- RISIKO orphan: `doc_search_index`, `doc_access_logs`, notifikasi bila
  user/dokumen dihapus — belum ada cleanup job terdokumentasi.
- Konsisten: `matter_parties` (baru) SUDAH ber-FK CASCADE dua sisi — pola
  yang seharusnya; tabel lama belum menyusul.
- SQLite dev vs MySQL prod: `IF NOT EXISTS` + `INSERT IGNORE` membuat drift
  skema mungkin; tidak ada checker drift otomatis.
