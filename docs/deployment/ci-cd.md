# Hướng dẫn thiết lập CI/CD (CI/CD Setup Guide)

Dự án sử dụng GitHub Actions làm hệ thống tích hợp và triển khai tự động.

## 1. Các Secrets cần cấu hình trên GitHub Repository
Để đảm bảo an toàn bảo mật, toàn bộ thông tin nhạy cảm của máy chủ VPS được lưu trữ dưới dạng GitHub Repository Secrets:

*   `SSH_PRIVATE_KEY`: Khóa SSH Private Key dùng để truy cập vào VPS.
*   `VPS_HOST`: Địa chỉ IP của máy chủ VPS (ví dụ: `[VPS_IP_PLACEHOLDER]`).
*   `VPS_USER`: Tên tài khoản SSH đăng nhập vào máy chủ (ví dụ: `root`).
*   `CLOUDINARY_URL`, `DATABASE_URL`, `RABBITMQ_URL`... và các khóa môi trường cấu hình ứng dụng.

## 2. Luồng CI/CD (Workflow)
*   **CI**: Tự động kích hoạt khi có PR vào nhánh `develop` hoặc `main`. Chạy kiểm tra fmt, tests và build thử.
*   **CD**: Tự động kích hoạt khi merge PR vào nhánh `main`. Build Docker Image, đẩy lên Docker Registry và kết nối SSH để deploy tự động trên VPS.
