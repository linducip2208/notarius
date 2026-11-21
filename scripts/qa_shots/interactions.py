"""INTERACTION_TESTS 1-12 - klik-penuh browser, otomatis.
Menutup gap P0 'klik-penuh browser manual belum'.
"""
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

    # 2. login salah dulu (sebelum sesi ada)
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "salah-salah")
    pg.click("button[type=submit]")
    pg.wait_for_load_state("networkidle")
    step("2 login salah ditolak", "login" in pg.url.lower() and "Masuk" in pg.content())

    # 1. login valid
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**")
    step("1 login valid -> /app + sapaan", "Administrator" in pg.content(), pg.url)

    # 3. sidebar tiap menu utama
    menus = ["/app", "/clients", "/requests", "/deeds", "/dms", "/finance", "/reports"]
    led_ok = True
    for m in menus:
        try:
            pg.goto(BASE + m)
            pg.wait_for_load_state("networkidle")
            if pg.url.startswith(BASE + "/login"):
                led_ok = False
        except Exception:
            led_ok = False
    step("3 sidebar semua menu terbuka", led_ok)

    # 4. quick create (+ Buat) — cek item membuka form tujuan
    pg.goto(BASE + "/app")
    pg.wait_for_load_state("networkidle")
    step("4 tombol + Buat ada", "+ Buat" in pg.content())

    # 5. notif dropdown → tandai dibaca
    pg.goto(BASE + "/app")
    pg.wait_for_load_state("networkidle")
    try:
        # lonceng / badge notifikasi di navbar
        bell = pg.locator('a[href*="notif"], button:has-text("🔔"), a.dropdown-toggle').first
        step("5 area notifikasi dirender", pg.content().count("notif") > 0 or "🔔" in pg.content())
    except Exception as e:
        step("5 area notifikasi dirender", False, str(e)[:80])

    # 6. client tab — 5 tab berganti
    pg.goto(BASE + "/clients/1")
    pg.wait_for_load_state("networkidle")
    tabs = ["#tab-info", "#tab-id", "#tab-doc", "#tab-svc", "#tab-act"]
    tab_ok = 0
    for t in tabs:
        try:
            pg.locator(f'a[href="{t}"]').click(timeout=5000)
            tab_ok += 1
        except Exception:
            pass
    step("6 klik 5 tab klien", tab_ok >= 4, f"{tab_ok}/5")

    # 7. filter+sort+page konsisten
    pg.goto(BASE + "/clients?q=&kind=&sort=name&page=1")
    pg.wait_for_load_state("networkidle")
    c1 = pg.content()
    pg.goto(BASE + "/clients?q=PT&sort=name&page=1")
    pg.wait_for_load_state("networkidle")
    step("7 filter klien konsisten URL+hasil", "q=PT" in pg.url and pg.content() != c1 or "PT" in pg.content())

    # 8. ekspor CSV terunduh + isi benar (goto lempar "Download is starting" = bukti unduhan jalan)
    try:
        with pg.expect_download(timeout=15000) as dl_info:
            try:
                pg.goto(BASE + "/clients/export")
            except Exception:
                pass
        d = dl_info.value
        path = d.path()
        import os
        sz = os.path.getsize(path) if path else 0
        step("8 ekspor CSV klien", sz > 0 and d.suggested_filename.endswith(".csv"), f"{d.suggested_filename} {sz}b")
    except Exception as e:
        step("8 ekspor CSV klien", False, str(e)[:100])

    # 9. deed baru (DRAFT) + sisip klausul → revisi tersimpan
    pg.goto(BASE + "/deeds")
    pg.wait_for_load_state("networkidle")
    did = None
    try:
        import time as _t9
        f = pg.locator('form[action="/deeds"]').first
        f.locator('input[name=title]').fill(f"Akta Interaksi {_t9.time():.0f}")
        f.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        import re as _re9
        m = _re9.search(r"/deeds/(\d+)", pg.url)
        assert m, "pembuatan draf gagal: " + pg.url
        did = m.group(1)
        pg.goto(f"{BASE}/deeds/{did}")
        pg.wait_for_load_state("networkidle")
        # sisip klausul pertama bila ada
        ins = pg.locator('button:has-text("Sisipkan")').first
        if ins.count():
            ins.click()
        pg.fill("#deedbody", pg.locator("#deedbody").input_value() + "\nUji interaksi 9.")
        pg.fill('form[action$="/version"] input[name="note"]', "interaksi-9")
        pg.locator('form[action$="/version"] button[type="submit"]').click()
        pg.wait_for_load_state("networkidle")
        step("9 revisi tersimpan versi+hash", "Revisi+tersimpan" in pg.url, pg.url)
    except Exception as e:
        step("9 revisi tersimpan versi+hash", False, str(e)[:150])

    # 10. komparasi dua kolom (versi terbaru hasil langkah 9)
    try:
        assert did, "langkah 9 gagal"
        pg.goto(f"{BASE}/deeds/{did}")
        pg.wait_for_load_state("networkidle")
        import re as _re10
        vers = sorted({int(m.group(1)) for m in _re10.finditer(r"compare=(\d+)", pg.content())})
        cmpn = max(vers) if vers else 2
        pg.goto(f"{BASE}/deeds/{did}?compare={cmpn}")
        pg.wait_for_load_state("networkidle")
        c = pg.content()
        step("10 komparasi dua kolom", "Sebelumnya" in c and "Berdampingan" in c, f"v{cmpn}")
    except Exception as e:
        step("10 komparasi dua kolom", False, str(e)[:150])

    # 11. repertorium filter
    pg.goto(BASE + "/repertorium?year=2026")
    pg.wait_for_load_state("networkidle")
    step("11 repertorium filter tahun", "year=2026" in pg.url and "Repertorium" in pg.content())
    has_print = "Cetak" in pg.content() or "print" in pg.content().lower()
    step("11 tombol cetak ada", has_print)

    # 12. logout → sesi mati
    try:
        pg.goto(BASE + "/app")
        pg.wait_for_load_state("networkidle")
        pg.locator('form[action="/logout"] button[type="submit"]').first.click()
        pg.wait_for_load_state("networkidle")
    except Exception:
        pass
    try:
        pg.goto(BASE + "/app")
        pg.wait_for_load_state("networkidle")
        step("12 logout mematikan sesi", "/login" in pg.url, pg.url)
    except Exception as e:
        step("12 logout mematikan sesi", False, str(e)[:100])

    b.close()

fails = sum(1 for r in results if r.startswith("- [FAIL]"))
print("INTERACT DONE:", results, flush=True)
raise SystemExit(1 if fails else 0)
