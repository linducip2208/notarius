# Docker: Go + Tesseract jadi satu

## Bangun (di server Linux / CI)
```bash
docker build -t notarius:latest -f deploy/Dockerfile .
```

## Jalankan
```bash
# /etc/notarius/env chmod 600, berisi DB_* + NOTARYOS_FIELD_KEY (+ opsional OCR_BINARY)
docker run -d --name notarius --restart unless-stopped \
  --env-file /etc/notarius/env -p 127.0.0.1:8080:8080 \
  -v notarius-storage:/data notarius:latest
```

## Catatan jujur
- Tesseract tetap binary terpisah di dalam image (tidak bisa di-`go build`);
  Go memanggilnya sebagai subprocess — crash OCR tidak menumbangkan service.
- Tanpa `tesseract-ocr-ind`, ekstraksi Indonesia gagal → fallback manual (by design).
- MySQL tetap di luar image (Laragon/Managed DB/RDS). Health: `/healthz`, readiness: `/readyz`.
- Update: build ulang image → `docker stop/rm` → run lagi (data di volume + MySQL, aman).
