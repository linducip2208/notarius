package app

import (
	"testing"

	"notaryos/internal/requests"
)

// Matriks event → status: target helper harus selalu konsisten dengan
// transisi legal requests.CanMove (sumber kebenaran tunggal).
func TestRequestEventMatrix(t *testing.T) {
	to, ok := requestEventTarget("deed_approved")
	if !ok || to != "IN_PROGRESS" {
		t.Fatalf("deed_approved = %q,%v; want IN_PROGRESS,true", to, ok)
	}
	if !requests.CanMove("APPROVED", to) {
		t.Fatal("deed_approved target harus legal dari APPROVED")
	}
	to, ok = requestEventTarget("handover_done")
	if !ok || to != "COMPLETED" {
		t.Fatalf("handover_done = %q,%v; want COMPLETED,true", to, ok)
	}
	if !requests.CanMove("IN_PROGRESS", to) {
		t.Fatal("handover_done target harus legal dari IN_PROGRESS")
	}
	if _, ok := requestEventTarget("acara_tidak_ada"); ok {
		t.Fatal("event tak dikenal harus ok=false (diam/fail-open)")
	}
	// Rantai IN_REVIEW→APPROVED→IN_PROGRESS harus legal berurutan
	// (menutup jalan buntu: akta disahkan padahal permohonan IN_REVIEW).
	if !requests.CanMove("IN_REVIEW", "APPROVED") {
		t.Fatal("rantai butuh IN_REVIEW→APPROVED yang legal")
	}
	if !requests.CanMove("APPROVED", "IN_PROGRESS") {
		t.Fatal("rantai butuh APPROVED→IN_PROGRESS yang legal")
	}
	// Fail-open tanpa DB: nil db / id tak valid tidak boleh panic.
	if advanceOnEvent(nil, 0, 0, "deed_approved", nil, nil) {
		t.Fatal("advanceOnEvent tanpa DB tak boleh bergerak")
	}
	if advanceOnEvent(nil, 1, 1, "event_asing", nil, nil) {
		t.Fatal("advanceOnEvent event asing tak boleh bergerak")
	}
}
