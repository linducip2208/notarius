package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/auth"
	"notaryos/internal/clients"
	"notaryos/internal/cms"
	"notaryos/internal/config"
	"notaryos/internal/corp"
	"notaryos/internal/cryptoxx"
	"notaryos/internal/deeds"
	"notaryos/internal/deliver"
	"notaryos/internal/dms"
	"notaryos/internal/doctpl"
	"notaryos/internal/files"
	"notaryos/internal/finance"
	"notaryos/internal/gov"
	mw "notaryos/internal/middleware"
	"notaryos/internal/money"
	"notaryos/internal/notify"
	"notaryos/internal/ocr"
	"notaryos/internal/plan"
	"notaryos/internal/rbac"
	"notaryos/internal/remind"
	"notaryos/internal/requests"
	"notaryos/internal/safehtml"
	"notaryos/internal/storage"
	"notaryos/internal/users"
	"notaryos/internal/wib"
	"notaryos/internal/workers"
)

//go:embed templates/*.gohtml
var tmplFS embed.FS

// reNIKFull menutupi NIK 16 digit berurutan pada tampilan (UU PDP: NIK
// penuh tidak boleh tampil di layar/log — hanya 4 digit terakhir).
var reNIKFull = regexp.MustCompile(`\d{16}`)

// maskNIKDisp menampilkan NIK/NPWP/No.KK dalam bentuk mask:
// "•••• •••••••• 1234" (hanya 4 digit terakhir yang terlihat).
func maskNIKDisp(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) < 4 {
		return "••••"
	}
	return "•••• •••••••• " + s[len(s)-4:]
}

// redactNIKDisp menutupi setiap run 16 digit pada teks bebas (mis. teks
// mentah OCR) dengan format mask yang sama, tanpa mengubah data tersimpan.
func redactNIKDisp(s string) string {
	return reNIKFull.ReplaceAllStringFunc(s, maskNIKDisp)
}

// App wires the modular monolith: router, stores, templates.
type App struct {
	cfg      config.Config
	db       *sql.DB
	log      *slog.Logger
	sessions auth.SessionStore
	tmpl     *template.Template
	audit    *audit.Logger
	users    *users.Store
	clients  *clients.Store
	reqs     *requests.Store
	deeds    *deeds.Store
	dms      *dms.Store
	fin      *finance.Store
	wills    *deeds.WillStore
	corp     *corp.Store
	tpl      *doctpl.Store
	govs     *gov.Store
	uploads  *files.Store
	web      *cms.Store
	ocr      ocr.Provider
	files    storage.Store
	pool     *workers.Pool
}

// Page is template data for every Tabler page.
type Page struct {
	Title      string
	Role       string
	UserName   string
	OfficeName string
	Acting     bool
	WIBDate    string
	Pending    int
	Unread     int
	TopNotifs  []map[string]any
	Flash      string
	CSRF       string
	Data       any
	Brand      BrandingView
}

// BrandingView carries white-label values into every template (safe, validated).
type BrandingView struct {
	AppName, ShortName, Tagline  string
	Primary, Secondary, Accent   string
	SidebarBG, HeaderBG          string
	ButtonRadius, FontStack      string
	LogoURL, FaviconURL, LoginBG string
	FooterText, CopyrightText    string
}

// New builds the app. db may be nil only in tests that never hit handlers.
func New(cfg config.Config, db *sql.DB, log *slog.Logger) (*App, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"can":     func(role, perm string) bool { return rbac.Has(role, perm) },
		"wib":     func(t time.Time) string { return wib.Format(t) },
		"wibdate": func(t time.Time) string { return wib.Date(t) },
		"css": func(s string) template.CSS {
			// Allowlist ketat: hanya karakter aman font-stack (huruf, koma, spasi, kutip, strip).
			var b strings.Builder
			for _, r := range s {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ',' || r == ' ' || r == '\'' || r == '"' || r == '-' {
					b.WriteRune(r)
				}
			}
			return template.CSS(b.String())
		},
		"lower":     func(s string) string { return strings.ToLower(s) },
		"hasPrefix": func(s, p string) bool { return strings.HasPrefix(s, p) },
		"hasSuffix": func(s, p string) bool { return strings.HasSuffix(s, p) },
		"maskNIK":   func(s string) string { return maskNIKDisp(s) },
		"redactNIK": func(s string) string { return redactNIKDisp(s) },
		"safeHTML":  func(s string) template.HTML { return template.HTML(safehtml.Sanitize(s)) },
		"cmsHTML":   func(s string) template.HTML { return template.HTML(cms.SanitizeHTML(s)) },
		"plaintext": func(s string) string { return safehtml.ToText(s) },
		"sub":       func(a, b int64) int64 { return a - b },
		// refLink renders a safe deep-link for task context references.
		"refLink": func(typ string, id int64) template.HTML {
			if id <= 0 {
				return ""
			}
			var href, label string
			switch typ {
			case "deed":
				href, label = "/deeds/"+strconv.FormatInt(id, 10), "akta"
			case "request":
				href, label = "/requests/"+strconv.FormatInt(id, 10), "permohonan"
			case "client":
				href, label = "/clients/"+strconv.FormatInt(id, 10), "klien"
			case "file":
				href, label = "/documents/"+strconv.FormatInt(id, 10), "berkas"
			default:
				return ""
			}
			return template.HTML(`<a href="` + href + `">` + label + ` #` + strconv.FormatInt(id, 10) + `</a>`)
		},
	}).ParseFS(tmplFS, "templates/*.gohtml")
	if err != nil {
		return nil, err
	}
	fstore, err := storage.FromEnv(cfg.StorageDir)
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
		a.dms = dms.NewStore(db)
		a.fin = finance.NewStore(db)
		a.wills = deeds.NewWillStore(db)
		a.corp = corp.NewStore(db)
		a.tpl = doctpl.NewStore(db)
		a.govs = gov.NewStore(db)
		a.uploads = files.NewStore(db, a.files, a.ocr)
		a.web = cms.NewStore(db)
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
		// Pengingat operasional harian (janji H-1, invoice telat, tugas mendesak).
		go func() {
			t := time.NewTicker(24 * time.Hour)
			defer t.Stop()
			for range t.C {
				a.runReminders()
			}
		}()
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

// runReminders executes the daily reminder scan for every office.
func (a *App) runReminders() (appt, overdue, tasks, sla int) {
	if a.db == nil {
		return 0, 0, 0, 0
	}
	rows, err := a.db.Query(`SELECT id FROM offices`)
	if err != nil {
		return 0, 0, 0, 0
	}
	defer rows.Close()
	for rows.Next() {
		var oid int64
		_ = rows.Scan(&oid)
		pa, po, pt, ps, _ := remind.RunOnce(a.db, oid, deliver.FromEnv())
		appt, overdue, tasks, sla = appt+pa, overdue+po, tasks+pt, sla+ps
		// Orang & dokumen: ultah, identitas kedaluwarsa, review wasiat tahunan (tercatat terpisah).
		pb, pe, pw, _ := remind.RunPeople(a.db, oid, deliver.FromEnv())
		if n := pb + pe + pw; n > 0 {
			a.audit.Record(audit.Entry{Action: "reminders.people", Entity: "office", EntityID: strconv.FormatInt(oid, 10), Detail: fmt.Sprintf("ultah=%d id-exp=%d wasiat=%d", pb, pe, pw)})
		}
		// Ringkasan pagi per staf.
		_, _ = remind.Digest(a.db, oid, deliver.FromEnv())
		// CMS: terbitkan halaman/artikel terjadwal yang jatuh tempo.
		if a.web != nil {
			a.web.PublishDue(oid)
		}
		// Retensi: ingatkan berkas yang mendekati kedaluwarsa (tanpa hapus otomatis).
		sla += a.notifyExpiring(oid)
	}
	return appt, overdue, tasks, sla
}

func (a *App) officeOf(r *http.Request) int64 {
	id := a.uid(r)
	if id == nil || a.db == nil {
		return 0
	}
	var office sql.NullInt64
	_ = a.db.QueryRow(`SELECT office_id FROM users WHERE id=?`, *id).Scan(&office)
	if !office.Valid {
		return 0
	}
	// Acting lintas-kantor: hanya super_admin + cookie bertanda HMAC + target ENTERPRISE.
	if mw.RoleOf(r) == rbac.RoleSuperAdmin {
		if c, err := r.Cookie(a.cfg.SessionName + "_act"); err == nil {
			if oid, ok := a.verifyActing(c.Value); ok {
				var pl string
				_ = a.db.QueryRow(`SELECT plan FROM offices WHERE id=?`, oid).Scan(&pl)
				if pl == "ENTERPRISE" {
					return oid
				}
			}
		}
	}
	return office.Int64
}

// actingCookieName is the signed cross-office scope cookie (super_admin only).
func (a *App) actingCookieName() string { return a.cfg.SessionName + "_act" }

func (a *App) signActing(officeID int64) string {
	key, err := cryptoxx.Key()
	if err != nil {
		return ""
	}
	mac := hmac.New(sha256.New, key)
	fmt.Fprintf(mac, "act:%d", officeID)
	return fmt.Sprintf("%d:%x", officeID, mac.Sum(nil))
}

func (a *App) verifyActing(token string) (int64, bool) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return 0, false
	}
	oid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || oid <= 0 {
		return 0, false
	}
	if a.signActing(oid) != token {
		return 0, false
	}
	return oid, true
}

// officeTier returns the office tier (fail-closed ke LITE bila tak terbaca).
func (a *App) officeTier(officeID int64) string {
	var p string
	if a.db != nil {
		_ = a.db.QueryRow(`SELECT plan FROM offices WHERE id=?`, officeID).Scan(&p)
	}
	if p == "" {
		return "LITE"
	}
	return p
}

