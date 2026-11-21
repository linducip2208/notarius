package files

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"notaryos/internal/ocr"
	"notaryos/internal/storage"
)

// maxUploadBytes caps single uploads at 100MB (dibaca eksplisit, bukan truncate diam-diam).
const maxUploadBytes = 100 << 20

// allowed MIME types for uploads (no executables, ever).
var allowedMIME = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true,
	"application/pdf": true,
}

// Meta describes a stored private file.
type Meta struct {
	ID        int64
	OwnerType string
	OwnerID   int64
	Ref       string
	Name      string
	MIME      string
	Size      int64
	OCRStatus string
}

type Store struct {
	db    *sql.DB
	files storage.Store
	ocr   ocr.Provider
}

func NewStore(db *sql.DB, fs storage.Store, o ocr.Provider) *Store {
	return &Store{db: db, files: fs, ocr: o}
}

// SaveUpload validates, stores bytes privately, and records metadata + sha256.
func (s *Store) SaveUpload(officeID int64, ownerType string, ownerID int64, name, mime string, src io.Reader, createdBy *int64) (int64, error) {
	mime = strings.ToLower(strings.TrimSpace(mime))
	if !allowedMIME[mime] {
		return 0, errors.New("tipe file tidak diizinkan (hanya JPG/PNG/WebP/PDF)")
	}
	ext := filepath.Ext(name)
	ref := "offices/" + strconv.FormatInt(officeID, 10) + "/" + ownerType + "/" + strconv.FormatInt(ownerID, 10) + "/" + stamp() + ext
	tmp, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()
	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(src, maxUploadBytes+1))
	cerr := tmp.Close()
	if err != nil {
		os.Remove(tmpName)
		return 0, err
	}
	if size > maxUploadBytes {
		os.Remove(tmpName)
		return 0, errors.New("berkas melebihi batas 100MB")
	}
	if cerr != nil {
		os.Remove(tmpName)
		return 0, cerr
	}
	// Magic-byte sniff: don't trust client Content-Type alone.
	sniffed := ""
	if fh, err := os.Open(tmpName); err == nil {
		head := make([]byte, 512)
		n, _ := fh.Read(head)
		fh.Close()
		sniffed = http.DetectContentType(head[:n])
		if i := strings.IndexByte(sniffed, ';'); i >= 0 {
			sniffed = strings.TrimSpace(sniffed[:i])
		}
	}
	mime = strings.ToLower(strings.TrimSpace(mime))
	if sniffed != "" && sniffed != "application/octet-stream" {
		// Trust the sniffed bytes over the header, except generic fallback.
		if allowedMIME[sniffed] {
			mime = sniffed
		} else if !allowedMIME[mime] {
			os.Remove(tmpName)
			return 0, errors.New("tipe file tidak diizinkan (terdeteksi: " + sniffed + ")")
		} else {
			// Header says allowed but bytes disagree → reject (anti-spoof).
			os.Remove(tmpName)
			return 0, errors.New("isi file tidak sesuai tipe yang dinyatakan (" + sniffed + ")")
		}
	} else if !allowedMIME[mime] {
		os.Remove(tmpName)
		return 0, errors.New("tipe file tidak diizinkan (hanya JPG/PNG/WebP/PDF)")
	}
	f, err := os.Open(tmpName)
	if err != nil {
		os.Remove(tmpName)
		return 0, err
	}
	defer f.Close()
	defer os.Remove(tmpName)
	if err := s.files.Save(ref, f); err != nil {
		return 0, err
	}
	digest := hex.EncodeToString(h.Sum(nil))
	res, err := s.db.Exec(`INSERT INTO files(office_id, owner_type, owner_id, ref, original_name, mime, size_bytes, sha256, created_by)
		VALUES(?,?,?,?,?,?,?,?,?)`, officeID, ownerType, ownerID, ref, name, mime, size, digest, createdBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// List returns files for an owner (office-scoped).
func (s *Store) List(officeID int64, ownerType string, ownerID int64) ([]Meta, error) {
	rows, err := s.db.Query(`SELECT id, owner_type, owner_id, ref, original_name, mime, size_bytes, ocr_status
		FROM files WHERE office_id=? AND owner_type=? AND owner_id=? ORDER BY id DESC`, officeID, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Meta
	for rows.Next() {
		var m Meta
		if err := rows.Scan(&m.ID, &m.OwnerType, &m.OwnerID, &m.Ref, &m.Name, &m.MIME, &m.Size, &m.OCRStatus); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// OCRResults returns extraction history for a file (newest first).
func (s *Store) OCRResults(fileID int64) ([]ocr.Result, error) {
	rows, err := s.db.Query(`SELECT provider, provenance, doc_type, fields, created_at
		FROM ai_results WHERE file_id=? ORDER BY id DESC`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ocr.Result
	for rows.Next() {
		var r ocr.Result
		var fields sql.NullString
		var at string
		if err := rows.Scan(&r.Provider, &r.Provenance, &r.DocType, &fields, &at); err != nil {
			return nil, err
		}
		r.Fields = map[string]string{}
		if fields.Valid {
			_ = json.Unmarshal([]byte(fields.String), &r.Fields)
		}
		if t, err := time.Parse("2006-01-02 15:04:05", at); err == nil {
			r.CreatedAt = t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *Store) RunOCR(fileID int64, _ string) (ocr.Result, error) {
	return s.RunOCRWithContext(context.Background(), fileID)
}

// RunOCRWithContext executes OCR with cancellation/timeout and persists the
// extracted text into ai_results.fields (searchable). Honest statuses:
// DONE (tersimpan), FAILED (engine/error). Never auto-verifies.
func (s *Store) RunOCRWithContext(ctx context.Context, fileID int64) (ocr.Result, error) {
	fail := func(err error) (ocr.Result, error) {
		_, _ = s.db.Exec(`UPDATE files SET ocr_status='FAILED' WHERE id=?`, fileID)
		return ocr.Result{}, err
	}
	var ref, mime, orig string
	var officeID int64
	if err := s.db.QueryRow(`SELECT ref, mime, original_name, office_id FROM files WHERE id=?`, fileID).
		Scan(&ref, &mime, &orig, &officeID); err != nil {
		return fail(err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	rc, err := s.files.Open(ref)
	if err != nil {
		return fail(err)
	}
	defer rc.Close()
	tmp, err := os.CreateTemp("", "ocr-*"+filepath.Ext(orig))
	if err != nil {
		return fail(err)
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, io.LimitReader(rc, 100<<20)); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fail(err)
	}
	tmp.Close()
	defer os.Remove(tmpName)
	_ = mime
	_ = officeID
	res, err := s.ocr.Extract(ctx, tmpName)
	if err != nil {
		return fail(err)
	}
	if res.Fields == nil {
		res.Fields = map[string]string{}
	}
	if _, ok := res.Fields["text"]; !ok {
		res.Fields["text"] = ""
	}
	// Truncate stored text to 64KB (FULLTEXT-friendly, bukan BLOB PDF).
	if len(res.Fields["text"]) > 65535 {
		res.Fields["text"] = res.Fields["text"][:65535]
	}
	raw, _ := json.Marshal(res.Fields)
	if _, err := s.db.Exec(`INSERT INTO ai_results(file_id, provider, provenance, doc_type, fields)
		VALUES(?,?,?,?,?)`, fileID, res.Provider, res.Provenance, string(res.DocType), string(raw)); err != nil {
		return res, err
	}
	_, _ = s.db.Exec(`UPDATE files SET ocr_status='DONE' WHERE id=?`, fileID)
	return res, nil
}

// Get returns metadata + open reader, enforcing office scope (otorisasi unduhan).
func (s *Store) Get(officeID, fileID int64) (Meta, io.ReadCloser, error) {
	var m Meta
	var ref string
	err := s.db.QueryRow(`SELECT id, owner_type, owner_id, ref, original_name, mime, size_bytes, ocr_status
		FROM files WHERE id=? AND office_id=?`, fileID, officeID).
		Scan(&m.ID, &m.OwnerType, &m.OwnerID, &ref, &m.Name, &m.MIME, &m.Size, &m.OCRStatus)
	if err != nil {
		return Meta{}, nil, err
	}
	m.Ref = ref
	rc, err := s.files.Open(ref)
	if err != nil {
		return Meta{}, nil, err
	}
	return m, rc, nil
}

func stamp() string { return time.Now().UTC().Format("20060102T150405Z0700") }
