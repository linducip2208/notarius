package app

// Ops growth: specimens, bank reconciliation, budgets, inquiry->request,
// deed attach+package, portal receipt, retention->task, backup, read-only mode,
// wills review/opening, gov follow-up. All office-scoped + audited.

import (
	"archive/zip"
	"compress/gzip"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/clients"
	mw "notaryos/internal/middleware"
	"notaryos/internal/money"
	"notaryos/internal/rbac"
	"notaryos/internal/validate"
)

// ---- specimens ----

func (a *App) specimens(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	cid, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.clients.Get(a.officeOf(r), cid); err != nil {
		http.NotFound(w, r)
		return
	}
	rows, _ := a.db.Query(`SELECT id, kind, file_id, note, created_at FROM client_specimens WHERE client_id=? ORDER BY id`, cid)
	type sp struct {
		ID            int64
		Kind          string
		File          int64
		Note, At      string
	}
	var out []sp
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x sp
			_ = rows.Scan(&x.ID, &x.Kind, &x.File, &x.Note, &x.At)
			out = append(out, x)
		}
	}
	fid := r.URL.Query().Get("compare")
	a.render(w, r, "specimens", "Spesimen", map[string]any{"ClientID": cid, "Rows": out, "Compare": fid})
}

func (a *App) specimenUpload(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseMultipartForm(16 << 20)
	cid, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.clients.Get(a.officeOf(r), cid); err != nil {
		http.NotFound(w, r)
		return
	}
	kind := strings.ToUpper(strings.TrimSpace(r.FormValue("kind")))
	switch kind {
	case "TTD", "CAP", "STEMPEL", "LAIN":
	default:
		kind = "TTD"
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "pilih berkas gambar terlebih dahulu", http.StatusBadRequest)
		return
	}
	defer f.Close()
	if a.overQuota(a.officeOf(r), hdr.Size) {
		http.Error(w, "kuota penyimpanan kantor terlampaui", http.StatusInsufficientStorage)
		return
	}
	fid, err := a.uploads.SaveUpload(a.officeOf(r), "specimen", cid, hdr.Filename, hdr.Header.Get("Content-Type"), f, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := a.db.Exec(`INSERT INTO client_specimens(office_id, client_id, kind, file_id, note, created_by) VALUES(?,?,?,?,?,?)`,
		a.officeOf(r), cid, kind, fid, strings.TrimSpace(r.FormValue("note")), a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "client.specimen.add", Entity: "client", EntityID: strconv.FormatInt(cid, 10), Detail: kind})
	http.Redirect(w, r, "/clients/"+strconv.FormatInt(cid, 10)+"/specimens?flash=Spesimen+disimpan", http.StatusSeeOther)
}

// ---- bank reconciliation ----

func (a *App) bankPage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewFinance) {
		a.forbidden(w, r)
		return
	}
	rows, _ := a.db.Query(`SELECT id, stmt_date, description, amount, matched_invoice_id FROM bank_mutations WHERE office_id=? ORDER BY stmt_date DESC, id DESC LIMIT 200`, a.officeOf(r))
	type mu struct {
		ID             int64
		Date, Desc     string
		Amount         string
		Matched        int64
	}
	var out []mu
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x mu
			var amt string
			var m sql.NullInt64
			_ = rows.Scan(&x.ID, &x.Date, &x.Desc, &amt, &m)
			x.Amount = amt
			if m.Valid {
				x.Matched = m.Int64
			}
			out = append(out, x)
		}
	}
	a.render(w, r, "bank", "Rekonsiliasi Bank", map[string]any{"Rows": out})
}

func (a *App) bankUpload(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseMultipartForm(8 << 20)
	f, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "pilih file CSV (tanggal,uraian,nominal)", http.StatusBadRequest)
		return
	}
	defer f.Close()
	rd := csv.NewReader(f)
	recs, err := rd.ReadAll()
	if err != nil || len(recs) == 0 {
		http.Error(w, "CSV tidak terbaca", http.StatusBadRequest)
		return
	}
	n := 0
	for i, rec := range recs {
		if len(rec) < 3 {
			continue
		}
		if i == 0 && strings.Contains(strings.ToLower(rec[0]), "tanggal") {
			continue // header
		}
		date := strings.TrimSpace(rec[0])
		desc := strings.TrimSpace(rec[1])
		amt := strings.TrimSpace(rec[2])
		amt = strings.ReplaceAll(strings.ReplaceAll(amt, ".", ""), ",", ".")
		if _, err := strconv.ParseFloat(amt, 64); err != nil {
			continue
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			continue
		}
		if _, err := a.db.Exec(`INSERT INTO bank_mutations(office_id, stmt_date, description, amount) VALUES(?,?,?,?)`,
			a.officeOf(r), date, desc, amt); err == nil {
			n++
		}
		if n >= 1000 {
			break
		}
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "bank.import", Entity: "office", EntityID: "self", Detail: strconv.Itoa(n) + " baris"})
	http.Redirect(w, r, "/finance/bank?flash="+strconv.Itoa(n)+"+mutasi+diimpor", http.StatusSeeOther)
}

