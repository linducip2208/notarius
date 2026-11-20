# ARCHITECTURE_REVIEW — modular monolith Go

## Bentuk saat ini (valid, pertahankan)
- `cmd/notaryos` tipis: config → DB open → migrate → app.New → graceful Shutdown (15 dtk, timeout baca/tulis).
- `internal/app`: Router chi + render + Page{Title,Role,UserName,Flash,CSRF,Data} + funcs can/wib.
- Domain per paket: auth, users, clients, requests, corp, deeds, doctpl, docengine, pdf, docx, files, storage, ocr, gov, finance, notify, mfa, cryptoxx, validate, workers, wib, audit, config, database.
- Transaksi hanya di alur uang/nomor: Move, CreateInvoice, RecordPayment(FOR UPDATE), SaveVersion, Approve(FOR UPDATE).
- Session: SQLStore + cleanup/jam; workers pool 4 untuk notifikasi best-effort.

## Keputusan sesi ini
- WIB: DB UTC, tampil Asia/Jakarta via `wib` + func template.
- Proxy: TRUSTED_PROXIES eksplisit; XFF hanya dari proxy tepercaya.
- Nomor: FOR UPDATE + UNIQUE backstop; counter table penuh ditunda (butuh migrasi + uji beban).
- File: temp-resolve OCR, MIME sniff, unduh scoped.
- UI: tambah deed_detail, documents, audit, office, tasks, expenses, tariffs, search, 403/404; sidebar gating via can().

## Utang yang diakui
- Migrasi tidak atomik per-file; split `;` naif (JSON berisi `;` aman karena satu statement per baris? tetap rapuh).
- Pagination LIMIT tanpa cursor/total; N+1 di timeline request dan corp 1+1+1.
- Index kurang: req(office,status), appt(office,time), gov(office,status), clients(office,phone), invoices(office,status,due).
- Tanpa soft-delete (protokol 25 th butuh arsip, bukan DROP).
- approvals/files polimorfik tanpa FK penuh; files.ref tanpa UNIQUE(sha256,office).

## Aturan ke depan
- Semua perubahan DB via migrasi bernomor; tanpa destructive tanpa backup+persetujuan.
- Semua nomor/uang dalam tx; semua baca lintas-kantor wajib office_id.
- Tanpa endpoint pemerintah palsu; tanpa tombol tanpa aksi; tanpa klaim VERIFIED tanpa uji.
