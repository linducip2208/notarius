package app

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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
	"notaryos/internal/wib"
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
	Title      string
	Role       string
	UserName   string
	OfficeName string
	WIBDate    string
	Pending    int
	Unread     int
	TopNotifs  []map[string]any
	Flash      string
	CSRF       string
	Data       any
}

// New builds the app. db may be nil only in tests that never hit handlers.
func New(cfg config.Config, db *sql.DB, log *slog.Logger) (*App, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"can":     func(role, perm string) bool { return rbac.Has(role, perm) },
		"wib":     func(t time.Time) string { return wib.Format(t) },
		"wibdate": func(t time.Time) string { return wib.Date(t) },
		"css":     func(s string) template.CSS { return template.CSS(s) },
		"lower":   func(s string) string { return strings.ToLower(s) },
	}).ParseFS(tmplFS, "templates/*.gohtml")
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
	mw.SetTrustedProxies(cfg.TrustedProxies)
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
	username, officeName := "", ""
	pending, unread := 0, 0
	var topNotifs []map[string]any
	if id := a.uid(r); id != nil && a.db != nil {
		_ = a.db.QueryRow(`SELECT u.name, COALESCE(o.name,'') FROM users u LEFT JOIN offices o ON o.id=u.office_id WHERE u.id=?`, *id).Scan(&username, &officeName)
		office := a.officeOf(r)
		pending, _, _, _ = a.reqs.Counts(office)
		unread = notify.Unread(a.db, *id)
		topNotifs, _ = notify.List(a.db, *id)
		if len(topNotifs) > 5 {
			topNotifs = topNotifs[:5]
		}
	}
	flash := r.URL.Query().Get("flash")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, name, Page{Title: title, Role: role, UserName: username, OfficeName: officeName, WIBDate: wib.Format(time.Now()), Pending: pending, Unread: unread, TopNotifs: topNotifs, Flash: flash, CSRF: mw.IssueCSRF(w, r, a.cfg.SessionName, a.cfg.TLSCookie), Data: data}); err != nil {
		a.log.Error("render", "page", name, "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// forbidden renders 403 with guidance instead of plain text.
func (a *App) forbidden(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
	a.render(w, r, "forbidden", "Akses Ditolak", map[string]string{
		"Role": mw.RoleOf(r),
	})
}

// notFound renders 404.
func (a *App) notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	a.render(w, r, "notfound", "Halaman Tidak Ditemukan", map[string]string{
		"Path": r.URL.Path,
	})
}

// Router builds all routes.
func (a *App) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(mw.SecurityHeaders)
	r.Use(mw.LoadSession(a.sessions, a.cfg.SessionName))
	limiter := mw.NewRateLimiter(10, time.Minute)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	r.Get("/readyz", a.readyz)
	r.Get("/", a.landing)
	r.Get("/robots.txt", a.robotsTxt)
	r.Get("/sitemap.xml", a.sitemapXML)
	r.Get("/llms.txt", a.llmsTxt)
	r.Get("/layanan", a.layananIndex)
	r.Get("/layanan/{code}", a.layananDetail)
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))
	r.Get("/login", a.loginForm)
	r.With(limiter.Limit).Post("/login", a.login)
	r.Get("/mfa/verify", a.mfaVerifyForm)
	r.With(limiter.Limit).Post("/mfa/verify", a.mfaVerify)

	r.Group(func(p chi.Router) {
		p.Use(mw.RequireAuth)
		p.Use(mw.CSRF(a.sessions, a.cfg.SessionName))
		p.Post("/logout", a.logout)
		p.Get("/app", a.dashboard)
		p.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/app", http.StatusFound)
		})
		p.Get("/clients", a.clientList)
		p.Get("/clients/export", a.clientExport)
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
		p.Get("/deeds/{id}", a.deedDetail)
		p.Post("/deeds/{id}/version", a.deedSaveVersion)
		p.Post("/deeds/{id}/approve", a.deedApprove)
		p.Post("/deeds/{id}/review", a.deedToReview)
		p.Get("/deeds/{id}/pdf", a.deedPDF)
		p.Get("/deeds/{id}/docx", a.deedDOCX)
		p.Get("/wills", a.willsPage)
		p.Post("/wills", a.willCreate)
		p.Post("/wills/{id}/handover", a.willHandover)
		p.Get("/repertorium", a.repertorium)
		p.Get("/repertorium/export", a.repertoriumExport)
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
		p.Get("/roles", a.rolesPage)
		p.Get("/mfa", a.mfaSetup)
		p.Post("/mfa", a.mfaEnable)
		p.Get("/portal", a.portal)
		p.Post("/notifications/read", a.notifRead)
		p.Get("/metrics", a.metrics)
		p.Get("/documents", a.documents)
		p.Get("/audit", a.auditPage)
		p.Get("/office", a.officePage)
		p.Post("/office", a.officeSave)
		p.Get("/tasks", a.tasksPage)
		p.Post("/tasks", a.taskCreate)
		p.Post("/tasks/{id}/done", a.taskDone)
		p.Get("/expenses", a.expensesPage)
		p.Post("/expenses", a.expenseCreate)
		p.Get("/tariffs", a.tariffsPage)
		p.Post("/tariffs", a.tariffSave)
		p.Get("/search", a.search)
	})
	r.NotFound(a.notFound)
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
		http.Redirect(w, r, "/app", http.StatusSeeOther)
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
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.cfg.SessionName); err == nil {
		a.sessions.Revoke(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionName, Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ---- public landing ----

// landing renders the public marketing page. Logged-in users go to /app.
func (a *App) landing(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != "" {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "landing", nil); err != nil {
		a.log.Error("render", "page", "landing", "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// robotsTxt allows all crawlers on public marketing routes only.
func (a *App) robotsTxt(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "User-agent: *")
	fmt.Fprintln(w, "Allow: /")
	fmt.Fprintln(w, "Allow: /layanan")
	fmt.Fprintln(w, "Allow: /assets/")
	fmt.Fprintln(w, "Disallow: /app")
	fmt.Fprintln(w, "Disallow: /clients")
	fmt.Fprintln(w, "Disallow: /requests")
	fmt.Fprintln(w, "Disallow: /deeds")
	fmt.Fprintln(w, "Disallow: /finance")
	fmt.Fprintln(w, "Disallow: /settings")
	fmt.Fprintln(w, "Disallow: /login")
	fmt.Fprintln(w, "Sitemap: /sitemap.xml")
}

// sitemapXML lists public marketing URLs (landing + layanan + statis).
func (a *App) sitemapXML(w http.ResponseWriter, r *http.Request) {
	base := "https://" + r.Host
	if r.Host == "" || strings.HasPrefix(r.Host, "127.") || strings.HasPrefix(r.Host, "localhost") {
		base = "https://notarius.example"
	}
	type url struct{ Loc, Changefreq, Priority string }
	urls := []url{
		{base + "/", "weekly", "1.0"},
		{base + "/layanan", "weekly", "0.9"},
	}
	if a.db != nil {
		rows, _ := a.db.Query(`SELECT code FROM services WHERE is_active=1 ORDER BY code`)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var code string
				_ = rows.Scan(&code)
				urls = append(urls, url{base + "/layanan/" + strings.ToLower(code), "monthly", "0.8"})
			}
		}
	}
	for _, s := range []string{"#fitur", "#solusi", "#harga", "#faq"} {
		urls = append(urls, url{base + "/" + s, "monthly", "0.5"})
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`))
	for _, u := range urls {
		fmt.Fprintf(w, "\n<url><loc>%s</loc><changefreq>%s</changefreq><priority>%s</priority></url>", u.Loc, u.Changefreq, u.Priority)
	}
	w.Write([]byte("\n</urlset>"))
}

// llmsTxt exposes a crawler- and LLM-friendly summary (GEO: factual, no hype).
func (a *App) llmsTxt(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Write([]byte(`# NOTARIUS

> Sistem manajemen kantor Notaris Indonesia (Go + MySQL + Tabler).
> Catatan: informasi produk di bawah ini; bukan nasihat hukum.

## Produk
- Klien perorangan & badan hukum, verifikasi identitas manusia, beneficial owner.
- Permohonan layanan 9 status; akta draf berversi dari template terapproved;
  pengesahan notaris mengunci final + repertorium otomatis.
- Register: repertorium, legalisasi/waarmerking/salinan/kutipan, protokol 25 tahun, wasiat terbatas.
- Keuangan: invoice multi-baris anti-overpay, pembayaran, piutang, pengeluaran, tarif.
- Keamanan: bcrypt-12, sesi server-side, MFA TOTP, RBAC 9 peran/19 izin, NIK AES-256-GCM,
  audit imutabel, HSTS+CSP+CSRF+rate-limit.
- Bukan PPAT/advokat. Tanpa API resmi AHU/OSS (asistensi manual + bukti).
- Harga: konsultasi via penawaran resmi (tidak ada harga publik).

## Halaman
- / (landing) · /layanan (indeks) · /layanan/{kode} (detail per layanan, syarat + alur)
- Aplikasi butuh akun: /login (demo: admin@contoh.id — sandi demo di halaman login).
`))
}

// layananIndex lists all active services (PSEO hub page).
func (a *App) layananIndex(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != "" {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	type svc struct {
		Code, Category, Name string
		Price                float64
	}
	var out []svc
	groups := map[string][]svc{}
	var order []string
	if a.db != nil {
		rows, _ := a.db.Query(`SELECT code, category, name, base_price FROM services WHERE is_active=1 ORDER BY category, name`)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var s svc
				_ = rows.Scan(&s.Code, &s.Category, &s.Name, &s.Price)
				out = append(out, s)
				if _, ok := groups[s.Category]; !ok {
					order = append(order, s.Category)
				}
				groups[s.Category] = append(groups[s.Category], s)
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "layanan_index", map[string]any{"Groups": groups, "Order": order, "Total": len(out)}); err != nil {
		a.log.Error("render", "page", "layanan_index", "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// layananDetail renders one programmatic SEO page per service code.
func (a *App) layananDetail(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != "" {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	code := strings.ToUpper(chi.URLParam(r, "code"))
	var name, category string
	var price float64
	var reqRaw sql.NullString
	err := a.db.QueryRow(`SELECT name, category, base_price, requirements FROM services WHERE code=? AND is_active=1`, code).Scan(&name, &category, &price, &reqRaw)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var reqs []string
	if reqRaw.Valid && reqRaw.String != "" {
		var arr []string
		if json.Unmarshal([]byte(reqRaw.String), &arr) == nil {
			reqs = arr
		} else {
			reqs = []string{reqRaw.String}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "layanan_detail", map[string]any{
		"Code": code, "Name": name, "Category": category, "Price": price, "Reqs": reqs,
	}); err != nil {
		a.log.Error("render", "page", "layanan_detail", "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// ---- dashboard ----

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	office := a.officeOf(r)
	pending, active, completed := 0, 0, 0
	var outstanding int64
	var monthRev int64
	month := time.Now().Format("2006-01")
	if a.db != nil {
		pending, active, completed, _ = a.reqs.Counts(office)
		outstanding, _ = a.fin.Outstanding(office)
		monthRev, _ = a.fin.Revenue(office, month)
	}
	recent, _ := a.reqs.List(office, "", 0)
	if len(recent) > 8 {
		recent = recent[:8]
	}
	// Agenda hari ini (mockup: Agenda Hari Ini).
	type agenda struct {
		At, Client, Purpose, Status string
	}
	var today []agenda
	if a.db != nil {
		rows, _ := a.db.Query(`SELECT a.scheduled_at, c.full_name, a.purpose, a.status
			FROM appointments a JOIN clients c ON c.id=a.client_id
			WHERE a.office_id=? AND DATE(a.scheduled_at)=CURDATE() ORDER BY a.scheduled_at LIMIT 10`, office)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var g agenda
				_ = rows.Scan(&g.At, &g.Client, &g.Purpose, &g.Status)
				g.At = g.At[11:16]
				today = append(today, g)
			}
		}
	}
	// Periode grafik: ?range=6|12 (default 6).
	n := 6
	if q := r.URL.Query().Get("range"); q == "12" {
		n = 12
	}
	months, reqSeries, deedSeries := a.lastNMonths(office, n)
	reqPts, deedPts := svgSeries(reqSeries, deedSeries)
	type dot struct {
		X, Y, V int
		M       string
	}
	mkDots := func(s []int) []dot {
		max, span := 1, len(s)-1
		if span < 1 {
			span = 1
		}
		for _, v := range s {
			if v > max {
				max = v
			}
		}
		var out []dot
		for i, v := range s {
			out = append(out, dot{X: 20 + i*(560/span), Y: 150 - v*130/max, V: v, M: months[i]})
		}
		return out
	}
	// Total klien + akta bulan ini + jadwal hari ini (kartu statistik, tanpa persenan).
	var totalClients, monthDeeds int
	if a.db != nil {
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM clients WHERE office_id=?`, office).Scan(&totalClients)
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE office_id=? AND DATE_FORMAT(created_at,'%Y-%m')=?`, office, month).Scan(&monthDeeds)
	}
	var conic string
	// Donat komposisi: Akta korporat vs Perjanjian vs Legalisasi vs Lainnya.
	type seg struct {
		Label, Color string
		Pct          int
	}
	var donut []seg
	if a.db != nil {
		var corp, agree, leg int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE office_id=? AND kind IN ('PT','CV','Firma','Yayasan')`, office).Scan(&corp)
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE office_id=? AND kind IN ('Agreement','POA','Prenup','Will','Other')`, office).Scan(&agree)
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM legalizations WHERE office_id=?`, office).Scan(&leg)
		tot := corp + agree + leg
		if tot == 0 {
			tot = 1
		}
		pc, pl, pa := corp*100/tot, leg*100/tot, agree*100/tot
		donut = []seg{
			{"Akta", "#206bc4", pc},
			{"Legalitas", "#e9a23b", pl},
			{"Perjanjian", "#16a37b", pa},
		}
		conic = fmt.Sprintf("conic-gradient(#206bc4 0 %d%%,#e9a23b %d%% %d%%,#16a37b %d%% 100%%)", pc, pc, pc+pl, pc+pl)
	}
	// Pekerjaan menunggu tindakan: berkas kurang, draf review, invoice belum lunas, janji mendatang.
	type work struct {
		Kind, Text, URL string
	}
	var pendingWork []work
	if a.db != nil {
		rows, _ := a.db.Query(`SELECT id, title FROM service_requests WHERE office_id=? AND status='NEED_DOCS' ORDER BY id DESC LIMIT 5`, office)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var t string
				_ = rows.Scan(&id, &t)
				pendingWork = append(pendingWork, work{"Berkas", t + " — dokumen belum lengkap", "/requests/" + strconv.FormatInt(id, 10)})
			}
		}
		rows2, _ := a.db.Query(`SELECT id, title FROM deed_drafts WHERE office_id=? AND status='IN_REVIEW' ORDER BY id DESC LIMIT 5`, office)
		if rows2 != nil {
			defer rows2.Close()
			for rows2.Next() {
				var id int64
				var t string
				_ = rows2.Scan(&id, &t)
				pendingWork = append(pendingWork, work{"Review", t + " — menunggu persetujuan notaris", "/deeds/" + strconv.FormatInt(id, 10)})
			}
		}
		rows3, _ := a.db.Query(`SELECT number FROM invoices WHERE office_id=? AND status IN ('ISSUED','PARTIAL') ORDER BY id DESC LIMIT 5`, office)
		if rows3 != nil {
			defer rows3.Close()
			for rows3.Next() {
				var num string
				_ = rows3.Scan(&num)
				pendingWork = append(pendingWork, work{"Bayar", "Invoice " + num + " — pembayaran tertunda", "/finance"})
			}
		}
	}
	// Aktivitas terbaru dari audit log (pelaku, waktu, objek, tautan).
	type activity struct {
		Action, Actor, At, Entity, EntityID string
		URL                                 string
	}
	var feed []activity
	if a.db != nil {
		rows, _ := a.db.Query(`SELECT a.action, COALESCE(u.name,'sistem'), a.created_at, a.entity, a.entity_id
			FROM audit_logs a LEFT JOIN users u ON u.id=a.actor_id ORDER BY a.id DESC LIMIT 10`)
		if rows != nil {
			defer rows.Close()
			link := map[string]string{"client": "/clients/", "service_request": "/requests/", "deed": "/deeds/", "invoice": "/finance", "will": "/wills", "gov_submission": "/gov"}
			for rows.Next() {
				var x activity
				_ = rows.Scan(&x.Action, &x.Actor, &x.At, &x.Entity, &x.EntityID)
				if base, ok := link[x.Entity]; ok {
					x.URL = base
					if x.Entity == "client" || x.Entity == "service_request" || x.Entity == "deed" {
						x.URL += x.EntityID
					}
				}
				feed = append(feed, x)
			}
		}
	}
	// Notifikasi terbaru pengguna.
	var notifs []map[string]any
	if id := a.uid(r); id != nil {
		notifs, _ = notify.List(a.db, *id)
		if len(notifs) > 5 {
			notifs = notifs[:5]
		}
	}
	a.render(w, r, "dashboard", "Dashboard", map[string]any{
		"Pending": pending, "Active": active, "Completed": completed,
		"Outstanding": outstanding, "Revenue": monthRev, "Recent": recent,
		"Role": mw.RoleOf(r), "Today": today, "Notifs": notifs,
		"Months": months, "ReqPts": reqPts, "DeedPts": deedPts, "RangeN": n,
		"ReqDots": mkDots(reqSeries), "DeedDots": mkDots(deedSeries),
		"Clients": totalClients, "MonthDeeds": monthDeeds, "Range": r.URL.Query().Get("range"),
		"Donut": donut, "Conic": conic, "Work": pendingWork, "Feed": feed,
	})
}

