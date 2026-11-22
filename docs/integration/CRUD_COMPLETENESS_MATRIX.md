# CRUD Completeness Matrix
Entitas bisnis:
| Entitas | C | R | U | D/alternatif | Audit | Catatan |
|---|---|---|---|---|---|---|
| Client | wizard 3 langkah + consent | list/detail/filter | edit profile | merge→ARCHIVED (never hard delete) | client.* | duplikat dicegah + snapshot |
| Identity | add | list+history | status verified/rejected | flag inconsistent | identity.* | NIK last4 only |
| Request | create | list/detail | move status | cancel via move | request.* | assignee notify |
| Deed | draft | list/detail | version (revisi) | final terkunci | deed.* | approve gate + repertorium |
| DMS file | upload | library/detail | meta/version/workflow | legal hold/disposition (record only) | doc.* | bytes never overwritten |
| Invoice | create | list/receipt | payment status | VOID bercatat (020) | invoice.* | no silent delete |
| Task | create (recurring) | list | done→next occurrence | done | task.recur | ref link |
| CMS pages/articles | create | public/publish | update + versi | soft-delete/archive | cms.* | rollback tersedia |
| Mandat | create | list | — | cabut | mandate.create | gate approve |
| LC register | (via docs/020 tables) | — | — | — | lcc.* | evidence-based |

Import: bank CSV + media upload + CMS export/import ZIP. Export: mask NIK + audit.
