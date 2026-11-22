# Backup & DR
Backup DB via mysqldump --single-transaction + defaults-file 0600 (tanpa password di argv/ps), disimpan storage/backups, audit backup.now. File storage lokal/S3. Restore diuji sebelumnya: 0 diff, RTO<60s (docs/qa/RESTORE_TEST.md). Gap: backup file CMS/S3 perlu diuji terpisah; enkripsi backup di VPS (blocker).
