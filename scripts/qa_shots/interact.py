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
    pg.on("dialog", lambda d: d.accept())
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**", timeout=15000)
    check("login", True)
    pg.goto(BASE + "/app")
    pg.wait_for_load_state("networkidle")
    # dropdown +Buat
    pg.locator(".dropdown", has_text="+ Buat").locator("button").click()
    pg.wait_for_timeout(500)
    check("dropdown +Buat terbuka", pg.locator(".dropdown-menu.show").count() > 0)
    check("dropdown +Buat item jalan", pg.locator('.dropdown-menu.show a[href="/clients/new"]').count() > 0)
    pg.keyboard.press("Escape")
    pg.wait_for_timeout(300)
    check("dropdown Escape menutup", pg.locator(".dropdown-menu.show").count() == 0)
    # notif dropdown
    pg.locator("a[title=Notifikasi]").click()
    pg.wait_for_timeout(500)
    check("dropdown notif terbuka", pg.locator(".dropdown-menu.show").count() > 0)
    pg.mouse.click(720, 400)
    check("klik-luar menutup dropdown", pg.locator(".dropdown-menu.show").count() == 0)
    # user dropdown
    pg.locator(".avatar-rounded").first.click()
    pg.wait_for_timeout(500)
    check("dropdown user terbuka", pg.locator(".dropdown-menu.show").count() > 0)
    pg.keyboard.press("Escape")
    # tabs
    pg.goto(BASE + "/clients/1")
    pg.wait_for_load_state("networkidle")
    ok = True
    for t in ["#tab-info", "#tab-id", "#tab-doc", "#tab-svc", "#tab-act"]:
        pg.locator(f'a[href="{t}"]').click()
        pg.wait_for_timeout(300)
        pane_ok = pg.locator(t).is_visible()
        link_ok = pg.locator(f'a[href="{t}"].active').count() > 0
        if not (pane_ok and link_ok):
            ok = False
    check("tabs 5 panel + link aktif", ok)
    # mobile sidebar
    m = b.new_context(viewport={"width": 390, "height": 844}).new_page()
    m.goto(BASE + "/login")
    m.fill("input[name=email]", "admin@contoh.id")
    m.fill("input[name=password]", "Admin123!")
    m.click("button[type=submit]")
    m.wait_for_url(BASE + "/app**", timeout=15000)
    m.locator(".navbar-toggler").first.click()
    m.wait_for_timeout(500)
    check("sidebar mobile terbuka", m.locator("#sidebar-menu.show").count() > 0)
    over = m.evaluate("() => document.documentElement.scrollWidth > innerWidth + 1")
    check("tanpa h-scroll 390px", not over)
    # landing navbar collapse (public; CMS beranda memakai .burger sendiri)
    q = b.new_context(viewport={"width": 390, "height": 844}).new_page()
    q.goto(BASE + "/")
    q.wait_for_load_state("networkidle")
    q.locator(".burger").first.click()
    q.wait_for_timeout(500)
    check("nav publik mobile terbuka", q.locator("#mnav.open").count() > 0)
    b.close()

print("INTERACT DONE, gagal:", fails if fails else "nihil")
raise SystemExit(1 if fails else 0)
