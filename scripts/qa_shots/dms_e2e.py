"""E2E DMS: login, upload PNG ke klien 1, jalankan OCR (Tesseract asli),
verifikasi file, pindah workflow, buat share, unduh via /s/.
Output: docs/qa/DMS_E2E_LOG.md (bukti lolos/gagal per langkah).
"""
import os

from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
PNG = r"D:\project laravel\notaris\web\assets\landing\notarius-hero.png"
LOG = []
SHARE_URL = ""


def step(name, ok, info="", fatal=True):
    LOG.append(f"- [{'PASS' if ok else 'FAIL'}] {name}" + (f" — {info}" if info else ""))
    print(LOG[-1], flush=True)
    if not ok and fatal:
        raise SystemExit(f"E2E BERHENTI di: {name} — {info}")


def main():
    try:
        run()
    finally:
        os.makedirs(os.path.dirname(r"D:\project laravel\notaris\docs\qa\DMS_E2E_LOG.md"), exist_ok=True)
        with open(r"D:\project laravel\notaris\docs\qa\DMS_E2E_LOG.md", "w", encoding="utf-8") as f:
            f.write("# DMS E2E LOG — " + SHARE_URL + "\n\n" + "\n".join(LOG) + "\n")
        print("LOG WRITTEN", flush=True)


def run():
    with sync_playwright() as p:
        b = p.chromium.launch()
        pg = b.new_page()
        # tombol verifikasi/restore/disposisi memakai confirm() JS — terima agar submit jalan
        pg.on("dialog", lambda d: d.accept())
        pg.goto(BASE + "/login")
        pg.fill('input[name="email"]', "admin@contoh.id")
        pg.fill('input[name="password"]', "Admin123!")
        pg.click('button[type="submit"]')
        pg.wait_for_url(BASE + "/app**")
        step("login dashboard", True)
        pg.goto(BASE + "/clients/1")
        pg.locator('a[href="#tab-doc"]').click()
        pg.locator('form[action$="/files"] input[type="file"]').set_input_files(PNG)
        pg.locator('form[action$="/files"] button[type="submit"]').click()
        pg.wait_for_load_state("networkidle")
        step("upload PNG", "Berkas diunggah" in pg.content())
        fid = None
        for row in pg.locator("table tbody tr").all():
            try:
                href = row.locator("a[href^='/files/']").first.get_attribute("href")
                if href and href.startswith("/files/") and href.count("/") == 2:
                    fid = href.split("/")[-1]
                    break
            except Exception:
                continue
        step("file tercatat di detail klien", fid is not None, f"id={fid}")
        pg.locator('a[href="#tab-doc"]').click()
        # OCR via detail klien (Tesseract asli bisa >30 dtk)
        pg.locator(f'form[action="/files/{fid}/ocr"] button').click(timeout=120000)
        pg.wait_for_load_state("networkidle", timeout=120000)
        step("OCR Tesseract berjalan", "OCR+selesai" in pg.content() or "OCR selesai" in pg.content(), pg.url)
        pg.goto(f"{BASE}/documents/{fid}")
        pg.wait_for_load_state("networkidle")
        step("detail DMS", "Metadata" in pg.content(), pg.url)
        # verifikasi manusia
        pg.select_option('form[action$="/verify"] select', "VERIFIED")
        pg.locator('form[action$="/verify"] button').click()
        pg.wait_for_load_state("networkidle")
        step("verifikasi manusia", "Verifikasi+dicatat" in pg.url, pg.url)
        # workflow (auto-move saat unggah: RECEIVED->CLASSIFICATION; lanjutkan)
        pg.select_option('form[action$="/workflow"] select', "ADMIN_REVIEW")
        pg.locator('form[action$="/workflow"] button').click()
        pg.wait_for_load_state("networkidle")
        step("workflow CLASSIFICATION->ADMIN_REVIEW", "Workflow+berpindah+ke+ADMIN_REVIEW" in pg.url, pg.url)
        # share terbatas
        pg.fill('form[action$="/share"] input[name="hours"]', "1")
        pg.fill('form[action$="/share"] input[name="max"]', "3")
        pg.locator('form[action$="/share"] button').click()
        pg.wait_for_load_state("networkidle")
        tok = pg.locator("code").first.inner_text().strip().lstrip("/s/")
        step("share token dibuat", len(tok) == 32, tok[:8] + "...")
        global SHARE_URL
        SHARE_URL = f"{BASE}/s/{tok}"
        # unduh anonim via share (goto melempar "Download is starting" = BUKTI unduhan jalan)
        ctx2 = b.new_context(accept_downloads=True)
        p2 = ctx2.new_page()
        try:
            with p2.expect_download(timeout=30000) as dl_info:
                try:
                    p2.goto(SHARE_URL)
                except Exception:
                    pass
            dl = dl_info.value
            step("unduh anonim via /s/", True, f"file={dl.suggested_filename}", fatal=False)
        except Exception as e:
            step("unduh anonim via /s/", False, str(e)[:120], fatal=False)
        ctx2.close()
        # search OCR content
        try:
            pg.goto(BASE + "/dms/search?q=notarius")
            pg.wait_for_load_state("networkidle")
            step("search OCR", "notarius-hero" in pg.content().lower() or "hero" in pg.content().lower(), fatal=False)
        except Exception as e:
            step("search OCR", False, str(e)[:100], fatal=False)
        b.close()
    print("RUN DONE", flush=True)


if __name__ == "__main__":
    main()