// bankMatch auto-links mutations to UNPAID invoices by exact amount (±0) within 30 days.
// Explicitly reports matched vs unmatched — never silently forces.
func (a *App) bankMatch(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	rows, _ := a.db.Query(`SELECT id, stmt_date, amount FROM bank_mutations WHERE office_id=? AND matched_invoice_id IS NULL`, a.officeOf(r))
	matched := 0
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var date, amt string
			_ = rows.Scan(&id, &date, &amt)
			var invID int64
			err := a.db.QueryRow(`SELECT i.id FROM invoices i WHERE i.office_id=? AND i.status IN ('ISSUED','PARTIAL')
				AND i.total=? AND i.created_at BETWEEN DATE_SUB(? ,INTERVAL 30 DAY) AND DATE_ADD(?, INTERVAL 7 DAY)
				AND NOT EXISTS (SELECT 1 FROM bank_mutations m WHERE m.matched_invoice_id=i.id)
				ORDER BY i.id LIMIT 1`, a.officeOf(r), amt, date, date).Scan(&invID)
			if err == nil {
				if _, err := a.db.Exec(`UPDATE bank_mutations SET matched_invoice_id=? WHERE id=?`, invID, id); err == nil {
					matched++
				}
			}
		}
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "bank.match", Entity: "office", EntityID: "self", Detail: strconv.Itoa(matched) + " cocok"})
	http.Redirect(w, r, "/finance/bank?flash="+strconv.Itoa(matched)+"+mutasi+cocok", http.StatusSeeOther)
}

func (a *App) bankManual(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	mid, _ := strconv.ParseInt(r.FormValue("mutation_id"), 10, 64)
	iid, _ := strconv.ParseInt(r.FormValue("invoice_id"), 10, 64)
	var mamt, itotal string
	var moid, ioid, ist int64
	_ = a.db.QueryRow(`SELECT amount, office_id FROM bank_mutations WHERE id=?`, mid).Scan(&mamt, &moid)
	_ = a.db.QueryRow(`SELECT total, office_id, status IN ('ISSUED','PARTIAL') FROM invoices WHERE id=?`, iid).Scan(&itotal, &ioid, &ist)
	if moid == 0 || moid != a.officeOf(r) || ioid != a.officeOf(r) {
		http.Error(w, "mutasi/invoice tidak ditemukan di kantor ini", http.StatusNotFound)
		return
	}
	if ist != 1 {
		http.Error(w, "hanya invoice ISSUED/PARTIAL yang bisa ditautkan", http.StatusBadRequest)
		return
	}
	mv, _ := money.Parse(mamt)
	iv, _ := money.Parse(itotal)
	if mv != iv {
		http.Error(w, "nominal mutasi tidak sama dengan total invoice — gunakan automatch untuk selisih", http.StatusBadRequest)
		return
	}
	if _, err := a.db.Exec(`UPDATE bank_mutations SET matched_invoice_id=? WHERE id=? AND office_id=?`, iid, mid, a.officeOf(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "bank.manual", Entity: "bank_mutation", EntityID: strconv.FormatInt(mid, 10), Detail: "invoice#" + strconv.FormatInt(iid, 10)})
	http.Redirect(w, r, "/finance/bank?flash=Cocok+manual+disimpan", http.StatusSeeOther)
}

// ---- budgets ----

