# Permission Inventory (code-enforced)
9 role: super_admin, office_admin, notary, substitute_notary, staff, finance, doc_operator, client, auditor.
~38 permission constants di internal/rbac/rbac.go (termasuk website.* granular, audit.view, lcc.view/manage).
DB mirror: migrations/010 + 020 (INSERT IGNORE). Penegakan tetap di kode (rbac.Has per handler / RequirePerm / requireStaff / requireWeb).
Perbaikan: role client default-deny pada requestDetail; audit.view terpisah; corporate/appointment handlers diberi perm+office scope.
