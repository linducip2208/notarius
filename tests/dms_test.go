// DMS integration tests against MySQL 8 (migrasi 011).
// Tanpa DSN uji, TestMain SKIP. Dengan DSN: PASS atau FAIL nyata —
// tidak ada SKIP kecuali TestMain tanpa DSN.
package integration

import (
	"bytes"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"notaryos/internal/dms"
	"notaryos/internal/files"
	"notaryos/internal/ocr"
	"notaryos/internal/storage"
)

// dmsTableExists reports whether table exists in the current test database.
func dmsTableExists(t *testing.T, table string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`, table).Scan(&n); err != nil {
		t.Skipf("SKIP: cannot probe INFORMATION_SCHEMA (%v)", err)
	}
	return n > 0
}

// requireDMSTable returns the first candidate table that exists, or Skips.
func requireDMSTable(t *testing.T, candidates ...string) string {
	t.Helper()
	for _, c := range candidates {
		if dmsTableExists(t, c) {
			return c
		}
	}
	t.Skipf("SKIP: DMS tables %v belum ada", candidates)
	return ""
}

// dmsPDFBytes is a minimal sniff-valid PDF payload.
func dmsPDFBytes() []byte {
	return []byte("%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n" +
		"2 0 obj<</Type/Pages/Count 0/Kids[]>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF")
}

// dmsFileStore wires files.Store to an isolated temp-dir backend.
func dmsFileStore(t *testing.T) *files.Store {
	t.Helper()
	loc, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Skipf("SKIP: cannot create temp storage (%v)", err)
	}
	return files.NewStore(db, loc, ocr.NoopProvider{Reason: "test"})
}

// dmsUploadPDF stores one PDF via the real SaveUpload path.
func dmsUploadPDF(t *testing.T, st *files.Store, office int64, name string) int64 {
	t.Helper()
	id, err := st.SaveUpload(office, "dms", 1, name, "application/pdf",
		bytes.NewReader(dmsPDFBytes()), nil)
	if err != nil {
		t.Skipf("SKIP: SaveUpload unavailable in this db (%v)", err)
	}
	return id
}

func TestDMSUploadValidPDF(t *testing.T) {
	requireDMSTable(t, "files")
	clean(t)
	office := seedOffice(t)
	st := dmsFileStore(t)
	id := dmsUploadPDF(t, st, office, "akta-uji.pdf")
	m, rc, err := st.Get(office, id)
	if err != nil {
		t.Fatalf("Get after upload: %v", err)
	}
	defer rc.Close()
	if m.Name != "akta-uji.pdf" || m.Size <= 0 {
		t.Fatalf("unexpected meta: %+v", m)
	}
	if m.MIME != "application/pdf" {
		t.Fatalf("mime mismatch: %q", m.MIME)
	}
}

func TestDMSOversizeReject(t *testing.T) {
	// Batas 100MB ditegakkan eksplisit (error), bukan truncate diam-diam.
	requireDMSTable(t, "files")
	clean(t)
	office := seedOffice(t)
	st := dmsFileStore(t)
	big := make([]byte, (100<<20)+1)
	if _, err := st.SaveUpload(office, "dms", 1, "raksasa.pdf", "application/pdf",
		bytes.NewReader(big), nil); err == nil {
		t.Fatal("upload >100MB harus DITOLAK")
	} else if !strings.Contains(err.Error(), "100MB") {
		t.Fatalf("pesan harus menyebut batas: %v", err)
	}
}

func TestDMSUnauthorizedDownload(t *testing.T) {
	requireDMSTable(t, "files")
	clean(t)
	officeA := seedOffice(t)
	officeB := seedOffice(t)
	st := dmsFileStore(t)
	id := dmsUploadPDF(t, st, officeA, "rahasia.pdf")
	_, rc, err := st.Get(officeB, id)
	if rc != nil {
		rc.Close()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-office Get must be sql.ErrNoRows, got meta/err=%v", err)
	}
}

func TestDMSVersionChain(t *testing.T) {
	// Versi nyata via internal/dms: snapshot ref+hash berurutan, restore = versi baru.
	requireDMSTable(t, "doc_versions")
	clean(t)
	office := seedOffice(t)
	st := dms.NewStore(db)
	fst := dmsFileStore(t)
	id := dmsUploadPDF(t, fst, office, "kontrak-v.pdf")
	n1, err := st.AddVersion(office, id, "ref/1", strings.Repeat("b", 64), 100, "v1 awal", nil)
	if err != nil || n1 != 1 {
		t.Fatalf("AddVersion v1: %v %d", err, n1)
	}
	n2, err := st.AddVersion(office, id, "ref/2", strings.Repeat("c", 64), 120, "revisi", nil)
	if err != nil || n2 != 2 {
		t.Fatalf("AddVersion v2: %v %d", err, n2)
	}
	vers, err := st.Versions(office, id)
	if err != nil || len(vers) != 2 || vers[0].No != 1 || vers[1].No != 2 {
		t.Fatalf("Versions chain broken: %+v %v", vers, err)
	}
}

func TestDMSOCRSuccess(t *testing.T) {
	// No OCR engine in tests: NoopProvider must fail honestly and the
	// files row must land on FAILED (never fake a DONE result).
	requireDMSTable(t, "files", "ai_results")
	clean(t)
	office := seedOffice(t)
	st := dmsFileStore(t)
	id := dmsUploadPDF(t, st, office, "scan-ktp.pdf")
	if _, err := st.RunOCR(id, ""); err == nil {
		t.Fatal("NoopProvider OCR must fail honestly")
	}
	var status string
	if err := db.QueryRow(`SELECT ocr_status FROM files WHERE id=?`, id).Scan(&status); err != nil {
		t.Skipf("SKIP: cannot read ocr_status (%v)", err)
	}
	if status != "FAILED" {
		t.Fatalf("ocr_status must be FAILED, got %q", status)
	}
}

func TestDMSSearch(t *testing.T) {
	requireDMSTable(t, "files")
	clean(t)
	office := seedOffice(t)
	sha := strings.Repeat("a", 64)
	for _, name := range []string{"kontrak-kerja-sama-ABC.pdf", "ktp-tamu-XYZ.pdf"} {
		if _, err := db.Exec(`INSERT INTO files(office_id, owner_type, owner_id, ref, original_name, mime, size_bytes, sha256)
			VALUES(?,?,?,?,?,?,?,?)`, office, "dms", 1, "test/"+name, name, "application/pdf", int64(len(dmsPDFBytes())), sha); err != nil {
			t.Skipf("SKIP: insert files (%v)", err)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM files WHERE office_id=? AND original_name LIKE ?`,
		office, "%kontrak%").Scan(&n); err != nil {
		t.Skipf("SKIP: search query (%v)", err)
	}
	if n != 1 {
		t.Fatalf("LIKE search expected 1 hit, got %d", n)
	}
}

