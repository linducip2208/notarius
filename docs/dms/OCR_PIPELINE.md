# DMS — Pipeline OCR

## 1. Kontrak provider (terverifikasi `internal/ocr/ocr.go`)

```go
type Provider interface {
    Name() string
    Extract(ctx context.Context, filePath string) (Result, error)
}
type Result struct {
    ID string; DocType DocType; Fields map[string]string
    Confidence float64; Provider, Provenance string
    CreatedAt time.Time; VerifiedByHuman bool; VerifiedBy *int64
}
```

`VerifiedByHuman` selalu `false` saat dibuat — verifikasi dilakukan terpisah.

## 2. Implementasi

| Provider | `Name()` | Kapan dipakai |
|---|---|---|
| Tesseract lokal (`internal/ocr/tesseract.go`) | `local-tesseract` | default; bahasa `ind` |
| `NoopProvider{Reason}` | `unconfigured` | engine belum terinstal → error deskriptif |
| `manual` / AI eksternal | sesuai nama | hanya bila dikonfigurasi & diotorisasi |

## 3. Tahap pipeline

1. Ambil biner via `Store.Open(ref)` → materialisasi ke path temp lokal.
2. `Classify(nama)` → saran `DocType` (KTP/KK/NPWP/OTHER).
3. Jalankan Tesseract `ind` → parse teks → `Fields` + `Confidence`.
4. Tulis `ai_results{file_id, provider, provenance, doc_type, fields, confidence}`.
5. Update `files.ocr_status` → `DONE` (atau `FAILED` + alasan).
6. Hapus file temp. Setiap tahap dicatat di audit.

## 4. Provenance

`provenance` wajib: versi biner Tesseract + kode bahasa + referensi
konfigurasi, agar hasil dapat direproduksi/diaudit.

## 5. Aturan keras

- Hasil OCR **bukan kebenaran**: wajib verifikasi manusia sebelum menyentuh
  `identities` atau draf akta (lihat komentar paket `ocr`).
- Local-first: tidak ada pengiriman data sensitif ke AI eksternal tanpa
  otorisasi eksplisit.
- OCR ulang menambah row baru; tidak menimpa riwayat.

## 6. Operasional

- Prasyarat host: biner Tesseract + data `ind` terinstal; bila tidak ada,
  aplikasi tetap jalan dengan `NoopProvider` + fallback manual.
- Kegagalan umum: scan buram → confidence rendah → minta unggah ulang/scan ulang.
