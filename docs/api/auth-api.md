# Auth & Me API Specs

Tài liệu này mô tả API identity hiện tại sau khi pivot B2B2C. MVP không sử dụng phone-first identity: đăng ký/đăng nhập vẫn thực hiện theo email/username và mã OTP gửi qua email; việc claim quyền lợi loyalty offline nằm trong Brand module.

---

## Flow 1: Đăng ký và xác thực tài khoản

### 1. Đăng ký tài khoản
*   **Endpoint:** `POST /api/v1/auth/register`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi người dùng (`users`) mới ở trạng thái chờ xác thực.
*   **Mô tả:** Tạo tài khoản mới và gửi mã OTP xác thực qua email. Quyền hạn (Role) và trạng thái người dùng (user status) tham chiếu [constants/identity.md](constants/identity.md).
*   **Response:** `201 Created`.

### 2. Xác thực OTP đăng ký
*   **Endpoint:** `POST /api/v1/auth/register/confirm-otp`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái người dùng (user status).
*   **Mô tả:** Xác nhận mã OTP gửi qua email để kích hoạt tài khoản.

### 3. Gửi lại OTP đăng ký
*   **Endpoint:** `POST /api/v1/auth/register/resend-otp`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo/gia hạn thời gian mã OTP đăng ký.
*   **Mô tả:** Gửi lại mã OTP đăng ký dựa trên email đã đăng ký trước đó.

---

## Flow 2: Đăng nhập và phiên làm việc

### 1. Đăng nhập
*   **Endpoint:** `POST /api/v1/auth/login`
*   **Tác nhân (Actor):** Khách hàng / Quản trị viên (User/Admin).
*   **Đối tượng ảnh hưởng:** Tạo phiên đăng nhập/token mới.
*   **Mô tả:** Đăng nhập bằng tên tài khoản (username) hoặc email kèm mật khẩu (password).

### 2. Refresh token
*   **Endpoint:** `POST /api/v1/auth/refresh-token`
*   **Tác nhân (Actor):** Người dùng có refresh token hợp lệ.
*   **Đối tượng ảnh hưởng:** Xoay vòng refresh token (refresh token rotation).
*   **Mô tả:** Sử dụng refresh token lưu trong cookie để cấp mã access token mới.

### 3. Đăng xuất
*   **Endpoint:** `POST /api/v1/auth/logout`
*   **Tác nhân (Actor):** Người dùng đã đăng nhập.
*   **Đối tượng ảnh hưởng:** Vô hiệu hóa token hiện tại.
*   **Mô tả:** Đăng xuất tài khoản và xóa bỏ cookie chứa token.

---

## Flow 3: Quên mật khẩu

### 1. Yêu cầu khôi phục mật khẩu
*   **Endpoint:** `POST /api/v1/auth/forgot-password`
*   **Tác nhân (Actor):** Người dùng (User).
*   **Đối tượng ảnh hưởng:** Tạo mã OTP khôi phục mật khẩu.
*   **Mô tả:** Gửi mã OTP khôi phục mật khẩu qua email của người dùng.

### 2. Gửi lại OTP khôi phục mật khẩu
*   **Endpoint:** `POST /api/v1/auth/forgot-password/resend-otp`
*   **Tác nhân (Actor):** Người dùng (User).
*   **Đối tượng ảnh hưởng:** Tạo/gia hạn thời hạn mã OTP khôi phục.
*   **Mô tả:** Gửi lại mã OTP khôi phục mật khẩu qua email.

### 3. Xác thực OTP khôi phục mật khẩu
*   **Endpoint:** `POST /api/v1/auth/forgot-password/confirm-otp`
*   **Tác nhân (Actor):** Người dùng (User).
*   **Đối tượng ảnh hưởng:** Tạo token tạm thời để thực hiện reset password.
*   **Mô tả:** Xác thực mã OTP và lưu lại token tạm thời hợp lệ.

### 4. Đặt lại mật khẩu
*   **Endpoint:** `POST /api/v1/auth/reset-password`
*   **Tác nhân (Actor):** Người dùng (User).
*   **Đối tượng ảnh hưởng:** Cập nhật mã hóa mật khẩu mới (password hash).
*   **Mô tả:** Thiết lập mật khẩu mới thông qua token tạm thời.

---

## Flow 4: Hồ sơ của tôi (Ho so cua toi)

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

---

## B2B2C Identity Rules

*   Hệ thống Auth không tự ý tạo tài khoản user từ số điện thoại offline.
*   Giao dịch mua sắm offline của nhãn hàng (Brand offline purchase) chỉ tạo hoặc truy vấn hồ sơ khách hàng nhãn hàng (`brand_customers`) trong phạm vi Brand module.
*   Quy trình Claim quyền lợi loyalty offline bắt buộc sử dụng endpoint `POST /api/v1/brands/claim`, không được sử dụng các endpoint auth thông thường.

---

## Flow 5: Admin quản lý người dùng

### 1. Lấy danh sách người dùng (users)
*   **Endpoint:** `GET /api/v1/admin/users`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc danh sách dữ liệu người dùng (`users`).
*   **Mô tả:** Lấy danh sách users hỗ trợ phân trang, tìm kiếm và lọc theo quyền hạn/trạng thái (role/status) theo đặc tả use case backend. Role/status tham chiếu [constants/identity.md](constants/identity.md).
*   **Response:**
    *   `200 OK`: Trả về danh sách user theo định dạng DTO admin.

### 2. Cập nhật trạng thái người dùng
*   **Endpoint:** `PATCH /api/v1/admin/users/:id/status`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái tài khoản `users.status`, thu hồi (revoke) refresh token khi cần thiết.
*   **Mô tả:** Khóa hoặc mở khóa lại tài khoản của user. Token truy cập (Access token) hiện tại có thể vẫn có hiệu lực cho đến khi hết hạn TTL theo cơ chế JWT stateless. Trạng thái `status` tham chiếu [constants/identity.md:UserStatus](constants/identity.md#2-trang-thai-tai-khoan-userstatus).
*   **Response:**
    *   `200 OK`: Trả về thông tin user sau khi cập nhật.
