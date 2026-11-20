# DATABASE_ERD (tekstual)
```
offices 1──* users *──1 roles *──* permissions (role_permissions)
offices 1──* clients 1──* identities, client_representatives
clients *──* service_requests (client_id) ; services 1──* service_requests
service_requests 1──* request_events ; 1──* appointments ; 1──* companies
companies 1──* shareholders, company_officers
service_requests 1──* deed_drafts 1──* deed_versions, signatures
deed_drafts 1──* repertorium, protocols ; doc_templates 1──* deed_drafts
clients 1──* legalizations, wills ; offices 1──* legalizations, wills
clients 1──* invoices 1──* invoice_lines, payments ; offices 1──* expenses
offices 1──* gov_submissions, files, notifications ; files 1──* ai_results
users 1──* audit_logs(actor), approvals ; retention_policies (acuan)
```
Isolasi: hampir semua tabel membawa `office_id`; akses lintas kantor ditolak di query.
