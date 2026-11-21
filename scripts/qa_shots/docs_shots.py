import os

from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:8080"
OUT = r"D:\project laravel\notaris\docs\screenshots\uiux"


def main():
    os.makedirs(OUT, exist_ok=True)
    with sync_playwright() as p:
        b = p.chromium.launch()
        pg = b.new_page(viewport={"width": 1440, "height": 900})
        for url, name in [("/docs", "docs-index"), ("/docs/ktp-masuk", "docs-step")]:
            pg.goto(BASE + url)
            pg.wait_for_load_state("networkidle")
            pg.screenshot(path=os.path.join(OUT, name + ".png"))
            print("shot:", name, flush=True)
        b.close()
    print("DONE")


if __name__ == "__main__":
    main()
