package app

// Billing extras: printable QR receipts, public verification, client payment
// confirmation (bukti transfer), and the reminder engine trigger.

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/skip2/go-qrcode"
	"notaryos/internal/audit"
	"notaryos/internal/cryptoxx"
	mw "notaryos/internal/middleware"
	"notaryos/internal/plan"
	"notaryos/internal/rbac"
	"notaryos/internal/verify"
)

// verifyCode builds the public code for an invoice.
func verifyCode(kind string, id int64) (string, error) {
	key, err := cryptoxx.Key()
	if err != nil {
		return "", err
	}
	return verify.Code(key, kind, id), nil
}

// ---- printable QR receipt (staff, office-scoped) ----

func (a *App) receipt(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewFinance) {
		a.forbidden(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var number, client, status string
	var total, paid int64
	var stotal, spaid, paidAt sql.NullString
	err := a.db.QueryRow(`SELECT i.number, c.full_name, i.status, i.total, i.paid,
		(SELECT MAX(paid_at) FROM payments WHERE invoice_id=i.id)
		FROM invoices i JOIN clients c ON c.id=i.client_id
		WHERE i.id=? AND i.office_id=?`, id, a.officeOf(r)).
		Scan(&number, &client, &status, &stotal, &spaid, &paidAt)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if stotal.Valid {
		total = decWhole(stotal.String)
	}
	if spaid.Valid {
		paid = decWhole(spaid.String)
	}
	paidAtStr := ""
	if paidAt.Valid {
		paidAtStr = paidAt.String
	}
	code, err := verifyCode("INV", id)
	if err != nil {
		http.Error(w, "kunci verifikasi belum dikonfigurasi (NOTARYOS_FIELD_KEY)", http.StatusServiceUnavailable)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "invoice.receipt", Entity: "invoice", EntityID: strconv.FormatInt(id, 10)})
	a.render(w, r, "receipt", "Kuitansi "+number, map[string]any{
		"ID": id, "Number": number, "Client": client, "Status": status,
		"Total": total, "Paid": paid, "PaidAt": paidAtStr, "Code": code,
	})
}

// ---- QR image (public; code tak tertebak HMAC) ----

