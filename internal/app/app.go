package app

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/auth"
	"notaryos/internal/clients"
	"notaryos/internal/config"
	"notaryos/internal/corp"
	"notaryos/internal/deeds"
	"notaryos/internal/doctpl"
	"notaryos/internal/files"
	"notaryos/internal/finance"
	"notaryos/internal/gov"
	mw "notaryos/internal/middleware"
	"notaryos/internal/notify"
	"notaryos/internal/ocr"
	"notaryos/internal/rbac"
	"notaryos/internal/requests"
	"notaryos/internal/storage"
	"notaryos/internal/users"
	"notaryos/internal/workers"
	"sync"
)

//go:embed templates/*.gohtml
var tmplFS embed.FS

// App wires the modular monolith: router, stores, templates.
type App struct {
	cfg       config.Config
	db        *sql.DB
	log       *slog.Logger
	sessions  auth.SessionStore
	tmpl      *template.Template
	audit     *audit.Logger
	users     *users.Store
	clients   *clients.Store
	reqs      *requests.Store
	deeds     *deeds.Store
	fin       *finance.Store
	wills     *deeds.WillStore
	corp      *corp.Store
	tpl       *doctpl.Store
	govs      *gov.Store
	uploads   *files.Store
	ocr       ocr.Provider
	files     storage.Store
	pool      *workers.Pool
	pending   map[string]pendingAuth
	pendingMu sync.Mutex
}

type pendingAuth struct {
	userID  int64
	expires time.Time
}

// Page is template data for every Tabler page.
type Page struct {
	Title    string
	Role     string
	UserName string
	Flash    string
	CSRF     string
	Data     any
}

// New builds the app. db may be nil only in tests that never hit handlers.
func New(cfg config.Config, db *sql.DB, log *slog.Logger) (*App, error) {
	tmpl, err := template.ParseFS(tmplFS, "templates/*.gohtml")
	if err != nil {
		return nil, err
	}
	fstore, err := storage.NewLocal(cfg.StorageDir)
	if err != nil {
		return nil, err
	}
	a := &App{
		cfg: cfg, db: db, log: log,
		sessions: auth.NewSQLStore(db, 12*time.Hour),
		tmpl:     tmpl,
		audit:    audit.New(db, log),
		files:    fstore,
		pool:     workers.New(4, log),
		pending:  map[string]pendingAuth{},
	}
	if tp, err := ocr.NewTesseract(cfg.OCRBinary); err != nil {
		a.ocr = ocr.NoopProvider{Reason: "atur OCR_BINARY di .env setelah menginstal Tesseract (lihat docs/BLOCKERS.md)"}
		log.Warn("ocr disabled", "err", err)
	} else {
		a.ocr = tp
		log.Info("ocr provider ready", "provenance", tp.Version)
	}
	if db != nil {
		a.users = users.NewStore(db)
		a.clients = clients.NewStore(db)
		a.reqs = requests.NewStore(db)
		a.deeds = deeds.NewStore(db)
		a.fin = finance.NewStore(db)
		a.wills = deeds.NewWillStore(db)
		a.corp = corp.NewStore(db)
		a.tpl = doctpl.NewStore(db)
		a.govs = gov.NewStore(db)
		a.uploads = files.NewStore(db, a.files, a.ocr)
		// Session hygiene: hapus sesi kedaluwarsa tiap jam (best-effort, B-07).
		if sqls, ok := a.sessions.(*auth.SQLStore); ok {
			go func() {
				t := time.NewTicker(time.Hour)
				defer t.Stop()
				for range t.C {
					_, _ = sqls.Cleanup()
				}
			}()
		}
	}
	return a, nil
}

func (a *App) uid(r *http.Request) *int64 {
	if v := r.Context().Value(mw.CtxUserID); v != nil {
		id := v.(int64)
		return &id
	}
	return nil
}

func (a *App) officeOf(r *http.Request) int64 {
	id := a.uid(r)
	if id == nil || a.db == nil {
		return 0
	}
	var office sql.NullInt64
	_ = a.db.QueryRow(`SELECT office_id FROM users WHERE id=?`, *id).Scan(&office)
	if office.Valid {
		return office.Int64
	}
	return 0
}

