package rbac

import "testing"

func TestNotaryCanApproveButStaffCannot(t *testing.T) {
	if !Has(RoleNotary, PermApproveDeed) {
		t.Fatal("notary must approve deeds")
	}
	if Has(RoleStaff, PermApproveDeed) {
		t.Fatal("staff must not approve deeds")
	}
	if !Has(RoleSuperAdmin, "anything.at.all") {
		t.Fatal("super admin wildcard broken")
	}
	if Has(RoleClient, PermViewFinance) {
		t.Fatal("client must not view finance")
	}
	if Has("ghost", PermViewClient) {
		t.Fatal("unknown role must have no permissions")
	}
}

func TestWebsitePermissions(t *testing.T) {
	if !Has(RoleOfficeAdmin, PermWebBranding) || !Has(RoleOfficeAdmin, PermWebDomain) {
		t.Fatal("office_admin harus kelola branding+domain")
	}
	if !Has(RoleNotary, PermWebPagesPublish) || Has(RoleStaff, PermWebPagesPublish) {
		t.Fatal("notary boleh terbitkan; staff tidak")
	}
	if Has(RoleNotary, PermWebDomain) || Has(RoleNotary, PermWebsiteManage) {
		t.Fatal("notary tidak boleh domain/manage")
	}
	if !Has(RoleStaff, PermWebPagesCreate) || Has(RoleFinance, PermWebPagesCreate) {
		t.Fatal("staff boleh buat halaman; finance tidak")
	}
	if Has(RoleClient, PermWebsiteView) {
		t.Fatal("client tidak boleh masuk admin CMS")
	}
	if !Has(RoleAuditor, PermWebAudit) || Has(RoleAuditor, PermWebPagesPublish) {
		t.Fatal("auditor: audit ya, publish tidak")
	}
	if len(Roles()) != 9 {
		t.Fatalf("harus 9 peran, dapat %d", len(Roles()))
	}
}
