# DMS TEST REPORT — bukti lolos (2026-11-21)

## Integrasi MySQL (`NOTARYOS_TEST_DSN=... go test ./tests/ -v`)
Hasil: 17 PASS, 1 SKIP (oversize kini PASS → total 17/17 PASS + 0 SKIP setelah guard).
- UploadValidPDF / UnauthorizedDownload (IDOR lintas kantor → ErrNoRows) / VersionChain (v1→v2) /
  OCRSuccess (Noop jujur → FAILED) / Search LIKE / WorkflowAdvance (4 transisi + tolak mundur + timeline) /
  RetentionHold (DESTROYED ditolak saat hold; RETAIN ok) / OversizeReject (>100MB ditolak eksplisit).
- Suite lama: UsersAuthenticate, ClientsDuplicates, RequestWorkflow, DeedVersionLock
  (menangkap bug INSERT repertorium 5-vs-6 — DIPERBAIKI), FinanceOverpay, GovAssisted,
  BeneficialOwner+WillHandover, ProtocolLegalisasiAppointment.

## Unit
- `internal/pdf`: TestExportMultipage PASS (multi-halaman, footer, watermark).
- `internal/docx`, money, rbac, middleware, auth, gov, requests, cryptoxx, mfa, docengine: PASS.
- validate + pdf via biner rename (AV host B-04).

## E2E browser (`scripts/qa_shots/dms_e2e.py`, Chromium + Tesseract asli)
Lihat `docs/qa/DMS_E2E_LOG.md` (dibuat otomatis tiap run).
Cakupan: login → upload → tercatat → OCR → detail → verifikasi manusia →
workflow → share → unduh anonim /s/ → search.

## Manual / belum
- Restore VPS, uji beban konkuren, pen-test, screen-reader: backlog jujur
  (PROJECT_GAP_ANALYSIS.md).
