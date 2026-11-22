package deeds

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"notaryos/internal/docengine"
)

// Deed is a deed draft with version history.
type Deed struct {
	ID        int64
	OfficeID  int64
	RequestID *int64
	Number    string
	Title     string
	Kind      string
	Status    string
	Version   int
	Content   string
	Locked    bool
}

// Legalization is a legalisasi/waarmerking/salinan register entry.
type Legalization struct {
	ID         int64
	Kind       string
	RegisterNo string
	ClientID   int64
	DocTitle   string
	Fee        float64
	Status     string
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// CreateDraft creates a DRAFT deed from an approved template only.
func (s *Store) CreateDraft(officeID int64, requestID *int64, title, kind string, templateID *int64, createdBy *int64) (int64, error) {
	if title == "" {
		return 0, errors.New("judul akta wajib diisi")
	}
	var body string
	if templateID != nil {
		var approved bool
		err := s.db.QueryRow(`SELECT body, approved FROM doc_templates WHERE id=? AND office_id=?`,
			*templateID, officeID).Scan(&body, &approved)
		if err == sql.ErrNoRows {
			return 0, errors.New("template tidak ditemukan")
		}
		if err != nil {
			return 0, err
		}
		if !approved {
			return 0, errors.New("template belum disetujui notaris — draf tidak boleh dibuat")
		}
	}
	res, err := s.db.Exec(`INSERT INTO deed_drafts(office_id, request_id, title, kind, status, template_id, content, created_by)
		VALUES(?,?,?,?, 'DRAFT',?,?,?)`, officeID, requestID, title, kind, templateID, body, createdBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// SaveVersion stores content as a new version (blocked when locked/final).
// The pre-update draft content is snapshotted as the current version number
// first (so v1 is never lost), then the new content becomes version+1.
func (s *Store) SaveVersion(deedID int64, content string, createdBy *int64, note string) error {
	var locked bool
	var status string
	var version int
	var curContent sql.NullString
	if err := s.db.QueryRow(`SELECT locked, status, version, content FROM deed_drafts WHERE id=?`,
		deedID).Scan(&locked, &status, &version, &curContent); err != nil {
		return err
	}
	if locked || (status != "DRAFT" && status != "IN_REVIEW") {
		return errors.New("akta final terkunci — buat revisi resmi baru, jangan ubah final")
	}
	hash := docengine.VersionHash(content, nil)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var have int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM deed_versions WHERE deed_id=? AND version=?`, deedID, version).Scan(&have)
	if have == 0 {
		oldBody := ""
		if curContent.Valid {
			oldBody = curContent.String
		}
		oldHash := docengine.VersionHash(oldBody, nil)
		if _, err := tx.Exec(`INSERT INTO deed_versions(deed_id, version, content, content_hash, created_by, note)
			VALUES(?,?,?,?,?,?)`, deedID, version, oldBody, oldHash, createdBy, "snapshot otomatis v"+strconv.Itoa(version)); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO deed_versions(deed_id, version, content, content_hash, created_by, note)
		VALUES(?,?,?,?,?,?)`, deedID, version+1, content, hash, createdBy, note); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE deed_drafts SET content=?, content_hash=?, version=version+1 WHERE id=?`,
		content, hash, deedID); err != nil {
		return err
	}
	return tx.Commit()
}

// Approve locks the deed with notary approval, assigns a number, and writes
// the automatic repertorium entry (UUJN tertib administrasi).
// Race-safe: deed row locked; repertorium range locked via FOR UPDATE predicate;
// on 1062 collision the tx retries once with a fresh number, else friendly error.
func (s *Store) Approve(deedID int64, number string, approverID int64) error {
	if number == "" {
		return errors.New("nomor akta wajib diisi saat pengesahan")
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := s.approveOnce(deedID, number, approverID); err == nil {
			return nil
		} else if isDupEntry(err) && attempt == 0 {
			continue
		} else if isDupEntry(err) {
			return errors.New("nomor akta/entri repertorium sudah dipakai — gunakan nomor lain")
		} else {
			return err
		}
	}
	return errors.New("pengesahan gagal setelah 2 percobaan — coba lagi")
}

func (s *Store) approveOnce(deedID int64, number string, approverID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	var officeID int64
	var title string
	if err := tx.QueryRow(`SELECT status, office_id, title FROM deed_drafts WHERE id=? FOR UPDATE`, deedID).
		Scan(&status, &officeID, &title); err != nil {
		return err
	}
	if status != "IN_REVIEW" {
		return errors.New("hanya draf IN_REVIEW yang dapat disetujui")
	}
	var dup int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE office_id=? AND number=? AND id<>?`, officeID, number, deedID).Scan(&dup)
	if dup > 0 {
		return errDupNumber()
	}
	if _, err := tx.Exec(`UPDATE deed_drafts SET status='APPROVED', locked=1, number=?, approved_by=?, approved_at=UTC_TIMESTAMP()
		WHERE id=?`, number, approverID, deedID); err != nil {
		if isDupEntry(err) {
			return errDupNumber()
		}
		return err
	}
	var next int64 = 1
	_ = tx.QueryRow(`SELECT COALESCE(MAX(entry_no),0)+1 FROM repertorium WHERE office_id=? FOR UPDATE`, officeID).Scan(&next)
	if _, err := tx.Exec(`INSERT INTO repertorium(office_id, entry_no, entry_date, deed_id, description)
		VALUES(?,?,CURDATE(),?,?)`, officeID, next, deedID, "Akta "+number+": "+title); err != nil {
		return err
	}
	return tx.Commit()
}

func errDupNumber() error { return errors.New("nomor akta sudah dipakai di kantor ini") }

func isDupEntry(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "1062") || strings.Contains(msg, "Duplicate entry")
}

