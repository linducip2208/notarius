# ACCESSIBILITY AUDIT — target WCAG 2.2 AA (relevan)

- Keyboard: semua aksi <a>/<button>/<form>; dropdown & tab pakai kontrol Tabler bawaan (fokus terlihat).
- Label: tiap input punya `<label>` atau placeholder deskriptif + `name`; gambar punya `alt`; grafik punya `role=img` + `aria-label`.
- Kontras: palet Tabler default (teks #17233C di #FFFFFF ≈ 14:1; sekunder #69758B ≈ 5:1).
- Gerak: `prefers-reduced-motion` menonaktifkan animasi (tokens.css).
- Fokus modal: dialog Tabler `data-bs-toggle` mengunci fokus bawaan.
- Uji pembaca layar & navigasi keyboard penuh: BELUM (tanpa sesi audit manual) — dicatat sebagai utang.
