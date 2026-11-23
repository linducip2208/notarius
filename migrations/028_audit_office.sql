-- 028_audit_office.sql — kolom office_id di audit_logs + indeks per kantor (aditif, tanpa ubah data lama).
-- Gaya 016-026: pernyataan polos, idempoten. SENGAJA tanpa IF NOT EXISTS —
-- MySQL 8 tidak mendukungnya pada ADD COLUMN / CREATE INDEX (syntax error =
-- boot ditolak). Idempoten dijamin runner: ALTER/CREATE INDEX ulang gagal
-- dengan 1060/1061/1068 yang ditoleransi. Kolom NULL = baris lama/sistem
-- (login gagal/kunci pre-auth, cron) yang kantornya tak diketahui.

-- 1) Kantor pemilik peristiwa audit (NULL = tak diketahui; contoh kolom
-- office: office_flags memakai office_id sebagai PRIMARY KEY per kantor).
ALTER TABLE audit_logs ADD COLUMN office_id BIGINT UNSIGNED NULL;

-- 2) Filter audit per kantor (daftar/rekap audit lintas kantor).
CREATE INDEX idx_audit_office ON audit_logs (office_id);
