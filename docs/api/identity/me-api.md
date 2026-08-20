# Me API Specs

Tài liệu này mô tả API Me (Hồ sơ của tôi) thuộc module Identity.

---

## Flow 4: Hồ sơ của tôi (Hồ sơ của tôi)

### 1. Lấy thông tin người dùng hiện tại (current user)
*   **Endpoint:** `GET /api/v1/me`
*   **Tác nhân (Actor):** Người dùng đã đăng nhập.
*   **Đối tượng ảnh hưởng:** Đọc bảng dữ liệu người dùng (`users`).
*   **Mô tả:** Trả về thông tin chi tiết tài khoản hiện tại.

### 2. Cập nhật thông tin cá nhân
*   **Endpoint:** `PUT /api/v1/me`
*   **Tác nhân (Actor):** Người dùng đã đăng nhập.
*   **Đối tượng ảnh hưởng:** Cập nhật hồ sơ thông tin (profile) của current user.
*   **Mô tả:** Cập nhật các trường thông tin trong profile.

### 3. Cập nhật hồ sơ số đo cơ thể (body profile)
*   **Endpoint:** `PUT /api/v1/me/body-profile`
*   **Tác nhân (Actor):** Người dùng đã đăng nhập.
*   **Đối tượng ảnh hưởng:** Cập nhật hồ sơ số đo cơ thể.
*   **Mô tả:** Cập nhật số đo hình thể để phục vụ chức năng gợi ý phối đồ AI.

### 4. Đổi mật khẩu
*   **Endpoint:** `PUT /api/v1/me/change-password`
*   **Tác nhân (Actor):** Người dùng đã đăng nhập.
*   **Đối tượng ảnh hưởng:** Cập nhật mã hóa mật khẩu mới (password hash).
*   **Mô tả:** Thay đổi mật khẩu cho current user.

### 5. Lấy chữ ký tải lên ảnh đại diện (upload avatar signature)
*   **Endpoint:** `GET /api/v1/me/avatar-signature`
*   **Tác nhân (Actor):** Người dùng đã đăng nhập.
*   **Đối tượng ảnh hưởng:** Không thay đổi dữ liệu nghiệp vụ.
*   **Mô tả:** Lấy Cloudinary signature để client upload trực tiếp ảnh đại diện lên Cloudinary.

### 6. Cập nhật ảnh đại diện (avatar)
*   **Endpoint:** `PUT /api/v1/me/avatar`
*   **Tác nhân (Actor):** Người dùng đã đăng nhập.
*   **Đối tượng ảnh hưởng:** Cập nhật ảnh đại diện của current user.
*   **Mô tả:** Cập nhật thông tin `avatarUrl` và `avatarPublicId`.