// requirePlan blocks below-tier features with an honest upgrade notice.
func (a *App) requirePlan(w http.ResponseWriter, r *http.Request, feature string) bool {
	if plan.Allowed(a.officeTier(a.officeOf(r)), feature) {
		return true
	}
	need := map[string]string{plan.FeatPortalPay: "PRO", plan.FeatShares: "PRO", plan.FeatChecklist: "PRO", plan.FeatActing: "ENTERPRISE"}[feature]
	http.Error(w, "Fitur ini membutuhkan paket "+need+" atau lebih tinggi. Hubungi administrator.", http.StatusPaymentRequired)
	return false
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
	acting := false
	actName := ""
	if c, err := r.Cookie(a.actingCookieName()); err == nil && mw.RoleOf(r) == rbac.RoleSuperAdmin {
		if oid, ok := a.verifyActing(c.Value); ok {
			acting = true
			_ = a.db.QueryRow(`SELECT name FROM offices WHERE id=?`, oid).Scan(&actName)
		}
	}
	if acting && actName != "" {
		officeName = actName + " (bertindak)"
	}
	brand := a.brandView(a.officeOf(r))
	if err := a.tmpl.ExecuteTemplate(w, name, Page{Title: title, Role: role, UserName: username, OfficeName: officeName, Acting: acting, WIBDate: wib.Format(time.Now()), Pending: pending, Unread: unread, TopNotifs: topNotifs, Flash: flash, CSRF: mw.IssueCSRF(w, r, a.cfg.SessionName, a.cfg.TLSCookie), Data: data, Brand: brand}); err != nil {
		a.log.Error("render", "page", name, "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// brandView resolves white-label values for an office (defaults when unset).
// Logged-out pages (office 0) fall back to the install's first office so the
// standalone install's login screen carries its own brand.
func (a *App) brandView(officeID int64) BrandingView {
	bv := BrandingView{
		AppName: "SaktiNotaris", ShortName: "SN",
		Primary: "#1f6feb", Secondary: "#101f3c", Accent: "#c9a844",
		SidebarBG: "#101f3c", HeaderBG: "#ffffff",
		ButtonRadius: "8px",
		FontStack:    cms.FontStack("system"),
	}
	if a.db == nil || a.web == nil {
		return bv
	}
	if officeID == 0 {
		_ = a.db.QueryRow(`SELECT COALESCE(MIN(id),0) FROM offices`).Scan(&officeID)
		if officeID == 0 {
			return bv
		}
	}
	b, err := a.web.BrandingOf(officeID)
	if err != nil {
		return bv
	}
	bv.AppName, bv.ShortName, bv.Tagline = b.AppName, b.ShortName, b.Tagline
	bv.Primary, bv.Secondary, bv.Accent = b.Primary, b.Secondary, b.Accent
	bv.SidebarBG, bv.HeaderBG = b.SidebarBG, b.HeaderBG
	bv.ButtonRadius = cms.ButtonRadius(b.ButtonStyle, b.BorderRadius)
	bv.FontStack = cms.FontStack(b.FontFamily)
	if b.LogoRef != "" {
		bv.LogoURL = "/media/by-ref?ref=" + url.QueryEscape(b.LogoRef)
	}
	if b.FaviconRef != "" {
		bv.FaviconURL = "/media/by-ref?ref=" + url.QueryEscape(b.FaviconRef)
	}
	if b.LoginBG != "" {
		bv.LoginBG = "/media/by-ref?ref=" + url.QueryEscape(b.LoginBG)
	}
	bv.FooterText, bv.CopyrightText = b.FooterText, b.CopyrightText
	return bv
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
	r.Use(a.readOnly)
	limiter := mw.NewRateLimiter(10, time.Minute)
	loginLimiter := mw.NewRateLimiter(20, time.Minute) // lockout akun 5x/15mnt tetap jadi perlindungan utama
	verifyLimiter := mw.NewRateLimiter(30, time.Minute)
	// Rate-limit longgar agar pemakaian normal tidak terblokir (staf upload batch):
	// upload 30/menit per IP, OCR 20/menit per IP (OCR berat). POST /login sudah
	// dibatasi loginLimiter di atas, jadi di sini hanya tambah 2 endpoint sensitif.
	uploadLimiter := mw.NewRateLimiter(30, time.Minute)
	ocrLimiter := mw.NewRateLimiter(20, time.Minute)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	r.Get("/readyz", a.readyz)
	r.With(verifyLimiter.Limit).Get("/s/{token}", a.dmsSharedDownload)
	r.With(verifyLimiter.Limit).Get("/verify", a.verifyPage)
	r.With(verifyLimiter.Limit).Get("/qr/inv/{id}", a.invoiceQR)
	r.With(a.pubGate).Get("/", a.publicHome)
	r.With(a.pubGate).Get("/tentang-kami", a.publicPageSlug("tentang-kami"))
	r.With(a.pubGate).Get("/kontak", a.publicContact)
	r.With(a.pubGate).Get("/privasi", a.publicPrivacy)
	r.With(a.pubGate).Get("/konsultasi", a.pubKonsultasiForm)
	r.With(a.pubGate, limiter.Limit).Post("/konsultasi", a.pubKonsultasiSave)
	r.With(a.pubGate).Get("/booking", a.pubBookingForm)
	r.With(a.pubGate, limiter.Limit).Post("/booking", a.pubBookingSave)
	r.With(a.pubGate).Get("/kalkulator", a.pubKalkulator)
	r.With(a.pubGate).Get("/tim", a.pubTeam)
	r.With(a.pubGate).Get("/galeri", a.pubGallery)
	r.With(a.pubGate).Get("/testimoni", a.pubTestimonials)
	r.With(a.pubGate).Get("/cari", a.pubSearch)
	r.With(a.pubGate).Get("/arsip", a.pubArchive)
	r.With(a.pubGate).Get("/arsip/{month}", a.pubArchive)
	r.With(a.pubGate).Get("/p/preview", a.pubPreview)
	r.With(a.pubGate).Get("/p/{slug}", a.publicGeneric)
	r.With(a.pubGate).Get("/artikel", a.publicArticles)
	r.With(a.pubGate).Get("/artikel/{slug}", a.publicArticle)
	r.With(a.pubGate).Get("/faq", a.publicFAQ)
	r.With(a.pubGate).Get("/layanan-kami", a.publicServices)
	r.With(a.pubGate).Get("/layanan-kami/{slug}", a.publicService)
	r.With(a.pubGate).Get("/media/{id}", a.mediaServe)
	r.With(a.pubGate).Get("/media/{id}/{file}", a.mediaServe)
	r.With(a.pubGate).Get("/media/by-ref", a.mediaByRef)
	r.With(a.pubGate).Get("/robots.txt", a.robotsTxt)
	r.With(a.pubGate).Get("/sitemap.xml", a.sitemapXML)
	r.With(a.pubGate).Get("/llms.txt", a.llmsTxt)
	r.Get("/layanan", a.layananIndex)
	r.Get("/layanan/{code}", a.layananDetail)
	r.Get("/docs", a.docsIndex)
	r.Get("/docs/{step}", a.docsStep)
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))
	setupLimiter := mw.NewRateLimiter(10, time.Minute) // /setup: ketat, sekali pakai
	r.Get("/login", a.loginForm)
	r.With(loginLimiter.Limit).Post("/login", a.login)
	r.Get("/setup", a.setupForm)
	r.With(setupLimiter.Limit, mw.CSRF(a.sessions, a.cfg.SessionName)).Post("/setup", a.setupSubmit)
	r.Get("/mfa/verify", a.mfaVerifyForm)
	r.With(loginLimiter.Limit, mw.CSRF(a.sessions, a.cfg.SessionName)).Post("/mfa/verify", a.mfaVerify)

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
		p.Post("/clients/new", a.clientNewSubmit)
		p.Post("/clients/bulk", a.clientBulk)
		p.Post("/clients/merge/preview", a.mergePreview)
		p.Post("/clients/merge", a.mergeDo)
		p.Get("/clients/{id}", a.clientDetail)
		p.Get("/clients/{id}/edit", a.clientEditForm)
		p.Post("/clients/{id}", a.clientUpdate)
		p.Post("/clients/{id}/identities", a.identityAdd)
		p.Post("/clients/{id}/notes", a.noteAdd)
		p.Post("/identities/{id}/verify", a.identityVerify)
		p.Post("/identities/{id}/status", a.identityStatus)
		p.Post("/identities/{id}/flag", a.identityFlag)
		p.Post("/clients/{id}/reps", a.repAdd)
		p.With(uploadLimiter.Limit).Post("/clients/{id}/files", a.fileUpload)
		p.Get("/clients/{id}/specimens", a.specimens)
		p.Post("/clients/{id}/specimens", a.specimenUpload)
		p.Get("/files/{id}", a.fileDownload)
		p.With(ocrLimiter.Limit).Post("/files/{id}/ocr", a.fileOCR)
		p.Get("/requests", a.requestList)
		p.Post("/requests", a.requestCreate)
		p.Get("/requests/{id}", a.requestDetail)
		p.Post("/requests/{id}/move", a.requestMove)
		p.Post("/requests/{id}/corp", a.corpSave)
		p.Post("/requests/{id}/corp/legal", a.corpLegalSave)
		p.Post("/requests/{id}/shareholders", a.shareholderAdd)
		p.Post("/requests/{id}/shareholders/{sid}/link", a.shareholderLink)
		p.Post("/requests/{id}/officers", a.officerAdd)
		p.Post("/requests/{id}/bo", a.boAdd)
		p.Get("/requests/{id}/wizard", a.corpWizard)
		p.Get("/requests/{id}/wizard-ajb", a.ajbWizard)
		p.Get("/requests/{id}/wizard-wasiat", a.willWizard)
		p.Get("/appointments", a.appointments)
		p.Post("/appointments", a.appointmentCreate)
		p.Post("/appointments/{id}/status", a.appointmentStatus)
		p.Get("/deeds", a.deedList)
		p.Post("/deeds", a.deedCreate)
		p.Get("/deeds/{id}", a.deedDetail)
		p.Post("/deeds/{id}/version", a.deedSaveVersion)
		p.Post("/deeds/{id}/approve", a.deedApprove)
		p.Post("/deeds/{id}/review", a.deedToReview)
		p.Post("/deeds/{id}/sign", a.deedSign)
		p.Get("/deeds/{id}/pdf", a.deedPDF)
		p.Get("/deeds/{id}/docx", a.deedDOCX)
		p.Post("/deeds/{id}/attach", a.deedAttach)
		p.Get("/deeds/{id}/package", a.deedPackage)
		p.Get("/wills", a.willsPage)
		p.Post("/wills", a.willCreate)
		p.Post("/wills/{id}/handover", a.willHandover)
		p.Post("/wills/{id}/review", a.willReview)
		p.Post("/wills/{id}/open", a.willOpen)
		p.Get("/repertorium", a.repertorium)
		p.Get("/repertorium/export", a.repertoriumExport)
		p.Get("/protocols", a.protocols)
		p.Post("/protocols", a.protocolCreate)
		p.Post("/protocols/{id}/status", a.protocolStatus)
		p.Get("/legalisasi", a.legalisasi)
		p.Post("/legalisasi", a.legalisasiCreate)
		p.Post("/legalisasi/{id}/advance", a.legalisasiAdvance)
		p.Get("/ppat", a.ppatRegister)
		p.Post("/ppat", a.ppatCreate)
		p.Post("/ppat/{id}/advance", a.ppatAdvance)
		p.Get("/ppat/{id}/blanko", a.ppatBlanko)
		p.Get("/gov", a.gov)
		p.Post("/gov", a.govCreate)
		p.Post("/gov/{id}/advance", a.govAdvance)
		p.Post("/gov/{id}/check", a.govCheck)
		p.Post("/gov/{id}/followup", a.govFollowup)
		p.Get("/finance", a.financePage)
		p.Post("/invoices", a.invoiceCreate)
		p.Post("/invoices/{id}/void", a.invoiceVoid)
		p.Post("/payments", a.paymentCreate)
		p.Get("/finance/bank", a.bankPage)
		p.Post("/finance/bank/upload", a.bankUpload)
		p.Post("/finance/bank/match", a.bankMatch)
		p.Post("/finance/bank/manual", a.bankManual)
		p.Post("/budgets", a.budgetSave)
		p.Get("/templates", a.templatesPage)
		p.Post("/templates", a.templateSave)
		p.Post("/templates/seed", a.templateSeed)
		p.Get("/templates/{id}", a.templateEdit)
		p.Post("/templates/{id}/approve", a.templateApprove)
		p.Post("/clauses", a.clauseSave)
		p.Get("/reports", a.reports)
		p.Get("/settings", a.settings)
		// Website CMS + White Label (standalone per kantor).
		p.Get("/cms", a.cmsDashboard)
		p.Get("/cms/branding", a.brandingPage)
		p.Post("/cms/branding", a.brandingSave)
		p.Get("/cms/site", a.sitePage)
		p.Post("/cms/site", a.siteSave)
		p.Post("/cms/domain", a.domainSave)
		p.Get("/cms/media", a.mediaPage)
		p.With(mw.NewRateLimiter(20, time.Minute).Limit).Post("/cms/media", a.mediaUpload)
		p.Post("/cms/media/{id}/delete", a.mediaDelete)
		p.Get("/cms/pages", a.cmsPages)
		p.Get("/cms/pages/new", a.cmsPageNew)
		p.Post("/cms/pages", a.cmsPageCreate)
		p.Get("/cms/pages/{id}", a.cmsPageEdit)
		p.Post("/cms/pages/{id}", a.cmsPageUpdate)
		p.Post("/cms/pages/{id}/duplicate", a.cmsPageDuplicate)
		p.Post("/cms/pages/{id}/delete", a.cmsPageDelete)
		p.Post("/cms/pages/{id}/sections", a.cmsSectionSave)
		p.Post("/cms/sections/{sid}/delete", a.cmsSectionDelete)
		p.Get("/cms/articles", a.cmsArticles)
		p.Get("/cms/articles/new", a.cmsArticleNew)
		p.Get("/cms/articles/{id}", a.cmsArticleEdit)
		p.Post("/cms/articles", a.cmsArticleSave)
		p.Post("/cms/articles/{id}", a.cmsArticleUpdate)
		p.Post("/cms/categories", a.cmsCategorySave)
		p.Get("/cms/services", a.cmsServices)
		p.Post("/cms/services", a.cmsServiceSave)
		p.Get("/cms/faq", a.cmsFAQ)
		p.Post("/cms/faq", a.cmsFAQSave)
		p.Post("/cms/faq/{id}/delete", a.cmsFAQDelete)
		p.Get("/cms/nav", a.cmsNav)
		p.Post("/cms/nav", a.cmsNavSave)
		p.Post("/cms/nav/{id}/delete", a.cmsNavDelete)
		p.Get("/cms/history", a.cmsHistory)
		p.Get("/cms/setup", a.setupPage)
		p.Post("/cms/setup", a.setupSave)
		p.Post("/cms/setup/sample", a.setupSample)
		// CMS growth: interaksi publik + engagement + versi + kualitas + statistik.
		p.Get("/cms/inquiries", a.cmsInquiries)
		p.Post("/cms/inquiries/{id}/status", a.cmsInquiryStatus)
		p.Get("/cms/slots", a.cmsSlots)
		p.Post("/cms/slots", a.cmsSlotSave)
		p.Post("/cms/slots/{id}/delete", a.cmsSlotDelete)
		p.Get("/cms/bookings", a.cmsBookings)
		p.Post("/cms/bookings/{id}/decide", a.cmsBookingDecide)
		p.Get("/cms/team", a.cmsTeam)
		p.Post("/cms/team", a.cmsTeamSave)
		p.Post("/cms/team/{id}/delete", a.cmsTeamDelete)
		p.Get("/cms/gallery", a.cmsGallery)
		p.Post("/cms/gallery", a.cmsGallerySave)
		p.Post("/cms/gallery/{id}/delete", a.cmsGalleryDelete)
		p.Get("/cms/testimonials", a.cmsTestimonials)
		p.Post("/cms/testimonials", a.cmsTestimonialSave)
		p.Post("/cms/testimonials/{id}/status", a.cmsTestimonialStatus)
		p.Get("/cms/versions", a.cmsVersions)
		p.Post("/cms/versions/{id}/rollback", a.cmsVersionRollback)
		p.Get("/cms/quality", a.cmsQuality)
		p.Get("/cms/stats", a.cmsStats)
		p.Get("/cms/export", a.cmsExport)
		p.Post("/users", a.userCreate)
		p.Post("/users/{id}/link", a.userLink)
		p.Get("/roles", a.rolesPage)
		p.Get("/offices", a.officesPage)
		p.Post("/offices/act", a.officeAct)
		p.Post("/offices/{id}/plan", a.officePlanSave)
		p.Get("/mfa", a.mfaSetup)
		p.Post("/mfa", a.mfaEnable)
		p.Get("/portal", a.portal)
		p.Post("/notifications/read", a.notifRead)
		p.Get("/metrics", a.metrics)
		p.Get("/portal/invoices", a.portalInvoices)
		p.Post("/portal/pay", a.portalPayConfirm)
		p.Get("/invoices/{id}/receipt", a.receipt)
		p.Post("/reminders/run", a.remindersRun)
		p.Get("/documents", a.documents)
		p.Get("/dms", a.dmsDashboard)
		p.Get("/dms/library", a.dmsLibrary)
		p.Get("/dms/library/export", a.dmsLibraryExport)
		p.Get("/documents/{id}", a.dmsDetail)
		p.Post("/documents/{id}/meta", a.dmsSaveMeta)
		p.Post("/documents/{id}/verify", a.dmsVerify)
		p.Get("/documents/{id}/preview", a.dmsPreview)
		p.Get("/documents/{id}/previewwm", a.dmsPreviewWM)
		p.Post("/documents/{id}/version", a.dmsNewVersion)
		p.Get("/documents/{id}/version/{v}", a.dmsDownloadVersion)
		p.Get("/documents/{id}/compare", a.dmsCompare)
		p.Post("/documents/{id}/restore/{v}", a.dmsRestoreVersion)
		p.Post("/documents/{id}/workflow", a.dmsMove)
		p.Post("/documents/{id}/approve", a.dmsApproveStep)
		p.Post("/documents/{id}/scan", a.dmsScan)
		p.Post("/documents/{id}/link", a.dmsLink)
		p.Post("/documents/{id}/favorite", a.dmsFavorite)
		p.Post("/documents/{id}/share", a.dmsShare)
		p.Post("/documents/{id}/archive", a.dmsArchiveMove)
		p.Post("/documents/{id}/hold", a.dmsHold)
		p.Post("/documents/{id}/disposition", a.dmsDisposition)
		p.Post("/documents/{id}/fields", a.dmsFieldsSave)
		p.Get("/dms/ocr", a.dmsOCRQueue)
		p.Post("/dms/ocr/run", a.dmsOCRRun)
		p.Get("/dms/search", a.dmsSearch)
		p.Get("/dms/archive", a.dmsArchive)
		p.Post("/dms/archive/locations", a.dmsArchiveLoc)
		p.Get("/dms/retention", a.dmsRetention)
		p.Post("/dms/retention/task", a.retentionTask)
		p.Post("/dms/holds/{hid}/release", a.dmsReleaseHold)
		p.Get("/dms/duplicates", a.dmsDuplicates)
		p.Post("/dms/duplicates/merge", a.dmsMerge)
		p.Get("/dms/search/adv", a.dmsSearchAdvanced)
		p.Post("/dms/bulk", a.dmsBulk)
		p.Get("/dms/fields", a.dmsFields)
		p.Post("/dms/fields", a.dmsFieldSave)
		p.Get("/dms/chain", a.dmsChain)
		p.Post("/dms/chain", a.dmsChainSave)
		p.Get("/dms/categories", a.dmsCategories)
		p.Get("/dms/categories", a.dmsCategories)
		p.Post("/dms/categories", a.dmsCategoryCreate)
		p.Get("/dms/checklists", a.dmsChecklists)
		p.Post("/dms/checklists", a.dmsChecklistCreate)
		p.Post("/dms/checklists/seed", a.dmsChecklistSeed)
		p.Post("/requests/{id}/checklist", a.dmsChecklistSet)
		p.Get("/qr", a.qrText)
		p.Get("/dms/archive/labels", a.dmsLabels)
		p.Get("/audit", a.auditPage)
		p.Get("/audit/export", a.auditExport)
		p.Get("/office", a.officePage)
		p.Post("/office", a.officeSave)
		p.Get("/office/export", a.officeExport)
		p.Get("/office/license", a.licensePage)
		p.Post("/office/license", a.licenseSave)
		p.Get("/office/mandates", a.mandatesPage)
		p.Post("/office/mandates", a.mandateSave)
		p.Post("/office/mandates/{id}/delete", a.mandateDelete)
		p.Post("/office/backup", a.backupNow)
		p.Get("/portal/invoices/{id}/receipt", a.portalReceipt)
		p.Post("/cms/inquiries/{id}/convert", a.inquiryToRequest)
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
	a.render(w, r, "login", "Masuk — SaktiNotaris", nil)
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "form tidak valid", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	ip := clientIPOf(r)
	// Penguncian akun: 5 gagal dalam 15 menit (per email+IP) → tolak 15 menit.
	// Kunci per pasangan email+IP agar penyerang tak bisa mengunci korban dari IP lain.
	var fails int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM login_attempts WHERE email=? AND ip=? AND success=0 AND created_at > DATE_SUB(NOW(), INTERVAL 15 MINUTE)`, email, ip).Scan(&fails)
	if fails >= 5 {
		a.audit.Record(audit.Entry{Action: "login.locked", Entity: "user", EntityID: email, Detail: ip})
		a.render(w, r, "login", "Masuk — SaktiNotaris", map[string]string{"Error": "Akun terkunci sementara setelah 5x gagal (15 menit). Coba lagi nanti.", "Email": email})
		return
	}
	u, err := a.users.Authenticate(email, r.FormValue("password"))
	if err != nil {
		_, _ = a.db.Exec(`INSERT INTO login_attempts(email, ip, success) VALUES(?,?,0)`, email, ip)
		// Audit tanpa password/hash: hanya email + IP untuk deteksi brute-force.
		a.audit.Record(audit.Entry{Action: "login.failed", Entity: "user", EntityID: email, Detail: ip})
		a.render(w, r, "login", "Masuk — SaktiNotaris", map[string]string{"Error": err.Error(), "Email": email})
		return
	}
	_, _ = a.db.Exec(`INSERT INTO login_attempts(email, ip, success) VALUES(?,?,1)`, email, ip)
	_, _ = a.db.Exec(`DELETE FROM login_attempts WHERE email=? AND success=0`, email)
	// Step-up: accounts with TOTP enrolled must pass a second factor.
	if a.users.MFAEnabled(u.ID) {
		tok := a.signPendingMFA(u.ID)
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
	// Catat IP/UA sesi; beri tahu bila login dari IP yang belum pernah dipakai akun ini.
	lip := clientIPOf(r)
	_, _ = a.db.Exec(`UPDATE sessions SET ip=?, user_agent=? WHERE token=?`, lip, r.UserAgent(), sess.Token)
	var seen int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id=? AND ip=? AND token<>?`, u.ID, lip, sess.Token).Scan(&seen)
	var ever int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id=?`, u.ID).Scan(&ever)
	if seen == 0 && ever > 1 {
		_, _ = a.db.Exec(`INSERT INTO notifications(user_id, title, body) VALUES(?,?,?)`, u.ID, "Login dari IP baru", "Akun Anda login dari "+lip+". Abaikan bila itu Anda.")
	}
	http.SetCookie(w, &http.Cookie{
		Name: a.cfg.SessionName, Value: sess.Token, Path: "/",
		Expires: sess.ExpiresAt, MaxAge: 43200, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.cfg.TLSCookie,
	})
	uid := u.ID
	a.audit.Record(audit.Entry{ActorID: &uid, Action: "login", Entity: "user", EntityID: strconv.FormatInt(u.ID, 10)})
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.cfg.SessionName); err == nil {
		uid := a.uid(r)
		a.sessions.Revoke(c.Value)
		a.audit.Record(audit.Entry{ActorID: uid, Action: "logout", Entity: "user", EntityID: ""})
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionName, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.cfg.TLSCookie})
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
	fmt.Fprintln(w, "Allow: /docs")
	fmt.Fprintln(w, "Allow: /verify")
	fmt.Fprintln(w, "Allow: /assets/")
	fmt.Fprintln(w, "Disallow: /app")
	fmt.Fprintln(w, "Disallow: /clients")
	fmt.Fprintln(w, "Disallow: /requests")
	fmt.Fprintln(w, "Disallow: /deeds")
	fmt.Fprintln(w, "Disallow: /finance")
	fmt.Fprintln(w, "Disallow: /portal")
	fmt.Fprintln(w, "Disallow: /admin")
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
	for _, u := range a.cmsSitemapURLs(base) {
		urls = append(urls, url{u, "weekly", "0.8"})
	}
	urls = append(urls, url{base + "/docs", "monthly", "0.9"})
	for _, st := range docSteps() {
		urls = append(urls, url{base + "/docs/" + st.Slug, "monthly", "0.7"})
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`))
	for _, u := range urls {
		fmt.Fprintf(w, "\n<url><loc>%s</loc><changefreq>%s</changefreq><priority>%s</priority></url>", u.Loc, u.Changefreq, u.Priority)
	}
	w.Write([]byte("\n</urlset>"))
}

