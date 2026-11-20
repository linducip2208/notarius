package validate

import "testing"

func TestNIK(t *testing.T) {
	if err := NIK("3174051209900001"); err != nil {
		t.Fatal(err)
	}
	if err := NIK("123"); err == nil {
		t.Fatal("short NIK must fail")
	}
	if err := NPWP("1234567890123456"); err != nil {
		t.Fatal(err)
	}
}
