from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
fails = []


def check(name, ok, extra=""):
    print(("- [PASS] " if ok else "- [FAIL] ") + name + (" " + extra if extra else ""), flush=True)
    if not ok:
        fails.append(name)


with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**", timeout=15000)
    # dismiss: dialog dibatalkan -> tetap di halaman, tanpa navigasi/aksi
    pg.goto(BASE + "/cms/faq")
    pg.wait_for_load_state("networkidle")
    n_before = pg.locator('table tbody tr').count()
    pg.once("dialog", lambda d: d.dismiss())
    try:
        pg.locator('form[action$="/delete"] button[type=submit]').first.click(timeout=8000)
    except Exception:
        pass
    pg.wait_for_timeout(1500)
    check("confirm dismiss: tidak ada aksi", "/cms/faq" in pg.url and pg.locator('table tbody tr').count() == n_before, pg.url)
    # accept on a harmless target: create then delete a FAQ
    pg.locator('form[action="/cms/faq"] input[name=question]').fill("Hapus Saya Uji")
    pg.locator('form[action="/cms/faq"] textarea[name=answer]').fill("jawab")
    pg.locator('form[action="/cms/faq"] button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    row = pg.locator('tr:has-text("Hapus Saya Uji")').first
    check("faq uji terbuat", row.count() > 0)
    pg.once("dialog", lambda d: d.accept())
    row.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    check("confirm accept: terhapus", pg.locator('tr:has-text("Hapus Saya Uji")').count() == 0, pg.url)
    b.close()
print("CONFIRM DONE, gagal:", fails if fails else "nihil")
raise SystemExit(1 if fails else 0)
