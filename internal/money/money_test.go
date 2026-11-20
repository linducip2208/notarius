package money

import "testing"

func TestParse(t *testing.T) {
	cases := map[string]int64{
		"1500000": 1500000, "1.500.000": 1500000, "Rp 1.500.000": 1500000, "0": 0,
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Fatalf("Parse(%q)=%v,%v want %v", in, got, err, want)
		}
	}
	if _, err := Parse("abc"); err == nil {
		t.Fatal("expected error for abc")
	}
}

func TestPPN(t *testing.T) {
	if got := PPN(1000000, 1100); got != 1110000 {
		t.Fatalf("PPN=%d", got)
	}
	if FormatIDR(1500000) != "Rp 1.500.000" {
		t.Fatalf("format=%s", FormatIDR(1500000))
	}
}

func TestMulOverflow(t *testing.T) {
	if _, err := Mul(9_000_000_000_000_000, 2); err == nil {
		t.Fatal("expected overflow error")
	}
}
