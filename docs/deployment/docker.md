# Hướng dẫn Dockerize ứng dụng (Docker Guide)

Hệ thống được đóng gói thông qua Dockerfile sử dụng mô hình Multi-stage build để tối ưu hóa dung lượng image đầu ra.

## 1. Build image nội bộ (Local Build)
Chạy lệnh sau để build image production ở máy local:
```bash
make docker-build
```

## 2. Docker Compose
Hệ thống sử dụng các file docker-compose khác nhau tùy môi trường:
*   `docker-compose.yml`: Môi trường development (bao gồm Postgres, Redis, RabbitMQ local).
*   `docker-compose.prod.yml`: Môi trường production.
