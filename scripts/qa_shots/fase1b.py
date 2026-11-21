"""Fase1B: user klien + portal bayar + pengingat manual."""
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
    step("login admin", True)
    # buat user klien (email unik per run)
    import time as _t
    email = f"klien{_t.time_ns() % 100000}@uji.id"
    pg.goto(BASE + "/settings")
    pg.wait_for_load_state("networkidle")
    pg.fill('form[action="/users"] input[name="name"]', "Klien Uji")
    pg.fill('form[action="/users"] input[name="email"]', email)
    pg.fill('form[action="/users"] input[name="password"]', "Rahasia123")
    pg.select_option('form[action="/users"] select[name="role"]', "client")
    pg.locator('form[action="/users"] button[type="submit"]').click()
    pg.wait_for_load_state("networkidle")
    step("user klien dibuat", "Pengguna+dibuat" in pg.content() or email in pg.content(), email)
    # tautkan user baru ke klien 1 (cari baris tabel yang memuat emailnya)
    import re as _re2
    _row = pg.locator("tr", has_text=email).first
    _href = _row.locator("form[action*=\\/link]").get_attribute("action")
    m = _re2.search(r"/users/(\d+)/link", _href or "")
    step("tombol taut ditemukan", bool(m), _href)
    pg.fill(f'form[action="/users/{m.group(1)}/link"] input[name="client_id"]', "1")
    pg.locator(f'form[action="/users/{m.group(1)}/link"] button').click()
    pg.wait_for_load_state("networkidle")
    step("akun ditautkan ke klien 1", "ditautkan" in pg.content())
    # login sebagai klien
    ctx2 = b.new_context()
    p2 = ctx2.new_page()
    p2.goto(BASE + "/login")
    p2.fill('input[name="email"]', email)
    p2.fill('input[name="password"]', "Rahasia123")
    p2.click('button[type="submit"]')
    p2.wait_for_url(BASE + "/app**")
    step("login klien", "/app" in p2.url or "portal" in p2.content().lower())
    p2.goto(BASE + "/portal/invoices")
    p2.wait_for_load_state("networkidle")
    step("portal tagihan tampil", "Tagihan Saya" in p2.content())
    # konfirmasi bayar dengan bukti
    p2.locator('form[action="/portal/pay"] input[type="file"]').first.set_input_files(PNG)
    p2.locator('form[action="/portal/pay"] button').first.click()
    p2.wait_for_load_state("networkidle")
    step("konfirmasi bayar terkirim", "diterima" in p2.content().lower() or "Konfirmasi" in p2.content())
    ctx2.close()
    # pengingat manual sebagai admin
    pg.goto(BASE + "/tasks")
    pg.wait_for_load_state("networkidle")
    pg.locator('form[action="/reminders/run"] button').click()
    pg.wait_for_load_state("networkidle")
    step("pengingat manual jalan", "Pengingat+dibangkitkan" in pg.content() or "Pengingat dibangkitkan" in pg.content())
    print("FASE1B DONE:", LOG, flush=True)
    b.close()
print("DONE")
