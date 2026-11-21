package app

// Client CRM handlers: profile edit, KYC workflow, notes, duplicate merge,
// bulk actions. All office-scoped + audited. NIK full never leaves the vault.

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/clients"
	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
)

func (a *App) clientEditForm(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := a.clients.GetFull(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var staff []map[string]any
	if rows, err := a.db.Query(`SELECT u.id, u.name FROM users u JOIN roles r ON r.id=u.role_id
		WHERE u.office_id=? AND u.is_active=1 AND r.name!='client' ORDER BY u.name`, a.officeOf(r)); err == nil {
		defer rows.Close()
		for rows.Next() {
			var uid int64
			var n string
			_ = rows.Scan(&uid, &n)
			staff = append(staff, map[string]any{"ID": uid, "Name": n})
		}
	}
	a.render(w, r, "client_edit", "Edit Klien", map[string]any{"C": c, "Staff": staff})
}

func (a *App) clientUpdate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	office := a.officeOf(r)
	old, err := a.clients.GetFull(office, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	pic, _ := strconv.ParseInt(r.FormValue("pic_user_id"), 10, 64)
	c := clients.ClientFull{Client: clients.Client{ID: id, OfficeID: office,
		Kind: old.Kind, FullName: strings.TrimSpace(r.FormValue("full_name")),
		Email: strings.TrimSpace(r.FormValue("email")), Phone: strings.TrimSpace(r.FormValue("phone")),
		Address: r.FormValue("address"), NPWP: strings.TrimSpace(r.FormValue("npwp"))},
		RegNumber:      strings.TrimSpace(r.FormValue("reg_number")),
		BirthPlace:     r.FormValue("birth_place"),
		BirthDate:      r.FormValue("birth_date"),
		Gender:         r.FormValue("gender"),
		Nationality:    r.FormValue("nationality"),
		Marital:        r.FormValue("marital_status"),
		Occupation:     r.FormValue("occupation"),
		Education:      r.FormValue("education"),
		Passport:       r.FormValue("passport_no"),
		IDAddress:      r.FormValue("id_address"),
		Domicile:       r.FormValue("domicile_address"),
		Province:       r.FormValue("province"),
		City:           r.FormValue("city"),
		District:       r.FormValue("district"),
		Village:        r.FormValue("village"),
		Postal:         r.FormValue("postal_code"),
		AltPhone:       r.FormValue("alt_phone"),
		WANumber:       r.FormValue("wa_number"),
		Correspondence: r.FormValue("correspondence_address"),
		FirstClientDate: r.FormValue("first_client_date"),
		Source:         r.FormValue("source"),
		PIC:            pic,
		Status:         r.FormValue("status"),
		InternalNotes:  r.FormValue("internal_notes"),
		Labels:         r.FormValue("labels"),
	}
	if c.Status != "ACTIVE" && c.Status != "INACTIVE" && c.Status != "ARCHIVED" {
		c.Status = old.Status
	}
	if err := a.clients.UpdateFull(office, c); err != nil {
		a.render(w, r, "client_edit", "Edit Klien", map[string]any{"C": c, "Error": err.Error()})
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "client.update", Entity: "client", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(id, 10)+"?flash=Profil+klien+disimpan", http.StatusSeeOther)
}

// identityStatus advances human verification (never auto from OCR).
func (a *App) identityStatus(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermVerifyIdentity) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var cid int64
	var status string
	_ = a.db.QueryRow(`SELECT i.client_id, COALESCE(i.status,'UNCHECKED') FROM identities i
		JOIN clients c ON c.id=i.client_id WHERE i.id=?`, id).Scan(&cid, &status)
	to := r.FormValue("to")
	if err := a.clients.IdentityStatus(a.officeOf(r), id, to, r.FormValue("note"), a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "identity.verify." + strings.ToLower(to), Entity: "identity", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("note")})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(cid, 10)+"?flash=Verifikasi+dicatat#tab-id", http.StatusSeeOther)
}

func (a *App) identityFlag(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermVerifyIdentity) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var cid int64
	_ = a.db.QueryRow(`SELECT client_id FROM identities WHERE id=?`, id).Scan(&cid)
	if err := a.clients.MarkInconsistent(a.officeOf(r), id, r.FormValue("flag") == "1"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "identity.flag", Entity: "identity", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("flag")})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(cid, 10)+"?flash=Penanda+diperbarui#tab-id", http.StatusSeeOther)
}

// noteAdd records a CRM entry (meeting/followup/complaint/note).
func (a *App) noteAdd(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	cid, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var occ *string
	if v := strings.TrimSpace(r.FormValue("occurred_at")); v != "" {
		occ = &v
	}
	if _, err := a.clients.AddNote(a.officeOf(r), cid, r.FormValue("kind"), r.FormValue("title"), r.FormValue("body"), occ, a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "client.note", Entity: "client", EntityID: strconv.FormatInt(cid, 10), Detail: r.FormValue("kind")})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(cid, 10)+"?flash=Catatan+tersimpan#tab-activity", http.StatusSeeOther)
}

// mergePreview shows what a merge would move (read-only).
func (a *App) mergePreview(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	winner, _ := strconv.ParseInt(r.FormValue("winner_id"), 10, 64)
	loser, _ := strconv.ParseInt(r.FormValue("loser_id"), 10, 64)
	mp, err := a.clients.MergePreview(a.officeOf(r), winner, loser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var wn, ln string
	_ = a.db.QueryRow(`SELECT full_name FROM clients WHERE id=?`, winner).Scan(&wn)
	_ = a.db.QueryRow(`SELECT full_name FROM clients WHERE id=?`, loser).Scan(&ln)
	a.render(w, r, "client_merge", "Gabung Duplikat — Pratinjau", map[string]any{
		"MP": mp, "Winner": wn, "Loser": ln,
	})
}

// mergeDo executes the merge (canonical kept, duplicate archived, full audit).
func (a *App) mergeDo(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	winner, _ := strconv.ParseInt(r.FormValue("winner_id"), 10, 64)
	loser, _ := strconv.ParseInt(r.FormValue("loser_id"), 10, 64)
	if err := a.clients.MergeClients(a.officeOf(r), winner, loser, a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "client.merge", Entity: "client",
		EntityID: strconv.FormatInt(winner, 10), Detail: "gabung #" + strconv.FormatInt(loser, 10)})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(winner, 10)+"?flash=Duplikat+digabung", http.StatusSeeOther)
}

// clientBulk updates status for many clients at once (with confirmation on-page).
func (a *App) clientBulk(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	to := r.FormValue("to")
	if to != "ACTIVE" && to != "INACTIVE" && to != "ARCHIVED" {
		http.Error(w, "status tidak dikenal", http.StatusBadRequest)
		return
	}
	office := a.officeOf(r)
	n := 0
	for _, sid := range r.Form["ids"] {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		if _, err := a.db.Exec(`UPDATE clients SET status=? WHERE id=? AND office_id=?`, to, id, office); err == nil {
			n++
		}
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "client.bulk", Entity: "client", EntityID: "bulk", Detail: fmt.Sprintf("%d -> %s", n, to)})
	http.Redirect(w, r, "/clients?flash="+strconv.Itoa(n)+"+klien+diperbarui", http.StatusSeeOther)
}