// svgSeries converts two monthly series into SVG polyline point strings (600x160).
func svgSeries(a, b []int) (string, string) {
	max := 1
	for _, v := range append(append([]int{}, a...), b...) {
		if v > max {
			max = v
		}
	}
	span := len(a) - 1
	if span < 1 {
		span = 1
	}
	pts := func(s []int) string {
		out := ""
		for i, v := range s {
			x := 20 + i*(560/span)
			y := 150 - v*130/max
			out += strconv.Itoa(x) + "," + strconv.Itoa(y) + " "
		}
		return out
	}
	return pts(a), pts(b)
}

// lastNMonths returns month labels + request counts + completed-deed counts.
func (a *App) lastNMonths(office int64, n int) (labels []string, reqs []int, deeds []int) {
	if n != 12 {
		n = 6
	}
	now := time.Now()
	for i := n - 1; i >= 0; i-- {
		m := now.AddDate(0, -i, 0)
		labels = append(labels, m.Format("Jan"))
		ym := m.Format("2006-01")
		var rc, dc int
		if a.db != nil {
			_ = a.db.QueryRow(`SELECT COUNT(*) FROM service_requests WHERE office_id=? AND DATE_FORMAT(created_at,'%Y-%m')=?`, office, ym).Scan(&rc)
			_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE office_id=? AND DATE_FORMAT(created_at,'%Y-%m')=? AND status IN ('APPROVED','SIGNED','ARCHIVED')`, office, ym).Scan(&dc)
		}
		reqs = append(reqs, rc)
		deeds = append(deeds, dc)
	}
	return labels, reqs, deeds
}

// ---- clients ----

func (a *App) clientList(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	q := r.URL.Query().Get("q")
	kind := r.URL.Query().Get("kind")
	sort := r.URL.Query().Get("sort")
	page := config.MustInt(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	perPage := config.MustInt(r.URL.Query().Get("per_page"), 20)
	if perPage != 10 && perPage != 20 && perPage != 50 {
		perPage = 20
	}
	list, total, _ := a.clients.ListFiltered(a.officeOf(r), q, kind, perPage, (page-1)*perPage, sort)
	pages := (total + perPage - 1) / perPage
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	prev, next := page-1, page+1
	if prev < 1 {
		prev = 1
	}
	if next > pages {
		next = pages
	}
	qs := "q=" + q + "&kind=" + kind + "&sort=" + sort + "&per_page=" + strconv.Itoa(perPage)
	if r.Header.Get("HX-Request") == "true" {
		_ = a.tmpl.ExecuteTemplate(w, "client_rows", list)
		return
	}
	a.render(w, r, "clients", "Daftar Klien", map[string]any{
		"List": list, "Q": q, "Kind": kind, "Sort": sort, "PerPage": perPage,
		"Page": page, "Pages": pages, "Total": total, "Prev": prev, "Next": next, "QS": qs,
	})
}

// clientExport downloads the filtered client list as CSV (permission-gated).
func (a *App) clientExport(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	list, _, _ := a.clients.ListFiltered(a.officeOf(r), r.URL.Query().Get("q"), r.URL.Query().Get("kind"), 200, 0, r.URL.Query().Get("sort"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"klien.csv\"")
	w.Write([]byte("\xef\xbb\xbf"))
	fmt.Fprintln(w, "ID,Nama,Jenis,Telepon,Email,NPWP,Dibuat")
	for _, c := range list {
		fmt.Fprintf(w, "%d,\"%s\",%s,%s,%s,%s,%s\n", c.ID,
			strings.ReplaceAll(c.FullName, `"`, `""`), c.Kind, c.Phone, c.Email, c.NPWP, c.CreatedAt)
	}
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
	// Riwayat layanan + akta + audit klien ini.
	type svcRow struct {
		ID, Title, Service, Status string
	}
	var svcs []svcRow
	if rows, err := a.db.Query(`SELECT r.id, r.title, s.name, r.status FROM service_requests r
		JOIN services s ON s.id=r.service_id WHERE r.client_id=? AND r.office_id=? ORDER BY r.id DESC LIMIT 20`, id, a.officeOf(r)); err == nil {
		defer rows.Close()
		for rows.Next() {
			var x svcRow
			_ = rows.Scan(&x.ID, &x.Title, &x.Service, &x.Status)
			svcs = append(svcs, x)
		}
	}
	type auditRow struct {
		Action, At string
	}
	var trail []auditRow
	if rows, err := a.db.Query(`SELECT action, created_at FROM audit_logs WHERE entity='client' AND entity_id=? ORDER BY id DESC LIMIT 20`, strconv.FormatInt(id, 10)); err == nil {
		defer rows.Close()
		for rows.Next() {
			var x auditRow
			_ = rows.Scan(&x.Action, &x.At)
			trail = append(trail, x)
		}
	}
	a.render(w, r, "client_detail", "Detail Klien", map[string]any{
		"Client": c, "Duplicates": dups, "Identities": ids, "Reps": reps, "Files": docs,
		"Services": svcs, "Trail": trail,
		"CanVerify": rbac.Has(mw.RoleOf(r), rbac.PermVerifyIdentity),
	})
}

// ---- requests ----

func (a *App) requestList(w http.ResponseWriter, r *http.Request) {
	clientID, _ := strconv.ParseInt(r.URL.Query().Get("client"), 10, 64)
	list, _ := a.reqs.List(a.officeOf(r), r.URL.Query().Get("status"), clientID)
	svcs, _ := a.reqs.Services()
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	var clientName string
	if clientID != 0 {
		if c, err := a.clients.Get(a.officeOf(r), clientID); err == nil {
			clientName = c.FullName
		}
	}
	a.render(w, r, "requests", "Permohonan", map[string]any{"Requests": list, "Services": svcs, "Clients": cl, "ClientName": clientName})
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
	tpls, _ := a.tpl.List(a.officeOf(r))
	a.render(w, r, "deeds", "Akta", map[string]any{"Deeds": list, "Templates": tpls})
}

func (a *App) deedCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermDraftDeed) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	var reqID *int64
	if v := r.FormValue("request_id"); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		reqID = &n
	}
	var tplID *int64
	if v := r.FormValue("template_id"); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		tplID = &n
	}
	id, err := a.deeds.CreateDraft(a.officeOf(r), reqID, r.FormValue("title"), r.FormValue("kind"), tplID, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uid := a.uid(r)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "deed.draft", Entity: "deed", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/deeds/"+strconv.FormatInt(id, 10)+"?flash=Draf+akta+dibuat", http.StatusSeeOther)
}