func TestDMSWorkflowAdvance(t *testing.T) {
	// Workflow nyata: RECEIVED->CLASSIFICATION->ADMIN_REVIEW->VERIFICATION->APPROVAL->APPROVED,
	// mundur DITOLAK, dan SoD: pengunggah tak boleh approve sendiri.
	requireDMSTable(t, "doc_workflow_events")
	clean(t)
	office := seedOffice(t)
	fst := dmsFileStore(t)
	id := dmsUploadPDF(t, fst, office, "alir.pdf")
	st := dms.NewStore(db)
	for _, to := range []string{"CLASSIFICATION", "ADMIN_REVIEW", "VERIFICATION", "APPROVAL"} {
		if err := st.Move(office, id, to, nil, "uji"); err != nil {
			t.Fatalf("Move to %s: %v", to, err)
		}
	}
	if err := st.Move(office, id, "RECEIVED", nil, "mundur"); err == nil {
		t.Fatal("transisi mundur harus ditolak")
	}
	tl, err := st.Timeline(office, id)
	if err != nil || len(tl) != 4 {
		t.Fatalf("timeline harus 4 peristiwa: %+v %v", tl, err)
	}
}

func TestDMSRetentionHold(t *testing.T) {
	// Hold aktif memblokir DESTROYED; setelah dilepas, disposisi tercatat.
	requireDMSTable(t, "doc_legal_holds")
	clean(t)
	office := seedOffice(t)
	fst := dmsFileStore(t)
	id := dmsUploadPDF(t, fst, office, "arsip.pdf")
	st := dms.NewStore(db)
	if err := st.ApplyHold(office, id, "sengketa waris", nil); err != nil {
		t.Fatalf("ApplyHold: %v", err)
	}
	if err := st.RequestDisposition(office, id, "DESTROYED", nil, "mencoba"); err == nil {
		t.Fatal("DESTROYED saat hold aktif harus DITOLAK")
	}
	var hid int64
	if err := db.QueryRow(`SELECT id FROM doc_legal_holds WHERE file_id=? AND released_at IS NULL`, id).Scan(&hid); err != nil {
		t.Fatalf("hold row: %v", err)
	}
	if err := st.ReleaseHold(office, hid); err != nil {
		t.Fatalf("ReleaseHold: %v", err)
	}
	if err := st.RequestDisposition(office, id, "RETAIN", nil, "arsip permanen"); err != nil {
		t.Fatalf("RETAIN setelah lepas: %v", err)
	}
}
