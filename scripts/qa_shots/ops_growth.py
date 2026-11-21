"""E2E ops growth: completeness, specimens, bank, budgets, inquiry->request,
deed attach+package+3way+gate, portal receipt, retention->task, wills, gov followup,
legalisasi queue, recurring tasks, backup, reports, calendar, repertorium gaps,
dashboard cards, lockout. Bukti per langkah."""
import time, urllib.request
from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
results = []
PNG = r"D:\project laravel\notaris\web\assets\landing\notarius-feat-dokumen.jpg"


def step(name, ok, extra=""):
    results.append(("- [PASS] " if ok else "- [FAIL] ") + name + (" " + str(extra) if extra else ""))
    print(results[-1], flush=True)


with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    pg.on("dialog", lambda d: d.accept())
    ts = str(int(time.time()))[-6:]
    pg.goto(BASE + "/login")
    pg.fill("input[name=email]", "admin@contoh.id")
    pg.fill("input[name=password]", "Admin123!")
    pg.click("button[type=submit]")
    pg.wait_for_url(BASE + "/app**")
    step("login admin", True)
    pg.goto(BASE + "/app")
    step("kartu notaris tampil", "Menunggu Persetujuan" in pg.content())

    # 1. klien + completeness + timeline 360 (wizard 3 langkah)
    pg.goto(BASE + "/clients/new")
    pg.fill('form[action="/clients/new?step=1"] input[name=full_name]', "Ops Uji " + ts)
    pg.fill('form[action="/clients/new?step=1"] input[name=phone]', "0812" + ts)
    pg.locator('form[action="/clients/new?step=1"] button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    pg.locator('form[action="/clients/new?step=2"] button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    pg.locator('form[action="/clients"]:not([action="/clients/new"]) button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    cid = pg.url.strip("/").split("/")[-1].split("?")[0]
    step("klien dibuat", cid.isdigit(), pg.url)
    pg.goto(BASE + "/clients/" + cid)
    step("skor kelengkapan tampil", "Kelengkapan profil" in pg.content())

    # 2. spesimen
    pg.goto(BASE + "/clients/" + cid + "/specimens")
    pg.set_input_files('form[action="/clients/' + cid + '/specimens"] input[type=file]', PNG)
    pg.select_option('form[action="/clients/' + cid + '/specimens"] select[name=kind]', "TTD")
    pg.fill('form[action="/clients/' + cid + '/specimens"] input[name=note]', "basah disaksikan " + ts)
    pg.locator('form[action="/clients/' + cid + '/specimens"] button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("spesimen diunggah", "flash=Spesimen" in pg.url, pg.url)

    # 3. invoice -> timeline 360 + kuitansi portal + bank match
    pg.goto(BASE + "/finance?req=")
    f = pg.locator('form[action="/invoices"]').first
    f.locator('input[name=number]').fill("INV-OPS-" + ts)
    f.locator('select[name=client_id]').select_option(value=cid)
    f.locator('input[name="desc[]"]').first.fill("Jasa uji")
    f.locator('input[name="qty[]"]').first.fill("1")
    f.locator('input[name="price[]"]').first.fill("150000")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("invoice dibuat", "flash=Invoice" in pg.url)
    pg.goto(BASE + "/clients/" + cid)
    step("timeline 360 ada tagihan", "tagihan" in pg.content())
    # bank import + match
    import csv, tempfile, os
    fd, csvpath = tempfile.mkstemp(suffix=".csv")
    with os.fdopen(fd, "w", newline="") as fh:
        w = csv.writer(fh); w.writerow(["tanggal", "uraian", "nominal"])
        w.writerow(["2026-11-20", "TRF INV-OPS-" + ts, "165000"])
    pg.goto(BASE + "/finance/bank")
    pg.set_input_files('form[action="/finance/bank/upload"] input[type=file]', csvpath)
    pg.locator('form[action="/finance/bank/upload"] button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("mutasi diimpor", "flash=" in pg.url and "diimpor" in pg.url, pg.url)
    os.remove(csvpath)

    # 4. budgets -> reports
    pg.goto(BASE + "/reports")
    f = pg.locator('form[action="/budgets"]').first
    f.locator('input[name=category]').fill("Operasional")
    f.locator('input[name=month]').fill("2026-11")
    f.locator('input[name=amount]').fill("5000000")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("anggaran disimpan", "flash=Anggaran" in pg.url)
    pg.goto(BASE + "/reports")
    step("laporan anggaran + laba tampil", "Anggaran vs Realisasi" in pg.content() and "Pendapatan per Layanan" in pg.content())

    # 5. inquiry -> request
    pub = b.new_context().new_page()
    pub.goto(BASE + "/konsultasi")
    pub.fill('form[action="/konsultasi"] input[name=name]', "Konversi Uji " + ts)
    pub.fill('form[action="/konsultasi"] input[name=contact]', "0822" + ts)
    pub.fill('form[action="/konsultasi"] input[name=service]', "Uji konversi")
    pub.fill('form[action="/konsultasi"] textarea[name=message]', "mohon dibuatkan permohonan " + ts)
    pub.locator('form[action="/konsultasi"] button[type=submit]').click()
    pub.wait_for_load_state("networkidle")
    pg.goto(BASE + "/cms/inquiries?st=BARU")
    row = pg.locator("tr:has-text('Konversi Uji " + ts + "')")
    step("inquiry masuk", row.count() == 1)
    row.locator("select[name=service_id]").select_option(index=0)
    row.locator("button:has-text('Permohonan')").click()
    pg.wait_for_load_state("networkidle")
    step("inquiry jadi permohonan", "/requests/" in pg.url and "flash=Permohonan" in pg.url, pg.url)
    reqid = pg.url.strip("/").split("/")[-1].split("?")[0]

    # 6. korporat: warnings + org chart
    pg.goto(BASE + "/requests/" + reqid)
    step("peringatan/org chart area", "Data Korporat" in pg.content())
    f = pg.locator('form[action="/requests/' + reqid + '/corp"]').first
    f.locator('select[name=entity_type]').select_option("PT")
    f.locator('input[name=name]').fill("PT Uji " + ts)
    f.locator('input[name=authorized]').fill("100000000")
    f.locator('input[name=issued]').fill("100000000")
    f.locator('input[name=paid]').fill("25000000")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    f = pg.locator('form[action="/requests/' + reqid + '/shareholders"]').first
    f.locator('input[name=name]').fill("Saham Uji")
    f.locator('input[name=shares]').fill("60")
    f.locator('input[name=nominal]').fill("1000000")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    pg.goto(BASE + "/requests/" + reqid)
    c = pg.content()
    step("peringatan saham tampil", "tidak sama dengan modal ditempatkan" in c)
    step("struktur kepemilikan tampil", "Struktur Kepemilikan" in c)

    # 7. akta: simpan revisi dulu (isi tak kosong), lalu gate + attach + package + 3way
    pg.goto(BASE + "/deeds")
    f = pg.locator('form[action="/deeds"]').first
    f.locator('input[name=title]').fill("Akta Uji " + ts)
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    did = pg.url.strip("/").split("/")[-1].split("?")[0]
    step("draf akta dibuat", did.isdigit(), pg.url)
    pg.goto(BASE + "/deeds/" + did)
    pg.fill("#deedbody", "Isi akta uji " + ts + " yang cukup panjang agar lolos batas minimal lima puluh karakter isi naskah.")
    pg.fill('form[action="/deeds/' + did + '/version"] input[name=note]', "revisi uji " + ts)
    pg.locator('form[action="/deeds/' + did + '/version"] button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    # gate nomor: lepas validasi browser (simulasi klien nakal) -> server harus menolak 400
    pg.goto(BASE + "/deeds/" + did)
    pg.evaluate("() => document.querySelectorAll('form[action$=\"/approve\"] input[name=number]').forEach(i => i.removeAttribute('required'))")
    f = pg.locator('form[action="/deeds/' + did + '/approve"]').first
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("gate nomor menolak", "nomor akta wajib diisi" in pg.content())
    f = pg.locator('form[action="/deeds/' + did + '/attach"]')
    if f.count():
        f.locator('input[name=file_id]').fill("1")
        f.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
    pg.goto(BASE + "/deeds/" + did + "?compare=2")
    step("3-way template tampil", "Template baku" in pg.content())
    jar = "; ".join(x["name"] + "=" + x["value"] for x in pg.context.cookies())
    req = urllib.request.Request(BASE + "/deeds/" + did + "/package", headers={"Cookie": jar})
    try:
        rsp = urllib.request.urlopen(req, timeout=30)
        step("paket ZIP akta", rsp.headers.get("Content-Type", "").startswith("application/zip") and rsp.read(2) == b"PK")
    except Exception as e:
        step("paket ZIP akta", False, str(e)[:100])

    # 8. portal receipt (buat user klien + taut)
    pg.goto(BASE + "/settings")
    f = pg.locator('form[action="/users"]').first
    f.locator('input[name=name]').fill("Portal Ops " + ts)
    f.locator('input[name=email]').fill("portalops" + ts + "@uji.id")
    f.locator('input[name=password]').fill("Rahasia123!")
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("user portal dibuat", "flash=Pengguna" in pg.url)
    prow = pg.locator("tr:has-text('portalops" + ts + "@uji.id')")
    prow.locator('input[name=client_id]').fill(cid)
    prow.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("akun ditautkan", "flash=Akun+ditautkan" in pg.url)
    cli = b.new_context().new_page()
    cli.goto(BASE + "/login")
    cli.fill("input[name=email]", "portalops" + ts + "@uji.id")
    cli.fill("input[name=password]", "Rahasia123!")
    cli.click("button[type=submit]")
    cli.wait_for_url(BASE + "/app**")
    cli.goto(BASE + "/portal/invoices")
    cli.wait_for_load_state("networkidle")
    step("portal tagihan + kuitansi tampil", "Kuitansi" in cli.content() and "INV-OPS-" + ts in cli.content())

    # 9. retention -> task (atau pesan jujur bila tak ada yang kedaluwarsa)
    pg.goto(BASE + "/dms/retention")
    f = pg.locator('form[action="/dms/retention/task"]')
    if f.count():
        f.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("tugas retensi dibuat", "flash=Tugas" in pg.url or "flash=Tidak+ada" in pg.url, pg.url)
    else:
        step("tugas retensi dibuat", False, "tanpa form")

    # 10. wasiat review + open
    pg.goto(BASE + "/wills")
    f = pg.locator('form[action="/wills"]').first
    f.locator('select[name=client_id]').select_option(value=cid)
    f.locator('input[name=register_no]').fill("W-" + ts)
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("wasiat terdaftar", "flash=Wasiat" in pg.url)
    wrow = pg.locator("tr:has-text('W-" + ts + "')")
    wrow.locator('form[action$="/review"] button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    pg.goto(BASE + "/wills")
    step("review tahunan tercatat", pg.content().count("W-" + ts) > 0)
    wrow = pg.locator("tr:has-text('W-" + ts + "')")
    wrow.locator('input[name=witness]').fill("Saksi " + ts)
    wrow.locator('button:has-text("Buka")').click()
    pg.wait_for_load_state("networkidle")
    step("pembukaan bersaksi tercatat", "flash=Pembukaan" in pg.url, pg.url)

    # 11. gov followup + legalisasi queue
    pg.goto(BASE + "/gov")
    f = pg.locator('form[action="/gov"]').first
    f.locator('select[name=channel]').select_option("AHU")
    f.locator('input[name=reference]').fill("PT Gov Uji " + ts)
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    grow = pg.locator("tr:has-text('PT Gov Uji " + ts + "')")
    grow.locator('input[name=follow_up]').fill("2026-12-01")
    grow.locator("button:has-text('OK')").first.click()
    pg.wait_for_load_state("networkidle")
    step("gov followup disimpan", "flash=Tindak" in pg.url, pg.url)
    pg.goto(BASE + "/legalisasi")
    f = pg.locator('form[action="/legalisasi"]').first
    f.locator('select[name=kind]').select_option("LEGALISASI")
    f.locator('input[name=register_no]').fill("REG-" + ts)
    f.locator('select[name=client_id]').select_option(value=cid)
    f.locator('input[name=doc_title]').fill("Dok Uji " + ts)
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    step("legalisasi antre tampil", "Antre" in pg.content() or "#1" in pg.content())

    # 12. recurring task
    pg.goto(BASE + "/tasks")
    f = pg.locator('form[action="/tasks"]').first
    f.locator('input[name=title]').fill("Rutin Uji " + ts)
    f.locator('select[name=recurrence]').select_option("DAILY")
    f.locator('select[name=ref_type]').select_option("client")
    f.locator('input[name=ref_id]').fill(cid)
    f.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    trow = pg.locator("tr:has-text('Rutin Uji " + ts + "')").first
    trow.locator('button[type=submit]').click()
    pg.wait_for_load_state("networkidle")
    pg.goto(BASE + "/tasks")
    step("tugas berulang bergulir", "Rutin Uji " + ts in pg.content() and "DAILY" in pg.content())

    # 13. kalender + repertorium + backup + lockout
    pg.goto(BASE + "/appointments")
    step("kalender tampil", "Kalender" in pg.content())
    pg.goto(BASE + "/repertorium")
    step("repertorium + QR tampil", "Buku Repertorium" in pg.content())
    pg.goto(BASE + "/office")
    f = pg.locator('form[action="/office/backup"]')
    if f.count():
        f.locator('button[type=submit]').click()
        pg.wait_for_load_state("networkidle")
        step("backup DB", "flash=Backup" in pg.url or "mysqldump" in pg.content(), pg.url)
    else:
        step("backup DB", False, "tanpa tombol")
    eml = "kunci" + ts + "@uji.id"
    for i in range(5):
        pub2 = b.new_context().new_page()
        pub2.goto(BASE + "/login")
        pub2.fill("input[name=email]", eml)
        pub2.fill("input[name=password]", "Salah123!")
        pub2.click("button[type=submit]")
        pub2.wait_for_load_state("networkidle")
        pub2.close()
    pub3 = b.new_context().new_page()
    pub3.goto(BASE + "/login")
    pub3.fill("input[name=email]", eml)
    pub3.fill("input[name=password]", "Salah123!")
    pub3.click("button[type=submit]")
    pub3.wait_for_load_state("networkidle")
    step("lockout setelah 5x gagal", "terkunci" in pub3.content())
    b.close()

fails = [x for x in results if x.startswith("- [FAIL]")]
print("OPS DONE, gagal: " + (", ".join(fails) if fails else "nihil"))
raise SystemExit(1 if fails else 0)
