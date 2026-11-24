package app

import (
	"strings"
	"testing"
)

// TestStandardWorkflowSeeds_SLA memastikan tiap langkah 3 template standar
// punya target sla_days wajar (>0) dan total per template <= 21 hari
// (migrasi 033): Intake 1, verifikasi/data 2, draf 3, review/sah 2, dst.
func TestStandardWorkflowSeeds_SLA(t *testing.T) {
	for _, s := range standardWorkflowSeeds() {
		total := 0
		for i, st := range s.steps {
			if st.slaDays <= 0 {
				t.Errorf("seed %q langkah #%d %q sla_days=%d (want >0)", s.code, i, st.key, st.slaDays)
			}
			if st.slaDays > 7 {
				t.Errorf("seed %q langkah %q sla_days=%d (want <=7 per langkah)", s.code, st.key, st.slaDays)
			}
			total += st.slaDays
		}
		if total > 21 {
			t.Errorf("seed %q total SLA=%d hari (want <=21)", s.code, total)
		}
		if total <= 0 {
			t.Errorf("seed %q total SLA=%d (want >0)", s.code, total)
		}
	}
}

// TestWorkflowEscalationTitle_Stability memastikan judul tugas eskalasi
// stabil (tanpa umur hari) sehingga dedup 1x via COUNT(title) tetap cocok
// keesokan harinya — pola yang sama dengan SLA dokumen & retensi.
func TestWorkflowEscalationTitle_Stability(t *testing.T) {
	a := wfEscalationTitle(42, "Resi", "Cek bersih & resi")
	b := wfEscalationTitle(42, "Resi", "Cek bersih & resi")
	if a != b || strings.TrimSpace(a) == "" {
		t.Fatalf("judul eskalasi tidak stabil: %q vs %q", a, b)
	}
	if strings.Contains(a, "hari") {
		t.Errorf("judul eskalasi %q memuat umur hari — dedup 1x akan jebol", a)
	}
	if got := wfEscalationTitle(7, "Intake", "Penerimaan"); got == a {
		t.Errorf("judul request berbeda harus beda: %q", got)
	}
}
