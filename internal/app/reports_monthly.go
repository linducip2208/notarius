package app

import (
	"database/sql"
	"net/http"
	"time"

	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
	"notaryos/internal/remind"
)

// Laporan bulanan 1-klik: rekap bulan berjalan siap cetak untuk pimpinan.
// Seluruh query agregat dipakai ulang dari halaman /reports lewat helper di
// bawah (reports + bulanan memanggil fungsi yang sama — tidak ada duplikasi
// logika berat). Tambahan khusus bulanan hanya metrik ringkas per bulan
// (reportMonthStats) yang memang belum ada di reports.

type reportBudgetRow struct {
	Category      string
	Budget, Spent int64
}

type reportProfitRow struct {
	Service      string
	Count, Total int64
}

type reportCatRow struct {
	Category       string
	Reqs           int64
	InvTotal, Paid int64
}

type reportTopRow struct {
	Service     string
	Count, Paid int64
}

// reportMonthStats adalah kartu ringkas bulan berjalan (semua office-scoped,
// bulan format YYYY-MM).
type reportMonthStats struct {
	ReqIn, ReqDone            int64
	InvIssued, InvIssuedTotal int64
	InvPaid, InvPaidTotal     int64
	InvOutstanding            int64
	DeedsSigned, Handovers    int64
}

func (a *App) reportBudgets(officeID int64, month string) []reportBudgetRow {
	var out []reportBudgetRow
	if a.db == nil {
		return out
	}
	rows, err := a.db.Query(`SELECT b.category, b.amount, COALESCE((SELECT SUM(e.amount) FROM expenses e
		WHERE e.office_id=b.office_id AND e.category=b.category AND DATE_FORMAT(e.spent_at,'%Y-%m-01')=b.month),0)
		FROM budgets b WHERE b.office_id=? AND b.month=? ORDER BY b.category`, officeID, month)
	if err != nil || rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var x reportBudgetRow
		var bamt, samt string
		_ = rows.Scan(&x.Category, &bamt, &samt)
		x.Budget = decWhole(bamt)
		x.Spent = decWhole(samt)
		out = append(out, x)
	}
	return out
}

func (a *App) reportProfits(officeID int64) []reportProfitRow {
	var out []reportProfitRow
	if a.db == nil {
		return out
	}
	rows, err := a.db.Query(`SELECT COALESCE(s.name,'(tanpa tautan layanan)'), COUNT(i.id), COALESCE(SUM(i.total),0)
		FROM invoices i LEFT JOIN service_requests r ON r.id=i.request_id LEFT JOIN services s ON s.id=r.service_id
		WHERE i.office_id=? AND i.status IN ('ISSUED','PARTIAL','PAID') GROUP BY 1 ORDER BY 3 DESC`, officeID)
	if err != nil || rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var x reportProfitRow
		var tot string
		_ = rows.Scan(&x.Service, &x.Count, &tot)
		x.Total = decWhole(tot)
		out = append(out, x)
	}
	return out
}

func (a *App) reportByCat(officeID int64) []reportCatRow {
	byCatMap := map[string]*reportCatRow{}
	if a.db != nil {
		if q1, err := a.db.Query(`SELECT s.category, COUNT(r.id) FROM service_requests r JOIN services s ON s.id=r.service_id WHERE r.office_id=? GROUP BY s.category`, officeID); err == nil && q1 != nil {
			defer q1.Close()
			for q1.Next() {
				var c string
				var n int64
				_ = q1.Scan(&c, &n)
				if _, ok := byCatMap[c]; !ok {
					byCatMap[c] = &reportCatRow{Category: c}
				}
				byCatMap[c].Reqs = n
			}
		}
		if q2, err := a.db.Query(`SELECT s.category, COALESCE(SUM(i.total),0) FROM invoices i JOIN service_requests r ON r.id=i.request_id JOIN services s ON s.id=r.service_id WHERE i.office_id=? AND i.status IN ('ISSUED','PARTIAL','PAID') GROUP BY s.category`, officeID); err == nil && q2 != nil {
			defer q2.Close()
			for q2.Next() {
				var c, tot string
				_ = q2.Scan(&c, &tot)
				if _, ok := byCatMap[c]; !ok {
					byCatMap[c] = &reportCatRow{Category: c}
				}
				byCatMap[c].InvTotal = decWhole(tot)
			}
		}
		if q3, err := a.db.Query(`SELECT s.category, COALESCE(SUM(i.paid),0) FROM invoices i JOIN service_requests r ON r.id=i.request_id JOIN services s ON s.id=r.service_id WHERE i.office_id=? AND i.status IN ('ISSUED','PARTIAL','PAID') GROUP BY s.category`, officeID); err == nil && q3 != nil {
			defer q3.Close()
			for q3.Next() {
				var c, tot string
				_ = q3.Scan(&c, &tot)
				if _, ok := byCatMap[c]; !ok {
					byCatMap[c] = &reportCatRow{Category: c}
				}
				byCatMap[c].Paid = decWhole(tot)
			}
		}
	}
	var out []reportCatRow
	for _, v := range byCatMap {
		out = append(out, *v)
	}
	return out
}

func (a *App) reportTopSvc(officeID int64, ym string) []reportTopRow {
	var out []reportTopRow
	if a.db == nil {
		return out
	}
	rows, err := a.db.Query(`SELECT COALESCE(s.name,'(tanpa tautan layanan)'), COUNT(i.id), COALESCE(SUM(i.paid),0)
		FROM invoices i LEFT JOIN service_requests r ON r.id=i.request_id LEFT JOIN services s ON s.id=r.service_id
		WHERE i.office_id=? AND DATE_FORMAT(i.created_at,'%Y-%m')=? AND i.status IN ('ISSUED','PARTIAL','PAID')
		GROUP BY 1 ORDER BY 3 DESC LIMIT 5`, officeID, ym)
	if err != nil || rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var x reportTopRow
		var paid string
		_ = rows.Scan(&x.Service, &x.Count, &paid)
		x.Paid = decWhole(paid)
		out = append(out, x)
	}
	return out
}

