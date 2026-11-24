-- 034_ppat_bpn_date.sql — tanggal lapor BPN untuk pengingat resi H-7/H-3/H-lewat.
-- Gaya 016-032: ALTER polos (runner menoleransi DDL duplikat 1060 untuk rerun).
--
-- ppat_register.bpn_date DATE NULL: tanggal resi BPN dilaporkan (diisi saat
-- status maju ke LAPOR_BPN; default CURDATE bila form tidak mengisi tanggal).
-- Jatuh tempo resi = bpn_date + 14 hari; bpn_date NULL (lapor sebelum migrasi
-- ini) memakai DATE(created_at) sebagai fallback. Tanpa backfill: baris lama
-- tetap NULL dan ikut fallback di query, bukan ditebak di sini.
ALTER TABLE ppat_register
  ADD COLUMN bpn_date DATE NULL;
