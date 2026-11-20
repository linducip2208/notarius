package requests

import "testing"

func TestWorkflowTransitions(t *testing.T) {
	if !CanMove("DRAFT", "SUBMITTED") {
		t.Fatal("DRAFT->SUBMITTED must be legal")
	}
	if CanMove("DRAFT", "COMPLETED") {
		t.Fatal("DRAFT->COMPLETED must be rejected")
	}
	if CanMove("COMPLETED", "DRAFT") {
		t.Fatal("completed deeds must be immutable in flow")
	}
	if !CanMove("IN_REVIEW", "NEED_DOCS") {
		t.Fatal("IN_REVIEW->NEED_DOCS must be legal")
	}
}
