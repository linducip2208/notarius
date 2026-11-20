# FEATURE_GAP_ANALYSIS — F-001…F-028 + temuan baru (status jujur)

Legenda: VERIFIED (kode+uji hijau), IMPLEMENTED (kode ada, uji kurang), IN_PROGRESS, BLOCKED, NOT_STARTED.

| ID | Nama | Lokasi | Status | Kekurangan | Risiko | Prioritas | Acceptance | Test |
|----|------|--------|--------|------------|--------|-----------|------------|------|
| F-001 | Auth sesi | auth, app login/logout/mfa | VERIFIED | — | Sesi dicuri bila HTTP | P0 | bcrypt12, sesi SQL, revoke, lockout implisit rate-limit | auth_test, login flow |
| F-002 | RBAC 9/19 | rbac, middleware, sidebar can() | IMPLEMENTED | Tanpa test handler-permission matrix | client lihat menu staf bila gating lolos | P0 | matrix 9 peran × routes kunci | baru: rbac handler test |
| F-003 | Kantor & pengguna | users, settings, office | IMPLEMENTED | tanpa deactivate/role-edit | akun zombie | P1 | CRUD+aktif/nonaktif+audit | users test |
| F-004 | Migrasi | database, 001-009 | IMPLEMENTED | per-file tidak atomik; split `;` naif | partial migrate | P1 | tx per-file + rerun idempoten | migration rerun test |
| F-005 | Audit log | audit, auditPage | IMPLEMENTED | UI baru belum diuji browser | — | P0 | semua mutasi P0 tercatat, imutabel | audit insert test |
| F-006/007/008 | Klien/duplikat/identitas | clients, client_detail | VERIFIED | tanpa edit/hapus | — | P0 | validasi NIK/email/telp, verifikasi manusia | integration TestClientsDuplicates |
| F-009 | Perwakilan & BO | clients reps, corp BO | VERIFIED | — | — | P0 | reps+BO pct/PEP | TestBeneficialOwner |
| F-010 | Katalog layanan | services seed | VERIFIED | — | — | P1 | 22 layanan aktif | seed count |
| F-011 | Permohonan workflow | requests Move+events | VERIFIED | tanpa filter status server-side penuh | — | P0 | 9 status legal, mundur ditolak | TestRequestWorkflow |
| F-012 | Korporat | corp, request_detail | VERIFIED | — | — | P0 | PT/CV/dll + saham+pengurus+BO | integration |
| F-013 | Perjanjian | deeds kind Agreement/POA/Prenup | IMPLEMENTED | tanpa wizard per-jenis | alur salah jenis | P1 | template per layanan + validasi | manual notaris |
| F-014 | Wasiat | wills register+handover | VERIFIED | — | akses bocor bila guard lupa | P0 | hanya notaris; handover sekali | TestWillHandover |
| F-015 | Akta draft+final | deeds, deed_detail | IMPLEMENTED | SaveVersion tanpa route dulu → kini ada; perlu uji browser | final diubah | P0 | review→approve, lock, repertorium | TestDeedVersionLock |
| F-016 | Repertorium | Approve auto-insert | IMPLEMENTED | MAX+1 race → FOR UPDATE baru; perlu uji 2-goroutine | nomor ganda | P0 | entry_no unik/urut per kantor | uji konkuren |
| F-017 | Protokol | protocols + create | IMPLEMENTED | UI baru; tanpa cetak | — | P1 | bendel unik/tahun | TestProtocol |
| F-018 | Legalisasi | legalisasi create+advance | IMPLEMENTED | UI advance baru | — | P0 | REGISTERED→COMPLETED→HANDED_OVER | TestLegalisasi |
| F-019 | Template+klausul | doctpl editor+approve | VERIFIED | — | template liar dipakai | P1 | hanya approved → draft | manual |
| F-020 | Doc engine | docengine | VERIFIED | — | — | P1 | render/missing/hash | docengine_test |
| F-021 | PDF/DOCX | pdf, docx | IMPLEMENTED | PDF single-page truncate jujur; perlu uji dokumen panjang | format rusak | P1 | PDF+DOCX valid, watermark DRAFT, pagination | docx_test + uji panjang |
| F-022 | OCR | ocr tesseract+noop, files RunOCR temp | IMPLEMENTED | butuh binary + uji KTP asli + review UI | data OCR dianggap benar | P1 | provenance + verifikasi manusia wajib | uji Tesseract lokal |
| F-023 | AHU/OSS asistensi | gov scoped advance/check | IMPLEMENTED |tanpa API resmi (by design) | klaim integrasi palsu | P0 | checklist+bukti manual | TestGovAssisted |
| F-024 | Upload+unduh privat | files+storage, /files/{id} | IMPLEMENTED | perlu uji spoof/traversal | exfiltrasi antar-kantor | P0 | MIME sniff, office-scope, nama sanitasi | upload test |
| F-025 | Invoice/bayar | finance | VERIFIED | — | overpay/double | P0 | int64, anti-overpay, VOID tolak | TestFinanceOverpay |
| F-026 | Portal terisolasi | portal, user_client_links | IMPLEMENTED | staf lihat semua (by design); perlu uji client A vs B | bocor antar-klien | P0 | server-side filter | portal isolation test |
| F-027 | Dashboard/laporan | reqs counts, finance outstanding, metrics | IMPLEMENTED | tanpa grafik/filter periode/export | angka dipercaya buta | P1 | real-query, no fabrikasi | counts test |
| F-028 | Deploy/backup | deploy/, backup.sh/restore.sh | IMPLEMENTED | belum uji di VPS; HSTS tanpa TLS percuma | restore gagal saat darurat | P0 | backup GPG + restore terbukti + rollback | uji restore |
| NEW | Tasks/pengingat | tasks (009) | IMPLEMENTED | tanpa notifikasi jatuh tempo | — | P1 | CRUD+done per kantor | manual |
| NEW | Expenses/kas | expenses | IMPLEMENTED | tanpa rekonsiliasi bank | — | P1 | catat+audit | manual |
| NEW | Tarif | price_items | IMPLEMENTED | tanpa PPN otomatis | — | P1 | CRUD | manual |
| NEW | Dokumen global | documents | IMPLEMENTED | tanpa preview/versi UI | — | P1 | search+unduh otorisasi | manual |
| NEW | Search global | /search | IMPLEMENTED | LIKE tanpa FTS | lambat skala besar | P2 | klien+permohonan+akta | manual |
| NEW | Error pages | 403/404 | IMPLEMENTED | tanpa 500 kustom | — | P1 | forbidden/notFound + pesan minta akses | browser test |
