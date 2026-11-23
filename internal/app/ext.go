package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/clients"
	"notaryos/internal/corp"
	"notaryos/internal/cryptoxx"
	"notaryos/internal/deliver"
	"notaryos/internal/diff"
	"notaryos/internal/docengine"
	"notaryos/internal/doctpl"
	"notaryos/internal/docx"
	"notaryos/internal/finance"
	"notaryos/internal/gov"
	"notaryos/internal/mfa"
	mw "notaryos/internal/middleware"
	"notaryos/internal/money"
	"notaryos/internal/notify"
	"notaryos/internal/ocr"
	"notaryos/internal/pdf"
	"notaryos/internal/rbac"
	"notaryos/internal/remind"
	"notaryos/internal/safehtml"
)

// ---- MFA ----

// signPendingMFA issues a stateless, HMAC-signed step-up token (uid:exp:mac).
// Survives restarts and multi-instance; 5-minute expiry; key from cryptoxx.
func (a *App) signPendingMFA(userID int64) string {
	exp := time.Now().Add(5 * time.Minute).Unix()
	msg := fmt.Sprintf("%d:%d", userID, exp)
	key, err := cryptoxx.Key()
	if err != nil {
		return ""
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("mfa-stepup:" + msg))
	return base64.RawURLEncoding.EncodeToString([]byte(msg)) + "." + hex.EncodeToString(mac.Sum(nil))
}

// verifyPendingMFA validates the step-up token and returns the user id.
func (a *App) verifyPendingMFA(token string) (int64, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return 0, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, false
	}
	key, err := cryptoxx.Key()
	if err != nil {
		return 0, false
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("mfa-stepup:" + string(raw)))
	if !hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(parts[1])) {
		return 0, false
	}
	var uid, exp int64
	if _, err := fmt.Sscanf(string(raw), "%d:%d", &uid, &exp); err != nil || uid <= 0 {
		return 0, false
	}
	if time.Now().Unix() > exp {
		return 0, false
	}
	return uid, true
}

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
	// Token stateless bertanda HMAC (tahan restart/replika): uid:exp:mac.
	uid, ok := a.verifyPendingMFA(c.Value)
	if !ok {
		http.Redirect(w, r, "/login?flash=Sesi+verifikasi+kedaluwarsa", http.StatusSeeOther)
		return
	}
	var role string
	_ = a.db.QueryRow(`SELECT r.name FROM users u JOIN roles r ON r.id=u.role_id WHERE u.id=?`, uid).Scan(&role)
	if !a.users.VerifyMFA(uid, r.FormValue("code")) {
		// Audit tanpa kode MFA: hanya uid + IP untuk deteksi brute-force TOTP.
		a.audit.Record(audit.Entry{ActorID: &uid, Action: "mfa.failed", Entity: "user", EntityID: strconv.FormatInt(uid, 10), Detail: clientIPOf(r)})
		a.render(w, r, "mfa_verify", "Verifikasi MFA", map[string]string{"Error": "Kode MFA salah."})
		return
	}
	a.audit.Record(audit.Entry{ActorID: &uid, Action: "login", Entity: "user", EntityID: strconv.FormatInt(uid, 10), Detail: "via MFA", OfficeID: a.officeOfUser(uid)})
	sess, err := a.sessions.Create(uid, role)
	if err != nil {
		http.Error(w, "gagal membuat sesi", http.StatusInternalServerError)
		return
	}
	lip := clientIPOf(r)
	_, _ = a.db.Exec(`UPDATE sessions SET ip=?, user_agent=? WHERE token=?`, lip, r.UserAgent(), sess.Token)
	var seen, ever int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id=? AND ip=? AND token<>?`, uid, lip, sess.Token).Scan(&seen)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id=?`, uid).Scan(&ever)
	if seen == 0 && ever > 1 {
		_, _ = a.db.Exec(`INSERT INTO notifications(user_id, title, body) VALUES(?,?,?)`, uid, "Login dari IP baru", "Akun Anda login dari "+lip+". Abaikan bila itu Anda.")
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionName + "_mfa", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: a.cfg.TLSCookie})
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionName, Value: sess.Token, Path: "/",
		Expires: sess.ExpiresAt, MaxAge: 43200, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.cfg.TLSCookie})
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
		a.render(w, r, "mfa", "Aktifkan MFA", map[string]any{"Secret": secret, "Error": "Kode salah — pindai ulang & coba lagi.", "Enabled": a.users.MFAEnabled(*id)})
		return
	}
	// Re-autentikasi: enroll MFA wajib membuktikan kata sandi saat ini.
	var email string
	_ = a.db.QueryRow(`SELECT email FROM users WHERE id=?`, *id).Scan(&email)
	if _, err := a.users.Authenticate(email, r.FormValue("password")); err != nil {
		a.render(w, r, "mfa", "Aktifkan MFA", map[string]any{"Secret": secret, "Error": "Kata sandi salah — enroll MFA dibatalkan.", "Enabled": a.users.MFAEnabled(*id)})
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
	// Legacy direct-verify kept for compatibility; routed through the audited
	// IdentityStatus path (requires verifier note? no — default note recorded).
	if !rbac.Has(mw.RoleOf(r), rbac.PermVerifyIdentity) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var cid int64
	_ = a.db.QueryRow(`SELECT c.id FROM identities i JOIN clients c ON c.id=i.client_id WHERE i.id=? AND c.office_id=?`, id, a.officeOf(r)).Scan(&cid)
	if cid == 0 {
		http.NotFound(w, r)
		return
	}
	if err := a.clients.IdentityStatus(a.officeOf(r), id, "VERIFIED", "verifikasi langsung (legacy)", a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "identity.verify", Entity: "identity", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(cid, 10)+"?flash=Identitas+terverifikasi#tab-id", http.StatusSeeOther)
}

func (a *App) repAdd(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
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
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
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
	if a.overQuota(a.officeOf(r), hdr.Size) {
		http.Error(w, "kuota penyimpanan kantor terlampaui — hubungi administrator", http.StatusInsufficientStorage)
		return
	}
	id, err := a.uploads.SaveUpload(a.officeOf(r), "client", clientID, hdr.Filename, hdr.Header.Get("Content-Type"), f, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.upload", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	// DMS: auto-relasi ke klien + antre OCR (best-effort, tak menghambat respons).
	_ = a.dms.AutoLink(a.officeOf(r), id, "client", clientID, a.uid(r))
	// Portal: beri tahu akun klien yang tertaut bahwa ada berkas baru.
	if urows, err := a.db.Query(`SELECT user_id FROM user_client_links WHERE client_id=?`, clientID); err == nil {
		defer urows.Close()
		for urows.Next() {
			var linkUID int64
			_ = urows.Scan(&linkUID)
			_, _ = a.db.Exec(`INSERT INTO notifications(user_id, title, body) VALUES(?,?,?)`, linkUID, "Berkas baru", "Kantor mengunggah berkas baru ke arsip Anda.")
		}
	}
	_ = a.dms.AutoLink(a.officeOf(r), id, "client", clientID, a.uid(r))
	if _, err := a.dms.EnqueueOCR(a.officeOf(r), id); err == nil {
		_ = a.dms.Move(a.officeOf(r), id, "CLASSIFICATION", a.uid(r), "otomatis saat unggah")
	}
	// Upload Quality Gate (jujur, tak memblokir): nilai berkas tersimpan.
	// PDF / decode gagal → abaikan gate, tetap flash sukses biasa.
	flash := "Berkas diunggah"
	if _, rc, gerr := a.uploads.Get(a.officeOf(r), id); gerr == nil {
		func() {
			defer rc.Close()
			tmp, terr := os.CreateTemp("", "quality-*")
			if terr != nil {
				return
			}
			tname := tmp.Name()
			defer os.Remove(tname)
			if _, terr := io.Copy(tmp, io.LimitReader(rc, 100<<20)); terr != nil {
				tmp.Close()
				return
			}
			tmp.Close()
			q, qerr := ocr.AssessImage(tname)
			if qerr != nil {
				return
			}
			switch q.Grade {
			case "D_RESHOOT_REQUIRED":
				flash = "Berkas tersimpan TAPI kualitas D (blur/gelap) — foto ulang lalu jalankan OCR ulang"
				a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.quality.warn", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: "grade=D score=" + strconv.FormatFloat(q.Score, 'f', 0, 64)})
			case "C_RETRY_RECOMMENDED":
				flash = "Berkas tersimpan, kualitas C — disarankan foto ulang"
				a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.quality.warn", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: "grade=C score=" + strconv.FormatFloat(q.Score, 'f', 0, 64)})
			}
		}()
	}
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(clientID, 10)+"?flash="+url.QueryEscape(flash), http.StatusSeeOther)
}

