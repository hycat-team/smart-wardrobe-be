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
*   **Response:** `200 OK` — `UploadSignatureResult`:

    | Field | JSON | Kiểu | Ý nghĩa |
    |-------|------|------|---------|
    | Signature | `signature` | string | Chữ ký Cloudinary |
    | Timestamp | `timestamp` | int64 | Thời điểm ký |
    | ApiKey | `apiKey` | string | API key Cloudinary |
    | PublicID | `publicId` | string | Public ID đích (bằng `userId`) |
    | Folder | `folder` | string | Thư mục lưu ảnh đại diện |
    | ResourceType | `resourceType` | string | `image` |
    | AllowedFormats | `allowedFormats` | string | Vắng — endpoint này không giới hạn định dạng |

> **`publicId` là BẮT BUỘC gửi lại (endpoint duy nhất trong hệ thống có `publicId`):** chữ ký
> đã bao gồm `public_id` và `overwrite`, và Cloudinary **tự tính lại chữ ký từ các field client
> thực sự gửi**. Bỏ `public_id` khỏi form (hoặc đổi giá trị) → **upload bị từ chối**.
> Mục đích: ảnh đại diện luôn ghi đè ảnh cũ thay vì tạo asset mới.
>
> **Form upload = các field đã ký + `file` + `api_key` + `signature`.** Endpoint này ký
> 4 field (`timestamp`, `folder`, `public_id`, `overwrite`) nên form gửi **đúng 7 field**:
>
> | Field | Giá trị | Bắt buộc |
> |---|---|---|
> | `file` | file ảnh | Có |
> | `api_key` | `apiKey` từ response | Có |
> | `timestamp` | `timestamp` từ response | Có |
> | `signature` | `signature` từ response | Có |
> | `folder` | `folder` từ response | Có |
> | `public_id` | `publicId` từ response | Có |
> | `overwrite` | `true` | Có |
>
> Thêm field nào ngoài danh sách trên (kể cả `eager`, `transformation`, `tags`) hoặc bỏ field
> nào đều làm lệch chữ ký → upload bị từ chối.
>
> Các endpoint signature khác (wardrobe item, outfit, brand item, brand logo, community post)
> **không** trả `publicId` và **không** được gửi `public_id`/`overwrite` lên Cloudinary.

### 6. Cập nhật ảnh đại diện (avatar)
*   **Endpoint:** `PUT /api/v1/me/avatar`
*   **Tác nhân (Actor):** Người dùng đã đăng nhập.
*   **Đối tượng ảnh hưởng:** Cập nhật ảnh đại diện của current user.
*   **Mô tả:** Cập nhật thông tin `avatarUrl` và `avatarPublicId`.
