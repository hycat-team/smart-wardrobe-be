# Kiến trúc Hướng Sự kiện (Event-Driven Design)

Kiến trúc hướng sự kiện giúp xử lý bất đồng bộ các tác vụ AI tiêu tốn nhiều thời gian và tài nguyên, đảm bảo API Gateway luôn phản hồi nhanh chóng.

## 1. Cơ chế hoạt động với RabbitMQ
*   Khi người dùng upload trang phục, API lưu thông tin tạm thời và đẩy một sự kiện `wardrobe.item.uploaded` vào hàng đợi RabbitMQ.
*   Worker tiêu thụ sự kiện (Event Consumer) nhận job và gửi ảnh qua AI API để xử lý tách nền và trích xuất nhãn.
*   Sau khi nhận kết quả từ AI, Worker cập nhật trạng thái Item thành `Active` và gửi thông báo qua WebSocket đến người dùng.

## 2. Ưu điểm kiến trúc
*   **Tránh timeout**: Client không phải chờ đợi phản hồi đồng bộ từ các mô hình AI chậm chạp.
*   **Đảm bảo độ tin cậy**: Hỗ trợ cơ chế retry và dead-letter queue (DLQ) khi gọi dịch vụ AI ngoài bị lỗi kết nối tạm thời.