func (a *App) fileOCR(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
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

// fileDownload streams a private file after office-scope + confidentiality authorization.
func (a *App) fileDownload(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) && !rbac.Has(mw.RoleOf(r), rbac.PermViewProtocol) {
		a.forbidden(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var conf string
	_ = a.db.QueryRow(`SELECT confidentiality FROM files WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&conf)
	if !a.canSeeDoc(r, conf) {
		a.forbidden(w, r)
		return
	}
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
	if r.URL.Query().Get("verify") == "1" {
		// Integritas on-demand: hash ulang dari storage, bandingkan dengan DB
		// SEBELUM byte dikirim; mismatch = 409 + karantina, bukan byte rusak.
		_, hr, herr := a.uploads.Get(a.officeOf(r), id)
		if herr != nil {
			http.NotFound(w, r)
			return
		}
		h := sha256.New()
		_, herr = io.Copy(h, hr)
		hr.Close()
		if herr != nil {
			http.Error(w, "gagal membaca berkas untuk verifikasi", http.StatusInternalServerError)
			return
		}
		var stored string
		_ = a.db.QueryRow(`SELECT sha256 FROM files WHERE id=?`, id).Scan(&stored)
		got := fmt.Sprintf("%x", h.Sum(nil))
		if stored != "" && stored != got {
			_, _ = a.db.Exec(`UPDATE files SET verification_status='REJECTED' WHERE id=?`, id)
			a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.hash.mismatch", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: "db=" + stored + " got=" + got})
			http.Error(w, "hash berkas tidak cocok dengan catatan (dikarantina sebagai REJECTED)", http.StatusConflict)
			return
		}
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.download.verify", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: got})
	}
	_, _ = io.Copy(w, rc)
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "file.download", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
}

// ---- request detail + corporate ----

// firstWord returns the first whitespace-separated token (for fuzzy name compare).
func firstWord(s string) string {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}

// partyView adalah satu baris matter_parties + nama klien (migrasi 026).
type partyView struct {
	ID         int64
	ClientID   int64
	ClientName string
	Role       string
}

func (a *App) requestDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	d, err := a.reqs.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Isolasi portal: klien hanya boleh melihat permohonannya sendiri.
	if mw.RoleOf(r) == rbac.RoleClient {
		uid := a.uid(r)
		var linked int64
		if uid != nil {
			linked = a.users.LinkedClient(*uid)
		}
		if linked == 0 || linked != d.ClientID {
			a.forbidden(w, r)
			return
		}
	} else if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	co, shs, offs, _ := a.corp.GetByRequest(id)
	// Peringatan konsistensi korporat (peringatan, bukan blokir).
	var corpWarn []string
	if co.ID != 0 {
		var totalShares int
		var pctSum float64
		for _, s := range shs {
			totalShares += s.Shares
			if co.Issued > 0 {
				pctSum += float64(s.Shares) / float64(co.Issued) * 100
			}
			var cname string
			_ = a.db.QueryRow(`SELECT c.full_name FROM clients c JOIN shareholders sh ON sh.client_id=c.id WHERE sh.id=?`, s.ID).Scan(&cname)
			if cname != "" && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(firstWord(cname))) && !strings.Contains(strings.ToLower(cname), strings.ToLower(firstWord(s.Name))) {
				corpWarn = append(corpWarn, "nama pemegang saham \""+s.Name+"\" tidak mirip klien tertaut ("+cname+") — periksa salah ketik")
			}
		}
		if len(shs) > 0 && co.Issued > 0 && (pctSum < 99.5 || pctSum > 100.5) {
			corpWarn = append(corpWarn, fmt.Sprintf("total %d lembar tidak sama dengan modal ditempatkan %d — selisih harus dijelaskan", totalShares, co.Issued))
		}
		if co.Paid > co.Issued {
			corpWarn = append(corpWarn, "modal disetor melebihi ditempatkan — tidak wajar")
		}
		if co.Issued > co.Authorized {
			corpWarn = append(corpWarn, "modal ditempatkan melebihi modal dasar — tidak wajar")
		}
	}
	var bos []corp.BeneficialOwner
	var legal corp.Legal
	if co.ID != 0 {
		bos, _ = a.corp.BeneficialOwners(co.ID)
		legal, _ = a.corp.LegalOf(co.ID)
	}
	subs, _ := a.govs.List(a.officeOf(r))
	checklist, _ := a.dms.ChecklistForRequest(a.officeOf(r), id)
	linked, _ := a.dms.ByRelated(a.officeOf(r), "request", id)
	done, total := 0, len(checklist)
	for _, c := range checklist {
		if c.Status == "VERIFIED" || c.Status == "RECEIVED" {
			done++
		}
	}
	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}
	// Tautan WA manual (tanpa layanan berbayar): nomor dari data klien,
	// pesan berisi ringkasan permohonan. Kosong = tombol disembunyikan.
	var cliPhone, cliWA sql.NullString
	_ = a.db.QueryRow(`SELECT phone, wa_number FROM clients WHERE id=?`, d.ClientID).Scan(&cliPhone, &cliWA)
	waPhone := ""
	waText := fmt.Sprintf("Halo %s, menindaklanjuti permohonan #%d %q (%s) di kantor kami.", d.ClientName, d.ID, d.Title, d.Status)
	if cliPhone.Valid || cliWA.Valid {
		waNum := ""
		if cliWA.Valid {
			waNum = cliWA.String
		}
		ph := ""
		if cliPhone.Valid {
			ph = cliPhone.String
		}
		waPhone = deliver.PreferWA(waNum, ph)
	}
	waLink := ""
	if waPhone != "" {
		waLink = deliver.WALink(waPhone, waText)
	}
	// PPAT terkait: 1 query JOIN deed_drafts.request_id (kecil, kolom seperlunya).
	type ppatRef struct {
		ID          int64
		No, Jenis   string
		Status      string
	}
	var ppatRefs []ppatRef
	if a.db != nil {
		if rs, err := a.db.Query(`SELECT p.id, p.akta_no, p.jenis, p.status
			FROM ppat_register p JOIN deed_drafts d ON d.id=p.deed_id
			WHERE d.request_id=? AND p.office_id=? ORDER BY p.id DESC`, id, a.officeOf(r)); err == nil {
			defer rs.Close()
			for rs.Next() {
				var x ppatRef
				if err := rs.Scan(&x.ID, &x.No, &x.Jenis, &x.Status); err == nil {
					ppatRefs = append(ppatRefs, x)
				}
			}
		}
	}
	// Para pihak matter (migrasi 026): JOIN clients, office-scoped via request.
	var parties []partyView
	if a.db != nil {
		if rs, err := a.db.Query(`SELECT mp.id, mp.client_id, c.full_name, mp.role
			FROM matter_parties mp JOIN clients c ON c.id=mp.client_id
			WHERE mp.office_id=? AND mp.request_id=? ORDER BY mp.id`, a.officeOf(r), id); err == nil {
			defer rs.Close()
			for rs.Next() {
				var p partyView
				if err := rs.Scan(&p.ID, &p.ClientID, &p.ClientName, &p.Role); err == nil {
					parties = append(parties, p)
				}
			}
		}
	}
	partyClients, _ := a.clients.List(a.officeOf(r), "", 200)
	wfSteps, wfCur, wfStatus, wfHas := a.workflowView(a.officeOf(r), id)
	// Penawaran resmi terikat permohonan (migrasi 030): daftar + prefill tarif.
	var quots []finance.Quotation
	var quotSvc string
	var quotPrice int64
	if a.db != nil {
		quots, _ = a.fin.QuotationsForRequest(a.officeOf(r), id)
		quotSvc, quotPrice = a.quotationPrefill(r, id)
	}
	a.render(w, r, "request_detail", "Detail Permohonan", map[string]any{
		"Req": d, "Company": co, "Legal": legal, "Shareholders": shs, "Officers": offs, "BOs": bos, "Gov": subs, "CorpWarn": corpWarn,
		"Checklist": checklist, "LinkedDocs": linked, "CheckDone": done, "CheckTotal": total, "CheckPct": pct,
		"SvcCode": a.svcCodeOf(id),
		"WAPhone": waPhone, "WAText": waText, "WALink": waLink,
		"Ppat": ppatRefs,
		"Parties": parties, "PartyClients": partyClients, "PartyRoles": partyRoles,
		"WF": wfSteps, "WFHas": wfHas, "WFCur": wfCur, "WFStatus": wfStatus,
		"Quotations": quots, "QuotSvc": quotSvc, "QuotPrice": quotPrice, "QuotDue": quotDefaultDue(),
	})
}

// svcCodeOf mengambil kode layanan permohonan (untuk link wizard yang sesuai).
// Gagal diam-diam -> "" (template menyembunyikan link wizard khusus).
func (a *App) svcCodeOf(reqID int64) string {
	var code sql.NullString
	_ = a.db.QueryRow(`SELECT s.code FROM service_requests r JOIN services s ON s.id=r.service_id WHERE r.id=?`, reqID).Scan(&code)
	if code.Valid {
		return code.String
	}
	return ""
}

func numForm(r *http.Request, k string) int64 {
	n, err := money.Parse(r.FormValue(k))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func parseIDR(s string) (float64, error) {
	v, err := money.Parse(s)
	return float64(v), err
}

func (a *App) corpSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
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
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.reqs.Get(a.officeOf(r), reqID); err != nil {
		http.NotFound(w, r)
		return
	}
	co, _, _, _ := a.corp.GetByRequest(reqID)
	if co.ID == 0 {
		http.Error(w, "simpan data perusahaan dulu", http.StatusBadRequest)
		return
	}
	shares, _ := strconv.Atoi(r.FormValue("shares"))
	nominal := numForm(r, "nominal")
	if err := a.corp.AddShareholder(co.ID, r.FormValue("name"), shares, nominal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "corp.shareholder.add", Entity: "company", EntityID: strconv.FormatInt(co.ID, 10)})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Pemegang+saham+ditambah", http.StatusSeeOther)
}

func (a *App) officerAdd(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.reqs.Get(a.officeOf(r), reqID); err != nil {
		http.NotFound(w, r)
		return
	}
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

// corpWizard renders the guided per-entity corporate flow with completeness
// progress. All saves reuse the existing corp endpoints (single source of truth).
func (a *App) corpWizard(w http.ResponseWriter, r *http.Request) {
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
	co, shs, offs, _ := a.corp.GetByRequest(reqID)
	var bos []corp.BeneficialOwner
	if co.ID != 0 {
		bos, _ = a.corp.BeneficialOwners(co.ID)
	}
	steps, warns := corp.WizardProgress(co, shs, offs, len(bos))
	done := 0
	for _, s := range steps {
		if s.Done {
			done++
		}
	}
	a.render(w, r, "corp_wizard", "Wizard Korporat", map[string]any{
		"Req": d, "Company": co, "Shareholders": shs, "Officers": offs, "BOs": bos,
		"Steps": steps, "Warnings": warns, "Done": done, "Total": len(steps),
	})
}

// corpLegalSave stores manually-verified corporate registry references (no AHU API).
func (a *App) corpLegalSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermDraftDeed) && !rbac.Has(mw.RoleOf(r), rbac.PermEditDeed) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.reqs.Get(a.officeOf(r), reqID); err != nil {
		http.NotFound(w, r)
		return
	}
	co, _, _, _ := a.corp.GetByRequest(reqID)
	if co.ID == 0 {
		http.Error(w, "simpan data perusahaan dulu", http.StatusBadRequest)
		return
	}
	err := a.corp.SaveLegal(co.ID, corp.Legal{
		DeedNo: r.FormValue("deed_no"), DeedDate: r.FormValue("deed_date"),
		AmendmentNo: r.FormValue("amendment_no"), AmendmentDate: r.FormValue("amendment_date"),
		AHUNo: r.FormValue("ahu_no"), NPWP: r.FormValue("npwp"), NIB: r.FormValue("nib"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "corp.legal.save", Entity: "company", EntityID: strconv.FormatInt(co.ID, 10)})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Data+legal+tersimpan", http.StatusSeeOther)
}

// shareholderLink binds a shareholder row to an existing client (for cross-linking).
func (a *App) shareholderLink(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	sid, _ := strconv.ParseInt(chi.URLParam(r, "sid"), 10, 64)
	cid, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	if _, err := a.reqs.Get(a.officeOf(r), reqID); err != nil {
		http.NotFound(w, r)
		return
	}
	co, _, _, _ := a.corp.GetByRequest(reqID)
	if co.ID == 0 {
		http.Error(w, "perusahaan tidak ditemukan", http.StatusNotFound)
		return
	}
	if _, err := a.clients.Get(a.officeOf(r), cid); err != nil {
		http.Error(w, "klien tidak ditemukan", http.StatusNotFound)
		return
	}
	if _, err := a.db.Exec(`UPDATE shareholders SET client_id=? WHERE id=? AND company_id=?`, nullCID(cid), sid, co.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "corp.shareholder.link", Entity: "company", EntityID: strconv.FormatInt(co.ID, 10), Detail: fmt.Sprintf("sh#%d->c#%d", sid, cid)})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Pemegang+saham+tertaut", http.StatusSeeOther)
}

func nullCID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

func (a *App) boAdd(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermCreateClient) && !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.reqs.Get(a.officeOf(r), reqID); err != nil {
		http.NotFound(w, r)
		return
	}
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

// partyRoles adalah daftar peran baku para pihak (allowlist tunggal).
var partyRoles = []string{"PENGHADAP", "PENJUAL", "PEMBELI", "KREDITUR", "DEBITUR", "PENJAMIN", "PEMBERI_KUASA", "PENERIMA_KUASA", "SAKSI", "PIHAK"}

func validPartyRole(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	for _, r := range partyRoles {
		if s == r {
			return true
		}
	}
	return false
}

// partyAdd menautkan klien existing sebagai pihak matter (office-scoped + audit).
func (a *App) partyAdd(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.reqs.Get(a.officeOf(r), reqID); err != nil {
		http.NotFound(w, r)
		return
	}
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	role := strings.ToUpper(strings.TrimSpace(r.FormValue("role")))
	if clientID <= 0 || !validPartyRole(role) {
		http.Error(w, "klien dan peran baku wajib diisi", http.StatusBadRequest)
		return
	}
	if _, err := a.clients.Get(a.officeOf(r), clientID); err != nil {
		http.Error(w, "klien tidak ditemukan di kantor ini", http.StatusNotFound)
		return
	}
	if _, err := a.db.Exec(`INSERT INTO matter_parties(office_id, request_id, client_id, role) VALUES(?,?,?,?)`,
		a.officeOf(r), reqID, clientID, role); err != nil {
		http.Error(w, "gagal menambah pihak (kemungkinan duplikat klien+peran)", http.StatusConflict)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "party.add", Entity: "service_request", EntityID: strconv.FormatInt(reqID, 10), Detail: fmt.Sprintf("c#%d:%s", clientID, role)})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Pihak+ditambahkan", http.StatusSeeOther)
}

// partyRemove melepas satu pihak matter (office-scoped + audit).
func (a *App) partyRemove(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	pid, _ := strconv.ParseInt(chi.URLParam(r, "pid"), 10, 64)
	if _, err := a.reqs.Get(a.officeOf(r), reqID); err != nil {
		http.NotFound(w, r)
		return
	}
	res, err := a.db.Exec(`DELETE FROM matter_parties WHERE id=? AND request_id=? AND office_id=?`, pid, reqID, a.officeOf(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.NotFound(w, r)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "party.remove", Entity: "service_request", EntityID: strconv.FormatInt(reqID, 10), Detail: "party#" + strconv.FormatInt(pid, 10)})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Pihak+dihapus", http.StatusSeeOther)
}

// ---- appointments ----

func (a *App) appointments(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	list, _ := a.reqs.Appointments(a.officeOf(r))
	cl, _ := a.clients.List(a.officeOf(r), "", 200)
	names := map[int64]string{}
	for _, c := range cl {
		names[c.ID] = c.FullName
	}
	// Agenda 14 hari ke depan dikelompokkan per tanggal (kalender mingguan sederhana).
	type item struct {
		Time, Client, Purpose, Status string
	}
	var week []map[string]any
	seen := map[string]int{}
	for _, ap := range list {
		day := ap.At.Format("2006-01-02")
		i, ok := seen[day]
		if !ok {
			week = append(week, map[string]any{"Day": day, "Items": []item{}})
			i = len(week) - 1
			seen[day] = i
			if len(week) >= 14 {
				break
			}
		}
		items := week[i]["Items"].([]item)
		week[i]["Items"] = append(items, item{Time: ap.At.Format("15:04"), Client: names[ap.ClientID], Purpose: ap.Purpose, Status: ap.Status})
	}
	a.render(w, r, "appointments", "Janji Temu", map[string]any{"List": list, "Clients": cl, "Week": week})
}

func (a *App) appointmentCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	if _, err := a.clients.Get(a.officeOf(r), clientID); err != nil {
		http.Error(w, "klien tidak ditemukan di kantor ini", http.StatusBadRequest)
		return
	}
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

// templateSeed memasang paket template KONSEP (DRAFT, belum approved) agar
// draf 1-klik langsung menemukan template. Notaris WAJIB review & approve
// sebelum dipakai — template konsep tidak bisa dipakai membuat draf.
func (a *App) templateSeed(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	office := a.officeOf(r)
	std := standardTemplateSeeds()
	made := 0
	for _, s := range std {
		var n int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM doc_templates WHERE office_id=? AND name=?`, office, s.name).Scan(&n)
		if n > 0 {
			continue
		}
		if _, err := a.tpl.Save(office, 0, s.name, s.cat, s.body); err == nil {
			made++
		}
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "template.seed", Entity: "office", EntityID: strconv.FormatInt(office, 10), Detail: strconv.Itoa(made) + " konsep"})
	http.Redirect(w, r, "/templates?flash="+url.QueryEscape(strconv.Itoa(made)+" template konsep dipasang — review lalu Approve"), http.StatusSeeOther)
}

