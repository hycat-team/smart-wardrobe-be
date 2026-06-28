# Quy chuẩn viết code (Coding Conventions)

Đảm bảo tính nhất quán và chất lượng nguồn mã nguồn của dự án.

## 1. Định dạng code (Formatting)
*   Sử dụng lệnh định dạng chuẩn của Go:
    ```bash
    make fmt
    ```
*   Trước khi commit code, hãy chạy kiểm tra định dạng và cấu trúc:
    ```bash
    make check
    ```

## 2. Xử lý lỗi (Error Handling)
*   Luôn trả về lỗi ở cuối danh sách tham số trả về của hàm và kiểm tra lỗi ngay lập tức.
*   Sử dụng cấu trúc lỗi định sẵn `apperror` để trả kết quả lỗi rõ ràng về cho front-end.

## 3. Tiếng Việt trong Swagger & API Response
*   Theo quy định trong `.agentrules`, tất cả các chú thích `@Summary`, `@Description`, `@Param` của Swagger và toàn bộ phản hồi lỗi/thông điệp trả về client phải viết bằng **Tiếng Việt có dấu**.
*   Các comment giải thích thuật toán/logic nội bộ viết bằng tiếng Anh.
