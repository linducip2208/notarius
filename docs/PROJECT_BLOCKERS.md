# PROJECT BLOCKERS — aktif

| ID | Blocker | Dampak | Jalan keluar |
|---|---|---|---|
| B-04 | AV host blokir `*.test.exe` | `go test` langsung gagal di Windows | Workaround rename biner (terbukti); CI Linux disarankan |
| B-06 | Tinjauan notaris belum ada | DILARANG produksi data sungguhan | Jadwalkan review praktisi atas alur akta/wasiat/retensi |
| B-08 | TLS publik + COOKIE_SECURE | Sesi/MFA rentan sniffing di HTTP | Terminasi TLS + env true sebelum go-live |
| B-09 | Uji restore VPS belum dilakukan | Backup belum terbukti | Jadwalkan restore ke DB kosong + catat RTO |
| B-10 | Tesseract di Linux | OCR butuh binary + traineddata | `apt install tesseract-ocr tesseract-ocr-ind` / Dockerfile tersedia |