const tplHead = `<p><strong>KONSEP — wajib direview &amp; disetujui notaris sebelum dipakai.</strong></p>` + "\n"

const tplPT = tplHead + `<h3>AKTA PENDIRIAN PERSEROAN TERBATAS {{company_name}}</h3>
<p>Pada hari ini, ______________________, menghadap kepada saya, {{notary_name}}, Notaris di {{notary_office}}, dengan dihadiri saksi-saksi, para penghadap yang telah dikenalkan:</p>
<p>{{shareholder_list}}</p>
<p>Para penghadap menerangkan mendirikan Perseroan Terbatas dengan ketentuan:</p>
<p>Nama: <strong>{{company_name}}</strong>, berkedudukan di {{company_address}}.<br/>
Modal dasar: Rp {{authorized_capital}}, ditempatkan Rp {{issued_capital}}.<br/>
Direksi: {{director_list}}</p>
<p><em>Lengkapi pasal modal, saham, RUPS, dan penutup sesuai kebutuhan — lalu kirim Review.</em></p>`

const tplCV = tplHead + `<h3>AKTA PENDIRIAN PERSEKUTUAN KOMANDITER (CV) {{company_name}}</h3>
<p>Menghadap kepada saya, {{notary_name}}, Notaris di {{notary_office}}, para penghadap:</p>
<p>{{shareholder_list}}</p>
<p>Para penghadap sepakat mendirikan CV bernama <strong>{{company_name}}</strong>, berkedudukan di {{company_address}}, dengan maksud dan tujuan usaha sebagaimana diuraikan dalam akta ini.</p>
<p><em>Lengkapi pasal sekutu aktif/pasif, modal, keuntungan, dan penutup — lalu kirim Review.</em></p>`

