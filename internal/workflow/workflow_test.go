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

func TestValidateTransitionRejectBackward(t *testing.T) {
	steps := fixtureSteps()
	done := map[string]bool{"INTAKE": true, "VERIFY": true}
	// Mundur VERIFY -> INTAKE: target sudah done + posisi mundur → ditolak.
	if err := ValidateTransition(steps, done, "VERIFY", "INTAKE", ChecklistSummary{}, nil); err == nil {
		t.Fatal("mundur ke langkah sebelumnya harus ditolak")
	}
	// Mundur BPN -> VERIFY (posisi mundur walau target belum done) → ditolak.
	if err := ValidateTransition(steps, nil, "BPN", "VERIFY", ChecklistSummary{}, nil); err == nil {
		t.Fatal("mundur dari BPN ke VERIFY harus ditolak")
	}
}

func TestValidateTransitionMultiSkipSkippable(t *testing.T) {
	steps := []Step{
		{Key: "A", SortOrder: 1, Skippable: false},
		{Key: "B", SortOrder: 2, Skippable: true},
		{Key: "C", SortOrder: 3, Skippable: true},
		{Key: "D", SortOrder: 4, Skippable: false},
	}
	done := map[string]bool{"A": true}
	// Lompat A -> D melewati dua langkah skippable berurutan: lolos.
	if err := ValidateTransition(steps, done, "A", "D", ChecklistSummary{}, nil); err != nil {
		t.Fatalf("lompat melewati 2 langkah skippable harus lolos: %v", err)
	}
}

func TestValidateTransitionMultiJumpBlockedByOneRequired(t *testing.T) {
	steps := []Step{
		{Key: "A", SortOrder: 1, Skippable: false},
		{Key: "B", SortOrder: 2, Skippable: true},
		{Key: "C", SortOrder: 3, Skippable: false},
		{Key: "D", SortOrder: 4, Skippable: false},
	}
	done := map[string]bool{"A": true}
	// Lompat A -> D melewati B (skippable) + C (wajib): ditolak karena C.
	if err := ValidateTransition(steps, done, "A", "D", ChecklistSummary{}, nil); err == nil {
		t.Fatal("lompat melewati satu langkah wajib di antara skippable harus ditolak")
	}
}

func TestValidateTransitionStartFromEmpty(t *testing.T) {
	steps := fixtureSteps()
	// Mulai dari awal ("") ke langkah pertama: lolos.
	if err := ValidateTransition(steps, nil, "", "INTAKE", ChecklistSummary{}, nil); err != nil {
		t.Fatalf("mulai dari awal ke INTAKE harus lolos: %v", err)
	}
	// Mulai dari awal langsung ke BPN melewati INTAKE/VERIFY wajib: ditolak.
	if err := ValidateTransition(steps, nil, "", "BPN", ChecklistSummary{}, map[string]bool{"BPN_RECEIPT": true}); err == nil {
		t.Fatal("mulai dari awal langsung ke BPN harus ditolak (melewati wajib)")
	}
}

func TestValidateTransitionRejectDoneTargetAndUnknown(t *testing.T) {
	steps := fixtureSteps()
	done := map[string]bool{"INTAKE": true}
	if err := ValidateTransition(steps, done, "INTAKE", "INTAKE", ChecklistSummary{}, nil); err == nil {
		t.Fatal("target yang sudah done harus ditolak")
	}
	if err := ValidateTransition(steps, nil, "", "TIDAK_ADA", ChecklistSummary{}, nil); err == nil {
		t.Fatal("toKey tak dikenal harus ditolak")
	}
	if err := ValidateTransition(steps, nil, "TIDAK_ADA", "INTAKE", ChecklistSummary{}, nil); err == nil {
		t.Fatal("fromKey tak dikenal harus ditolak")
	}
	if err := ValidateTransition(steps, nil, "INTAKE", "  ", ChecklistSummary{}, nil); err == nil {
		t.Fatal("toKey kosong harus ditolak")
	}
}

func TestNextStepsNilDoneAndUnsorted(t *testing.T) {
	// done nil = semua pending, tetap terurut walau input acak.
	rev := []Step{fixtureSteps()[3], fixtureSteps()[2], fixtureSteps()[1], fixtureSteps()[0]}
	got := NextSteps(rev, nil)
	if len(got) != 4 || got[0].Key != "INTAKE" || got[3].Key != "BPN" {
		t.Fatalf("done nil harus kembalikan semua terurut: %+v", got)
	}
}
