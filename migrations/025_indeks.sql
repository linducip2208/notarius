-- 025_indeks.sql — indeks susulan untuk query panas per kantor (tanpa ubah kolom/tabel, tanpa seed, tanpa drop).
-- Gaya 016-024: pernyataan polos, idempoten. Hanya CREATE INDEX untuk filter panas
-- yang belum terindeks; kolom/tabel disilang-cek ke CREATE TABLE masing-masing.
-- Catatan kompatibilitas: SENGAJA tanpa IF NOT EXISTS — MySQL 8 tidak mendukungnya
-- pada CREATE INDEX (syntax error = boot ditolak). Idempoten dijamin runner:
-- CREATE INDEX ulang gagal dengan 1061 duplicate key yang ditoleransi.

-- 1) Daftar permohonan per kantor+status (002: service_requests punya
-- idx_req_status(status) dan idx_req_client(client_id) saja; belum ada
-- komposit office_id+status).
CREATE INDEX idx_req_office_status ON service_requests (office_id, status);

-- 2) Draf akta per kantor + join request→draf (003: deed_drafts punya
-- uq_deed_number(office_id, number) dan idx_deed_status(status) saja;
-- kolom request_id belum terindeks).
CREATE INDEX idx_deed_request ON deed_drafts (request_id);
CREATE INDEX idx_deed_office_status ON deed_drafts (office_id, status);

-- 3) Checklist per request+status (011: doc_checklist_records hanya punya
-- uq_checkrec(request_id, item_id); filter status per request belum terindeks).
CREATE INDEX idx_checkrec_req_status ON doc_checklist_records (request_id, status);

-- 4) PPAT per kantor+status (021: ppat_register hanya punya
-- uq_ppat(office_id, akta_no); filter status per kantor belum terindeks).
CREATE INDEX idx_ppat_office_status ON ppat_register (office_id, status);