const tplAJB = tplHead + `<h3>AKTA JUAL BELI No. ___</h3>
<p>Pada hari ini, ______________________, menghadap kepada saya, PPAT di wilayah kerja ______________________, dengan dihadiri saksi-saksi:</p>
<p>I. Nama: ______________________, NIK: ______________________ (PIHAK PERTAMA/Penjual).<br/>
II. Nama: {{client_name}}, NIK: {{client_nik}} (PIHAK KEDUA/Pembeli).</p>
<p>Pihak Pertama menjual kepada Pihak Kedua sebidang tanah sebagaimana Sertifikat Hak ______ No. ______, luas ______ m2, terletak di ______________________, dengan harga Rp ______________________.</p>
<p><em>Lengkapi uraian objek, pembayaran, PPh/BPHTB, dan penutup sesuai blanko — lalu kirim Review.</em></p>`

const tplHibah = tplHead + `<h3>AKTA HIBAH No. ___</h3>
<p>Menghadap kepada saya, PPAT di wilayah kerja ______________________, Pihak Pertama menerangkan menghibahkan kepada Pihak Kedua:</p>
<p>Nama penerima: {{client_name}}, NIK: {{client_nik}}, berupa ______________________ (uraikan objek), yang diterima dengan rasa syukur oleh Pihak Kedua.</p>
<p><em>Lengkapi uraian objek, syarat hibah, dan penutup — lalu kirim Review.</em></p>`

const tplKuasa = tplHead + `<h3>SURAT KUASA No. ___</h3>
<p>Yang bertanda tangan di bawah ini, Nama: {{client_name}}, NIK: {{client_nik}}, memberi kuasa kepada ______________________, NIK ______________________, untuk ______________________ (ruang lingkup kuasa).</p>
<p>Kuasa ini berlaku sejak ______________________ sampai ______________________. Dibuat di hadapan {{notary_name}}, Notaris di {{notary_office}}.</p>`

const tplSewa = tplHead + `<h3>PERJANJIAN SEWA MENYEWA</h3>
<p>PIHAK PERTAMA (Pemilik): {{client_name}}, NIK {{client_nik}}.<br/>PIHAK KEDUA (Penyewa): ______________________.</p>
<p>Objek sewa: ______________________. Jangka waktu: ______ s/d ______. Harga sewa: Rp ______________________ dibayar ______________________.</p>
<p><em>Lengkapi hak-kewajiban, denda, dan penyelesaian sengketa — lalu kirim Review.</em></p>`

const tplPrenup = tplHead + `<h3>PERJANJIAN KAWIN (PRENUP)</h3>
<p>Calon suami: ______________________. Calon istri: {{client_name}}.</p>
<p>Para pihak sepakat memisahkan harta bawaan dan harta yang diperoleh selama perkawinan sebagaimana daftar terlampir, serta pengaturannya bila perkawinan berakhir.</p>
<p>Dibuat sebelum perkawinan dilangsungkan, di hadapan {{notary_name}}, Notaris di {{notary_office}}.</p>`

const tplPK = tplHead + `<h3>PERJANJIAN KERJASAMA</h3>
<p>PIHAK PERTAMA: {{client_name}} ({{company_name}}, {{company_address}}).<br/>PIHAK KEDUA: ______________________.</p>
<p>Maksud dan tujuan: ______________________. Hak dan kewajiban, jangka waktu, serta penyelesaian perselisihan diuraikan dalam pasal-pasal berikut.</p>
<p><em>Lengkapi pasal tujuan, kontribusi, keuntungan, dan penutup — lalu kirim Review.</em></p>`

// tplWasiat adalah template KONSEP wasiat (DRAFT, belum approved).
// Akses terbatas (hanya peran notaris/auditor, lihat willsPage & deedDetail);
// pengesahan wajib mencatat 2 saksi.
const tplWasiat = tplHead + `<h3>SURAT WASIAT / TESTAMEN</h3>
<p>Pewasiat: {{client_name}}, NIK {{client_nik}}, beralamat di ______________________.</p>
<p>Ahli waris / penerima wasiat: ______________________ (hubungan: ______________________).</p>
<p>Objek wasiat: ______________________ (uraikan harta + bukti kepemilikan).</p>
<p>Dibuat di hadapan {{notary_name}}, Notaris di {{notary_office}}, dengan dihadiri 2 (dua) orang saksi yang dicatat saat pengesahan.</p>
<p><em>Catatan: data wasiat akses terbatas — hanya notaris & auditor yang dapat melihat; staf/portal diblokir. Lengkapi daftar harta, saksi, dan penutup — lalu kirim Review.</em></p>`

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
	// Gerbang pra-pengesahan: nomor + isi wajib ada; tampilkan jujur bila belum.
	var number, body, st string
	_ = a.db.QueryRow(`SELECT COALESCE(number,''), COALESCE(content,''), status FROM deed_drafts WHERE id=?`, id).Scan(&number, &body, &st)
	if strings.TrimSpace(r.FormValue("number")) == "" && strings.TrimSpace(number) == "" {
		http.Error(w, "nomor akta wajib diisi sebelum pengesahan", http.StatusBadRequest)
		return
	}
	if len(strings.TrimSpace(safehtml.ToText(body))) < 50 {
		http.Error(w, "isi akta terlalu pendek/belum ada — lengkapi draf dulu", http.StatusBadRequest)
		return
	}
	if st == "APPROVED" {
		http.Error(w, "akta sudah disahkan — gunakan revisi tercatat untuk perubahan", http.StatusConflict)
		return
	}
	// Mandat pengganti: substitute_notary wajib punya mandat aktif hari ini.
	if mw.RoleOf(r) == rbac.RoleSubstitute {
		var m int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM substitute_mandates WHERE office_id=? AND user_id=? AND start_date<=CURDATE() AND end_date>=CURDATE()`,
			office, approver).Scan(&m)
		if m == 0 {
			http.Error(w, "notaris pengganti wajib memiliki mandat aktif (SK + periode) — catat di administrasi jabatan", http.StatusForbidden)
			return
		}
	}
	// Formalitas: saksi + pembacaan wajib tercatat (kolom baru, datarkan dari form).
	w1, w2 := strings.TrimSpace(r.FormValue("witness1")), strings.TrimSpace(r.FormValue("witness2"))
	rdPlace := strings.TrimSpace(r.FormValue("read_place"))
	if w1 == "" || w2 == "" || rdPlace == "" {
		http.Error(w, "formalitas belum lengkap: dua saksi + tempat pembacaan wajib diisi", http.StatusBadRequest)
		return
	}
	_, _ = a.db.Exec(`UPDATE deed_drafts SET witness1_name=?, witness1_id=?, witness2_name=?, witness2_id=?, read_at=NOW(), read_place=?, renvoi_note=? WHERE id=?`,
		w1, nullStrOpt(r.FormValue("witness1_id")), w2, nullStrOpt(r.FormValue("witness2_id")), rdPlace, nullStrOpt(r.FormValue("renvoi_note")), id)
	// Aturan workflow per layanan (menggantikan cek checklist manual):
	// checklist, identitas pihak, jumlah pihak, dan kesesuaian jenis akta
	// dinilai engine dari service_workflow_rules. Syarat saksi + tempat
	// pembacaan + isi + hash + mandat di sekitar blok ini TETAP berlaku.
	var reqID sql.NullInt64
	_ = a.db.QueryRow(`SELECT request_id FROM deed_drafts WHERE id=?`, id).Scan(&reqID)
	if reqID.Valid {
		if blocks := workflowGate(a.db, office, reqID.Int64, "", "approve"); len(blocks) > 0 {
			http.Error(w, "aturan workflow permohonan #"+strconv.FormatInt(reqID.Int64, 10)+": "+strings.Join(blocks, "; "), http.StatusConflict)
			return
		}
	}
	// Integritas: bandingkan hash isi (docengine, 16 hex) dengan yang tersimpan.
	// Self-heal hanya bila kolom kosong (draf lama); mismatch nyata ditolak.
	var stored string
	_ = a.db.QueryRow(`SELECT COALESCE(content_hash,'') FROM deed_drafts WHERE id=?`, id).Scan(&stored)
	fresh := docengine.VersionHash(body, nil)
	if stored == "" {
		_, _ = a.db.Exec(`UPDATE deed_drafts SET content_hash=? WHERE id=?`, fresh, id)
	} else if stored != fresh {
		http.Error(w, "hash isi tidak cocok dengan tersimpan — kemungkinan perubahan di luar versi; hubungi administrator", http.StatusConflict)
		return
	}
	if err := a.deeds.Approve(id, r.FormValue("number"), approver); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	seal, _ := verifyCode("DEED", id)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "deed.approve", Entity: "deed", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("number") + " segel:" + seal, OfficeID: officePtr(office)})
	// Tutup loop workflow: akta disahkan → step kunci "sah" DONE
	// (best-effort, fail-open).
	if reqID.Valid {
		aid, aname := a.workflowActorName(r)
		a.autoWorkflowStep(office, reqID.Int64, aid, aname, "akta #"+strconv.FormatInt(id, 10)+" disahkan", "sah")
		// Status mengikuti kejadian: APPROVED → IN_PROGRESS bila CanMove
		// mengizinkan, else diam (tetap manual). Fail-open: tak blokir alur.
		advanceOnEvent(a.db, office, reqID.Int64, "deed_approved", uid, a.audit)
	}
	var entryNo int64
	_ = a.db.QueryRow(`SELECT entry_no FROM repertorium WHERE office_id=? AND deed_id=?`, office, id).Scan(&entryNo)
	a.audit.Record(audit.Entry{ActorID: uid, Action: "repertorium.insert", Entity: "repertorium", EntityID: strconv.FormatInt(entryNo, 10), Detail: "akta#" + strconv.FormatInt(id, 10), OfficeID: officePtr(office)})
	http.Redirect(w, r, "/deeds/"+strconv.FormatInt(id, 10)+"?flash=Akta+disahkan+segel:"+seal, http.StatusSeeOther)
}

// deedSign records an internal electronic signature (hash tersertifikasi sistem,
// BUKAN tanda tangan PSrE/BSrE). Only notary, only APPROVED deeds, one per signer
// per content hash — replay of the same hash is rejected honestly.
func (a *App) deedSign(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) {
		http.Error(w, "hanya notaris yang dapat menandatangani akta", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	office := a.officeOf(r)
	var status, body, number string
	if err := a.db.QueryRow(`SELECT status, content, number FROM deed_drafts WHERE id=? AND office_id=?`, id, office).
		Scan(&status, &body, &number); err != nil {
		http.NotFound(w, r)
		return
	}
	if status != "APPROVED" {
		http.Error(w, "akta harus disahkan (APPROVED) sebelum ditandatangani", http.StatusBadRequest)
		return
	}
	uid := a.uid(r)
	var signerID int64
	var signerName string
	if uid != nil {
		signerID = *uid
		_ = a.db.QueryRow(`SELECT name FROM users WHERE id=?`, signerID).Scan(&signerName)
	}
	seal, _ := verifyCode("DEED", id)
	// Ikat waktu penandatanganan (nonce): dua penandatangan berbeda tak pernah
	// menghasilkan hash sama; replay hash lama tetap ditolak.
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	hash := sha256Of(fmt.Sprintf("%s|%s|%d|%s|%s|%s", body, number, signerID, signerName, seal, now))
	var dup int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_signatures WHERE deed_id=? AND content_hash=?`, id, hash).Scan(&dup)
	if dup > 0 {
		http.Error(w, "isi yang sama sudah ditandatangani — ubah revisi dulu bila perlu tanda baru", http.StatusConflict)
		return
	}
	if _, err := a.db.Exec(`INSERT INTO deed_signatures(office_id, deed_id, signer_user_id, signer_name, content_hash, seal_code, note)
		VALUES(?,?,?,?,?,?,?)`, office, id, signerID, signerName, hash, seal, r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Transisi status APPROVED → SIGNED (office-scoped; hanya dari APPROVED).
	// SIGNED bersifat final laporannya: rekap dasbor sudah menghitung
	// SIGNED (app.go monthly query IN APPROVED/SIGNED/ARCHIVED), tombol
	// tanda tangan hanya tampil untuk APPROVED, dan penandatangan kedua
	// otomatis ditolak gate status di atas. SIGNED → ARCHIVED lewat
	// deedArchive di bawah (arsip minuta final, tanpa hapus data).
	if _, err := a.db.Exec(`UPDATE deed_drafts SET status='SIGNED' WHERE id=? AND office_id=? AND status='APPROVED'`, id, office); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: uid, Action: "deed.sign", Entity: "deed", EntityID: strconv.FormatInt(id, 10), Detail: "hash:" + hash[:16] + " segel:" + seal, OfficeID: officePtr(office)})
	http.Redirect(w, r, "/deeds/"+strconv.FormatInt(id, 10)+"?flash=Tanda+tangan+internal+tercatat", http.StatusSeeOther)
}

