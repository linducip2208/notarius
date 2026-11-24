-- 035_user_avatar.sql — foto profil pengguna (referensi ke files.id, nullable).
-- Tanpa FK keras agar hapus berkas tak memblokir; serve memverifikasi office.
ALTER TABLE users ADD COLUMN avatar_ref BIGINT UNSIGNED NULL;
