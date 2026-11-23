// Package remind — SLA workflow DMS (sla.go).
//
// SLAForWorkflow memetakan status workflow dokumen ke (dueDays, priority).
// OverdueDocs memindai dokumen kantor yang macet melewati SLA sejak aktivitas
// workflow terakhir (doc_workflow_events terbaru, jatuh kembali ke files.created_at
// karena tabel files tidak memiliki kolom updated_at).
package remind

import (
	"database/sql"
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
// Cara pakai (dashboard sengaja tidak diubah): panggil ServiceOverdue(db,
// officeID) dari handler/laporan baru bila ingin menampilkan permohonan yang
// melewati target umur layanannya, mis. badge "SLA terlewati (N)" di dashboard
// atau baris per permohonan dengan link /requests/{ID}. Ambang per layanan
// berasal dari ServiceSLA(kodeLayanan); override per kantor dapat disimpan di
// office_flags.sla_days_json (format bebas, mis. {"PPAT-AJB":10}) dan
// diterapkan oleh pemanggil — fungsi di bawah tidak membacanya agar perilaku
// default tetap deterministik.

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
// target from ServiceSLA. Rows are oldest first, capped at 100.
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
	for rows.Next() {
		var it ServiceOverdueItem
		if err := rows.Scan(&it.ID, &it.Title, &it.Code, &it.Service, &it.Status, &it.AgeDays); err != nil {
			continue
		}
		it.SLADays = ServiceSLA(it.Code)
		if it.AgeDays > it.SLADays {
			out = append(out, it)
		}
		if len(out) >= 100 {
			break
		}
	}
	return out, rows.Err()
}
