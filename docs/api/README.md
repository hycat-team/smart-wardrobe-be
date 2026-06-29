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
*   API đối tác thương hiệu: `/api/v1/brand-portal/...` (đối với Brand Portal) hoặc `/api/v1/brands/...` (đối với phía User tương tác với Brand)

## 4. Quản lý Hằng số (Constants)
*   Mọi API có sử dụng các giá trị hằng số (Status, Type, Role, v.v.) trong Request Body hoặc Response Body phải được đặc tả giá trị rõ ràng trong tài liệu hằng số tương ứng.
*   Bắt buộc phải dẫn liên kết (link) đến tài liệu hằng số tại thuộc tính đó để Frontend có thể đối chiếu giá trị hợp lệ.

## 5. Phân nhóm theo Luồng Nghiệp vụ (User Flows)
*   Các API không đứng đơn lẻ mà phục vụ một chuỗi nghiệp vụ (ví dụ: Quy trình tham gia thành viên và quy đổi đặc quyền) phải được nhóm lại theo từng Luồng sử dụng (User Flows).
*   Mỗi luồng cần thể hiện rõ thứ tự gọi các API và các bên liên quan (Actors).

## 6. Tính Hiện Hữu
*   Tài liệu API chỉ ghi nhận các API thực tế đã được triển khai (implemented) trong codebase. 
*   Các tính năng chưa phát triển (Future backlog) không được liệt kê để tránh gây nhầm lẫn khi tích hợp.
