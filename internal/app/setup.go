package app

// Setup wizard awal (satu-kali): membuat kantor + admin super_admin pertama.
// Guard keras: aktif HANYA bila tabel users kosong (0 baris). Begitu ada
// >=1 user, GET maupun POST dialihkan ke /login sehingga endpoint tidak
// bisa dipakai untuk ambil-alih.

import (
	"net/http"
	"strconv"
	"strings"

	"notaryos/internal/audit"
	"notaryos/internal/auth"
	"notaryos/internal/rbac"
	"notaryos/internal/validate"
)

// setupNeeded reports whether initial setup is still allowed: true only
// when the users table exists and holds zero rows. Fail-closed (false)
// on nil DB or any query error.
func (a *App) setupNeeded() bool {
	if a.db == nil {
		return false
	}
	var n int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return false
	}
	return n == 0
}

// setupForm renders the one-time setup form. Disabled once any user exists.
func (a *App) setupForm(w http.ResponseWriter, r *http.Request) {
	if !a.setupNeeded() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	a.render(w, r, "setup", "Pengaturan Awal — SaktiNotaris", nil)
}

// setupSubmit validates the form and creates the first office + super_admin.
// Guard ganda: cek sebelum validasi dan lagi di dalam transaksi (anti-race).
func (a *App) setupSubmit(w http.ResponseWriter, r *http.Request) {
	if !a.setupNeeded() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "form tidak valid", http.StatusBadRequest)
		return
	}
	office := strings.TrimSpace(r.FormValue("office_name"))
	notary := strings.TrimSpace(r.FormValue("notary_name"))
	admin := strings.TrimSpace(r.FormValue("admin_name"))
	email := strings.TrimSpace(r.FormValue("admin_email"))
	pw := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	keep := map[string]string{
		"OfficeName": office, "NotaryName": notary,
		"AdminName": admin, "AdminEmail": email,
	}
	fail := func(msg string) {
		keep["Error"] = msg
		a.render(w, r, "setup", "Pengaturan Awal — SaktiNotaris", keep)
	}

	if err := validate.Required("nama kantor", office); err != nil {
		fail("nama kantor wajib diisi")
		return
	}
	if err := validate.Required("nama notaris", notary); err != nil {
		fail("nama notaris wajib diisi")
		return
	}
	if err := validate.Required("nama admin", admin); err != nil {
		fail("nama admin wajib diisi")
		return
	}
	if err := validate.Email(email); err != nil {
		fail("email admin tidak valid")
		return
	}
	if len(pw) < 12 {
		fail("kata sandi minimal 12 karakter")
		return
	}
	if pw != confirm {
		fail("konfirmasi kata sandi tidak cocok")
		return
	}
	var taken int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM users WHERE email=?`, email).Scan(&taken); err != nil {
		http.Error(w, "gagal memeriksa email", http.StatusInternalServerError)
		return
	}
	if taken > 0 {
		fail("email sudah terdaftar")
		return
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		fail("kata sandi tidak memenuhi syarat")
		return
	}

	tx, err := a.db.Begin()
	if err != nil {
		http.Error(w, "gagal memulai transaksi", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	// Guard ulang di dalam transaksi: tolak bila user sudah ada (anti-race).
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil || n > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var roleID int64
	if err := tx.QueryRow(`SELECT id FROM roles WHERE name=?`, rbac.RoleSuperAdmin).Scan(&roleID); err != nil {
		http.Error(w, "role super_admin belum tersedia (jalankan migrasi)", http.StatusInternalServerError)
		return
	}
	// Kolom offices NOT NULL yang tak ditanya di form diisi "-" agar bisa
	// dilengkapi kemudian di /office; email kantor = email admin.
	res, err := tx.Exec(`INSERT INTO offices(name, notary_name, sk_number, address, phone, email)
		VALUES(?, ?, '-', '-', '-', ?)`, office, notary, email)
	if err != nil {
		http.Error(w, "gagal membuat kantor: "+err.Error(), http.StatusInternalServerError)
		return
	}
	officeID, err := res.LastInsertId()
	if err != nil {
		http.Error(w, "gagal membaca id kantor", http.StatusInternalServerError)
		return
	}
	ures, err := tx.Exec(`INSERT INTO users(office_id, role_id, name, email, password_hash, password_changed_at)
		VALUES(?, ?, ?, ?, ?, NOW())`, officeID, roleID, admin, email, hash)
	if err != nil {
		fail("gagal membuat admin: " + err.Error())
		return
	}
	userID, _ := ures.LastInsertId()
	if err := tx.Commit(); err != nil {
		http.Error(w, "gagal menyimpan pengaturan awal", http.StatusInternalServerError)
		return
	}
	// Audit bila memungkinkan (best-effort, tak pernah menggagalkan request).
	a.audit.Record(audit.Entry{ActorID: &userID, Action: "setup.complete",
		Entity: "office", EntityID: strconv.FormatInt(officeID, 10), Detail: email})
	http.Redirect(w, r, "/login?flash=Pengaturan+awal+selesai.+Silakan+masuk.", http.StatusSeeOther)
}