// ---- registers ----

func (a *App) repertorium(w http.ResponseWriter, r *http.Request) {
	type row struct {
		No, Date, Desc, Parties string
	}
	q := r.URL.Query()
	year, month, search := q.Get("year"), q.Get("month"), q.Get("q")
	where := ` WHERE office_id=?`
	args := []any{a.officeOf(r)}
	if year != "" {
		where += ` AND YEAR(entry_date)=?`
		args = append(args, year)
	}
	if month != "" {
		where += ` AND MONTH(entry_date)=?`
		args = append(args, month)
	}
	if search != "" {
		where += ` AND (entry_no LIKE ? OR description LIKE ? OR parties LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	var rows []row
	if a.db != nil {
		rs, err := a.db.Query(`SELECT entry_no, entry_date, description, parties FROM repertorium`+where+` ORDER BY entry_no DESC LIMIT 200`, args...)
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
	a.render(w, r, "repertorium", "Repertorium", map[string]any{
		"Rows": rows, "Year": year, "Month": month, "Q": search,
	})
}

// repertoriumExport downloads the filtered register as CSV (server-side numbering).
func (a *App) repertoriumExport(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) && !rbac.Has(mw.RoleOf(r), rbac.PermViewReports) {
		a.forbidden(w, r)
		return
	}
	q := r.URL.Query()
	year, month, search := q.Get("year"), q.Get("month"), q.Get("q")
	where := ` WHERE office_id=?`
	args := []any{a.officeOf(r)}
	if year != "" {
		where += ` AND YEAR(entry_date)=?`
		args = append(args, year)
	}
	if month != "" {
		where += ` AND MONTH(entry_date)=?`
		args = append(args, month)
	}
	if search != "" {
		where += ` AND (entry_no LIKE ? OR description LIKE ? OR parties LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"repertorium.csv\"")
	w.Write([]byte("\xef\xbb\xbf"))
	fmt.Fprintln(w, "No,Tanggal,Uraian,Para Pihak")
	rs, err := a.db.Query(`SELECT entry_no, entry_date, description, parties FROM repertorium`+where+` ORDER BY entry_no`, args...)
	if err != nil {
		return
	}
	defer rs.Close()
	for rs.Next() {
		var no, date, desc string
		var p sql.NullString
		_ = rs.Scan(&no, &date, &desc, &p)
		parties := ""
		if p.Valid {
			parties = p.String
		}
		fmt.Fprintf(w, "%s,%s,\"%s\",\"%s\"\n", no, date,
			strings.ReplaceAll(desc, `"`, `""`), strings.ReplaceAll(parties, `"`, `""`))
	}
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "legalisasi.create", Entity: "legalization", EntityID: r.FormValue("register_no")})
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "invoice.create", Entity: "invoice", EntityID: r.FormValue("number")})
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
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "payment.create", Entity: "invoice", EntityID: strconv.FormatInt(invID, 10)})
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
	list, _ := a.reqs.List(a.officeOf(r), "", 0)
	a.render(w, r, "portal", "Portal Klien", map[string]any{"Requests": list})
}