// llmsTxt exposes a crawler- and LLM-friendly summary (GEO: factual, no hype).
// The service list renders live from the services catalog (incl. PPAT + bundel)
// so the count can never drift from the sitemap/PSEO pages.
func (a *App) llmsTxt(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Write([]byte(`# SaktiNotaris

> Sistem manajemen kantor Notaris Indonesia (Go + MySQL + Tabler).
> Catatan: informasi produk di bawah ini; bukan nasihat hukum.

## Produk
- Klien perorangan & badan hukum, verifikasi identitas manusia, beneficial owner.
- Permohonan layanan 9 status; akta draf berversi dari template terapproved;
  pengesahan notaris mengunci final + repertorium otomatis.
- Register: repertorium, legalisasi/waarmerking/salinan/kutipan, protokol 25 tahun, wasiat terbatas.
- PPAT: register akta tanah terpisah (AJB, hibah, tukar-menukar, APHB, inbreng, SKMHT, APHT) + lapor BPN.
- Paket bundel: pendirian PT/CV + NIB/NPWP, AJB + balik nama (satu permohonan, syarat gabungan).
- Keuangan: invoice multi-baris anti-overpay, pembayaran, piutang, pengeluaran, tarif.
- Keamanan: bcrypt-12, sesi server-side, MFA TOTP, RBAC 9 peran/19 izin, NIK AES-256-GCM,
  audit imutabel, HSTS+CSP+CSRF+rate-limit.
- Notaris & PPAT (bukan advokat). Tanpa API resmi AHU/OSS/BPN (asistensi manual + bukti).
- Harga: konsultasi via penawaran resmi (tidak ada harga publik).

## Layanan (katalog aktif — tiap baris = satu halaman /layanan/{kode lower-case}, sama dengan sitemap.xml)
`))
	if a.db != nil {
		rows, _ := a.db.Query(`SELECT code, category, name FROM services WHERE is_active=1 ORDER BY category, code`)
		if rows != nil {
			defer rows.Close()
			lastCat := ""
			for rows.Next() {
				var code, cat, name string
				_ = rows.Scan(&code, &cat, &name)
				if cat != lastCat {
					lastCat = cat
					fmt.Fprintf(w, "\n### %s\n", cat)
				}
				fmt.Fprintf(w, "- %s: %s (/layanan/%s)\n", code, name, strings.ToLower(code))
			}
		}
	}
	w.Write([]byte(`
## Halaman
- / (landing) · /layanan (indeks) · /layanan/{kode} (detail per layanan, syarat + alur)
- /docs (tutorial 8 langkah) · /docs/{langkah} (ktp-masuk, ocr-proses, review-verifikasi,
  permohonan-layanan, akta-template, dokumen-sah, keuangan-invoice, portal-pantau)
- /verify?code=... (verifikasi publik kuitansi/akta via kode HMAC)
- Aplikasi butuh akun: /login (demo: admin@contoh.id — sandi demo di halaman login).
`))
}

