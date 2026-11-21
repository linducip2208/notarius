from playwright.sync_api import sync_playwright
import time

BASE = "http://127.0.0.1:8080"
with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page()
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**")
    ts = str(int(time.time()))[-6:]
    em = f"portal-akta-{ts}@uji.id"
    pg.goto(BASE + "/settings")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/users"]').first
    f.locator("input[name=name]").fill("Portal Akta")
    f.locator("input[name=email]").fill(em)
    f.locator("input[name=password]").fill("Portal1234")
    f.locator("select[name=role]").select_option("client")
    f.locator("button[type=submit]").click()
    pg.wait_for_load_state("networkidle")
    print("USER OK", flush=True)
    row = pg.locator(f'tr:has-text("{em}")').first
    row.locator("input[name=client_id]").fill("1")
    row.locator("button[type=submit]").click()
    pg.wait_for_load_state("networkidle")
    print("LINK OK", flush=True)
    pg.close()
    ctx = b.new_context()
    q = ctx.new_page()
    q.goto(BASE + "/login")
    q.fill("input[name=email]", em)
    q.fill("input[name=password]", "Portal1234")
    q.click("button[type=submit]")
    q.wait_for_url(BASE + "/app**", timeout=15000)
    q.goto(BASE + "/portal")
    q.wait_for_load_state("networkidle")
    c = q.content()
    print("PORTAL AKTA SAYA:", "Akta Saya" in c, "| PDF link:", "/pdf" in c, flush=True)
    b.close()
print("DONE")