func (a *App) render(w http.ResponseWriter, r *http.Request, name string, title string, data any) {
	role := mw.RoleOf(r)
	username := ""
	if id := a.uid(r); id != nil && a.db != nil {
		_ = a.db.QueryRow(`SELECT name FROM users WHERE id=?`, *id).Scan(&username)
	}
	flash := r.URL.Query().Get("flash")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, name, Page{Title: title, Role: role, UserName: username, Flash: flash, CSRF: mw.IssueCSRF(w, r, a.cfg.SessionName, a.cfg.TLSCookie), Data: data}); err != nil {
		a.log.Error("render", "page", name, "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// Router builds all routes.
func (a *App) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(mw.SecurityHeaders)
	r.Use(mw.LoadSession(a.sessions, a.cfg.SessionName))
	limiter := mw.NewRateLimiter(10, time.Minute)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	r.Get("/readyz", a.readyz)
	r.Get("/login", a.loginForm)
	r.With(limiter.Limit).Post("/login", a.login)
	r.Post("/logout", a.logout)
	r.Get("/mfa/verify", a.mfaVerifyForm)
	r.With(limiter.Limit).Post("/mfa/verify", a.mfaVerify)

	r.Group(func(p chi.Router) {
		p.Use(mw.RequireAuth)
		p.Use(mw.CSRF(a.sessions, a.cfg.SessionName))
		p.Get("/", a.dashboard)
		p.Get("/clients", a.clientList)
		p.Post("/clients", a.clientCreate)
		p.Get("/clients/new", a.clientNew)
		p.Get("/clients/{id}", a.clientDetail)
		p.Post("/clients/{id}/identities", a.identityAdd)
		p.Post("/identities/{id}/verify", a.identityVerify)
		p.Post("/clients/{id}/reps", a.repAdd)
		p.Post("/clients/{id}/files", a.fileUpload)
		p.Get("/files/{id}", a.fileDownload)
		p.Post("/files/{id}/ocr", a.fileOCR)
		p.Get("/requests", a.requestList)
		p.Post("/requests", a.requestCreate)
		p.Get("/requests/{id}", a.requestDetail)
		p.Post("/requests/{id}/move", a.requestMove)
		p.Post("/requests/{id}/corp", a.corpSave)
		p.Post("/requests/{id}/shareholders", a.shareholderAdd)
		p.Post("/requests/{id}/officers", a.officerAdd)
		p.Post("/requests/{id}/bo", a.boAdd)
		p.Get("/appointments", a.appointments)
		p.Post("/appointments", a.appointmentCreate)
		p.Post("/appointments/{id}/status", a.appointmentStatus)
		p.Get("/deeds", a.deedList)
		p.Post("/deeds", a.deedCreate)
		p.Post("/deeds/{id}/approve", a.deedApprove)
		p.Post("/deeds/{id}/review", a.deedToReview)
		p.Get("/deeds/{id}/pdf", a.deedPDF)
		p.Get("/deeds/{id}/docx", a.deedDOCX)
		p.Get("/wills", a.willsPage)
		p.Post("/wills", a.willCreate)
		p.Post("/wills/{id}/handover", a.willHandover)
		p.Get("/repertorium", a.repertorium)
		p.Get("/protocols", a.protocols)
		p.Post("/protocols", a.protocolCreate)
		p.Get("/legalisasi", a.legalisasi)
		p.Post("/legalisasi", a.legalisasiCreate)
		p.Post("/legalisasi/{id}/advance", a.legalisasiAdvance)
		p.Get("/gov", a.gov)
		p.Post("/gov", a.govCreate)
		p.Post("/gov/{id}/advance", a.govAdvance)
		p.Post("/gov/{id}/check", a.govCheck)
		p.Get("/finance", a.financePage)
		p.Post("/invoices", a.invoiceCreate)
		p.Post("/payments", a.paymentCreate)
		p.Get("/templates", a.templatesPage)
		p.Post("/templates", a.templateSave)
		p.Get("/templates/{id}", a.templateEdit)
		p.Post("/templates/{id}/approve", a.templateApprove)
		p.Post("/clauses", a.clauseSave)
		p.Get("/reports", a.reports)
		p.Get("/settings", a.settings)
		p.Post("/users", a.userCreate)
		p.Post("/users/{id}/link", a.userLink)
		p.Get("/mfa", a.mfaSetup)
		p.Post("/mfa", a.mfaEnable)
		p.Get("/portal", a.portal)
		p.Post("/notifications/read", a.notifRead)
		p.Get("/metrics", a.metrics)
	})
	return r
}

func (a *App) readyz(w http.ResponseWriter, _ *http.Request) {
	if a.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}
	if err := a.db.Ping(); err != nil {
		http.Error(w, "database unreachable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ready"))
}

// ---- auth handlers ----

