from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
results = []

def step(name, ok, extra=""):
    results.append(("- [PASS] " if ok else "- [FAIL] ") + name + (" " + str(extra) if extra else ""))
    print(results[-1], flush=True)

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    # tombol Sahkan/Tandatangani memakai confirm() JS — terima otomatis agar submit jalan
    pg.on("dialog", lambda d: d.accept())
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    try:
        pg.wait_for_url(BASE + "/app**", timeout=10000)
        step("login", True)
    except Exception as e:
        step("login", False, e)
        print("FASE3 DONE:", results, flush=True)
        b.close()
        raise SystemExit(1)

    # 1. offices page (super_admin = admin@contoh.id?)
    pg.goto(BASE + "/offices")
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    step("halaman multi-kantor tampil", "Multi-Kantor" in c or "Kantor & Paket" in c)

    # 2. set plan ENTERPRISE utk kantor 1 agar acting diizinkan
    try:
        frm = pg.locator('form[action="/offices/1/plan"]').first
        frm.locator('select[name=plan]').select_option("ENTERPRISE")
        frm.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("set paket ENTERPRISE", "ENTERPRISE" in pg.content())
    except Exception as e:
        step("set paket ENTERPRISE", False, e)

    # 3. acting ke kantor 1
    try:
        frm = pg.locator('form[action="/offices/act"] input[value="1"]').first
        pg.locator('form[action="/offices/act"]').first.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("acting lintas-kantor", "bertindak" in pg.content().lower())
    except Exception as e:
        step("acting lintas-kantor", False, e)

    # 4. TTE: buat akta -> review -> approve -> sign -> verify
    pg.goto(BASE + "/deeds")
    pg.wait_for_load_state("networkidle")
    try:
        # form draf baru: cari form post /deeds
        f = pg.locator('form[action="/deeds"]').first
        f.locator('input[name=title]').fill("Akta TTE Uji")
        try:
            f.locator('select[name=kind]').select_option("PT")
        except Exception:
            pass
        f.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        did = pg.url.strip("/").split("/")[-1].split("?")[0]
        step("akta TTE dibuat", did.isdigit(), pg.url)
    except Exception as e:
        step("akta TTE dibuat", False, e)
        did = "1"

    pg.goto(f"{BASE}/deeds/{did}")
    pg.wait_for_load_state("networkidle")
    try:
        pg.locator('form[action$="/review"] button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("kirim review", True)
    except Exception as e:
        step("kirim review", False, e)
    pg.goto(f"{BASE}/deeds/{did}")
    pg.wait_for_load_state("networkidle")
    try:
        import time as _t
        uniq = f"TTE-{int(_t.time()) % 100000}"
        nform = pg.locator('form[action$="/approve"]').first
        nform.locator('input[name=number]').fill(uniq)
        nform.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        c = pg.content()
        step("approve + segel", "APPROVED" in c and "Segel verifikasi" in c, pg.url)
    except Exception as e:
        step("approve + segel", False, e)
    pg.goto(f"{BASE}/deeds/{did}")
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    step("segel tampil di detail", "Segel verifikasi" in c and "DEED-" in c)
    import re as _re0
    _m0 = _re0.search(r"DEED-\d+-[0-9A-F]{8}", c)
    if _m0:
        pg.goto(f"{BASE}/verify?code={_m0.group(0)}")
        pg.wait_for_load_state("networkidle")
        vc = pg.content()
        step("verify segel akta", "TERVERIFIKASI" in vc and "Akta" in vc, _m0.group(0))
    else:
        step("verify segel akta", False, "segel-tak-ditemukan")
    pg.goto(f"{BASE}/deeds/{did}")
    pg.wait_for_load_state("networkidle")
    try:
        sform = pg.locator('form[action$="/sign"]').first
        sform.locator('input[name=note]').fill("uji TTE")
        sform.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        c = pg.content()
        import re as _re
        has_hash = bool(_re.search(r"[0-9a-f]{64}", c))
        step("tanda tangan internal", "Tanda+tangan+internal+tercatat" in pg.url and has_hash, pg.url)
    except Exception as e:
        step("tanda tangan internal", False, e)
    pg.goto(f"{BASE}/deeds/{did}")
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    import re as _re2
    m = _re2.search(r"[0-9a-f]{64}", c)
    step("hash TTE tampil", "TTE internal" in c and m is not None, m.group(0)[:16] if m else "tanpa-hash")

    # 5. plan gate: turunkan ke LITE lalu buktikan blokir 402, kembali ke PRO
    pg.goto(BASE + "/offices")
    pg.wait_for_load_state("networkidle")
    try:
        frm = pg.locator('form[action="/offices/1/plan"]').first
        frm.locator('select[name=plan]').select_option("LITE")
        frm.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("turun ke LITE", True)
        pg.goto(BASE + "/dms/checklists")
        pg.wait_for_load_state("networkidle")
        step("gate LITE blokir checklist (402)", "membutuhkan paket PRO" in pg.content(), pg.url)
    except Exception as e:
        step("turun ke LITE", False, e)
    # kembalikan ke PRO di akhir
    try:
        pg.goto(BASE + "/offices")
        pg.wait_for_load_state("networkidle")
        frm = pg.locator('form[action="/offices/1/plan"]').first
        frm.locator('select[name=plan]').select_option("PRO")
        frm.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("kembali ke PRO", True)
    except Exception as e:
        step("kembali ke PRO", False, e)

    b.close()
print("FASE3 DONE:", results, flush=True)
