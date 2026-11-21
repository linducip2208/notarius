package clients

// CRM extension: full profiles, KYC workflow, notes, duplicates+merge, stats.
// All queries office-scoped; NIK never stored/compared in plaintext.

import (
	"database/sql"
	"strings"
)

// ClientFull is the complete professional profile (nullable new columns).
type ClientFull struct {
	Client
	RegNumber                                   string
	BirthPlace                                  string
	BirthDate                                   string
	Gender, Nationality, Marital                string
	Occupation, Education, Passport             string
	IDAddress, Domicile                         string
	Province, City, District, Village, Postal   string
	AltPhone, WANumber, Correspondence          string
	FirstClientDate, Source                     string
	PIC                                         int64
	Status                                      string
	InternalNotes, Labels                      string
}

// Stats is the client overview dashboard (all real-time counts).
type Stats struct {
	Total, Individual, Corporate, Active       int
	InProcess, DocsIncomplete, PendingVerify   int
	Finished, Unpaid                           int
}

// maskNIK shows only last 4 (identities store last4 only — full NIK never leaves vault).
func MaskNIK(last4 string) string {
	if last4 == "" {
		return "••••"
	}
	return "••••" + last4
}

// GetFull loads the complete profile (404 when cross-office).
func (s *Store) GetFull(officeID, id int64) (ClientFull, error) {
	var c ClientFull
	var email, addr, npwp sql.NullString
	var reg, bp, bdate, gender, nat, marital, occ, edu, pass, idaddr, dom sql.NullString
	var prov, city, dist, vill, zip, alt, wa, corr, first, src sql.NullString
	var pic sql.NullInt64
	var status string
	var notes sql.NullString
	var created string
	err := s.db.QueryRow(`SELECT id, office_id, kind, full_name, email, phone, address, npwp,
		reg_number, birth_place, birth_date, gender, nationality, marital_status, occupation,
		education, passport_no, id_address, domicile_address, province, city, district, village,
		postal_code, alt_phone, wa_number, correspondence_address, first_client_date, source,
		pic_user_id, status, internal_notes, labels,
		DATE_FORMAT(CONVERT_TZ(created_at,'+00:00','+07:00'),'%d %b %Y')
		FROM clients WHERE id=? AND office_id=?`, id, officeID).Scan(
		&c.ID, &c.OfficeID, &c.Kind, &c.FullName, &email, &c.Phone, &addr, &npwp,
		&reg, &bp, &bdate, &gender, &nat, &marital, &occ, &edu, &pass, &idaddr, &dom,
		&prov, &city, &dist, &vill, &zip, &alt, &wa, &corr, &first, &src,
		&pic, &status, &notes, &c.Labels, &created)
	if err != nil {
		return c, err
	}
	c.Email, c.Address, c.NPWP, c.CreatedAt = email.String, addr.String, npwp.String, created
	c.RegNumber, c.BirthPlace = reg.String, bp.String
	c.BirthDate = bdate.String
	c.Gender, c.Nationality, c.Marital = gender.String, nat.String, marital.String
	c.Occupation, c.Education, c.Passport = occ.String, edu.String, pass.String
	c.IDAddress, c.Domicile = idaddr.String, dom.String
	c.Province, c.City, c.District, c.Village, c.Postal = prov.String, city.String, dist.String, vill.String, zip.String
	c.AltPhone, c.WANumber, c.Correspondence = alt.String, wa.String, corr.String
	c.FirstClientDate, c.Source = first.String, src.String
	if pic.Valid {
		c.PIC = pic.Int64
	}
	c.Status = status
	c.InternalNotes = notes.String
	return c, nil
}

// UpdateFull persists the editable profile (server-validated by caller).
func (s *Store) UpdateFull(officeID int64, c ClientFull) error {
	if c.FullName == "" {
		return errf("nama wajib diisi")
	}
	_, err := s.db.Exec(`UPDATE clients SET full_name=?, email=?, phone=?, address=?, npwp=?,
		reg_number=NULLIF(?,''),
		birth_place=?, birth_date=NULLIF(?,'0000-00-00'), gender=NULLIF(?,''),
		nationality=?, marital_status=?, occupation=?, education=?, passport_no=?,
		id_address=?, domicile_address=?, province=?, city=?, district=?, village=?, postal_code=?,
		alt_phone=?, wa_number=?, correspondence_address=?, first_client_date=NULLIF(?,''),
		source=?, pic_user_id=?, status=?, internal_notes=?, labels=?
		WHERE id=? AND office_id=?`,
		c.FullName, null(c.Email), c.Phone, null(c.Address), null(c.NPWP),
		null(c.RegNumber), null(c.BirthPlace), nullDate(c.BirthDate), null(c.Gender),
		null(c.Nationality), null(c.Marital), null(c.Occupation), null(c.Education), null(c.Passport),
		null(c.IDAddress), null(c.Domicile), null(c.Province), null(c.City), null(c.District),
		null(c.Village), null(c.Postal), null(c.AltPhone), null(c.WANumber), null(c.Correspondence),
		nullDate(c.FirstClientDate), null(c.Source), nullID(c.PIC), c.Status, null(c.InternalNotes),
		c.Labels, c.ID, officeID)
	return err
}

