package plan

import "testing"

func TestTiers(t *testing.T) {
	if !Allowed("PRO", FeatPortalPay) || !Allowed("ENTERPRISE", FeatActing) {
		t.Fatal("tier PRO/ENTERPRISE harus lolos fiturnya")
	}
	if Allowed("LITE", FeatShares) || Allowed("LITE", FeatActing) || Allowed("PRO", FeatActing) {
		t.Fatal("LITE/PRO tidak boleh membuka fitur di atasnya")
	}
	if Allowed("BOGUS", FeatShares) {
		t.Fatal("plan tak dikenal harus fail-closed")
	}
	if Rank("LITE") != 0 || Rank("PRO") != 1 || Rank("ENTERPRISE") != 2 {
		t.Fatal("urutan tier salah")
	}
}