// layananIndex lists all active services (PSEO hub page, public for everyone).
func (a *App) layananIndex(w http.ResponseWriter, r *http.Request) {
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
				var price string
				_ = rows.Scan(&s.Code, &s.Category, &s.Name, &price)
				fmt.Sscanf(price, "%f", &s.Price)
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

// layananDetail renders one programmatic SEO page per service code (public).
func (a *App) layananDetail(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	var name, category string
	var price float64
	var sprice, reqRaw sql.NullString
	err := a.db.QueryRow(`SELECT name, category, base_price, requirements FROM services WHERE code=? AND is_active=1`, code).Scan(&name, &category, &sprice, &reqRaw)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if sprice.Valid {
		fmt.Sscanf(sprice.String, "%f", &price)
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
	base := "https://" + r.Host
	if r.Host == "" || strings.HasPrefix(r.Host, "127.") || strings.HasPrefix(r.Host, "localhost") {
		base = "https://notarius.example"
	}
	if err := a.tmpl.ExecuteTemplate(w, "layanan_detail", map[string]any{
		"Code": code, "Name": name, "Category": category, "Price": price, "Reqs": reqs,
		"Canon": base + "/layanan/" + strings.ToLower(code),
	}); err != nil {
		a.log.Error("render", "page", "layanan_detail", "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// ---- tutorial publik: KTP masuk -> dokumen legalitas sah ----

type docStep struct {
	Slug, Title, Img, Alt, Intro string
	Points                       []string
	Honest                       string
	CTA, CTAURL                  string
}

func docSteps() []docStep {
	return []docStep{
		{"ktp-masuk", "1. Klien Baru & Cek Duplikat", "notarius-doc-ktp.jpg", "Tambah klien perorangan dan badan hukum di SaktiNotaris",
			"Tambah klien perorangan / PT / CV / Yayasan di /clients — sistem otomatis cegah data ganda.",
			[]string{"Buka /clients → + Tambah Klien → isi jenis, nama, NIK/NPWP, telepon, email, alamat", "Peringatan duplikat otomatis: telepon sama / nama mirip (SOUNDS LIKE) → gabungkan via Merge", "Untuk PT/CV: lanjut isi Pemegang Saham, Pengurus & Beneficial Owner (BO, TPPU/PEP) di detail permohonan"},
			"NIK/NPWP wajib valid. Data ganda merusak repertorium — selalu cek peringatan duplikat dulu.",
			"Tambah klien sekarang", "/clients"},
		{"ocr-proses", "2. Upload Dokumen & OCR", "notarius-doc-ocr.jpg", "Upload berkas dan proses OCR di Document Center",
			"Unggah KTP, KK, NPWP & pendukung ke Document Center — OCR lokal Tesseract membantu membaca.",
			[]string{"Buka detail klien → tab Dokumen → Unggah (JPG/PNG/WebP/PDF, maks 100MB, privat per kantor)", "Klik Jalankan OCR di /documents → status PENDING → DONE/FAILED + hash SHA-256", "Hasil + provenance (engine, versi, waktu) tersimpan di riwayat AI"},
			"OCR hanya membantu baca. Stempel / tulisan tangan sering salah baca — wajib verifikasi manusia.",
			"Buka Document Center", "/documents"},
		{"review-verifikasi", "3. Verifikasi Manusia", "notarius-doc-review.jpg", "Staf memverifikasi identitas sebelum dipakai ke akta",
			"Bandingkan hasil OCR dengan fisik dokumen, lalu sahkan — tercatat siapa & kapan.",
			[]string{"Buka berkas → cocokkan NIK/nama dengan dokumen asli", "Klik Verifikasi — tersimpan verified_by / verified_at + audit log", "Hanya identitas Sudah Diverifikasi yang bisa ditarik ke draf akta"},
			"OCR tidak pernah auto-verify. Tanpa verifikasi manusia, data tidak boleh masuk akta.",
			"Verifikasi identitas", "/clients"},
		{"permohonan-layanan", "4. Permohonan & Workflow Layanan", "notarius-doc-permohonan.svg", "Buat permohonan layanan dan gerakkan status workflow",
			"Pilih klien + katalog 32 layanan → buat permohonan → gerakkan 9 status hingga selesai.",
			[]string{"Buka /requests → pilih Klien + Layanan + judul → Buat", "Gerakkan status: DRAFT → SUBMITTED → IN_REVIEW → APPROVED → IN_PROGRESS → COMPLETED (transisi legal divalidasi)", "Untuk korporat: isi Data Korporat, Legal AHU/OSS checklist + bukti di tab yang sama; atur Janji/Reminder bila perlu"},
			"Status tidak bisa loncat. Bukti AHU/OSS hanya checklist asistensi — bukan klaim pengajuan otomatis.",
			"Buat permohonan", "/requests"},
		{"akta-template", "5. Draf Akta dari Template", "notarius-doc-akta.svg", "Buat draf akta berversi dari template yang disetujui notaris",
			"Semua akta hanya dari template Approved — revisi berversi, review, lalu kunci final.",
			[]string{"Buka /deeds → Buat draf → pilih Template Approved (variabel {{nama}} terisi dari identitas terverifikasi)", "Simpan revisi → versi tercatat + hash; klik Preview / missing-vars sebelum kirim", "Kirim Review → notaris Approve + nomor akta → terkunci; ekspor PDF/DOCX (watermark DRAFT bila belum sah)"},
			"Template Draft tidak bisa dipakai. Pengesahan hanya oleh peran notaris — staf hanya sampai Review.",
			"Buat draf akta", "/deeds"},
		{"dokumen-sah", "6. Register Resmi: Repertorium, Protokol, Legalisasi", "notarius-doc-register.svg", "Repertorium, protokol dan legalisasi tercatat otomatis dan manual",
			"Akta yang disahkan otomatis masuk repertorium — protokol & legalisasi dicatat di registernya.",
			[]string{"Repertorium /repertorium: nomor unik per kantor, otomatis saat pengesahan + cek nomor lompat (gaps) + Ekspor CSV", "Protokol /protocols: bendel minuta/salinan/grosse per tahun + lokasi arsip, retensi 25 tahun", "Legalisasi /legalisasi: LEGALISASI/WAARMERKING/SALINAN/KUTIPAN → REGISTERED → COMPLETED → HANDED_OVER + biaya; Wasiat /wills & serah-terima hanya notaris"},
			"Keabsahan berasal dari pengesahan notaris sesuai UUJN — bukan dari aplikasi. Nomor register tidak boleh manual ganda.",
			"Lihat repertorium", "/repertorium"},
		{"keuangan-invoice", "7. Invoice, Bayar & Rekonsiliasi Bank", "notarius-doc-keuangan.svg", "Buat invoice multi-baris, catat pembayaran dan rekonsiliasi bank",
			"Invoice rupiah presisi (int64) + PPN 11% otomatis — kelebihan bayar ditolak sistem.",
			[]string{"Buka /finance → Invoice Baru (multi-baris) → PPN otomatis → kirim ke klien", "Catat pembayaran per invoice — anti-overpay; void tercatat audit", "Impor mutasi di /finance/bank → auto-match → cocok manual; pantau Piutang & Penerimaan bulan berjalan"},
			"Jangan hapus invoice lunas — gunakan Void agar jejak audit utuh.",
			"Buka keuangan", "/finance"},
		{"portal-pantau", "8. Portal Klien & Dashboard Laporan", "notarius-doc-legal.jpg", "Klien pantau mandiri, pimpinan pantau operasional real-time",
			"Klien hanya lihat datanya sendiri — pimpinan lihat angka real-query tanpa fabrikasi.",
			[]string{"Klien login → /portal: status permohonan, draf akta terkait & invoice + konfirmasi bayar + unduh kwitansi", "Pimpinan → /app dashboard: pending/aktif/selesai, piutang, agenda hari ini + grafik 6/12 bulan", "Laporan /reports + /metrics: semua angka dari query langsung — bukan contoh statis"},
			"Isolasi portal ditegakkan server-side. Akun klien yang belum ditautkan tidak melihat data apa pun.",
			"Pantau dashboard", "/app"},
	}
}

// docsIndex renders the tutorial hub (public, incl. logged-in users).
func (a *App) docsIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "docs_index", map[string]any{"Steps": docSteps()}); err != nil {
		a.log.Error("render", "page", "docs_index", "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// docsStep renders one tutorial step (public, incl. logged-in users).
func (a *App) docsStep(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "step")
	steps := docSteps()
	idx := -1
	for i, s := range steps {
		if s.Slug == slug {
			idx = i
			break
		}
	}
	if idx < 0 {
		http.NotFound(w, r)
		return
	}
	var prev, next *docStep
	if idx > 0 {
		prev = &steps[idx-1]
	}
	if idx < len(steps)-1 {
		next = &steps[idx+1]
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "docs_step", map[string]any{"Step": steps[idx], "Prev": prev, "Next": next, "Idx": idx + 1, "Total": len(steps)}); err != nil {
		a.log.Error("render", "page", "docs_step", "err", err)
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
		canWill := willReader(mw.RoleOf(r))
		rows, _ := a.db.Query(`SELECT a.action, COALESCE(u.name,'sistem'), a.created_at, a.entity, a.entity_id
			FROM audit_logs a LEFT JOIN users u ON u.id=a.actor_id
			WHERE (u.office_id=? OR a.actor_id IS NULL)
			ORDER BY a.id DESC LIMIT 10`, office)
		if rows != nil {
			defer rows.Close()
			link := map[string]string{"client": "/clients/", "service_request": "/requests/", "deed": "/deeds/", "invoice": "/finance", "will": "/wills", "gov_submission": "/gov"}
			for rows.Next() {
				var x activity
				_ = rows.Scan(&x.Action, &x.Actor, &x.At, &x.Entity, &x.EntityID)
				if !canWill && (x.Entity == "will" || strings.HasPrefix(x.Action, "will.")) {
					continue
				}
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
	// Kata sandi kedaluwarsa (>180 hari): banner pengingat.
	pwStale := false
	if id := a.uid(r); id != nil {
		var days int
		_ = a.db.QueryRow(`SELECT COALESCE(DATEDIFF(NOW(), password_changed_at), 9999) FROM users WHERE id=?`, *id).Scan(&days)
		pwStale = days > 180
	}
	// Penegakan ringan MFA untuk peran sensitif wasiat (peringatan saja,
	// tanpa blokir login agar tak mengunci kantor): willReader yang belum
	// enroll TOTP (users.mfa_secret kosong) diminta aktifkan via /mfa.
	mfaWarn := false
	if id := a.uid(r); id != nil && a.db != nil {
		if willReader(mw.RoleOf(r)) && !a.users.MFAEnabled(*id) {
			mfaWarn = true
		}
	}
	// Kartu notaris: menunggu persetujuan, siap TTE, macet SLA.
	var toApprove, readySign, stuck int
	if a.db != nil {
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE office_id=? AND status IN ('REVIEW','REVISION')`, office).Scan(&toApprove)
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE office_id=? AND status='APPROVED'`, office).Scan(&readySign)
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM service_requests r LEFT JOIN services s ON s.id=r.service_id WHERE r.office_id=? AND r.status IN ('SUBMITTED','IN_REVIEW','NEED_DOCS') AND r.updated_at < DATE_SUB(UTC_TIMESTAMP(), INTERVAL COALESCE(s.sla_days,3) DAY)`, office).Scan(&stuck)
	}
	// Widget dashboard "SLA + antrean": 5 permohonan paling mendesak lewat SLA
	// (remind.ServiceOverdue, tanpa migrasi) + dokumen DMS overdue via
	// remind.OverdueDocs(db, office) yang signature-nya cocok
	// (func(db *sql.DB, officeID int64) ([]OverdueDoc, error)), jadi dipakai
	// langsung. Best-effort: gagal query → widget disembunyikan template.
	var slaTop []remind.ServiceOverdueItem
	var docOver []remind.OverdueDoc
	if a.db != nil {
		if items, err := remind.ServiceOverdue(a.db, office); err == nil {
			if len(items) > 5 {
				items = items[:5]
			}
			slaTop = items
		}
		if docs, err := remind.OverdueDocs(a.db, office); err == nil {
			if len(docs) > 5 {
				docs = docs[:5]
			}
			docOver = docs
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
		"PwStale": pwStale, "ToApprove": toApprove, "ReadySign": readySign, "Stuck": stuck,
		"SLAOverdue": slaTop, "DocOverdue": docOver, "MFAWarn": mfaWarn,
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
	q := r.URL.Query()
	page := config.MustInt(q.Get("page"), 1)
	if page < 1 {
		page = 1
	}
	perPage := config.MustInt(q.Get("per_page"), 20)
	if perPage != 10 && perPage != 20 && perPage != 50 && perPage != 100 {
		perPage = 20
	}
	f := clients.ListFilter{
		Q: q.Get("q"), Kind: q.Get("kind"), Status: q.Get("status"), Service: q.Get("service"),
		Province: q.Get("province"), City: q.Get("city"),
		From: q.Get("from"), To: q.Get("to"), Docs: q.Get("docs"), Sort: q.Get("sort"),
		Limit: perPage, Offset: (page - 1) * perPage,
	}
	list, total, _ := a.clients.ListAdvanced(a.officeOf(r), f)
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
	var qs strings.Builder
	for _, k := range []string{"q", "kind", "status", "service", "province", "city", "from", "to", "docs", "sort"} {
		qs.WriteString(k + "=" + q.Get(k) + "&")
	}
	qs.WriteString("per_page=" + strconv.Itoa(perPage))
	stats, _ := a.clients.Stats(a.officeOf(r))
	svcs, _ := a.reqs.Services()
	var provinces []string
	if rows, err := a.db.Query(`SELECT DISTINCT province FROM clients WHERE office_id=? AND province<>'' ORDER BY province`, a.officeOf(r)); err == nil {
		defer rows.Close()
		for rows.Next() {
			var p string
			_ = rows.Scan(&p)
			provinces = append(provinces, p)
		}
	}
	if r.Header.Get("HX-Request") == "true" {
		_ = a.tmpl.ExecuteTemplate(w, "client_rows", list)
		return
	}
	a.render(w, r, "clients", "Daftar Klien", map[string]any{
		"List": list, "Q": f.Q, "Kind": f.Kind, "Status": f.Status, "Service": f.Service,
		"Province": f.Province, "City": f.City, "From": f.From, "To": f.To, "Docs": f.Docs,
		"Sort": f.Sort, "PerPage": perPage, "Page": page, "Pages": pages, "Total": total,
		"Prev": prev, "Next": next, "QS": qs.String(), "Stats": stats, "Services": svcs,
		"Provinces": provinces,
	})
}

// clientExport downloads the filtered client list as CSV (permission-gated).
// NIK is NEVER exported in full — only masked last-4 (UU PDP).
func (a *App) clientExport(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	q := r.URL.Query()
	f := clients.ListFilter{Q: q.Get("q"), Kind: q.Get("kind"), Status: q.Get("status"),
		Service: q.Get("service"), Province: q.Get("province"), City: q.Get("city"),
		From: q.Get("from"), To: q.Get("to"), Docs: q.Get("docs"), Sort: q.Get("sort"),
		Limit: 5000}
	list, _, _ := a.clients.ListAdvanced(a.officeOf(r), f)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"klien.csv\"")
	w.Write([]byte("\xef\xbb\xbf"))
	fmt.Fprintln(w, "ID,Registrasi,Nama,Jenis,Status,Telepon,Email,NPWP,NIK(4 digit),Domisili,Dibuat")
	for _, c := range list {
		full, _ := a.clients.GetFull(a.officeOf(r), c.ID)
		last4 := ""
		if ids, _ := a.clients.Identities(c.ID); len(ids) > 0 {
			last4 = ids[0].NIKLast4
		}
		masked := "••••"
		if last4 != "" {
			masked = "••••" + last4
		}
		fmt.Fprintf(w, "%d,\"%s\",\"%s\",%s,%s,%s,%s,%s,%s,\"%s\",%s\n", c.ID,
			strings.ReplaceAll(full.RegNumber, `"`, `""`),
			strings.ReplaceAll(c.FullName, `"`, `""`), c.Kind, full.Status, c.Phone, c.Email,
			c.NPWP, masked, strings.ReplaceAll(full.Domicile, `"`, `""`), c.CreatedAt)
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "client.export", Entity: "client", EntityID: "csv"})
}

func (a *App) clientNew(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermCreateClient) {
		a.forbidden(w, r)
		return
	}
	step := r.URL.Query().Get("step")
	if step == "" {
		step = "1"
	}
	a.render(w, r, "client_wizard", "Klien Baru", map[string]any{"Step": step, "F": map[string]string{}})
}

// clientNewSubmit advances the 3-step wizard, carrying values as hidden fields.
// Nothing is saved until the final POST to /clients (atomic, validated twice).
func (a *App) clientNewSubmit(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermCreateClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	f := map[string]string{}
	for _, k := range []string{"kind", "full_name", "phone", "email", "nik", "reg_number",
		"birth_place", "birth_date", "gender", "nationality", "marital_status", "occupation",
		"education", "passport_no", "npwp", "id_address", "domicile_address", "province", "city",
		"district", "village", "postal_code", "alt_phone", "wa_number", "correspondence_address",
		"first_client_date", "source"} {
		f[k] = strings.TrimSpace(r.FormValue(k))
	}
	step := r.FormValue("step")
	if step == "" {
		step = "1"
	}
	next := "2"
	if step == "2" {
		next = "3"
	}
	var dups []clients.DuplicateCandidate
	if step == "1" {
		if f["full_name"] == "" || f["phone"] == "" {
			a.render(w, r, "client_wizard", "Klien Baru", map[string]any{"Step": "1", "F": f, "Error": "nama dan telepon wajib diisi"})
			return
		}
		// Non-blocking duplicate warning (review before final save).
		dups, _ = a.clients.FindDuplicatesEx(a.officeOf(r), 0, f["full_name"], f["phone"], f["email"], f["npwp"], last4of(f["nik"]), f["birth_date"])
	}
	a.render(w, r, "client_wizard", "Klien Baru", map[string]any{"Step": next, "F": f, "Dups": dups})
}

func last4of(nik string) string {
	nik = strings.TrimSpace(nik)
	if len(nik) >= 4 {
		return nik[len(nik)-4:]
	}
	return ""
}

func (a *App) clientCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermCreateClient) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	if r.FormValue("consent") != "1" {
		a.render(w, r, "client_wizard", "Klien Baru", map[string]any{"Step": "3", "F": formMap(r), "Error": "persetujuan klien wajib dicentang"})
		return
	}
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
		a.render(w, r, "client_wizard", "Klien Baru", map[string]any{"Step": "1", "F": formMap(r), "Error": err.Error()})
		return
	}
	// Enrich with wizard fields (step 2-3) in one update.
	uid := a.uid(r)
	pic, _ := strconv.ParseInt(r.FormValue("pic_user_id"), 10, 64)
	_ = a.clients.UpdateFull(a.officeOf(r), clients.ClientFull{
		Client: clients.Client{ID: id, OfficeID: a.officeOf(r), Kind: r.FormValue("kind"),
			FullName: r.FormValue("full_name"), Email: r.FormValue("email"), Phone: r.FormValue("phone"),
			Address: r.FormValue("address"), NPWP: r.FormValue("npwp")},
		RegNumber: autoReg(id, r.FormValue("reg_number")), BirthPlace: r.FormValue("birth_place"),
		BirthDate: r.FormValue("birth_date"), Gender: r.FormValue("gender"),
		Nationality: def(r.FormValue("nationality"), "Indonesia"), Marital: r.FormValue("marital_status"),
		Occupation: r.FormValue("occupation"), Education: r.FormValue("education"),
		Passport: r.FormValue("passport_no"), IDAddress: r.FormValue("id_address"),
		Domicile: r.FormValue("domicile_address"), Province: r.FormValue("province"),
		City: r.FormValue("city"), District: r.FormValue("district"), Village: r.FormValue("village"),
		Postal: r.FormValue("postal_code"), AltPhone: r.FormValue("alt_phone"),
		WANumber: r.FormValue("wa_number"), Correspondence: r.FormValue("correspondence_address"),
		FirstClientDate: r.FormValue("first_client_date"), Source: r.FormValue("source"),
		PIC: pic, Status: "ACTIVE",
	})
	a.audit.Record(audit.Entry{ActorID: uid, Action: "client.create", Entity: "client", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(id, 10)+"?flash=Klien+berhasil+dibuat", http.StatusSeeOther)
}

func formMap(r *http.Request) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"kind", "full_name", "phone", "email", "npwp", "address"} {
		out[k] = r.FormValue(k)
	}
	return out
}

