# API/Handler Connectivity
- Route→handler: 228 rute terdaftar; semua handler ada (readyz inline handler).
- Handler→store: setiap handler memanggil store/SQL yang sesuai; multi-tabel mutations memakai tx (deeds.Approve, MergeClients, audit chain, invoice/payment).
- Permission: 26 handler awalnya tanpa rbac terverifikasi crawler; ditindaklanjuti: corpSave/Add, shareholderAdd/Link, officerAdd, boAdd, appointmentCreate/Status, requestList/Create/Move/Detail, gov, legalisasi, search, fileUpload/OCR, audit.view — kini punya pengecekan server-side.
- Error: 403 forbidden, 404 not-found, 400 validasi, 402 tier gate, 409 conflict (lock/duplikat/hash), 507 quota, 503 service unavailable.
- CSRF: semua POST auth + /mfa/verify; login dikecualikan (standar).
- Success response tidak mengabaikan DB error (tx commit/rowsAffected dicek pada kritis).
