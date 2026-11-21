// Package dms implements enterprise document lifecycle metadata on MySQL.
// Binaries stay in private storage (internal/storage); this package never
// touches file bytes, only refs/hashes. Every method is office-scoped;
// fail-closed: unknown office or missing row returns an error, never data.
package dms

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
)

// Allowed workflow transitions (spec F10).
var wfNext = map[string][]string{
	"RECEIVED":          {"CLASSIFICATION"},
	"CLASSIFICATION":    {"ADMIN_REVIEW"},
	"ADMIN_REVIEW":      {"VERIFICATION", "REVISION_REQUIRED"},
	"VERIFICATION":      {"APPROVAL", "REVISION_REQUIRED"},
	"REVISION_REQUIRED": {"ADMIN_REVIEW"},
	"APPROVAL":          {"APPROVED", "REJECTED"},
	"APPROVED":          {"FINALIZED"},
	"FINALIZED":         {"ARCHIVED"},
	"ARCHIVED":          {},
	"REJECTED":          {},
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) officeOfFile(fileID, officeID int64) error {
	var oid int64
	if err := s.db.QueryRow(`SELECT office_id FROM files WHERE id=?`, fileID).Scan(&oid); err != nil {
		return err
	}
	if oid != officeID {
		return errors.New("dokumen tidak ditemukan")
	}
	return nil
}

// ---- categories (configurable per office) ----

type Category struct {
	ID         int64
	Name, Desc string
}

