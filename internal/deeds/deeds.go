package deeds

import (
	"database/sql"
	"errors"

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
func (s *Store) SaveVersion(deedID int64, content string, createdBy *int64, note string) error {
	var locked bool
	var status string
	var version int
	if err := s.db.QueryRow(`SELECT locked, status, version FROM deed_drafts WHERE id=?`,
		deedID).Scan(&locked, &status, &version); err != nil {
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
func (s *Store) Approve(deedID int64, number string, approverID int64) error {
	if number == "" {
		return errors.New("nomor akta wajib diisi saat pengesahan")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	var officeID int64
	var title string
	if err := tx.QueryRow(`SELECT status, office_id, title FROM deed_drafts WHERE id=?`, deedID).
		Scan(&status, &officeID, &title); err != nil {
		return err
	}
	if status != "IN_REVIEW" {
		return errors.New("hanya draf IN_REVIEW yang dapat disetujui")
	}
	if _, err := tx.Exec(`UPDATE deed_drafts SET status='APPROVED', number=?, approved_by=?, approved_at=UTC_TIMESTAMP()
		WHERE id=?`, number, approverID, deedID); err != nil {
		return err
	}
	var next int64 = 1
	_ = tx.QueryRow(`SELECT COALESCE(MAX(entry_no),0)+1 FROM repertorium WHERE office_id=?`, officeID).Scan(&next)
	if _, err := tx.Exec(`INSERT INTO repertorium(office_id, entry_no, entry_date, deed_id, description)
		VALUES(?,?,CURDATE(),?,?,?)`, officeID, next, deedID, "Akta "+number+": "+title); err != nil {
		return err
	}
	return tx.Commit()
}

// List returns deeds newest first.
func (s *Store) List(officeID int64) ([]Deed, error) {
	rows, err := s.db.Query(`SELECT id, office_id, request_id, number, title, kind, status, version, locked
		FROM deed_drafts WHERE office_id=? ORDER BY id DESC LIMIT 200`, officeID)
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
func (s *Store) RegisterLegalization(officeID int64, kind, regNo string, clientID int64, title string, fee float64) (int64, error) {
	switch kind {
	case "LEGALISASI", "WAARMERKING", "SALINAN", "KUTIPAN":
	default:
		return 0, errors.New("jenis register tidak dikenal")
	}
	if regNo == "" || title == "" {
		return 0, errors.New("nomor register dan judul dokumen wajib diisi")
	}
	res, err := s.db.Exec(`INSERT INTO legalizations(office_id, kind, register_no, client_id, doc_title, fee)
		VALUES(?,?,?,?,?,?)`, officeID, kind, regNo, clientID, title, fee)
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
