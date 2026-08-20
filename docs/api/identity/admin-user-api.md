# Admin User API Specs

Tài liệu này mô tả API Admin User thuộc module Identity - quản lý người dùng bởi quản trị viên.

---

## Flow 5: Admin quản lý người dùng

### 1. Lấy danh sách người dùng (users)
*   **Endpoint:** `GET /api/v1/admin/users`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc danh sách dữ liệu người dùng (`users`).
*   **Mô tả:** Lấy danh sách users hỗ trợ phân trang, tìm kiếm và lọc theo quyền hạn/trạng thái (role/status) theo đặc tả use case backend. Role/status tham chiếu [constants.md](constants.md).
*   **Response:**
    *   `200 OK`: Trả về danh sách user theo định dạng DTO admin.

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
*   **Mô tả:** Lấy thông tin chi tiết tài khoản người dùng và trạng thái gói cước.
*   **Response:**
    *   `200 OK`: Trả về thông tin chi tiết user.

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
