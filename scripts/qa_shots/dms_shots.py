import os
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
OUT = r"D:\project laravel\notaris\docs\screenshots\uiux"

PAGES = [
    ("dms-dashboard", "/dms"),
    ("dms-library", "/dms/library"),
    ("dms-detail", None),  # diisi id pertama dari library
    ("dms-search", "/dms/search?q=akta"),
    ("dms-ocr", "/dms/ocr"),
    ("dms-archive", "/dms/archive"),
    ("dms-retention", "/dms/retention"),
    ("dms-duplicates", "/dms/duplicates"),
    ("dms-categories", "/dms/categories"),
    ("dms-checklists", "/dms/checklists"),
]

with sync_playwright() as p:
    b = p.chromium.launch()
    ctx = b.new_context(viewport={"width": 1440, "height": 900})
    pg = ctx.new_page()
    pg.goto(BASE + "/login")
    pg.fill('input[name="email"]', "admin@contoh.id")
    pg.fill('input[name="password"]', "Admin123!")
    pg.click('button[type="submit"]')
    pg.wait_for_url(BASE + "/app**")
    # ambil id dokumen pertama
    pg.goto(BASE + "/dms/library")
    pg.wait_for_load_state("networkidle")
    first = None
    try:
        href = pg.locator("table tbody tr td a[href^='/documents/']").first.get_attribute("href")
        if href:
            first = href.split("/")[-1]
    except Exception:
        pass
    for name, url in PAGES:
        if url is None:
            if not first:
                print("skip dms-detail (no docs)", flush=True)
                continue
            url = f"/documents/{first}"
        pg.goto(BASE + url)
        pg.wait_for_load_state("networkidle")
        pg.screenshot(path=os.path.join(OUT, name + ".png"))
        print("shot:", name, flush=True)
    b.close()
print("DONE")