func autoReg(id int64, v string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fmt.Sprintf("KLI-%06d", id)
}

func def(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return v
}

func (a *App) clientDetail(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) && mw.RoleOf(r) != rbac.RoleClient {
		a.forbidden(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := a.clients.GetFull(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if mw.RoleOf(r) == rbac.RoleClient {
		uid := a.uid(r)
		var linked int64
		if uid != nil {
			linked = a.users.LinkedClient(*uid)
		}
		if linked == 0 || linked != c.ID {
			a.forbidden(w, r)
			return
		}
	}
	dups, _ := a.clients.FindDuplicatesEx(a.officeOf(r), id, c.FullName, c.Phone, c.Email, c.NPWP, "", c.BirthDate)
	ids, _ := a.clients.Identities(c.ID)
	verHist := map[int64][]map[string]any{}
	for _, it := range ids {
		if h, _ := a.clients.VerificationHistory(it.ID); len(h) > 0 {
			verHist[it.ID] = h
		}
	}
	reps, _ := a.clients.Reps(c.ID)
	docs, _ := a.uploads.List(a.officeOf(r), "client", c.ID)
	notes, _ := a.clients.Notes(c.ID, 20)
	timeline, _ := a.clients.Timeline(c.ID, 30)
	completeness, missing, _ := a.clients.Completeness(a.officeOf(r), c.ID)
	// Riwayat layanan + akta + keuangan.
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
	// Ringkasan keuangan dari modul finance yang sama (tanpa duplikasi hitung).
	var fin struct {
		Invoices, Paid, Outstanding, Overdue int64
	}
	_ = a.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(paid),0), COALESCE(SUM(total-paid),0)
		FROM invoices WHERE client_id=? AND office_id=? AND status IN ('ISSUED','PARTIAL')`, id, a.officeOf(r)).Scan(&fin.Invoices, &fin.Paid, &fin.Outstanding)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM invoices WHERE client_id=? AND office_id=? AND status IN ('ISSUED','PARTIAL') AND due_date IS NOT NULL AND due_date < CURDATE()`, id, a.officeOf(r)).Scan(&fin.Overdue)
	a.render(w, r, "client_detail", "Detail Klien", map[string]any{
		"Client": c, "Duplicates": dups, "Identities": ids, "VerHist": verHist, "Reps": reps, "Files": docs,
		"Services": svcs, "Trail": trail, "Notes": notes, "Timeline": timeline,
		"Fin":       map[string]int64{"Invoices": fin.Invoices, "Paid": fin.Paid, "Outstanding": fin.Outstanding, "Overdue": fin.Overdue},
		"CanVerify": rbac.Has(mw.RoleOf(r), rbac.PermVerifyIdentity),
		"CanEdit":   rbac.Has(mw.RoleOf(r), rbac.PermEditClient),
		"Complete":  completeness, "Missing": missing,
	})
}

// ---- requests ----

func (a *App) requestList(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	clientID, _ := strconv.ParseInt(r.URL.Query().Get("client"), 10, 64)
	list, _ := a.reqs.List(a.officeOf(r), r.URL.Query().Get("status"), clientID)
	svcs, _ := a.reqs.Services()
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	// Fokus Notaris: layanan Notaris di grup atas, PPAT di grup bawah.
	var notarySvcs, ppatSvcs []requests.Service
	for _, s := range svcs {
		if s.Category == "ppat" {
			ppatSvcs = append(ppatSvcs, s)
		} else {
			notarySvcs = append(notarySvcs, s)
		}
	}
	var clientName string
	if clientID != 0 {
		if c, err := a.clients.Get(a.officeOf(r), clientID); err == nil {
			clientName = c.FullName
		}
	}
	a.render(w, r, "requests", "Permohonan", map[string]any{"Requests": list, "Services": svcs, "NotarySvcs": notarySvcs, "PpatSvcs": ppatSvcs, "Clients": cl, "ClientName": clientName, "St": r.URL.Query().Get("status"), "ClientID": clientID})
}

func (a *App) requestCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermCreateClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	serviceID, _ := strconv.ParseInt(r.FormValue("service_id"), 10, 64)
	notes := r.FormValue("notes")
	// Lampiran terstruktur dari form pintar per kategori (tanpa migrasi:
	// disimpan sebagai baris [Auto] di catatan + record anak yang sudah ada).
	var auto []string
	add := func(k, v string) {
		if v = strings.TrimSpace(v); v != "" {
			auto = append(auto, k+": "+v)
		}
	}
	firstOf := func(keys ...string) string {
		for _, k := range keys {
			for _, v := range r.Form[k] {
				if strings.TrimSpace(v) != "" {
					return v
				}
			}
		}
		return ""
	}
	add("Pihak/para pihak", firstOf("x_parties"))
	add("Objek", r.FormValue("x_object"))
	add("Nilai/transaksi", r.FormValue("x_value"))
	add("Sertifikat", r.FormValue("x_cert"))
	add("Luas/lokasi", strings.TrimSpace(r.FormValue("x_area")+" "+r.FormValue("x_location")))
	add("Pewasiat", r.FormValue("x_testator"))
	add("Ahli waris", r.FormValue("x_heirs"))
	if len(auto) > 0 {
		if notes != "" {
			notes += "\n"
		}
		notes += "[Auto " + r.FormValue("x_svccode") + "] " + strings.Join(auto, " | ")
	}
	id, err := a.reqs.Create(a.officeOf(r), clientID, serviceID, r.FormValue("title"), notes, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uid := a.uid(r)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "request.create", Entity: "service_request", EntityID: strconv.FormatInt(id, 10)})
	// Checklist cerdas (paket PRO+): salin item template layanan sebagai MISSING otomatis.
	// LITE dilewati diam-diam — fitur memang di luar paketnya.
	if plan.Allowed(a.officeTier(a.officeOf(r)), plan.FeatChecklist) {
		_ = a.dms.AutoChecklist(a.officeOf(r), id, serviceID)
	}
	// Korporat: langsung bentuk record perusahaan anak agar wizard/wizard-lanjut tinggal isi.
	if code := r.FormValue("x_svccode"); code != "" {
		if ent := corpEntityOf(code); ent != "" {
			_, _ = a.corp.UpsertCompany(corp.Company{
				RequestID:  &id,
				EntityType: ent,
				Name:       r.FormValue("x_coname"),
				Domicile:   r.FormValue("x_cocity"),
				Address:    r.FormValue("x_coaddr"),
				Authorized: numForm(r, "x_coauth"),
				Issued:     numForm(r, "x_coissued"),
				Paid:       numForm(r, "x_copaid"),
			})
		}
	}
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(id, 10)+"?flash=Permohonan+dibuat.+Lengkapi+data+di+bawah", http.StatusSeeOther)
}