func (a *App) budgetSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageFinance) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	cat := strings.TrimSpace(r.FormValue("category"))
	month := strings.TrimSpace(r.FormValue("month")) // YYYY-MM
	if cat == "" || month == "" {
		http.Error(w, "kategori dan bulan wajib diisi", http.StatusBadRequest)
		return
	}
	if len(month) == 7 {
		month += "-01"
	}
	if _, err := time.Parse("2006-01-02", month); err != nil {
		http.Error(w, "bulan tidak valid", http.StatusBadRequest)
		return
	}
	amtVal, err := money.Parse(strings.TrimSpace(r.FormValue("amount")))
	if err != nil || amtVal < 0 {
		http.Error(w, "nominal tidak valid", http.StatusBadRequest)
		return
	}
	if _, err := a.db.Exec(`INSERT INTO budgets(office_id, category, month, amount) VALUES(?,?,?,?)
		ON DUPLICATE KEY UPDATE amount=VALUES(amount)`, a.officeOf(r), cat, month, amtVal); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/reports?flash=Anggaran+disimpan", http.StatusSeeOther)
}

// ---- inquiry -> request ----

func (a *App) inquiryToRequest(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermWebInquiries) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var name, contact, service, message, status string
	if err := a.db.QueryRow(`SELECT name, contact, service, message, status FROM website_inquiries WHERE id=? AND office_id=?`,
		id, a.officeOf(r)).Scan(&name, &contact, &service, &message, &status); err != nil {
		http.NotFound(w, r)
		return
	}
	if status != "BARU" {
		http.Error(w, "inquiry sudah diproses/dikonversi (status "+status+") — ubah status ke BARU dulu bila yakin ingin membuat permohonan lagi", http.StatusConflict)
		return
	}
	svcID, _ := strconv.ParseInt(r.FormValue("service_id"), 10, 64)
	if svcID <= 0 {
		_ = a.db.QueryRow(`SELECT id FROM services WHERE is_active=1 ORDER BY id LIMIT 1`).Scan(&svcID)
	}
	if svcID <= 0 {
		http.Error(w, "katalog layanan kosong", http.StatusBadRequest)
		return
	}
	if validate.Phone(contact) != nil {
		http.Error(w, "kontak inquiry bukan nomor HP valid — minta klien melengkapi via konsultasi ulang", http.StatusBadRequest)
		return
	}
	cid, err := a.clients.Create(clients.Client{OfficeID: a.officeOf(r), Kind: "individual", FullName: name, Phone: contact})
	if err != nil {
		// Klien mungkin sudah ada (nama sama): pakai yang cocok, jangan ganda diam-diam.
		dups, _ := a.clients.FindDuplicatesEx(a.officeOf(r), 0, name, contact, "", "", "", "")
		if len(dups) == 0 {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cid = dups[0].ID
	}
	rid, err := a.reqs.Create(a.officeOf(r), cid, svcID, strings.TrimSpace(service+" — "+name), message, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = a.db.Exec(`UPDATE website_inquiries SET status='DIPROSES' WHERE id=? AND office_id=?`, id, a.officeOf(r))
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "inquiry.convert", Entity: "inquiry", EntityID: strconv.FormatInt(id, 10), Detail: fmt.Sprintf("client#%d request#%d", cid, rid)})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(rid, 10)+"?flash=Permohonan+dari+konsultasi+dibuat", http.StatusSeeOther)
}

// ---- deed attach + package ----

func (a *App) deedAttach(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditDeed) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	deedID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	fileID, _ := strconv.ParseInt(r.FormValue("file_id"), 10, 64)
	var n int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM deed_drafts WHERE id=? AND office_id=?`, deedID, a.officeOf(r)).Scan(&n)
	if n == 0 {
		http.NotFound(w, r)
		return
	}
	if err := a.dms.AddRelation(a.officeOf(r), fileID, "deed", deedID, "lampiran"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "deed.attach", Entity: "deed", EntityID: strconv.FormatInt(deedID, 10), Detail: "file#" + strconv.FormatInt(fileID, 10)})
	http.Redirect(w, r, "/deeds/"+strconv.FormatInt(deedID, 10)+"?flash=Berkas+ditautkan", http.StatusSeeOther)
}

// deedPackage exports deed content + metadata + signatures as one ZIP (review pack).
func (a *App) deedPackage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewProtocol) {
		a.forbidden(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	d, body, err := a.deeds.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="akta-%d.zip"`, id))
	zw := zip.NewWriter(w)
	defer zw.Close()
	put := func(name, content string) {
		f, err := zw.Create(name)
		if err != nil {
			return
		}
		_, _ = io.WriteString(f, content)
	}
	put("akta.html", body)
	put("metadata.txt", fmt.Sprintf("ID: %d\nNomor: %s\nJudul: %s\nJenis: %s\nStatus: %s\nVersi: %d\n",
		d.ID, d.Number, d.Title, d.Kind, d.Status, d.Version))
	sigs := deedSigs(a.db, id)
	var sb strings.Builder
	for _, s := range sigs {
		fmt.Fprintf(&sb, "%s | %s | %s\n", s.Signer, s.At, s.Hash)
	}
	put("tanda-tangan.txt", sb.String())
	rels, _ := a.dms.RelationsOf(a.officeOf(r), id)
	_ = rels
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "deed.package", Entity: "deed", EntityID: strconv.FormatInt(id, 10)})
}

