-- deploy/mysql/init.sql — create database and least-privilege app user.
CREATE DATABASE IF NOT EXISTS notaryos CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'notaryos'@'%' IDENTIFIED BY 'change-me-strong-password';
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, INDEX, REFERENCES
  ON notaryos.* TO 'notaryos'@'%';
FLUSH PRIVILEGES;
