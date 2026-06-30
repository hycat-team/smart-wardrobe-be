# Quy chuẩn viết code (Coding Conventions)

Đảm bảo tính nhất quán và chất lượng nguồn mã của dự án.

## 1. Định dạng code (Formatting)

- Sử dụng lệnh định dạng chuẩn của Go:
    ```bash
    make fmt
    ```
- Trước khi commit code, hãy chạy kiểm tra định dạng và cấu trúc:
    ```bash
    make check
    ```

## 2. Xử lý lỗi (Error Handling)

- Luôn trả về lỗi ở cuối danh sách tham số trả về của hàm và kiểm tra lỗi ngay lập tức.
- Sử dụng cấu trúc lỗi định sẵn `apperror` để trả kết quả lỗi rõ ràng về cho front-end.

## 3. Tiếng Việt trong Swagger & API Response

- Theo quy định trong `.agentrules`, tất cả các chú thích `@Summary`, `@Description`, `@Param` của Swagger và toàn bộ phản hồi lỗi/thông điệp trả về client phải viết bằng **Tiếng Việt có dấu**.
- Các comment giải thích thuật toán/logic nội bộ viết bằng tiếng Anh.

## 4. Cấu trúc application, usecase và handler

- Các hàm mapper chuyển đổi giữa entity/domain model và DTO/responses phải đặt trong thư mục `application/mapper` của module tương ứng. Không để mapper trong file usecase.
- Trong `application/mapper`, được phép tách nhiều file mapper theo nhóm nghiệp vụ để tránh một file quá dài.
- Các helper chỉ phục vụ cho một usecase/file usecase phải được tách sang file riêng cùng package và tên file phải có hậu tố `_helper.go`.
- Không tạo helper nếu hàm chỉ bọc một biểu thức đơn giản và không làm rõ ý nghĩa nghiệp vụ. Ưu tiên inline hoặc dùng lại utility trong `pkg/utils`.
- Các message trả về ở presentation handler phải được khai báo thành biến/hằng số ở phần đầu file handler theo pattern hiện có, không hard-code trực tiếp trong từng response.
