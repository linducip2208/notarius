// Package cms implements White Label branding + Website CMS storage for a
// single office (standalone, no cross-office reads). All writes are validated;
// section settings JSON is schema-checked, never free-form.
package cms

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	slugRe  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// Branding is the per-office white label identity + theme.
type Branding struct {
	AppName, ShortName, Tagline                 string
	Primary, Secondary, Accent                  string
	SidebarBG, HeaderBG                         string
	ButtonStyle                                 string
	BorderRadius                                int
	FontFamily                                  string
	LogoRef, FaviconRef, LoginBG                string
	FooterText, CopyrightText                   string
	DefaultLang, Timezone, DateFormat, Currency string
}

var fonts = map[string]string{
	"system": "system-ui,-apple-system,'Segoe UI',Roboto,Arial,sans-serif",
	"serif":  "Georgia,'Times New Roman',serif",
	"mono":   "ui-monospace,SFMono-Regular,Menlo,monospace",
}

// FontStack maps the stored choice to a local-only CSS stack.
func FontStack(choice string) string {
	if s, ok := fonts[choice]; ok {
		return s
	}
	return fonts["system"]
}

// ButtonRadius maps style+radius to CSS.
func ButtonRadius(style string, r int) string {
	if r < 0 {
		r = 0
	}
	if r > 24 {
		r = 24
	}
	switch style {
	case "square":
		return "2px"
	case "pill":
		return "999px"
	default:
		return fmt.Sprintf("%dpx", r)
	}
}

func cleanColor(v, fb string) string {
	if colorRe.MatchString(v) {
		return strings.ToLower(v)
	}
	return fb
}

// SiteSettings holds public site + domain configuration (honest, no fake TLS claims).
type SiteSettings struct {
	SiteTitle, SiteDesc, DefaultOG string
	BaseURL, PrimaryDomain         string
	EnforceHTTPS                   bool
	DomainVerified                 bool
	LastSeenHost                   string
	SetupDone                      bool
}

type Store struct {
	db *sql.DB
	// cache: rendered public fragments per office, invalidated on writes.
	mu    sync.RWMutex
	cache map[string]cached
}

type cached struct {
	at   time.Time
	data any
}

const cacheTTL = 5 * time.Minute

func NewStore(db *sql.DB) *Store { return &Store{db: db, cache: map[string]cached{}} }

func (s *Store) Invalidate(officeID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := fmt.Sprintf("o%d:", officeID)
	for k := range s.cache {
		if strings.HasPrefix(k, p) {
			delete(s.cache, k)
		}
	}
}

func (s *Store) getCached(key string, fill func() (any, error)) (any, error) {
	s.mu.RLock()
	if c, ok := s.cache[key]; ok && time.Since(c.at) < cacheTTL {
		s.mu.RUnlock()
		return c.data, nil
	}
	s.mu.RUnlock()
	v, err := fill()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[key] = cached{at: time.Now(), data: v}
	s.mu.Unlock()
	return v, nil
}

// ---- branding ----

// BrandingOf returns branding with safe defaults when the row is missing.
func (s *Store) BrandingOf(officeID int64) (Branding, error) {
	var b Branding
	var radius int
	var style, font string
	err := s.db.QueryRow(`SELECT app_name, short_name, tagline, primary_color, secondary_color,
		accent_color, sidebar_bg, header_bg, button_style, border_radius, font_family,
		logo_ref, favicon_ref, login_bg, footer_text, copyright_text,
		default_lang, timezone, date_format, currency
		FROM office_branding WHERE office_id=?`, officeID).Scan(
		&b.AppName, &b.ShortName, &b.Tagline, &b.Primary, &b.Secondary, &b.Accent,
		&b.SidebarBG, &b.HeaderBG, &style, &radius, &font, &b.LogoRef, &b.FaviconRef,
		&b.LoginBG, &b.FooterText, &b.CopyrightText,
		&b.DefaultLang, &b.Timezone, &b.DateFormat, &b.Currency)
	if err == sql.ErrNoRows {
		return DefaultBranding(), nil
	}
	if err != nil {
		return Branding{}, err
	}
	b.ButtonStyle = style
	b.BorderRadius = radius
	b.FontFamily = font
	return b, nil
}

// DefaultBranding is the fallback (never empty, never broken CSS).
func DefaultBranding() Branding {
	return Branding{
		AppName: "SaktiNotaris", ShortName: "SN",
		Primary: "#1f6feb", Secondary: "#101f3c", Accent: "#c9a844",
		SidebarBG: "#101f3c", HeaderBG: "#ffffff",
		ButtonStyle: "rounded", BorderRadius: 8, FontFamily: "system",
		DefaultLang: "id", Timezone: "Asia/Jakarta", DateFormat: "02 Jan 2006", Currency: "IDR",
	}
}

