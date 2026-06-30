# Phân quyền và Vai trò (Auth & Roles)

Hệ thống phân quyền theo mô hình RBAC (Role-Based Access Control) cho cả người dùng cá nhân lẫn nhân viên doanh nghiệp.

## 1. Người dùng cá nhân (B2C Roles)

- **Guest (Khách)**: Chưa đăng nhập, chỉ được xem danh sách gói cước hoặc trang công khai.
- **Standard User (Hội viên thường)**: Giới hạn tủ đồ (tối đa 50 món) và giới hạn quota AI hàng ngày.
- **Premium User (Hội viên trả phí)**: Không giới hạn tủ đồ và mở rộng hạn mức sử dụng AI phối đồ.

## 2. Người dùng doanh nghiệp (B2B Roles)

- **Brand Owner**: Toàn quyền cấu hình Brand Portal, quản trị thanh toán dịch vụ.
- **Brand Staff**: Nhân viên của brand.

## 3. Hệ thống quản trị viên (System Admin)

- Quản trị viên hệ thống có toàn quyền xem xét trạng thái các thương hiệu, phê duyệt đăng ký đối tác mới và xử lý sự cố.
