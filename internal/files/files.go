package files

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"notaryos/internal/ocr"
	"notaryos/internal/storage"
)

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
	size, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(src, 100<<20))
	cerr := tmp.Close()
	if err != nil {
		os.Remove(tmpName)
		return 0, err
	}
	if cerr != nil {
		os.Remove(tmpName)
		return 0, cerr
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

// RunOCR executes the provider and stores the result with provenance.
// Human verification is a separate explicit step (VerifyOCR).
func (s *Store) RunOCR(fileID int64, tmpPath string) (ocr.Result, error) {
	res, err := s.ocr.Extract(nil, tmpPath)
	if err != nil {
		_, _ = s.db.Exec(`UPDATE files SET ocr_status='FAILED' WHERE id=?`, fileID)
		return ocr.Result{}, err
	}
	_, err = s.db.Exec(`INSERT INTO ai_results(file_id, provider, provenance, doc_type)
		VALUES(?,?,?,?)`, fileID, res.Provider, res.Provenance, string(res.DocType))
	if err != nil {
		return res, err
	}
	_, _ = s.db.Exec(`UPDATE files SET ocr_status='DONE' WHERE id=?`, fileID)
	return res, nil
}

func stamp() string { return time.Now().UTC().Format("20060102T150405Z0700") }
