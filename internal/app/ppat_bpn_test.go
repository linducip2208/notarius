package app

import "testing"

// TestPPATBPNState memastikan badge progresif resi BPN: H-7..H-0 kuning,
// lewat jatuh tempo merah, status lain netral. Jatuh tempo = bpn_date + 14
// hari (days = umur sejak tanggal lapor; dueIn = 14 - days).
func TestPPATBPNState(t *testing.T) {
	cases := []struct {
		status        string
		days          int
		dueIn         int
		warn, overdue bool
	}{
		{"LAPOR_BPN", 0, 14, false, false},  // baru lapor
		{"LAPOR_BPN", 7, 7, true, false},    // H-7 kuning
		{"LAPOR_BPN", 11, 3, true, false},   // H-3 kuning (batas auto-tugas)
		{"LAPOR_BPN", 14, 0, true, false},   // H-0 masih kuning
		{"LAPOR_BPN", 15, -1, false, true},  // lewat 1 hari merah
		{"LAPOR_BPN", 30, -16, false, true}, // lewat lama merah
		{"AKTA_JADI", 30, -16, false, false},
		{"SERTIFIKAT_TERBIT", 30, -16, false, false},
		{"DRAFT_AKTA", 0, 14, false, false},
	}
	for _, c := range cases {
		dueIn, warn, overdue := ppatBPNState(c.status, c.days)
		if dueIn != c.dueIn || warn != c.warn || overdue != c.overdue {
			t.Errorf("ppatBPNState(%q,%d) = (%d,%v,%v); mau (%d,%v,%v)",
				c.status, c.days, dueIn, warn, overdue, c.dueIn, c.warn, c.overdue)
		}
	}
}

// TestPPATRemindTitle memastikan judul tugas pengingat deterministik
// (syarat idempoten: tanpa tugas OPEN berjudul sama, jangan duplikat).
func TestPPATRemindTitle(t *testing.T) {
	if got := ppatRemindTitle(12, "A/1"); got != "cek resi BPN #12 (A/1)" {
		t.Fatalf("judul tak terduga: %q", got)
	}
	if ppatRemindTitle(1, "x") == ppatRemindTitle(12, "x") {
		t.Fatal("judul dua register berbeda harus berbeda")
	}
}
