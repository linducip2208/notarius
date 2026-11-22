# Module Dependency Matrix (evidence-based)
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