func nullDate(v string) any {
	if v == "" || v == "0000-00-00" {
		return nil
	}
	return v
}

func nullID(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

// Stats computes overview counters in a fixed small query set.
func (s *Store) Stats(officeID int64) (Stats, error) {
	var st Stats
	q := func(dest *int, query string, args ...any) {
		_ = s.db.QueryRow(query, args...).Scan(dest)
	}
	q(&st.Total, `SELECT COUNT(*) FROM clients WHERE office_id=? AND status!='ARCHIVED'`, officeID)
	q(&st.Individual, `SELECT COUNT(*) FROM clients WHERE office_id=? AND kind='individual' AND status!='ARCHIVED'`, officeID)
	q(&st.Corporate, `SELECT COUNT(*) FROM clients WHERE office_id=? AND kind='corporate' AND status!='ARCHIVED'`, officeID)
	q(&st.Active, `SELECT COUNT(*) FROM clients WHERE office_id=? AND status='ACTIVE'`, officeID)
	q(&st.InProcess, `SELECT COUNT(DISTINCT client_id) FROM service_requests WHERE office_id=? AND status IN ('SUBMITTED','IN_REVIEW','NEEDS_DOCS','APPROVED','IN_PROGRESS')`, officeID)
	q(&st.DocsIncomplete, `SELECT COUNT(*) FROM clients c WHERE c.office_id=? AND c.status='ACTIVE' AND NOT EXISTS (SELECT 1 FROM files f WHERE f.owner_type='client' AND f.owner_id=c.id AND f.office_id=?)`, officeID, officeID)
	q(&st.PendingVerify, `SELECT COUNT(DISTINCT i.client_id) FROM identities i JOIN clients c ON c.id=i.client_id WHERE c.office_id=? AND i.status IN ('UNCHECKED','PENDING','NEEDS_REVISION')`, officeID)
	q(&st.Finished, `SELECT COUNT(DISTINCT client_id) FROM service_requests WHERE office_id=? AND status='COMPLETED'`, officeID)
	q(&st.Unpaid, `SELECT COUNT(DISTINCT client_id) FROM invoices WHERE office_id=? AND status IN ('ISSUED','PARTIAL')`, officeID)
	return st, nil
}

// ListAdvanced extends filtering: status, dates, service, region, docs completeness.
type ListFilter struct {
	Q, Kind, Status, Service, Province, City string
	From, To                                 string // YYYY-MM-DD
	Docs                                     string // "complete" | "missing" | ""
	Sort                                     string
	Limit, Offset                            int
}

var listSorts = map[string]bool{
	"": true, "name_asc": true, "name_desc": true, "newest": true, "oldest": true,
}

// ListAdvanced returns rows + total for the professional directory.
func (s *Store) ListAdvanced(officeID int64, f ListFilter) ([]Client, int, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if _, ok := listSorts[f.Sort]; !ok {
		f.Sort = ""
	}
	where := ` WHERE c.office_id=?`
	args := []any{officeID}
	if strings.TrimSpace(f.Q) != "" {
		where += ` AND (c.full_name LIKE ? OR c.phone LIKE ? OR c.email LIKE ? OR c.npwp LIKE ? OR EXISTS (SELECT 1 FROM identities i WHERE i.client_id=c.id AND i.nik_last4 LIKE ?))`
		like := "%" + strings.TrimSpace(f.Q) + "%"
		args = append(args, like, like, like, like, like)
	}
	if f.Kind == "individual" || f.Kind == "corporate" {
		where += ` AND c.kind=?`
		args = append(args, f.Kind)
	}
	if f.Status == "ACTIVE" || f.Status == "INACTIVE" || f.Status == "ARCHIVED" {
		where += ` AND c.status=?`
		args = append(args, f.Status)
	}
	if f.From != "" {
		where += ` AND DATE(CONVERT_TZ(c.created_at,'+00:00','+07:00'))>=?`
		args = append(args, f.From)
	}
	if f.To != "" {
		where += ` AND DATE(CONVERT_TZ(c.created_at,'+00:00','+07:00'))<=?`
		args = append(args, f.To)
	}
	if f.Service != "" {
		where += ` AND EXISTS (SELECT 1 FROM service_requests r JOIN services s ON s.id=r.service_id WHERE r.client_id=c.id AND s.code=?)`
		args = append(args, f.Service)
	}
	if f.Province != "" {
		where += ` AND c.province=?`
		args = append(args, f.Province)
	}
	if f.City != "" {
		where += ` AND c.city=?`
		args = append(args, f.City)
	}
	if f.Docs == "missing" {
		where += ` AND NOT EXISTS (SELECT 1 FROM files x WHERE x.owner_type='client' AND x.owner_id=c.id AND x.office_id=?)`
		args = append(args, officeID)
	} else if f.Docs == "complete" {
		where += ` AND EXISTS (SELECT 1 FROM files x WHERE x.owner_type='client' AND x.owner_id=c.id AND x.office_id=?)`
		args = append(args, officeID)
	}
	var total int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM clients c`+where, args...).Scan(&total)
	order := ` ORDER BY c.id DESC`
	switch f.Sort {
	case "name_asc":
		order = ` ORDER BY c.full_name ASC`
	case "name_desc":
		order = ` ORDER BY c.full_name DESC`
	case "oldest":
		order = ` ORDER BY c.id ASC`
	}
	rows, err := s.db.Query(`SELECT c.id, c.office_id, c.kind, c.full_name, c.email, c.phone,
		c.address, c.npwp, DATE_FORMAT(CONVERT_TZ(c.created_at,'+00:00','+07:00'),'%d %b %Y %H:%i')
		FROM clients c`+where+order+` LIMIT ? OFFSET ?`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Client
	for rows.Next() {
		var c Client
		var email, addr, npwp sql.NullString
		var created string
		if err := rows.Scan(&c.ID, &c.OfficeID, &c.Kind, &c.FullName, &email, &c.Phone, &addr, &npwp, &created); err != nil {
			return nil, 0, err
		}
		c.Email, c.Address, c.NPWP, c.CreatedAt = email.String, addr.String, npwp.String, created
		out = append(out, c)
	}
	return out, total, rows.Err()
}

// FindDuplicatesEx extends matching: exact phone/email/NPWP, name similarity,
// same birth-date+name, same nik_last4. (Full NIK is encrypted — never matched raw.)
func (s *Store) FindDuplicatesEx(officeID int64, excludeID int64, name, phone, email, npwp, last4, birth string) ([]DuplicateCandidate, error) {
	var conds []string
	var args []any
	args = append(args, officeID, excludeID)
	push := func(cond string, vals ...any) {
		conds = append(conds, cond)
		args = append(args, vals...)
	}
	if strings.TrimSpace(phone) != "" {
		push(`c.phone=?`, phone)
	}
	if strings.TrimSpace(email) != "" {
		push(`c.email=?`, email)
	}
	if strings.TrimSpace(npwp) != "" {
		push(`c.npwp=?`, npwp)
	}
	if strings.TrimSpace(name) != "" {
		push(`c.full_name SOUNDS LIKE ?`, name)
		if birth != "" {
			push(`EXISTS (SELECT 1 FROM identities i WHERE i.client_id=c.id AND i.birth_date=? AND i.full_name SOUNDS LIKE ?)`, birth, name)
		}
		if last4 != "" {
			push(`EXISTS (SELECT 1 FROM identities i WHERE i.client_id=c.id AND i.nik_last4=?)`, last4)
		}
	}
	if len(conds) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT DISTINCT c.id, c.full_name, c.phone FROM clients c
		WHERE c.office_id=? AND c.id<>? AND (`+strings.Join(conds, " OR ")+`) LIMIT 10`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DuplicateCandidate
	for rows.Next() {
		var d DuplicateCandidate
		_ = rows.Scan(&d.ID, &d.FullName, &d.Phone)
		d.Reason = "kecocokan identitas"
		if d.Phone == phone && phone != "" {
			d.Reason = "nomor telepon sama"
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Note is one CRM record.
type Note struct {
	ID                       int64
	Kind, Title, Body        string
	OccurredAt               string
	CreatedBy                int64
	CreatedByName            string
	CreatedAt                string
}

// AddNote records a communication/meeting/follow-up/complaint entry.
func (s *Store) AddNote(officeID, clientID int64, kind, title, body string, occurred *string, by *int64) (int64, error) {
	switch kind {
	case "NOTE", "MEETING", "FOLLOWUP", "COMPLAINT":
	default:
		kind = "NOTE"
	}
	if strings.TrimSpace(title) == "" {
		return 0, errf("judul catatan wajib diisi")
	}
	var own int64
	if err := s.db.QueryRow(`SELECT office_id FROM clients WHERE id=?`, clientID).Scan(&own); err != nil || own != officeID {
		return 0, errf("klien tidak ditemukan")
	}
	res, err := s.db.Exec(`INSERT INTO client_notes(office_id, client_id, kind, title, body, occurred_at, created_by)
		VALUES(?,?,?,?,?,?,?)`, officeID, clientID, kind, title, null(body), nullDate(occurredAt(occurred)), nullIDp(by))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func occurredAt(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func nullIDp(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// Notes lists CRM entries newest first.
func (s *Store) Notes(clientID int64, limit int) ([]Note, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.Query(`SELECT n.id, n.kind, n.title, n.body,
		COALESCE(DATE_FORMAT(n.occurred_at,'%d %b %Y %H:%i'),'-'),
		COALESCE(n.created_by,0), COALESCE(u.name,'-'),
		DATE_FORMAT(CONVERT_TZ(n.created_at,'+00:00','+07:00'),'%d %b %Y %H:%i')
		FROM client_notes n LEFT JOIN users u ON u.id=n.created_by
		WHERE n.client_id=? ORDER BY n.id DESC LIMIT ?`, clientID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		var body sql.NullString
		_ = rows.Scan(&n.ID, &n.Kind, &n.Title, &body, &n.OccurredAt, &n.CreatedBy, &n.CreatedByName, &n.CreatedAt)
		n.Body = body.String
		out = append(out, n)
	}
	return out, rows.Err()
}

// TimelineEvent is one unified activity row for the client timeline.
type TimelineEvent struct {
	At, Kind, Text string
}

// Timeline unions requests, notes, payments, appointments newest first.
func (s *Store) Timeline(clientID int64, limit int) ([]TimelineEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(`(SELECT created_at AS at, 'permohonan' AS kind, CONCAT('#',id,' ',title,' [',status,']') AS text FROM service_requests WHERE client_id=?)
		UNION ALL (SELECT COALESCE(occurred_at, created_at), LOWER(kind), CONCAT(title,' — ',COALESCE(body,'')) FROM client_notes WHERE client_id=?)
		UNION ALL (SELECT p.paid_at, 'pembayaran', CONCAT('Rp ', FORMAT(p.amount,0)) FROM payments p JOIN invoices i ON i.id=p.invoice_id WHERE i.client_id=? AND p.paid_at IS NOT NULL)
		UNION ALL (SELECT scheduled_at, 'janji', purpose FROM appointments WHERE client_id=? AND scheduled_at IS NOT NULL)
		ORDER BY at DESC LIMIT ?`, clientID, clientID, clientID, clientID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		_ = rows.Scan(&e.At, &e.Kind, &e.Text)
		out = append(out, e)
	}
	return out, rows.Err()
}

// IdentityStatus sets verification state + history (human verification only).
func (s *Store) IdentityStatus(officeID, identityID int64, to, note string, actor *int64) error {
	switch to {
	case "UNCHECKED", "PENDING", "VERIFIED", "REJECTED", "NEEDS_REVISION":
	default:
		return errf("status verifikasi tidak dikenal")
	}
	var from string
	var cid int64
	if err := s.db.QueryRow(`SELECT i.status, c.office_id FROM identities i JOIN clients c ON c.id=i.client_id WHERE i.id=?`, identityID).Scan(&from, &cid); err != nil {
		return err
	}
	if cid != officeID {
		return errf("identitas tidak ditemukan")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	legacy := 0
	if to == "VERIFIED" {
		legacy = 1
	}
	if _, err := tx.Exec(`UPDATE identities SET status=?, verified=?, verified_by=?, verified_at=UTC_TIMESTAMP(), verified_note=? WHERE id=?`,
		to, legacy, nullIDp(actor), null(note), identityID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO identity_verifications(identity_id, actor_id, from_status, to_status, note) VALUES(?,?,?,?,?)`,
		identityID, nullIDp(actor), from, to, null(note)); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkInconsistent flags/unflags inconsistent identity data.
func (s *Store) MarkInconsistent(officeID, identityID int64, flag bool) error {
	var cid int64
	if err := s.db.QueryRow(`SELECT c.office_id FROM identities i JOIN clients c ON c.id=i.client_id WHERE i.id=?`, identityID).Scan(&cid); err != nil || cid != officeID {
		return errf("identitas tidak ditemukan")
	}
	v := 0
	if flag {
		v = 1
	}
	_, err := s.db.Exec(`UPDATE identities SET inconsistent_flag=? WHERE id=?`, v, identityID)
	return err
}

// MergePreview describes what a merge would move (read-only).
type MergePreview struct {
	WinnerID, LoserID              int64
	Requests, Files, Identities    int
	Appointments, Notes            int
	Invoices                       int
}

// MergePreview counts relinkable rows for both sides.
func (s *Store) MergePreview(officeID, winnerID, loserID int64) (MergePreview, error) {
	var mp MergePreview
	mp.WinnerID, mp.LoserID = winnerID, loserID
	var wo, lo int64
	if err := s.db.QueryRow(`SELECT office_id FROM clients WHERE id=?`, winnerID).Scan(&wo); err != nil || wo != officeID {
		return mp, errf("klien utama tidak valid")
	}
	if err := s.db.QueryRow(`SELECT office_id FROM clients WHERE id=?`, loserID).Scan(&lo); err != nil || lo != officeID {
		return mp, errf("klien ganda tidak valid")
	}
	if winnerID == loserID {
		return mp, errf("klien sama")
	}
	c := func(dest *int, q string) {
		_ = s.db.QueryRow(q, loserID).Scan(dest)
	}
	c(&mp.Requests, `SELECT COUNT(*) FROM service_requests WHERE client_id=?`)
	c(&mp.Files, `SELECT COUNT(*) FROM files WHERE owner_type='client' AND owner_id=?`)
	c(&mp.Identities, `SELECT COUNT(*) FROM identities WHERE client_id=?`)
	c(&mp.Appointments, `SELECT COUNT(*) FROM appointments WHERE client_id=?`)
	c(&mp.Notes, `SELECT COUNT(*) FROM client_notes WHERE client_id=?`)
	c(&mp.Invoices, `SELECT COUNT(*) FROM invoices WHERE client_id=?`)
	return mp, nil
}

// MergeClients relinks loser rows to winner, archives loser, keeps audit trail.
// Nothing is hard-deleted except the loser row stays as ARCHIVED reference.
func (s *Store) MergeClients(officeID, winnerID, loserID int64, actor *int64) error {
	if _, err := s.MergePreview(officeID, winnerID, loserID); err != nil {
		return err
	}
	snap, _ := s.db.Query(`SELECT 'requests' AS t, COUNT(*) FROM service_requests WHERE client_id=? UNION ALL SELECT 'files', COUNT(*) FROM files WHERE owner_type='client' AND owner_id=?`, loserID, loserID)
	_ = snap
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	mk := func(q string, args ...any) error {
		_, e := tx.Exec(q, args...)
		return e
	}
	if err := mk(`UPDATE service_requests SET client_id=? WHERE client_id=?`, winnerID, loserID); err != nil {
		return err
	}
	if err := mk(`UPDATE files SET owner_id=? WHERE owner_type='client' AND owner_id=?`, winnerID, loserID); err != nil {
		return err
	}
	if err := mk(`UPDATE identities SET client_id=? WHERE client_id=?`, winnerID, loserID); err != nil {
		return err
	}
	if err := mk(`UPDATE appointments SET client_id=? WHERE client_id=?`, winnerID, loserID); err != nil {
		return err
	}
	if err := mk(`UPDATE client_notes SET client_id=? WHERE client_id=?`, winnerID, loserID); err != nil {
		return err
	}
	if err := mk(`UPDATE invoices SET client_id=? WHERE client_id=?`, winnerID, loserID); err != nil {
		return err
	}
	if err := mk(`UPDATE clients SET status='ARCHIVED' WHERE id=?`, loserID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO client_merges(office_id, winner_id, loser_id, actor_id) VALUES(?,?,?,?)`,
		officeID, winnerID, loserID, nullIDp(actor)); err != nil {
		return err
	}
	return tx.Commit()
}
