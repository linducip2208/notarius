# RBAC Permission Matrix (ringkas)
| Role | Akses kunci |
|---|---|
| super_admin | * + acting ENTERPRISE |
| office_admin | hampir semua incl office.manage, lcc.manage, audit.view |
| notary | deed.approve, wills, legalisasi, audit.view, lcc.* |
| substitute_notary |sama seperti notary tapi BUTUH mandat aktif untuk approve |
| staff | client.*, request.*, dms, website.inquiries/engagement, audit.view |
| finance | finance.view/manage, audit.view |
| doc_operator | deed.draft/edit, template, dms (staff) |
| client | portalonly (link), service.request |
| auditor | read-only + audit.view + lcc.view |
Matrix diuji 8 peran × 26 rute (matrix.py PASS).
