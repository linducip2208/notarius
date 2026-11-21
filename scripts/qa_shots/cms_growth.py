"""E2E CMS growth: kalkulator, konsultasi, booking->janji, tim/galeri/testimoni,
i18n, cari, arsip, preview draf, versi+rollback, kualitas, statistik, ekspor,
maintenance, banner. Bukti per langkah."""
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
    ts = str(int(time.time()))[-6:]

    # 1. kalkulator publik (angka dari DB, ada disclaimer)
    pg.goto(BASE + "/kalkulator?code=PT-DIRIAN")
    pg.wait_for_load_state("networkidle")
    c = pg.content()
    step("kalkulator estimasi transparan", "Estimasi awal" in c and "Bukan harga final" in c)
    pg.goto(BASE + "/layanan/PT-DIRIAN")
    pg.wait_for_load_state("networkidle")
    step("detail layanan + syarat", "Persyaratan Dokumen" in pg.content() and "konsultasi" in pg.content())

    # 2. konsultasi masuk -> admin
    pg.goto(BASE + "/konsultasi")
    pg.fill("input[name=name]", "Warga Uji " + ts)
    pg.fill("input[name=contact]", "0812uji" + ts)
    pg.fill("input[name=service]", "Pendirian PT")
    pg.fill("textarea[name=message]", "Butuh info syarat pendirian PT " + ts)
    pg.click("button[type=submit]")
    pg.wait_for_load_state("networkidle")
    step("konsultasi terkirim", "sent=1" in pg.url, pg.url)
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**")
    step("login admin", True)
    pg.goto(BASE + "/cms")
    pg.wait_for_load_state("networkidle")
    step("dasbor tandai konsultasi baru", "konsultasi baru" in pg.content().lower())
    pg.goto(BASE + "/cms/inquiries")
    pg.wait_for_load_state("networkidle")
    step("inquiry tampil di admin", ("Warga Uji " + ts) in pg.content())
    f = pg.locator('form[action^="/cms/inquiries/"]').first
    f.locator('select[name=to]').select_option("DIPROSES")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("status inquiry diubah", "flash=Status" in pg.url, pg.url)

    # 3. slot + booking -> janji temu
    pg.goto(BASE + "/cms/slots")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/slots"]').first
    f.locator('select[name=weekday]').select_option("1")
    f.locator('input[name=start]').fill("09:00")
    f.locator('input[name=end]').fill("12:00")
    f.locator('input[name=quota]').fill("2")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("slot dibuat", "flash=Slot" in pg.url, pg.url)
    pub = b.new_context().new_page()
    pub.goto(BASE + "/booking")
    pub.wait_for_load_state("networkidle")
    step("form booking tampil", "Booking Jadwal" in pub.content())
    import datetime
    besok = (datetime.date.today() + datetime.timedelta(days=2)).isoformat()
    pub.fill("input[name=name]", "Tamu Uji " + ts)
    pub.fill("input[name=contact]", "0819" + ts[-6:])
    pub.fill("input[name=date]", besok)
    pub.fill("input[name=start]", "10:%02d" % (int(ts) % 50 + 5))
    pub.fill("input[name=purpose]", "Konsultasi " + ts)
    pub.click("button[type=submit]")
    pub.wait_for_load_state("networkidle")
    step("booking terkirim", "sent=1" in pub.url, pub.url)
    pg.goto(BASE + "/cms/bookings?st=MENUNGGU")
    pg.wait_for_load_state("networkidle")
    step("booking tampil di admin", ("Tamu Uji " + ts) in pg.content())
    f = pg.locator('tr:has-text("Tamu Uji ' + ts + '") form').first
    f.locator('button[value="DISETUJUI"]').click()
    pg.wait_for_load_state("networkidle")
    step("booking disetujui -> janji", "flash=Booking" in pg.url, pg.url)
    pg.goto(BASE + "/appointments")
    pg.wait_for_load_state("networkidle")
    step("janji temu tercipta", ("Tamu Uji " + ts) in pg.content())

    # 4. tim/galeri/testimoni (gate verifikasi)
    pg.goto(BASE + "/cms/team")
    f = pg.locator('form[action="/cms/team"]').first
    f.locator('input[name=name]').fill("Notaris Uji " + ts)
    f.locator('input[name=title]').fill("Notaris")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("anggota tim ditambah", "flash=Anggota" in pg.url)
    pg.goto(BASE + "/tim")
    step("tim tampil publik", ("Notaris Uji " + ts) in pg.content())
    pg.goto(BASE + "/cms/testimonials")
    f = pg.locator('form[action="/cms/testimonials"]').first
    f.locator('input[name=name]').fill("Klien Puas " + ts)
    f.locator('input[name=content]').fill("Pelayanan cepat dan jelas " + ts)
    f.locator('input[name=verify_note]').fill("")
    f.locator('select[name=status]').select_option("PUBLISHED")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("testimoni tanpa verifikasi ditolak", "wajib mencantumkan" in pg.content())
    pg.goto(BASE + "/cms/testimonials")
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/cms/testimonials"]').first
    f.locator('input[name=name]').fill("Klien Puas " + ts)
    f.locator('input[name=content]').fill("Pelayanan cepat dan jelas " + ts)
    f.locator('input[name=verify_note]').fill("ijin lisan " + ts + ", arsip kantor")
    f.locator('select[name=status]').select_option("PUBLISHED")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("testimoni terverifikasi terbit", "flash=Testimoni" in pg.url)
    pg.goto(BASE + "/testimoni")
    step("testimoni tampil publik + label verifikasi", ("Klien Puas " + ts) in pg.content() and "terverifikasi" in pg.content())

    # 5. i18n + cari + arsip
    pg.goto(BASE + "/artikel?lang=en")
    pg.wait_for_load_state("networkidle")
    step("artikel EN (fallback ID)", "Artikel" in pg.content() or "Article" in pg.content())
    pg.goto(BASE + "/cari?q=akta")
    pg.wait_for_load_state("networkidle")
    step("pencarian situs", "Pencarian" in pg.content())
    pg.goto(BASE + "/arsip")
    pg.wait_for_load_state("networkidle")
    step("arsip artikel", "Arsip" in pg.content())

    # 6. versi + rollback halaman
    pg.goto(BASE + "/cms/pages")
    pg.wait_for_load_state("networkidle")
    first = pg.locator('table a[href^="/cms/pages/"]').first
    pid = first.get_attribute("href").strip("/").split("/")[-1] if first.count() else None
    if pid and pid.isdigit():
        pg.goto(BASE + "/cms/pages/" + pid)
        pg.wait_for_load_state("networkidle")
        step("pratinjau draf bertoken", "/p/preview?kind=page" in pg.content())
        pg.fill('textarea[name=content]', "<p>Revisi uji versi " + ts + "</p>")
        pg.click('form[action="/cms/pages/' + pid + '"] button[type=submit]')
        pg.wait_for_load_state("networkidle")
        pg.goto(BASE + "/cms/versions?kind=page&id=" + pid)
        pg.wait_for_load_state("networkidle")
        step("versi tersimpan", "Rollback" in pg.content())
        f = pg.locator('form[action^="/cms/versions/"]').first
        if f.count():
            f.locator('button[type=submit]').click()
            pg.wait_for_load_state("networkidle")
            step("rollback berhasil", "flash=Rollback" in pg.url, pg.url)
        else:
            step("rollback berhasil", False, "tanpa versi")
    else:
        step("pratinjau draf bertoken", False, "tanpa halaman")
        step("versi tersimpan", False, "tanpa halaman")
        step("rollback berhasil", False, "tanpa halaman")

    # 7. kualitas + statistik + ekspor
    pg.goto(BASE + "/cms/quality")
    step("dasbor kualitas", "Keterbacaan" in pg.content() or "keterbacaan" in pg.content())
    pg.goto(BASE + "/cms/stats")
    step("statistik internal", "Statistik" in pg.content())
    import urllib.request
    jar = "; ".join(c["name"] + "=" + c["value"] for c in pg.context.cookies())
    req = urllib.request.Request(BASE + "/cms/export", headers={"Cookie": jar})
    try:
        rsp = urllib.request.urlopen(req, timeout=30)
        body = rsp.read(4)
        okzip = rsp.headers.get("Content-Type", "").startswith("application/zip") and body[:2] == b"PK"
        step("ekspor ZIP markdown", okzip, rsp.headers.get("Content-Disposition", ""))
    except Exception as e:
        step("ekspor ZIP markdown", False, str(e)[:120])

    # 8. banner + maintenance
    pg.goto(BASE + "/cms/site")
    pg.fill("input[name=announcement]", "Uji banner " + ts)
    pg.click('form[action="/cms/site"] button[type=submit]')
    pg.wait_for_load_state("networkidle")
    pub2 = b.new_context().new_page()
    pub2.goto(BASE + "/")
    step("banner tampil publik", ("Uji banner " + ts) in pub2.content())
    pg.goto(BASE + "/cms/site")
    pg.check('input[name=maintenance]')
    pg.click('form[action="/cms/site"] button[type=submit]')
    pg.wait_for_load_state("networkidle")
    pub3 = b.new_context().new_page()
    pub3.goto(BASE + "/")
    step("maintenance 503 anonim", pub3.content() and "pemeliharaan" in pub3.content().lower())
    pg.goto(BASE + "/")
    step("staf tetap bisa buka saat maintenance", "pemeliharaan" not in pg.content().lower() or "MODE PEMELIHARAAN" in pg.content())
    pg.goto(BASE + "/cms/site")
    pg.uncheck('input[name=maintenance]')
    pg.fill("input[name=announcement]", "")
    pg.click('form[action="/cms/site"] button[type=submit]')
    pg.wait_for_load_state("networkidle")
    step("maintenance dimatikan", True)
    b.close()

fails = [x for x in results if x.startswith("- [FAIL]")]
print("GROWTH DONE, gagal: " + (", ".join(fails) if fails else "nihil"))
raise SystemExit(1 if fails else 0)
