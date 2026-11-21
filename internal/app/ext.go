package app

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/clients"
	"notaryos/internal/corp"
	"notaryos/internal/doctpl"
	"notaryos/internal/docx"
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
	http.Redirect(w, r, "/app", http.StatusSeeOther)
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
		"Secret": secret, "URI": mfa.ProvisioningURI(secret, email, "SaktiNotaris"),
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "identity.add", Entity: "client", EntityID: strconv.FormatInt(clientID, 10), Detail: r.FormValue("id_type")})
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "rep.add", Entity: "client", EntityID: strconv.FormatInt(clientID, 10)})
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
	// DMS: auto-relasi ke klien + antre OCR (best-effort, tak menghambat respons).
	_ = a.dms.AutoLink(a.officeOf(r), id, "client", clientID, a.uid(r))
	if _, err := a.dms.EnqueueOCR(a.officeOf(r), id); err == nil {
		_ = a.dms.Move(a.officeOf(r), id, "CLASSIFICATION", a.uid(r), "otomatis saat unggah")
	}
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(clientID, 10)+"?flash=Berkas+diunggah", http.StatusSeeOther)
}

func (a *App) fileOCR(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var oid int64
	if err := a.db.QueryRow(`SELECT office_id FROM files WHERE id=?`, id).Scan(&oid); err != nil {
		http.NotFound(w, r)
		return
	}
	if oid != a.officeOf(r) {
		http.NotFound(w, r)
		return
	}
	_, err := a.uploads.RunOCRWithContext(r.Context(), id)
	if err != nil {
		// Honest failure: engine not installed (B-01). Staff falls back to manual entry.
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.ocr.failed", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
		http.Error(w, "OCR belum dapat dijalankan: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.ocr", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/clients?flash=OCR+selesai", http.StatusSeeOther)
}

// fileDownload streams a private file after office-scope authorization (F-024).
func (a *App) fileDownload(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) && !rbac.Has(mw.RoleOf(r), rbac.PermViewProtocol) {
		a.forbidden(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	m, rc, err := a.uploads.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	name := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(m.Name, `"`, "_"), "\r", "_"), "\n", "_")
	if strings.TrimSpace(name) == "" {
		name = "berkas"
	}
	w.Header().Set("Content-Type", m.MIME)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, rc)
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.download", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
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
	checklist, _ := a.dms.ChecklistForRequest(a.officeOf(r), id)
	linked, _ := a.dms.ByRelated(a.officeOf(r), "request", id)
	a.render(w, r, "request_detail", "Detail Permohonan", map[string]any{
		"Req": d, "Company": co, "Shareholders": shs, "Officers": offs, "BOs": bos, "Gov": subs,
		"Checklist": checklist, "LinkedDocs": linked,
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "corp.save", Entity: "service_request", EntityID: strconv.FormatInt(reqID, 10)})
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "corp.shareholder.add", Entity: "company", EntityID: strconv.FormatInt(co.ID, 10)})
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "corp.officer.add", Entity: "company", EntityID: strconv.FormatInt(co.ID, 10)})
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "appointment.create", Entity: "appointment", EntityID: r.FormValue("client_id")})
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
	if !rbac.Has(mw.RoleOf(r), rbac.PermGovSubmit) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	err := a.govs.AdvanceScoped(a.officeOf(r), id, gov.Status(r.FormValue("to")), r.FormValue("receipt_no"), r.FormValue("receipt_ref"), r.FormValue("notes"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "gov.advance", Entity: "gov_submission", EntityID: strconv.FormatInt(id, 10), Detail: string(r.FormValue("to"))})
	http.Redirect(w, r, "/gov?flash=Status+pengajuan+diperbarui", http.StatusSeeOther)
}

func (a *App) govCheck(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermGovSubmit) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	idx, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		http.Error(w, "index checklist tidak valid", http.StatusBadRequest)
		return
	}
	if err := a.govs.ToggleCheckScoped(a.officeOf(r), id, idx); err != nil {
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "template.save", Entity: "doc_template", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/templates/"+strconv.FormatInt(id, 10)+"?flash=Template+disimpan", http.StatusSeeOther)
}

func (a *App) templateEdit(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "clause.save", Entity: "clause", EntityID: r.FormValue("title")})
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

