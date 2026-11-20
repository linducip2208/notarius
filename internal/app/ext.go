package app

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/clients"
	"notaryos/internal/corp"
	"notaryos/internal/doctpl"
	"notaryos/internal/gov"
	"notaryos/internal/mfa"
	mw "notaryos/internal/middleware"
	"notaryos/internal/money"
	"notaryos/internal/notify"
	"notaryos/internal/pdf"
	"notaryos/internal/rbac"
)

// ---- MFA ----

func (a *App) mfaVerifyForm(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "mfa_verify", "Verifikasi MFA", nil)
}

func (a *App) mfaVerify(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	c, err := r.Cookie(a.cfg.SessionName + "_mfa")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	a.pendingMu.Lock()
	p, ok := a.pending[c.Value]
	if ok {
		delete(a.pending, c.Value)
	}
	a.pendingMu.Unlock()
	if !ok || time.Now().After(p.expires) {
		http.Redirect(w, r, "/login?flash=Sesi+verifikasi+kedaluwarsa", http.StatusSeeOther)
		return
	}
	var role string
	_ = a.db.QueryRow(`SELECT r.name FROM users u JOIN roles r ON r.id=u.role_id WHERE u.id=?`, p.userID).Scan(&role)
	if !a.users.VerifyMFA(p.userID, r.FormValue("code")) {
		a.pendingMu.Lock()
		a.pending[c.Value] = pendingAuth{userID: p.userID, expires: time.Now().Add(5 * time.Minute)}
		a.pendingMu.Unlock()
		a.render(w, r, "mfa_verify", "Verifikasi MFA", map[string]string{"Error": "Kode MFA salah."})
		return
	}
	sess, err := a.sessions.Create(p.userID, role)
	if err != nil {
		http.Error(w, "gagal membuat sesi", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionName + "_mfa", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionName, Value: sess.Token, Path: "/",
		Expires: sess.ExpiresAt, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.cfg.TLSCookie})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) mfaSetup(w http.ResponseWriter, r *http.Request) {
	id := a.uid(r)
	if id == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	secret, _ := mfa.NewSecret()
	var email string
	_ = a.db.QueryRow(`SELECT email FROM users WHERE id=?`, *id).Scan(&email)
	a.render(w, r, "mfa", "Aktifkan MFA", map[string]any{
		"Secret": secret, "URI": mfa.ProvisioningURI(secret, email, "NOTARYOS"),
		"Enabled": a.users.MFAEnabled(*id),
	})
}

func (a *App) mfaEnable(w http.ResponseWriter, r *http.Request) {
	id := a.uid(r)
	if id == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	secret, code := r.FormValue("secret"), r.FormValue("code")
	if !mfa.Verify(secret, code, time.Now()) {
		a.render(w, r, "mfa", "Aktifkan MFA", map[string]any{"Secret": secret, "Error": "Kode salah — pindai ulang & coba lagi."})
		return
	}
	if err := a.users.SetMFASecret(*id, secret); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: id, Action: "mfa.enable", Entity: "user", EntityID: strconv.FormatInt(*id, 10)})
	http.Redirect(w, r, "/settings?flash=MFA+aktif", http.StatusSeeOther)
}

// ---- identities & reps ----