// ---- portal receipt (client-scoped) ----

func (a *App) portalReceipt(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != rbac.RoleClient {
		a.forbidden(w, r)
		return
	}
	id := a.uid(r)
	var clientID int64
	if id != nil {
		clientID = a.users.LinkedClient(*id)
	}
	invID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var number, status, cname string
	var stotal, spaid string
	err := a.db.QueryRow(`SELECT i.number, i.status, c.full_name, i.total, i.paid FROM invoices i
		JOIN clients c ON c.id=i.client_id WHERE i.id=? AND i.office_id=? AND i.client_id=?`,
		invID, a.officeOf(r), clientID).Scan(&number, &status, &cname, &stotal, &spaid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	code, err := verifyCode("INV", invID)
	if err != nil {
		http.Error(w, "kunci verifikasi belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}
	kind := ""
	var first string
	if err := a.db.QueryRow(`SELECT description FROM invoice_lines WHERE invoice_id=? ORDER BY id LIMIT 1`, invID).Scan(&first); err == nil {
		switch {
		case len(first) >= 4 && first[:4] == "[DP]":
			kind = "DP"
		case len(first) >= 11 && first[:11] == "[PELUNASAN]":
			kind = "PELUNASAN"
		}
	}
	type payRow struct {
		Amount    int64
		Method    string
		Reference string
		PaidAt    string
	}
	var pays []payRow
	if rs, err := a.db.Query(`SELECT amount, method, COALESCE(reference,''), paid_at FROM payments WHERE invoice_id=? ORDER BY paid_at, id`, invID); err == nil && rs != nil {
		defer rs.Close()
		for rs.Next() {
			var p payRow
			var samt, sat sql.NullString
			if err := rs.Scan(&samt, &p.Method, &p.Reference, &sat); err != nil {
				continue
			}
			if samt.Valid {
				p.Amount = decWhole(samt.String)
			}
			if sat.Valid {
				p.PaidAt = sat.String
			}
			pays = append(pays, p)
		}
	}
	a.render(w, r, "receipt", "Kuitansi "+number, map[string]any{
		"ID": invID, "Number": number, "Client": cname, "Status": status,
		"Total": decWhole(stotal), "Paid": decWhole(spaid),
		"Outstanding": decWhole(stotal) - decWhole(spaid), "Code": code,
		"Kind": kind, "Payments": pays,
	})
}

// ---- retention -> task (aturan otomatis, bukan manual) ----

// retentionTask membuat tugas tindak lanjut dari aturan retensi:
//   - bendel protokol berumur >25 tahun (kolom year) -> tugas arsip;
//   - berkas CONFIDENTIAL/RESTRICTED berumur >10 tahun (created_at, karena
//     tabel files tidak memiliki kolom updated_at) -> tugas review retensi.
//
// Dibatasi 50 baris per jenis; tugas file bertaut ref_type='file' dan tidak
// diduplikasi bila masih ada yang OPEN untuk ref yang sama.
func (a *App) retentionTask(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return
	}
	office := a.officeOf(r)
	made := 0

	// 1. Protokol >25 tahun -> tugas arsip.
	if prows, err := a.db.Query(`SELECT id, bundle_no, year FROM protocols
		WHERE office_id=? AND year <= YEAR(CURDATE())-25 ORDER BY year LIMIT 50`, office); err == nil && prows != nil {
		defer prows.Close()
		for prows.Next() {
			var id int64
			var bundle string
			var year int
			_ = prows.Scan(&id, &bundle, &year)
			title := fmt.Sprintf("Arsipkan protokol #%d (bundel %s/%d): melewati retensi 25 tahun", id, bundle, year)
			var n int
			_ = a.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE office_id=? AND status='OPEN' AND title=?`, office, title).Scan(&n)
			if n > 0 {
				continue
			}
			if _, err := a.db.Exec(`INSERT INTO tasks(office_id, title, due_date, created_by) VALUES(?,?,CURDATE()+INTERVAL 7 DAY,?)`,
				office, title, a.uid(r)); err == nil {
				made++
			}
		}
	}

	// 2. Berkas CONFIDENTIAL/RESTRICTED berumur >10 tahun -> tugas review retensi.
	if frows, err := a.db.Query(`SELECT id, COALESCE(display_name,original_name) FROM files
		WHERE office_id=? AND confidentiality IN ('CONFIDENTIAL','RESTRICTED')
		AND created_at < DATE_SUB(CURDATE(), INTERVAL 10 YEAR)
		ORDER BY created_at LIMIT 50`, office); err == nil && frows != nil {
		defer frows.Close()
		for frows.Next() {
			var id int64
			var name string
			_ = frows.Scan(&id, &name)
			var n int
			_ = a.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE office_id=? AND status='OPEN' AND ref_type='file' AND ref_id=?`,
				office, id).Scan(&n)
			if n > 0 {
				continue
			}
			title := fmt.Sprintf("Review retensi: dok #%d (%s) berumur >10 tahun", id, name)
			if _, err := a.db.Exec(`INSERT INTO tasks(office_id, title, due_date, created_by, ref_type, ref_id) VALUES(?,?,CURDATE()+INTERVAL 7 DAY,?, 'file', ?)`,
				office, title, a.uid(r), id); err == nil {
				made++
			}
		}
	}

	if made == 0 {
		http.Redirect(w, r, "/tasks?flash=Tidak+ada+aturan+retensi+yang+jatuh+tempo", http.StatusSeeOther)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "retention.task", Entity: "office", EntityID: "self", Detail: strconv.Itoa(made) + " tugas aturan retensi"})
	http.Redirect(w, r, "/tasks?flash=Tugas+retensi+dibuat", http.StatusSeeOther)
}