// canViewDeed allows staff with client view (office-scoped), or a portal
// client linked to the deed's own client (server-side, never trust URL alone).
func (a *App) canViewDeed(r *http.Request, deedID int64) bool {
	if rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		var oid int64
		if err := a.db.QueryRow(`SELECT office_id FROM deed_drafts WHERE id=?`, deedID).Scan(&oid); err != nil {
			return false
		}
		return oid == a.officeOf(r)
	}
	if mw.RoleOf(r) == rbac.RoleClient {
		if id := a.uid(r); id != nil {
			linked := a.users.LinkedClient(*id)
			if linked == 0 {
				return false
			}
			var cid int64
			err := a.db.QueryRow(`SELECT r.client_id FROM deed_drafts d
				JOIN service_requests r ON r.id=d.request_id
				WHERE d.id=? AND d.office_id=?`, deedID, a.officeOf(r)).Scan(&cid)
			return err == nil && cid == linked
		}
	}
	return false
}

func (a *App) deedPDF(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if !a.canViewDeed(r, id) {
		a.forbidden(w, r)
		return
	}
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

func (a *App) deedDOCX(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if !a.canViewDeed(r, id) {
		a.forbidden(w, r)
		return
	}
	var title, body, status string
	err := a.db.QueryRow(`SELECT title, content, status FROM deed_drafts WHERE id=? AND office_id=?`, id, a.officeOf(r)).
		Scan(&title, &body, &status)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	draft := status != "SIGNED" && status != "ARCHIVED"
	out := docx.Export(title, body, draft)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", "attachment; filename=\"akta-"+strconv.FormatInt(id, 10)+".docx\"")
	w.Write(out)
}

func (a *App) protocolCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewProtocol) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	year, _ := strconv.Atoi(r.FormValue("year"))
	if err := a.deeds.CreateProtocol(a.officeOf(r), r.FormValue("bundle_no"), year, r.FormValue("minuta_ref")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "protocol.create", Entity: "protocol", EntityID: r.FormValue("bundle_no")})
	http.Redirect(w, r, "/protocols?flash=Bendel+protokol+dicatat", http.StatusSeeOther)
}

func (a *App) legalisasiAdvance(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermLegalize) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := a.deeds.AdvanceLegalization(a.officeOf(r), id, r.FormValue("to")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "legalisasi.advance", Entity: "legalization", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("to")})
	http.Redirect(w, r, "/legalisasi?flash=Status+register+diperbarui", http.StatusSeeOther)
}

func (a *App) appointmentStatus(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := a.reqs.SetAppointmentStatus(a.officeOf(r), id, r.FormValue("to")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "appointment.status", Entity: "appointment", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("to")})
	http.Redirect(w, r, "/appointments?flash=Janji+diperbarui", http.StatusSeeOther)
}

func (a *App) metrics(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewReports) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	pending, active, completed := 0, 0, 0
	var out int64
	if a.db != nil {
		pending, active, completed, _ = a.reqs.Counts(a.officeOf(r))
		out, _ = a.fin.Outstanding(a.officeOf(r))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("notaryos_requests_pending " + strconv.Itoa(pending) + "\n"))
	w.Write([]byte("notaryos_requests_active " + strconv.Itoa(active) + "\n"))
	w.Write([]byte("notaryos_requests_completed " + strconv.Itoa(completed) + "\n"))
	w.Write([]byte("notaryos_outstanding_rupiah " + strconv.FormatInt(out, 10) + "\n"))
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "user.link", Entity: "user", EntityID: strconv.FormatInt(uid, 10), Detail: "client:" + strconv.FormatInt(cid, 10)})
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
		a.forbidden(w, r)
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

// ---- deed detail + versioning ----