func (a *App) identityAdd(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermCreateClient) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.clients.Get(a.officeOf(r), clientID); err != nil {
		http.NotFound(w, r)
		return
	}
	_, err := a.clients.AddIdentity(clientID, r.FormValue("id_type"), r.FormValue("nik"), r.FormValue("full_name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(clientID, 10)+"?flash=Identitas+disimpan", http.StatusSeeOther)
}

func (a *App) identityVerify(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermVerifyIdentity) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	uid := a.uid(r)
	var vid int64
	if uid != nil {
		vid = *uid
	}
	if err := a.clients.VerifyIdentity(id, vid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: uid, Action: "identity.verify", Entity: "identity", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/clients?flash=Identitas+terverifikasi", http.StatusSeeOther)
}

func (a *App) repAdd(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.clients.Get(a.officeOf(r), clientID); err != nil {
		http.NotFound(w, r)
		return
	}
	if err := a.clients.AddRep(clientID, clients.Rep{
		Name: r.FormValue("name"), Position: r.FormValue("position"),
		Phone: r.FormValue("phone"), Authority: r.FormValue("authority"),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(clientID, 10)+"?flash=Perwakilan+disimpan", http.StatusSeeOther)
}

// ---- uploads & OCR ----

func (a *App) fileUpload(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseMultipartForm(32 << 20)
	clientID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.clients.Get(a.officeOf(r), clientID); err != nil {
		http.NotFound(w, r)
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "pilih file terlebih dahulu", http.StatusBadRequest)
		return
	}
	defer f.Close()
	id, err := a.uploads.SaveUpload(a.officeOf(r), "client", clientID, hdr.Filename, hdr.Header.Get("Content-Type"), f, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.upload", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(clientID, 10)+"?flash=Berkas+diunggah", http.StatusSeeOther)
}

func (a *App) fileOCR(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, err := a.uploads.RunOCR(id, "")
	if err != nil {
		// Honest failure: engine not installed (B-01). Staff falls back to manual entry.
		http.Error(w, "OCR belum dapat dijalankan: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/clients?flash=OCR+selesai", http.StatusSeeOther)
}

// ---- request detail + corporate ----

func (a *App) requestDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	d, err := a.reqs.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	co, shs, offs, _ := a.corp.GetByRequest(id)
	var bos []corp.BeneficialOwner
	if co.ID != 0 {
		bos, _ = a.corp.BeneficialOwners(co.ID)
	}
	subs, _ := a.govs.List(a.officeOf(r))
	a.render(w, r, "request_detail", "Detail Permohonan", map[string]any{
		"Req": d, "Company": co, "Shareholders": shs, "Officers": offs, "BOs": bos, "Gov": subs,
	})
}

func numForm(r *http.Request, k string) float64 {
	f, _ := parseIDR(r.FormValue(k))
	return f
}

func parseIDR(s string) (float64, error) {
	v, err := money.Parse(s)
	return float64(v), err
}

func (a *App) corpSave(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.reqs.Get(a.officeOf(r), reqID); err != nil {
		http.NotFound(w, r)
		return
	}
	_, err := a.corp.UpsertCompany(corp.Company{
		RequestID:  &reqID,
		EntityType: r.FormValue("entity_type"),
		Name:       r.FormValue("name"),
		Domicile:   r.FormValue("domicile"),
		Address:    r.FormValue("address"),
		Authorized: numForm(r, "authorized"),
		Issued:     numForm(r, "issued"),
		Paid:       numForm(r, "paid"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Data+korporat+disimpan", http.StatusSeeOther)
}

func (a *App) shareholderAdd(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	co, _, _, _ := a.corp.GetByRequest(reqID)
	if co.ID == 0 {
		http.Error(w, "simpan data perusahaan dulu", http.StatusBadRequest)
		return
	}
	shares, _ := strconv.Atoi(r.FormValue("shares"))
	nominal, _ := parseIDR(r.FormValue("nominal"))
	if err := a.corp.AddShareholder(co.ID, r.FormValue("name"), shares, nominal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Pemegang+saham+ditambah", http.StatusSeeOther)
}

func (a *App) officerAdd(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	co, _, _, _ := a.corp.GetByRequest(reqID)
	if co.ID == 0 {
		http.Error(w, "simpan data perusahaan dulu", http.StatusBadRequest)
		return
	}
	if err := a.corp.AddOfficer(co.ID, r.FormValue("role"), r.FormValue("name"), r.FormValue("title")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Pengurus+ditambah", http.StatusSeeOther)
}

func (a *App) boAdd(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermCreateClient) && !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	co, _, _, _ := a.corp.GetByRequest(reqID)
	if co.ID == 0 {
		http.Error(w, "simpan data perusahaan dulu", http.StatusBadRequest)
		return
	}
	pct, _ := strconv.ParseFloat(r.FormValue("pct"), 64)
	pep := r.FormValue("pep") == "1" || r.FormValue("pep") == "on"
	if err := a.corp.AddBeneficialOwner(co.ID, r.FormValue("name"), r.FormValue("nik"), pct, pep); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "corp.bo.add", Entity: "company", EntityID: strconv.FormatInt(co.ID, 10)})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Beneficial+owner+dicatat", http.StatusSeeOther)
}

// ---- appointments ----

func (a *App) appointments(w http.ResponseWriter, r *http.Request) {
	list, _ := a.reqs.Appointments(a.officeOf(r))
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	a.render(w, r, "appointments", "Janji Temu", map[string]any{"List": list, "Clients": cl})
}

func (a *App) appointmentCreate(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	at, err := time.Parse("2006-01-02T15:04", r.FormValue("scheduled_at"))
	if err != nil {
		http.Error(w, "format jadwal tidak valid", http.StatusBadRequest)
		return
	}
	if _, err := a.reqs.Schedule(a.officeOf(r), clientID, nil, at, r.FormValue("purpose")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/appointments?flash=Janji+disimpan", http.StatusSeeOther)
}

// ---- gov assisted ----

func (a *App) govCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermGovSubmit) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	ch := gov.Channel(r.FormValue("channel"))
	if ch != gov.ChannelAHU && ch != gov.ChannelOSS {
		http.Error(w, "kanal tidak dikenal", http.StatusBadRequest)
		return
	}
	id, err := a.govs.Create(a.officeOf(r), ch, r.FormValue("reference"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "gov.create", Entity: "gov_submission", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/gov?flash=Pengajuan+dibuat", http.StatusSeeOther)
}

func (a *App) govAdvance(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	err := a.govs.Advance(id, gov.Status(r.FormValue("to")), r.FormValue("receipt_no"), r.FormValue("receipt_ref"), r.FormValue("notes"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/gov?flash=Status+pengajuan+diperbarui", http.StatusSeeOther)
}

func (a *App) govCheck(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermGovSubmit) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	idx, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		http.Error(w, "index checklist tidak valid", http.StatusBadRequest)
		return
	}
	if err := a.govs.ToggleCheck(id, idx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "gov.check", Entity: "gov_submission", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/gov?flash=Checklist+diperbarui", http.StatusSeeOther)
}

// ---- templates & clauses ----

func (a *App) templateSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, err := a.tpl.Save(a.officeOf(r), 0, r.FormValue("name"), r.FormValue("category"), r.FormValue("body"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/templates/"+strconv.FormatInt(id, 10)+"?flash=Template+disimpan", http.StatusSeeOther)
}

func (a *App) templateEdit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	t, err := a.tpl.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	vars := map[string]string{
		"notary_name": "Nama Notaris, S.H.", "notary_office": "Kantor Notaris",
		"client_name": "Nama Klien", "client_nik": "3174XXXXXXXXXXXX",
		"company_name": "PT Contoh", "company_address": "Jl. Contoh",
	}
	rendered, missing := doctpl.Preview(t.Body, vars)
	a.render(w, r, "template_edit", "Editor Template", map[string]any{
		"T": t, "Preview": rendered, "Missing": missing,
		"Vars": []string{"notary_name", "notary_office", "client_name", "client_nik", "company_name", "company_address", "shareholder_list", "authorized_capital", "issued_capital", "director_list"},
	})
}

func (a *App) templateApprove(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) {
		http.Error(w, "hanya notaris yang dapat menyetujui template", http.StatusForbidden)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	uid := a.uid(r)
	var approver int64
	if uid != nil {
		approver = *uid
	}
	if err := a.tpl.Approve(id, approver); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: uid, Action: "template.approve", Entity: "doc_template", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/templates/"+strconv.FormatInt(id, 10)+"?flash=Template+disetujui", http.StatusSeeOther)
}

func (a *App) clauseSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	if _, err := a.tpl.SaveClause(a.officeOf(r), r.FormValue("title"), r.FormValue("body"), r.FormValue("tags")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/templates?flash=Klausul+disimpan", http.StatusSeeOther)
}

// ---- deed PDF export ----

func (a *App) deedToReview(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermDraftDeed) && !rbac.Has(mw.RoleOf(r), rbac.PermEditDeed) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	office := a.officeOf(r)
	var status string
	var oid int64
	if err := a.db.QueryRow(`SELECT status, office_id FROM deed_drafts WHERE id=?`, id).Scan(&status, &oid); err != nil {
		http.NotFound(w, r)
		return
	}
	if oid != office {
		http.NotFound(w, r)
		return
	}
	if status != "DRAFT" {
		http.Error(w, "hanya draf DRAFT yang dapat dikirim ke review", http.StatusBadRequest)
		return
	}
	if _, err := a.db.Exec(`UPDATE deed_drafts SET status='IN_REVIEW' WHERE id=?`, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "deed.review", Entity: "deed", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/deeds?flash=Draf+dikirim+ke+review+notaris", http.StatusSeeOther)
}

func (a *App) deedApprove(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) {
		http.Error(w, "hanya notaris yang dapat menyetujui akta", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	office := a.officeOf(r)
	var oid int64
	if err := a.db.QueryRow(`SELECT office_id FROM deed_drafts WHERE id=?`, id).Scan(&oid); err != nil {
		http.NotFound(w, r)
		return
	}
	if oid != office {
		http.NotFound(w, r)
		return
	}
	uid := a.uid(r)
	var approver int64
	if uid != nil {
		approver = *uid
	}
	if err := a.deeds.Approve(id, r.FormValue("number"), approver); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: uid, Action: "deed.approve", Entity: "deed", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("number")})
	http.Redirect(w, r, "/deeds?flash=Akta+disetujui+dan+masuk+repertorium", http.StatusSeeOther)
}

func (a *App) deedPDF(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var title, body, status string
	err := a.db.QueryRow(`SELECT title, content, status FROM deed_drafts WHERE id=? AND office_id=?`, id, a.officeOf(r)).
		Scan(&title, &body, &status)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	draft := status != "SIGNED" && status != "ARCHIVED"
	out := pdf.Export(title, body, draft)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"akta-"+strconv.FormatInt(id, 10)+".pdf\"")
	w.Write(out)
}

// ---- user admin ----

func (a *App) userCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	office := a.officeOf(r)
	id, err := a.users.Create(ptrInt(office), r.FormValue("role"), r.FormValue("name"), r.FormValue("email"), r.FormValue("phone"), r.FormValue("password"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "user.create", Entity: "user", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/settings?flash=Pengguna+dibuat", http.StatusSeeOther)
}

func ptrInt(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func (a *App) userLink(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	uid, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	cid, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	if _, err := a.clients.Get(a.officeOf(r), cid); err != nil {
		http.Error(w, "klien tidak ditemukan", http.StatusBadRequest)
		return
	}
	if err := a.users.LinkClient(uid, cid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings?flash=Akun+ditautkan", http.StatusSeeOther)
}

// ---- wills (restricted: notary roles only) ----

func willReader(role string) bool {
	return role == rbac.RoleNotary || role == rbac.RoleSuperAdmin ||
		role == rbac.RoleSubstitute || role == rbac.RoleAuditor
}

func (a *App) willsPage(w http.ResponseWriter, r *http.Request) {
	if !willReader(mw.RoleOf(r)) {
		http.Error(w, "Forbidden: data wasiat hanya untuk notaris.", http.StatusForbidden)
		return
	}
	list, _ := a.wills.List(a.officeOf(r))
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	a.render(w, r, "wills", "Register Wasiat", map[string]any{"Wills": list, "Clients": cl})
}

func (a *App) willCreate(w http.ResponseWriter, r *http.Request) {
	role := mw.RoleOf(r)
	if role != rbac.RoleNotary && role != rbac.RoleSuperAdmin && role != rbac.RoleSubstitute {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	id, err := a.wills.Register(a.officeOf(r), clientID, nil, r.FormValue("register_no"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "will.register", Entity: "will", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/wills?flash=Wasiat+terdaftar", http.StatusSeeOther)
}

func (a *App) willHandover(w http.ResponseWriter, r *http.Request) {
	role := mw.RoleOf(r)
	if role != rbac.RoleNotary && role != rbac.RoleSuperAdmin && role != rbac.RoleSubstitute {
		http.Error(w, "Forbidden: serah-terima hanya oleh notaris", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := a.wills.Handover(id, r.FormValue("handed_to"), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "will.handover", Entity: "will", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("handed_to")})
	http.Redirect(w, r, "/wills?flash=Serah-terima+wasiat+dicatat", http.StatusSeeOther)
}

func (a *App) notifRead(w http.ResponseWriter, r *http.Request) {
	if id := a.uid(r); id != nil {
		_ = notify.ReadAll(a.db, *id)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// notifCount is a template helper wired in render below.
func (a *App) unreadCount(r *http.Request) int {
	if id := a.uid(r); id != nil {
		return notify.Unread(a.db, *id)
	}
	return 0
}
