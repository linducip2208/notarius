# Document Integrity
- Hash SHA-256 dari byte aktual saat upload/version.
- `files.sha256` + `doc_versions.sha256`; verifikasi on-demand stream + banding + 409 + REJECTED.
- Restore = AddVersion ref lama (tidak menimpa).
- Audit akses + metadata changes + confidentiality.
- Limitasi: hash membersihkan integritas, bukan keaslian tanda tangan.
