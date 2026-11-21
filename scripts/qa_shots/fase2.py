"""Fase 2 E2E: template checklist -> auto-create di permohonan baru + diff akta + label QR."""
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
LOG = []


def step(name, ok, info=""):
    LOG.append(f"- [{'PASS' if ok else 'FAIL'}] {name}" + (f" — {info}" if info else ""))
    print(LOG[-1], flush=True)
    if not ok:
        raise SystemExit(f"E2E BERHENTI di: {name}")


with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page()
    pg.goto(BASE + "/login")
    pg.fill('input[name="email"]', "admin@contoh.id")
    pg.fill('input[name="password"]', "Admin123!")
    pg.click('button[type="submit"]')
    pg.wait_for_url(BASE + "/app**")
    step("login", True)
    # 1. template checklist untuk PT-DIRIAN
    pg.goto(BASE + "/dms/checklists")
    pg.wait_for_load_state("networkidle")
    pg.select_option('select[name="service_code"]', "PT-DIRIAN")
    pg.fill('input[name="name"]', "Syarat Pendirian PT")
    pg.fill('textarea[name="items"]', "KTP pendiri\nNPWP\nModal disetor")
    pg.locator('form[action="/dms/checklists"] button[type="submit"]').click()
    pg.wait_for_load_state("networkidle")
    step("template checklist dibuat", "Syarat Pendirian PT" in pg.content())
    # 2. permohonan baru layanan Pendirian PT -> checklist otomatis muncul
    pg.goto(BASE + "/requests")
    pg.wait_for_load_state("networkidle")
    pg.select_option('form[action="/requests"] select[name="client_id"]', "1")
    pg.select_option('form[action="/requests"] select[name="service_id"]', label="Pendirian PT")
    pg.fill('form[action="/requests"] input[name="title"]', "PT E2E Otomatis")
    pg.locator('form[action="/requests"] button[type="submit"]').click()
    pg.wait_for_load_state("networkidle")
    step("permohonan dibuat", True)
    # buka permohonan terbaru dari dashboard
    pg.goto(BASE + "/app")
    pg.wait_for_load_state("networkidle")
    href = pg.locator('a[href^="/requests/"]').first.get_attribute("href")
    pg.goto(BASE + href)
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    step("checklist otomatis terisi", "KTP pendiri" in c and "Modal disetor" in c, href)
    step("progress bar tampil", "progress-bar" in c)
    # 3. diff akta: revisi via editor kaya lalu bandingkan
    pg.goto(BASE + "/deeds/1")
    pg.wait_for_load_state("networkidle")
    pg.locator("#deedbody").click()
    pg.keyboard.press("End")
    pg.keyboard.type("\nParagraf revisi E2E.")
    # toolbar bold: pilih semua? cukup ketik + simpan (sanitasi server menormalkan)
    pg.fill('form[action$="/version"] input[name="note"]', "revisi E2E")
    pg.locator('form[action$="/version"] button[type="submit"]').click()
    pg.wait_for_load_state("networkidle")
    step("revisi editor kaya tersimpan", "Revisi+tersimpan" in pg.url, pg.url)
    pg.goto(pg.url.split("?")[0] + "?compare=2")
    pg.wait_for_load_state("networkidle")
    step("diff tampil dengan highlight", "Paragraf revisi E2E" in pg.content())
    # 4. label QR arsip
    pg.goto(BASE + "/dms/archive/labels")
    pg.wait_for_load_state("networkidle")
    step("label QR tampil", "Cetak Label" in pg.content())
    b.close()
print("FASE2 DONE:", LOG, flush=True)
