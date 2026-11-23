package app

import "testing"

// TestMatchStepKey_Tabel memverifikasi ekstraksi murni matchStepKey dari
// autoWorkflowStep: pencocokan substring case-insensitive, first-match-wins
// sesuai urutan template. File BARU — test existing tidak diubah.
func TestMatchStepKey_Tabel(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		kw   string
		want string
	}{
		{name: "Draf cocok draf", keys: []string{"Intake", "Draf", "Sahkan"}, kw: "draf", want: "Draf"},
		{name: "case-insensitive kw kapital", keys: []string{"Intake", "Draf", "Sahkan"}, kw: "DRAF", want: "Draf"},
		{name: "case-insensitive key kapital", keys: []string{"DRAF"}, kw: "draf", want: "DRAF"},
		{name: "Sahkan cocok sah", keys: []string{"Syarat", "Draf", "Review", "Sahkan"}, kw: "sah", want: "Sahkan"},
		{name: "Syarat TIDAK cocok sah", keys: []string{"Syarat"}, kw: "sah", want: ""},
		{name: "syarat cocok Syarat", keys: []string{"Syarat", "Draf"}, kw: "syarat", want: "Syarat"},
		{name: "akta cocok Akta", keys: []string{"Objek", "Resi", "Akta", "Lapor"}, kw: "akta", want: "Akta"},
		{name: "lapor cocok Lapor", keys: []string{"Akta", "Lapor", "Serah"}, kw: "lapor", want: "Lapor"},
		{name: "tanpa kecocokan", keys: []string{"Intake", "Pihak"}, kw: "draf", want: ""},
		{name: "kw kosong", keys: []string{"Draf"}, kw: "  ", want: ""},
		{name: "keys kosong", keys: nil, kw: "draf", want: ""},
		// Ekspektasi JUJUR sesuai semantik contains: label "Pengesahan"
		// (BUKAN step_key) memang mengandung "sah" berurutan (…e-S-A-H-an).
		// Matcher produksi hanya menerima step_key sehingga kasus ini tak
		// pernah terjadi; bila suatu hari step_key "Pengesahan" mendahului
		// "Sahkan", contains akan memilihnya — didokumentasikan, bukan bug
		// yang disembunyikan.
		{name: "contains: Pengesahan mengandung sah", keys: []string{"Pengesahan", "Sahkan"}, kw: "sah", want: "Pengesahan"},
		{name: "first-match-wins urut template", keys: []string{"Draf", "DrafUlang"}, kw: "draf", want: "Draf"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := matchStepKey(c.keys, c.kw); got != c.want {
				t.Errorf("matchStepKey(%q, %q) = %q, want %q", c.keys, c.kw, got, c.want)
			}
		})
	}
}

// TestMatchStepKey_DomainSeeds mengunci keputusan matcher untuk domain nyata:
// pada seluruh step_key standardWorkflowSeeds, kw "sah" hanya cocok "Sahkan"
// (tidak pernah "Syarat"/"Serah"), sehingga contains aman dan word-boundary
// eksak justru dilarang (akan memutus pemetaan "sah"→"Sahkan").
func TestMatchStepKey_DomainSeeds(t *testing.T) {
	wantSah := map[string]string{"PT-DIRIAN": "Sahkan", "PPAT-AJB": "", "KUASA": "Sahkan"}
	wantDraf := map[string]string{"PT-DIRIAN": "Draf", "PPAT-AJB": "", "KUASA": "Draf"}
	wantAkta := map[string]string{"PT-DIRIAN": "", "PPAT-AJB": "Akta", "KUASA": ""}
	for _, s := range standardWorkflowSeeds() {
		var keys []string
		for _, st := range s.steps {
			keys = append(keys, st.key)
		}
		if got := matchStepKey(keys, "sah"); got != wantSah[s.code] {
			t.Errorf("%s: kw sah = %q, want %q", s.code, got, wantSah[s.code])
		}
		if got := matchStepKey(keys, "draf"); got != wantDraf[s.code] {
			t.Errorf("%s: kw draf = %q, want %q", s.code, got, wantDraf[s.code])
		}
		if got := matchStepKey(keys, "akta"); got != wantAkta[s.code] {
			t.Errorf("%s: kw akta = %q, want %q", s.code, got, wantAkta[s.code])
		}
		// Syarat tidak boleh false-positive oleh kw "sah" di template mana pun.
		for _, k := range keys {
			if k == "Syarat" && matchStepKey([]string{k}, "sah") != "" {
				t.Errorf("%s: step_key Syarat salah cocok kw sah", s.code)
			}
		}
	}
}
