# Giám sát Hệ thống (Monitoring)

Theo dõi sức khỏe và hiệu năng của hệ thống backend Closy.

## 1. Hệ thống Log (Logging)
*   Hệ thống sử dụng thư viện `zap` để ghi nhận các log thông tin và lỗi của ứng dụng.
*   Log được xuất ra console và lưu vào các file log xoay vòng định kỳ (log rotation) đặt tại thư mục log được cấu hình trong file `.env`.

## 2. API Health Check
*   Đường dẫn giám sát sức khỏe của API: `/api/v1/health`.
*   Cấu hình công cụ giám sát (Uptime Kuma hoặc tương đương) gửi thông báo về kênh Slack/Telegram của đội kỹ thuật nếu endpoint này không phản hồi quá 2 phút.