// List returns deeds newest first (optional status filter).
func (s *Store) List(officeID int64, status ...string) ([]Deed, error) {
	q := `SELECT id, office_id, request_id, number, title, kind, status, version, locked
		FROM deed_drafts WHERE office_id=?`
	args := []any{officeID}
	if len(status) > 0 && status[0] != "" {
		q += ` AND status=?`
		args = append(args, status[0])
	}
	q += ` ORDER BY id DESC LIMIT 200`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Deed
	for rows.Next() {
		var d Deed
		var num sql.NullString
		if err := rows.Scan(&d.ID, &d.OfficeID, &d.RequestID, &num, &d.Title, &d.Kind, &d.Status, &d.Version, &d.Locked); err != nil {
			return nil, err
		}
		if num.Valid {
			d.Number = num.String
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RegisterLegalization creates a repertorium-style register entry with fee.
// deedID links SALINAN/KUTIPAN/GROSSE to their minuta (nullable for standalone legalisasi).
func (s *Store) RegisterLegalization(officeID int64, kind, regNo string, clientID int64, title string, fee float64, deedID *int64) (int64, error) {
	switch kind {
	case "LEGALISASI", "WAARMERKING", "SALINAN", "KUTIPAN", "GROSSE":
	default:
		return 0, errors.New("jenis register tidak dikenal")
	}
	if regNo == "" || title == "" {
		return 0, errors.New("nomor register dan judul dokumen wajib diisi")
	}
	if deedID != nil {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE id=? AND office_id=?`, *deedID, officeID).Scan(&n); err != nil || n == 0 {
			return 0, errors.New("akta minuta tidak ditemukan di kantor ini")
		}
	}
	res, err := s.db.Exec(`INSERT INTO legalizations(office_id, kind, register_no, client_id, doc_title, fee, deed_id)
		VALUES(?,?,?,?,?,?,?)`, officeID, kind, regNo, clientID, title, fee, deedID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AdvanceLegalization moves REGISTERED → COMPLETED → HANDED_OVER (office-scoped).
func (s *Store) AdvanceLegalization(officeID, id int64, to string) error {
	var cur, kind string
	var oid int64
	if err := s.db.QueryRow(`SELECT status, office_id, kind FROM legalizations WHERE id=?`, id).Scan(&cur, &oid, &kind); err != nil {
		return err
	}
	if oid != officeID {
		return errors.New("register tidak ditemukan")
	}
	allowed := map[string][]string{
		"REGISTERED": {"COMPLETED"},
		"COMPLETED":  {"HANDED_OVER"},
	}
	ok := false
	for _, n := range allowed[cur] {
		if n == to {
			ok = true
		}
	}
	if !ok {
		return errors.New("transisi register tidak valid: " + cur + " → " + to)
	}
	_, err := s.db.Exec(`UPDATE legalizations SET status=?, completed_at=IF(?='COMPLETED',UTC_TIMESTAMP(),completed_at) WHERE id=?`, to, to, id)
	return err
}

// Version is one saved deed content revision.
type Version struct {
	No        int
	Content   string
	Hash      string
	Note      string
	CreatedBy *int64
	CreatedAt string
}

// Versions lists content revisions for a deed (office-scoped).
func (s *Store) Versions(officeID, deedID int64) ([]Version, error) {
	var oid int64
	if err := s.db.QueryRow(`SELECT office_id FROM deed_drafts WHERE id=?`, deedID).Scan(&oid); err != nil {
		return nil, err
	}
	if oid != officeID {
		return nil, errors.New("akta tidak ditemukan")
	}
	rows, err := s.db.Query(`SELECT version, content, content_hash, created_by, created_at, note FROM deed_versions WHERE deed_id=? ORDER BY version`, deedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Version
	for rows.Next() {
		var v Version
		var note sql.NullString
		if err := rows.Scan(&v.No, &v.Content, &v.Hash, &v.CreatedBy, &v.CreatedAt, &note); err != nil {
			return nil, err
		}
		if note.Valid {
			v.Note = note.String
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Get loads one deed (office-scoped).
func (s *Store) Get(officeID, deedID int64) (Deed, string, error) {
	var d Deed
	var num, content sql.NullString
	var req sql.NullInt64
	err := s.db.QueryRow(`SELECT id, office_id, request_id, number, title, kind, status, version, content, locked
		FROM deed_drafts WHERE id=? AND office_id=?`, deedID, officeID).
		Scan(&d.ID, &d.OfficeID, &req, &num, &d.Title, &d.Kind, &d.Status, &d.Version, &content, &d.Locked)
	if err != nil {
		return Deed{}, "", err
	}
	if req.Valid {
		v := req.Int64
		d.RequestID = &v
	}
	if num.Valid {
		d.Number = num.String
	}
	body := ""
	if content.Valid {
		body = content.String
	}
	return d, body, nil
}

// CreateProtocol records a protocol bundle (office-scoped, unique per year).
func (s *Store) CreateProtocol(officeID int64, bundle string, year int, minutaRef string) error {
	if bundle == "" || year < 2000 || year > 2100 {
		return errors.New("bendel/tahun protokol tidak valid")
	}
	var ref any
	if minutaRef != "" {
		ref = minutaRef
	}
	_, err := s.db.Exec(`INSERT INTO protocols(office_id, bundle_no, year, minuta_ref) VALUES(?,?,?,?)`,
		officeID, bundle, year, ref)
	return err
}