func (a *App) deedDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	d, body, err := a.deeds.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	vers, _ := a.deeds.Versions(a.officeOf(r), id)
	clauses, _ := a.tpl.Clauses(a.officeOf(r))
	// Panel properti: pembuat, tanggal, pengesah, hash, template.
	type props struct {
		Created, Creator, ApprovedBy, ApprovedAt, Hash, Template string
	}
	var pr props
	_ = a.db.QueryRow(`SELECT d.created_at, COALESCE(u.name,''), COALESCE(a.name,''), COALESCE(d.approved_at,''), COALESCE(d.content_hash,''), COALESCE(t.name,'')
		FROM deed_drafts d LEFT JOIN users u ON u.id=d.created_by LEFT JOIN users a ON a.id=d.approved_by
		LEFT JOIN doc_templates t ON t.id=d.template_id WHERE d.id=?`, id).
		Scan(&pr.Created, &pr.Creator, &pr.ApprovedBy, &pr.ApprovedAt, &pr.Hash, &pr.Template)
	// Komparasi berdampingan: ?compare=N tampilkan vN vs vN-1 (jujur, tanpa diff palsu).
	var cmpA, cmpB string
	var cmpN int
	if c := r.URL.Query().Get("compare"); c != "" {
		fmt.Sscanf(c, "%d", &cmpN)
		contents := map[int]string{1: body}
		for _, v := range vers {
			contents[v.No] = v.Content
		}
		if aStr, ok := contents[cmpN]; ok {
			cmpA = aStr
			cmpB = contents[cmpN-1]
		}
	}
	a.render(w, r, "deed_detail", "Detail Akta #"+strconv.FormatInt(id, 10), map[string]any{
		"Deed": d, "Body": body, "Versions": vers, "Clauses": clauses, "Props": pr,
		"CmpA": cmpA, "CmpB": cmpB, "CmpN": cmpN,
		"CanApprove": rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed),
	})
}

func (a *App) deedSaveVersion(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermDraftDeed) && !rbac.Has(mw.RoleOf(r), rbac.PermEditDeed) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	// office check via Get
	if _, _, err := a.deeds.Get(a.officeOf(r), id); err != nil {
		http.NotFound(w, r)
		return
	}
	if err := a.deeds.SaveVersion(id, r.FormValue("content"), a.uid(r), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "deed.version", Entity: "deed", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/deeds/"+strconv.FormatInt(id, 10)+"?flash=Revisi+tersimpan", http.StatusSeeOther)
}

// ---- documents center redirects to DMS library ----

func (a *App) documents(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/dms", http.StatusSeeOther)
}

// ---- audit log ----

func (a *App) auditPage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewReports) {
		a.forbidden(w, r)
		return
	}
	rows, err := a.db.Query(`SELECT a.id, u.name, a.action, a.entity, a.entity_id, a.created_at
		FROM audit_logs a LEFT JOIN users u ON u.id=a.actor_id
		ORDER BY a.id DESC LIMIT 200`)
	if err != nil {
		http.Error(w, "gagal memuat audit", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type entry struct {
		ID, Actor, Action, Entity, EntityID, At string
	}
	var out []entry
	for rows.Next() {
		var e entry
		var actor sql.NullString
		var id int64
		_ = rows.Scan(&id, &actor, &e.Action, &e.Entity, &e.EntityID, &e.At)
		e.ID = strconv.FormatInt(id, 10)
		if actor.Valid {
			e.Actor = actor.String
		}
		out = append(out, e)
	}
	a.render(w, r, "audit", "Audit Log", out)
}

// ---- office profile ----

func (a *App) officePage(w http.ResponseWriter, r *http.Request) {
	var name, notary, sk, addr, phone, email string
	_ = a.db.QueryRow(`SELECT o.name, o.notary_name, o.sk_number, o.address, o.phone, o.email
		FROM offices o JOIN users u ON u.office_id=o.id WHERE u.id=?`, *a.uid(r)).Scan(&name, &notary, &sk, &addr, &phone, &email)
	a.render(w, r, "office", "Profil Kantor", map[string]string{
		"Name": name, "Notary": notary, "SK": sk, "Address": addr, "Phone": phone, "Email": email,
	})
}

func (a *App) officeSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageOffice) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	_, err := a.db.Exec(`UPDATE offices o JOIN users u ON u.office_id=o.id SET o.name=?, o.notary_name=?, o.sk_number=?, o.address=?, o.phone=?, o.email=? WHERE u.id=?`,
		r.FormValue("name"), r.FormValue("notary"), r.FormValue("sk"), r.FormValue("address"), r.FormValue("phone"), r.FormValue("email"), *a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "office.update", Entity: "office", EntityID: "self"})
	http.Redirect(w, r, "/office?flash=Profil+kantor+disimpan", http.StatusSeeOther)
}

