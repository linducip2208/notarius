# testdata — fixture KTP PALSU

`dummy_ktp.png` (1200x760) adalah gambar KTP **PALSU yang jelas-jelas dummy**:

- Nama: `CONTOH SAJA` — NIK: `0000000000000000` — bukan data orang nyata.
- Dibuat oleh `generate_dummy.go` (stdlib saja, font bitmap 3x5):
  `go run generate_dummy.go` dari direktori ini untuk regenerasi.
- Font balok TIDAK dimaksudkan terbaca Tesseract — fixture ini HANYA untuk
  quality-gate (`AssessImage`) + preprocessing (`BuildVariants`), yang murni
  fungsi piksel tanpa Tesseract/DB sehingga **selalu jalan di CI**.
- Akurasi parsing diuji dari konstanta teks di `ktp_parse_test.go`
  (tanpa gambar, tanpa OCR). Uji OCR ujung-ke-ujung (Tesseract asli)
  bersifat opt-in lokal: `NOTARYOS_TEST_TESSERACT=1` — lihat
  `docs/TEST_STRATEGY.md`.
