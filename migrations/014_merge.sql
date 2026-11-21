-- 014_merge.sql — izinkan relasi antar-berkas (hasil gabung duplikat DIGABUNG_KE).
ALTER TABLE doc_relations MODIFY related_type ENUM('client','request','deed','repertorium','will','legalization','invoice','service','company','file') NOT NULL;
