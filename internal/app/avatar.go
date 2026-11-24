package app

// Foto profil pengguna: unggah tervalidasi + auto-crop persegi + serve office-scoped.
// Jujur: tanpa pustaka crop UI — crop persegi-otomatis (tengah) + re-encode
// (strip metadata) + allowlist raster; PDF/SVG ditolak untuk avatar.

import (
	"bytes"
	"database/sql"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"

	"notaryos/internal/audit"
	"notaryos/internal/cms"
	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
)

// avatarUpload mengganti foto profil milik sendiri (maks 10 MB, raster saja).
func (a *App) avatarUpload(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	uid := a.uid(r)
	if uid == nil || a.db == nil {
		http.Error(w, "sesi tidak valid", http.StatusUnauthorized)
		return
	}
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		http.Error(w, "berkas terlalu besar (maks 10 MB)", http.StatusBadRequest)
		return
	}
	f, hdr, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "pilih berkas foto", http.StatusBadRequest)
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (10<<20)+1))
	if err != nil || len(raw) == 0 || len(raw) > 10<<20 {
		http.Error(w, "ukuran tidak valid (maks 10 MB)", http.StatusBadRequest)
		return
	}
	head := raw
	if len(raw) > 512 {
		head = raw[:512]
	}
	mime, err := cms.ValidateMedia(hdr.Filename, int64(len(raw)), head)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch mime {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
	default:
		http.Error(w, "foto harus JPG/PNG/GIF/WebP", http.StatusBadRequest)
		return
	}
	// Auto-crop persegi tengah (best-effort; gagal decode = tanpa crop).
	cropped := raw
	cropName := hdr.Filename
	if sq, ok := squareCropPNG(raw); ok {
		cropped = sq
		cropName = "avatar.png"
	}
	head = cropped
	if len(cropped) > 512 {
		head = cropped[:512]
	}
	mime, err = cms.ValidateMedia(cropName, int64(len(cropped)), head)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	safe, _, _, err := cms.Reencode(mime, cropped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	office := a.officeOf(r)
	if a.overQuota(office, int64(len(safe))) {
		http.Error(w, "kuota penyimpanan kantor terlampaui — hubungi administrator", http.StatusInsufficientStorage)
		return
	}
	fid, err := a.uploads.SaveUpload(office, "user", *uid, cropName, mime, bytes.NewReader(safe), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := a.db.Exec(`UPDATE users SET avatar_ref=? WHERE id=? AND office_id=?`, fid, *uid, office); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: uid, Action: "user.avatar", Entity: "user", EntityID: strconv.FormatInt(*uid, 10), Detail: "file#" + strconv.FormatInt(fid, 10)})
	http.Redirect(w, r, "/settings?flash=Foto+profil+diperbarui", http.StatusSeeOther)
}

// squareCropPNG memotong bagian tengah terluas menjadi persegi (PNG).
// Gagal decode/encode = ok=false (panggil crop dilewati, bukan error fatal).
func squareCropPNG(raw []byte) ([]byte, bool) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, false
	}
	b := img.Bounds()
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}
	if side <= 0 {
		return nil, false
	}
	ox := b.Min.X + (b.Dx()-side)/2
	oy := b.Min.Y + (b.Dy()-side)/2
	sq := image.NewRGBA(image.Rect(0, 0, side, side))
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			sq.Set(x, y, img.At(ox+x, oy+y))
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, sq); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}

// avatarServe mengalirkan foto profil milik sendiri (office-scoped, privat).
func (a *App) avatarServe(w http.ResponseWriter, r *http.Request) {
	uid := a.uid(r)
	if uid == nil || a.db == nil {
		http.NotFound(w, r)
		return
	}
	office := a.officeOf(r)
	var ref sql.NullInt64
	if err := a.db.QueryRow(`SELECT avatar_ref FROM users WHERE id=? AND office_id=?`, *uid, office).Scan(&ref); err != nil || !ref.Valid || ref.Int64 <= 0 {
		http.NotFound(w, r)
		return
	}
	m, rc, err := a.uploads.Get(office, ref.Int64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", m.MIME)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = io.Copy(w, rc)
}
