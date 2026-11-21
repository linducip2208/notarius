# Panduan Manajemen Klien & CRM (SaktiNotaris)

## Prinsip
- NIK tidak pernah disimpan plaintext (hanya 4 digit terakhir).
- Status identitas: UNVERIFIED → PENDING → VERIFIED / INCONSISTENT.
- Duplikat tidak pernah dihapus otomatis — selalu via pratinjau gabung + snapshot + audit.

## Alur
1. **Daftar** (`/clients`): tambah perorangan/badan, opsional NIK (terverifikasi manual dari KTP fisik).
2. **Cari & saring** (`/clients/search/adv`): nama, kota, status identitas, tanggal, urut.
3. **Detail** (`/clients/{id}`): statistik, catatan kronologis, penanda inkonsisten, tautan berkas/permohonan.
4. **Duplikat** (`/clients/duplicates`): skor kemiripan nama (Jaro-Winkler) + NIK-last4 sama.
5. **Gabung** (`/clients/merge/preview` → `POST /clients/merge`): pilih kanonis, arsipkan yang kalah, pindahkan berkas/catatan, snapshot JSON + audit.
6. **Ekspor** (`/clients/export`): CSV untuk arsip offline.

## Korporat (per permohonan)
- Data perusahaan, pemegang saham (taut ke klien), pengurus, BO+TPPU/PEP.
- Legalitas (`corp/legal`): nomor akta, tanggal, SK AHU, NIB/NPWP — **salinan manual dari dokumen fisik, bukan integrasi AHU/OSS**.
- Wizard korporat (`/requests/{id}/wizard`) memandu langkah + checklist berkas.

## Peran
- Staf (staff, doc_operator, notary, office_admin, super_admin): kelola klien.
- finance/auditor: baca + catatan; auditor read-only untuk data sensitif.
- client (portal): hanya data sendiri via `/portal`.
