# Sao lưu và Phục hồi Dữ liệu (Backup & Restore)

Đảm bảo an toàn dữ liệu cơ sở dữ liệu PostgreSQL.

## 1. Sao lưu tự động (Automated Backup)
*   Một cronjob chạy hàng ngày lúc 2:00 AM trên VPS thực hiện dump cơ sở dữ liệu:
    ```bash
    pg_dump -U [DB_USER] -h localhost [DB_NAME] > /backups/db_$(date +%F).sql
    ```
*   Nén và đẩy file backup lên lưu trữ đám mây an toàn.

## 2. Phục hồi dữ liệu (Restore)
Để khôi phục dữ liệu từ một bản backup:
```bash
psql -U [DB_USER] -d [DB_NAME] -f /backups/db_backup_file.sql
```
