# Log Tinjauan Regulasi (Regulatory Review Log)

> Seluruh layanan berstatus NEEDS_LEGAL_REVIEW hingga ada tinjauan notaris penasihat.

## Format Entri

| Kolom | Isi |
|---|---|
| Tanggal | YYYY-MM-DD |
| Reviewer | Nama & peran (mis. Notaris Penasihat) |
| Regulasi | Nama generik (mis. UUJN 2/2014) — tanpa nomor pasal spesifik |
| Cakupan | Dokumen/layanan yang ditinjau |
| Temuan | Ringkasan temuan |
| Tindak lanjut | Perbaikan / pemilik / tenggat |
| Status | OPEN / IN_PROGRESS / CLOSED / NEEDS_LEGAL_REVIEW |

## Template Kosong (salin per review)

```
Tanggal:
Reviewer:
Regulasi:
Cakupan:
Temuan:
Tindak lanjut:
Status: NEEDS_LEGAL_REVIEW
```

## Contoh Entri — Belum Direview

| Kolom | Isi |
|---|---|
| Tanggal | 2026-11-24 |
| Reviewer | (belum ada — menunggu notaris penasihat) |
| Regulasi | UUJN 2/2014; UU PT 40/2007; PP 24/2016; UU PDP 27/2022 (nama generik) |
| Cakupan | SERVICE_AUTHORITY_MATRIX, LEGAL_REQUIREMENT_MATRIX, NOTARY_PRACTICE_VALIDATION |
| Temuan | Belum ditinjau; nomor pasal/ayat sengaja tidak dicantumkan |
| Tindak lanjut | Jadwalkan review penasihat; pemilik: pimpinan kantor |
| Status | NEEDS_LEGAL_REVIEW |

## Riwayat

| Tanggal | Reviewer | Hasil singkat | Status |
|---|---|---|---|
| 2026-11-24 | — | Inisiasi log, menunggu review pertama | NEEDS_LEGAL_REVIEW |