// ---- tasks ----

func (a *App) tasksPage(w http.ResponseWriter, r *http.Request) {
	rows, _ := a.db.Query(`SELECT id, title, due_date, status FROM tasks WHERE office_id=? ORDER BY status, due_date LIMIT 200`, a.officeOf(r))
	type task struct {
		ID                 int64
		Title, Due, Status string
	}
	var out []task
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t task
			var due sql.NullString
			_ = rows.Scan(&t.ID, &t.Title, &due, &t.Status)
			if due.Valid {
				t.Due = due.String
			}
			out = append(out, t)
		}
	}
	a.render(w, r, "tasks", "Tugas", out)
}

func (a *App) taskCreate(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.FormValue("title") == "" {
		http.Error(w, "judul tugas wajib diisi", http.StatusBadRequest)
		return
	}
	var due any
	if r.FormValue("due") != "" {
		due = r.FormValue("due")
	}
	_, err := a.db.Exec(`INSERT INTO tasks(office_id, title, due_date, created_by) VALUES(?,?,?,?)`,
		a.officeOf(r), r.FormValue("title"), due, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/tasks?flash=Tugas+ditambah", http.StatusSeeOther)
}

func (a *App) taskDone(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, _ = a.db.Exec(`UPDATE tasks SET status='DONE' WHERE id=? AND office_id=?`, id, a.officeOf(r))
	http.Redirect(w, r, "/tasks?flash=Tugas+selesai", http.StatusSeeOther)
}

// ---- expenses + tariffs ----

func (a *App) expensesPage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewFinance) {
		a.forbidden(w, r)
		return
	}
	rows, _ := a.db.Query(`SELECT id, category, description, amount, spent_at FROM expenses WHERE office_id=? ORDER BY spent_at DESC LIMIT 200`, a.officeOf(r))
	type exp struct {
		ID            int64
		Cat, Desc, At string
		Amount        float64
	}
	var out []exp
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var e exp
			var samt string
			_ = rows.Scan(&e.ID, &e.Cat, &e.Desc, &samt, &e.At)
			fmt.Sscanf(samt, "%f", &e.Amount)
			out = append(out, e)
		}
	}
	a.render(w, r, "expenses", "Pengeluaran & Kas", out)
}

func (a *App) expenseCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	var amount float64
	_, _ = fmt.Sscanf(r.FormValue("amount"), "%f", &amount)
	if amount <= 0 || r.FormValue("description") == "" {
		http.Error(w, "nominal dan uraian wajib diisi", http.StatusBadRequest)
		return
	}
	spent := r.FormValue("spent_at")
	if spent == "" {
		spent = time.Now().Format("2006-01-02")
	}
	_, err := a.db.Exec(`INSERT INTO expenses(office_id, category, description, amount, spent_at, created_by)
		VALUES(?,?,?,?,?,?)`, a.officeOf(r), r.FormValue("category"), r.FormValue("description"), amount, spent, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "expense.create", Entity: "expense", EntityID: r.FormValue("description")})
	http.Redirect(w, r, "/expenses?flash=Pengeluaran+dicatat", http.StatusSeeOther)
}

func (a *App) tariffsPage(w http.ResponseWriter, r *http.Request) {
	rows, _ := a.db.Query(`SELECT id, name, amount, is_active FROM price_items ORDER BY name LIMIT 200`)
	type tariff struct {
		ID     int64
		Name   string
		Amount float64
		Active bool
	}
	var out []tariff
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t tariff
			var samt string
			_ = rows.Scan(&t.ID, &t.Name, &samt, &t.Active)
			fmt.Sscanf(samt, "%f", &t.Amount)
			out = append(out, t)
		}
	}
	svcs, _ := a.reqs.Services()
	a.render(w, r, "tariffs", "Tarif Layanan", map[string]any{"Tariffs": out, "Services": svcs})
}

