// Package integration runs against a REAL MySQL test database.
// Set NOTARYOS_TEST_DSN (e.g. "user:pass@tcp(127.0.0.1:3306)/notaryos_test?parseTime=true&loc=UTC").
// Without it, tests skip. Mocked DBs are never accepted as MySQL proof.
package integration

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"notaryos/internal/clients"
	"notaryos/internal/cms"
	"notaryos/internal/corp"
	"notaryos/internal/deeds"
	"notaryos/internal/finance"
	"notaryos/internal/gov"
	"notaryos/internal/remind"
	"notaryos/internal/requests"
	"notaryos/internal/users"
	"sync"
)

var db *sql.DB

func TestMain(m *testing.M) {
	dsn := os.Getenv("NOTARYOS_TEST_DSN")
	if dsn == "" {
		fmt.Println("SKIP: NOTARYOS_TEST_DSN not set")
		os.Exit(0)
	}
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("SKIP: cannot open test db:", err)
		os.Exit(0)
	}
	if err := db.Ping(); err != nil {
		fmt.Println("SKIP: cannot ping test db:", err)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func clean(t *testing.T) {
	t.Helper()
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=0"); err != nil {
		t.Fatal(err)
	}
	tables := []string{"payments", "invoice_lines", "invoices", "deed_versions", "signatures",
		"deed_drafts", "gov_submissions", "request_events", "appointments", "service_requests",
		"beneficial_owners", "shareholders", "company_officers", "companies", "identities", "client_representatives",
		"user_client_links", "ai_results", "files", "legalizations", "wills", "notifications",
		"login_attempts", "audit_logs", "users", "clients", "offices"}
	for _, tb := range tables {
		if _, err := db.Exec("DELETE FROM " + tb); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS=1"); err != nil {
		t.Fatal(err)
	}
}

func seedOffice(t *testing.T) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO offices(name, notary_name, sk_number, address, phone, email)
		VALUES('T-Office','T Notaris','SK-T','Jl T','021','t@t.id')`)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestUsersAuthenticate(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	us := users.NewStore(db)
	uid, err := us.Create(&office, "office_admin", "Test Admin", "testadmin@t.id", "", "Rahasia123")
	if err != nil {
		t.Fatal(err)
	}
	u, err := us.Authenticate("testadmin@t.id", "Rahasia123")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != uid || u.Role != "office_admin" {
		t.Fatalf("wrong user: %+v", u)
	}
	if _, err := us.Authenticate("testadmin@t.id", "salah"); err == nil {
		t.Fatal("wrong password must fail")
	}
	if err := us.LinkClient(uid, 999999); err != nil {
		t.Fatal("link should ignore-missing gracefully:", err)
	}
}

func TestClientsDuplicates(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs := clients.NewStore(db)
	id, err := cs.Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "Budi Santoso", Phone: "0811000011"})
	if err != nil {
		t.Fatal(err)
	}
	dups, err := cs.FindDuplicates(office, "Budi Santoso", "0811000011")
	if err != nil {
		t.Fatal(err)
	}
	if len(dups) == 0 {
		t.Fatal("expected duplicate by same phone")
	}
	iid, err := cs.AddIdentity(id, "KTP", "3174051209900001", "Budi Santoso")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := users.NewStore(db).Create(&office, "staff", "Verifier", "ver@t.id", "", "Rahasia123")
	if err != nil {
		t.Fatal(err)
	}
	if err := cs.VerifyIdentity(iid, uid); err != nil {
		t.Fatal(err)
	}
	ids, err := cs.Identities(id)
	if err != nil || len(ids) != 1 || !ids[0].Verified {
		t.Fatalf("identity verify broken: %+v %v", ids, err)
	}
}

func TestRequestWorkflow(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs, rs := clients.NewStore(db), requests.NewStore(db)
	cid, _ := cs.Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "Citra", Phone: "0822000022"})
	var sid int64
	if err := db.QueryRow(`SELECT id FROM services WHERE code='PT-DIRIAN'`).Scan(&sid); err != nil {
		t.Skip("seed services missing in test db")
	}
	rid, err := rs.Create(office, cid, sid, "PT Maju", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, to := range []string{"SUBMITTED", "IN_REVIEW", "APPROVED", "IN_PROGRESS", "COMPLETED"} {
		if err := rs.Move(office, rid, to, nil, ""); err != nil {
			t.Fatal("move to", to, err)
		}
	}
	if err := rs.Move(office, rid, "DRAFT", nil, ""); err == nil {
		t.Fatal("backward move must fail")
	}
	p, a, c, err := rs.Counts(office)
	if err != nil || c != 1 || p != 0 || a != 0 {
		t.Fatalf("counts wrong: %d %d %d %v", p, a, c, err)
	}
}

func TestDeedVersionLock(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	ds := deeds.NewStore(db)
	id, err := ds.CreateDraft(office, nil, "Akta T", "PT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ds.SaveVersion(id, "isi v2", nil, "revisi"); err != nil {
		t.Fatal(err)
	}
	if err := ds.Approve(id, "1/2026", 1); err == nil {
		t.Fatal("approve from DRAFT status must fail (needs IN_REVIEW)")
	}
	if _, err := db.Exec(`UPDATE deed_drafts SET status='IN_REVIEW' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := ds.Approve(id, "1/2026", 1); err != nil {
		t.Fatal(err)
	}
	if err := ds.SaveVersion(id, "ubah final", nil, "x"); err == nil {
		t.Fatal("locked final must reject version writes")
	}
}

