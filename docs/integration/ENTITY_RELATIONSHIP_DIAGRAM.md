# ERD (schema aktual, ringkas)
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
