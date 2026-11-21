"""E2E wizard korporat: buat permohonan PT -> wizard -> isi identitas+modal+direktur+BO -> 5/5."""
import time
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
results = []


def step(name, ok, extra=""):
    results.append(("- [PASS] " if ok else "- [FAIL] ") + name + (" " + str(extra) if extra else ""))
    print(results[-1], flush=True)


with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    pg.on("dialog", lambda d: d.accept())
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**")
    step("login", True)
    # permohonan baru (layanan pertama di select)
    pg.goto(BASE + "/requests")
    pg.wait_for_load_state("networkidle")
    try:
        f = pg.locator('form[action="/requests"]').first
        f.locator('input[name="title"]').fill(f"PT Wizard {int(time.time()) % 100000}")
        try:
            f.locator('select[name="client_id"]').select_option(index=1)
        except Exception:
            pass
        try:
            f.locator('select[name="service_id"]').select_option(index=1)
        except Exception:
            pass
        f.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("permohonan dibuat", "flash" in pg.url, pg.url)
    except Exception as e:
        step("permohonan dibuat", False, str(e)[:120])
    # buka permohonan terbaru
    pg.goto(BASE + "/app")
    pg.wait_for_load_state("networkidle")
    href = pg.locator('a[href^="/requests/"]').first.get_attribute("href")
    rid = href.split("/")[-1]
    pg.goto(f"{BASE}/requests/{rid}/wizard")
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    step("wizard tampil + progress", "Wizard" in c and "langkah" in c, rid)
    # 1. identitas + modal PT
    f = pg.locator('form[action$="/corp"]').first
    f.locator('select[name=entity_type]').select_option("PT")
    f.locator('input[name=name]').fill(f"PT Wizard Uji {rid}")
    f.locator('input[name=domicile]').fill("Jakarta")
    f.locator('input[name=authorized]').fill("100000000")
    f.locator('input[name=issued]').fill("50000000")
    f.locator('input[name=paid]').fill("25000000")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    pg.goto(f"{BASE}/requests/{rid}/wizard")
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    import re
    m = re.search(r"(\d+)/5 langkah", c)
    step("identitas+modal tersimpan", m and int(m.group(1)) >= 2, m.group(0) if m else "?")
    # 2. pemegang saham
    f = pg.locator('form[action$="/shareholders"]').first
    f.locator('input[name=name]').fill("Pemegang A")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    # 3. direktur
    pg.goto(f"{BASE}/requests/{rid}/wizard")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action$="/officers"]').first
    f.locator('select[name=role]').select_option("DIREKTUR")
    f.locator('input[name=name]').fill("Direktur A")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    # 4. BO
    pg.goto(f"{BASE}/requests/{rid}/wizard")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action$="/bo"]').first
    f.locator('input[name=name]').fill("BO A")
    f.locator('input[name=pct]').fill("60")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    pg.goto(f"{BASE}/requests/{rid}/wizard")
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    m = re.search(r"(\d+)/5 langkah", c)
    step("wizard 5/5 lengkap", m and m.group(1) == "5", m.group(0) if m else "?")
    b.close()

fails = sum(1 for r in results if r.startswith("- [FAIL]"))
print("WIZARD DONE:", results, flush=True)
raise SystemExit(1 if fails else 0)