// deedArchive menutup lifecycle akta: SIGNED → ARCHIVED (arsip minuta final).
// Guard: PermApproveDeed (setara deedApprove/deedSign/deedHandover —
// melepas minuta dari peredaran aktif). Hanya dari status SIGNED; status
// lain ditolak (ARCHIVED yang sudah arsip → 409, selain itu → 400).
// Tanpa hapus data, tanpa ubah minuta (hanya kolom status yang berubah).
func (a *App) deedArchive(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) {
		http.Error(w, "hanya notaris yang dapat mengarsipkan akta", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	office := a.officeOf(r)
	var status string
	if err := a.db.QueryRow(`SELECT status FROM deed_drafts WHERE id=? AND office_id=?`, id, office).Scan(&status); err != nil {
		http.NotFound(w, r)
		return
	}
	if status == "ARCHIVED" {
		http.Error(w, "akta sudah diarsipkan", http.StatusConflict)
		return
	}
	if status != "SIGNED" {
		http.Error(w, "hanya akta SIGNED yang dapat diarsipkan (status saat ini: "+status+")", http.StatusBadRequest)
		return
	}
	res, err := a.db.Exec(`UPDATE deed_drafts SET status='ARCHIVED' WHERE id=? AND office_id=? AND status='SIGNED'`, id, office)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "gagal mengarsipkan: status berubah bersamaan", http.StatusConflict)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "deed.archive", Entity: "deed", EntityID: strconv.FormatInt(id, 10), OfficeID: officePtr(office)})
	http.Redirect(w, r, "/deeds/"+strconv.FormatInt(id, 10)+"?flash=Akta+diarsipkan", http.StatusSeeOther)
}

// deedHandover records serah dokumen akta ke klien (tanda terima serah dokumen).
// Guard: PermApproveDeed, BUKAN PermEditDeed — serah minuta/salinan/grosse
// melepas dokumen otentik dari penyimpanan notaris sehingga sensitivitasnya
// setara pengesahan & TTE internal (deedApprove/deedSign memakai izin yang
// sama) dan meniru willHandover yang notaris-only. Staf/operator dokumen
// (pemegang PermEditDeed) boleh menyiapkan draf, tetapi tidak boleh
// menyerahkan minuta. PermApproveDeed dipegang Notary + Substitute +
// SuperAdmin (rbac.rolePerms), persis set peran willHandover.
func (a *App) deedHandover(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) {
		a.forbidden(w, r)
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
	to := strings.TrimSpace(r.FormValue("handed_to"))
	if to == "" {
		http.Error(w, "nama penerima serah-terima wajib diisi", http.StatusBadRequest)
		return
	}
	var creator any
	if uid := a.uid(r); uid != nil {
		creator = *uid
	}
	if _, err := a.db.Exec(`INSERT INTO deed_handovers(office_id, deed_id, handed_to, handed_at, note, created_by)
		VALUES(?,?,?,UTC_TIMESTAMP(),?,?)`, office, id, to, nullStrOpt(r.FormValue("note")), creator); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "deed.handover", Entity: "deed", EntityID: strconv.FormatInt(id, 10), Detail: to, OfficeID: officePtr(office)})
	// Status mengikuti kejadian + tutup loop workflow engine (keduanya
	// best-effort, fail-open, meniru deedApprove): step "Serah" DONE bila
	// akta tertaut permohonan, dan permohonan IN_PROGRESS → COMPLETED bila
	// CanMove mengizinkan (else diam, tetap manual).
	if reqID := deedRequestID(a.db, id); reqID != 0 {
		aid, aname := a.workflowActorName(r)
		a.autoWorkflowStep(office, reqID, aid, aname, "akta #"+strconv.FormatInt(id, 10)+" diserahkan ke "+to, "serah")
		advanceOnEvent(a.db, office, reqID, "handover_done", a.uid(r), a.audit)
	}
	http.Redirect(w, r, "/deeds/"+strconv.FormatInt(id, 10)+"?flash=Serah-terima+akta+dicatat", http.StatusSeeOther)
}

// deedHandoverView adalah satu baris riwayat serah dokumen untuk deed_detail.
type deedHandoverView struct {
	To, At, Note, By string
}

// deedHandovers memuat riwayat serah dokumen satu akta (office-scoped).
func (a *App) deedHandovers(officeID, deedID int64) []deedHandoverView {
	var out []deedHandoverView
	if a.db == nil {
		return out
	}
	rs, err := a.db.Query(`SELECT h.handed_to, COALESCE(h.handed_at,''), COALESCE(h.note,''), COALESCE(u.name,'')
		FROM deed_handovers h LEFT JOIN users u ON u.id=h.created_by
		WHERE h.office_id=? AND h.deed_id=? ORDER BY h.id`, officeID, deedID)
	if err != nil {
		return out
	}
	defer rs.Close()
	for rs.Next() {
		var h deedHandoverView
		if err := rs.Scan(&h.To, &h.At, &h.Note, &h.By); err == nil {
			out = append(out, h)
		}
	}
	return out
}

