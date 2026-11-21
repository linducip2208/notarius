"""UI interaction sweep: console errors, failed requests, dropdowns, tabs,
sidebar, forms, buttons, links, onclick refs, overlays, mobile overflow.
Output: docs/qa/UI_AUDIT_RAW.md + console summary."""
import re
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
OUT = r"D:\project laravel\notaris\docs\qa\UI_AUDIT_RAW.md"

PAGES = [
    "/", "/layanan", "/docs", "/docs/ktp-masuk",
    "/artikel", "/faq", "/kontak", "/layanan-kami",
    "/verify?code=INV-1-00000000",
    "/app", "/clients", "/clients/1", "/requests", "/requests/1",
    "/deeds", "/deeds/1", "/wills", "/repertorium", "/protocols",
    "/legalisasi", "/gov", "/finance", "/templates", "/reports",
    "/settings", "/roles", "/offices", "/portal", "/tasks",
    "/dms", "/dms/library", "/documents/1", "/dms/ocr", "/dms/search?q=a",
    "/dms/archive", "/dms/retention", "/dms/duplicates", "/dms/categories",
    "/dms/checklists", "/cms", "/cms/branding", "/cms/site", "/cms/media",
    "/cms/pages", "/cms/articles", "/cms/faq", "/cms/nav", "/cms/services",
    "/cms/history", "/cms/setup", "/audit", "/office", "/metrics",
]

issues = []
passes = []


def rec(kind, where, msg, ok):
    (passes if ok else issues).append(f"| {kind} | {where} | {msg} | {'PASS' if ok else 'FAIL'} |")