// SaveBranding validates (colors, enum, font, radius) then upserts + invalidates.
func (s *Store) SaveBranding(officeID int64, b Branding) error {
	b.Primary = cleanColor(b.Primary, "#1f6feb")
	b.Secondary = cleanColor(b.Secondary, "#101f3c")
	b.Accent = cleanColor(b.Accent, "#c9a844")
	b.SidebarBG = cleanColor(b.SidebarBG, "#101f3c")
	b.HeaderBG = cleanColor(b.HeaderBG, "#ffffff")
	switch b.ButtonStyle {
	case "square", "rounded", "pill":
	default:
		b.ButtonStyle = "rounded"
	}
	if _, ok := fonts[b.FontFamily]; !ok {
		b.FontFamily = "system"
	}
	if b.BorderRadius < 0 || b.BorderRadius > 24 {
		b.BorderRadius = 8
	}
	if b.AppName == "" {
		return errors.New("nama aplikasi wajib diisi")
	}
	_, err := s.db.Exec(`INSERT INTO office_branding(office_id, app_name, short_name, tagline,
		primary_color, secondary_color, accent_color, sidebar_bg, header_bg, button_style,
		border_radius, font_family, logo_ref, favicon_ref, login_bg, footer_text, copyright_text,
		default_lang, timezone, date_format, currency) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE app_name=VALUES(app_name), short_name=VALUES(short_name),
		tagline=VALUES(tagline), primary_color=VALUES(primary_color), secondary_color=VALUES(secondary_color),
		accent_color=VALUES(accent_color), sidebar_bg=VALUES(sidebar_bg), header_bg=VALUES(header_bg),
		button_style=VALUES(button_style), border_radius=VALUES(border_radius), font_family=VALUES(font_family),
		logo_ref=VALUES(logo_ref), favicon_ref=VALUES(favicon_ref), login_bg=VALUES(login_bg),
		footer_text=VALUES(footer_text), copyright_text=VALUES(copyright_text),
		default_lang=VALUES(default_lang), timezone=VALUES(timezone), date_format=VALUES(date_format),
		currency=VALUES(currency)`,
		officeID, b.AppName, b.ShortName, b.Tagline, b.Primary, b.Secondary, b.Accent,
		b.SidebarBG, b.HeaderBG, b.ButtonStyle, b.BorderRadius, b.FontFamily,
		b.LogoRef, b.FaviconRef, b.LoginBG, b.FooterText, b.CopyrightText,
		b.DefaultLang, b.Timezone, b.DateFormat, b.Currency)
	if err != nil {
		return err
	}
	s.Invalidate(officeID)
	return nil
}

// ---- site settings ----

func (s *Store) SiteOf(officeID int64) (SiteSettings, error) {
	var st SiteSettings
	var https, verified, setup int
	err := s.db.QueryRow(`SELECT site_title, site_desc, default_og, base_url, primary_domain,
		enforce_https, domain_verified, last_seen_host, setup_done
		FROM website_settings WHERE office_id=?`, officeID).Scan(
		&st.SiteTitle, &st.SiteDesc, &st.DefaultOG, &st.BaseURL, &st.PrimaryDomain,
		&https, &verified, &st.LastSeenHost, &setup)
	if err == sql.ErrNoRows {
		return SiteSettings{}, nil
	}
	if err != nil {
		return SiteSettings{}, err
	}
	st.EnforceHTTPS = https == 1
	st.DomainVerified = verified == 1
	st.SetupDone = setup == 1
	return st, nil
}

// SaveSite validates hostnames/URLs (rejects javascript: and friends) then upserts.
func (s *Store) SaveSite(officeID int64, st SiteSettings) error {
	if st.PrimaryDomain != "" && !validHost(st.PrimaryDomain) {
		return errors.New("domain utama tidak valid")
	}
	if st.BaseURL != "" && !validHTTPURL(st.BaseURL) {
		return errors.New("base URL harus http(s)://host valid")
	}
	https, verified, setup := 0, 0, 0
	if st.EnforceHTTPS {
		https = 1
	}
	if st.DomainVerified {
		verified = 1
	}
	if st.SetupDone {
		setup = 1
	}
	_, err := s.db.Exec(`INSERT INTO website_settings(office_id, site_title, site_desc, default_og,
		base_url, primary_domain, enforce_https, domain_verified, setup_done)
		VALUES(?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE site_title=VALUES(site_title),
		site_desc=VALUES(site_desc), default_og=VALUES(default_og), base_url=VALUES(base_url),
		primary_domain=VALUES(primary_domain), enforce_https=VALUES(enforce_https),
		domain_verified=VALUES(domain_verified), setup_done=VALUES(setup_done)`,
		officeID, st.SiteTitle, st.SiteDesc, st.DefaultOG, st.BaseURL, st.PrimaryDomain,
		https, verified, setup)
	if err != nil {
		return err
	}
	s.Invalidate(officeID)
	return nil
}

// NoteHost records the host actually seen (for honest domain verification UX).
func (s *Store) NoteHost(officeID int64, host string) {
	if h := strings.ToLower(strings.TrimSpace(host)); h != "" {
		_, _ = s.db.Exec(`UPDATE website_settings SET last_seen_host=? WHERE office_id=?`, h, officeID)
	}
}

var hostRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)

func validHost(h string) bool {
	h = strings.ToLower(strings.TrimSpace(h))
	if len(h) == 0 || len(h) > 253 {
		return false
	}
	if strings.Contains(h, "://") || strings.Contains(h, "/") {
		return false
	}
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	return hostRe.MatchString(h)
}

func validHTTPURL(u string) bool {
	u = strings.TrimSpace(u)
	if !(strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")) {
		return false
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	host := rest
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		host = rest[:i]
	}
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return validHost(host)
}

// ValidNavURL allows relative paths and http(s) absolute URLs only.
func ValidNavURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	low := strings.ToLower(u)
	if strings.HasPrefix(low, "javascript:") || strings.HasPrefix(low, "data:") ||
		strings.HasPrefix(low, "vbscript:") || strings.HasPrefix(low, "file:") {
		return false
	}
	if strings.HasPrefix(u, "/") {
		return !strings.HasPrefix(u, "//")
	}
	return validHTTPURL(u)
}
