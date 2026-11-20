package gov

import "testing"

func TestAssistedFlow(t *testing.T) {
	s := NewSubmission(ChannelAHU, "PT Maju Jaya")
	if len(s.Checklist) == 0 {
		t.Fatal("AHU checklist must not be empty")
	}
	if err := s.Advance(StatusReady); err != nil {
		t.Fatal(err)
	}
	if err := s.Advance(StatusDraft); err == nil {
		t.Fatal("backward move must be rejected")
	}
}