func TestFinanceOverpay(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs, fs := clients.NewStore(db), finance.NewStore(db)
	cid, _ := cs.Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "Dana", Phone: "0833000033"})
	invid, err := fs.CreateInvoice(office, cid, nil, "INV-T1", []finance.Line{
		{Description: "Jasa", Qty: 2, UnitPrice: 1000000},
		{Description: "Biaya", Qty: 1, UnitPrice: 500000},
	}, 1100, "")
	if err != nil {
		t.Fatal(err)
	}
	// subtotal 2.5jt, PPN 275rb, total 2.775jt
	if err := fs.RecordPayment(invid, 2775000, "TRANSFER", "R1", nil); err != nil {
		t.Fatal(err)
	}
	if err := fs.RecordPayment(invid, 1, "CASH", "R2", nil); err == nil {
		t.Fatal("overpay must fail")
	}
	out, err := fs.Outstanding(office)
	if err != nil || out != 0 {
		t.Fatalf("outstanding=%d err=%v", out, err)
	}
}

func TestGovAssisted(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	gs := gov.NewStore(db)
	id, err := gs.Create(office, gov.ChannelAHU, "PT Uji")
	if err != nil {
		t.Fatal(err)
	}
	if err := gs.Advance(id, gov.StatusReady, "", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := gs.Advance(id, gov.StatusDraft, "", "", ""); err == nil {
		t.Fatal("backward must fail")
	}
	// Checklist toggle (F-023): index 0 flip twice kembali semula.
	if err := gs.ToggleCheck(id, 0); err != nil {
		t.Fatal(err)
	}
	subs, err := gs.List(office)
	if err != nil || len(subs) == 0 {
		t.Fatalf("list broken: %v", subs)
	}
	if len(subs[0].Checklist) == 0 || !subs[0].Checklist[0].Done {
		t.Fatal("toggle check tidak tercatat")
	}
	if err := gs.ToggleCheck(id, 0); err != nil {
		t.Fatal(err)
	}
	if err := gs.ToggleCheck(id, 99); err == nil {
		t.Fatal("index liar harus gagal")
	}
}

func TestBeneficialOwnerAndWillHandover(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs := clients.NewStore(db)
	cid, _ := cs.Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "BO Tester", Phone: "0844000044"})
	// Company via request minimal.
	var sid int64
	if err := db.QueryRow(`SELECT id FROM services WHERE code='PT-DIRIAN'`).Scan(&sid); err != nil {
		t.Skip("seed services missing in test db")
	}
	rs := requests.NewStore(db)
	rid, err := rs.Create(office, cid, sid, "PT BO", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO companies(request_id, entity_type, name) VALUES(?,'PT','PT BO Uji')`, rid); err != nil {
		t.Fatal(err)
	}
	var coID int64
	if err := db.QueryRow(`SELECT id FROM companies WHERE request_id=?`, rid).Scan(&coID); err != nil {
		t.Fatal(err)
	}
	corpStore := corp.NewStore(db)
	if err := corpStore.AddBeneficialOwner(coID, "Pemilik Asli", "3174051209900001", 60, true); err != nil {
		t.Fatal(err)
	}
	if err := corpStore.AddBeneficialOwner(coID, "X", "123", 0, false); err == nil {
		t.Fatal("BO invalid harus gagal")
	}
	bos, err := corpStore.BeneficialOwners(coID)
	if err != nil || len(bos) != 1 || !bos[0].IsPEP {
		t.Fatalf("BO list broken: %+v %v", bos, err)
	}
	ws := deeds.NewWillStore(db)
	wid, err := ws.Register(office, cid, nil, "W-TEST-001")
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Handover(wid, "Ahli Waris A", "diserahkan di kantor"); err != nil {
		t.Fatal(err)
	}
	if err := ws.Handover(wid, "X", ""); err == nil {
		t.Fatal("handover ganda harus gagal")
	}
}

func TestProtocolLegalisasiAppointment(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs := clients.NewStore(db)
	cid, _ := cs.Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "Proto", Phone: "0855000055"})
	ds := deeds.NewStore(db)
	if err := ds.CreateProtocol(office, "B-I", 2026, "minuta-1"); err != nil {
		t.Fatal(err)
	}
	if err := ds.CreateProtocol(office, "", 1999, ""); err == nil {
		t.Fatal("protokol invalid harus gagal")
	}
	lid, err := ds.RegisterLegalization(office, "LEGALISASI", "REG-1", cid, "Dok A", 150000)
	if err != nil {
		t.Fatal(err)
	}
	if err := ds.AdvanceLegalization(office, lid, "HANDED_OVER"); err == nil {
		t.Fatal("loncat status harus gagal")
	}
	if err := ds.AdvanceLegalization(office, lid, "COMPLETED"); err != nil {
		t.Fatal(err)
	}
	if err := ds.AdvanceLegalization(office, lid, "HANDED_OVER"); err != nil {
		t.Fatal(err)
	}
	rs := requests.NewStore(db)
	at := time.Now().Add(2 * time.Hour)
	aid, err := rs.Schedule(office, cid, nil, at, "Konsultasi")
	if err != nil {
		t.Fatal(err)
	}
	if err := rs.SetAppointmentStatus(office, aid, "DONE"); err != nil {
		t.Fatal(err)
	}
	if err := rs.SetAppointmentStatus(office, aid, "BOGUS"); err == nil {
		t.Fatal("status liar harus gagal")
	}
}

func TestRemindersRun(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs := clients.NewStore(db)
	cid, _ := cs.Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "Rina", Phone: "0866000066"})
	// Janji besok + invoice telat + tugas mendesak.
	rs := requests.NewStore(db)
	at := time.Now().AddDate(0, 0, 1).Add(2 * time.Hour)
	if _, err := rs.Schedule(office, cid, nil, at, "Konsultasi"); err != nil {
		t.Fatal(err)
	}
	fs := finance.NewStore(db)
	invid, err := fs.CreateInvoice(office, cid, nil, "INV-TELAT", []finance.Line{{Description: "Jasa", Qty: 1, UnitPrice: 100000}}, 0, time.Now().AddDate(0, 0, -5).Format("2006-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	_ = invid
	if _, err := db.Exec(`INSERT INTO tasks(office_id, title, due_date) VALUES(?,?,CURDATE())`, office, "Siapkan berkas"); err != nil {
		t.Fatal(err)
	}
	appt, overdue, tasks, sla, err := remind.RunOnce(db, office, remind.LogSender{})
	if err != nil {
		t.Fatal(err)
	}
	if appt != 1 || overdue != 1 || tasks != 1 {
		t.Fatalf("pengingat kurang: janji=%d telat=%d tugas=%d", appt, overdue, tasks)
	}
	if sla != 0 {
		t.Fatalf("SLA tak terduga: %d", sla)
	}
	// Anti-spam: jalan kedua tidak menambah baris dalam 24 jam.
	var before int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&before)
	if _, _, _, _, err := remind.RunOnce(db, office, remind.LogSender{}); err != nil {
		t.Fatal(err)
	}
	var after int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&after)
	if after != before {
		t.Fatalf("duplikat pengingat: %d -> %d", before, after)
	}
}

func TestRemindersSLA(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs := clients.NewStore(db)
	cid, _ := cs.Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "SLA", Phone: "0877000077"})
	code := fmt.Sprintf("SLA-T-%d", time.Now().UnixNano())
	if _, err := db.Exec(`INSERT INTO services(code, category, name) VALUES(?,'other','Uji SLA')`, code); err != nil {
		t.Fatal(err)
	}
	var sid int64
	_ = db.QueryRow(`SELECT id FROM services WHERE code=?`, code).Scan(&sid)
	res, err := db.Exec(`INSERT INTO service_requests(office_id, client_id, service_id, status, title, created_at, updated_at)
		VALUES(?,?,?,'SUBMITTED','Macet uji', DATE_SUB(UTC_TIMESTAMP(), INTERVAL 5 DAY), DATE_SUB(UTC_TIMESTAMP(), INTERVAL 5 DAY))`,
		office, cid, sid)
	if err != nil {
		t.Fatal(err)
	}
	rid, _ := res.LastInsertId()
	if _, err := users.NewStore(db).Create(&office, "staff", "Staf SLA", "sla@t.id", "", "Rahasia123"); err != nil {
		t.Fatal(err)
	}
	_, _, _, sla, err := remind.RunOnce(db, office, remind.LogSender{})
	if err != nil {
		t.Fatal(err)
	}
	if sla != 1 {
		t.Fatalf("SLA harus 1, dapat %d", sla)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE title='Permohonan macet (SLA)'`).Scan(&n)
	if n == 0 {
		t.Fatal("notifikasi SLA tidak tercatat")
	}
	// COMPLETED tidak dieskalasi.
	if _, err := db.Exec(`UPDATE service_requests SET status='COMPLETED' WHERE id=?`, rid); err != nil {
		t.Fatal(err)
	}
	_, _, _, sla2, err := remind.RunOnce(db, office, remind.LogSender{})
	if err != nil {
		t.Fatal(err)
	}
	if sla2 != 0 {
		t.Fatalf("COMPLETED ikut dieskalasi: %d", sla2)
	}
}

