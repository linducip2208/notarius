package integration

import (
	"fmt"
	"testing"
	"time"

	"notaryos/internal/cms"
)

// TestCMSPerf seeds a realistic dataset (100 pages, 500 articles, 1000 media rows)
// in the TEST database only and times public reads. Honest thresholds for local MySQL.
func TestCMSPerf(t *testing.T) {
	clean(t)
	office := seedOffice(t)
	cs := cms.NewStore(db)
	t0 := time.Now()
	for i := 0; i < 100; i++ {
		if _, err := cs.SavePage(office, &cms.Page{Title: fmt.Sprintf("Halaman %d", i),
			Slug: fmt.Sprintf("halaman-%d", i), Content: "<p>Isi uji beban.</p>",
			Status: cms.StatusPublished}, nil); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 500; i++ {
		if _, err := cs.SaveArticle(office, &cms.Article{Title: fmt.Sprintf("Artikel %d", i),
			Slug: fmt.Sprintf("artikel-%d", i), Content: "<p>Isi.</p>",
			Status: cms.StatusPublished}, nil); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 1000; i++ {
		if _, err := cs.SaveMedia(office, &cms.Media{Ref: fmt.Sprintf("cms/perf/%d.bin", i),
			OriginalName: "b.bin", MIME: "application/pdf", Size: 10}, nil); err != nil {
			t.Fatal(err)
		}
	}
	seedDur := time.Since(t0)
	t.Logf("seed 1600 rows: %v", seedDur)
	t1 := time.Now()
	if _, _, err := cs.PublicPage(office, "halaman-50", "id"); err != nil {
		t.Fatal(err)
	}
	pageDur := time.Since(t1)
	t2 := time.Now()
	list, total, err := cs.PublicArticles(office, "id", "Artikel", 9, 0)
	if err != nil || total != 500 || len(list) != 9 {
		t.Fatalf("listing salah: %v %d %d", err, total, len(list))
	}
	listDur := time.Since(t2)
	t.Logf("public page: %v | articles search page1: %v", pageDur, listDur)
	if pageDur > 500*time.Millisecond || listDur > 500*time.Millisecond {
		t.Fatalf("terlalu lambat: page=%v list=%v", pageDur, listDur)
	}
}