func (a *App) loginForm(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	a.render(w, r, "login", "Masuk — NOTARYOS", nil)
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "form tidak valid", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	u, err := a.users.Authenticate(email, r.FormValue("password"))
	if err != nil {
		a.render(w, r, "login", "Masuk — NOTARYOS", map[string]string{"Error": err.Error(), "Email": email})
		return
	}
	// Step-up: accounts with TOTP enrolled must pass a second factor.
	if a.users.MFAEnabled(u.ID) {
		tok, _ := auth.NewCSRFToken()
		a.pendingMu.Lock()
		a.pending[tok] = pendingAuth{userID: u.ID, expires: time.Now().Add(5 * time.Minute)}
		a.pendingMu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionName + "_mfa", Value: tok, Path: "/",
			HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: a.cfg.TLSCookie, MaxAge: 300})
		http.Redirect(w, r, "/mfa/verify", http.StatusSeeOther)
		return
	}
	sess, err := a.sessions.Create(u.ID, u.Role)
	if err != nil {
		http.Error(w, "gagal membuat sesi", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: a.cfg.SessionName, Value: sess.Token, Path: "/",
		Expires: sess.ExpiresAt, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.cfg.TLSCookie,
	})
	uid := u.ID
	a.audit.Record(audit.Entry{ActorID: &uid, Action: "login", Entity: "user", EntityID: strconv.FormatInt(u.ID, 10)})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.cfg.SessionName); err == nil {
		a.sessions.Revoke(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionName, Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ---- dashboard ----

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	office := a.officeOf(r)
	pending, active, completed := 0, 0, 0
	var outstanding int64
	if a.db != nil {
		pending, active, completed, _ = a.reqs.Counts(office)
		outstanding, _ = a.fin.Outstanding(office)
	}
	recent, _ := a.reqs.List(office, "")
	if len(recent) > 8 {
		recent = recent[:8]
	}
	a.render(w, r, "dashboard", "Dashboard", map[string]any{
		"Pending": pending, "Active": active, "Completed": completed,
		"Outstanding": outstanding, "Recent": recent, "Role": mw.RoleOf(r),
	})
}

// ---- clients ----

func (a *App) clientList(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	list, _ := a.clients.List(a.officeOf(r), r.URL.Query().Get("q"), 50)
	if r.Header.Get("HX-Request") == "true" {
		_ = a.tmpl.ExecuteTemplate(w, "client_rows", list)
		return
	}
	a.render(w, r, "clients", "Klien", list)
}

func (a *App) clientNew(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "client_new", "Klien Baru", nil)
}

func (a *App) clientCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermCreateClient) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, err := a.clients.Create(clients.Client{
		OfficeID: a.officeOf(r),
		Kind:     r.FormValue("kind"),
		FullName: r.FormValue("full_name"),
		Email:    r.FormValue("email"),
		Phone:    r.FormValue("phone"),
		Address:  r.FormValue("address"),
		NPWP:     r.FormValue("npwp"),
	})
	if err != nil {
		a.render(w, r, "client_new", "Klien Baru", map[string]string{"Error": err.Error()})
		return
	}
	uid := a.uid(r)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "client.create", Entity: "client", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(id, 10)+"?flash=Klien+berhasil+dibuat", http.StatusSeeOther)
}

func (a *App) clientDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := a.clients.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	dups, _ := a.clients.FindDuplicates(a.officeOf(r), c.FullName, c.Phone)
	ids, _ := a.clients.Identities(c.ID)
	reps, _ := a.clients.Reps(c.ID)
	docs, _ := a.uploads.List(a.officeOf(r), "client", c.ID)
	a.render(w, r, "client_detail", "Detail Klien", map[string]any{
		"Client": c, "Duplicates": dups, "Identities": ids, "Reps": reps, "Files": docs,
		"CanVerify": rbac.Has(mw.RoleOf(r), rbac.PermVerifyIdentity),
	})
}

// ---- requests ----

func (a *App) requestList(w http.ResponseWriter, r *http.Request) {
	list, _ := a.reqs.List(a.officeOf(r), r.URL.Query().Get("status"))
	svcs, _ := a.reqs.Services()
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	a.render(w, r, "requests", "Permohonan", map[string]any{"Requests": list, "Services": svcs, "Clients": cl})
}