// ---- backup ----

func (a *App) backupNow(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageOffice) {
		a.forbidden(w, r)
		return
	}
	mysqldump, err := exec.LookPath("mysqldump")
	if err != nil {
		http.Error(w, "mysqldump tidak ditemukan di server — backup manual via skrip deploy", http.StatusServiceUnavailable)
		return
	}
	// Tanpa gpg / GPG_RECIPIENT backup DITOLAK: dump plaintext tidak boleh dibuat.
	gpgBin, err := exec.LookPath("gpg")
	if err != nil {
		http.Error(w, "gpg tidak ditemukan di server — backup dibatalkan agar tidak menghasilkan file plaintext", http.StatusServiceUnavailable)
		return
	}
	recipient := strings.TrimSpace(os.Getenv("GPG_RECIPIENT"))
	if recipient == "" {
		http.Error(w, "GPG_RECIPIENT belum dikonfigurasi — backup plaintext ditolak demi keamanan", http.StatusInternalServerError)
		return
	}
	dir := "storage/backups"
	_ = os.MkdirAll(dir, 0o750)
	// Pola sama seperti deploy/backup/backup.sh: mysqldump | gzip | gpg --encrypt.
	// Streaming langsung ke gpg — tidak ada file plaintext di disk.
	name := fmt.Sprintf("%s/backup-%s.sql.gz.gpg", dir, time.Now().Format("20060102-150405"))
	host := getenv("DB_HOST", "127.0.0.1")
	user := getenv("DB_USER", "notaryos")
	pass := os.Getenv("DB_PASS")
	dbname := getenv("DB_NAME", "notaryos")
	// Kredensial via defaults-file 0600 (tidak lewat argv/ps).
	cnf, err := os.CreateTemp("", "notaryos-backup-*.cnf")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(cnf.Name())
	_ = os.Chmod(cnf.Name(), 0o600)
	fmt.Fprintf(cnf, "[client]\nhost=%s\nuser=%s\npassword=%s\n", host, user, strings.ReplaceAll(pass, "\n", ""))
	_ = cnf.Close()
	dump := exec.Command(mysqldump, "--defaults-extra-file="+cnf.Name(), "--single-transaction", "--routines", dbname)
	crypt := exec.Command(gpgBin, "--batch", "--yes", "--encrypt", "--recipient", recipient, "-o", name)
	gpgIn, err := crypt.StdinPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	gz := gzip.NewWriter(gpgIn)
	dump.Stdout = gz
	var dumpErr, cryptErr strings.Builder
	dump.Stderr = &dumpErr
	crypt.Stderr = &cryptErr
	start := time.Now()
	if err := crypt.Start(); err != nil {
		_ = gz.Close()
		http.Error(w, "backup gagal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := dump.Start(); err != nil {
		_ = gz.Close()
		_ = gpgIn.Close()
		_ = crypt.Wait()
		http.Error(w, "backup gagal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	dumpWait := dump.Wait()
	_ = gz.Close()
	_ = gpgIn.Close()
	cryptWait := crypt.Wait()
	if dumpWait != nil || cryptWait != nil {
		_ = os.Remove(name)
		http.Error(w, "backup gagal: "+strings.TrimSpace(dumpErr.String()+" "+cryptErr.String()), http.StatusInternalServerError)
		return
	}
	dur := time.Since(start).Round(time.Second)
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "backup.now", Entity: "office", EntityID: "self", Detail: name + " " + dur.String()})
	flash := "Backup+terenkripsi+selesai+" + dur.String()
	if n := countLegacyPlaintextBackups(dir); n > 0 {
		flash += "+--+PERINGATAN:+masih+ada+" + strconv.Itoa(n) + "+file+backup+plaintext+lama+(backup-*.sql)+yang+harus+dihapus+manual+di+bawah"
	}
	http.Redirect(w, r, "/office?flash="+flash, http.StatusSeeOther)
}