func (s *Store) Categories(officeID int64) ([]Category, error) {
	rows, err := s.db.Query(`SELECT id, name, description FROM doc_categories WHERE office_id=? ORDER BY name`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		var d sql.NullString
		_ = rows.Scan(&c.ID, &c.Name, &d)
		if d.Valid {
			c.Desc = d.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCategory(officeID int64, name, desc string) (int64, error) {
	if name == "" {
		return 0, errors.New("nama kategori wajib diisi")
	}
	res, err := s.db.Exec(`INSERT INTO doc_categories(office_id, name, description) VALUES(?,?,?)`, officeID, name, desc)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ---- versions ----

type Version struct {
	No        int
	Ref, Hash string
	Size      int64
	Note      string
	By        *int64
	At        string
}

func (s *Store) AddVersion(officeID, fileID int64, ref, sha string, size int64, note string, by *int64) (int, error) {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return 0, err
	}
	var next int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(version_no),0)+1 FROM doc_versions WHERE file_id=?`, fileID).Scan(&next)
	if _, err := s.db.Exec(`INSERT INTO doc_versions(file_id, version_no, ref, sha256, size_bytes, note, created_by)
		VALUES(?,?,?,?,?,?,?)`, fileID, next, ref, sha, size, note, by); err != nil {
		return 0, err
	}
	return next, nil
}

func (s *Store) Versions(officeID, fileID int64) ([]Version, error) {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT version_no, ref, sha256, size_bytes, note, created_by, created_at
		FROM doc_versions WHERE file_id=? ORDER BY version_no`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Version
	for rows.Next() {
		var v Version
		var note sql.NullString
		_ = rows.Scan(&v.No, &v.Ref, &v.Hash, &v.Size, &note, &v.By, &v.At)
		if note.Valid {
			v.Note = note.String
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ---- relations ----

type Relation struct {
	ID          int64
	FileID      int64
	RelatedType string
	RelatedID   int64
	Relation    string
	At          string
}

func (s *Store) AddRelation(officeID, fileID int64, rtype string, rid int64, rel string) error {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return err
	}
	switch rtype {
	case "client", "request", "deed", "repertorium", "will", "legalization", "invoice", "service", "company":
	default:
		return errors.New("tipe relasi tidak dikenal")
	}
	if rel == "" {
		rel = "lampiran"
	}
	_, err := s.db.Exec(`INSERT INTO doc_relations(file_id, related_type, related_id, relation)
		VALUES(?,?,?,?)`, fileID, rtype, rid, rel)
	return err
}

func (s *Store) RelationsOf(officeID, fileID int64) ([]Relation, error) {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, file_id, related_type, related_id, relation, created_at
		FROM doc_relations WHERE file_id=? ORDER BY id`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Relation
	for rows.Next() {
		var x Relation
		_ = rows.Scan(&x.ID, &x.FileID, &x.RelatedType, &x.RelatedID, &x.Relation, &x.At)
		out = append(out, x)
	}
	return out, rows.Err()
}

// ByRelated finds files linked to a business object (reverse lookup).
func (s *Store) ByRelated(officeID int64, rtype string, rid int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT r.file_id FROM doc_relations r JOIN files f ON f.id=r.file_id
		WHERE f.office_id=? AND r.related_type=? AND r.related_id=? ORDER BY r.id`, officeID, rtype, rid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		out = append(out, id)
	}
	return out, rows.Err()
}

// ---- workflow ----

type WFEvent struct {
	From, To, Note, Actor, At string
}

func (s *Store) WorkflowStatus(officeID, fileID int64) (string, error) {
	var st string
	var oid int64
	if err := s.db.QueryRow(`SELECT workflow_status, office_id FROM files WHERE id=?`, fileID).Scan(&st, &oid); err != nil {
		return "", err
	}
	if oid != officeID {
		return "", errors.New("dokumen tidak ditemukan")
	}
	return st, nil
}

// Move advances workflow with separation-of-duties: final APPROVAL must come
// from someone other than the uploader.
func (s *Store) Move(officeID, fileID int64, to string, actorID *int64, note string) error {
	var cur, createdBy sql.NullString
	var cb sql.NullInt64
	var oid int64
	if err := s.db.QueryRow(`SELECT workflow_status, office_id, created_by FROM files WHERE id=?`, fileID).
		Scan(&cur, &oid, &cb); err != nil {
		return err
	}
	_ = createdBy
	if oid != officeID {
		return errors.New("dokumen tidak ditemukan")
	}
	ok := false
	for _, n := range wfNext[cur.String] {
		if n == to {
			ok = true
		}
	}
	if !ok {
		return errors.New("transisi workflow tidak valid: " + cur.String + " → " + to)
	}
	if to == "APPROVED" && actorID != nil && cb.Valid && *actorID == cb.Int64 {
		return errors.New("pemisahan tugas: pengunggah tidak boleh menyetujui sendiri")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE files SET workflow_status=? WHERE id=?`, to, fileID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO doc_workflow_events(file_id, actor_id, from_status, to_status, note)
		VALUES(?,?,?,?,?)`, fileID, actorID, cur.String, to, note); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Timeline(officeID, fileID int64) ([]WFEvent, error) {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT e.from_status, e.to_status, e.note, u.name, e.created_at
		FROM doc_workflow_events e LEFT JOIN users u ON u.id=e.actor_id
		WHERE e.file_id=? ORDER BY e.id`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WFEvent
	for rows.Next() {
		var e WFEvent
		var from, note, actor sql.NullString
		_ = rows.Scan(&from, &e.To, &note, &actor, &e.At)
		if from.Valid {
			e.From = from.String
		}
		if note.Valid {
			e.Note = note.String
		}
		if actor.Valid {
			e.Actor = actor.String
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---- OCR jobs ----

type OCRJob struct {
	ID, FileID int64
	Status     string
	Attempts   int
	LastErr    string
}

func (s *Store) EnqueueOCR(officeID, fileID int64) (int64, error) {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO doc_ocr_jobs(file_id, status) VALUES(?,'QUEUED')`, fileID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Claim pops one QUEUED job atomically (SKIP LOCKED for multi-worker).
func (s *Store) ClaimOCR() (OCRJob, bool) {
	tx, err := s.db.Begin()
	if err != nil {
		return OCRJob{}, false
	}
	defer tx.Rollback()
	var j OCRJob
	err = tx.QueryRow(`SELECT id, file_id, status, attempts FROM doc_ocr_jobs
		WHERE status='QUEUED' ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&j.ID, &j.FileID, &j.Status, &j.Attempts)
	if err != nil {
		return OCRJob{}, false
	}
	if _, err := tx.Exec(`UPDATE doc_ocr_jobs SET status='RUNNING', attempts=attempts+1 WHERE id=?`, j.ID); err != nil {
		return OCRJob{}, false
	}
	if err := tx.Commit(); err != nil {
		return OCRJob{}, false
	}
	j.Status = "RUNNING"
	return j, true
}

func (s *Store) FinishOCR(jobID int64, ok bool, errMsg string) error {
	if ok {
		_, err := s.db.Exec(`UPDATE doc_ocr_jobs SET status='DONE', last_error=NULL WHERE id=?`, jobID)
		return err
	}
	_, err := s.db.Exec(`UPDATE doc_ocr_jobs SET status=IF(attempts>=3,'FAILED','QUEUED'), last_error=? WHERE id=?`, errMsg, jobID)
	return err
}

func (s *Store) OCRQueue(officeID int64) ([]OCRJob, error) {
	rows, err := s.db.Query(`SELECT j.id, j.file_id, j.status, j.attempts, j.last_error
		FROM doc_ocr_jobs j JOIN files f ON f.id=j.file_id
		WHERE f.office_id=? ORDER BY j.id DESC LIMIT 50`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OCRJob
	for rows.Next() {
		var j OCRJob
		var e sql.NullString
		_ = rows.Scan(&j.ID, &j.FileID, &j.Status, &j.Attempts, &e)
		if e.Valid {
			j.LastErr = e.String
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// ---- favorites ----

func (s *Store) ToggleFavorite(userID, fileID, officeID int64) (bool, error) {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return false, err
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM doc_favorites WHERE user_id=? AND file_id=?`, userID, fileID).Scan(&n)
	if n > 0 {
		_, err := s.db.Exec(`DELETE FROM doc_favorites WHERE user_id=? AND file_id=?`, userID, fileID)
		return false, err
	}
	_, err := s.db.Exec(`INSERT INTO doc_favorites(user_id, file_id) VALUES(?,?)`, userID, fileID)
	return true, err
}

// ---- shares (expiring, limited, revocable) ----

type Share struct {
	ID                      int64
	FileID                  int64
	Token                   string
	ExpiresAt               string
	MaxDownloads, Downloads int
	Revoked                 bool
}

func (s *Store) CreateShare(officeID, fileID int64, hours int, maxDl int64, by *int64) (Share, error) {
	var sh Share
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return sh, err
	}
	if hours <= 0 || hours > 24*30 {
		return sh, errors.New("masa berlaku 1 jam–30 hari")
	}
	if maxDl <= 0 || maxDl > 100 {
		return sh, errors.New("batas unduhan 1–100")
	}
	tok := make([]byte, 16)
	if _, err := rand.Read(tok); err != nil {
		return sh, err
	}
	sh.Token = hex.EncodeToString(tok)
	res, err := s.db.Exec(`INSERT INTO doc_shares(file_id, token, expires_at, max_downloads, created_by)
		VALUES(?,?,DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? HOUR),?,?)`, fileID, sh.Token, hours, maxDl, by)
	if err != nil {
		return sh, err
	}
	sh.ID, _ = res.LastInsertId()
	sh.FileID = fileID
	return sh, nil
}

func (s *Store) UseShare(token string) (int64, error) {
	var id, fid int64
	var exp string
	var maxDl, dls int
	var revoked bool
	err := s.db.QueryRow(`SELECT id, file_id, expires_at, max_downloads, downloads, revoked
		FROM doc_shares WHERE token=?`, token).Scan(&id, &fid, &exp, &maxDl, &dls, &revoked)
	if err != nil {
		return 0, errors.New("tautan tidak valid")
	}
	var expired int
	_ = s.db.QueryRow(`SELECT expires_at < UTC_TIMESTAMP() FROM doc_shares WHERE id=?`, id).Scan(&expired)
	if revoked || expired == 1 || dls >= maxDl {
		return 0, errors.New("tautan kedaluwarsa/dicabut/habis")
	}
	if _, err := s.db.Exec(`UPDATE doc_shares SET downloads=downloads+1 WHERE id=?`, id); err != nil {
		return 0, err
	}
	return fid, nil
}

func (s *Store) RevokeShare(officeID, shareID int64) error {
	res, err := s.db.Exec(`UPDATE doc_shares s JOIN files f ON f.id=s.file_id
		SET s.revoked=1 WHERE s.id=? AND f.office_id=?`, shareID, officeID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("tautan tidak ditemukan")
	}
	return nil
}

// ---- archive ----

type Location struct {
	ID                     int64
	Code, Room, Shelf, Box string
}

func (s *Store) Locations(officeID int64) ([]Location, error) {
	rows, err := s.db.Query(`SELECT id, code, room, shelf, box FROM doc_archive_locations WHERE office_id=? ORDER BY code`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Location
	for rows.Next() {
		var l Location
		var r, sh, b sql.NullString
		_ = rows.Scan(&l.ID, &l.Code, &r, &sh, &b)
		if r.Valid {
			l.Room = r.String
		}
		if sh.Valid {
			l.Shelf = sh.String
		}
		if b.Valid {
			l.Box = b.String
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) CreateLocation(officeID int64, code, room, shelf, box string) (int64, error) {
	if code == "" {
		return 0, errors.New("kode lokasi wajib diisi")
	}
	res, err := s.db.Exec(`INSERT INTO doc_archive_locations(office_id, code, room, shelf, box) VALUES(?,?,?,?,?)`,
		officeID, code, room, shelf, box)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) MoveArchive(officeID, fileID int64, locID *int64, movement string, actorID *int64, note string) error {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return err
	}
	switch movement {
	case "STORE", "RETRIEVE", "RETURN", "TRANSFER":
	default:
		return errors.New("jenis perpindahan tidak dikenal")
	}
	if locID != nil {
		var n int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM doc_archive_locations WHERE id=? AND office_id=?`, *locID, officeID).Scan(&n)
		if n == 0 {
			return errors.New("lokasi tidak ditemukan")
		}
	}
	_, err := s.db.Exec(`INSERT INTO doc_archive_movements(file_id, location_id, movement, actor_id, note)
		VALUES(?,?,?,?,?)`, fileID, locID, movement, actorID, note)
	return err
}

func (s *Store) MoveHistory(officeID, fileID int64) ([]map[string]any, error) {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT m.movement, l.code, u.name, m.note, m.created_at
		FROM doc_archive_movements m LEFT JOIN doc_archive_locations l ON l.id=m.location_id
		LEFT JOIN users u ON u.id=m.actor_id WHERE m.file_id=? ORDER BY m.id`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var mv string
		var loc, actor, note sql.NullString
		var at string
		_ = rows.Scan(&mv, &loc, &actor, &note, &at)
		out = append(out, map[string]any{"Move": mv, "Loc": loc.String, "Actor": actor.String, "Note": note.String, "At": at})
	}
	return out, rows.Err()
}

// ---- retention & legal hold ----

func (s *Store) ApplyHold(officeID, fileID int64, reason string, by *int64) error {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return err
	}
	if reason == "" {
		return errors.New("alasan hold wajib diisi")
	}
	_, err := s.db.Exec(`INSERT INTO doc_legal_holds(file_id, office_id, reason, applied_by) VALUES(?,?,?,?)`,
		fileID, officeID, reason, by)
	return err
}

func (s *Store) ReleaseHold(officeID, holdID int64) error {
	res, err := s.db.Exec(`UPDATE doc_legal_holds SET released_at=UTC_TIMESTAMP()
		WHERE id=? AND office_id=? AND released_at IS NULL`, holdID, officeID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("hold tidak ditemukan")
	}
	return nil
}

func (s *Store) ActiveHolds(officeID, fileID int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM doc_legal_holds
		WHERE file_id=? AND office_id=? AND released_at IS NULL`, fileID, officeID).Scan(&n)
	return n, err
}

// RequestDisposition records RETAIN/DESTROY decisions; DESTROYED requires no active hold.
func (s *Store) RequestDisposition(officeID, fileID int64, action string, by *int64, note string) error {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return err
	}
	switch action {
	case "RETAIN", "DESTROY_REQUEST", "DESTROYED":
	default:
		return errors.New("aksi disposisi tidak dikenal")
	}
	if action == "DESTROYED" {
		if n, _ := s.ActiveHolds(officeID, fileID); n > 0 {
			return errors.New("legal hold aktif: penghancuran DITOLAK")
		}
	}
	_, err := s.db.Exec(`INSERT INTO doc_dispositions(file_id, office_id, action, decided_by, note)
		VALUES(?,?,?,?,?)`, fileID, officeID, action, by, note)
	return err
}

// ---- access log ----

func (s *Store) LogAccess(fileID int64, userID *int64, action, ip, ua string) {
	_, _ = s.db.Exec(`INSERT INTO doc_access_logs(file_id, user_id, action, ip, user_agent)
		VALUES(?,?,?,?,?)`, fileID, userID, action, ip, ua)
}

func (s *Store) AccessLog(officeID, fileID int64) ([]map[string]any, error) {
	if err := s.officeOfFile(fileID, officeID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT u.name, l.action, l.ip, l.created_at FROM doc_access_logs l
		LEFT JOIN users u ON u.id=l.user_id WHERE l.file_id=? ORDER BY l.id DESC LIMIT 100`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var actor sql.NullString
		var act, ip, at string
		_ = rows.Scan(&actor, &act, &ip, &at)
		out = append(out, map[string]any{"Actor": actor.String, "Action": act, "IP": ip, "At": at})
	}
	return out, rows.Err()
}

// ---- search index ----

func (s *Store) IndexContent(fileID int64, text string) error {
	if len(text) > 200000 {
		text = text[:200000]
	}
	_, err := s.db.Exec(`INSERT INTO doc_search_index(file_id, content) VALUES(?,?)
		ON DUPLICATE KEY UPDATE content=VALUES(content)`, fileID, text)
	return err
}

// Search runs FULLTEXT first, LIKE fallback; always office-scoped.
// Restricted docs are excluded unless canSeeRestricted.
func (s *Store) Search(officeID int64, q string, canSeeRestricted bool, limit int) ([]int64, error) {
	if q = trimQ(q); q == "" {
		return nil, errors.New("kata kunci kosong")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	restr := ""
	if !canSeeRestricted {
		restr = ` AND f.confidentiality != 'RESTRICTED'`
	}
	var ids []int64
	rows, err := s.db.Query(`SELECT DISTINCT f.id FROM files f
		LEFT JOIN doc_search_index i ON i.file_id=f.id
		WHERE f.office_id=?`+restr+` AND MATCH(i.content) AGAINST(? IN NATURAL LANGUAGE MODE)
		LIMIT ?`, officeID, q, limit)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			_ = rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
	}
	if len(ids) > 0 {
		return ids, nil
	}
	like := "%" + q + "%"
	rows2, err := s.db.Query(`SELECT DISTINCT f.id FROM files f
		LEFT JOIN doc_search_index i ON i.file_id=f.id
		WHERE f.office_id=?`+restr+` AND (f.original_name LIKE ? OR f.doc_number LIKE ? OR i.content LIKE ?)
		LIMIT ?`, officeID, like, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var id int64
		_ = rows2.Scan(&id)
		ids = append(ids, id)
	}
	return ids, rows2.Err()
}

func trimQ(q string) string {
	if len(q) > 200 {
		q = q[:200]
	}
	out := ""
	for _, r := range q {
		if r == '%' || r == '_' || r == '\\' {
			continue
		}
		out += string(r)
	}
	return out
}

// ---- checklist ----

// AutoChecklist copies template items for a service into MISSING records for a new request.
func (s *Store) AutoChecklist(officeID, requestID, serviceID int64) int {
	var code string
	if err := s.db.QueryRow(`SELECT code FROM services WHERE id=?`, serviceID).Scan(&code); err != nil {
		return 0
	}
	rows, err := s.db.Query(`SELECT i.id FROM doc_checklist_items i
		JOIN doc_checklist_templates t ON t.id=i.template_id
		WHERE t.office_id=? AND t.service_code=?`, officeID, code)
	if err != nil {
		return 0
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var itemID int64
		_ = rows.Scan(&itemID)
		if _, err := s.db.Exec(`INSERT IGNORE INTO doc_checklist_records(office_id, request_id, item_id)
			VALUES(?,?,?)`, officeID, requestID, itemID); err == nil {
			n++
		}
	}
	return n
}

type ChecklistItem struct {
	ID       int64
	Label    string
	Required bool
	Status   string
	Note     string
}

func (s *Store) CreateChecklistTemplate(officeID int64, serviceCode, name string, items []string) (int64, error) {
	if name == "" || len(items) == 0 {
		return 0, errors.New("nama dan minimal satu item wajib diisi")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO doc_checklist_templates(office_id, service_code, name) VALUES(?,?,?)`,
		officeID, serviceCode, name)
	if err != nil {
		return 0, err
	}
	tid, _ := res.LastInsertId()
	for _, it := range items {
		if it == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO doc_checklist_items(template_id, label) VALUES(?,?)`, tid, it); err != nil {
			return 0, err
		}
	}
	return tid, tx.Commit()
}

func (s *Store) ChecklistForRequest(officeID, requestID int64) ([]ChecklistItem, error) {
	rows, err := s.db.Query(`SELECT i.id, i.label, i.is_required,
		COALESCE(r.status,'MISSING'), COALESCE(r.note,'')
		FROM doc_checklist_items i
		JOIN doc_checklist_templates t ON t.id=i.template_id
		JOIN service_requests sr ON sr.service_id=(SELECT id FROM services WHERE code=t.service_code LIMIT 1)
		LEFT JOIN doc_checklist_records r ON r.item_id=i.id AND r.request_id=sr.id
		WHERE t.office_id=? AND sr.id=? AND sr.office_id=?
		ORDER BY i.id`, officeID, requestID, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChecklistItem
	for rows.Next() {
		var c ChecklistItem
		var req int
		_ = rows.Scan(&c.ID, &c.Label, &req, &c.Status, &c.Note)
		c.Required = req == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) SetChecklistItem(officeID, requestID, itemID int64, status string, by *int64, note string) error {
	var oid int64
	if err := s.db.QueryRow(`SELECT office_id FROM service_requests WHERE id=?`, requestID).Scan(&oid); err != nil {
		return err
	}
	if oid != officeID {
		return errors.New("permohonan tidak ditemukan")
	}
	switch status {
	case "MISSING", "RECEIVED", "VERIFIED", "REJECTED", "NEEDS_REVISION":
	default:
		return errors.New("status checklist tidak dikenal")
	}
	_, err := s.db.Exec(`INSERT INTO doc_checklist_records(office_id, request_id, item_id, status, verified_by, verified_at, note)
		VALUES(?,?,?,?,?,IF(? IN ('VERIFIED','REJECTED'),UTC_TIMESTAMP(),NULL),?)
		ON DUPLICATE KEY UPDATE status=VALUES(status), verified_by=VALUES(verified_by),
		verified_at=VALUES(verified_at), note=VALUES(note)`,
		officeID, requestID, itemID, status, by, status, note)
	return err
}

// ---- dashboard stats (all real queries) ----

type Stats struct {
	Total, PendingVerify, PendingApproval, Missing, Expiring, OCRQueued int
	StorageBytes                                                        int64
	Duplicates                                                          int
}

func (s *Store) Stats(officeID int64) (Stats, error) {
	var st Stats
	q := func(dest any, query string, args ...any) {
		_ = s.db.QueryRow(query, args...).Scan(dest)
	}
	q(&st.Total, `SELECT COUNT(*) FROM files WHERE office_id=?`, officeID)
	q(&st.PendingVerify, `SELECT COUNT(*) FROM files WHERE office_id=? AND verification_status='PENDING'`, officeID)
	q(&st.PendingApproval, `SELECT COUNT(*) FROM files WHERE office_id=? AND workflow_status='APPROVAL'`, officeID)
	q(&st.Missing, `SELECT COUNT(*) FROM doc_checklist_records WHERE office_id=? AND status='MISSING'`, officeID)
	q(&st.Expiring, `SELECT COUNT(*) FROM files WHERE office_id=? AND expiry_date IS NOT NULL AND expiry_date <= DATE_ADD(CURDATE(), INTERVAL 30 DAY)`, officeID)
	q(&st.OCRQueued, `SELECT COUNT(*) FROM doc_ocr_jobs j JOIN files f ON f.id=j.file_id WHERE f.office_id=? AND j.status IN ('QUEUED','RUNNING')`, officeID)
	q(&st.StorageBytes, `SELECT COALESCE(SUM(size_bytes),0) FROM files WHERE office_id=?`, officeID)
	q(&st.Duplicates, `SELECT COUNT(*) FROM (SELECT sha256 FROM files WHERE office_id=? GROUP BY sha256 HAVING COUNT(*)>1) d`, officeID)
	return st, nil
}

// DuplicateCandidates lists sha256 groups with >1 file (review, never auto-delete).
func (s *Store) DuplicateCandidates(officeID int64) ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT f.sha256, GROUP_CONCAT(f.id), COUNT(*) FROM files f
		WHERE f.office_id=? GROUP BY f.sha256 HAVING COUNT(*)>1 LIMIT 50`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var sha, ids string
		var n int
		_ = rows.Scan(&sha, &ids, &n)
		out = append(out, map[string]any{"SHA": sha, "IDs": ids, "N": n})
	}
	return out, rows.Err()
}

// MergeDuplicates links dupIDs into canonicalID (same SHA-256, same office).
// No bytes are deleted — merged files stay downloadable; relations record
// DIGABUNG_KE so detail pages banner-redirect to the canonical record.
func (s *Store) MergeDuplicates(officeID, canonicalID int64, dupIDs []int64) (int, error) {
	var sha string
	if err := s.db.QueryRow(`SELECT sha256 FROM files WHERE id=? AND office_id=?`,
		canonicalID, officeID).Scan(&sha); err != nil {
		return 0, errors.New("kanonis tidak ditemukan di kantor ini")
	}
	if sha == "" {
		return 0, errors.New("kanonis tanpa hash — tolak gabung")
	}
	n := 0
	for _, id := range dupIDs {
		if id == canonicalID {
			continue
		}
		var dsha string
		if err := s.db.QueryRow(`SELECT sha256 FROM files WHERE id=? AND office_id=?`,
			id, officeID).Scan(&dsha); err != nil || dsha != sha {
			continue // beda isi/kantor — lewati jujur, bukan gagal total
		}
		var have int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM doc_relations WHERE file_id=? AND related_type='file' AND related_id=? AND relation='DIGABUNG_KE'`,
			id, canonicalID).Scan(&have)
		if have > 0 {
			continue
		}
		if _, err := s.db.Exec(`INSERT INTO doc_relations(file_id, related_type, related_id, relation)
			VALUES(?,'file',?,'DIGABUNG_KE')`, id, canonicalID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
func (s *Store) Expiring(officeID int64) ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT id, original_name, expiry_date FROM files
		WHERE office_id=? AND expiry_date IS NOT NULL AND expiry_date <= DATE_ADD(CURDATE(), INTERVAL 30 DAY)
		ORDER BY expiry_date LIMIT 50`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var name, exp string
		_ = rows.Scan(&id, &name, &exp)
		out = append(out, map[string]any{"ID": id, "Name": name, "Expiry": exp})
	}
	return out, rows.Err()
}

// SearchFiles returns file metas for ids (office-scoped, order preserved loosely).
func (s *Store) SearchFiles(officeID int64, ids []int64) ([]map[string]any, error) {
	var out []map[string]any
	for _, id := range ids {
		var name, mime, status string
		var size int64
		err := s.db.QueryRow(`SELECT original_name, mime, size_bytes, ocr_status FROM files
			WHERE id=? AND office_id=?`, id, officeID).Scan(&name, &mime, &size, &status)
		if err != nil {
			continue
		}
		out = append(out, map[string]any{"ID": id, "Name": name, "MIME": mime, "Size": size, "OCR": status})
		if len(out) >= 50 {
			break
		}
	}
	return out, nil
}

// AutoLinkRelations links a newly uploaded file to its owner context.
func (s *Store) AutoLink(officeID, fileID int64, ownerType string, ownerID int64, by *int64) error {
	fileOwner := map[string]string{"client": "client", "request": "request", "deed": "deed"}
	rtype, ok := fileOwner[ownerType]
	if !ok {
		return nil
	}
	return s.AddRelation(officeID, fileID, rtype, ownerID, "unggahan-awal")
}