// corpEntityOf memetakan kode layanan ke tipe badan usaha ("" bila bukan korporat).
func corpEntityOf(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	switch {
	case strings.HasPrefix(code, "PT-"):
		return "PT"
	case strings.HasPrefix(code, "CV-"):
		return "CV"
	case code == "FIRMA":
		return "Firma"
	case strings.HasPrefix(code, "YAYASAN"):
		return "Yayasan"
	case code == "KOPERASI":
		return "Koperasi"
	case code == "BUNDEL-PT-OSS":
		return "PT"
	case code == "BUNDEL-CV-NIB":
		return "CV"
	default:
		return ""
	}
}

func (a *App) requestMove(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	target := strings.ToUpper(strings.TrimSpace(r.FormValue("to")))
	// Gate mode ketat checklist: ke IN_REVIEW/APPROVED ditolak selama masih
	// ada item checklist berstatus MISSING. Bila strict mati (atau tabel flag
	// belum ada), perilaku lama dipertahankan.
	if (target == "IN_REVIEW" || target == "APPROVED") && a.strictChecklistEnabled(a.officeOf(r)) && a.dms != nil {
		if items, err := a.dms.ChecklistForRequest(a.officeOf(r), id); err == nil {
			miss := 0
			for _, it := range items {
				if it.Status == "MISSING" {
					miss++
				}
			}
			if miss > 0 {
				msg := fmt.Sprintf("Mode ketat checklist aktif: masih ada %d dokumen berstatus MISSING — lengkapi di checklist bawah.", miss)
				http.Redirect(w, r, "/requests/"+strconv.FormatInt(id, 10)+"?flash="+url.QueryEscape(msg)+"#checklist", http.StatusSeeOther)
				return
			}
		}
	}
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
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	list, _ := a.deeds.List(a.officeOf(r), r.URL.Query().Get("status"))
	// Filter kind di Go (tanpa ubah signature Store.List): cocokkan persis
	// dengan nilai ENUM kind agar aman dari injeksi (perbandingan string Go).
	kind := r.URL.Query().Get("kind")
	if kind != "" {
		kept := list[:0]
		for _, d := range list {
			if d.Kind == kind {
				kept = append(kept, d)
			}
		}
		list = kept
	}
	tpls, _ := a.tpl.List(a.officeOf(r))
	a.render(w, r, "deeds", "Akta", map[string]any{"Deeds": list, "Templates": tpls, "St": r.URL.Query().Get("status"), "Kind": kind})
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
	// 1-klik dari permohonan: bila template tidak dipilih eksplisit, petakan
	// otomatis template approved milik kantor berdasarkan kode layanan.
	// Perilaku template_id eksplisit tidak diubah.
	var svcCode string
	if reqID != nil && a.db != nil {
		_ = a.db.QueryRow(`SELECT sv.code FROM service_requests r
			JOIN services sv ON sv.id=r.service_id
			WHERE r.id=? AND r.office_id=?`, *reqID, a.officeOf(r)).Scan(&svcCode)
	}
	kind := r.FormValue("kind")
	if kind == "" && svcCode != "" {
		kind = serviceDeedKind(svcCode)
	}
	title := r.FormValue("title")
	if title == "" && reqID != nil && a.db != nil {
		var svcName, reqTitle string
		if err := a.db.QueryRow(`SELECT sv.name, r.title FROM service_requests r
			JOIN services sv ON sv.id=r.service_id
			WHERE r.id=? AND r.office_id=?`, *reqID, a.officeOf(r)).Scan(&svcName, &reqTitle); err == nil {
			title = svcName + " — " + reqTitle
		}
	}
	if tplID == nil && svcCode != "" && a.db != nil {
		if id := autoDeedTemplate(a.db, a.officeOf(r), svcCode, kind); id != 0 {
			tplID = &id
		}
	}
	id, err := a.deeds.CreateDraft(a.officeOf(r), reqID, title, kind, tplID, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uid := a.uid(r)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "deed.draft", Entity: "deed", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/deeds/"+strconv.FormatInt(id, 10)+"?flash=Draf+akta+dibuat", http.StatusSeeOther)
}

// serviceDeedKind memetakan kode layanan ke deed kind ("" bila tak dikenal).
func serviceDeedKind(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	switch {
	case strings.HasPrefix(c, "PT-"):
		return "PT"
	case strings.HasPrefix(c, "CV-"):
		return "CV"
	case c == "FIRMA":
		return "Firma"
	case strings.HasPrefix(c, "YAYASAN"):
		return "Yayasan"
	case c == "KOPERASI":
		return "Koperasi"
	case c == "PPAT-AJB":
		return "AJB"
	case c == "PPAT-HIBAH":
		return "Hibah"
	case c == "PPAT-TUKAR":
		return "TukarMenukar"
	case c == "PPAT-APHB":
		return "APHB"
	case c == "PPAT-INBRENG":
		return "Inbreng"
	case c == "PPAT-SKMHT":
		return "SKMHT"
	case c == "PPAT-APHT":
		return "APHT"
	case c == "KUASA":
		return "POA"
	case c == "PRENUP":
		return "Prenup"
	case strings.HasPrefix(c, "WASIAT"):
		return "Will"
	case c == "PK-UMUM", c == "HUTANG", c == "SEWA", c == "JUALBELI-NONPPAT":
		return "Agreement"
	case c == "LEGALISASI", c == "WAARMERKING", c == "SALINAN", c == "KUTIPAN",
		c == "AHU-BANTU", c == "OSS-BANTU":
		return "Other"
	case c == "BUNDEL-PT-OSS":
		return "PT"
	case c == "BUNDEL-CV-NIB":
		return "CV"
	case c == "BUNDEL-AJB-BALIKNAMA":
		return "AJB"
	default:
		return ""
	}
}

// deedTemplateKeyword memetakan kode layanan ke kata kunci pencarian template.
func deedTemplateKeyword(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	switch {
	case strings.HasPrefix(c, "PT-"):
		return "PT"
	case strings.HasPrefix(c, "CV-"):
		return "CV"
	case c == "FIRMA":
		return "Firma"
	case strings.HasPrefix(c, "YAYASAN"):
		return "Yayasan"
	case c == "KOPERASI":
		return "Koperasi"
	case c == "PPAT-AJB":
		return "AJB"
	case c == "PPAT-HIBAH":
		return "Hibah"
	case c == "PPAT-TUKAR":
		return "Tukar"
	case c == "PPAT-APHB":
		return "APHB"
	case c == "PPAT-INBRENG":
		return "Inbreng"
	case c == "PPAT-SKMHT":
		return "SKMHT"
	case c == "PPAT-APHT":
		return "APHT"
	case c == "KUASA":
		return "Kuasa"
	case c == "PRENUP":
		return "Kawin"
	case strings.HasPrefix(c, "WASIAT"):
		return "wasiat"
	case c == "PK-UMUM", c == "HUTANG", c == "SEWA", c == "JUALBELI-NONPPAT":
		return "Perjanjian"
	case c == "BUNDEL-PT-OSS":
		return "PT"
	case c == "BUNDEL-CV-NIB":
		return "CV"
	case c == "BUNDEL-AJB-BALIKNAMA":
		return "AJB"
	default:
		return ""
	}
}

// autoDeedTemplate memilih template approved milik kantor yang paling cocok
// untuk kode layanan: cocok kata kunci dulu, lalu kategori sejenis kind,
// terakhir template approved pertama. Mengembalikan 0 bila tidak ada.
func autoDeedTemplate(db *sql.DB, officeID int64, svcCode, kind string) int64 {
	var id int64
	if kw := deedTemplateKeyword(svcCode); kw != "" {
		like := "%" + kw + "%"
		if err := db.QueryRow(`SELECT id FROM doc_templates
			WHERE office_id=? AND approved=1 AND (name LIKE ? OR category LIKE ?)
			ORDER BY id LIMIT 1`, officeID, like, like).Scan(&id); err == nil {
			return id
		}
	}
	if kind != "" {
		like := "%" + kind + "%"
		if err := db.QueryRow(`SELECT id FROM doc_templates
			WHERE office_id=? AND approved=1 AND (name LIKE ? OR category LIKE ?)
			ORDER BY id LIMIT 1`, officeID, like, like).Scan(&id); err == nil {
			return id
		}
	}
	if err := db.QueryRow(`SELECT id FROM doc_templates
		WHERE office_id=? AND approved=1 ORDER BY id LIMIT 1`, officeID).Scan(&id); err == nil {
		return id
	}
	return 0
}

// ---- registers ----

func (a *App) repertorium(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		a.forbidden(w, r)
		return
	}
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
		"Rows": rows, "Year": year, "Month": month, "Q": search, "Gaps": a.repertoriumGaps(a.officeOf(r), year),
	})
}

// repertoriumGaps finds missing entry_no in a year (numbering discipline, honest gaps).
func (a *App) repertoriumGaps(officeID int64, year string) []int64 {
	q := `SELECT entry_no FROM repertorium WHERE office_id=?`
	args := []any{officeID}
	if year != "" {
		q += ` AND YEAR(entry_date)=?`
		args = append(args, year)
	}
	q += ` ORDER BY entry_no`
	rows, err := a.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var gaps []int64
	var prev int64
	first := true
	for rows.Next() {
		var n int64
		_ = rows.Scan(&n)
		if !first && n > prev+1 {
			for m := prev + 1; m < n && len(gaps) < 50; m++ {
				gaps = append(gaps, m)
			}
		}
		prev, first = n, false
	}
	return gaps
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
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewProtocol) {
		a.forbidden(w, r)
		return
	}
	type row struct {
		ID        int64
		Bundle    string
		Year      int
		Ref       string
		Status    string
		Loc, Hold string
	}
	var rows []row
	if a.db != nil {
		rs, err := a.db.Query(`SELECT p.id, p.bundle_no, p.year, p.minuta_ref, p.status, COALESCE(l.code,''), COALESCE(p.holder,'')
			FROM protocols p LEFT JOIN doc_archive_locations l ON l.id=p.location_id WHERE p.office_id=? ORDER BY p.year DESC, p.bundle_no LIMIT 200`, a.officeOf(r))
		if err == nil {
			defer rs.Close()
			for rs.Next() {
				var x row
				var ref sql.NullString
				_ = rs.Scan(&x.ID, &x.Bundle, &x.Year, &ref, &x.Status, &x.Loc, &x.Hold)
				if ref.Valid {
					x.Ref = ref.String
				}
				rows = append(rows, x)
			}
		}
	}
	locs, _ := a.dms.Locations(a.officeOf(r))
	a.render(w, r, "protocols", "Protokol", map[string]any{"Rows": rows, "Locs": locs})
}

func (a *App) legalisasi(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	type row struct {
		ID                       int64
		Kind, Reg, Title, Status string
		Created                  string
		Queue                    int
		Fee                      float64
	}
	var rows []row
	if a.db != nil {
		rs, err := a.db.Query(`SELECT id, kind, register_no, doc_title, fee, status, DATE_FORMAT(created_at,'%d %b %Y') FROM legalizations WHERE office_id=? ORDER BY id DESC LIMIT 200`, a.officeOf(r))
		if err == nil {
			defer rs.Close()
			for rs.Next() {
				var x row
				var sfee, created string
				_ = rs.Scan(&x.ID, &x.Kind, &x.Reg, &x.Title, &sfee, &x.Status, &created)
				fmt.Sscanf(sfee, "%f", &x.Fee)
				x.Created = created
				if x.Status == "REGISTERED" {
					var q int
					_ = a.db.QueryRow(`SELECT COUNT(*) FROM legalizations WHERE office_id=? AND status='REGISTERED' AND id<=?`, a.officeOf(r), x.ID).Scan(&q)
					x.Queue = q
				}
				rows = append(rows, x)
			}
		}
	}
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	deeds, _ := a.deeds.List(a.officeOf(r))
	a.render(w, r, "legalisasi", "Legalisasi & Waarmerking", map[string]any{"Rows": rows, "Clients": cl, "Deeds": deeds})
}

func (a *App) legalisasiCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermLegalize) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	if _, err := a.clients.Get(a.officeOf(r), clientID); err != nil {
		http.Error(w, "klien tidak ditemukan di kantor ini", http.StatusBadRequest)
		return
	}
	var fee int64
	if f, err := money.Parse(r.FormValue("fee")); err == nil && f >= 0 {
		fee = f
	} else if strings.TrimSpace(r.FormValue("fee")) != "" && strings.TrimSpace(r.FormValue("fee")) != "0" {
		http.Error(w, "biaya tidak valid", http.StatusBadRequest)
		return
	}
	var deedID *int64
	if v := strings.TrimSpace(r.FormValue("deed_id")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			deedID = &n
		}
	}
	_, err := a.deeds.RegisterLegalization(a.officeOf(r), r.FormValue("kind"), r.FormValue("register_no"), clientID, r.FormValue("doc_title"), float64(fee), deedID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "legalisasi.create", Entity: "legalization", EntityID: r.FormValue("register_no")})
	http.Redirect(w, r, "/legalisasi?flash=Register+tersimpan", http.StatusSeeOther)
}

