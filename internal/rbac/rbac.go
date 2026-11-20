// Package rbac implements role-based access control for NOTARYOS.
package rbac

// Roles
const (
	RoleSuperAdmin  = "super_admin"
	RoleOfficeAdmin = "office_admin"
	RoleNotary      = "notary"
	RoleSubstitute  = "substitute_notary"
	RoleStaff       = "staff"
	RoleFinance     = "finance"
	RoleDocOperator = "doc_operator"
	RoleClient      = "client"
	RoleAuditor     = "auditor"
)

// Permissions (granular, enforced server-side).
const (
	PermViewClient      = "client.view"
	PermCreateClient    = "client.create"
	PermEditClient      = "client.edit"
	PermViewIdentity    = "identity.view"
	PermVerifyIdentity  = "identity.verify"
	PermDraftDeed       = "deed.draft"
	PermEditDeed        = "deed.edit"
	PermApproveDeed     = "deed.approve"
	PermViewProtocol    = "protocol.view"
	PermManageRepert    = "repertorium.manage"
	PermViewFinance     = "finance.view"
	PermManageFinance   = "finance.manage"
	PermManageTemplates = "template.manage"
	PermManageUsers     = "users.manage"
	PermViewReports     = "reports.view"
	PermManageOffice    = "office.manage"
	PermRequestService  = "service.request"
	PermLegalize        = "legalization.process"
	PermGovSubmit       = "government.submit"
)

var rolePerms = map[string][]string{
	RoleSuperAdmin: {"*"},
	RoleOfficeAdmin: {
		PermViewClient, PermCreateClient, PermEditClient, PermViewIdentity,
		PermDraftDeed, PermEditDeed, PermViewProtocol, PermManageRepert,
		PermViewFinance, PermManageTemplates, PermManageUsers, PermViewReports,
		PermManageOffice, PermLegalize, PermGovSubmit,
	},
	RoleNotary: {
		PermViewClient, PermCreateClient, PermEditClient, PermViewIdentity, PermVerifyIdentity,
		PermDraftDeed, PermEditDeed, PermApproveDeed, PermViewProtocol, PermManageRepert,
		PermViewFinance, PermManageTemplates, PermViewReports, PermLegalize, PermGovSubmit,
	},
	RoleSubstitute: {
		PermViewClient, PermViewIdentity, PermDraftDeed, PermEditDeed, PermApproveDeed,
		PermViewProtocol, PermViewReports, PermLegalize,
	},
	RoleStaff: {
		PermViewClient, PermCreateClient, PermEditClient, PermViewIdentity,
		PermDraftDeed, PermEditDeed, PermViewReports, PermRequestService,
	},
	RoleFinance: {
		PermViewClient, PermViewFinance, PermManageFinance, PermViewReports,
	},
	RoleDocOperator: {
		PermViewClient, PermDraftDeed, PermEditDeed, PermViewProtocol, PermManageTemplates,
	},
	RoleClient:  {PermRequestService},
	RoleAuditor: {PermViewClient, PermViewFinance, PermViewProtocol, PermViewReports},
}

// Has reports whether role grants permission.
func Has(role, perm string) bool {
	perms, ok := rolePerms[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == "*" || p == perm {
			return true
		}
	}
	return false
}

// Roles returns all defined roles.
func Roles() []string {
	out := make([]string, 0, len(rolePerms))
	for r := range rolePerms {
		out = append(out, r)
	}
	return out
}

// PermissionsOf returns the permission list for a role.
func PermissionsOf(role string) []string {
	return append([]string(nil), rolePerms[role]...)
}