// countLegacyPlaintextBackups menghitung dump plaintext lawas (backup-*.sql)
// yang pernah ditulis sebelum backup terenkripsi. File .sql.gz.gpg tidak dihitung.
func countLegacyPlaintextBackups(dir string) int {
	matches, _ := filepath.Glob(filepath.Join(dir, "backup-*.sql"))
	n := 0
	for _, m := range matches {
		if strings.HasSuffix(m, ".sql") && !strings.Contains(filepath.Base(m), ".gpg") {
			n++
		}
	}
	return n
}

// backupPurgePlaintext menghapus dump plaintext lawas (backup-*.sql) setelah
// konfirmasi ganda (confirm dialog + tombol khusus). File terenkripsi
// .sql.gz.gpg tidak pernah disentuh.
func (a *App) backupPurgePlaintext(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageOffice) {
		a.forbidden(w, r)
		return
	}
	dir := "storage/backups"
	matches, _ := filepath.Glob(filepath.Join(dir, "backup-*.sql"))
	removed := 0
	for _, m := range matches {
		if !strings.HasSuffix(m, ".sql") || strings.Contains(filepath.Base(m), ".gpg") {
			continue
		}
		if err := os.Remove(m); err == nil {
			removed++
		}
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "backup.purge-plaintext", Entity: "office", EntityID: "self", Detail: strconv.Itoa(removed) + " file plaintext dihapus"})
	http.Redirect(w, r, "/office?flash="+strconv.Itoa(removed)+"+file+backup+plaintext+lama+dihapus", http.StatusSeeOther)
}

// ---- read-only emergency mode ----

