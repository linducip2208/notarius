package corp

import "testing"

func TestWizardPT(t *testing.T) {
	c := Company{ID: 1, EntityType: "PT", Name: "PT Uji", Domicile: "Jakarta", Authorized: 100, Issued: 50, Paid: 25}
	shs := []Shareholder{{Name: "A", Shares: 50}}
	offs := []Officer{{Role: "DIREKTUR", Name: "D"}}
	steps, warns := WizardProgress(c, shs, offs, 1)
	if len(steps) != 5 {
		t.Fatalf("steps harus 5, dapat %d", len(steps))
	}
	for _, s := range steps {
		if !s.Done {
			t.Fatalf("langkah %s harus selesai", s.Key)
		}
	}
	if len(warns) != 0 {
		t.Fatalf("peringatan tak terduga: %v", warns)
	}
}

func TestWizardCV(t *testing.T) {
	c := Company{ID: 2, EntityType: "CV", Name: "CV Uji", Domicile: "Bandung"}
	offs := []Officer{{Role: "SEKUTU", Name: "S1"}, {Role: "SEKUTU", Name: "S2"}}
	steps, _ := WizardProgress(c, nil, offs, 0)
	byKey := map[string]bool{}
	for _, s := range steps {
		byKey[s.Key] = s.Done
	}
	if !byKey["modal"] {
		t.Fatal("modal CV harus otomatis selesai")
	}
	if !byKey["pemilik"] {
		t.Fatal("2 sekutu harus melengkapi pemilik")
	}
	if byKey["bo"] {
		t.Fatal("BO CV tanpa catatan harus belum selesai")
	}
}

func TestWizardWarnings(t *testing.T) {
	c := Company{ID: 3, EntityType: "PT", Name: "PT W", Domicile: "Sby", Authorized: 100, Issued: 120, Paid: 10}
	_, warns := WizardProgress(c, nil, nil, 0)
	if len(warns) != 2 {
		t.Fatalf("harap 2 peringatan, dapat %v", warns)
	}
}