// ---- PPAT register (Buku PPAT terpisah dari Repertorium Notaris) ----

func (a *App) ppatRegister(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	type row struct {
		ID               int64
		No, Date, Jenis  string
		Penjual, Pembeli string
		Objek            string
		Status, BPN      string
		DeedID           int64
		Days             int
		Overdue          bool
	}
	var rows []row
	if a.db != nil {
		rs, err := a.db.Query(`SELECT id, akta_no, DATE_FORMAT(akta_date,'%d %b %Y'), jenis,
			COALESCE(penjual,''), COALESCE(pembeli,''),
			CONCAT(COALESCE(objek_sertifikat,''),' ',COALESCE(objek_luas,''),' ',COALESCE(objek_lokasi,'')),
			status, COALESCE(bpn_receipt,''), COALESCE(deed_id,0),
			DATEDIFF(CURDATE(), COALESCE(akta_date, DATE(created_at)))
			FROM ppat_register WHERE office_id=? ORDER BY id DESC LIMIT 200`, a.officeOf(r))
		if err == nil {
			defer rs.Close()
			for rs.Next() {
				var x row
				var dt sql.NullString
				_ = rs.Scan(&x.ID, &x.No, &dt, &x.Jenis, &x.Penjual, &x.Pembeli, &x.Objek, &x.Status, &x.BPN, &x.DeedID, &x.Days)
				if dt.Valid {
					x.Date = dt.String
				}
				// Pengingat resi BPN: LAPOR_BPN > 14 hari tanpa terbit = overdue.
				x.Overdue = x.Status == "LAPOR_BPN" && x.Days > 14
				rows = append(rows, x)
			}
		}
	}
	deeds, _ := a.deeds.List(a.officeOf(r))
	a.render(w, r, "ppat", "Register PPAT", map[string]any{"Rows": rows, "Deeds": deeds})
}