func (a *App) invoiceQR(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	key, err := cryptoxx.Key()
	if err != nil {
		http.Error(w, "verifikasi belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}
	code := verify.Code(key, "INV", id)
	link := code
	if a.cfg.AppBaseURL != "" {
		link = a.cfg.AppBaseURL + "/verify?code=" + code
	}
	png, err := qrcode.Encode(link, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "gagal membuat QR", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(png)
}

// ---- public verification ----

func (a *App) verifyPage(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	data := map[string]any{"Code": code}
	if code != "" {
		key, err := cryptoxx.Key()
		if err == nil {
			if kind, id, err := verify.Parse(key, code); err == nil {
				switch kind {
				case "INV":
					var number, status, office string
					var stotal, spaid sql.NullString
					if err := a.db.QueryRow(`SELECT i.number, i.status, i.total, i.paid, o.name
						FROM invoices i JOIN offices o ON o.id=i.office_id WHERE i.id=?`, id).
						Scan(&number, &status, &stotal, &spaid, &office); err == nil {
						data["Found"] = true
						data["Kind"] = "Kuitansi"
						data["Number"] = number
						data["Status"] = status
						data["Total"] = decWhole(stotal.String)
						data["Paid"] = decWhole(spaid.String)
						data["Office"] = office
					} else {
						data["Error"] = "dokumen tidak ditemukan"
					}
				case "DEED":
					var number, title, status, office string
					if err := a.db.QueryRow(`SELECT d.number, d.title, d.status, o.name
						FROM deed_drafts d JOIN offices o ON o.id=d.office_id WHERE d.id=?`, id).
						Scan(&number, &title, &status, &office); err == nil {
						data["Found"] = true
						data["Kind"] = "Akta"
						data["Number"] = number + " — " + title
						data["Status"] = status
						data["Total"] = int64(0)
						data["Paid"] = int64(0)
						data["Office"] = office
						var nsig int
						_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_signatures WHERE deed_id=?`, id).Scan(&nsig)
						data["Sigs"] = nsig
					} else {
						data["Error"] = "dokumen tidak ditemukan"
					}
				default:
					data["Error"] = "jenis dokumen tidak dikenal"
				}
			} else {
				data["Error"] = err.Error()
			}
		} else {
			data["Error"] = "layanan verifikasi belum dikonfigurasi"
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "verify", Page{Title: "Verifikasi Dokumen", Data: data, Brand: a.brandView(0)}); err != nil {
		http.Error(w, "gagal merender", http.StatusInternalServerError)
	}
}

// ---- client payment confirmation (bukti transfer) ----

func (a *App) portalInvoices(w http.ResponseWriter, r *http.Request) {
	role := mw.RoleOf(r)
	if role != rbac.RoleClient {
		http.Redirect(w, r, "/finance", http.StatusSeeOther)
		return
	}
	id := a.uid(r)
	var clientID int64
	if id != nil {
		clientID = a.users.LinkedClient(*id)
	}
	if clientID == 0 {
		a.render(w, r, "portal", "Portal Klien", map[string]string{"Note": "Akun Anda belum ditautkan."})
		return
	}
	type inv struct {
		ID          int64
		Number      string
		Total, Paid int64
		Outstanding int64
		Status, Due string
	}
	var out []inv
	rows, _ := a.db.Query(`SELECT id, number, total, paid, status, due_date FROM invoices
		WHERE client_id=? AND office_id=? ORDER BY id DESC`, clientID, a.officeOf(r))
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var v inv
			var due, stotal, spaid sql.NullString
			if err := rows.Scan(&v.ID, &v.Number, &stotal, &spaid, &v.Status, &due); err != nil {
				a.log.Error("portal invoices scan", "err", err)
				continue
			}
			v.Total = decWhole(stotal.String)
			v.Paid = decWhole(spaid.String)
			v.Outstanding = v.Total - v.Paid
			if v.Outstanding < 0 {
				v.Outstanding = 0
			}
			if due.Valid {
				v.Due = due.String
			}
			out = append(out, v)
		}
	}
	a.render(w, r, "portal_invoices", "Tagihan Saya", out)
}

func decWhole(s string) int64 {
	var n int64
	neg := false
	for i, r := range s {
		if i == 0 && r == '-' {
			neg = true
			continue
		}
		if r == '.' {
			break
		}
		if r >= '0' && r <= '9' {
			n = n*10 + int64(r-'0')
		}
	}
	if neg {
		n = -n
	}
	return n
}

func (a *App) portalPayConfirm(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != rbac.RoleClient {
		a.forbidden(w, r)
		return
	}
	if !a.requirePlan(w, r, plan.FeatPortalPay) {
		return
	}
	uid := a.uid(r)
	var clientID int64
	if uid != nil {
		clientID = a.users.LinkedClient(*uid)
	}
	if clientID == 0 {
		http.Error(w, "akun belum ditautkan", http.StatusForbidden)
		return
	}
	_ = r.ParseMultipartForm(100 << 20)
	invID, _ := strconv.ParseInt(r.FormValue("invoice_id"), 10, 64)
	var oid, cid int64
	if err := a.db.QueryRow(`SELECT office_id, client_id FROM invoices WHERE id=?`, invID).Scan(&oid, &cid); err != nil || cid != clientID {
		http.Error(w, "tagihan tidak ditemukan", http.StatusNotFound)
		return
	}
	f, hdr, err := r.FormFile("bukti")
	if err != nil {
		http.Error(w, "unggah foto bukti transfer", http.StatusBadRequest)
		return
	}
	defer f.Close()
	if a.overQuota(oid, hdr.Size) {
		http.Error(w, "kuota penyimpanan kantor terlampaui — hubungi kantor", http.StatusInsufficientStorage)
		return
	}
	fid, err := a.uploads.SaveUpload(oid, "payment_proof", invID, hdr.Filename, hdr.Header.Get("Content-Type"), f, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = a.dms.AutoLink(oid, fid, "invoice", invID, uid)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "payment.confirm", Entity: "invoice", EntityID: strconv.FormatInt(invID, 10), Detail: fmt.Sprintf("bukti #%d", fid)})
	a.render(w, r, "portal_pay_ok", "Konfirmasi Terkirim", map[string]any{"Inv": invID, "File": fid})
}

// remindersRun triggers the daily scan manually (admin).
func (a *App) remindersRun(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageOffice) {
		a.forbidden(w, r)
		return
	}
	appt, overdue, tasks, sla := a.runReminders()
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "reminders.run", Entity: "system", EntityID: "manual", Detail: fmt.Sprintf("janji=%d telat=%d tugas=%d sla=%d", appt, overdue, tasks, sla)})
	http.Redirect(w, r, "/tasks?flash=Pengingat+dibangkitkan", http.StatusSeeOther)
}

// ---- multi-office (ENTERPRISE): consolidated view + acting ----

func (a *App) officesPage(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != rbac.RoleSuperAdmin {
		a.forbidden(w, r)
		return
	}
	type row struct {
		ID          int64
		Name, Plan  string
		Clients     int
		Pending     int
		Outstanding int64
		Acting      bool
	}
	var out []row
	rs, _ := a.db.Query(`SELECT id, name, plan FROM offices ORDER BY id`)
	if rs != nil {
		defer rs.Close()
		for rs.Next() {
			var o row
			_ = rs.Scan(&o.ID, &o.Name, &o.Plan)
			_ = a.db.QueryRow(`SELECT COUNT(*) FROM clients WHERE office_id=?`, o.ID).Scan(&o.Clients)
			pending, _, _, _ := a.reqs.Counts(o.ID)
			o.Pending = pending
			o.Outstanding, _ = a.fin.Outstanding(o.ID)
			if c, err := r.Cookie(a.actingCookieName()); err == nil {
				if oid, ok := a.verifyActing(c.Value); ok && oid == o.ID {
					o.Acting = true
				}
			}
			out = append(out, o)
		}
	}
	a.render(w, r, "offices", "Multi-Kantor", map[string]any{"Offices": out})
}

func (a *App) officeAct(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != rbac.RoleSuperAdmin {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	if v := r.FormValue("office_id"); v == "" {
		http.SetCookie(w, &http.Cookie{Name: a.actingCookieName(), Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/offices?flash=Kembali+ke+kantor+sendiri", http.StatusSeeOther)
		return
	} else {
		oid, _ := strconv.ParseInt(v, 10, 64)
		var pl string
		_ = a.db.QueryRow(`SELECT plan FROM offices WHERE id=?`, oid).Scan(&pl)
		if !plan.Allowed(pl, plan.FeatActing) {
			http.Error(w, "Kantor tujuan membutuhkan paket ENTERPRISE untuk dikelola lintas-kantor.", http.StatusPaymentRequired)
			return
		}
		tok := a.signActing(oid)
		if tok == "" {
			http.Error(w, "kunci penandatanganan belum dikonfigurasi", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: a.actingCookieName(), Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: a.cfg.TLSCookie, MaxAge: 8 * 3600})
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "office.act", Entity: "office", EntityID: strconv.FormatInt(oid, 10)})
		http.Redirect(w, r, "/app?flash=Bertindak+untuk+kantor+%23"+strconv.FormatInt(oid, 10), http.StatusSeeOther)
	}
}

func (a *App) officePlanSave(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != rbac.RoleSuperAdmin {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	oid, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	switch p := r.FormValue("plan"); p {
	case "LITE", "PRO", "ENTERPRISE":
		if _, err := a.db.Exec(`UPDATE offices SET plan=? WHERE id=?`, p, oid); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "office.plan", Entity: "office", EntityID: strconv.FormatInt(oid, 10), Detail: p})
	default:
		http.Error(w, "paket tidak dikenal", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/offices?flash=Paket+diperbarui", http.StatusSeeOther)
}
