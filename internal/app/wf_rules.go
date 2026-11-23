package app

import (
	"database/sql"
	"fmt"
	"strings"
)

// Workflow Rules Engine — pengganti if ad-hoc yang tersebar di
// requestMove (app.go), deedApprove (ext.go), dan ppatAdvance (app.go).
// Sumber aturan: tabel service_workflow_rules (migrasi 026_matter_workflow.sql).
//
// Kontrak helper:
//
//	func workflowGate(db *sql.DB, officeID, reqID int64, svcCode, kind string) []string
//
//   - svcCode boleh "" → di-resolve dari service_requests JOIN services.
//   - kind adalah JENIS GATE: "move" | "approve" | "ppat". Nilai lain/asing
//     diperlakukan sebagai gate umum (tanpa cek resi BPN); bila bukan nama
//     gate yang dikenal dan tidak kosong, dipakai juga sebagai petunjuk
//     deed-kind bila lookup DB tidak menemukan draf akta tertaut.
//   - Kembalian: daftar alasan blokir (bukan error). Kosong/nil = lolos.
//
// Keputusan desain fail-open vs fail-closed per cek (penting, dibaca dulu):
//   1. db == nil atau reqID <= 0 → nil (fail-open; handler hilir tetap
//      memvalidasi eksistensi baris dan mengembalikan 404/400-nya sendiri).
//   2. Tabel service_workflow_rules HILANG (DB lama pra-026) → nil, fail-open
//      penuh (perilaku lama dipertahankan). HANYA kasus ini yang fail-open
//      di level engine.
//   3. Baris rule untuk svcCode tak ada → default KETAT SEDANG: checklist
//      diwajibkan (tapi hanya efektif bila template checklist ada — bila
//      tidak ada item, cek ini lolos sendiri), identitas pihak wajib
//      VERIFIED, min_parties=1 (selalu terpenuhi klien utama), tanpa syarat
//      deed_kind dan tanpa resi BPN.
//   4. Sub-tabel pendukung hilang (matter_parties / identities / checklist /
//      ppat_register pra-migrasi): cek yang bersangkutan dilewati (fail-open
//      per-cek) agar kantor dengan DB lama tidak terkunci total. Ini
//      disengaja dan dicatat di tiap titik.
//   5. Request/service tidak ditemukan (svcCode tak bisa di-resolve) → nil;
//      Move/Approve hilir akan menolak dengan error-nya sendiri.
//   6. Cek resi BPN HANYA untuk gate "ppat" (dan ppatAdvance memfilter blok
//      ini kecuali target LAPOR_BPN; resi susulan dari form juga memfilter).
//      Gate "move"/"approve" tidak pernah memblokir soal resi.
//   7. Cek deed_kind hanya bila rule menetapkan require_deed_kind DAN ada
//      akta/draf tertaut yang bisa dibandingkan; tanpa draf tertaut → lewat
//      (requestMove tanpa akta tidak diblokir soal jenis akta).
//   8. Checklist: item berstatus selain VERIFIED/RECEIVED memblokir. MISSING
//      dan REJECTED/NEEDS_REVISION dilaporkan sebagai blok terpisah agar
//      pesan requestMove lama ("masih ada N dokumen berstatus MISSING…")
//      tetap bisa dipertahankan formatnya oleh pemanggil.
type workflowRule struct {
	requireChecklist        bool
	requireVerifiedIdentity bool
	requireDeedKind         string
	requireBPNReceipt       bool
	minParties              int
}

// defaultWorkflowRule adalah aturan aman untuk service_code tanpa baris rule.
func defaultWorkflowRule() workflowRule {
	return workflowRule{
		requireChecklist:        true, // efektif hanya bila template ada
		requireVerifiedIdentity: true,
		requireDeedKind:         "",
		requireBPNReceipt:       false,
		minParties:              1,
	}
}

// isTableMissing mengenali error "tabel belum ada" lintas driver
// (MySQL 1146 vs SQLite "no such table") untuk fail-open yang ditargetkan.
func isTableMissing(err error) bool {
	if err == nil {
		return false
	}
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "doesn't exist") ||
		strings.Contains(m, "no such table") ||
		strings.Contains(m, "unknown table") ||
		strings.Contains(m, "1146")
}

// isUnknownColumn mengenali error "kolom belum ada" (DB lama pra-migrasi).
func isUnknownColumn(err error) bool {
	if err == nil {
		return false
	}
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "unknown column") ||
		strings.Contains(m, "no such column") ||
		strings.Contains(m, "1054")
}

