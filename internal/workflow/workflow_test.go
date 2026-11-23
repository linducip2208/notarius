package workflow

import "testing"

func fixtureSteps() []Step {
	return []Step{
		{Key: "INTAKE", Label: "Penerimaan", SortOrder: 1, RequireChecklist: false, Skippable: false},
		{Key: "VERIFY", Label: "Verifikasi", SortOrder: 2, RequireChecklist: true, Skippable: false},
		{Key: "OPTIONAL_REVIEW", Label: "Telaah opsional", SortOrder: 3, RequireChecklist: false, Skippable: true},
		{Key: "BPN", Label: "Lapor BPN", SortOrder: 4, RequireChecklist: false, RequireEvidence: "BPN_RECEIPT", Skippable: false},
	}
}

func TestNextStepsOrderAndFilter(t *testing.T) {
	got := NextSteps(fixtureSteps(), map[string]bool{"INTAKE": true})
	if len(got) != 3 || got[0].Key != "VERIFY" || got[1].Key != "OPTIONAL_REVIEW" || got[2].Key != "BPN" {
		t.Fatalf("urutan pending salah: %+v", got)
	}
}

func TestNextStepsAllDoneEmpty(t *testing.T) {
	done := map[string]bool{"INTAKE": true, "VERIFY": true, "OPTIONAL_REVIEW": true, "BPN": true}
	if got := NextSteps(fixtureSteps(), done); len(got) != 0 {
		t.Fatalf("harus kosong, dapat %+v", got)
	}
}

func TestIsStepBlockedChecklist(t *testing.T) {
	verify := Step{Key: "VERIFY", RequireChecklist: true}
	if !IsStepBlocked(verify, ChecklistSummary{Missing: 2}) {
		t.Fatal("MISSING>0 harus memblokir langkah wajib checklist")
	}
	if !IsStepBlocked(verify, ChecklistSummary{Invalid: 1}) {
		t.Fatal("Invalid>0 harus memblokir langkah wajib checklist")
	}
	if IsStepBlocked(verify, ChecklistSummary{}) {
		t.Fatal("checklist bersih tidak boleh memblokir")
	}
}

func TestIsStepBlockedNoRequirement(t *testing.T) {
	free := Step{Key: "INTAKE", RequireChecklist: false}
	if IsStepBlocked(free, ChecklistSummary{Missing: 9, Invalid: 9}) {
		t.Fatal("langkah tanpa syarat checklist tidak boleh diblokir")
	}
}

func TestValidateTransitionSequentialOK(t *testing.T) {
	steps := fixtureSteps()
	done := map[string]bool{"INTAKE": true}
	if err := ValidateTransition(steps, done, "INTAKE", "VERIFY", ChecklistSummary{}, nil); err != nil {
		t.Fatalf("maju berurutan harus lolos: %v", err)
	}
}

func TestValidateTransitionRejectJumpOverRequired(t *testing.T) {
	steps := fixtureSteps()
	// Lompat INTAKE -> BPN melewati VERIFY (wajib) dan OPTIONAL_REVIEW (skippable).
	if err := ValidateTransition(steps, nil, "INTAKE", "BPN", ChecklistSummary{}, map[string]bool{"BPN_RECEIPT": true}); err == nil {
		t.Fatal("lompat melewati langkah wajib harus ditolak")
	}
}

func TestValidateTransitionAllowSkipSkippable(t *testing.T) {
	steps := fixtureSteps()
	done := map[string]bool{"INTAKE": true, "VERIFY": true}
	// VERIFY -> BPN melewati OPTIONAL_REVIEW yang skippable: lolos bila bukti ada.
	if err := ValidateTransition(steps, done, "VERIFY", "BPN", ChecklistSummary{}, map[string]bool{"BPN_RECEIPT": true}); err != nil {
		t.Fatalf("skip langkah skippable harus lolos: %v", err)
	}
}

func TestValidateTransitionRejectBlockedAndMissingEvidence(t *testing.T) {
	steps := fixtureSteps()
	done := map[string]bool{"INTAKE": true}
	if err := ValidateTransition(steps, done, "INTAKE", "VERIFY", ChecklistSummary{Missing: 1}, nil); err == nil {
		t.Fatal("target terblokir checklist harus ditolak")
	}
	done2 := map[string]bool{"INTAKE": true, "VERIFY": true, "OPTIONAL_REVIEW": true}
	if err := ValidateTransition(steps, done2, "OPTIONAL_REVIEW", "BPN", ChecklistSummary{}, nil); err == nil {
		t.Fatal("bukti BPN_RECEIPT yang hilang harus ditolak")
	}
}
