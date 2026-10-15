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

## 3. Nhà cung cấp danh tính (AuthProvider)
*   **Đường dẫn package:** `services/account/internal/modules/identity/domain/constants/authprovider`
*   **Các giá trị hợp lệ:**
    *   `local`: Tài khoản đăng ký bằng email + mật khẩu (xác thực qua OTP).
    *   `google`: Tài khoản tạo/liên kết qua Google (định danh theo Google `sub`).

## 4. Loại hoạt động người dùng (UserActivityType)
*   **Đường dẫn package:** `services/account/internal/modules/identity/domain/constants/useractivitytype`
*   **Các giá trị hợp lệ:**
    *   `login`: Phiên được cấp qua đăng nhập mật khẩu hoặc Google.
    *   `refresh`: Phiên được gia hạn khi xoay refresh token.