func sha256Of(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func deedSeal(deedID int64) string {
	s, _ := verifyCode("DEED", deedID)
	return s
}

type deedSig struct {
	Signer, Hash, Seal, At, Note string
}

func deedSigs(db *sql.DB, deedID int64) []deedSig {
	var out []deedSig
	if db == nil {
		return out
	}
	var officeID int64
	if err := db.QueryRow(`SELECT office_id FROM deed_drafts WHERE id=?`, deedID).Scan(&officeID); err != nil {
		return out
	}
	rs, err := db.Query(`SELECT s.signer_name, s.content_hash, s.seal_code, s.created_at, s.note
		FROM deed_signatures s JOIN deed_drafts d ON d.id=s.deed_id
		WHERE s.deed_id=? AND d.office_id=? AND s.office_id=? ORDER BY s.id`, deedID, officeID, officeID)
	if err != nil {
		return out
	}
	defer rs.Close()
	for rs.Next() {
		var s deedSig
		_ = rs.Scan(&s.Signer, &s.Hash, &s.Seal, &s.At, &s.Note)
		out = append(out, s)
	}
	return out
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
	out := pdf.Export(title, safehtml.ToText(body), draft)
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
	out := docx.Export(title, safehtml.ToText(body), draft)
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

// protocolStatus records location/status/holder movements (in/out/return audit trail).
func (a *App) protocolStatus(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewProtocol) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	to := strings.ToUpper(strings.TrimSpace(r.FormValue("status")))
	switch to {
	case "IN_STORAGE", "OUT", "RETURNED":
	default:
		http.Error(w, "status tidak dikenal", http.StatusBadRequest)
		return
	}
	var loc any
	if v := strings.TrimSpace(r.FormValue("location_id")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			var nloc int
			_ = a.db.QueryRow(`SELECT COUNT(*) FROM doc_archive_locations WHERE id=? AND office_id=?`, n, a.officeOf(r)).Scan(&nloc)
			if nloc == 0 {
				http.Error(w, "lokasi tidak dikenal", http.StatusBadRequest)
				return
			}
			loc = n
		}
	}
	holder := strings.TrimSpace(r.FormValue("holder"))
	res, err := a.db.Exec(`UPDATE protocols SET status=?, location_id=COALESCE(?,location_id), holder=NULLIF(?,''), handover_note=CONCAT(COALESCE(handover_note,''),'\n',?) WHERE id=? AND office_id=?`,
		to, loc, holder, time.Now().Format("2006-01-02 15:04")+" "+to+" "+holder, id, a.officeOf(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		http.NotFound(w, r)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "protocol.move", Entity: "protocol", EntityID: strconv.FormatInt(id, 10), Detail: to + " " + holder})
	http.Redirect(w, r, "/protocols?flash=Status+protokol+diperbarui", http.StatusSeeOther)
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
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
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
	if _, err := a.clients.Get(a.officeOf(r), clientID); err != nil {
		http.Error(w, "klien tidak ditemukan di kantor ini", http.StatusBadRequest)
		return
	}
	id, err := a.wills.Register(a.officeOf(r), clientID, nil, r.FormValue("register_no"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "will.register", Entity: "will", EntityID: strconv.FormatInt(id, 10), OfficeID: officePtr(a.officeOf(r))})
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
	if err := a.wills.Handover(a.officeOf(r), id, r.FormValue("handed_to"), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "will.handover", Entity: "will", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("handed_to"), OfficeID: officePtr(a.officeOf(r))})
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
	// Wasiat terbatas: isi akta berjenis wasiat hanya untuk peran notaris
	// (+auditor baca), meniru gerbang willsPage. Akta non-wasiat tidak berubah.
	if d.Kind == "Will" && !willReader(mw.RoleOf(r)) {
		a.forbidden(w, r)
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
		LEFT JOIN doc_templates t ON t.id=d.template_id WHERE d.id=? AND d.office_id=?`, id, a.officeOf(r)).
		Scan(&pr.Created, &pr.Creator, &pr.ApprovedBy, &pr.ApprovedAt, &pr.Hash, &pr.Template)
	// Komparasi: vN vs vN-1 dari riwayat tersimpan; draf berjalan = versi terbaru.
	// v1 draf lama mungkin tak tersimpan (dibuat sebelum snapshot otomatis) → tulis jujur.
	var cmpA, cmpB string
	var cmpN int
	var diffOps []diff.Op
	var diffAdd, diffDel int
	if c := r.URL.Query().Get("compare"); c != "" {
		fmt.Sscanf(c, "%d", &cmpN)
		contents := map[int]string{}
		for _, v := range vers {
			contents[v.No] = safehtml.ToText(v.Content)
		}
		contents[d.Version] = safehtml.ToText(body)
		if aStr, ok := contents[cmpN]; ok {
			cmpA = aStr
			if bStr, ok2 := contents[cmpN-1]; ok2 {
				cmpB = bStr
			} else {
				cmpB = "(versi awal draf lama tidak tersimpan — perbandingan dimulai dari versi tercatat pertama)"
			}
			diffOps = diff.Compare(cmpB, cmpA)
			diffAdd, diffDel = diff.Stats(diffOps)
		}
	}
	// 3-way: template sebagai kolom ketiga (isi mentah bervariabel — pembanding struktur).
	var tplBody string
	var tplID sql.NullInt64
	_ = a.db.QueryRow(`SELECT template_id FROM deed_drafts WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&tplID)
	if tplID.Valid {
		_ = a.db.QueryRow(`SELECT body FROM doc_templates WHERE id=? AND office_id=?`, tplID.Int64, a.officeOf(r)).Scan(&tplBody)
	}
	// Jadwal pembacaan (lunak, non-blokir): janji temu tertaut via request_id
	// akta — appointments tak punya kolom deed_id (migrasi 002), jadi relasi
	// via deed.request_id. Tak pernah memblokir approve, hanya data + teks.
	reqID := deedRequestID(a.db, id)
	readings, hasDone := a.deedReadings(a.officeOf(r), reqID)
	a.render(w, r, "deed_detail", "Detail Akta #"+strconv.FormatInt(id, 10), map[string]any{
		"Deed": d, "Body": body, "Versions": vers, "Clauses": clauses, "Props": pr,
		"CmpA": cmpA, "CmpB": cmpB, "CmpN": cmpN, "DiffOps": diffOps, "DiffAdd": diffAdd, "DiffDel": diffDel,
		"TplBody":    safehtml.ToText(tplBody),
		"CanApprove": rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed),
		"CanSign":    rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) && d.Status == "APPROVED",
		"CanArchive": rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) && d.Status == "SIGNED",
		"Seal":       deedSeal(id), "Sigs": deedSigs(a.db, id),
		"Timeline": a.deedTimeline(a.officeOf(r), id),
		"Handovers":  a.deedHandovers(a.officeOf(r), id),
		"CanHandover": rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed),
		"Attached": a.deedFiles(a.officeOf(r), id),
		"CanEdit":  rbac.Has(mw.RoleOf(r), rbac.PermEditDeed),
		"Parties":  a.matterParties(a.officeOf(r), reqID),
		"Readings": readings, "HasReadingDone": hasDone,
	})
}

// deedRequestID mengambil request_id draf (0 bila tanpa tautan permohonan).
func deedRequestID(db *sql.DB, deedID int64) int64 {
	if db == nil {
		return 0
	}
	var req sql.NullInt64
	_ = db.QueryRow(`SELECT request_id FROM deed_drafts WHERE id=?`, deedID).Scan(&req)
	if req.Valid {
		return req.Int64
	}
	return 0
}

