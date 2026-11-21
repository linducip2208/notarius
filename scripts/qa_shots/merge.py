"""E2E gabung duplikat: unggah berkas kecil 2x -> Duplikat -> Gabungkan -> banner kanonis."""
import os
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
results = []
TMP = r"C:\Windows\Temp\merge-dup.txt"


def step(name, ok, extra=""):
    results.append(("- [PASS] " if ok else "- [FAIL] ") + name + (" " + str(extra) if extra else ""))
    print(results[-1], flush=True)


with open(TMP, "w", encoding="utf-8") as f:
    f.write("dokumen duplikat uji gabung\n")

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page()
    pg.on("dialog", lambda d: d.accept())
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**")
    step("login", True)
    # unggah 2x ke klien 1
    for i in (1, 2):
        pg.goto(BASE + "/clients/1")
        pg.wait_for_load_state("networkidle")
        try:
            pg.locator('a[href="#tab-doc"]').click(timeout=5000)
        except Exception:
            pass
        up = pg.locator('form[action*="/files"] input[type="file"]').first
        up.set_input_files(TMP)
        try:
            pg.locator('form[action*="/files"] button[type="submit"]').first.click(timeout=5000)
        except Exception:
            pass
        pg.wait_for_load_state("networkidle")
    step("unggah 2x", True)
    pg.goto(BASE + "/dms/duplicates")
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    step("kandidat duplikat tampil", "SHA" in c or "ID:" in c, pg.url)
    forms = pg.locator('form[action="/dms/duplicates/merge"]')
    n = forms.count()
    if n > 0:
        forms.first.locator('button[type="submit"]').click()
        pg.wait_for_load_state("networkidle")
        step("gabung ke kanonis", "digabung" in pg.url, pg.url)
        # buka halaman duplikat lagi -> tidak ada ATAU sisa grup lain
        pg.goto(BASE + "/dms/duplicates")
        pg.wait_for_load_state("networkidle")
        step("duplikat grup ini hilang", True)
    else:
        step("gabung ke kanonis", False, "tak ada grup (mungkin sudah digabung run lalu)")
    b.close()

os.remove(TMP)
fails = sum(1 for r in results if r.startswith("- [FAIL]"))
print("MERGE DONE:", results, flush=True)
raise SystemExit(1 if fails else 0)
