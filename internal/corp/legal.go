package corp

import "database/sql"

// Legal holds notarial-assistive corporate registry references (manual entry only).
// No AHU/OSS integration is claimed — fields are archives of manually checked docs.
type Legal struct {
	DeedNo, DeedDate          string
	AmendmentNo, AmendmentDate string
	AHUNo, NPWP, NIB          string
}

// LegalOf reads the legal reference fields for a company.
func (s *Store) LegalOf(companyID int64) (Legal, error) {
	var l Legal
	var dn, dd, an, ad, ahu, npwp, nib sql.NullString
	err := s.db.QueryRow(`SELECT deed_no, deed_date, amendment_no, amendment_date, ahu_no, npwp, nib
		FROM companies WHERE id=?`, companyID).Scan(&dn, &dd, &an, &ad, &ahu, &npwp, &nib)
	if err != nil {
		return l, err
	}
	l.DeedNo, l.DeedDate = dn.String, dd.String
	l.AmendmentNo, l.AmendmentDate = an.String, ad.String
	l.AHUNo, l.NPWP, l.NIB = ahu.String, npwp.String, nib.String
	return l, nil
}

// SaveLegal upserts the legal reference fields (empty strings clear to NULL).
func (s *Store) SaveLegal(companyID int64, l Legal) error {
	_, err := s.db.Exec(`UPDATE companies SET deed_no=NULLIF(?,''),
		deed_date=NULLIF(?,'0000-00-00'), amendment_no=NULLIF(?,''),
		amendment_date=NULLIF(?,'0000-00-00'), ahu_no=NULLIF(?,''),
		npwp=NULLIF(?,'') , nib=NULLIF(?,'') WHERE id=?`,
		l.DeedNo, emptyDate(l.DeedDate), l.AmendmentNo, emptyDate(l.AmendmentDate),
		l.AHUNo, l.NPWP, l.NIB, companyID)
	return err
}

func emptyDate(v string) string {
	if v == "" {
		return "0000-00-00"
	}
	return v
}
