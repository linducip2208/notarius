package app

import (
	"database/sql"
	"strconv"

	"notaryos/internal/audit"
	"notaryos/internal/requests"
)

// Status mengikuti kejadian (tanpa ubah alur manual).
//
// Satu-satunya penggerak status permohonan adalah reqs.Move + CanMove
// (internal/requests) via requestMove (app.go); deedApprove (ext.go) tidak
// menyentuh status permohonan. Helper di file ini menutup celah itu secara
// fail-open: kejadian domain boleh memajukan status, tetapi HANYA bila
// transisi legal menurut requests.CanMove — selebihnya diam (tetap manual).
//
// Matriks event → status yang didukung:
//
//	event "deed_approved" → target IN_PROGRESS (legal langsung hanya dari
//	APPROVED; dari IN_REVIEW ditempuh via rantai IN_REVIEW→APPROVED→IN_PROGRESS)
//	event "handover_done" → target COMPLETED  (legal hanya dari IN_PROGRESS)
//
// ENUM service_requests.status (migrasi 002_clients.sql):
// DRAFT → SUBMITTED → IN_REVIEW ⇄ NEED_DOCS → APPROVED → IN_PROGRESS → COMPLETED,
// plus cabang REJECTED / CANCELLED. Lihat transitions di internal/requests/requests.go.

// requestEventTarget memetakan nama kejadian ke target status permohonan.
// Ok=false untuk event tak dikenal (pemanggil harus diam).
func requestEventTarget(event string) (to string, ok bool) {
	switch event {
	case "deed_approved":
		return "IN_PROGRESS", true
	case "handover_done":
		return "COMPLETED", true
	default:
		return "", false
	}
}

// advanceOnEvent memajukan status permohonan mengikuti kejadian domain.
//
// Fail-open penuh: event tak dikenal, status kini tak mengizinkan transisi
// (CanMove=false), reqID/office tak valid, DB nil, atau error query/exec —
// semuanya diam tanpa mengembalikan error agar alur utama tak pernah diblokir.
// Bila bergerak, pergerakan dicatat ganda: baris request_events (via
// requests.Store.Move) + audit log action "request.move" detail "auto:<event>".
//
// Rantai otomatis (menutup jalan buntu utama: akta disahkan padahal
// permohonan masih IN_REVIEW): untuk "deed_approved", bila status kini
// IN_REVIEW, sistem memajukan IN_REVIEW→APPROVED→IN_PROGRESS berurutan,
// masing-masing hanya bila CanMove mengizinkan. Dari status lain yang tak
// mengarah ke IN_PROGRESS, tetap diam (manual).
func advanceOnEvent(db *sql.DB, officeID, reqID int64, event string, actorID *int64, alog *audit.Logger) (moved bool) {
	defer func() { _ = recover() }()
	to, ok := requestEventTarget(event)
	if !ok || db == nil || alog == nil || officeID <= 0 || reqID <= 0 {
		return false
	}
	var from string
	if err := db.QueryRow(`SELECT status FROM service_requests WHERE id=? AND office_id=?`, reqID, officeID).Scan(&from); err != nil {
		return false
	}
	if from == to {
		return false
	}
	// Rantai: IN_REVIEW --deed disahkan--> APPROVED --> IN_PROGRESS.
	if event == "deed_approved" && from == "IN_REVIEW" {
		if !requests.CanMove(from, "APPROVED") {
			return false
		}
		if err := requests.NewStore(db).Move(officeID, reqID, "APPROVED", actorID, "auto:"+event); err != nil {
			return false
		}
		alog.Record(audit.Entry{ActorID: actorID, Action: "request.move",
			Entity: "service_request", EntityID: strconv.FormatInt(reqID, 10),
			Detail: "auto:" + event + " " + from + "→APPROVED", OfficeID: &officeID})
		from = "APPROVED"
	}
	if !requests.CanMove(from, to) {
		return false // transisi tak legal: tetap manual, diam
	}
	if err := requests.NewStore(db).Move(officeID, reqID, to, actorID, "auto:"+event); err != nil {
		return false
	}
	alog.Record(audit.Entry{ActorID: actorID, Action: "request.move",
		Entity: "service_request", EntityID: strconv.FormatInt(reqID, 10),
		Detail: "auto:" + event + " " + from + "→" + to, OfficeID: &officeID})
	return true
}

// HOOK SERAH TERIMA (untuk agen paralel yang mengerjakan serah-terima):
//
// Tabel serah-terima permohonan (IN_PROGRESS → COMPLETED) BELUM ADA saat
// helper ini ditulis — jadi TIDAK ada pemanggilan aktif untuk "handover_done".
// Setelah tabel/handler serah-terima ada, panggil SATU baris ini tepat setelah
// serah tercatat sukses (komit DB), dengan pola yang sama seperti
// "deed_approved" di deedApprove (ext.go):
//
//	advanceOnEvent(a.db, officeID, reqID, "handover_done", actorID, a.audit)
//
// Perilaku: permohonan IN_PROGRESS → COMPLETED + audit request.move;
// status lain (DRAFT/SUBMITTED/IN_REVIEW/NEED_DOCS/APPROVED/...) → diam,
// alur manual tetap berlaku. Jangan panggil sebelum serah terkomit sukses.
//
// CATATAN: willHandover (ext.go, wills.Handover) adalah serah-terima minuta
// WASiat (tabel wills, tanpa tautan request_id) — BUKAN serah-terima
// permohonan, jadi JANGAN pasang hook handover_done di sana.
