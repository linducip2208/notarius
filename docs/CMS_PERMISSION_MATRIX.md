# CMS Permission Matrix — siapa boleh apa (ditegakkan server-side)

Legenda: V=view, C=create, U=update, D=delete, P=publish. ` website.view` = masuk area CMS.

| Izin | super_admin | office_admin | notary | substitute | staff | finance | doc_operator | auditor | client |
|---|---|---|---|---|---|---|---|---|---|
| website.view | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| website.manage (setup) | ✓ | ✓ | — | — | — | — | — | — | — |
| pages.view | ✓ | ✓ | ✓ | ✓ | ✓ | — | — | — | — |
| pages.create/update | ✓ | ✓ | ✓ | ✓ | ✓(no publish) | — | — | — | — |
| pages.delete | ✓ | ✓ | — | — | — | — | — | — | — |
| pages.publish | ✓ | ✓ | ✓ | ✓ | — | — | — | — | — |
| articles.manage | ✓ | ✓ | ✓ | ✓ | ✓* | — | — | — | — |
| navigation.manage | ✓ | ✓ | — | — | — | — | — | — | — |
| media.manage | ✓ | ✓ | ✓ | ✓ | ✓ | — | — | — | — |
| seo.manage | ✓ | ✓ | — | — | — | — | — | — | — |
| branding.manage | ✓ | ✓ | — | — | — | — | — | — | — |
| domain.manage | ✓ | ✓ | — | — | — | — | — | — | — |
| audit.view (riwayat) | ✓ | ✓ | — | — | — | — | — | ✓ | — |

Bukti: `internal/rbac/rbac_test.go::TestWebsitePermissions` (unit) +
`scripts/qa_shots/matrix.py` 28 rute (E2E, bukti `docs/qa/PERMISSION_MATRIX.md`).
Catatan: staff boleh buat/update tapi publish dikunci (naik ke REVIEW);
*artikel staff tanpa izin publish otomatis jadi REVIEW (terbit oleh notary+).
auditor baca + audit trail saja; klien tidak masuk admin CMS sama sekali.
