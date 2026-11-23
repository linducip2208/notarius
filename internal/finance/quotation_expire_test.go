package finance

import "testing"

func TestIsExpired(t *testing.T) {
	cases := []struct {
		name       string
		validUntil string
		today      string
		want       bool
	}{
		{"lewat sehari", "2026-11-22", "2026-11-23", true},
		{"lewat jauh", "2026-01-01", "2026-11-23", true},
		{"hari-H belum kedaluwarsa", "2026-11-23", "2026-11-23", false},
		{"masa depan", "2026-12-01", "2026-11-23", false},
		{"kosong", "", "2026-11-23", false},
		{"today kosong", "2026-11-22", "", false},
		{"tanggal tak valid", "bukan-tanggal", "2026-11-23", false},
		{"today tak valid", "2026-11-22", "kemarin", false},
		{"format salah", "23-11-2026", "2026-11-23", false},
		{"dengan spasi", " 2026-11-22 ", "2026-11-23", true},
	}
	for _, c := range cases {
		if got := IsExpired(c.validUntil, c.today); got != c.want {
			t.Errorf("%s: IsExpired(%q,%q)=%v want %v", c.name, c.validUntil, c.today, got, c.want)
		}
	}
}