func (a *App) reportSLAOver(officeID int64) []remind.ServiceOverdueItem {
	if a.db == nil {
		return nil
	}
	over, err := remind.ServiceOverdue(a.db, officeID)
	if err != nil {
		return nil
	}
	if len(over) > 20 {
		over = over[:20]
	}
	return over
}

func (a *App) reportHandoverMonth(officeID int64, ym string) int64 {
	if a.db == nil {
		return 0
	}
	var dh, lg int64
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_handovers WHERE office_id=? AND DATE_FORMAT(COALESCE(handed_at,created_at),'%Y-%m')=?`, officeID, ym).Scan(&dh)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM legalizations WHERE office_id=? AND status='HANDED_OVER' AND DATE_FORMAT(completed_at,'%Y-%m')=?`, officeID, ym).Scan(&lg)
	return dh + lg
}

// reportMonthStats menghitung kartu ringkas bulan berjalan. Semua query
// ringan (COUNT/SUM tunggal, office-scoped, filter bulan YYYY-MM).
func (a *App) reportMonthStats(officeID int64, ym string) reportMonthStats {
	var st reportMonthStats
	if a.db == nil {
		return st
	}
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM service_requests WHERE office_id=? AND DATE_FORMAT(created_at,'%Y-%m')=?`, officeID, ym).Scan(&st.ReqIn)
	// Selesai bulan ini: akurat via request_events (to_status COMPLETED),
	// bukan sekadar status saat ini.
	_ = a.db.QueryRow(`SELECT COUNT(DISTINCT e.request_id) FROM request_events e
		JOIN service_requests r ON r.id=e.request_id
		WHERE r.office_id=? AND e.to_status='COMPLETED' AND DATE_FORMAT(e.created_at,'%Y-%m')=?`, officeID, ym).Scan(&st.ReqDone)
	var issuedTot, paidTot, outTot sql.NullString
	_ = a.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total),0) FROM invoices
		WHERE office_id=? AND DATE_FORMAT(created_at,'%Y-%m')=? AND status IN ('ISSUED','PARTIAL','PAID')`, officeID, ym).Scan(&st.InvIssued, &issuedTot)
	_ = a.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total),0) FROM invoices
		WHERE office_id=? AND DATE_FORMAT(created_at,'%Y-%m')=? AND status='PAID'`, officeID, ym).Scan(&st.InvPaid, &paidTot)
	_ = a.db.QueryRow(`SELECT COALESCE(SUM(total-paid),0) FROM invoices
		WHERE office_id=? AND DATE_FORMAT(created_at,'%Y-%m')=? AND status IN ('ISSUED','PARTIAL')`, officeID, ym).Scan(&outTot)
	if issuedTot.Valid {
		st.InvIssuedTotal = decWhole(issuedTot.String)
	}
	if paidTot.Valid {
		st.InvPaidTotal = decWhole(paidTot.String)
	}
	if outTot.Valid {
		st.InvOutstanding = decWhole(outTot.String)
	}
	// Akta disahkan bulan ini: deed_drafts SIGNED dengan approved_at
	// (fallback updated_at) pada bulan berjalan.
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts
		WHERE office_id=? AND status='SIGNED' AND DATE_FORMAT(COALESCE(approved_at,updated_at),'%Y-%m')=?`, officeID, ym).Scan(&st.DeedsSigned)
	st.Handovers = a.reportHandoverMonth(officeID, ym)
	return st
}

// reportDataset mengumpulkan seluruh dataset /reports (dipakai ulang oleh
// handler reports dan reportsBulanan agar tidak ada duplikasi logika).
func (a *App) reportDataset(officeID int64) map[string]any {
	pending, active, completed := 0, 0, 0
	var out int64
	if a.db != nil {
		pending, active, completed, _ = a.reqs.Counts(officeID)
		out, _ = a.fin.Outstanding(officeID)
	}
	month := time.Now().Format("2006-01-01")
	ym := time.Now().Format("2006-01")
	return map[string]any{
		"Pending": pending, "Active": active, "Completed": completed, "Outstanding": out,
		"Budgets": a.reportBudgets(officeID, month), "Profits": a.reportProfits(officeID),
		"Month": ym, "ByCat": a.reportByCat(officeID),
		"TopSvc": a.reportTopSvc(officeID, ym), "SLAOver": a.reportSLAOver(officeID),
		"HandoverMonth": a.reportHandoverMonth(officeID, ym),
	}
}

// reportsBulanan merender rekap 1 halaman bulan berjalan siap cetak
// (PDF-via-browser) untuk pimpinan.
func (a *App) reportsBulanan(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewReports) {
		a.forbidden(w, r)
		return
	}
	officeID := a.officeOf(r)
	ym := time.Now().Format("2006-01")
	data := a.reportDataset(officeID)
	stats := a.reportMonthStats(officeID, ym)
	var officeName, notary, address, phone, email string
	if a.db != nil {
		_ = a.db.QueryRow(`SELECT name, notary_name, address, phone, email FROM offices WHERE id=?`, officeID).Scan(&officeName, &notary, &address, &phone, &email)
	}
	data["Stats"] = stats
	data["Office"] = map[string]any{
		"Name": officeName, "Notary": notary, "Address": address, "Phone": phone, "Email": email,
	}
	data["PrintedAt"] = time.Now().Format("02-01-2006 15:04")
	a.render(w, r, "reports_bulanan", "Rekap Bulanan "+ym, data)
}
