# Hằng số Nghiệp vụ Identity & Người dùng (Identity Constants)

Các hằng số dùng trong APIs liên quan đến Phân quyền, Xác thực và Người dùng:

## 1. Vai trò của tài khoản (RoleSlug)
*   **Đường dẫn package:** `internal/shared/domain/constants/identity/roleslug`
*   **Các giá trị hợp lệ:**
    *   `admin`: Quản trị viên tối cao của hệ thống Closy.
    *   `user`: Người dùng cá nhân thông thường.

## 2. Trạng thái tài khoản (UserStatus)
*   **Đường dẫn package:** `internal/shared/domain/constants/identity/userstatus`
*   **Các giá trị hợp lệ:**
    *   `active`: Tài khoản đang hoạt động bình thường.
    *   `inactive`: Tài khoản chưa được kích hoạt qua OTP.
    *   `suspended`: Tài khoản bị khóa do vi phạm điều khoản sử dụng.