// dropWorkflowBlocks membuang blok dengan prefix tertentu (dipakai ppatAdvance
// untuk mengabaikan blok resi saat resi susulan tersedia di form / target
// belum memerlukan resi).
func dropWorkflowBlocks(blocks []string, prefix string) []string {
	kept := blocks[:0]
	for _, b := range blocks {
		if !strings.HasPrefix(b, prefix) {
			kept = append(kept, b)
		}
	}
	return kept
}

// workflowGate menilai aturan workflow satu permohonan dan mengembalikan
// daftar alasan blokir. Lihat komentar tipe workflowRule untuk semantik
// fail-open/fail-closed.
func workflowGate(db *sql.DB, officeID, reqID int64, svcCode, kind string) []string {
	if db == nil || reqID <= 0 {
		return nil
	}
	gate := strings.ToLower(strings.TrimSpace(kind))
	var deedHint string
	switch gate {
	case "move", "approve", "ppat":
		// ok
	case "":
		gate = "move" // default konservatif: cek umum tanpa resi BPN
	default:
		// Bukan nama gate yang dikenal → anggap petunjuk deed-kind,
		// terapkan cek umum (tanpa resi BPN).
		deedHint = strings.TrimSpace(kind)
		gate = "move"
	}

	// Resolve service code bila kosong.
	svc := strings.ToUpper(strings.TrimSpace(svcCode))
	if svc == "" {
		var code sql.NullString
		if err := db.QueryRow(`SELECT s.code FROM service_requests r
			JOIN services s ON s.id=r.service_id
			WHERE r.id=? AND r.office_id=?`, reqID, officeID).Scan(&code); err != nil || !code.Valid {
			return nil // request tak dikenal → fail-open, handler hilir menolak
		}
		svc = strings.ToUpper(strings.TrimSpace(code.String))
		if svc == "" {
			return nil
		}
	}

	// Muat rule. Tabel hilang → fail-open penuh (DB lama pra-026).
	rule := defaultWorkflowRule()
	if err := db.QueryRow(`SELECT require_checklist, require_verified_identity,
		COALESCE(require_deed_kind,''), require_bpn_receipt, min_parties
		FROM service_workflow_rules WHERE service_code=?`, svc).Scan(
		&rule.requireChecklist, &rule.requireVerifiedIdentity,
		&rule.requireDeedKind, &rule.requireBPNReceipt, &rule.minParties); err != nil {
		if isTableMissing(err) {
			return nil
		}
		if err != sql.ErrNoRows {
			// Error non-fatal lain (koneksi dsb): jatuh ke default ketat
			// sedang, bukan fail-open — lihat keputusan desain #2/#3.
			rule = defaultWorkflowRule()
		} else {
			rule = defaultWorkflowRule()
		}
	}
	var blocks []string

	// Saklar mode ketat kantor (office_flags.strict_checklist, fail-open false
	// seperti legacy): bila MATI, gate "move" lolos total (perilaku lama
	// requestMove) dan gate "ppat" hanya menahan resi BPN (perilaku lama
	// ppatAdvance). Gate "approve" selalu penuh (deedApprove memang tanpa
	// switch sejak awal). Ini menjaga toggle di /office tetap bermakna.
	strict := workflowStrictEnabled(db, officeID)
	if !strict {
		if gate == "move" {
			return nil
		}
		if gate == "ppat" {
			if rule.requireBPNReceipt {
				if empties, found, ok := workflowBPNReceipts(db, officeID, reqID); ok && found && empties {
					blocks = append(blocks, `resi BPN kosong — nomor resi BPN asli wajib diisi sebelum lapor BPN (tanda "-" tidak diterima)`)
				}
			}
			return blocks
		}
	}

	// 1) Checklist: hitung status item template milik layanan (join sama
	// seperti dms.ChecklistForRequest agar konsisten dengannya).
	if rule.requireChecklist {
		miss, bad, ok := workflowChecklistCounts(db, officeID, reqID)
		if ok {
			if miss > 0 {
				blocks = append(blocks, fmt.Sprintf("masih ada %d dokumen berstatus MISSING — lengkapi di checklist bawah", miss))
			}
			if bad > 0 {
				blocks = append(blocks, fmt.Sprintf("masih ada %d dokumen berstatus REJECTED/NEEDS_REVISION — selesaikan tindak lanjut", bad))
			}
		}
		// ok == false (tabel checklist hilang/rusak) → lewati cek ini.
	}

	// Kumpulkan pihak: klien utama + matter_parties (distinct).
	partyIDs := workflowPartyIDs(db, officeID, reqID)

	// 2) Identitas: tiap pihak wajib punya ≥1 identitas VERIFIED.
	if rule.requireVerifiedIdentity {
		for _, cid := range partyIDs {
			ok, skipped := workflowIdentityVerified(db, cid)
			if skipped {
				continue // tabel identities hilang → lewati cek ini
			}
			if !ok {
				blocks = append(blocks, fmt.Sprintf("identitas belum VERIFIED: %s", workflowClientLabel(db, cid)))
			}
		}
	}

	// 3) Jumlah pihak minimum.
	if rule.minParties > 0 {
		total := workflowPartyCount(db, officeID, reqID, partyIDs)
		if total < rule.minParties {
			blocks = append(blocks, fmt.Sprintf("pihak kurang: baru %d dari minimal %d pihak — tambah pihak terkait", total, rule.minParties))
		}
	}

	// 4) Jenis akta: hanya bila rule menetapkan dan ada akta tertaut.
	if req := strings.TrimSpace(rule.requireDeedKind); req != "" {
		actual := workflowLinkedDeedKind(db, officeID, reqID, gate)
		if actual == "" {
			actual = deedHint
		}
		if actual != "" && !strings.EqualFold(strings.TrimSpace(actual), req) {
			blocks = append(blocks, fmt.Sprintf("deed_kind mismatch: aturan %s mewajibkan akta %s, draf aktual %s", svc, req, strings.TrimSpace(actual)))
		}
	}

	// 5) Resi BPN: hanya gate ppat; tanpa baris ppat tertaut → lewati
	// (cek resi eksplisit di handler tetap berlaku untuk nilai form).
	if rule.requireBPNReceipt && gate == "ppat" {
		if empties, found, ok := workflowBPNReceipts(db, officeID, reqID); ok && found && empties {
			blocks = append(blocks, `resi BPN kosong — nomor resi BPN asli wajib diisi sebelum lapor BPN (tanda "-" tidak diterima)`)
		}
	}

	return blocks
}

