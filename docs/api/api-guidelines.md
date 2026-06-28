# Quy chuẩn Thiết kế API (API Guidelines)

Tất cả các API được phát triển trong dự án Closy tuân thủ chuẩn RESTful:

## 1. Định dạng JSON
*   Mọi yêu cầu gửi lên và phản hồi trả về sử dụng định dạng JSON chuẩn.
*   Đặt tên thuộc tính theo quy tắc camelCase (ví dụ: `wardrobeItemId`).

## 2. Mã trạng thái HTTP (HTTP Status Codes)
*   `200 OK`: Yêu cầu xử lý thành công.
*   `201 Created`: Tạo thực thể mới thành công.
*   `400 Bad Request`: Lỗi định dạng dữ liệu đầu vào.
*   `401 Unauthorized`: Chưa đăng nhập hoặc token hết hạn.
*   `403 Forbidden`: Không có quyền truy cập tài nguyên.
*   `404 Not Found`: Đường dẫn hoặc thực thể không tồn tại.
*   `500 Internal Server Error`: Lỗi phát sinh từ hệ thống máy chủ.

## 3. Tiền tố đường dẫn (Endpoints Prefix)
*   API công khai/người dùng: `/api/v1/...`
*   API đối tác thương hiệu: `/api/v1/brand/...`
