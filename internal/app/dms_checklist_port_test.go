package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestChecklistPort_RoundTrip memastikan bundle {checklists, workflows}
// selamat melewati marshal -> unmarshal -> validasi (murni, tanpa DB).
func TestChecklistPort_RoundTrip(t *testing.T) {
	orig := checklistPortBundle{
		Checklists: []checklistPort{
			{
				ServiceCode: "KUASA",
				Name:        "Standar Surat Kuasa",
				Items:       []string{"KTP pemberi & penerima kuasa", "Ruang lingkup kuasa"},
			},
		},
		Workflows: []workflowPort{
			{
				ServiceCode: "KUASA",
				Name:        "Alur Surat Kuasa",
				Steps: []checklistStepPort{
					{Key: "Intake", Label: "Penerimaan", RequiredRole: "staff"},
				},
			},
		},
	}

	raw, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	var got checklistPortBundle
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	if err := validateChecklistPortBundle(&got); err != nil {
		t.Fatalf("bundle round-trip tidak valid: %v", err)
	}
	if len(got.Checklists) != 1 || len(got.Workflows) != 1 {
		t.Fatalf("bundle round-trip = %d paket+%d alur, want 1+1",
			len(got.Checklists), len(got.Workflows))
	}
	if got.Checklists[0].ServiceCode != "KUASA" || len(got.Checklists[0].Items) != 2 {
		t.Errorf("checklist round-trip rusak: %+v", got.Checklists[0])
	}
	if got.Workflows[0].Steps[0].Key != "Intake" {
		t.Errorf("workflow round-trip rusak: %+v", got.Workflows[0])
	}
}

// TestValidateChecklistPortBundle_Rejects memastikan bundle rusak ditolak.
func TestValidateChecklistPortBundle_Rejects(t *testing.T) {
	mkItems := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "item"
		}
		return out
	}
	validChecklist := func() checklistPort {
		return checklistPort{ServiceCode: "KUASA", Name: "Standar", Items: []string{"KTP"}}
	}
	validWorkflow := func() workflowPort {
		return workflowPort{ServiceCode: "KUASA", Name: "Alur",
			Steps: []checklistStepPort{{Key: "Intake", Label: "Penerimaan", RequiredRole: "staff"}}}
	}

	cases := []struct {
		name   string
		mutate func(*checklistPortBundle)
		want   string // substring pesan error yang diharapkan
	}{
		{
			name:   "bundle kosong",
			mutate: func(b *checklistPortBundle) { b.Checklists = nil; b.Workflows = nil },
			want:   "minimal 1 checklist",
		},
		{
			name:   "items kosong",
			mutate: func(b *checklistPortBundle) { b.Checklists[0].Items = nil; b.Workflows = nil },
			want:   "minimal 1 item",
		},
		{
			name: "lebih dari 200 item",
			mutate: func(b *checklistPortBundle) {
				b.Checklists[0].Items = mkItems(checklistPortMaxItems + 1)
				b.Workflows = nil
			},
			want: "maks 200 item",
		},
		{
			name:   "service_code kosong",
			mutate: func(b *checklistPortBundle) { b.Checklists[0].ServiceCode = "   "; b.Workflows = nil },
			want:   "service_code dan name wajib diisi",
		},
		{
			name:   "name checklist kosong",
			mutate: func(b *checklistPortBundle) { b.Checklists[0].Name = ""; b.Workflows = nil },
			want:   "service_code dan name wajib diisi",
		},
		{
			name:   "item kosong di tengah",
			mutate: func(b *checklistPortBundle) { b.Checklists[0].Items = []string{"KTP", "  "}; b.Workflows = nil },
			want:   "kosong",
		},
		{
			name:   "workflow tanpa langkah",
			mutate: func(b *checklistPortBundle) { b.Checklists = nil; b.Workflows[0].Steps = nil },
			want:   "minimal 1 langkah",
		},
		{
			name: "workflow langkah tanpa key",
			mutate: func(b *checklistPortBundle) {
				b.Checklists = nil
				b.Workflows[0].Steps = []checklistStepPort{{Key: "", Label: "Penerimaan"}}
			},
			want: "butuh key dan label",
		},
		{
			name: "terlalu banyak paket",
			mutate: func(b *checklistPortBundle) {
				b.Checklists = make([]checklistPort, checklistPortMaxPackages+1)
				for i := range b.Checklists {
					b.Checklists[i] = validChecklist()
				}
				b.Workflows = nil
			},
			want: "terlalu banyak paket",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := checklistPortBundle{
				Checklists: []checklistPort{validChecklist()},
				Workflows:  []workflowPort{validWorkflow()},
			}
			c.mutate(&b)
			err := validateChecklistPortBundle(&b)
			if err == nil {
				t.Fatalf("want error mengandung %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %q, want mengandung %q", err.Error(), c.want)
			}
		})
	}
}

// TestChecklistPort_InvalidJSON memastikan byte bukan-JSON gagal di-decode
// sebelum sampai ke validator.
func TestChecklistPort_InvalidJSON(t *testing.T) {
	for _, raw := range []string{"", "{bukan json", "[1,2", "null["} {
		var b checklistPortBundle
		if err := json.Unmarshal([]byte(raw), &b); err == nil {
			t.Errorf("unmarshal %q = nil, want error", raw)
		}
	}
}

// TestValidateChecklistPortBundle_Normalize memastikan validator menormalisasi
// spasi dan memberi default required_role=staff (perilaku murni import).
func TestValidateChecklistPortBundle_Normalize(t *testing.T) {
	b := checklistPortBundle{
		Checklists: []checklistPort{
			{ServiceCode: "  KUASA ", Name: " Standar ", Items: []string{"  KTP  "}},
		},
		Workflows: []workflowPort{
			{ServiceCode: "KUASA", Name: "Alur",
				Steps: []checklistStepPort{{Key: "Intake", Label: "Penerimaan"}}},
		},
	}
	if err := validateChecklistPortBundle(&b); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if b.Checklists[0].ServiceCode != "KUASA" || b.Checklists[0].Items[0] != "KTP" {
		t.Errorf("normalisasi checklist gagal: %+v", b.Checklists[0])
	}
	if b.Workflows[0].Steps[0].RequiredRole != "staff" {
		t.Errorf("default RequiredRole = %q, want staff", b.Workflows[0].Steps[0].RequiredRole)
	}
}

// TestChecklistPort_Idempotent menandai cakupan yang SENGAJA tidak diuji:
// deduplikasi (service_code, name) hidup di dmsChecklistsImport dan memakai
// SELECT/INSERT ke DB (doc_checklist_templates, workflow_templates), jadi
// butuh DB/integrasi dan tidak diuji di unit murni ini.
func TestChecklistPort_Idempotent(t *testing.T) {
	t.Skip("butuh DB: idempotensi import (skip paket (service_code,name) yang sudah ada) " +
		"menempel di handler dmsChecklistsImport via a.db; diuji di level integrasi, bukan unit murni")
}