// matterParties memuat daftar pihak matter (read-only untuk deedDetail).
func (a *App) matterParties(officeID, reqID int64) []partyView {
	var out []partyView
	if a.db == nil || reqID == 0 {
		return out
	}
	rs, err := a.db.Query(`SELECT mp.id, mp.client_id, c.full_name, mp.role
		FROM matter_parties mp JOIN clients c ON c.id=mp.client_id
		WHERE mp.office_id=? AND mp.request_id=? ORDER BY mp.id`, officeID, reqID)
	if err != nil {
		return out
	}
	defer rs.Close()
	for rs.Next() {
		var p partyView
		if err := rs.Scan(&p.ID, &p.ClientID, &p.ClientName, &p.Role); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// deedReadings memuat janji temu tertaut via request_id draf (read-only untuk
// deedDetail) + flag lunak apakah sudah ada pembacaan DONE. appointments tak
// punya kolom deed_id (migrasi 002) sehingga relasi via deed.request_id.
// Tak pernah memblokir approve — hanya data list + teks peringatan di kartu.
func (a *App) deedReadings(officeID, reqID int64) ([]map[string]any, bool) {
	var out []map[string]any
	if a.db == nil || reqID == 0 {
		return out, false
	}
	rows, err := a.db.Query(`SELECT id, scheduled_at, purpose, status FROM appointments
		WHERE office_id=? AND request_id=? ORDER BY scheduled_at`, officeID, reqID)
	if err != nil {
		return out, false
	}
	defer rows.Close()
	hasDone := false
	for rows.Next() {
		var id int64
		var at, purpose, status string
		if err := rows.Scan(&id, &at, &purpose, &status); err == nil {
			if status == "DONE" {
				hasDone = true
			}
			out = append(out, map[string]any{"ID": id, "At": at, "Purpose": purpose, "Status": status})
		}
	}
	return out, hasDone
}

// deedFiles lists DMS files linked to a deed (lampiran dua arah).
func (a *App) deedFiles(officeID, deedID int64) []map[string]any {
	fids, _ := a.dms.ByRelated(officeID, "deed", deedID)
	var out []map[string]any
	for _, fid := range fids {
		var name string
		_ = a.db.QueryRow(`SELECT COALESCE(display_name,original_name) FROM files WHERE id=? AND office_id=?`, fid, officeID).Scan(&name)
		if name != "" {
			out = append(out, map[string]any{"ID": fid, "Name": name})
		}
	}
	return out
}

// deedTimeline unions versions, approval, and TTE signatures chronologically.
// officeID scopes every sub-query so a deed id from another office yields nothing.
func (a *App) deedTimeline(officeID, deedID int64) []map[string]any {
	type ev struct {
		at, kind, text string
	}
	var evs []ev
	rows, err := a.db.Query(`SELECT v.version, v.note, v.created_at FROM deed_versions v
		JOIN deed_drafts d ON d.id=v.deed_id WHERE v.deed_id=? AND d.office_id=? ORDER BY v.id`, deedID, officeID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v int
			var note, at string
			_ = rows.Scan(&v, &note, &at)
			evs = append(evs, ev{at, "Revisi", "v" + strconv.Itoa(v) + " — " + note})
		}
	}
	var approvedBy, approvedAt, number string
	_ = a.db.QueryRow(`SELECT COALESCE(u.name,''), COALESCE(d.approved_at,''), COALESCE(d.number,'')
		FROM deed_drafts d LEFT JOIN users u ON u.id=d.approved_by WHERE d.id=? AND d.office_id=?`, deedID, officeID).
		Scan(&approvedBy, &approvedAt, &number)
	if approvedAt != "" {
		evs = append(evs, ev{approvedAt, "Pengesahan", "No. " + number + " oleh " + approvedBy})
	}
	srows, _ := a.db.Query(`SELECT signer_name, created_at FROM deed_signatures WHERE deed_id=? AND office_id=? ORDER BY id`, deedID, officeID)
	if srows != nil {
		defer srows.Close()
		for srows.Next() {
			var who, at string
			_ = srows.Scan(&who, &at)
			evs = append(evs, ev{at, "TTE internal", "ditandatangani " + who})
		}
	}
	var out []map[string]any
	for _, e := range evs {
		out = append(out, map[string]any{"At": e.at, "Kind": e.kind, "Text": e.text})
	}
	return out
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
	if err := a.deeds.SaveVersion(id, safehtml.Sanitize(r.FormValue("content")), a.uid(r), r.FormValue("note")); err != nil {
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
	if !rbac.Has(mw.RoleOf(r), rbac.PermAuditView) {
		a.forbidden(w, r)
		return
	}
	rows, err := a.db.Query(`SELECT a.id, u.name, a.action, a.entity, a.entity_id, a.created_at
		FROM audit_logs a LEFT JOIN users u ON u.id=a.actor_id
		WHERE (u.office_id=? OR a.actor_id IS NULL)
		ORDER BY a.id DESC LIMIT 200`, a.officeOf(r))
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

// auditExport downloads the last 1000 audit rows as CSV (same guard as auditPage).
// Optional q filters server-side on action/entity/entity_id/actor.
func (a *App) auditExport(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermAuditView) {
		a.forbidden(w, r)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	where := ` WHERE (u.office_id=? OR a.actor_id IS NULL)`
	args := []any{a.officeOf(r)}
	if q != "" {
		where += ` AND (a.action LIKE ? OR a.entity LIKE ? OR a.entity_id LIKE ? OR u.name LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like, like, like)
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"audit.csv\"")
	w.Write([]byte("\xef\xbb\xbf"))
	fmt.Fprintln(w, "waktu,aktor,aksi,entitas,id_entitas,detail")
	rs, err := a.db.Query(`SELECT a.created_at, COALESCE(u.name,''), a.action, a.entity, COALESCE(a.entity_id,''), COALESCE(a.detail,'')
		FROM audit_logs a LEFT JOIN users u ON u.id=a.actor_id`+where+` ORDER BY a.id DESC LIMIT 1000`, args...)
	if err != nil {
		return
	}
	defer rs.Close()
	esc := func(s string) string { return strings.ReplaceAll(s, `"`, `""`) }
	for rs.Next() {
		var at, actor, action, entity, entityID, detail string
		_ = rs.Scan(&at, &actor, &action, &entity, &entityID, &detail)
		fmt.Fprintf(w, "%s,\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n",
			at, esc(actor), esc(action), esc(entity), esc(entityID), esc(detail))
	}
}

// ---- office profile ----

func (a *App) officePage(w http.ResponseWriter, r *http.Request) {
	var name, notary, sk, addr, phone, email, plan string
	_ = a.db.QueryRow(`SELECT o.name, o.notary_name, o.sk_number, o.address, o.phone, o.email, o.plan
		FROM offices o JOIN users u ON u.office_id=o.id WHERE u.id=?`, *a.uid(r)).Scan(&name, &notary, &sk, &addr, &phone, &email, &plan)
	officeID := a.officeOf(r)
	var strictInt int
	_ = a.db.QueryRow(`SELECT strict_checklist FROM office_flags WHERE office_id=?`, officeID).Scan(&strictInt)
	var slaJSON sql.NullString
	_ = a.db.QueryRow(`SELECT sla_days_json FROM office_flags WHERE office_id=?`, officeID).Scan(&slaJSON)
	// Kartu aktivasi kantor (Go-Live): 5 langkah office-scoped, fail-open 0
	// bila tabel belum termigrasi agar halaman profil tidak pernah rusak.
	countOf := func(q string, args ...any) int {
		var n int
		if err := a.db.QueryRow(q, args...).Scan(&n); err != nil {
			return 0
		}
		return n
	}
	checklistN := countOf(`SELECT COUNT(*) FROM doc_checklist_templates WHERE office_id=?`, officeID)
	docTplN := countOf(`SELECT COUNT(*) FROM doc_templates WHERE office_id=?`, officeID)
	docApprN := countOf(`SELECT COUNT(*) FROM doc_templates WHERE office_id=? AND approved=1`, officeID)
	invN := countOf(`SELECT COUNT(*) FROM invoices WHERE office_id=?`, officeID)
	reqN := countOf(`SELECT COUNT(*) FROM service_requests WHERE office_id=?`, officeID)
	flagN := countOf(`SELECT COUNT(*) FROM office_flags WHERE office_id=?`, officeID)
	hasChecklist := checklistN > 0
	hasDocTpl := docTplN > 0
	hasDocAppr := docApprN > 0
	templatesOK := hasDocTpl && hasDocAppr
	hasTrial := invN > 0 || reqN > 0
	hasRequest := reqN > 0
	strictKnown := flagN > 0
	done := 0
	for _, ok := range []bool{hasChecklist, templatesOK, hasTrial, hasRequest, strictKnown} {
		if ok {
			done++
		}
	}
	activation := map[string]any{
		"Checklist": hasChecklist, "DocTemplate": hasDocTpl, "DocApproved": hasDocAppr,
		"TemplatesOK": templatesOK, "Trial": hasTrial, "HasRequest": hasRequest,
		"StrictKnown": strictKnown, "Done": done, "Total": 5,
	}
	a.render(w, r, "office", "Profil Kantor", map[string]any{
		"Name": name, "Notary": notary, "SK": sk, "Address": addr, "Phone": phone, "Email": email, "Plan": plan,
		"Strict": strictInt == 1, "SLAJSON": slaJSON.String, "Activation": activation,
		"LegacyPlaintext": countLegacyPlaintextBackups("storage/backups"),
	})
}

// strictChecklistEnabled reports whether the office enforces the strict
// checklist gate. Fail-open (false) on any error so old DBs without the
// office_flags table keep the legacy behavior.
func (a *App) strictChecklistEnabled(officeID int64) bool {
	if a.db == nil {
		return false
	}
	var v int
	if err := a.db.QueryRow(`SELECT strict_checklist FROM office_flags WHERE office_id=?`, officeID).Scan(&v); err != nil {
		return false
	}
	return v == 1
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
	// Toggle mode ketat checklist + override SLA per layanan — keduanya
	// ditulis ke office_flags. sla_days_json harus objek JSON valid
	// (mis. {"PPAT-AJB": 5, "default": 10}); JSON rusak ditolak 400.
	strict := 0
	if v := r.FormValue("strict_checklist"); v == "1" || v == "on" {
		strict = 1
	}
	slaRaw := strings.TrimSpace(r.FormValue("sla_days_json"))
	if _, err := remind.ParseSLAOverrides(slaRaw); err != nil {
		http.Error(w, "sla_days_json tidak valid: "+err.Error()+` — contoh valid: {"PPAT-AJB": 5, "default": 10}`, http.StatusBadRequest)
		return
	}
	var slaVal any
	if slaRaw == "" {
		slaVal = nil
	} else {
		slaVal = slaRaw
	}
	// Best-effort untuk tabel yang belum termigrasi: bila INSERT ... SELECT
	// sla_days_json gagal (kolom/tabel belum ada), fallback ke pernyataan
	// lama yang hanya menulis strict_checklist agar simpan profil kantor
	// tidak pernah rusak oleh flag ini.
	if _, err := a.db.Exec(`INSERT INTO office_flags(office_id, strict_checklist, sla_days_json) VALUES(?,?,?)
		ON DUPLICATE KEY UPDATE strict_checklist=VALUES(strict_checklist), sla_days_json=VALUES(sla_days_json)`, a.officeOf(r), strict, slaVal); err != nil {
		_, _ = a.db.Exec(`INSERT INTO office_flags(office_id, strict_checklist) VALUES(?,?)
		ON DUPLICATE KEY UPDATE strict_checklist=VALUES(strict_checklist)`, a.officeOf(r), strict)
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "office.update", Entity: "office", EntityID: "self"})
	http.Redirect(w, r, "/office?flash=Profil+kantor+disimpan", http.StatusSeeOther)
}

// licensePage shows the standalone install license status (non-enforcing).
func (a *App) licensePage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageOffice) {
		a.forbidden(w, r)
		return
	}
	var key sql.NullString
	var exp sql.NullString
	_ = a.db.QueryRow(`SELECT license_key, DATE_FORMAT(license_expires,'%Y-%m-%d') FROM offices WHERE id=?`, a.officeOf(r)).Scan(&key, &exp)
	status := "Belum ada lisensi — mode evaluasi internal."
	if key.Valid && key.String != "" {
		status = "Lisensi tercatat."
	}
	a.render(w, r, "license", "Lisensi Instalasi", map[string]string{
		"Key": key.String, "Expires": exp.String, "Status": status,
	})
}

// licenseSave records the license (informational only — never locks the office out).
func (a *App) licenseSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageOffice) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	key := strings.TrimSpace(r.FormValue("key"))
	exp := strings.TrimSpace(r.FormValue("expires"))
	if _, err := a.db.Exec(`UPDATE offices SET license_key=?, license_expires=NULLIF(?,'') WHERE id=?`, key, exp, a.officeOf(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "office.license", Entity: "office", EntityID: "self"})
	http.Redirect(w, r, "/office/license?flash=Lisensi+dicatat", http.StatusSeeOther)
}