func (a *App) requestCreate(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	serviceID, _ := strconv.ParseInt(r.FormValue("service_id"), 10, 64)
	id, err := a.reqs.Create(a.officeOf(r), clientID, serviceID, r.FormValue("title"), r.FormValue("notes"), a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uid := a.uid(r)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "request.create", Entity: "service_request", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/requests?flash=Permohonan+dibuat", http.StatusSeeOther)
}

func (a *App) requestMove(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := a.reqs.Move(a.officeOf(r), id, r.FormValue("to"), a.uid(r), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uid := a.uid(r)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "request.move", Entity: "service_request", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("to")})
	// Notify assignee in background (best effort, never blocks the request).
	to := r.FormValue("to")
	a.pool.Submit(func(ctx context.Context) error {
		var assignee sql.NullInt64
		_ = a.db.QueryRow(`SELECT assignee_id FROM service_requests WHERE id=?`, id).Scan(&assignee)
		if assignee.Valid && (uid == nil || assignee.Int64 != *uid) {
			return notify.Create(a.db, assignee.Int64, "Permohonan #"+strconv.FormatInt(id, 10)+" → "+to, "Status diperbarui.")
		}
		return nil
	})
	http.Redirect(w, r, "/requests?flash=Status+diperbarui", http.StatusSeeOther)
}

// ---- deeds ----

func (a *App) deedList(w http.ResponseWriter, r *http.Request) {
	list, _ := a.deeds.List(a.officeOf(r))
	a.render(w, r, "deeds", "Akta", list)
}

func (a *App) deedCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermDraftDeed) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	var reqID *int64
	if v := r.FormValue("request_id"); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		reqID = &n
	}
	id, err := a.deeds.CreateDraft(a.officeOf(r), reqID, r.FormValue("title"), r.FormValue("kind"), nil, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uid := a.uid(r)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "deed.draft", Entity: "deed", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/deeds?flash=Draf+akta+dibuat", http.StatusSeeOther)
}

// ---- registers ----

func (a *App) repertorium(w http.ResponseWriter, r *http.Request) {
	type row struct {
		No, Date, Desc, Parties string
	}
	var rows []row
	if a.db != nil {
		rs, err := a.db.Query(`SELECT entry_no, entry_date, description, parties FROM repertorium WHERE office_id=? ORDER BY entry_no DESC LIMIT 200`, a.officeOf(r))
		if err == nil {
			defer rs.Close()
			for rs.Next() {
				var x row
				var p sql.NullString
				_ = rs.Scan(&x.No, &x.Date, &x.Desc, &p)
				if p.Valid {
					x.Parties = p.String
				}
				rows = append(rows, x)
			}
		}
	}
	a.render(w, r, "repertorium", "Repertorium", rows)
}

func (a *App) protocols(w http.ResponseWriter, r *http.Request) {
	type row struct {
		Bundle string
		Year   int
		Ref    string
	}
	var rows []row
	if a.db != nil {
		rs, err := a.db.Query(`SELECT bundle_no, year, minuta_ref FROM protocols WHERE office_id=? ORDER BY year DESC, bundle_no LIMIT 200`, a.officeOf(r))
		if err == nil {
			defer rs.Close()
			for rs.Next() {
				var x row
				var ref sql.NullString
				_ = rs.Scan(&x.Bundle, &x.Year, &ref)
				if ref.Valid {
					x.Ref = ref.String
				}
				rows = append(rows, x)
			}
		}
	}
	a.render(w, r, "protocols", "Protokol", rows)
}

func (a *App) legalisasi(w http.ResponseWriter, r *http.Request) {
	type row struct {
		ID                       int64
		Kind, Reg, Title, Status string
		Fee                      float64
	}
	var rows []row
	if a.db != nil {
		rs, err := a.db.Query(`SELECT id, kind, register_no, doc_title, fee, status FROM legalizations WHERE office_id=? ORDER BY id DESC LIMIT 200`, a.officeOf(r))
		if err == nil {
			defer rs.Close()
			for rs.Next() {
				var x row
				_ = rs.Scan(&x.ID, &x.Kind, &x.Reg, &x.Title, &x.Fee, &x.Status)
				rows = append(rows, x)
			}
		}
	}
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	a.render(w, r, "legalisasi", "Legalisasi & Waarmerking", map[string]any{"Rows": rows, "Clients": cl})
}

func (a *App) legalisasiCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermLegalize) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	var fee float64
	_, _ = fmt.Sscanf(r.FormValue("fee"), "%f", &fee)
	_, err := a.deeds.RegisterLegalization(a.officeOf(r), r.FormValue("kind"), r.FormValue("register_no"), clientID, r.FormValue("doc_title"), fee)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/legalisasi?flash=Register+tersimpan", http.StatusSeeOther)
}

