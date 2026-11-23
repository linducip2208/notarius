package app

import (
	"strings"
	"testing"
)

// TestStandardWorkflowSeeds_Shape memastikan 3 template standar wire
// workflow (027) sesuai kontrak: PT-DIRIAN 9 langkah, PPAT-AJB 7
// (Intake,Pihak,Objek,Resi,Akta,Lapor,Serah), KUASA 5 — murni tanpa DB.
func TestStandardWorkflowSeeds_Shape(t *testing.T) {
	seeds := standardWorkflowSeeds()
	byCode := map[string]workflowSeed{}
	for _, s := range seeds {
		if _, dup := byCode[s.code]; dup {
			t.Errorf("duplikat service_code %q", s.code)
		}
		byCode[s.code] = s
	}
	want := map[string]int{"PT-DIRIAN": 9, "PPAT-AJB": 7, "KUASA": 5}
	for code, n := range want {
		s, ok := byCode[code]
		if !ok {
			t.Errorf("seed workflow %q tidak ditemukan", code)
			continue
		}
		if len(s.steps) != n {
			t.Errorf("seed %q = %d langkah, want %d", code, len(s.steps), n)
		}
		if strings.TrimSpace(s.name) == "" {
			t.Errorf("seed %q nama kosong", code)
		}
		seen := map[string]bool{}
		for i, st := range s.steps {
			if strings.TrimSpace(st.key) == "" || strings.TrimSpace(st.label) == "" {
				t.Errorf("seed %q langkah #%d key/label kosong", code, i)
			}
			if seen[st.key] {
				t.Errorf("seed %q duplikat step_key %q", code, st.key)
			}
			seen[st.key] = true
			if st.requiredRole != "staff" && st.requiredRole != "notary" {
				t.Errorf("seed %q langkah %q required_role=%q (want staff/notary)", code, st.key, st.requiredRole)
			}
		}
	}
	// Urutan kunci sesuai spesifikasi tugas.
	ajb := byCode["PPAT-AJB"]
	var keys []string
	for _, st := range ajb.steps {
		keys = append(keys, st.key)
	}
	if got, wantKeys := strings.Join(keys, ","), "Intake,Pihak,Objek,Resi,Akta,Lapor,Serah"; got != wantKeys {
		t.Errorf("PPAT-AJB = %q, want %q", got, wantKeys)
	}
	// Tiap template wajib memuat langkah notaris (Review/Sahkan/Akta).
	for code, s := range byCode {
		found := false
		for _, st := range s.steps {
			if st.requiredRole == "notary" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("seed %q tanpa langkah notaris", code)
		}
	}
}