// workflowStrictEnabled membaca saklar mode ketat kantor (office_flags).
// Fail-open false bila tabel/baris hilang — sama seperti legacy agar
// kantor lama/tanpa flag tidak tiba-tiba terkunci.
func workflowStrictEnabled(db *sql.DB, officeID int64) bool {
	if db == nil || officeID <= 0 {
		return false
	}
	var v int
	if err := db.QueryRow(`SELECT strict_checklist FROM office_flags WHERE office_id=?`, officeID).Scan(&v); err != nil {
		return false
	}
	return v == 1
}

// workflowChecklistCounts menghitung item checklist: MISSING vs status buruk
// lain (REJECTED/NEEDS_REVISION/dll). ok=false bila tabel pendukung hilang.
func workflowChecklistCounts(db *sql.DB, officeID, reqID int64) (miss, bad int, ok bool) {
	rows, err := db.Query(`SELECT COALESCE(r.status,'MISSING'), COUNT(*)
		FROM doc_checklist_items i
		JOIN doc_checklist_templates t ON t.id=i.template_id
		JOIN service_requests sr ON sr.service_id=(SELECT id FROM services WHERE code=t.service_code LIMIT 1)
		LEFT JOIN doc_checklist_records r ON r.item_id=i.id AND r.request_id=sr.id
		WHERE t.office_id=? AND sr.id=? AND sr.office_id=?
		GROUP BY COALESCE(r.status,'MISSING')`, officeID, reqID, officeID)
	if err != nil {
		return 0, 0, false
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return 0, 0, false
		}
		switch strings.ToUpper(strings.TrimSpace(st)) {
		case "VERIFIED", "RECEIVED":
			// ok
		case "MISSING":
			miss += n
		default:
			bad += n
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, false
	}
	return miss, bad, true
}

// workflowPartyIDs mengembalikan ID klien pihak perkara: klien utama
// service_requests + baris matter_parties (distinct). Tabel matter_parties
// yang hilang → hanya klien utama.
func workflowPartyIDs(db *sql.DB, officeID, reqID int64) []int64 {
	var out []int64
	seen := map[int64]bool{}
	var main sql.NullInt64
	if err := db.QueryRow(`SELECT client_id FROM service_requests WHERE id=? AND office_id=?`, reqID, officeID).Scan(&main); err != nil || !main.Valid {
		return nil
	}
	out = append(out, main.Int64)
	seen[main.Int64] = true
	rows, err := db.Query(`SELECT DISTINCT client_id FROM matter_parties WHERE request_id=?`, reqID)
	if err != nil {
		return out // tabel hilang → fail-open per-cek
	}
	defer rows.Close()
	for rows.Next() {
		var cid sql.NullInt64
		if err := rows.Scan(&cid); err != nil || !cid.Valid || seen[cid.Int64] {
			continue
		}
		seen[cid.Int64] = true
		out = append(out, cid.Int64)
	}
	return out
}

