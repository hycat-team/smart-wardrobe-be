# Backup & Restore

Ensure the data safety of the PostgreSQL database.

## 1. Automated Backup
*   A cronjob runs daily at 2:00 AM on the VPS to perform a database dump:
    ```bash
    pg_dump -U [DB_USER] -h localhost [DB_NAME] > /backups/db_$(date +%F).sql
    ```
*   Compress and push the backup file to secure cloud storage.

## 2. Restore
To restore data from a backup:
```bash
psql -U [DB_USER] -d [DB_NAME] -f /backups/db_backup_file.sql
```
