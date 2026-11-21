"""Fase1 verifikasi: kuitansi QR + /verify publik + portal bayar + pengingat."""
import time
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
PNG = r"D:\project laravel\notaris\web\assets\landing\notarius-hero.png"
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
    # buat invoice via form keuangan
    pg.goto(BASE + "/finance")
    pg.wait_for_load_state("networkidle")
    n = f"INV-E2E-{int(time.time()) % 100000}"
    pg.fill('input[name="number"]', n)
    pg.select_option('select[name="client_id"]', "1")
    pg.fill('input[name="desc[]"] >> nth=0', "Jasa notaris")
    pg.fill('input[name="qty[]"] >> nth=0', "1")
    pg.fill('input[name="price[]"] >> nth=0', "1000000")
    pg.locator('form[action="/invoices"] button[type="submit"]').click()
    pg.wait_for_load_state("networkidle")
    step("invoice dibuat", "Invoice+dibuat" in pg.content() or n in pg.content())
    # buka kuitansi: ambil id dari link
    href = pg.locator('a[href*="/receipt"]').first.get_attribute("href")
    inv_id = href.split("/")[2]
    pg.goto(BASE + href)
    pg.wait_for_load_state("networkidle")
    step("kuitansi tampil", "KUITANSI" in pg.content() and n in pg.content(), href)
    qr = pg.locator('img[src*="/qr/"]').count() > 0
    step("QR tampil", qr)
    code = pg.locator("code").first.inner_text().strip()
    step("kode verifikasi tampil", code.startswith("INV-"), code)
    # verifikasi publik anonim
    ctx2 = b.new_context()
    p2 = ctx2.new_page()
    p2.goto(f"{BASE}/verify?code={code}")
    p2.wait_for_load_state("networkidle")
    step("verifikasi publik TERVERIFIKASI", "TERVERIFIKASI" in p2.content())
    p2.goto(f"{BASE}/verify?code=INV-1-FFFFFFFF")
    step("kode palsu ditolak", "tidak cocok" in p2.content().lower() or "tidak ditemukan" in p2.content().lower())
    ctx2.close()
    print("FASE1-A DONE:", LOG, flush=True)
    b.close()
print("DONE")
