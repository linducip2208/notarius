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
