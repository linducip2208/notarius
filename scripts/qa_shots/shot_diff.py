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
    # deed terbaru (dibuat fase2) -> compare
    pg.goto(BASE + "/deeds")
    pg.wait_for_load_state("networkidle")
    link = None
    try:
        link = pg.locator('a[href*="/deeds/"]').first.get_attribute("href")
    except Exception:
        pass
    if link:
        pg.goto(f"{BASE}/deeds/1?compare=3")
        pg.wait_for_load_state("networkidle")
        pg.screenshot(path=f"{OUT}/deed-diff.png")
        print("shot deed-diff", flush=True)
    b.close()
print("DONE")
