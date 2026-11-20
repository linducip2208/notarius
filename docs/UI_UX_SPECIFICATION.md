# UI_UX_SPECIFICATION (Tabler)
Keputusan: **Tabler** menggantikan Metronic (instruksi user 2026-11-20; DECISIONS.md D-01).
Tabler MIT; dimuat via CDN jsDelivr (`@tabler/core@1`, htmx 1.9.12) — tidak ada aset
vendor di repo, tidak ada masalah lisensi. Bila kelak di-vendor-kan, taruh di `web/assets/`
dengan salinan lisensi MIT Tabler.

Identitas: permukaan putih bersih, aksen navy/blue Tabler, sidebar vertikal, kartu
`row-deck`, badge status, tabel responsif, empty-state jujur ("Belum ada …" bukan angka
palsu). 17 halaman (§Q) semuanya ter-render server via `internal/app/templates/`.
Interaksi HTMX: pencarian klien live (`hx-get /clients`, partial `client_rows`).
Responsif + aksesibel: label form, kontras Tabler default, tanpa menu mati.
