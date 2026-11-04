# Admin User API Specs

Tài liệu này mô tả API Admin User thuộc module Identity - quản lý người dùng bởi quản trị viên.

---

## Flow 5: Admin quản lý người dùng

### 1. Lấy danh sách người dùng (users)
*   **Endpoint:** `GET /api/v1/admin/users`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc danh sách dữ liệu người dùng (`users`).
*   **Mô tả:** Lấy danh sách users hỗ trợ phân trang, tìm kiếm và lọc theo quyền hạn/trạng thái (role/status) theo đặc tả use case backend. Role/status tham chiếu [constants.md](constants.md).
*   **Query Parameters:**
    *   `q` (string, optional): Tìm tương đối theo username, email, họ, tên.
    *   `userId` (string, optional): Mã người dùng (UUID) — lọc chính xác, dùng index khoá chính. Khi có `userId` thì `q` bị bỏ qua. UUID sai định dạng trả `400`.
    *   `roleSlug` (string, optional): Vai trò (`user`, `admin`).
    *   `isActive` (boolean, optional): `true` = đang hoạt động, `false` = bị khoá.
    *   `page`, `limit` (integer, optional): Phân trang.
*   **Response:**
    *   `200 OK`: Trả về danh sách user theo định dạng DTO admin. Mỗi phần tử mang `subscription` (gói hiệu lực: `planName`, `planSlug`, `expiresAt`, hạn mức, quota AI) để màn hình quản trị hiển thị gói ngay trong bảng.

### 2. Tạo tài khoản người dùng
*   **Endpoint:** `POST /api/v1/admin/users`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi người dùng (`users`) mới (mặc định vai trò user).
*   **Mô tả:** Cho phép quản trị viên tạo tài khoản người dùng mới.
*   **Response:**
    *   `200 OK`: Trả về thông tin user vừa tạo.

### 3. Chi tiết tài khoản người dùng
*   **Endpoint:** `GET /api/v1/admin/users/{id}`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc thông tin chi tiết người dùng.
*   **Mô tả:** Lấy thông tin chi tiết tài khoản người dùng. Gói cước hiệu lực nằm trong `subscription` của chính user — cùng shape với từng phần tử của danh sách người dùng (`GET /admin/users`), nên chỉ có một nguồn dữ liệu gói.
    *   `subscription.aiOutfitDailyQuota` / `subscription.aiChatDailyQuota` là **hạn mức/ngày của gói** (số gốc), không phải số lượt còn lại hôm nay. Số lượt còn lại (nếu cần) đọc ở API quota của chính người dùng: `GET /api/v1/subscriptions/me/daily-quota` (`outfitRecommendCount`, `aiUsageCount`, `lastResetDate`).
*   **Response:**
    *   `200 OK`: Trả về `UserRes` (kèm `subscription`).

### 4. Cập nhật trạng thái người dùng
*   **Endpoint:** `PATCH /api/v1/admin/users/:id/status`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái tài khoản `users.status`, thu hồi (revoke) refresh token khi cần thiết.
*   **Mô tả:** Khóa hoặc mở khóa lại tài khoản của user. Token truy cập (Access token) hiện tại có thể vẫn có hiệu lực cho đến khi hết hạn TTL theo cơ chế JWT stateless. Trạng thái `status` tham chiếu [constants.md](constants.md#2-trang-thai-tai-khoan-userstatus).
*   **Response:**
    *   `200 OK`: Trả về thông tin user sau khi cập nhật.

### 5. Đổi vai trò tài khoản
*   **Endpoint:** `PATCH /api/v1/admin/users/{id}/role`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật vai trò (role) của tài khoản người dùng.
*   **Mô tả:** Đổi vai trò (role) của tài khoản người dùng. Chỉ có tài khoản admin seed mới được quyền nâng cấp tài khoản lên admin. Vai trò tham chiếu [constants.md](constants.md).
*   **Response:**
    *   `200 OK`: Trả về thông tin user sau khi cập nhật.

### 6. Danh sách phiên đăng nhập
*   **Endpoint:** `GET /api/v1/admin/users/{id}/sessions`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc danh sách phiên làm việc của người dùng.
*   **Mô tả:** Lấy danh sách các phiên làm việc / thiết bị đăng nhập của người dùng.
*   **Response:**
    *   `200 OK`: Trả về danh sách phiên đăng nhập.

### 7. Thu hồi toàn bộ phiên đăng nhập
*   **Endpoint:** `POST /api/v1/admin/users/{id}/sessions/revoke`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Vô hiệu hóa tất cả các phiên làm việc hiện tại của người dùng.
*   **Mô tả:** Thu hồi và vô hiệu hóa tất cả các phiên làm việc hiện tại của người dùng.
*   **Response:**
    *   `200 OK`: Thu hồi thành công.
