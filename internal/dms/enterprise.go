package dms

// DMS enterprise: custom fields, approval matrix, quotas, analytics,
// confidentiality gating (server-side). No fake integrations.

import (
	"database/sql"
	"errors"
	"strings"
)

// ---- custom fields ----

// CustomField is an office-defined metadata slot.
type CustomField struct {
	ID                  int64
	Name, Type, Options string
	Required            bool
	Position            int
}

// CustomFields lists slots in order.
func (s *Store) CustomFields(officeID int64) ([]CustomField, error) {
	rows, err := s.db.Query(`SELECT id, name, ftype, options, required, position
		FROM doc_custom_fields WHERE office_id=? ORDER BY position, id`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CustomField
	for rows.Next() {
		var f CustomField
		var req int
		_ = rows.Scan(&f.ID, &f.Name, &f.Type, &f.Options, &req, &f.Position)
		f.Required = req == 1
		out = append(out, f)
	}
	return out, rows.Err()
}

// SaveCustomField creates/updates a slot (type + options validated).
func (s *Store) SaveCustomField(officeID int64, f CustomField) (int64, error) {
	switch f.Type {
	case "text", "number", "date", "select", "bool":
	default:
		return 0, errors.New("tipe field tidak dikenal")
	}
	if strings.TrimSpace(f.Name) == "" {
		return 0, errors.New("nama field wajib")
	}
	if f.ID == 0 {
		res, err := s.db.Exec(`INSERT INTO doc_custom_fields(office_id, name, ftype, options, required, position)
			VALUES(?,?,?,?,?,?)`, officeID, f.Name, f.Type, f.Options, boolInt(f.Required), f.Position)
		if err != nil {
			return 0, dupErrDMS(err)
		}
		return res.LastInsertId()
	}
	var own int64
	if err := s.db.QueryRow(`SELECT office_id FROM doc_custom_fields WHERE id=?`, f.ID).Scan(&own); err != nil || own != officeID {
		return 0, errors.New("field tidak ditemukan")
	}
	_, err := s.db.Exec(`UPDATE doc_custom_fields SET name=?, ftype=?, options=?, required=?, position=? WHERE id=?`,
		f.Name, f.Type, f.Options, boolInt(f.Required), f.Position, f.ID)
	return f.ID, err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func dupErrDMS(err error) error {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return errors.New("nama field sudah dipakai")
	}
	return err
}

// SetFieldValues replaces all custom values for a file (validated per slot).
func (s *Store) SetFieldValues(officeID, fileID int64, vals map[int64]string) error {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return err
	}
	fields, err := s.CustomFields(officeID)
	if err != nil {
		return err
	}
	byID := map[int64]CustomField{}
	for _, f := range fields {
		byID[f.ID] = f
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM doc_field_values WHERE file_id=?`, fileID); err != nil {
		return err
	}
	for fid, v := range vals {
		f, ok := byID[fid]
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if v == "" {
			if f.Required {
				return errors.New("field wajib: " + f.Name)
			}
			continue
		}
		switch f.Type {
		case "number":
			if !isNumber(v) {
				return errors.New("bukan angka: " + f.Name)
			}
		case "date":
			if len(v) != 10 || v[4] != '-' || v[7] != '-' {
				return errors.New("tanggal harus YYYY-MM-DD: " + f.Name)
			}
		case "bool":
			if v != "0" && v != "1" {
				return errors.New("boolean harus 0/1: " + f.Name)
			}
		case "select":
			allowed := false
			for _, o := range strings.Split(f.Options, ",") {
				if strings.TrimSpace(o) == v {
					allowed = true
					break
				}
			}
			if !allowed {
				return errors.New("pilihan tidak dikenal: " + f.Name)
			}
		}
		if _, err := tx.Exec(`INSERT INTO doc_field_values(file_id, field_id, value) VALUES(?,?,?)`, fileID, fid, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func isNumber(v string) bool {
	if v == "" {
		return false
	}
	dot := false
	for i, ch := range v {
		if ch >= '0' && ch <= '9' {
			continue
		}
		if ch == '.' && i > 0 && !dot {
			dot = true
			continue
		}
		return false
	}
	return true
}

// FieldValues returns saved custom values keyed by field id.
func (s *Store) FieldValues(fileID int64) (map[int64]string, error) {
	rows, err := s.db.Query(`SELECT field_id, value FROM doc_field_values WHERE file_id=?`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var fid int64
		var v string
		_ = rows.Scan(&fid, &v)
		out[fid] = v
	}
	return out, rows.Err()
}

// ---- approval matrix ----

// ApprovalStep is one matrix step.
type ApprovalStep struct {
	Order int
	Role  string
}

// ApprovalChain returns the office default chain.
func (s *Store) ApprovalChain(officeID int64) ([]ApprovalStep, error) {
	rows, err := s.db.Query(`SELECT step_order, role_required FROM doc_approval_chains WHERE office_id=? ORDER BY step_order`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ApprovalStep
	for rows.Next() {
		var st ApprovalStep
		_ = rows.Scan(&st.Order, &st.Role)
		out = append(out, st)
	}
	return out, rows.Err()
}

// SaveChain replaces the office chain (roles must exist; max 5 steps).
func (s *Store) SaveChain(officeID int64, roles []string) error {
	if len(roles) > 5 {
		return errors.New("maksimal 5 tingkat")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM doc_approval_chains WHERE office_id=?`, officeID); err != nil {
		return err
	}
	for i, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		switch r {
		case "super_admin", "office_admin", "notary", "substitute_notary", "staff", "finance", "doc_operator", "auditor":
		default:
			return errors.New("peran tak dikenal: " + r)
		}
		if _, err := tx.Exec(`INSERT INTO doc_approval_chains(office_id, step_order, role_required) VALUES(?,?,?)`, officeID, i+1, r); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// EnsureApprovals seeds pending rows for a file from the chain (idempotent).
func (s *Store) EnsureApprovals(officeID, fileID int64) error {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return err
	}
	chain, err := s.ApprovalChain(officeID)
	if err != nil || len(chain) == 0 {
		return err
	}
	for _, st := range chain {
		var n int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM doc_approvals WHERE file_id=? AND step_order=?`, fileID, st.Order).Scan(&n)
		if n == 0 {
			if _, err := s.db.Exec(`INSERT INTO doc_approvals(office_id, file_id, step_order, role_required) VALUES(?,?,?,?)`,
				officeID, fileID, st.Order, st.Role); err != nil {
				return err
			}
		}
	}
	return nil
}

// DecideApproval records one step decision (role-checked by caller via Has).
func (s *Store) DecideApproval(officeID, fileID int64, step int, approver int64, approve bool, note string) error {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return err
	}
	st := "APPROVED"
	if !approve {
		st = "REJECTED"
	}
	res, err := s.db.Exec(`UPDATE doc_approvals SET approver_id=?, status=?, note=?, decided_at=UTC_TIMESTAMP()
		WHERE file_id=? AND step_order=? AND status='PENDING'`, approver, st, note, fileID, step)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("langkah tidak menunggu keputusan")
	}
	return nil
}

// ApprovalState reports pending/total for a file.
func (s *Store) ApprovalState(fileID int64) (pending, total int) {
	_ = s.db.QueryRow(`SELECT COUNT(*), SUM(status='PENDING') FROM doc_approvals WHERE file_id=?`, fileID).Scan(&total, &pending)
	return pending, total
}

// ---- quotas & analytics ----

// StorageUsage sums private bytes for an office.
func (s *Store) StorageUsage(officeID int64) int64 {
	var n sql.NullInt64
	_ = s.db.QueryRow(`SELECT SUM(size_bytes) FROM files WHERE office_id=?`, officeID).Scan(&n)
	return n.Int64
}

// WorkflowStats counts files per workflow status.
func (s *Store) WorkflowStats(officeID int64) (map[string]int, error) {
	rows, err := s.db.Query(`SELECT workflow_status, COUNT(*) FROM files WHERE office_id=? GROUP BY workflow_status`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var st string
		var n int
		_ = rows.Scan(&st, &n)
		out[st] = n
	}
	return out, rows.Err()
}

// StorageByMIME aggregates bytes per MIME.
func (s *Store) StorageByMIME(officeID int64) (map[string]int64, error) {
	rows, err := s.db.Query(`SELECT mime, SUM(size_bytes) FROM files WHERE office_id=? GROUP BY mime ORDER BY 2 DESC LIMIT 10`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var m string
		var n int64
		_ = rows.Scan(&m, &n)
		out[m] = n
	}
	return out, rows.Err()
}

// ---- confidentiality ----

// CanAccess reports whether a role may open content at the given level.
// PUBLIC/INTERNAL: all staff. CONFIDENTIAL: notary chain + finance + auditor(read).
// RESTRICTED: notary chain only. Clients never (portal uses canViewDeed paths).
func CanAccess(role, level string) bool {
	switch level {
	case "", "PUBLIC", "INTERNAL":
		return role != "client" && role != ""
	case "CONFIDENTIAL":
		switch role {
		case "super_admin", "office_admin", "notary", "substitute_notary", "finance", "auditor":
			return true
		}
		return false
	case "RESTRICTED":
		switch role {
		case "super_admin", "office_admin", "notary", "substitute_notary":
			return true
		}
		return false
	}
	return false
}
