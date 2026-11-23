package app

// Penawaran resmi terikat permohonan (migrasi 030).
// Alur: buat dari request_detail (prefill tarif layanan) -> DRAFT -> SENT ->
// ACCEPTED (via konversi ke invoice ISSUED) / REJECTED / EXPIRED.
// Invoice hasil konversi memakai matematika PPN + status yang sama dengan
// jalur invoiceCreate (ISSUED, paid=0) sehingga anti-overpay tak berubah.

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/finance"
	mw "notaryos/internal/middleware"
	"notaryos/internal/money"
	"notaryos/internal/rbac"
)

// quotationList mendaftar seluruh penawaran kantor.
func (a *App) quotationList(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewFinance) {
		a.forbidden(w, r)
		return
	}
	// Lazy-expiry di titik list (BUKAN di convert): convertOnce sudah punya
	// guard per-baris yang menandai EXPIRED + menolak konversi, jadi bulk
	// UPDATE di convert juga akan dobel catat. List = satu-satunya penulis
	// bulk; audit agregat satu baris bila ada yang kedaluwarsa.
	if n, err := a.fin.ExpireOverdueQuotations(a.officeOf(r)); err == nil && n > 0 {
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "quotation.expire", Entity: "quotation",
			EntityID: "overdue", Detail: fmt.Sprintf("%d penawaran SENT kedaluwarsa otomatis", n),
			OfficeID: officePtr(a.officeOf(r))})
	}
	list, _ := a.fin.ListQuotations(a.officeOf(r))
	a.render(w, r, "quotations", "Penawaran", map[string]any{"Quotations": list})
}

// quotationCreate membuat penawaran DRAFT dari permohonan.
// Prefill tarif: bila baris kosong dan layanan punya base_price > 0, dibuat
// satu baris "Jasa {nama layanan}" (contoh angka, bukan janji final).
func (a *App) quotationCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("request_id")), 10, 64)
	if reqID <= 0 {
		http.Error(w, "permohonan wajib dipilih", http.StatusBadRequest)
		return
	}
	req, err := a.reqs.Get(a.officeOf(r), reqID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	descs := r.Form["desc[]"]
	qtys := r.Form["qty[]"]
	prices := r.Form["price[]"]
	var lines []finance.Line
	for i := range descs {
		if i >= len(qtys) || i >= len(prices) || strings.TrimSpace(descs[i]) == "" {
			continue
		}
		q, _ := strconv.Atoi(strings.TrimSpace(qtys[i]))
		p, perr := money.Parse(prices[i])
		if perr != nil || p < 0 {
			http.Error(w, "harga baris tidak valid: "+descs[i], http.StatusBadRequest)
			return
		}
		lines = append(lines, finance.Line{Description: strings.TrimSpace(descs[i]), Qty: q, UnitPrice: p})
	}
	if len(lines) == 0 {
		// Prefill jujur dari tarif layanan (services.base_price bila ada).
		var svcName, priceStr string
		if err := a.db.QueryRow(`SELECT s.name, s.base_price FROM service_requests r
			JOIN services s ON s.id=r.service_id WHERE r.id=? AND r.office_id=?`, reqID, a.officeOf(r)).
			Scan(&svcName, &priceStr); err == nil {
			if p := decWhole(priceStr); p > 0 {
				lines = append(lines, finance.Line{Description: "Jasa " + svcName, Qty: 1, UnitPrice: p})
			}
		}
	}
	if len(lines) == 0 {
		http.Error(w, "isi minimal satu rincian (atau pastikan layanan punya tarif)", http.StatusBadRequest)
		return
	}
	id, err := a.fin.CreateQuotation(a.officeOf(r), &reqID,
		strings.TrimSpace(r.FormValue("valid_until")), strings.TrimSpace(r.FormValue("notes")), lines, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = req
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "quotation.create", Entity: "quotation",
		EntityID: strconv.FormatInt(id, 10), OfficeID: officePtr(a.officeOf(r))})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Penawaran+dibuat+(DRAFT)", http.StatusSeeOther)
}

// quotationStatus memajukan status manual (SENT/REJECTED/EXPIRED).
func (a *App) quotationStatus(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	to := strings.ToUpper(strings.TrimSpace(r.FormValue("to")))
	if err := a.fin.SetQuotationStatus(a.officeOf(r), id, to); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	q, _ := a.fin.GetQuotation(a.officeOf(r), id)
	back := "/quotations?flash=Status+penawaran+diperbarui"
	if strings.TrimSpace(r.FormValue("back")) != "/quotations" && q.RequestID != nil {
		back = "/requests/" + strconv.FormatInt(*q.RequestID, 10) + "?flash=Status+penawaran+diperbarui"
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "quotation.status", Entity: "quotation",
		EntityID: strconv.FormatInt(id, 10), Detail: to, OfficeID: officePtr(a.officeOf(r))})
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// quotationConvert menyetujui penawaran -> invoice ISSUED (atomik di store).
func (a *App) quotationConvert(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	invID, err := a.fin.ConvertToInvoice(a.officeOf(r), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "quotation.convert", Entity: "invoice",
		EntityID: strconv.FormatInt(invID, 10), Detail: "from-quotation:" + strconv.FormatInt(id, 10),
		OfficeID: officePtr(a.officeOf(r))})
	http.Redirect(w, r, "/finance?flash=Penawaran+disetujui+%E2%86%92+Invoice+%23"+strconv.FormatInt(invID, 10), http.StatusSeeOther)
}

// quotationPrefill mengambil nama layanan + tarif untuk prefill form di request_detail.
func (a *App) quotationPrefill(r *http.Request, requestID int64) (service string, price int64) {
	var name, priceStr string
	if err := a.db.QueryRow(`SELECT s.name, s.base_price FROM service_requests r
		JOIN services s ON s.id=r.service_id WHERE r.id=? AND r.office_id=?`, requestID, a.officeOf(r)).Scan(&name, &priceStr); err != nil {
		return "", 0
	}
	return name, decWhole(priceStr)
}

// quotDefaultDue adalah batas berlaku default: hari ini + 14 hari.
func quotDefaultDue() string { return time.Now().AddDate(0, 0, 14).Format("2006-01-02") }
