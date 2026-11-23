// Test integrasi Workflow Engine (migrasi 027) melawan MySQL asli.
// Mengikuti pola TestMain SKIP yang sama: tanpa NOTARYOS_TEST_DSN,
// seluruh paket diskip oleh TestMain di integration_test.go (BUKAN hijau).
package integration

import (
	"testing"

	"notaryos/internal/clients"
	"notaryos/internal/requests"
	"notaryos/internal/users"
)

func TestWorkflowInstanceLifecycle(t *testing.T) {
	clean(t)

	// Seed layanan khusus workflow; IGNORE agar idempoten antar run.
	if _, err := db.Exec(`INSERT IGNORE INTO services(code, category, name) VALUES('TEST-WF','other','Layanan Uji Workflow')`); err != nil {
		t.Fatal(err)
	}
	var sid int64
	if err := db.QueryRow(`SELECT id FROM services WHERE code='TEST-WF'`).Scan(&sid); err != nil {
		t.Skip("seed services TEST-WF tidak tersedia di test db")
	}

	office := seedOffice(t)
	uid, err := users.NewStore(db).Create(&office, "staff", "Staf WF", "wf@t.id", "", "Rahasia123")
	if err != nil {
		t.Fatal(err)
	}
	cid, err := clients.NewStore(db).Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "WF Tester", Phone: "0888000088"})
	if err != nil {
		t.Fatal(err)
	}
	rid, err := requests.NewStore(db).Create(office, cid, sid, "Permohonan WF", "", &uid)
	if err != nil {
		t.Fatal(err)
	}

	// Template + 3 steps berurutan.
	res, err := db.Exec(`INSERT INTO workflow_templates(office_id, service_code, name, version, active)
		VALUES(?,'TEST-WF','Template Uji WF',1,1)`, office)
	if err != nil {
		t.Fatal(err)
	}
	tplID, _ := res.LastInsertId()
	steps := []struct {
		key   string
		label string
		order int
	}{
		{"INTAKE", "Penerimaan", 1},
		{"VERIFY", "Verifikasi", 2},
		{"COMPLETE", "Penyelesaian", 3},
	}
	for _, s := range steps {
		if _, err := db.Exec(`INSERT INTO workflow_steps(template_id, step_key, label, sort_order, required_role)
			VALUES(?,?,?,?,'staff')`, tplID, s.key, s.label, s.order); err != nil {
			t.Fatal(err)
		}
	}

	// Instance mulai di step pertama; semua state PENDING.
	res, err = db.Exec(`INSERT INTO workflow_instances(office_id, request_id, template_id, current_step, status)
		VALUES(?,?,?, 'INTAKE','ACTIVE')`, office, rid, tplID)
	if err != nil {
		t.Fatal(err)
	}
	instID, _ := res.LastInsertId()
	for _, s := range steps {
		if _, err := db.Exec(`INSERT INTO workflow_step_states(instance_id, step_key, status)
			VALUES(?,?,'PENDING')`, instID, s.key); err != nil {
			t.Fatal(err)
		}
	}
	advance := func(doneKey, nextKey, actor string) {
		t.Helper()
		if _, err := db.Exec(`UPDATE workflow_step_states SET status='DONE', actor=?, note=?
			WHERE instance_id=? AND step_key=?`, actor, "maju "+doneKey, instID, doneKey); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE workflow_instances SET current_step=? WHERE id=?`, nextKey, instID); err != nil {
			t.Fatal(err)
		}
	}

	// Majukan 2 step: INTAKE -> VERIFY -> COMPLETE.
	advance("INTAKE", "VERIFY", "staf-wf")
	advance("VERIFY", "COMPLETE", "staf-wf")

	// Verifikasi state DB: current_step + per-step status.
	var current, istatus string
	if err := db.QueryRow(`SELECT current_step, status FROM workflow_instances WHERE id=?`, instID).Scan(&current, &istatus); err != nil {
		t.Fatal(err)
	}
	if current != "COMPLETE" || istatus != "ACTIVE" {
		t.Fatalf("instance salah: current_step=%q status=%q, harus COMPLETE/ACTIVE", current, istatus)
	}
	got := map[string]string{}
	rows, err := db.Query(`SELECT step_key, status FROM workflow_step_states WHERE instance_id=?`, instID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var k, s string
		if err := rows.Scan(&k, &s); err != nil {
			t.Fatal(err)
		}
		got[k] = s
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got["INTAKE"] != "DONE" || got["VERIFY"] != "DONE" || got["COMPLETE"] != "PENDING" {
		t.Fatalf("step_states salah: %+v", got)
	}

	// Cleanup targeted agar idempoten (provisioned rows) — service TEST-WF dibiarkan (seed IGNORE).
	for _, q := range []struct {
		query string
		args  []any
	}{
		{`DELETE FROM workflow_step_states WHERE instance_id=?`, []any{instID}},
		{`DELETE FROM workflow_instances WHERE id=?`, []any{instID}},
		{`DELETE FROM workflow_steps WHERE template_id=?`, []any{tplID}},
		{`DELETE FROM workflow_templates WHERE id=?`, []any{tplID}},
		{`DELETE FROM request_events WHERE request_id=?`, []any{rid}},
		{`DELETE FROM service_requests WHERE id=?`, []any{rid}},
		{`DELETE FROM clients WHERE id=?`, []any{cid}},
		{`DELETE FROM users WHERE id=?`, []any{uid}},
		{`DELETE FROM offices WHERE id=?`, []any{office}},
	} {
		if _, err := db.Exec(q.query, q.args...); err != nil {
			t.Fatal("cleanup:", err)
		}
	}
}