// workflowPartyCount menghitung pihak distinct (klien utama + matter_parties).
// partyIDs boleh hasil workflowPartyIDs agar hemat query; bila tabel hilang
// total = jumlah partyIDs (minimal klien utama).
func workflowPartyCount(db *sql.DB, officeID, reqID int64, partyIDs []int64) int {
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM (
		SELECT r.client_id AS cid FROM service_requests r WHERE r.id=? AND r.office_id=?
		UNION
		SELECT mp.client_id FROM matter_parties mp WHERE mp.request_id=?
	) p`, reqID, officeID, reqID).Scan(&total)
	if err != nil {
		return len(partyIDs)
	}
	return total
}

// workflowIdentityVerified melaporkan apakah klien punya ≥1 identitas VERIFIED.
// skipped=true bila skema identities belum ada (fail-open per-cek).
// Mendukung skema lama (kolom verified TINYINT) maupun baru (status ENUM 016).
func workflowIdentityVerified(db *sql.DB, clientID int64) (verified, skipped bool) {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM identities WHERE client_id=? AND status='VERIFIED'`, clientID).Scan(&n); err == nil {
		return n > 0, false
	} else if !isTableMissing(err) && !isUnknownColumn(err) {
		// Error tak dikenal (bukan soal skema) → lewati cek, jangan kunci.
		return false, true
	}
	// Fallback skema lama pra-016.
	if err := db.QueryRow(`SELECT COUNT(*) FROM identities WHERE client_id=? AND verified=1`, clientID).Scan(&n); err != nil {
		return false, true
	}
	return n > 0, false
}

// workflowClientLabel me-label klien untuk pesan blokir (tanpa NIK — UU PDP).
func workflowClientLabel(db *sql.DB, clientID int64) string {
	var name sql.NullString
	if err := db.QueryRow(`SELECT full_name FROM clients WHERE id=?`, clientID).Scan(&name); err != nil || !name.Valid || strings.TrimSpace(name.String) == "" {
		return fmt.Sprintf("klien #%d", clientID)
	}
	return fmt.Sprintf("%s (klien #%d)", strings.TrimSpace(name.String), clientID)
}

// workflowLinkedDeedKind membaca jenis akta tertaut permohonan:
// gate ppat → ppat_register.jenis (fallback deed_drafts.kind),
// gate lain → deed_drafts.kind (draf pertama). Kosong = tak ada tautan.
func workflowLinkedDeedKind(db *sql.DB, officeID, reqID int64, gate string) string {
	if gate == "ppat" {
		var jenis sql.NullString
		if err := db.QueryRow(`SELECT p.jenis FROM ppat_register p
			JOIN deed_drafts d ON d.id=p.deed_id
			WHERE d.request_id=? AND d.office_id=? AND p.office_id=?
			ORDER BY p.id LIMIT 1`, reqID, officeID, officeID).Scan(&jenis); err == nil && jenis.Valid {
			if s := strings.TrimSpace(jenis.String); s != "" {
				return s
			}
		} else if err != nil && !isTableMissing(err) && err != sql.ErrNoRows {
			// error non-skema → abaikan lookup ini, lanjut fallback
		}
	}
	var kind sql.NullString
	if err := db.QueryRow(`SELECT kind FROM deed_drafts WHERE request_id=? AND office_id=? ORDER BY id LIMIT 1`, reqID, officeID).Scan(&kind); err != nil || !kind.Valid {
		return ""
	}
	return strings.TrimSpace(kind.String)
}

// workflowBPNReceipts memeriksa resi BPN pada baris ppat tertaut permohonan.
// found=false bila tak ada baris ppat tertaut (cek dilewati); ok=false bila
// tabel pendukung hilang (fail-open per-cek). empties=true bila semua resi
// kosong atau "-" (tanda fiktif, konsisten dengan gate lama).
func workflowBPNReceipts(db *sql.DB, officeID, reqID int64) (empties, found, ok bool) {
	rows, err := db.Query(`SELECT COALESCE(p.bpn_receipt,'') FROM ppat_register p
		JOIN deed_drafts d ON d.id=p.deed_id
		WHERE d.request_id=? AND d.office_id=? AND p.office_id=?`, reqID, officeID, officeID)
	if err != nil {
		return false, false, false
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var bpn sql.NullString
		if err := rows.Scan(&bpn); err != nil {
			return false, false, false
		}
		n++
		v := strings.TrimSpace(bpn.String)
		if !bpn.Valid || v == "" || v == "-" {
			continue
		}
		// Ada minimal satu resi asli → tidak kosong.
		return false, true, true
	}
	if err := rows.Err(); err != nil {
		return false, false, false
	}
	if n == 0 {
		return false, false, true
	}
	return true, true, true
}