func (a *App) tariffSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	var amount float64
	_, _ = fmt.Sscanf(r.FormValue("amount"), "%f", &amount)
	if r.FormValue("name") == "" || amount < 0 {
		http.Error(w, "nama dan tarif tidak valid", http.StatusBadRequest)
		return
	}
	var sid any
	if r.FormValue("service_id") != "" {
		sid = r.FormValue("service_id")
	}
	_, err := a.db.Exec(`INSERT INTO price_items(service_id, name, amount) VALUES(?,?,?)`, sid, r.FormValue("name"), amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/tariffs?flash=Tarif+disimpan", http.StatusSeeOther)
}

// ---- global search ----

func (a *App) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	cl, _ := a.clients.List(a.officeOf(r), q, 20)
	type hit struct {
		Kind, Title, URL string
	}
	var out []hit
	for _, c := range cl {
		out = append(out, hit{"Klien", c.FullName + " — " + c.Phone, "/clients/" + strconv.FormatInt(c.ID, 10)})
	}
	if a.db != nil && q != "" {
		rows, _ := a.db.Query(`SELECT id, title, status FROM service_requests WHERE office_id=? AND (title LIKE CONCAT('%',?,'%')) ORDER BY id DESC LIMIT 20`, a.officeOf(r), q)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var title, status string
				_ = rows.Scan(&id, &title, &status)
				out = append(out, hit{"Permohonan", title + " (" + status + ")", "/requests/" + strconv.FormatInt(id, 10)})
			}
		}
		rows2, _ := a.db.Query(`SELECT id, title, status FROM deed_drafts WHERE office_id=? AND (title LIKE CONCAT('%',?,'%')) ORDER BY id DESC LIMIT 20`, a.officeOf(r), q)
		if rows2 != nil {
			defer rows2.Close()
			for rows2.Next() {
				var id int64
				var title, status string
				_ = rows2.Scan(&id, &title, &status)
				out = append(out, hit{"Akta", title + " (" + status + ")", "/deeds/" + strconv.FormatInt(id, 10)})
			}
		}
	}
	a.render(w, r, "search", "Pencarian", map[string]any{"Q": q, "Hits": out})
}

func (a *App) notifRead(w http.ResponseWriter, r *http.Request) {
	if id := a.uid(r); id != nil {
		_ = notify.ReadAll(a.db, *id)
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

// ---- roles & permissions (matrix audit) ----

// rolesPage shows the effective permission matrix (code-enforced, DB-mirrored)
// plus users per role. Read-only: permission changes require code + migration.
func (a *App) rolesPage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		a.forbidden(w, r)
		return
	}
	var roles []string
	rrows, _ := a.db.Query(`SELECT name FROM roles ORDER BY name`)
	if rrows != nil {
		defer rrows.Close()
		for rrows.Next() {
			var n string
			_ = rrows.Scan(&n)
			roles = append(roles, n)
		}
	}
	var perms []string
	prows, _ := a.db.Query(`SELECT name FROM permissions ORDER BY name`)
	if prows != nil {
		defer prows.Close()
		for prows.Next() {
			var n string
			_ = prows.Scan(&n)
			perms = append(perms, n)
		}
	}
	granted := map[string]bool{}
	grows, _ := a.db.Query(`SELECT r.name, p.name FROM role_permissions rp
		JOIN roles r ON r.id=rp.role_id JOIN permissions p ON p.id=rp.permission_id`)
	if grows != nil {
		defer grows.Close()
		for grows.Next() {
			var rn, pn string
			_ = grows.Scan(&rn, &pn)
			granted[rn+"|"+pn] = true
		}
	}
	type urow struct {
		Name, Email, Role string
	}
	var users []urow
	urows, _ := a.db.Query(`SELECT u.name, u.email, r.name FROM users u
		JOIN roles r ON r.id=u.role_id ORDER BY r.name, u.name`)
	if urows != nil {
		defer urows.Close()
		for urows.Next() {
			var u urow
			_ = urows.Scan(&u.Name, &u.Email, &u.Role)
			users = append(users, u)
		}
	}
	a.render(w, r, "roles", "Role & Permission", map[string]any{
		"Roles": roles, "Perms": perms, "Granted": granted, "Users": users,
	})
}

// notifCount is a template helper wired in render below.
func (a *App) unreadCount(r *http.Request) int {
	if id := a.uid(r); id != nil {
		return notify.Unread(a.db, *id)
	}
	return 0
}
