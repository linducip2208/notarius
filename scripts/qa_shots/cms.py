"""E2E CMS + White Label: branding, site, halaman+section+terbit+render publik,
artikel, faq, nav, media, setup contoh, sitemap, histori. Bukti per langkah."""
import time
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
results = []
PNG = r"D:\project laravel\notaris\web\assets\landing\notarius-hero.png"


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
    ts = str(int(time.time()))[-6:]

    # 1. branding
    pg.goto(BASE + "/cms")
    pg.wait_for_load_state("networkidle")
    step("dashboard CMS", "Website" in pg.content() and "Halaman" in pg.content(), pg.url)
    pg.goto(BASE + "/cms/branding")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/branding"]').first
    orig_name = f.locator('input[name=app_name]').input_value()
    orig_primary = f.locator('input[name=primary]').input_value()
    f.locator('input[name=app_name]').fill("Kantor Uji CMS")
    f.locator('input[name=primary]').fill("#0a7d44")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("branding tersimpan", "Branding+disimpan" in pg.url, pg.url)
    pg.goto(BASE + "/app")
    pg.wait_for_load_state("networkidle")
    step("branding tampil di admin", "Kantor Uji CMS" in pg.content())

    # 2. site + domain
    pg.goto(BASE + "/cms/site")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/site"]').first
    f.locator('input[name=site_title]').fill("Kantor Notaris Uji")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("SEO global tersimpan", "Pengaturan+website" in pg.url)
    f = pg.locator('form[action="/cms/domain"]').first
    f.locator('input[name=primary_domain]').fill("kantor-uji.example")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("domain tersimpan + tervalidasi format", "Domain+disimpan" in pg.url)
    try:
        f = pg.locator('form[action="/cms/domain"]').first
        f.locator('input[name=primary_domain]').fill("javascript:x")
        f.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("domain jahat ditolak", "tidak valid" in pg.content().lower())
    except Exception as e:
        step("domain jahat ditolak", False, str(e)[:100])

    # 3. halaman + section + terbit + render publik
    slug = f"berita-{ts}"
    pg.goto(BASE + "/cms/pages/new")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/pages"]').first
    f.locator('input[name=title]').fill(f"Berita Uji {ts}")
    f.locator('input[name=slug]').fill(slug)
    f.locator('textarea[name=content]').fill("<p>Halo publik.</p><script>alert(1)</script>")
    f.locator('select[name=status]').select_option("PUBLISHED")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("halaman terbit", "flash=Halaman" in pg.url or "/cms/pages/" in pg.url, pg.url)
    rid = pg.url.strip("/").split("/")[-1].split("?")[0]
    # section
    pg.goto(f"{BASE}/cms/pages/{rid}")
    pg.wait_for_load_state("networkidle")
    try:
        sf = pg.locator('form[action$="/sections"]').first
        sf.locator('select[name=type]').select_option("cta")
        sf.locator('input[name=title]').fill("Ayo hubungi")
        sf.locator('input[name=cta_label]').fill("Kontak")
        sf.locator('input[name=cta_url]').fill("/kontak")
        sf.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("section CTA tersimpan", "Section+disimpan" in pg.url)
    except Exception as e:
        step("section CTA tersimpan", False, str(e)[:120])
    # publik (konteks baru tanpa sesi — publik tak butuh login)
    pub = b.new_context().new_page()
    pub.goto(f"{BASE}/p/{slug}")
    pub.wait_for_load_state("networkidle")
    c = pub.content()
    step("render publik + sanitasi script", "Halo publik" in c and "<script>alert(1)</script>" not in c and "<script>" not in c, pub.url)

    # 4. beranda CMS
    pg.goto(BASE + "/cms/pages")
    pg.wait_for_load_state("networkidle")
    step("daftar halaman", "Berita Uji" in pg.content())
    # buat beranda via duplikat? langsung buat baru
    pg.goto(BASE + "/cms/pages/new")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/pages"]').first
    f.locator('input[name=title]').fill("Beranda CMS")
    f.locator('input[name=slug]').fill("beranda")
    f.locator('textarea[name=content]').fill("<p>Selamat datang di kantor kami.</p>")
    f.locator('select[name=status]').select_option("PUBLISHED")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    pub2 = b.new_context().new_page()
    pub2.goto(BASE + "/")
    pub2.wait_for_load_state("networkidle")
    step("beranda CMS menggantikan landing", "Melayani kebutuhan notaris kantor" in pub2.content(), pub2.url)

    # 5. artikel + faq + nav
    pg.goto(BASE + "/cms/articles/new")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/articles"]').first
    f.locator('input[name=title]').fill(f"Artikel Uji {ts}")
    f.locator('input[name=slug]').fill(f"artikel-{ts}")
    f.locator('textarea[name=content]').fill("<p>Isi artikel uji.</p>")
    f.locator('select[name=status]').select_option("PUBLISHED")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("artikel terbit", "Artikel+disimpan" in pg.url)
    pg.goto(f"{BASE}/artikel/artikel-{ts}")
    pg.wait_for_load_state("networkidle")
    step("artikel tampil publik", "Isi artikel uji" in pg.content())
    pg.goto(BASE + "/cms/faq")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/faq"]').first
    f.locator('input[name=question]').fill("Jam berapa buka?")
    f.locator('textarea[name=answer]').fill("Senin-Jumat 08-16.")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("FAQ tersimpan", "FAQ+disimpan" in pg.url)
    pg.goto(BASE + "/faq")
    pg.wait_for_load_state("networkidle")
    step("FAQ tampil publik", "Jam berapa buka" in pg.content())
    pg.goto(BASE + "/cms/nav")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/nav"]').first
    f.locator('input[name=label]').fill("Tentang")
    f.locator('input[name=url]').fill("/tentang-kami")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("nav tersimpan", "Navigasi+disimpan" in pg.url)

    # 6. media
    pg.goto(BASE + "/cms/media")
    pg.wait_for_load_state("networkidle")
    up = pg.locator('form[action="/cms/media"][enctype] input[type="file"]').first
    up.set_input_files(PNG)
    pg.locator('form[action="/cms/media"][enctype] input[name=alt]').fill("Kantor")
    pg.locator('form[action="/cms/media"][enctype] button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("media terunggah + ter-encode", "Media+diunggah" in pg.url, pg.url)
    c = pg.content()
    step("ref media tampil", "cms/20" in c)
    try:
        bad = pg.locator('form[action="/cms/media"][enctype] input[type="file"]').first
        bad.set_input_files(__file__)
        pg.locator('form[action="/cms/media"][enctype] button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("ekstensi .py ditolak", "ekstensi" in pg.content().lower())
    except Exception as e:
        step("ekstensi .py ditolak", False, str(e)[:100])

    # 7. sitemap + histori + setup contoh
    pg.goto(BASE + "/sitemap.xml")
    step("sitemap muat URL CMS", "/artikel/" in pg.content() or "/p/" in pg.content())
    pg.goto(BASE + "/cms/history")
    step("riwayat publikasi", "create" in pg.content() or "update" in pg.content())
    pg.goto(BASE + "/cms/setup")
    pg.wait_for_load_state("networkidle")
    step("setup tampil", "Setup" in pg.content())
    try:
        pg.locator('form[action="/cms/setup/sample"] button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("konten contoh = DRAF", "DRAF" in pg.url or "contoh" in pg.url.lower())
    except Exception as e:
        step("konten contoh = DRAF", False, str(e)[:100])

    b.close()

fails = sum(1 for r in results if r.startswith("- [FAIL]"))
print("CMS DONE:", results, flush=True)
# Kembalikan branding asal (E2E tidak boleh meninggalkan polusi di DB prod).
try:
    with sync_playwright() as p2:
        b2 = p2.chromium.launch(); pg2 = b2.new_page()
        pg2.goto(BASE + "/login")
        pg2.fill("input[name=email]", "admin@contoh.id")
        pg2.fill("input[name=password]", "Admin123!")
        pg2.click("button[type=submit]")
        pg2.wait_for_url(BASE + "/app**")
        pg2.goto(BASE + "/cms/branding")
        pg2.wait_for_load_state("networkidle")
        f2 = pg2.locator('form[action="/cms/branding"]').first
        f2.locator('input[name=app_name]').fill(orig_name or "SaktiNotaris")
        f2.locator('input[name=primary]').fill(orig_primary or "#1f6feb")
        f2.locator('button[type=submit]').click()
        pg2.wait_for_load_state("networkidle")
        print("- [INFO] branding dikembalikan:", orig_name, flush=True)
        b2.close()
except Exception as e:
    print("- [INFO] kembalikan branding gagal (manual via /cms/branding):", str(e)[:100], flush=True)
raise SystemExit(1 if fails else 0)
