// Package remind — SLA workflow DMS (sla.go).
//
// SLAForWorkflow memetakan status workflow dokumen ke (dueDays, priority).
// OverdueDocs memindai dokumen kantor yang macet melewati SLA sejak aktivitas
// workflow terakhir (doc_workflow_events terbaru, jatuh kembali ke files.created_at
// karena tabel files tidak memiliki kolom updated_at).
package remind

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// SLAForWorkflow returns the SLA in days and priority for a DMS workflow status.
func SLAForWorkflow(status string) (dueDays int, priority string) {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "VERIFICATION":
		return 1, "HIGH"
	case "ADMIN_REVIEW":
		return 2, "MEDIUM"
	case "APPROVAL":
		return 2, "HIGH"
	case "REVISION_REQUIRED":
		return 3, "MEDIUM"
	default:
		return 7, "LOW"
	}
}

// OverdueDoc is one document stuck past its workflow SLA.
type OverdueDoc struct {
	ID         int64
	Name       string
	Workflow   string
	LastActive string
	SLADays    int
}

// OverdueDocs lists documents in VERIFICATION/ADMIN_REVIEW/APPROVAL whose last
// workflow activity (latest doc_workflow_events.created_at, fallback to
// files.created_at as files has no updated_at column) exceeds the per-status
// SLA interval.
func OverdueDocs(db *sql.DB, officeID int64) ([]OverdueDoc, error) {
	rows, err := db.Query(`
		SELECT f.id, COALESCE(f.display_name, f.original_name), f.workflow_status,
			COALESCE(MAX(e.created_at), f.created_at),
			CASE f.workflow_status
				WHEN 'VERIFICATION' THEN 1
				WHEN 'ADMIN_REVIEW' THEN 2
				WHEN 'APPROVAL' THEN 2
				ELSE 7
			END
		FROM files f
		LEFT JOIN doc_workflow_events e ON e.file_id = f.id
		WHERE f.office_id = ?
		  AND f.workflow_status IN ('VERIFICATION','ADMIN_REVIEW','APPROVAL')
		GROUP BY f.id
		HAVING COALESCE(MAX(e.created_at), MAX(f.created_at)) <
			DATE_SUB(NOW(), INTERVAL CASE f.workflow_status
				WHEN 'VERIFICATION' THEN 1
				WHEN 'ADMIN_REVIEW' THEN 2
				WHEN 'APPROVAL' THEN 2
				ELSE 7
			END DAY)
		ORDER BY 4 LIMIT 100`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OverdueDoc
	for rows.Next() {
		var d OverdueDoc
		if err := rows.Scan(&d.ID, &d.Name, &d.Workflow, &d.LastActive, &d.SLADays); err != nil {
			continue
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ---- SLA per layanan (permohonan) ----
//
// Format override per kantor (office_flags.sla_days_json, TEXT NULL):
//
//	{"PPAT-AJB": 5, "LEGALISASI": 3, "default": 10}
//
//   - Kunci = kode layanan persis seperti di services.code (case-insensitive,
//     spasi diabaikan). Contoh "ppat-ajb" cocok dengan "PPAT-AJB".
//   - Kunci khusus "default" (case-insensitive) mengganti angka fallback 7
//     untuk kode yang tidak tercantum.
//   - Nilai = target umur dalam hari, bilangan bulat > 0 dan <= 365.
//   - Kosong/NULL = tidak ada override, semua kode memakai ServiceSLA.
//   - JSON rusak / bukan objek / nilai tak valid -> seluruh override
//     diabaikan (fallback ke default lama), tidak pernah error ke pemanggil.
//
// Cara pakai: ServiceSLA(code) untuk default deterministik (tidak menyentuh
// DB); ServiceSLAFor(db, officeID, code) bila ingin override kantor berlaku
// (dashboard & laporan); ServiceOverdue(db, officeID) sudah memakai
// ServiceSLAFor sehingga override otomatis berlaku di laporan & dashboard.

// ServiceSLA returns the target age in days for a service code:
// LEGALISASI/WAARMERKING 2, AJB/SKMHT/APHT 7, PT-DIRIAN/YAYASAN/KOPERASI 14,
// WASIAT 3, AHU/OSS 10, default 7.
func ServiceSLA(code string) int {
	c := strings.ToUpper(strings.TrimSpace(code))
	switch {
	case strings.Contains(c, "LEGALISASI") || strings.Contains(c, "WAARMERKING"):
		return 2
	case strings.Contains(c, "AJB") || strings.Contains(c, "SKMHT") || strings.Contains(c, "APHT"):
		return 7
	case strings.HasPrefix(c, "PT-") || strings.HasPrefix(c, "CV-") || c == "FIRMA" ||
		strings.Contains(c, "YAYASAN") || strings.Contains(c, "KOPERASI"):
		return 14
	case strings.Contains(c, "WASIAT"):
		return 3
	case strings.Contains(c, "AHU") || strings.Contains(c, "OSS"):
		return 10
	default:
		return 7
	}
}

// ServiceOverdueItem is one open request older than its service SLA.
type ServiceOverdueItem struct {
	ID      int64
	Title   string
	Code    string
	Service string
	Status  string
	AgeDays int
	SLADays int
}

// ServiceOverdue lists OPEN requests (any status outside
// COMPLETED/REJECTED/CANCELLED) whose age in days exceeds the per-service
// target. The target honors the per-office override in
// office_flags.sla_days_json via ServiceSLAFor (tolerant: missing
// table/broken JSON falls back to ServiceSLA). Rows are oldest first,
// capped at 100.
func ServiceOverdue(db *sql.DB, officeID int64) ([]ServiceOverdueItem, error) {
	rows, err := db.Query(`
		SELECT r.id, r.title, s.code, s.name, r.status, DATEDIFF(NOW(), r.created_at)
		FROM service_requests r JOIN services s ON s.id=r.service_id
		WHERE r.office_id=? AND r.status NOT IN ('COMPLETED','REJECTED','CANCELLED')
		ORDER BY r.created_at LIMIT 200`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ServiceOverdueItem
	overrides := loadSLAOverrides(db, officeID)
	for rows.Next() {
		var it ServiceOverdueItem
		if err := rows.Scan(&it.ID, &it.Title, &it.Code, &it.Service, &it.Status, &it.AgeDays); err != nil {
			continue
		}
		it.SLADays = ResolveSLA(it.Code, overrides)
		if it.AgeDays > it.SLADays {
			out = append(out, it)
		}
		if len(out) >= 100 {
			break
		}
	}
	return out, rows.Err()
}

// ---- Override SLA per kantor (office_flags.sla_days_json) ----

// ParseSLAOverrides parses the raw sla_days_json value into a normalized
// override map (keys upper-cased/trimmed, including "DEFAULT").
//
// Valid input: JSON object mapping service code (or "default") to a positive
// integer number of days, e.g. `{"PPAT-AJB": 5, "default": 10}`.
// Empty/whitespace-only input yields (nil, nil) — no overrides.
// Any invalid input (broken JSON, non-object, empty key, non-numeric or
// out-of-range value) yields a non-nil error; callers that must fail open
// should ignore the map on error and fall back to ServiceSLA.
func ParseSLAOverrides(raw string) (map[string]int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var decoded map[string]json.Number
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	if err := dec.Decode(&decoded); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(decoded))
	for k, v := range decoded {
		key := strings.ToUpper(strings.TrimSpace(k))
		if key == "" {
			return nil, fmt.Errorf("sla_days_json: empty service code key")
		}
		n, err := v.Int64()
		if err != nil {
			return nil, fmt.Errorf("sla_days_json: value for %q must be an integer number of days", k)
		}
		if n <= 0 || n > 365 {
			return nil, fmt.Errorf("sla_days_json: value for %q must be 1..365 days", k)
		}
		out[key] = int(n)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// ResolveSLA returns the effective SLA in days for a service code given an
// already-parsed override map (nil map = no overrides):
// exact code match first, then "DEFAULT" key, else ServiceSLA(code).
func ResolveSLA(code string, overrides map[string]int) int {
	if len(overrides) > 0 {
		if v, ok := overrides[strings.ToUpper(strings.TrimSpace(code))]; ok {
			return v
		}
		if v, ok := overrides["DEFAULT"]; ok {
			return v
		}
	}
	return ServiceSLA(code)
}

// loadSLAOverrides reads office_flags.sla_days_json for one office.
// Fail-open: nil DB, missing table/row, NULL/empty value, or broken JSON
// all yield nil (caller falls back to ServiceSLA defaults).
func loadSLAOverrides(db *sql.DB, officeID int64) map[string]int {
	if db == nil {
		return nil
	}
	var raw sql.NullString
	if err := db.QueryRow(`SELECT sla_days_json FROM office_flags WHERE office_id=?`, officeID).Scan(&raw); err != nil {
		return nil
	}
	if !raw.Valid {
		return nil
	}
	m, err := ParseSLAOverrides(raw.String)
	if err != nil {
		return nil
	}
	return m
}

// ServiceSLAFor returns the effective SLA in days for a service code in one
// office: exact-code override, else "default" override, else ServiceSLA.
// Fail-open: nil DB, missing table/row, or broken JSON falls back to
// ServiceSLA(code) so behavior matches the old default.
func ServiceSLAFor(db *sql.DB, officeID int64, code string) int {
	return ResolveSLA(code, loadSLAOverrides(db, officeID))
}
