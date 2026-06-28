# Luồng Số hóa Tủ đồ (User Wardrobe Flow)

Mô tả luồng xử lý từ khi người dùng tải lên hình ảnh trang phục cho đến khi lưu trữ thành công vào tủ đồ ảo.

```mermaid
sequenceDiagram
    autonumber
    actor User as Người dùng
    participant App as Ứng dụng Mobile
    participant API as Backend API
    participant AI as AI Service
    database DB as Cơ sở dữ liệu

    User->>App: Chọn ảnh trang phục & bấm Upload
    App->>API: Gửi request upload ảnh (multipart)
    API->>API: Lưu trữ ảnh gốc lên Cloud Storage
    API-->>App: Trả về ItemID tạm thời (Trạng thái: Processing)
    App-->>User: Hiển thị trạng thái "Đang xử lý..."
    
    rect rgb(240, 240, 240)
        note over API, AI: Xử lý nền bất đồng bộ (Background job)
        API->>AI: Gửi ảnh yêu cầu phân tích tách nền & gắn tag
        AI-->>API: Trả về ảnh đã tách nền + nhãn thuộc tính (Màu sắc, loại áo...)
        API->>DB: Cập nhật thông tin Item & Chuyển trạng thái hoạt động (Active)
    end

    App->>API: Poll trạng thái hoặc nhận WebSocket notification
    API-->>App: Trả về thông tin Item đã xử lý hoàn tất
    App-->>User: Hiển thị trang phục đẹp mắt trong tủ đồ
```