// ---- gov assisted ----

func (a *App) gov(w http.ResponseWriter, r *http.Request) {
	subs, _ := a.govs.List(a.officeOf(r))
	a.render(w, r, "gov", "AHU & OSS (Asistensi)", map[string]any{"Subs": subs})
}

// ---- finance ----

func (a *App) financePage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewFinance) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	invs, _ := a.fin.List(a.officeOf(r))
	out, _ := a.fin.Outstanding(a.officeOf(r))
	month := time.Now().Format("2006-01")
	rev, _ := a.fin.Revenue(a.officeOf(r), month)
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	a.render(w, r, "finance", "Keuangan", map[string]any{
		"Invoices": invs, "Outstanding": out, "Revenue": rev, "Month": month, "Clients": cl,
	})
}

func (a *App) invoiceCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	descs := r.Form["desc[]"]
	qtys := r.Form["qty[]"]
	prices := r.Form["price[]"]
	var lines []finance.Line
	for i := range descs {
		if i >= len(qtys) || i >= len(prices) || descs[i] == "" {
			continue
		}
		q, _ := strconv.Atoi(qtys[i])
		p, _ := parseIDR(prices[i])
		lines = append(lines, finance.Line{Description: descs[i], Qty: q, UnitPrice: int64(p)})
	}
	_, err := a.fin.CreateInvoice(a.officeOf(r), clientID, nil, r.FormValue("number"), lines, 1100, r.FormValue("due_date"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/finance?flash=Invoice+dibuat", http.StatusSeeOther)
}

func (a *App) paymentCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	invID, _ := strconv.ParseInt(r.FormValue("invoice_id"), 10, 64)
	var amount float64
	_, _ = fmt.Sscanf(r.FormValue("amount"), "%f", &amount)
	if err := a.fin.RecordPayment(invID, int64(amount), r.FormValue("method"), r.FormValue("reference"), a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/finance?flash=Pembayaran+dicatat", http.StatusSeeOther)
}

// ---- misc pages ----

func (a *App) templatesPage(w http.ResponseWriter, r *http.Request) {
	tpls, _ := a.tpl.List(a.officeOf(r))
	clauses, _ := a.tpl.Clauses(a.officeOf(r))
	a.render(w, r, "templates", "Template Dokumen", map[string]any{"Templates": tpls, "Clauses": clauses})
}

func (a *App) reports(w http.ResponseWriter, r *http.Request) {
	pending, active, completed := 0, 0, 0
	var out int64
	if a.db != nil {
		pending, active, completed, _ = a.reqs.Counts(a.officeOf(r))
		out, _ = a.fin.Outstanding(a.officeOf(r))
	}
	a.render(w, r, "reports", "Laporan", map[string]any{
		"Pending": pending, "Active": active, "Completed": completed, "Outstanding": out,
	})
}

func (a *App) settings(w http.ResponseWriter, r *http.Request) {
	list, _ := a.users.List(a.officeOf(r))
	a.render(w, r, "settings", "Pengaturan", map[string]any{"Users": list, "Roles": rbac.Roles()})
}

// portal shows the signed-in client's own requests (strict server-side isolation)
// or all for staff.
func (a *App) portal(w http.ResponseWriter, r *http.Request) {
	role := mw.RoleOf(r)
	if role == rbac.RoleClient {
		id := a.uid(r)
		var clientID int64
		if id != nil {
			clientID = a.users.LinkedClient(*id)
		}
		if clientID == 0 {
			a.render(w, r, "portal", "Portal Klien", map[string]any{"Note": "Akun Anda belum ditautkan ke data klien. Hubungi kantor."})
			return
		}
		rows, err := a.db.Query(`SELECT r.id, r.title, r.status FROM service_requests r
			WHERE r.client_id=? AND r.office_id=? ORDER BY r.id DESC`, clientID, a.officeOf(r))
		if err != nil {
			http.Error(w, "gagal memuat data", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var list []map[string]any
		for rows.Next() {
			var id int64
			var title, status string
			_ = rows.Scan(&id, &title, &status)
			list = append(list, map[string]any{"ID": id, "Title": title, "Status": status})
		}
		a.render(w, r, "portal", "Portal Klien", map[string]any{"Mine": list})
		return
	}
	list, _ := a.reqs.List(a.officeOf(r), "")
	a.render(w, r, "portal", "Portal Klien", map[string]any{"Requests": list})
}
