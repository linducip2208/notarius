"""Screenshot visual QA: login + tangkap halaman kunci (desktop 1440 + mobile 390).
Pakai: D:\\python\\python.exe scripts/qa_shots/shots.py  (server harus jalan di :8080)
Output: docs/screenshots/uiux/*.png
"""
import os
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..",
                   "docs", "screenshots", "uiux")

PAGES = [
    ("landing", "/", False),
    ("dashboard", "/app", True),
    ("clients", "/clients", True),
    ("requests", "/requests", True),
    ("deeds", "/deeds", True),
    ("repertorium", "/repertorium", True),
    ("documents", "/documents", True),
    ("finance", "/finance", True),
    ("roles", "/roles", True),
]


def main():
    os.makedirs(OUT, exist_ok=True)
    with sync_playwright() as p:
        b = p.chromium.launch()
        # desktop: login sekali, pakai ulang konteks
        ctx = b.new_context(viewport={"width": 1440, "height": 900})
        pg = ctx.new_page()
        pg.goto(BASE + "/login")
        pg.fill('input[name="email"]', "admin@contoh.id")
        pg.fill('input[name="password"]', "Admin123!")
        pg.click('button[type="submit"]')
        pg.wait_for_load_state("networkidle")
        for name, url, _ in PAGES:
            if url == "/":
                c2 = b.new_context(viewport={"width": 1440, "height": 900})
                p2 = c2.new_page()
                p2.goto(BASE + url)
                p2.wait_for_load_state("networkidle")
                p2.screenshot(path=os.path.join(OUT, f"{name}-desktop.png"))
                c2.close()
                continue
            pg.goto(BASE + url)
            pg.wait_for_load_state("networkidle")
            pg.screenshot(path=os.path.join(OUT, f"{name}-desktop.png"))
            print("shot:", name, flush=True)
        ctx.close()
        # mobile: landing + dashboard
        m = b.new_context(viewport={"width": 390, "height": 844},
                          user_agent="Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile",
                          is_mobile=True, has_touch=True)
        mp = m.new_page()
        mp.goto(BASE + "/")
        mp.wait_for_load_state("networkidle")
        mp.screenshot(path=os.path.join(OUT, "landing-mobile.png"))
        mp.goto(BASE + "/login")
        mp.fill('input[name="email"]', "admin@contoh.id")
        mp.fill('input[name="password"]', "Admin123!")
        mp.click('button[type="submit"]')
        mp.wait_for_load_state("networkidle")
        mp.screenshot(path=os.path.join(OUT, "dashboard-mobile.png"))
        print("shot: mobile", flush=True)
        m.close()
        b.close()
    print("DONE ->", OUT)


if __name__ == "__main__":
    main()