func (a *App) ppatCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	jenis := r.FormValue("jenis")
	switch jenis {
	case "AJB", "Hibah", "TukarMenukar", "APHB", "Inbreng", "SKMHT", "APHT":
	default:
		http.Error(w, "jenis akta PPAT tidak dikenal", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(r.FormValue("akta_no")) == "" {
		http.Error(w, "nomor akta PPAT wajib diisi", http.StatusBadRequest)
		return
	}
	var deedID any
	if v := strings.TrimSpace(r.FormValue("deed_id")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			deedID = n
		}
	}
	var aktaDate any
	if v := strings.TrimSpace(r.FormValue("akta_date")); v != "" {
		aktaDate = v
	}
	if _, err := a.db.Exec(`INSERT INTO ppat_register(office_id, deed_id, akta_no, akta_date, jenis, penjual, pembeli, objek_sertifikat, objek_luas, objek_lokasi)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, a.officeOf(r), deedID, r.FormValue("akta_no"), aktaDate, jenis,
		r.FormValue("penjual"), r.FormValue("pembeli"), r.FormValue("objek_sertifikat"), r.FormValue("objek_luas"), r.FormValue("objek_lokasi")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "ppat.create", Entity: "ppat_register", EntityID: r.FormValue("akta_no")})
	http.Redirect(w, r, "/ppat?flash=Register+PPAT+tersimpan", http.StatusSeeOther)
}

func (a *App) ppatAdvance(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var cur string
	_ = a.db.QueryRow(`SELECT status FROM ppat_register WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&cur)
	next := map[string]string{
		"DRAFT_AKTA": "AKTA_JADI", "AKTA_JADI": "LAPOR_BPN", "LAPOR_BPN": "SERTIFIKAT_TERBIT",
	}[cur]
	if next == "" {
		http.Error(w, "status sudah final", http.StatusBadRequest)
		return
	}
	bpn := r.FormValue("bpn_receipt")
	if next == "LAPOR_BPN" && strings.TrimSpace(bpn) == "" {
		bpn = "-"
	}
	_, err := a.db.Exec(`UPDATE ppat_register SET status=?, bpn_receipt=CASE WHEN ?<>'' THEN ? ELSE bpn_receipt END WHERE id=? AND office_id=?`,
		next, bpn, bpn, id, a.officeOf(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "ppat.advance." + strings.ToLower(next), Entity: "ppat_register", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/ppat?flash=Status+PPAT+diperbarui", http.StatusSeeOther)
}

// ppatBlanko renders a print-friendly draft sheet for copying onto the
// official BPN form. Honest: this is a working concept, NOT the official blanko.
func (a *App) ppatBlanko(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	type data struct {
		No, Date, Jenis, Penjual, Pembeli, Objek, Status, BPN string
		DeedTitle, DeedBody                                   string
		Office, OfficeAddr                                    string
	}
	var d data
	var deedID sql.NullInt64
	var aktaDate sql.NullString
	_ = a.db.QueryRow(`SELECT akta_no, DATE_FORMAT(akta_date,'%d %B %Y'), jenis,
		COALESCE(penjual,''), COALESCE(pembeli,''),
		TRIM(CONCAT(COALESCE(objek_sertifikat,''),' ',COALESCE(objek_luas,''),' ',COALESCE(objek_lokasi,''))),
		status, COALESCE(bpn_receipt,''), deed_id
		FROM ppat_register WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(
		&d.No, &d.Date, &d.Jenis, &d.Penjual, &d.Pembeli, &d.Objek, &d.Status, &d.BPN, &deedID)
	if d.No == "" {
		http.NotFound(w, r)
		return
	}
	_ = aktaDate
	if deedID.Valid {
		_ = a.db.QueryRow(`SELECT COALESCE(title,''), COALESCE(content,'') FROM deed_drafts WHERE id=? AND office_id=?`,
			deedID.Int64, a.officeOf(r)).Scan(&d.DeedTitle, &d.DeedBody)
	}
	_ = a.db.QueryRow(`SELECT COALESCE(name,''), COALESCE(address,'') FROM offices WHERE id=?`, a.officeOf(r)).Scan(&d.Office, &d.OfficeAddr)
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "ppat.blanko", Entity: "ppat_register", EntityID: strconv.FormatInt(id, 10)})
	a.render(w, r, "ppat_blanko", "Blanko PPAT #"+d.No, map[string]any{"D": d})
}

// ---- gov assisted ----

func (a *App) gov(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
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
	reqID, _ := strconv.ParseInt(r.URL.Query().Get("req"), 10, 64)
	// Suggest tarif otomatis: bila ?req= ada, ambil base_price + nama layanan
	// permohonan itu (office-scoped). Diteruskan ke template sebagai .Data.Suggest.
	var suggestVal any
	if reqID > 0 && a.db != nil {
		var svcName, priceStr string
		if err := a.db.QueryRow(`SELECT s.name, s.base_price FROM service_requests r JOIN services s ON s.id=r.service_id WHERE r.id=? AND r.office_id=?`, reqID, a.officeOf(r)).Scan(&svcName, &priceStr); err == nil {
			suggestVal = map[string]any{"Price": decWhole(priceStr), "ServiceName": svcName}
		}
	}
	// Perkaya baris invoice dengan jenis (DP/PELUNASAN/FULL) + progres bayar.
	// Jenis dibaca dari prefix baris pertama invoice_lines ("[DP] ..."/"[PELUNASAN] ...")
	// karena skema invoices tidak memiliki kolom jenis/catatan — tanpa migrasi.
	type invView struct {
		finance.Invoice
		Kind        string
		Pct         int
		Overdue     bool
		OverdueDays int
	}
	views := make([]invView, 0, len(invs))
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, iv := range invs {
		v := invView{Invoice: iv}
		if iv.Total > 0 && iv.Paid >= 0 {
			p := iv.Paid * 100 / iv.Total
			if p < 0 {
				p = 0
			}
			if p > 100 {
				p = 100
			}
			v.Pct = int(p)
		}
		if a.db != nil {
			var first string
			if err := a.db.QueryRow(`SELECT l.description FROM invoice_lines l JOIN invoices i ON i.id=l.invoice_id WHERE l.invoice_id=? AND i.office_id=? ORDER BY l.id LIMIT 1`, iv.ID, a.officeOf(r)).Scan(&first); err == nil {
				switch {
				case strings.HasPrefix(first, "[DP]"):
					v.Kind = "DP"
				case strings.HasPrefix(first, "[PELUNASAN]"):
					v.Kind = "PELUNASAN"
				}
			}
		}
		views = append(views, v)
	}
	// Flag jatuh tempo: today > due_date dan outstanding > 0.
	// DueDate dari kolom DATE (diparse 10 char pertama agar toleran format driver).
	for i := range views {
		dueStr := strings.TrimSpace(views[i].DueDate)
		if len(dueStr) < 10 || views[i].Outstanding <= 0 {
			continue
		}
		if due, err := time.Parse("2006-01-02", dueStr[:10]); err == nil {
			dueLocal := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, now.Location())
			if days := int(today.Sub(dueLocal).Hours() / 24); days > 0 {
				views[i].Overdue = true
				views[i].OverdueDays = days
			}
		}
	}
	// Overdue di atas (partisi stabil, tanpa mengubah ORDER BY store).
	ordered := make([]invView, 0, len(views))
	for _, v := range views {
		if v.Overdue {
			ordered = append(ordered, v)
		}
	}
	for _, v := range views {
		if !v.Overdue {
			ordered = append(ordered, v)
		}
	}
	views = ordered
	a.render(w, r, "finance", "Keuangan", map[string]any{
		"Invoices": views, "Outstanding": out, "Revenue": rev, "Month": month, "Clients": cl, "ReqID": reqID, "Suggest": suggestVal,
	})
}

func (a *App) invoiceCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	// Taut opsional ke permohonan (divalidasi satu kantor + klien sama).
	var reqID *int64
	if v := strings.TrimSpace(r.FormValue("request_id")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			var rcid, roid int64
			if err := a.db.QueryRow(`SELECT client_id, office_id FROM service_requests WHERE id=?`, n).Scan(&rcid, &roid); err == nil && rcid == clientID && roid == a.officeOf(r) {
				reqID = &n
			}
		}
	}
	descs := r.Form["desc[]"]
	qtys := r.Form["qty[]"]
	prices := r.Form["price[]"]
	// Jenis invoice (FULL/DP/PELUNASAN). Skema invoices tidak punya kolom
	// jenis/catatan, jadi DP/PELUNASAN di-encode sebagai prefix baris pertama
	// "[DP] ..."/"[PELUNASAN] ..." — tanpa migrasi.
	kind := strings.ToUpper(strings.TrimSpace(r.FormValue("invoice_kind")))
	if kind != "DP" && kind != "PELUNASAN" {
		kind = "FULL"
	}
	var lines []finance.Line
	for i := range descs {
		if i >= len(qtys) || i >= len(prices) || descs[i] == "" {
			continue
		}
		q, _ := strconv.Atoi(qtys[i])
		p, perr := money.Parse(prices[i])
		if perr != nil || p < 0 {
			http.Error(w, "harga baris tidak valid: "+descs[i], http.StatusBadRequest)
			return
		}
		lines = append(lines, finance.Line{Description: descs[i], Qty: q, UnitPrice: p})
	}
	if kind != "FULL" && len(lines) > 0 {
		prefix := "[" + kind + "] "
		if !strings.HasPrefix(lines[0].Description, "[DP]") && !strings.HasPrefix(lines[0].Description, "[PELUNASAN]") {
			lines[0].Description = prefix + lines[0].Description
		}
	}
	_, err := a.fin.CreateInvoice(a.officeOf(r), clientID, reqID, r.FormValue("number"), lines, 1100, r.FormValue("due_date"))
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
	amount, err := money.Parse(r.FormValue("amount"))
	if err != nil || amount <= 0 {
		http.Error(w, "nominal tidak valid", http.StatusBadRequest)
		return
	}
	if err := a.fin.RecordPayment(invID, amount, r.FormValue("method"), r.FormValue("reference"), a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "payment.create", Entity: "invoice", EntityID: strconv.FormatInt(invID, 10)})
	http.Redirect(w, r, "/finance?flash=Pembayaran+dicatat", http.StatusSeeOther)
}

func (a *App) invoiceVoid(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	invID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := a.fin.VoidInvoice(a.officeOf(r), invID, r.FormValue("reason")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "invoice.void", Entity: "invoice", EntityID: strconv.FormatInt(invID, 10), Detail: r.FormValue("reason")})
	http.Redirect(w, r, "/finance?flash=Invoice+di-void", http.StatusSeeOther)
}

// ---- misc pages ----

func (a *App) templatesPage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermDraftDeed) {
		a.forbidden(w, r)
		return
	}
	tpls, _ := a.tpl.List(a.officeOf(r))
	clauses, _ := a.tpl.Clauses(a.officeOf(r))
	a.render(w, r, "templates", "Template Dokumen", map[string]any{
		"Templates": tpls, "Clauses": clauses, "Readiness": a.serviceReadiness(a.officeOf(r)),
	})
}

// serviceReadiness menilai kesiapan jual per layanan:
// checklist + template approved + harga + sudah dipakai. Tanpa query per baris.
func (a *App) serviceReadiness(office int64) []map[string]any {
	svcs, _ := a.reqs.Services()
	tpls, _ := a.tpl.List(office)
	lower := func(s string) string { return strings.ToLower(s) }
	hasCL := map[string]bool{}
	if rs, err := a.db.Query(`SELECT service_code, COUNT(*) FROM doc_checklist_templates WHERE office_id=? GROUP BY service_code`, office); err == nil && rs != nil {
		defer rs.Close()
		for rs.Next() {
			var code string
			var n int
			_ = rs.Scan(&code, &n)
			if n > 0 {
				hasCL[code] = true
			}
		}
	}
	used := map[int64]int{}
	if rs, err := a.db.Query(`SELECT service_id, COUNT(*) FROM service_requests WHERE office_id=? GROUP BY service_id`, office); err == nil && rs != nil {
		defer rs.Close()
		for rs.Next() {
			var sid int64
			var n int
			_ = rs.Scan(&sid, &n)
			used[sid] = n
		}
	}
	var out []map[string]any
	for _, s := range svcs {
		kw := lower(deedTemplateKeyword(s.Code))
		tplOK, tplName := false, ""
		if kw != "" {
			for _, t := range tpls {
				if !t.Approved {
					continue
				}
				if strings.Contains(lower(t.Name+" "+t.Category), kw) {
					tplOK, tplName = true, t.Name
					break
				}
			}
		}
		score := 0
		if hasCL[s.Code] {
			score += 25
		}
		if tplOK {
			score += 40
		}
		if s.BasePrice > 0 {
			score += 15
		}
		if used[s.ID] > 0 {
			score += 20
		}
		status := "BELUM"
		if score >= 80 {
			status = "SIAP JUAL"
		} else if score >= 40 {
			status = "KURANG"
		}
		out = append(out, map[string]any{
			"Code": s.Code, "Name": s.Name, "Cat": s.Category, "Price": s.BasePrice,
			"CL": hasCL[s.Code], "Tpl": tplName, "TplOK": tplOK,
			"Used": used[s.ID], "Score": score, "Status": status,
		})
	}
	return out
}

func (a *App) reports(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewReports) {
		a.forbidden(w, r)
		return
	}
	pending, active, completed := 0, 0, 0
	var out int64
	if a.db != nil {
		pending, active, completed, _ = a.reqs.Counts(a.officeOf(r))
		out, _ = a.fin.Outstanding(a.officeOf(r))
	}
	// Anggaran vs realisasi bulan berjalan per kategori.
	type brow struct {
		Category      string
		Budget, Spent int64
	}
	var budgets []brow
	month := time.Now().Format("2006-01-01")
	brows, _ := a.db.Query(`SELECT b.category, b.amount, COALESCE((SELECT SUM(e.amount) FROM expenses e
		WHERE e.office_id=b.office_id AND e.category=b.category AND DATE_FORMAT(e.spent_at,'%Y-%m-01')=b.month),0)
		FROM budgets b WHERE b.office_id=? AND b.month=? ORDER BY b.category`, a.officeOf(r), month)
	if brows != nil {
		defer brows.Close()
		for brows.Next() {
			var x brow
			var bamt, samt string
			_ = brows.Scan(&x.Category, &bamt, &samt)
			x.Budget = decWhole(bamt)
			x.Spent = decWhole(samt)
			budgets = append(budgets, x)
		}
	}
	// Laba per layanan: hanya invoice yang tertaut permohonan (jujur; sisanya "tanpa tautan").
	type prow struct {
		Service      string
		Count, Total int64
	}
	var profits []prow
	prows, _ := a.db.Query(`SELECT COALESCE(s.name,'(tanpa tautan layanan)'), COUNT(i.id), COALESCE(SUM(i.total),0)
		FROM invoices i LEFT JOIN service_requests r ON r.id=i.request_id LEFT JOIN services s ON s.id=r.service_id
		WHERE i.office_id=? AND i.status IN ('ISSUED','PARTIAL','PAID') GROUP BY 1 ORDER BY 3 DESC`, a.officeOf(r))
	if prows != nil {
		defer prows.Close()
		for prows.Next() {
			var x prow
			var tot string
			_ = prows.Scan(&x.Service, &x.Count, &tot)
			x.Total = decWhole(tot)
			profits = append(profits, x)
		}
	}
	// Rekap per kategori layanan: jumlah permohonan + total invoice + total paid
	// (3 query GROUP BY kategori layanan, office-scoped).
	type catRow struct {
		Category       string
		Reqs           int64
		InvTotal, Paid int64
	}
	byCatMap := map[string]*catRow{}
	if a.db != nil {
		if q1, err := a.db.Query(`SELECT s.category, COUNT(r.id) FROM service_requests r JOIN services s ON s.id=r.service_id WHERE r.office_id=? GROUP BY s.category`, a.officeOf(r)); err == nil && q1 != nil {
			defer q1.Close()
			for q1.Next() {
				var c string
				var n int64
				_ = q1.Scan(&c, &n)
				if _, ok := byCatMap[c]; !ok {
					byCatMap[c] = &catRow{Category: c}
				}
				byCatMap[c].Reqs = n
			}
		}
		if q2, err := a.db.Query(`SELECT s.category, COALESCE(SUM(i.total),0) FROM invoices i JOIN service_requests r ON r.id=i.request_id JOIN services s ON s.id=r.service_id WHERE i.office_id=? AND i.status IN ('ISSUED','PARTIAL','PAID') GROUP BY s.category`, a.officeOf(r)); err == nil && q2 != nil {
			defer q2.Close()
			for q2.Next() {
				var c, tot string
				_ = q2.Scan(&c, &tot)
				if _, ok := byCatMap[c]; !ok {
					byCatMap[c] = &catRow{Category: c}
				}
				byCatMap[c].InvTotal = decWhole(tot)
			}
		}
		if q3, err := a.db.Query(`SELECT s.category, COALESCE(SUM(i.paid),0) FROM invoices i JOIN service_requests r ON r.id=i.request_id JOIN services s ON s.id=r.service_id WHERE i.office_id=? AND i.status IN ('ISSUED','PARTIAL','PAID') GROUP BY s.category`, a.officeOf(r)); err == nil && q3 != nil {
			defer q3.Close()
			for q3.Next() {
				var c, tot string
				_ = q3.Scan(&c, &tot)
				if _, ok := byCatMap[c]; !ok {
					byCatMap[c] = &catRow{Category: c}
				}
				byCatMap[c].Paid = decWhole(tot)
			}
		}
	}
	var byCat []catRow
	for _, v := range byCatMap {
		byCat = append(byCat, *v)
	}
	// Top 5 layanan bulan berjalan: nama layanan + jumlah invoice + total dibayar,
	// filter bulan berjalan, office-scoped.
	type topRow struct {
		Service     string
		Count, Paid int64
	}
	var topSvc []topRow
	if a.db != nil {
		ym := time.Now().Format("2006-01")
		if trows, err := a.db.Query(`SELECT COALESCE(s.name,'(tanpa tautan layanan)'), COUNT(i.id), COALESCE(SUM(i.paid),0)
			FROM invoices i LEFT JOIN service_requests r ON r.id=i.request_id LEFT JOIN services s ON s.id=r.service_id
			WHERE i.office_id=? AND DATE_FORMAT(i.created_at,'%Y-%m')=? AND i.status IN ('ISSUED','PARTIAL','PAID')
			GROUP BY 1 ORDER BY 3 DESC LIMIT 5`, a.officeOf(r), ym); err == nil && trows != nil {
			defer trows.Close()
			for trows.Next() {
				var x topRow
				var paid string
				_ = trows.Scan(&x.Service, &x.Count, &paid)
				x.Paid = decWhole(paid)
				topSvc = append(topSvc, x)
			}
		}
	}
	// Papan SLA: permohonan terbuka yang melewati target umur layanannya (batasi 20).
	// ServiceOverdue hanya memakai kolom yang sudah ada (r.created_at untuk umur).
	var slaOver []remind.ServiceOverdueItem
	if a.db != nil {
		if over, err := remind.ServiceOverdue(a.db, a.officeOf(r)); err == nil {
			if len(over) > 20 {
				over = over[:20]
			}
			slaOver = over
		}
	}
	a.render(w, r, "reports", "Laporan", map[string]any{
		"Pending": pending, "Active": active, "Completed": completed, "Outstanding": out,
		"Budgets": budgets, "Profits": profits, "Month": time.Now().Format("2006-01"), "ByCat": byCat,
		"TopSvc": topSvc, "SLAOver": slaOver,
	})
}

func (a *App) settings(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	list, _ := a.users.List(a.officeOf(r))
	a.render(w, r, "settings", "Pengaturan", map[string]any{"Users": list, "Roles": rbac.Roles()})
}

// portalNextAction maps the 9-stage request status to a client-facing next step.
func portalNextAction(status string) string {
	switch status {
	case "DRAFT":
		return "Lengkapi draf lalu kirim permohonan"
	case "SUBMITTED":
		return "Menunggu verifikasi awal kantor"
	case "IN_REVIEW":
		return "Sedang direview notaris/staf"
	case "NEED_DOCS":
		return "Lengkapi dokumen"
	case "APPROVED":
		return "Disetujui, menunggu proses minuta"
	case "IN_PROGRESS":
		return "Menunggu notaris"
	case "COMPLETED":
		return "Selesai, unduh PDF"
	case "REJECTED":
		return "Ditolak, hubungi kantor"
	case "CANCELLED":
		return "Dibatalkan"
	default:
		return "Hubungi kantor untuk info lanjut"
	}
}

// portalStep maps status to the 5-stage visual stepper:
// 1 Diajukan > 2 Dokumen > 3 Review > 4 Proses > 5 Selesai. 0 = terminal gagal.
func portalStep(status string) int {
	switch status {
	case "DRAFT", "SUBMITTED":
		return 1
	case "NEED_DOCS":
		return 2
	case "IN_REVIEW":
		return 3
	case "APPROVED", "IN_PROGRESS":
		return 4
	case "COMPLETED":
		return 5
	default:
		return 0
	}
}

// portalTimeline returns the 5 latest request_events for one request (newest first).
func (a *App) portalTimeline(requestID int64) []map[string]any {
	rows, err := a.db.Query(`SELECT e.from_status, e.to_status, COALESCE(u.name,'sistem'), e.created_at, COALESCE(e.note,'')
		FROM request_events e LEFT JOIN users u ON u.id=e.actor_id
		WHERE e.request_id=? ORDER BY e.id DESC LIMIT 5`, requestID)
	if err != nil || rows == nil {
		return nil
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var from, to, actor, at, note string
		_ = rows.Scan(&from, &to, &actor, &at, &note)
		out = append(out, map[string]any{"From": from, "To": to, "Actor": actor, "At": at, "Note": note})
	}
	return out
}

// portalDeedID finds one downloadable deed for the request (0 when none).
// Wasiat (kind 'Will') dikecualikan: isinya notaris-only via /wills + /deeds guard.
func (a *App) portalDeedID(requestID int64, r *http.Request) int64 {
	var id int64
	_ = a.db.QueryRow(`SELECT d.id FROM deed_drafts d WHERE d.request_id=? AND d.office_id=? AND d.kind<>'Will' ORDER BY d.id DESC LIMIT 1`, requestID, a.officeOf(r)).Scan(&id)
	return id
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
			list = append(list, map[string]any{"ID": id, "Title": title, "Status": status,
				"NextAction": portalNextAction(status), "Step": portalStep(status), "Failed": status == "REJECTED" || status == "CANCELLED",
				"Timeline": a.portalTimeline(id), "DeedID": a.portalDeedID(id, r)})
		}
		// Akta milik klien (unduh PDF diizinkan via canViewDeed server-side).
		// Wasiat (kind 'Will') dikecualikan: terbatas notaris, tidak tampil di portal.
		drows, _ := a.db.Query(`SELECT d.id, d.title, d.number, d.status FROM deed_drafts d
			JOIN service_requests r ON r.id=d.request_id
			WHERE r.client_id=? AND d.office_id=? AND d.kind<>'Will' ORDER BY d.id DESC`, clientID, a.officeOf(r))
		var deeds []map[string]any
		if drows != nil {
			defer drows.Close()
			for drows.Next() {
				var did int64
				var title, number, status string
				_ = drows.Scan(&did, &title, &number, &status)
				deeds = append(deeds, map[string]any{"ID": did, "Title": title, "Number": number, "Status": status})
			}
		}
		a.render(w, r, "portal", "Portal Klien", map[string]any{"Mine": list, "Deeds": deeds})
		return
	}
	list, _ := a.reqs.List(a.officeOf(r), "", 0)
	a.render(w, r, "portal", "Portal Klien", map[string]any{"Requests": list})
}