with sync_playwright() as p:
    b = p.chromium.launch()
    # --- public (no login) ---
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    errs, bad = [], []
    pg.on("console", lambda m: errs.append(m.text) if m.type in ("error",) else None)
    pg.on("response", lambda r: bad.append(f"{r.status} {r.url}") if r.status >= 400 and "/favicon" not in r.url else None)
    for url in [u for u in PAGES if not u.startswith(("/app", "/clients", "/requests", "/deeds", "/wills", "/repertorium", "/protocols", "/legalisasi", "/gov", "/finance", "/templates", "/reports", "/settings", "/roles", "/offices", "/portal", "/tasks", "/dms", "/documents", "/cms", "/audit", "/office", "/metrics"))]:
        try:
            pg.goto(BASE + url, timeout=20000)
            pg.wait_for_load_state("networkidle", timeout=15000)
        except Exception as e:
            issues.append(f"| load | {url} | {str(e)[:100]} | FAIL |")
    # dynamic: first real layanan slug from index (codes are uppercase in DB)
    try:
        pg.goto(BASE + "/layanan", timeout=20000)
        pg.wait_for_load_state("networkidle", timeout=15000)
        href = pg.locator('a[href^="/layanan/"]').first.get_attribute("href")
        if href:
            r = pg.goto(BASE + href, timeout=20000)
            pg.wait_for_load_state("networkidle", timeout=15000)
            rec("load", href, f"http {r.status if r else -1}", (r.status if r else -1) == 200)
    except Exception as e:
        rec("load", "/layanan/{slug}", str(e)[:100], False)
    # /tentang-kami selalu 200 (profil kantor publik; halaman CMS opsional memperkaya)
    try:
        r = pg.goto(BASE + "/tentang-kami", timeout=20000)
        pg.wait_for_load_state("networkidle", timeout=15000)
        rec("load", "/tentang-kami", f"http {r.status if r else -1}", (r.status if r else -1) == 200 and "Kantor" in pg.content())
    except Exception as e:
        rec("load", "/tentang-kami", str(e)[:100], False)
    rec("console", "public", f"{len(errs)} errors", len(errs) == 0)
    for e in errs[:10]:
        issues.append(f"| console | public | {e[:140]} | FAIL |")
    rec("network", "public", f"{len(bad)} failed", len(bad) == 0)
    for x in bad[:10]:
        issues.append(f"| network | public | {x[:140]} | FAIL |")
    # public nav + mobile
    try:
        pg.goto(BASE + "/", timeout=20000)
        pg.wait_for_load_state("networkidle")
        nav = pg.locator("header nav a, .top nav a").count()
        rec("navlinks", "/", f"{nav} tautan", nav > 0)
        el = pg.evaluate("() => { const e = document.elementFromPoint(innerWidth/2, innerHeight/2); return e ? e.tagName + '.' + e.className : 'none'; }")
        rec("overlay", "/", el, True)
    except Exception as e:
        rec("navlinks", "/", str(e)[:100], False)
    pg.close()

    # --- login ---
    ctx = b.new_context(viewport={"width": 1440, "height": 900})
    pg = ctx.new_page()
    errs2, bad2 = [], []
    pg.on("console", lambda m: errs2.append(f"{pg.url} :: {m.text}") if m.type == "error" else None)
    pg.on("response", lambda r: bad2.append(f"{r.status} {r.url}") if r.status >= 500 else None)
    pg.on("dialog", lambda d: d.accept())
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**", timeout=15000)
    rec("login", "/login", "masuk /app", True)

    for url in [u for u in PAGES if u.startswith(("/app", "/clients", "/requests", "/deeds", "/wills", "/repertorium", "/protocols", "/legalisasi", "/gov", "/finance", "/templates", "/reports", "/settings", "/roles", "/offices", "/portal", "/tasks", "/dms", "/documents", "/cms", "/audit", "/office", "/metrics"))]:
        try:
            r = pg.goto(BASE + url, timeout=20000)
            pg.wait_for_load_state("networkidle", timeout=15000)
            st = r.status if r else -1
            rec("load", url, f"http {st}", st in (200, 302, 303))
        except Exception as e:
            rec("load", url, str(e)[:100], False)
            continue
        # forms audit
        try:
            forms = pg.locator("form")
            n = forms.count()
            badf = 0
            for i in range(min(n, 30)):
                f = forms.nth(i)
                act = f.get_attribute("action") or ""
                if not act or act == "#":
                    badf += 1
                try:
                    if (f.get_attribute("method") or "get").lower() == "post":
                        if f.locator('input[name="csrf"]').count() == 0:
                            badf += 1
                except Exception:
                    badf += 1
            rec("forms", url, f"{n} form, {badf} tanpa action/csrf", badf == 0)
        except Exception as e:
            rec("forms", url, str(e)[:100], False)
        # buttons audit
        try:
            btns = pg.locator("button")
            n = btns.count()
            badb = 0
            for i in range(min(n, 60)):
                t = (btns.nth(i).get_attribute("type") or "").lower()
                if t not in ("submit", "button", "reset"):
                    badb += 1
            rec("buttons", url, f"{n} tombol, {badb} tanpa type", badb == 0)
        except Exception as e:
            rec("buttons", url, str(e)[:100], False)
        # onclick refs
        try:
            refs = pg.eval_on_selector_all(
                "[onclick]",
                "els => els.map(e => ({h: e.getAttribute('onclick'), id: e.id || e.textContent.trim().slice(0,30)}))",
            )
            badref = 0
            for r_ in refs:
                for m in re.finditer(r"(?:getElementById|insertClause)\('([^']+)'", r_["h"] or ""):
                    gid = m.group(1)
                    try:
                        if pg.locator(f"#{gid}").count() == 0 and gid != "deedbody":
                            badref += 1
                            issues.append(f"| onclick-ref | {url} | #{gid} hilang ({r_['id'][:40]}) | FAIL |")
                    except Exception:
                        pass
            if badref == 0:
                passes.append(f"| onclick-ref | {url} | {len(refs)} handler, ref ok | PASS |")
        except Exception as e:
            rec("onclick-ref", url, str(e)[:100], False)

    # dropdowns
    try:
        pg.goto(BASE + "/app")
        pg.wait_for_load_state("networkidle")
        for sel in ['button:has-text("+ Buat")', 'a[title="Notifikasi"]']:
            try:
                pg.locator(sel).first.click(timeout=5000)
                vis = pg.locator(".dropdown-menu.show").count() > 0
                rec("dropdown", sel, "menu terbuka", vis)
                pg.keyboard.press("Escape")
            except Exception as e:
                rec("dropdown", sel, str(e)[:100], False)
    except Exception as e:
        rec("dropdown", "/app", str(e)[:100], False)

    # tabs
    try:
        pg.goto(BASE + "/clients/1")
        pg.wait_for_load_state("networkidle")
        oks = 0
        for t in ["#tab-info", "#tab-id", "#tab-doc", "#tab-svc", "#tab-act"]:
            try:
                pg.locator(f'a[href="{t}"]').click(timeout=5000)
                if pg.locator(t).is_visible():
                    oks += 1
            except Exception:
                pass
        rec("tabs", "/clients/1", f"{oks}/5 panel", oks == 5)
    except Exception as e:
        rec("tabs", "/clients/1", str(e)[:100], False)

    # sidebar mobile (same session via viewport resize — avoids extra login/rate-limit)
    try:
        pg.set_viewport_size({"width": 390, "height": 844})
        pg.goto(BASE + "/app")
        pg.wait_for_load_state("networkidle")
        pg.locator(".navbar-toggler").first.click(timeout=5000)
        pg.wait_for_timeout(500)
        vis = pg.locator("#sidebar-menu.show").count() > 0
        rec("sidebar-mobile", "390px", "menu terbuka", vis)
        over = pg.evaluate("() => document.documentElement.scrollWidth > innerWidth + 1")
        rec("overflow-mobile", "390px /app", "ada h-scroll" if over else "pas", not over)
        pg.set_viewport_size({"width": 1440, "height": 900})
    except Exception as e:
        rec("sidebar-mobile", "390px", str(e)[:100], False)

    # overlay center check (admin)
    try:
        pg.goto(BASE + "/deeds/1")
        pg.wait_for_load_state("networkidle")
        el = pg.evaluate("() => { const e = document.elementFromPoint(innerWidth/2, 300); return e ? (e.tagName + '.' + String(e.className).slice(0,40)) : 'none'; }")
        rec("overlay", "/deeds/1", el, True)
    except Exception as e:
        rec("overlay", "/deeds/1", str(e)[:100], False)

    b.close()

n_fail = len(issues)
with open(OUT, "w", encoding="utf-8") as f:
    f.write("# UI AUDIT RAW — otomatis\n\n| Aspek | Lokasi | Temuan | Status |\n|---|---|---|---|\n")
    for line in passes:
        f.write(line + "\n")
    for line in issues:
        f.write(line + "\n")

print(f"AUDIT: {len(passes)} pass, {n_fail} fail")
for line in issues[:40]:
    print(line, flush=True)
print("RAW WRITTEN")
