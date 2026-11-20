package auth

import "testing"

func TestPasswordAndSession(t *testing.T) {
	h, err := HashPassword("rahasia123")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "rahasia123") || CheckPassword(h, "salah") {
		t.Fatal("bcrypt verify broken")
	}
	if _, err := HashPassword("pendek"); err == nil {
		t.Fatal("short password must be rejected")
	}
	st := NewStore(0)
	s, err := st.Create(7, "notary")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := st.Get(s.Token)
	if !ok || got.UserID != 7 {
		t.Fatal("session roundtrip broken")
	}
	st.Revoke(s.Token)
	if _, ok := st.Get(s.Token); ok {
		t.Fatal("revoked session must be invalid")
	}
}
