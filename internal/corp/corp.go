package corp

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Company, Shareholder, Officer mirror the corporate tables.
type Company struct {
	ID         int64
	RequestID  *int64
	EntityType string
	Name       string
	Domicile   string
	Address    string
	Authorized int64
	Issued     int64
	Paid       int64
}

type Shareholder struct {
	ID      int64
	Name    string
	Shares  int
	Nominal int64
}

type Officer struct {
	ID    int64
	Role  string
	Name  string
	Title string
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// UpsertCompany creates or updates the company record for a request.
func (s *Store) UpsertCompany(c Company) (int64, error) {
	if c.Name == "" {
		return 0, errors.New("nama perusahaan wajib diisi")
	}
	switch c.EntityType {
	case "PT", "CV", "Firma", "Yayasan", "Koperasi", "Perkumpulan":
	default:
		return 0, errors.New("bentuk badan tidak dikenal")
	}
	if c.RequestID != nil {
		var id int64
		err := s.db.QueryRow(`SELECT id FROM companies WHERE request_id=?`, *c.RequestID).Scan(&id)
		if err == nil {
			_, err = s.db.Exec(`UPDATE companies SET entity_type=?, name=?, domicile=?, address=?,
				authorized_capital=?, issued_capital=?, paid_capital=? WHERE id=?`,
				c.EntityType, c.Name, c.Domicile, c.Address, c.Authorized, c.Issued, c.Paid, id)
			return id, err
		}
		if err != sql.ErrNoRows {
			return 0, err
		}
	}
	res, err := s.db.Exec(`INSERT INTO companies(request_id, entity_type, name, domicile, address,
		authorized_capital, issued_capital, paid_capital) VALUES(?,?,?,?,?,?,?,?)`,
		c.RequestID, c.EntityType, c.Name, c.Domicile, c.Address, c.Authorized, c.Issued, c.Paid)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetByRequest loads company + shareholders + officers for a request.
func (s *Store) GetByRequest(requestID int64) (Company, []Shareholder, []Officer, error) {
	var c Company
	var dom, addr sql.NullString
	var sauth, siss, spaid string
	err := s.db.QueryRow(`SELECT id, request_id, entity_type, name, domicile, address,
		authorized_capital, issued_capital, paid_capital FROM companies WHERE request_id=?`, requestID).
		Scan(&c.ID, &c.RequestID, &c.EntityType, &c.Name, &dom, &addr, &sauth, &siss, &spaid)
	if err != nil {
		if err == sql.ErrNoRows {
			return Company{}, nil, nil, nil
		}
		return Company{}, nil, nil, err
	}
	c.Authorized = decRupiah(sauth)
	c.Issued = decRupiah(siss)
	c.Paid = decRupiah(spaid)
	if dom.Valid {
		c.Domicile = dom.String
	}
	if addr.Valid {
		c.Address = addr.String
	}
	sh, err := s.list(`SELECT id, name, shares, nominal_per_share FROM shareholders WHERE company_id=?`, c.ID)
	if err != nil {
		return c, nil, nil, err
	}
	var shs []Shareholder
	for _, r := range sh {
		shs = append(shs, Shareholder{ID: r[0].(int64), Name: r[1].(string), Shares: int(r[2].(int64)), Nominal: r[3].(int64)})
	}
	rows, err := s.db.Query(`SELECT id, role, name, title FROM company_officers WHERE company_id=?`, c.ID)
	if err != nil {
		return c, shs, nil, err
	}
	defer rows.Close()
	var offs []Officer
	for rows.Next() {
		var o Officer
		var title sql.NullString
		if err := rows.Scan(&o.ID, &o.Role, &o.Name, &title); err != nil {
			return c, shs, nil, err
		}
		if title.Valid {
			o.Title = title.String
		}
		offs = append(offs, o)
	}
	return c, shs, offs, rows.Err()
}

func (s *Store) list(q string, id int64) ([][]any, error) {
	rows, err := s.db.Query(q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][]any
	for rows.Next() {
		var id int64
		var name string
		var shares int64
		var nomStr string
		if err := rows.Scan(&id, &name, &shares, &nomStr); err != nil {
			return nil, err
		}
		out = append(out, []any{id, name, shares, decRupiah(nomStr)})
	}
	return out, rows.Err()
}

// decRupiah parses DECIMAL(18,2) into whole rupiah (drops cent fraction honestly).
func decRupiah(s string) int64 {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "."); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}
func (s *Store) AddShareholder(companyID int64, name string, shares int, nominal int64) error {
	if name == "" || shares <= 0 || nominal < 0 {
		return errors.New("data pemegang saham tidak valid")
	}
	_, err := s.db.Exec(`INSERT INTO shareholders(company_id, name, shares, nominal_per_share) VALUES(?,?,?,?)`,
		companyID, name, shares, nominal)
	return err
}

// AddOfficer appends a director/commissioner/officer row.
func (s *Store) AddOfficer(companyID int64, role, name, title string) error {
	switch role {
	case "DIREKTUR", "KOMISARIS", "PENGURUS", "PENGAWAS", "SEKUTU":
	default:
		return errors.New("jabatan tidak dikenal")
	}
	if name == "" {
		return errors.New("nama pengurus wajib diisi")
	}
	_, err := s.db.Exec(`INSERT INTO company_officers(company_id, role, name, title) VALUES(?,?,?,?)`,
		companyID, role, name, title)
	return err
}

// BeneficialOwner is the ultimate natural-person owner (UU TPPU/beneficial ownership).
type BeneficialOwner struct {
	ID           int64
	Name         string
	NIKLast4     string
	OwnershipPct float64
	IsPEP        bool
}

// AddBeneficialOwner records a PEP-flagged ultimate owner (F-009 complete).
func (s *Store) AddBeneficialOwner(companyID int64, name, nik string, pct float64, pep bool) error {
	if name == "" {
		return errors.New("nama beneficial owner wajib diisi")
	}
	if pct <= 0 || pct > 100 {
		return errors.New("persentase kepemilikan harus 1-100")
	}
	var total float64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(ownership_pct),0) FROM beneficial_owners WHERE company_id=?`, companyID).Scan(&total)
	if total+pct > 100 {
		return errors.New("total kepemilikan melebihi 100% — periksa kembali")
	}
	var last4 any
	if len(nik) >= 4 {
		last4 = nik[len(nik)-4:]
	}
	_, err := s.db.Exec(`INSERT INTO beneficial_owners(company_id, name, nik_last4, ownership_pct, is_pep)
		VALUES(?,?,?,?,?)`, companyID, name, last4, pct, boolToInt(pep))
	return err
}

// BeneficialOwners lists ultimate owners for a company.
func (s *Store) BeneficialOwners(companyID int64) ([]BeneficialOwner, error) {
	rows, err := s.db.Query(`SELECT id, name, nik_last4, ownership_pct, is_pep FROM beneficial_owners WHERE company_id=? ORDER BY id`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BeneficialOwner
	for rows.Next() {
		var b BeneficialOwner
		var last4 sql.NullString
		var pep int
		var pct string
		if err := rows.Scan(&b.ID, &b.Name, &last4, &pct, &pep); err != nil {
			return nil, err
		}
		fmt.Sscanf(pct, "%f", &b.OwnershipPct)
		if last4.Valid {
			b.NIKLast4 = last4.String
		}
		b.IsPEP = pep == 1
		out = append(out, b)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// WizardStep is one guided section of the corporate wizard.
type WizardStep struct {
	Key   string
	Title string
	Done  bool
	Hint  string
}

// WizardProgress computes per-entity-type completeness steps plus honest warnings.
// Pure function — unit tested.
func WizardProgress(c Company, shs []Shareholder, offs []Officer, nbos int) (steps []WizardStep, warnings []string) {
	hasCo := c.ID != 0 && c.Name != "" && c.Domicile != "" && c.EntityType != ""
	steps = append(steps, WizardStep{Key: "identitas", Title: "Bentuk & identitas usaha",
		Done: hasCo, Hint: "Pilih bentuk (PT/CV/Yayasan/dll), isi nama + domisili"})
	needsCapital := c.EntityType == "PT" || c.EntityType == "Firma" || c.EntityType == "Koperasi"
	capDone := hasCo && (!needsCapital || (c.Authorized > 0 && c.Issued > 0 && c.Paid > 0))
	capHint := "CV/Yayasan tak wajib modal — langkah ini otomatis selesai"
	if needsCapital {
		capHint = "Isi modal dasar, ditempatkan, dan disetor"
	}
	steps = append(steps, WizardStep{Key: "modal", Title: "Permodalan", Done: capDone, Hint: capHint})
	var nSekutu, nDirektur, nPengurus, nPengawas int
	for _, o := range offs {
		switch o.Role {
		case "SEKUTU":
			nSekutu++
		case "DIREKTUR":
			nDirektur++
		case "PENGURUS":
			nPengurus++
		case "PENGAWAS":
			nPengawas++
		}
	}
	var ownDone bool
	var ownHint string
	switch c.EntityType {
	case "CV":
		ownDone = nSekutu >= 2
		ownHint = "Minimal 2 sekutu (aktif + pasif)"
	case "Yayasan":
		ownDone = nPengurus >= 1 && nPengawas >= 1
		ownHint = "Minimal 1 pengurus + 1 pengawas"
	default: // PT dkk: pemegang saham
		ownDone = len(shs) >= 1
		ownHint = "Minimal 1 pemegang saham"
	}
	steps = append(steps, WizardStep{Key: "pemilik", Title: "Pemilik/pendiri", Done: ownDone, Hint: ownHint})
	var mgrDone bool
	var mgrHint string
	switch c.EntityType {
	case "CV":
		mgrDone = nSekutu >= 1
		mgrHint = "Sekutu aktif pengelola"
	case "Yayasan":
		mgrDone = nPengurus >= 3
		mgrHint = "Lazimnya ketua + sekretaris + bendahara"
	default:
		mgrDone = nDirektur >= 1
		mgrHint = "Minimal 1 direktur"
	}
	steps = append(steps, WizardStep{Key: "pengurus", Title: "Pengurus", Done: mgrDone, Hint: mgrHint})
	boNeed := c.EntityType == "PT" || c.EntityType == "CV" || c.EntityType == "Firma"
	boDone := !boNeed || nbos >= 1
	steps = append(steps, WizardStep{Key: "bo", Title: "Beneficial Owner", Done: boDone,
		Hint: "Wajib untuk PT/CV/Firma (ketentuan transparansi BO)"})
	if needsCapital && c.Authorized > 0 && c.Paid*4 < c.Authorized {
		warnings = append(warnings, "Modal disetor lazimnya ≥25% modal dasar (UU PT) — periksa kembali")
	}
	if c.Issued > c.Authorized {
		warnings = append(warnings, "Modal ditempatkan melebihi modal dasar — tidak sah")
	}
	return steps, warnings
}
