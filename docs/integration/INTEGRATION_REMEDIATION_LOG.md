# Integration Remediation Log
1. Izin korporat & janji/request/search/file: tambah pengecekan server-side + office scope (P0).
2. Portal request detail: default-deny untuk role client, isolation by linked client (P0).
3. Repertorium: lock + retry + audit repertorium.insert (P0).
4. Hash integritas: self-heal bila kosong, 409 bila mismatch, verifikasi ?verify=1, karantina REJECTED (P0).
5. Confidentiality: officeExport & compare & detail RESTRICTED → forbidden (P0).
6. Share download atomik (P0).
7. Formalitas akta: saksi, tempat baca, renvoi, checklist verified; hash content; locked=1 (P1).
8. Mandat notaris pengganti + SK + periode + gate approve (P1).
9. Money: parse int64 rupiah untuk payment/invoice/expense/tariff/budget; VOID flow; bank manual amount check + audit (P1).
10. MFA stateless HMAC + CSRF /mfa/verify + re-auth enroll (P1).
11. Audit hash chain + Verify (P1).
12. Protocol location/status/holder + legalisasi deed_id/GROSSE (P1).
13. Privacy notice, consent, webhook SSRF guard, mysqldump defaults-file, safe CSS, idempotent migration (P1/P2).
