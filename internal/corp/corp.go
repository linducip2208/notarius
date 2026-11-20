package corp

import (
	"database/sql"
	"errors"
)

// Company, Shareholder, Officer mirror the corporate tables.
type Company struct {
	ID         int64
	RequestID  *int64
	EntityType string
	Name       string
	Domicile   string
	Address    string
	Authorized float64
	Issued     float64
	Paid       float64
}

type Shareholder struct {
	ID      int64
	Name    string
	Shares  int
	Nominal float64
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
	err := s.db.QueryRow(`SELECT id, request_id, entity_type, name, domicile, address,
		authorized_capital, issued_capital, paid_capital FROM companies WHERE request_id=?`, requestID).
		Scan(&c.ID, &c.RequestID, &c.EntityType, &c.Name, &dom, &addr, &c.Authorized, &c.Issued, &c.Paid)
	if err != nil {
		if err == sql.ErrNoRows {
			return Company{}, nil, nil, nil
		}
		return Company{}, nil, nil, err
	}
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
		shs = append(shs, Shareholder{ID: r[0].(int64), Name: r[1].(string), Shares: int(r[2].(int64)), Nominal: r[3].(float64)})
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
		var nom float64
		if err := rows.Scan(&id, &name, &shares, &nom); err != nil {
			return nil, err
		}
		out = append(out, []any{id, name, shares, nom})
	}
	return out, rows.Err()
}

// AddShareholder appends a shareholder row.
func (s *Store) AddShareholder(companyID int64, name string, shares int, nominal float64) error {
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
		if err := rows.Scan(&b.ID, &b.Name, &last4, &b.OwnershipPct, &pep); err != nil {
			return nil, err
		}
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