// ---- tasks ----

func (a *App) tasksPage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	rows, _ := a.db.Query(`SELECT id, title, due_date, status, recurrence, ref_type, COALESCE(ref_id,0) FROM tasks WHERE office_id=? ORDER BY status, due_date LIMIT 200`, a.officeOf(r))
	type task struct {
		ID                 int64
		Title, Due, Status string
		Rec, RefType       string
		RefID              int64
		DeedTitle          string
		DeedStatus         string
		DeedVersion        int
		DeedPreview        string
		DeedLocked         bool
	}
	var out []task
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t task
			var due sql.NullString
			_ = rows.Scan(&t.ID, &t.Title, &due, &t.Status, &t.Rec, &t.RefType, &t.RefID)
			if due.Valid {
				t.Due = due.String
			}
			out = append(out, t)
		}
	}
	// Hydrate deed drafts untuk preview + edit cepat langsung dari Todo.
	for i := range out {
		if out[i].RefType != "deed" || out[i].RefID <= 0 {
			continue
		}
		var title, status, content sql.NullString
		var ver sql.NullInt64
		var locked sql.NullBool
		_ = a.db.QueryRow(`SELECT title, status, version, LEFT(content,2000), locked FROM deed_drafts WHERE id=? AND office_id=?`,
			out[i].RefID, a.officeOf(r)).Scan(&title, &status, &ver, &content, &locked)
		if title.Valid {
			out[i].DeedTitle = title.String
			out[i].DeedStatus = status.String
			out[i].DeedPreview = safehtml.ToText(content.String)
			if ver.Valid {
				out[i].DeedVersion = int(ver.Int64)
			}
			out[i].DeedLocked = locked.Valid && locked.Bool
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
	rec := strings.ToUpper(strings.TrimSpace(r.FormValue("recurrence")))
	switch rec {
	case "DAILY", "WEEKLY", "MONTHLY":
	default:
		rec = "NONE"
	}
	refType := strings.ToLower(strings.TrimSpace(r.FormValue("ref_type")))
	switch refType {
	case "deed", "request", "client", "file":
	default:
		refType = ""
	}
	var refID any
	if refType != "" {
		if n, err := strconv.ParseInt(r.FormValue("ref_id"), 10, 64); err == nil && n > 0 {
			refID = n
		} else {
			refType = ""
		}
	}
	_, err := a.db.Exec(`INSERT INTO tasks(office_id, title, due_date, created_by, recurrence, ref_type, ref_id) VALUES(?,?,?,?,?,?,?)`,
		a.officeOf(r), r.FormValue("title"), due, a.uid(r), rec, nullStrOpt(refType), refID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/tasks?flash=Tugas+ditambah", http.StatusSeeOther)
}

func (a *App) taskDone(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var title, rec, due string
	var refType sql.NullString
	var refID sql.NullInt64
	_ = a.db.QueryRow(`SELECT title, recurrence, DATE_FORMAT(due_date,'%Y-%m-%d'), ref_type, ref_id FROM tasks WHERE id=? AND office_id=?`,
		id, a.officeOf(r)).Scan(&title, &rec, &due, &refType, &refID)
	_, _ = a.db.Exec(`UPDATE tasks SET status='DONE' WHERE id=? AND office_id=?`, id, a.officeOf(r))
	// Tugas berulang: jadwalkan periode berikut dari tenggat lama (atau hari ini bila kosong).
	if rec == "DAILY" || rec == "WEEKLY" || rec == "MONTHLY" {
		base := time.Now()
		if t, err := time.Parse("2006-01-02", due); err == nil {
			base = t
		}
		switch rec {
		case "DAILY":
			base = base.AddDate(0, 0, 1)
		case "WEEKLY":
			base = base.AddDate(0, 0, 7)
		case "MONTHLY":
			base = base.AddDate(0, 1, 0)
		}
		var rt, ri any
		if refType.Valid && refType.String != "" {
			rt, ri = refType.String, refID.Int64
		}
		_, _ = a.db.Exec(`INSERT INTO tasks(office_id, title, due_date, created_by, recurrence, ref_type, ref_id) VALUES(?,?,?,?,?,?,?)`,
			a.officeOf(r), title, base.Format("2006-01-02"), a.uid(r), rec, rt, ri)
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "task.recur", Entity: "task", EntityID: strconv.FormatInt(id, 10), Detail: rec})
	}
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
	amount, err := money.Parse(r.FormValue("amount"))
	if err != nil || amount <= 0 || r.FormValue("description") == "" {
		http.Error(w, "nominal dan uraian wajib diisi", http.StatusBadRequest)
		return
	}
	spent := r.FormValue("spent_at")
	if spent == "" {
		spent = time.Now().Format("2006-01-02")
	}
	_, err = a.db.Exec(`INSERT INTO expenses(office_id, category, description, amount, spent_at, created_by)
		VALUES(?,?,?,?,?,?)`, a.officeOf(r), r.FormValue("category"), r.FormValue("description"), amount, spent, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "expense.create", Entity: "expense", EntityID: r.FormValue("description")})
	http.Redirect(w, r, "/expenses?flash=Pengeluaran+dicatat", http.StatusSeeOther)
}

func (a *App) tariffsPage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	rows, _ := a.db.Query(`SELECT id, name, amount, is_active FROM price_items ORDER BY name LIMIT 200`)
	type tariff struct {
		ID     int64
		Name   string
		Amount int64
		Active bool
	}
	var out []tariff
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t tariff
			var samt string
			_ = rows.Scan(&t.ID, &t.Name, &samt, &t.Active)
			t.Amount, _ = money.Parse(samt)
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
	amount, err := money.Parse(r.FormValue("amount"))
	if r.FormValue("name") == "" || err != nil || amount < 0 {
		http.Error(w, "nama dan tarif tidak valid", http.StatusBadRequest)
		return
	}
	var sid any
	if r.FormValue("service_id") != "" {
		sid = r.FormValue("service_id")
	}
	_, err = a.db.Exec(`INSERT INTO price_items(service_id, name, amount) VALUES(?,?,?)`, sid, r.FormValue("name"), amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/tariffs?flash=Tarif+disimpan", http.StatusSeeOther)
}

// ---- global search ----

func (a *App) search(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
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
		rows3, _ := a.db.Query(`SELECT akta_no, jenis, COALESCE(penjual,''), COALESCE(pembeli,''), COALESCE(objek_sertifikat,''), status FROM ppat_register WHERE office_id=? AND (akta_no LIKE CONCAT('%',?,'%') OR penjual LIKE CONCAT('%',?,'%') OR pembeli LIKE CONCAT('%',?,'%') OR objek_sertifikat LIKE CONCAT('%',?,'%')) ORDER BY id DESC LIMIT 10`, a.officeOf(r), q, q, q, q)
		if rows3 != nil {
			defer rows3.Close()
			for rows3.Next() {
				var no, jenis, penjual, pembeli, sert, status string
				_ = rows3.Scan(&no, &jenis, &penjual, &pembeli, &sert, &status)
				title := "Akta " + no + " (" + jenis + ")"
				parties := strings.Trim(penjual+" → "+pembeli, " →")
				if parties != "" {
					title += " — " + parties
				}
				if sert != "" {
					title += " [" + sert + "]"
				}
				out = append(out, hit{"PPAT", title + " (" + status + ")", "/ppat"})
			}
		}
		rows4, _ := a.db.Query(`SELECT id, title, status FROM tasks WHERE office_id=? AND (title LIKE CONCAT('%',?,'%')) ORDER BY id DESC LIMIT 10`, a.officeOf(r), q)
		if rows4 != nil {
			defer rows4.Close()
			for rows4.Next() {
				var id int64
				var title, status string
				_ = rows4.Scan(&id, &title, &status)
				out = append(out, hit{"Tugas", title + " (" + status + ")", "/tasks"})
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
