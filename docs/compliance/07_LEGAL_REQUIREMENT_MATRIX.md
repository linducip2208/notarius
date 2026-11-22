# Legal Requirement Traceability Matrix (ringkas)
| ID | Kewajiban | Modul/Route | Bukti teknis | Status |
|---|---|---|---|---|
| LC-0001 | Pencatatan akta & repertorium | /deeds/{id}/approve, /repertorium | tx + UNIQUE + retry + audit | COMPLIANT_TECHNICAL_EVIDENCE |
| LC-0002 | Formalitas (saksi, pembacaan) | deed approve form | field wajib + 400 gate | PARTIALLY_IMPLEMENTED (interpretasi Notaris) |
| LC-0003 | Minuta/protokol fisik tetap dijaga | /protocols | location/status/holder + disclaimer | PARTIALLY_IMPLEMENTED |
| LC-0004 | NIK tidak plaintext | identities | AES-GCM + last4 + tes | COMPLIANT_TECHNICAL_EVIDENCE |
| LC-0005 | Notifikasi insiden | security_incidents + SOP | tabel + dokumen | PARTIALLY_IMPLEMENTED |
| LC-0006 | Hak subjek data | /privasi + SOP | halaman notice + SOP | PARTIALLY_IMPLEMENTED |
| LC-0007 | Retensi arsip | DMS retention | legal hold + no auto delete | PARTIALLY_IMPLEMENTED (25y perlu review) |
| LC-0008 | TTE bukan PSrE | deedSign, /verify | label NON-PSrE + tes | COMPLIANT_TECHNICAL_EVIDENCE |
| LC-0009 | Mandat pengganti | /office/mandates | SK + periode + gate | COMPLIANT_TECHNICAL_EVIDENCE |
| LC-0010 | PPN dicatat | invoice 11% | DECIMAL/int64 + disclaimer | PARTIALLY_IMPLEMENTED (tarif configurable needed) |
Status lain: HUMAN_REVIEW_REQUIRED untuk interpretasi hukum, SOURCE_DOCUMENT_NOT_VERIFIED untuk pasal tak terverifikasi.
