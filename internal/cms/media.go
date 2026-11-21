package cms

import (
	"bytes"
	"database/sql"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Reencode decodes jpeg/png/gif and re-encodes (strips EXIF/scripts), returning
// safe bytes + dimensions. webp/svg/pdf pass through untouched (validated).
func Reencode(mime string, raw []byte) ([]byte, int, int, error) {
	switch mime {
	case "image/jpeg", "image/png", "image/gif":
	default:
		return raw, 0, 0, nil
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, errors.New("gambar rusak/tak terdecode")
	}
	b := img.Bounds()
	if mime == "image/png" {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, 0, 0, err
		}
		return buf.Bytes(), b.Dx(), b.Dy(), nil
	}
	// jpeg/gif → png aman (tanpa metadata aktif).
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

// Media is one public library record (bytes live in storage under cms/{office}/).
type Media struct {
	ID                       int64
	Ref                    string
	OriginalName, MIME       string
	Size                   int64
	Width, Height           int
	Alt, Caption, Folder    string
	CreatedBy               sql.NullInt64
	CreatedAt               time.Time
}

var mediaExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".pdf": true, ".svg": true,
}

var mediaMIMEs = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
	"image/svg+xml": true, "application/pdf": true,
}

// ValidateMedia sniffs content, checks extension+MIME coherence and size.
func ValidateMedia(filename string, size int64, head []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if !mediaExts[ext] {
		return "", errors.New("ekstensi tidak diizinkan")
	}
	if size <= 0 || size > 10<<20 {
		return "", errors.New("ukuran 1 byte–10 MB")
	}
	sniffed := http.DetectContentType(head)
	if sniffed == "image/svg+xml" {
		s := strings.ToLower(string(head))
		if strings.Contains(s, "<script") || strings.Contains(s, "onload=") || strings.Contains(s, "javascript:") {
			return "", errors.New("SVG aktif ditolak")
		}
	}
	mime, _, _ := strings.Cut(sniffed, ";")
	if !mediaMIMEs[mime] {
		return "", errors.New("tipe berkas tidak diizinkan: " + mime)
	}
	// Extension-MIME coherence (anti spoofing).
	switch ext {
	case ".jpg", ".jpeg":
		if mime != "image/jpeg" {
			return "", errors.New("ekstensi/MIME tidak cocok")
		}
	case ".png":
		if mime != "image/png" {
			return "", errors.New("ekstensi/MIME tidak cocok")
		}
	case ".gif":
		if mime != "image/gif" {
			return "", errors.New("ekstensi/MIME tidak cocok")
		}
	case ".pdf":
		if mime != "application/pdf" {
			return "", errors.New("ekstensi/MIME tidak cocok")
		}
	}
	return mime, nil
}

// MediaRef builds the randomized storage reference (never the original name).
func MediaRef(orig string, now time.Time, rand string) string {
	ext := strings.ToLower(filepath.Ext(orig))
	return "cms/" + now.UTC().Format("2006/01") + "/" + rand + ext
}

// SaveMedia records the library row (bytes must already be stored via storage).
func (s *Store) SaveMedia(officeID int64, m *Media, actorID *int64) (int64, error) {
	if m.Ref == "" || m.MIME == "" {
		return 0, errors.New("ref dan MIME wajib")
	}
	res, err := s.db.Exec(`INSERT INTO website_media(office_id, ref, original_name, mime,
		size_bytes, width, height, alt, caption, folder, created_by)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`, officeID, m.Ref, m.OriginalName, m.MIME,
		m.Size, m.Width, m.Height, m.Alt, m.Caption, folderOf(m.Folder), nullInt64(actorID))
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	s.Invalidate(officeID)
	return id, nil
}

func folderOf(f string) string {
	f = strings.ToLower(strings.TrimSpace(f))
	if f == "" {
		return "umum"
	}
	for _, ch := range f {
		if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
			return "umum"
		}
	}
	if len(f) > 40 {
		return "umum"
	}
	return f
}

// ListMedia searches/filters with pagination (admin).
func (s *Store) ListMedia(officeID int64, q, folder, mime string, limit, offset int) ([]Media, int, error) {
	where := `office_id=?`
	args := []any{officeID}
	if folder != "" {
		where += ` AND folder=?`
		args = append(args, folder)
	}
	if mime != "" {
		where += ` AND mime=?`
		args = append(args, mime)
	}
	if strings.TrimSpace(q) != "" {
		where += ` AND (original_name LIKE ? OR alt LIKE ? OR caption LIKE ?)`
		like := "%" + strings.TrimSpace(q) + "%"
		args = append(args, like, like, like)
	}
	var total int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM website_media WHERE `+where, args...).Scan(&total)
	rows, err := s.db.Query(`SELECT id, ref, original_name, mime, size_bytes, width, height,
		alt, caption, folder, created_by, created_at FROM website_media WHERE `+where+`
		ORDER BY id DESC LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Media
	for rows.Next() {
		var m Media
		_ = rows.Scan(&m.ID, &m.Ref, &m.OriginalName, &m.MIME, &m.Size, &m.Width,
			&m.Height, &m.Alt, &m.Caption, &m.Folder, &m.CreatedBy, &m.CreatedAt)
		out = append(out, m)
	}
	return out, total, rows.Err()
}

// DeleteMedia removes the row (and caller deletes bytes) unless in use.
func (s *Store) DeleteMedia(officeID, id int64) (string, error) {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM website_media_usage WHERE media_id=?`, id).Scan(&n)
	if n > 0 {
		return "", errors.New("media masih dipakai konten — lepas dulu")
	}
	var ref string
	if err := s.db.QueryRow(`SELECT ref FROM website_media WHERE id=? AND office_id=?`,
		id, officeID).Scan(&ref); err != nil {
		return "", errors.New("media tidak ditemukan")
	}
	if _, err := s.db.Exec(`DELETE FROM website_media WHERE id=? AND office_id=?`, id, officeID); err != nil {
		return "", err
	}
	s.Invalidate(officeID)
	return ref, nil
}

// TrackUsage records where a media item is used (blocks blind deletes).
func (s *Store) TrackUsage(mediaID int64, usedIn string, usedID int64) {
	switch usedIn {
	case "page", "article", "service", "section":
	default:
		return
	}
	_, _ = s.db.Exec(`INSERT IGNORE INTO website_media_usage(media_id, used_in, used_id) VALUES(?,?,?)`,
		mediaID, usedIn, usedID)
}

var _ = mime.TypeByExtension
