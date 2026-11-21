from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
OUT = r"D:\project laravel\notaris\docs\screenshots\uiux"

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**")
    pg.goto(BASE + "/finance")
    pg.wait_for_load_state("networkidle")
    href = pg.locator('a[href*="/receipt"]').first.get_attribute("href")
    print("receipt:", href, flush=True)
    pg.goto(BASE + href)
    pg.wait_for_load_state("networkidle")
    pg.screenshot(path=OUT + "/receipt-desktop.png")
    print("shot receipt", flush=True)
    b.close()
print("DONE")
