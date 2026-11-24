package app

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	mw "notaryos/internal/middleware"
)

// willWizard renders the guided 3-card Wasiat flow for one service request.
// Restricted: notary roles only (same gate as willsPage). Read-only guide:
// all saves reuse existing endpoints (POST /wills, POST /requests/{id}/checklist,
// POST /deeds). No new POST endpoint here.
func (a *App) willWizard(w http.ResponseWriter, r *http.Request) {
	if !willReader(mw.RoleOf(r)) {
		http.Error(w, "Forbidden: data wasiat hanya untuk notaris.", http.StatusForbidden)
		return
	}
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	d, err := a.reqs.Get(a.officeOf(r), reqID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	checklist, _ := a.dms.ChecklistForRequest(a.officeOf(r), reqID)
	done := 0
	for _, c := range checklist {
		if c.Status == "VERIFIED" || c.Status == "RECEIVED" {
			done++
		}
	}
	pct := 0
	if len(checklist) > 0 {
		pct = done * 100 / len(checklist)
	}

	// Wills related to this request: same client, or linked via a deed draft
	// of this request. wills has no request_id column, so both paths are needed.
	type willRow struct {
		ID         int64
		ClientID   int64
		ClientName string
		RegisterNo string
		Status     string
	}
	var wrows []willRow
	if a.db != nil {
		rs, err := a.db.Query(`SELECT w.id, w.client_id, c.full_name,
			COALESCE(w.register_no,''), w.status
			FROM wills w JOIN clients c ON c.id=w.client_id
			WHERE w.office_id=? AND (w.client_id=?
				OR w.deed_id IN (SELECT id FROM deed_drafts WHERE request_id=?))
			ORDER BY w.id DESC LIMIT 20`, a.officeOf(r), d.ClientID, reqID)
		if err == nil {
			defer rs.Close()
			for rs.Next() {
				var x willRow
				var reg sql.NullString
				_ = rs.Scan(&x.ID, &x.ClientID, &x.ClientName, &reg, &x.Status)
				x.RegisterNo = reg.String
				wrows = append(wrows, x)
			}
		}
	}

	a.render(w, r, "wasiat_wizard", "Wizard Wasiat", map[string]any{
		"Req":       d,
		"SvcCode":   a.svcCodeOf(reqID),
		"Checklist": checklist, "CheckDone": done, "CheckTotal": len(checklist), "CheckPct": pct,
		"Wills": wrows,
	})
}
