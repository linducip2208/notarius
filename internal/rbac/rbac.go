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
	PermWebsiteView     = "website.view"
	PermWebsiteManage   = "website.manage"
	PermWebPagesView    = "website.pages.view"
	PermWebPagesCreate  = "website.pages.create"
	PermWebPagesUpdate  = "website.pages.update"
	PermWebPagesDelete  = "website.pages.delete"
	PermWebPagesPublish = "website.pages.publish"
	PermWebArticles     = "website.articles.manage"
	PermWebNav          = "website.navigation.manage"
	PermWebMedia        = "website.media.manage"
	PermWebSEO          = "website.seo.manage"
	PermWebBranding     = "website.branding.manage"
	PermWebDomain       = "website.domain.manage"
	PermWebAudit        = "website.audit.view"
	PermWebInquiries    = "website.inquiries.manage"
	PermWebEngagement   = "website.engagement.manage"
	PermAuditView       = "audit.view"
	PermLCCView         = "lcc.view"
	PermLCCManage       = "lcc.manage"
)

var rolePerms = map[string][]string{
	RoleSuperAdmin: {"*"},
	RoleOfficeAdmin: {
		PermViewClient, PermCreateClient, PermEditClient, PermViewIdentity,
		PermDraftDeed, PermEditDeed, PermViewProtocol, PermManageRepert,
		PermViewFinance, PermManageTemplates, PermManageUsers, PermViewReports,
		PermManageOffice, PermLegalize, PermGovSubmit,
		PermWebsiteView, PermWebsiteManage, PermWebPagesView, PermWebPagesCreate,
		PermWebPagesUpdate, PermWebPagesDelete, PermWebPagesPublish, PermWebArticles,
		PermWebNav, PermWebMedia, PermWebSEO, PermWebBranding, PermWebDomain, PermWebAudit,
		PermWebInquiries, PermWebEngagement, PermAuditView, PermLCCView, PermLCCManage,
	},
	RoleNotary: {
		PermViewClient, PermCreateClient, PermEditClient, PermViewIdentity, PermVerifyIdentity,
		PermDraftDeed, PermEditDeed, PermApproveDeed, PermViewProtocol, PermManageRepert,
		PermViewFinance, PermManageTemplates, PermViewReports, PermLegalize, PermGovSubmit,
		PermWebsiteView, PermWebPagesView, PermWebPagesCreate, PermWebPagesUpdate,
		PermWebPagesPublish, PermWebArticles, PermWebMedia,
		PermWebInquiries, PermWebEngagement, PermAuditView, PermLCCView, PermLCCManage,
	},
	RoleSubstitute: {
		PermViewClient, PermViewIdentity, PermDraftDeed, PermEditDeed, PermApproveDeed,
		PermViewProtocol, PermViewReports, PermLegalize,
		PermWebsiteView, PermWebPagesView, PermWebPagesCreate, PermWebPagesUpdate,
		PermWebPagesPublish, PermWebArticles, PermWebMedia,
		PermWebInquiries, PermWebEngagement, PermAuditView, PermLCCView, PermLCCManage,
	},
	RoleStaff: {
		PermViewClient, PermCreateClient, PermEditClient, PermViewIdentity,
		PermDraftDeed, PermEditDeed, PermViewReports, PermRequestService,
		PermWebsiteView, PermWebPagesView, PermWebPagesCreate, PermWebPagesUpdate, PermWebMedia,
		PermWebArticles,
		PermWebInquiries, PermWebEngagement, PermAuditView,
	},
	RoleFinance: {
		PermViewClient, PermViewFinance, PermManageFinance, PermViewReports,
		PermWebsiteView, PermAuditView,
	},
	RoleDocOperator: {
		PermViewClient, PermDraftDeed, PermEditDeed, PermViewProtocol, PermManageTemplates,
		PermWebsiteView,
	},
	RoleClient:  {PermRequestService},
	RoleAuditor: {PermViewClient, PermViewFinance, PermViewProtocol, PermViewReports, PermWebsiteView, PermWebAudit, PermAuditView, PermLCCView},
}

// ReadOnly reports whether role must never mutate operational data.
// Auditor and client are read-only by design: auditor supervises,
// client acts only via service requests and portal payments.
func ReadOnly(role string) bool {
	return role == RoleAuditor || role == RoleClient
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
