package app

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
)

// ajbWizard renders the guided 4-step AJB flow for one service request.
// Read-only guide: all saves reuse existing endpoints (POST /ppat,
// POST /requests/{id}/checklist). No new POST endpoint here.
func (a *App) ajbWizard(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermDraftDeed) && !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	d, err := a.reqs.Get(a.officeOf(r), reqID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Company is best-effort (AJB requests usually have none); wizard still works.
	co, _, _, _ := a.corp.GetByRequest(reqID)

	penjual, pembeli := splitParties(ajbNoteField(d.Notes, "Pihak/para pihak"))
	sertifikat := ajbNoteField(d.Notes, "Sertifikat")
	luasLokasi := ajbNoteField(d.Notes, "Luas/lokasi")
	if luasLokasi == "" {
		luasLokasi = ajbNoteField(d.Notes, "Luas")
	}
	nilai := ajbNoteField(d.Notes, "Nilai/transaksi")
	if nilai == "" {
		nilai = ajbNoteField(d.Notes, "Objek")
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

	type ppatRow struct {
		ID                       int64
		No, Jenis                string
		Penjual, Pembeli         string
		Sertifikat, Luas, Lokasi string
		Status                   string
	}
	var prows []ppatRow
	if a.db != nil {
		rs, err := a.db.Query(`SELECT id, akta_no, jenis,
			COALESCE(penjual,''), COALESCE(pembeli,''),
			COALESCE(objek_sertifikat,''), COALESCE(objek_luas,''), COALESCE(objek_lokasi,''),
			status FROM ppat_register
			WHERE office_id=? AND deed_id IN (SELECT id FROM deed_drafts WHERE office_id=? AND request_id=?)
			ORDER BY id DESC LIMIT 20`, a.officeOf(r), a.officeOf(r), reqID)
		if err == nil {
			defer rs.Close()
			for rs.Next() {
				var x ppatRow
				_ = rs.Scan(&x.ID, &x.No, &x.Jenis, &x.Penjual, &x.Pembeli, &x.Sertifikat, &x.Luas, &x.Lokasi, &x.Status)
				prows = append(prows, x)
			}
		}
	}

	a.render(w, r, "ajb_wizard", "Wizard AJB", map[string]any{
		"Req": d, "Company": co,
		"Penjual": penjual, "Pembeli": pembeli,
		"Sertifikat": sertifikat, "LuasLokasi": luasLokasi, "Nilai": nilai,
		"Notes":     d.Notes,
		"Checklist": checklist, "CheckDone": done, "CheckTotal": len(checklist), "CheckPct": pct,
		"PpatRows": prows,
	})
}

// ajbNoteField extracts "Key: value" from the structured "[Auto ...] k: v | k2: v2"
// notes line written by requestCreate (case-insensitive, stops at " | " or newline).
func ajbNoteField(notes, key string) string {
	lower := strings.ToLower(notes)
	k := strings.ToLower(key) + ":"
	i := strings.Index(lower, k)
	if i < 0 {
		return ""
	}
	v := strings.TrimSpace(notes[i+len(k):])
	if j := strings.Index(v, " | "); j >= 0 {
		v = v[:j]
	}
	if j := strings.Index(v, "\n"); j >= 0 {
		v = v[:j]
	}
	return strings.TrimSpace(v)
}

// splitParties splits "Penjual → Pembeli" (also accepts "->") into its two sides.
func splitParties(s string) (string, string) {
	s = strings.TrimSpace(s)
	for _, sep := range []string{"→", "->", ">"} {
		if i := strings.Index(s, sep); i >= 0 {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+len(sep):])
		}
	}
	return s, ""
}
