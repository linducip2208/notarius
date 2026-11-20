# DATABASE_DESIGN
MySQL 8.x, utf8mb4, InnoDB. 35 domain §G dipetakan ke 30+ tabel di 4 migrasi:
- `001_core`: offices, roles, permissions, role_permissions, users, login_attempts, audit_logs.
- `002_clients`: clients, identities (nik_cipher siap enkripsi), client_representatives,
  services, service_requests, request_events, appointments, companies, shareholders, company_officers.
- `003_deeds`: deed_drafts, deed_versions, repertorium, protocols, legalizations, wills,
  doc_templates, clauses, approvals, signatures.
- `004_ops`: price_items, invoices, invoice_lines, payments, expenses, gov_submissions,
  notifications, files, ai_results, retention_policies (+ seed roles/permissions).
Keuangan DECIMAL(18,2); kode memakai int64 rupiah. Kunci: FK + UNIQUE (nomor akta,
repertorium, register legalisasi, invoice, bendel protokol per kantor/tahun) + indeks
status/waktu. Retensi: minuta 25 thn, repertorium 25 thn, register 10 thn, keuangan 10 thn.
