// Package workflow adalah fondasi Workflow Engine generik: tipe data
// Template/Step/Instance + fungsi murni yang bisa di-unit-test tanpa DB.
//
// SENGAJA tanpa database/sql dan tanpa import internal lain: seluruh aturan
// urutan, blokir syarat, dan skip kondisional diekspresikan sebagai fungsi
// murni atas slice/map agar deterministik. Handler existing (workflowGate di
// internal/app/wf_rules.go) tidak disentuh; paket ini belum dipakai handler
// mana pun (tahap fondasi).
package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// Step adalah satu langkah dalam template workflow.
type Step struct {
	Key              string
	Label            string
	SortOrder        int
	RequiredRole     string
	RequireChecklist bool
	RequireEvidence  string
	Skippable        bool
}

// Template adalah cetakan workflow per layanan per kantor per versi.
type Template struct {
	ID          int64
	OfficeID    int64
	ServiceCode string
	Name        string
	Version     int
	Active      bool
	Steps       []Step
}

// Instance adalah workflow berjalan untuk satu service_request.
type Instance struct {
	ID          int64
	OfficeID    int64
	RequestID   int64
	TemplateID  int64
	CurrentStep string
	Status      string
}

// ChecklistSummary adalah ringkasan syarat checklist untuk satu langkah.
// Missing = item berstatus MISSING; Invalid = REJECTED/NEEDS_REVISION/dll.
type ChecklistSummary struct {
	Missing int
	Invalid int
}

// ordered mengembalikan salinan steps terurut SortOrder (lalu Key).
func ordered(steps []Step) []Step {
	out := append([]Step(nil), steps...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// NextSteps mengembalikan langkah yang belum selesai (done[key] != true)
// dalam urutan eksekusi. done boleh nil (= semua pending).
func NextSteps(steps []Step, done map[string]bool) []Step {
	var out []Step
	for _, s := range ordered(steps) {
		if done != nil && done[s.Key] {
			continue
		}
		out = append(out, s)
	}
	return out
}

// IsStepBlocked melaporkan apakah langkah tertahan syarat checklist:
// langkah yang mewajibkan checklist (RequireChecklist) diblokir bila masih
// ada item MISSING atau Invalid. Langkah tanpa syarat tidak pernah diblokir.
func IsStepBlocked(step Step, summary ChecklistSummary) bool {
	if !step.RequireChecklist {
		return false
	}
	return summary.Missing > 0 || summary.Invalid > 0
}

// ValidateTransition memvalidasi perpindahan fromKey -> toKey atas template
// steps dengan status done saat ini. fromKey "" berarti mulai dari awal.
// evidence memetakan jenis bukti (RequireEvidence) ke ketersediaannya;
// nil berarti tidak ada bukti tersedia.
//
// Aturan:
//  1. toKey harus ada di template; langkah yang sudah done tak bisa dituju ulang.
//  2. Lompat maju hanya boleh melewati langkah Skippable; langkah wajib
//     yang dilewati menolak transisi.
//  3. fromKey selain "" harus langkah yang dikenal (sudah done atau langkah
//     pending paling depan — toleran untuk pemanggil stateless).
//  4. Target yang IsStepBlocked menolak transisi.
//  5. Target dengan RequireEvidence mewajibkan evidence[jenis] == true.
func ValidateTransition(steps []Step, done map[string]bool, fromKey, toKey string, summary ChecklistSummary, evidence map[string]bool) error {
	if strings.TrimSpace(toKey) == "" {
		return fmt.Errorf("toKey kosong")
	}
	ord := ordered(steps)
	idx := map[string]int{}
	for i, s := range ord {
		idx[s.Key] = i
	}
	toIdx, ok := idx[toKey]
	if !ok {
		return fmt.Errorf("langkah tujuan %q tidak dikenal", toKey)
	}
	if done != nil && done[toKey] {
		return fmt.Errorf("langkah %q sudah selesai", toKey)
	}
	if fromKey != "" {
		if _, ok := idx[fromKey]; !ok {
			return fmt.Errorf("langkah asal %q tidak dikenal", fromKey)
		}
	}
	// Posisi awal: setelah fromKey (atau -1 bila mulai dari awal).
	fromIdx := -1
	if fromKey != "" {
		fromIdx = idx[fromKey]
	}
	if toIdx < fromIdx {
		return fmt.Errorf("mundur dari %q ke %q ditolak", fromKey, toKey)
	}
	// Semua langkah di (fromIdx, toIdx) yang dilewati harus Skippable,
	// kecuali yang sudah done.
	for _, s := range ord {
		i := idx[s.Key]
		if i <= fromIdx || i >= toIdx {
			continue
		}
		if done != nil && done[s.Key] {
			continue
		}
		if !s.Skippable {
			return fmt.Errorf("langkah wajib %q belum selesai — tidak bisa lompat ke %q", s.Key, toKey)
		}
	}
	target := ord[toIdx]
	if IsStepBlocked(target, summary) {
		return fmt.Errorf("langkah %q diblokir: checklist belum lengkap", toKey)
	}
	if req := strings.TrimSpace(target.RequireEvidence); req != "" {
		if evidence == nil || !evidence[req] {
			return fmt.Errorf("langkah %q butuh bukti %q", toKey, req)
		}
	}
	return nil
}