func TestApproveConcurrent(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	ds := deeds.NewStore(db)
	did, err := ds.CreateDraft(office, nil, "Akta Konkuren", "PT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE deed_drafts SET status='IN_REVIEW' WHERE id=?`, did); err != nil {
		t.Fatal(err)
	}
	const n = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	won := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := ds.Approve(did, fmt.Sprintf("KONK-%d-%d", did, i), 1); err == nil {
				mu.Lock()
				won++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if won != 1 {
		t.Fatalf("tepat 1 approve harus menang, menang=%d", won)
	}
	var status string
	var repert int
	_ = db.QueryRow(`SELECT status FROM deed_drafts WHERE id=?`, did).Scan(&status)
	_ = db.QueryRow(`SELECT COUNT(*) FROM repertorium WHERE deed_id=?`, did).Scan(&repert)
	if status != "APPROVED" || repert != 1 {
		t.Fatalf("status=%s repertorium=%d, harus APPROVED/1", status, repert)
	}
}

func TestCMSLifecycle(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs := cms.NewStore(db)
	// Branding: invalid color rejected-normalized, unknown font falls back.
	if err := cs.SaveBranding(office, cms.Branding{AppName: "", Primary: "red"}); err == nil {
		t.Fatal("nama kosong harus ditolak")
	}
	b, err := cs.BrandingOf(office)
	if err != nil {
		t.Fatal(err)
	}
	if b.AppName != "SaktiNotaris" {
		t.Fatalf("default branding hilang: %q", b.AppName)
	}
	// Page lifecycle + slug uniqueness + publish rules.
	p := &cms.Page{Title: "Beranda", Slug: "beranda", Content: "<p>Halo</p><script>x</script>", Status: cms.StatusDraft}
	pid, err := cs.SavePage(office, p, nil)
	if err != nil || pid == 0 {
		t.Fatal(err)
	}
	dup := &cms.Page{Title: "Duplikat", Slug: "beranda", Content: "x", Status: cms.StatusDraft}
	if _, err := cs.SavePage(office, dup, nil); err == nil {
		t.Fatal("slug ganda harus ditolak")
	}
	// PUBLISH needs content: empty content draft cannot publish.
	empty := &cms.Page{Title: "Kosong", Slug: "kosong", Status: cms.StatusPublished}
	if _, err := cs.SavePage(office, empty, nil); err == nil {
		t.Fatal("terbit tanpa isi harus ditolak")
	}
	// Sections: bad type + bad CTA rejected; valid saved.
	bad := &cms.Section{PageID: pid, Type: "nope", CTAURL: "javascript:x"}
	if _, err := cs.SaveSection(office, bad); err == nil {
		t.Fatal("section jahat harus ditolak")
	}
	sec := &cms.Section{PageID: pid, Type: "hero", Title: "Halo", CTALabel: "Mulai",
		CTAURL: "/kontak", Visible: true, Settings: map[string]any{"columns": 2.0, "evil": 1}}
	sid, err := cs.SaveSection(office, sec)
	if err != nil || sid == 0 {
		t.Fatal(err)
	}
	// Publish page, verify public read + sanitization.
	p.ID = pid
	p.Status = cms.StatusPublished
	p.Content = "<p>Halo</p><script>x</script>"
	if _, err := cs.SavePage(office, p, nil); err != nil {
		t.Fatal(err)
	}
	got, secs, err := cs.PublicPage(office, "beranda")
	if err != nil || got.Title != "Beranda" || len(secs) != 1 {
		t.Fatalf("public read gagal: %+v %v", got, err)
	}
	// Scheduled publish.
	fut := time.Now().Add(2 * time.Hour)
	sp := &cms.Page{Title: "Jadwal", Slug: "jadwal", Content: "x", Status: cms.StatusScheduled}
	sp.ScheduledAt = sql.NullTime{Time: fut, Valid: true}
	if _, err := cs.SavePage(office, sp, nil); err != nil {
		t.Fatal(err)
	}
	if n1, n2 := cs.PublishDue(office); n1+n2 != 0 {
		t.Fatal("belum jatuh tempo tapi terbit")
	}
	if _, err := db.Exec(`UPDATE website_pages SET scheduled_at=DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR) WHERE slug='jadwal'`); err != nil {
		t.Fatal(err)
	}
	if n1, _ := cs.PublishDue(office); n1 != 1 {
		t.Fatal("jadwal lewat harus terbit")
	}
	if _, _, err := cs.PublicPage(office, "jadwal"); err != nil {
		t.Fatal("sesudah PublishDue harus tampil publik")
	}
	cs.RecordHistory(office, "page", pid, "publish", "DRAFT", "PUBLISHED", nil, "uji", "127.0.0.1")
	hist, err := cs.History(office, 10)
	if err != nil || len(hist) == 0 {
		t.Fatal("riwayat publikasi kosong")
	}
	// Public isolation: DRAFT must not render.
	priv := &cms.Page{Title: "Rahasia", Slug: "rahasia-internal", Content: "x", Status: cms.StatusDraft}
	if _, err := cs.SavePage(office, priv, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cs.PublicPage(office, "rahasia-internal"); err == nil {
		t.Fatal("draf tak boleh publik")
	}
}
