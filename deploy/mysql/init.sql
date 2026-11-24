-- deploy/mysql/init.sql — create database and least-privilege app user.
-- Termasuk DB drill restore (notaryos_restore): skrip restore.sh berjalan
-- dengan kredensial app, jadi kedua DB harus sudah ada + di-grant sejak awal.
CREATE DATABASE IF NOT EXISTS notaryos CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS notaryos_restore CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'notaryos'@'%' IDENTIFIED BY 'change-me-strong-password';
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, INDEX, REFERENCES
  ON notaryos.* TO 'notaryos'@'%';
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, INDEX, REFERENCES
  ON notaryos_restore.* TO 'notaryos'@'%';
FLUSH PRIVILEGES;
