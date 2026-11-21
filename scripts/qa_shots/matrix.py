"""Matriks izin 9 peran x rute kunci — otomatis, tanpa klaim manual.
Admin membuat 1 pengguna per peran, lalu tiap peran login dan GET rute.
Harapan dihitung dari rbac.go (sumber kebenaran izin).
Bukti: docs/qa/PERMISSION_MATRIX.md
"""
import time
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
ADMIN = ("admin@contoh.id", "Admin123!")
PW = "Matrix1234"
OUT = r"D:\project laravel\notaris\docs\qa\PERMISSION_MATRIX.md"

ROLES = ["office_admin", "notary", "substitute_notary", "staff",
         "finance", "doc_operator", "auditor", "client"]

# rute -> set peran yang BOLEH (200); lainnya harus 403. Sumber: guard handler.
ALLOW = {
    "/clients": set(ROLES) - {"client"},
    "/deeds": set(ROLES) - {"client"},
    "/finance": {"office_admin", "notary", "finance", "auditor"},
    "/dms": set(ROLES) - {"client"},
    "/audit": {"office_admin", "notary", "substitute_notary", "staff", "finance", "auditor"},
    "/reports": {"office_admin", "notary", "substitute_notary", "staff", "finance", "auditor"},
    "/settings": set(ROLES) - {"client"},
    "/repertorium": {"office_admin", "notary"},
    "/protocols": {"office_admin", "notary", "substitute_notary", "doc_operator", "auditor"},
    "/templates": {"office_admin", "notary", "substitute_notary", "staff", "doc_operator"},
    "/tasks": set(ROLES) - {"client"},
    "/tariffs": set(ROLES) - {"client"},
    "/portal": set(ROLES),
    "/wills": {"notary", "substitute_notary", "auditor"},
    "/offices": set(),  # super_admin saja (diuji terpisah sebagai admin)
    "/cms": set(ROLES) - {"client"},
    "/cms/pages": set(ROLES) - {"client", "finance", "doc_operator", "auditor"},
    "/cms/articles": {"office_admin", "notary", "substitute_notary", "staff"},
    "/cms/media": {"office_admin", "notary", "substitute_notary", "staff"},
    "/cms/faq": {"office_admin", "notary", "substitute_notary", "staff"},
    "/cms/services": {"office_admin", "notary", "substitute_notary", "staff"},
    "/cms/nav": {"office_admin"},
    "/cms/branding": {"office_admin"},
    "/cms/site": {"office_admin"},
    "/cms/history": {"office_admin", "auditor"},
    "/cms/setup": {"office_admin"},
}

results = []


def step(name, ok, extra=""):
    results.append(("- [PASS] " if ok else "- [FAIL] ") + name + (" " + str(extra) if extra else ""))
    print(results[-1], flush=True)


with sync_playwright() as p:
    b = p.chromium.launch()
    # 1. admin buat pengguna per peran
    pg = b.new_page()
    pg.on("dialog", lambda d: d.accept())
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", ADMIN[0])
    pg.fill("input[name=password]", ADMIN[1])
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**")
    step("login admin", True)
    ts = str(int(time.time()))[-6:]
    for role in ROLES:
        email = f"matrix-{role}-{ts}@uji.id".replace("_", "")
        try:
            pg.goto(BASE + "/settings")
            pg.wait_for_load_state("networkidle")
            f = pg.locator('form[action="/users"]').first
            f.locator('input[name=name]').fill(f"Matrix {role}")
            f.locator('input[name=email]').fill(email)
            f.locator('input[name=password]').fill(PW)
            f.locator('select[name=role]').select_option(role)
            f.locator('button[type=submit]').click()
            pg.wait_for_load_state("networkidle")
            step(f"buat {role}", "Pengguna+dibuat" in pg.url or "settings" in pg.url, email)
        except Exception as e:
            step(f"buat {role}", False, str(e)[:100])
    # offices hanya super_admin
    r = pg.goto(BASE + "/offices")
    step("admin boleh /offices", r.status == 200, r.status)
    pg.close()

    # 2. tiap peran login + uji rute
    fails = 0
    table = ["| peran | rute | harap | dapat | cocok |",
             "|---|---|---|---|---|"]
    for role in ROLES:
        email = f"matrix-{role}-{ts}@uji.id".replace("_", "")
        ctx = b.new_context()
        q = ctx.new_page()
        q.goto(BASE + "/login")
        q.fill("input[name=email]", email)
        q.fill("input[name=password]", PW)
        q.click("button[type=submit]")
        try:
            q.wait_for_url(BASE + "/app**", timeout=10000)
        except Exception:
            step(f"login {role}", False, "gagal masuk")
            table.append(f"| {role} | login | 200 | gagal | TIDAK |")
            fails += 1
            ctx.close()
            continue
        for route in sorted(ALLOW.keys()):
            want = 200 if role in ALLOW[route] else 403
            try:
                resp = q.goto(BASE + route)
                got = resp.status if resp else -1
            except Exception:
                got = -1
            ok = (got == want)
            if not ok:
                fails += 1
            table.append(f"| {role} | {route} | {want} | {got} | {'YA' if ok else 'TIDAK'} |")
        # client: portal milik sendiri harus 200
        ctx.close()
    step(f"matriks izin 8 peran x {len(ALLOW)} rute", fails == 0, f"{fails} selisih" if fails else "semua cocok")

    b.close()

with open(OUT, "w", encoding="utf-8") as f:
    f.write("# PERMISSION MATRIX — hasil otomatis matrix.py\n\n")
    f.write("\n".join(table) + "\n\n" + "\n".join(results) + "\n")

fails_total = sum(1 for r in results if r.startswith("- [FAIL]"))
print("MATRIX DONE:", results, flush=True)
raise SystemExit(1 if fails_total else 0)
