# Hướng dẫn Triển khai VPS (VPS Deployment Guide)

Tài liệu hướng dẫn triển khai ứng dụng Closy lên máy chủ VPS.

## 1. Yêu cầu Hệ thống Máy chủ (Prerequisites)
*   Hệ điều hành: Ubuntu 22.04 LTS hoặc mới hơn.
*   Cài đặt sẵn: Docker Engine và Docker Compose.
*   Cấu hình Domain trỏ về IP của VPS (ví dụ: `api.[DOMAIN].com`).

## 2. Thư mục cài đặt trên VPS
Mã nguồn deployment và các file cấu hình được đặt tại thư mục:
```bash
/opt/closy
```

## 3. Khởi chạy ứng dụng
Chạy lệnh compose trên máy chủ VPS:
```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

## 4. Cấu hình Nginx & Tự động Gia hạn SSL (Certbot)
*   Sử dụng Certbot để xin cấp chứng chỉ SSL miễn phí từ Let's Encrypt.
*   **SSL Auto-Renew**: Thiết lập cronjob định kỳ hàng tuần để gia hạn chứng chỉ tự động:
    ```bash
    0 0 * * * certbot renew --post-hook "docker exec nginx-container nginx -s reload"
    ```
    *(Thay `nginx-container` bằng tên container Nginx thực tế của bạn).*