func (a *App) readOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("READ_ONLY") == "1" && r.Method != http.MethodGet && r.Method != http.MethodHead {
			if !strings.HasPrefix(r.URL.Path, "/login") {
				http.Error(w, "mode darurat baca-saja aktif — perubahan ditahan sementara", http.StatusServiceUnavailable)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ---- wills review + witnessed opening ----

func willWriter(role string) bool {
	return role == rbac.RoleNotary || role == rbac.RoleSuperAdmin || role == rbac.RoleSubstitute
}

func (a *App) willReview(w http.ResponseWriter, r *http.Request) {
	if !willWriter(mw.RoleOf(r)) {
		a.forbidden(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.db.Exec(`UPDATE wills SET review_date=CURDATE() WHERE id=? AND office_id=?`, id, a.officeOf(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "will.review", Entity: "will", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/wills?flash=Review+tahunan+dicatat", http.StatusSeeOther)
}

// willOpen records a witnessed opening (notary + named witness + note). Handover stays separate.
func (a *App) willOpen(w http.ResponseWriter, r *http.Request) {
	if !willWriter(mw.RoleOf(r)) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	witness := strings.TrimSpace(r.FormValue("witness"))
	note := strings.TrimSpace(r.FormValue("note"))
	if witness == "" {
		http.Error(w, "nama saksi pembuka wajib diisi (prinsip dua pihak)", http.StatusBadRequest)
		return
	}
	if _, err := a.db.Exec(`UPDATE wills SET opened_by=?, opened_witness=?, opened_at=NOW(), open_note=? WHERE id=? AND office_id=? AND opened_at IS NULL`,
		a.uid(r), witness, note, id, a.officeOf(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "will.open", Entity: "will", EntityID: strconv.FormatInt(id, 10), Detail: "saksi: " + witness})
	http.Redirect(w, r, "/wills?flash=Pembukaan+bersaksi+dicatat", http.StatusSeeOther)
}

// ---- gov follow-up ----

func (a *App) govFollowup(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermGovSubmit) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	fu := strings.TrimSpace(r.FormValue("follow_up"))
	if fu == "" {
		_, _ = a.db.Exec(`UPDATE gov_submissions SET follow_up=NULL WHERE id=? AND office_id=?`, id, a.officeOf(r))
	} else {
		if _, err := time.Parse("2006-01-02", fu); err != nil {
			http.Error(w, "tanggal tidak valid", http.StatusBadRequest)
			return
		}
		_, _ = a.db.Exec(`UPDATE gov_submissions SET follow_up=? WHERE id=? AND office_id=?`, fu, id, a.officeOf(r))
	}
	http.Redirect(w, r, "/gov?flash=Tindak+lanjut+disimpan", http.StatusSeeOther)
}

func getenv(k, fb string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fb
}

// ---- substitute mandates (W-12) ----

func (a *App) mandatesPage(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		a.forbidden(w, r)
		return
	}
	rows, _ := a.db.Query(`SELECT m.id, u.name, m.sk_no, m.start_date, m.end_date, m.scope FROM substitute_mandates m
		JOIN users u ON u.id=m.user_id WHERE m.office_id=? ORDER BY m.end_date DESC`, a.officeOf(r))
	type m struct {
		ID                          int64
		Name, SK, Start, End, Scope string
	}
	var out []m
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x m
			_ = rows.Scan(&x.ID, &x.Name, &x.SK, &x.Start, &x.End, &x.Scope)
			out = append(out, x)
		}
	}
	urows, _ := a.db.Query(`SELECT u.id, u.name FROM users u JOIN roles r ON r.id=u.role_id WHERE u.office_id=? AND r.name='substitute_notary' AND u.is_active=1 ORDER BY u.name`, a.officeOf(r))
	type u struct {
		ID   int64
		Name string
	}
	var users []u
	if urows != nil {
		defer urows.Close()
		for urows.Next() {
			var x u
			_ = urows.Scan(&x.ID, &x.Name)
			users = append(users, x)
		}
	}
	a.render(w, r, "mandates", "Mandat Notaris Pengganti", map[string]any{"Rows": out, "Users": users})
}

func (a *App) mandateSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	uid, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	sk, start, end, scope := strings.TrimSpace(r.FormValue("sk_no")), r.FormValue("start_date"), r.FormValue("end_date"), strings.TrimSpace(r.FormValue("scope"))
	if uid <= 0 || sk == "" || start == "" || end == "" || start > end {
		http.Error(w, "mandat wajib: pengguna, SK, periode valid", http.StatusBadRequest)
		return
	}
	var role, roid int64
	_ = a.db.QueryRow(`SELECT role_id, office_id FROM users WHERE id=?`, uid).Scan(&role, &roid)
	var rname string
	_ = a.db.QueryRow(`SELECT name FROM roles WHERE id=?`, role).Scan(&rname)
	if rname != "substitute_notary" || roid != a.officeOf(r) {
		http.Error(w, "hanya akun substitute_notary kantor ini", http.StatusBadRequest)
		return
	}
	if scope == "" {
		scope = "penuh"
	}
	if _, err := a.db.Exec(`INSERT INTO substitute_mandates(office_id, user_id, sk_no, start_date, end_date, scope, created_by) VALUES(?,?,?,?,?,?,?)`,
		a.officeOf(r), uid, sk, start, end, scope, a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "mandate.create", Entity: "mandate", EntityID: sk})
	http.Redirect(w, r, "/office/mandates?flash=Mandat+dicatat", http.StatusSeeOther)
}

func (a *App) mandateDelete(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		a.forbidden(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, _ = a.db.Exec(`DELETE FROM substitute_mandates WHERE id=? AND office_id=?`, id, a.officeOf(r))
	http.Redirect(w, r, "/office/mandates?flash=Mandat+dihapus", http.StatusSeeOther)
}
