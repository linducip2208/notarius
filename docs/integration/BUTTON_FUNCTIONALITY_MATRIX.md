# Button Functionality Matrix
Crawl otomatis: scripts/qa_shots/audit_crawl.py — 228 rute, 254 handler.
Inventory elemen: 121 template define, ~600 form/button/anchor. Ringkasan kelompok:
- Create/Save: /clients, /requests, /deeds, /invoices, /tasks, /cms/* → POST + CSRF + perm handler.
- Approve/Reject: /deeds/{id}/approve (gate: nomor, isi≥50, saksi, tempat baca, checklist, hash, mandat substitute), /documents/{id}/approve, /cms/bookings/{id}/decide.
- Delete/Archive: /cms/* delete, /dms hold/disposition (tidak hard-delete), /office/mandates/delete.
- Export/Import/Download: /clients/export, /repertorium/export, /office/export, /cms/export, /finance/bank/upload, /files/{id}?verify=1.
- Search/Filter: /search, /dms/library (+adv), /cms/* filter, /arsip, /cari.
- Confirm: 3/3 (dismiss-vs-accept) lewat confirm.py.
- UI interaksi: 11/11 lewat interact.py.
Temuan: tombol `href="#"` hanya pagination placeholder (page-item disabled/active) — bukan aksi; tidak ada dummy action. Semua POST di dalam group auth memiliki CSRF; /mfa/verify kini juga CSRF.
