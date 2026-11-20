# USER_ROLES
9 peran (§H master): super_admin, office_admin, notary, substitute_notary, staff,
finance, doc_operator, client, auditor. Matriks izin di `internal/rbac/rbac.go`
(sumber kebenaran) — 19 izin granular: client.view/create/edit, identity.view/verify,
deed.draft/edit/approve, protocol.view, repertorium.manage, finance.view/manage,
template.manage, users.manage, reports.view, office.manage, service.request,
legalization.process, government.submit.
Aturan keras: menu disembunyikan ≠ aman; setiap handler memanggil RequirePerm/cek rbac.
Peran client terisolasi ke datanya sendiri; auditor read-only.
