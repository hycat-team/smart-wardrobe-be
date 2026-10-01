# Auth API Specs

Tài liệu này mô tả API Auth thuộc module Identity hiện tại sau khi pivot B2B2C. MVP không sử dụng phone-first identity: đăng ký/đăng nhập vẫn thực hiện theo email/username và mã OTP gửi qua email; việc claim quyền lợi loyalty offline nằm trong Brand module.

---

## Flow 1: Đăng ký và xác thực tài khoản

### 1. Đăng ký tài khoản
*   **Endpoint:** `POST /api/v1/auth/register`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi người dùng (`users`) mới ở trạng thái chờ xác thực.
*   **Mô tả:** Tạo tài khoản mới và gửi mã OTP xác thực qua email. Quyền hạn (Role) và trạng thái người dùng (user status) tham chiếu [constants.md](constants.md).
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
*   **Lỗi thường gặp:**
    | HTTP | Message | Nguyên nhân |
    |---|---|---|
    | 400 | `Sai tài khoản hoặc mật khẩu.` | Sai tên tài khoản/email, hoặc sai mật khẩu |
    | 400 | `Tài khoản của bạn chưa có mật khẩu. Nếu bạn đã đăng ký bằng Google, vui lòng đăng nhập bằng Google. Nếu muốn đăng nhập bằng mật khẩu, hãy sử dụng chức năng Quên mật khẩu để thiết lập mật khẩu mới.` | Tài khoản chưa từng thiết lập mật khẩu (đăng ký bằng Google, hoặc do quản trị viên tạo). Frontend hiển thị nguyên message này, kèm lối tắt tới luồng Quên mật khẩu / đăng nhập Google |
    | 403 | `Tài khoản đã bị khoá hoặc vô hiệu hoá. Vui lòng liên hệ CSKH.` | Tài khoản không còn Active hoặc bị đưa vào blacklist |

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

## Flow 4: Đăng nhập Google

Hướng dẫn tích hợp frontend (web + mobile): [google-login-frontend-guide.md](google-login-frontend-guide.md).
Hằng số nhà cung cấp danh tính: [constants.md](constants.md) §3.

### 1. Đăng nhập bằng Google ID token (mobile / web tuỳ chọn)
*   **Endpoint:** `POST /api/v1/auth/google`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo tài khoản mới (nếu chưa có) hoặc liên kết/tạo phiên cho tài khoản hiện hữu.
*   **Mô tả:** Xác thực Google ID token, resolve tài khoản theo định danh nhà cung cấp (`sub`) và email đã xác thực, sau đó cấp phiên (access/refresh token).

### 2. Bắt đầu đăng nhập Google trên web (redirect)
*   **Endpoint:** `GET /api/v1/auth/google?redirectUrl=<URL>`
*   **Tác nhân (Actor):** Khách hàng (Customer) trên trình duyệt.
*   **Đối tượng ảnh hưởng:** Tạo `state` một lần lưu server-side.
*   **Mô tả:** Validate `redirectUrl` (scheme + host chính xác theo `server.front_end_origins`) rồi chuyển hướng (302) sang Google.

### 3. Callback Google (redirect)
*   **Endpoint:** `GET /api/v1/auth/google/callback`
*   **Tác nhân (Actor):** Google gọi về backend.
*   **Đối tượng ảnh hưởng:** Tạo phiên đăng nhập (cookie) hoặc chuyển hướng kèm lỗi.
*   **Mô tả:** Đổi authorization code, xác minh danh tính, tạo phiên và chuyển hướng về `redirectUrl`. Lỗi sau khi đã có đích hợp lệ trả `302 ...?error=<code>`; lỗi validate đích trả JSON 400.

---

## B2B2C Identity Rules

*   Hệ thống Auth không tự ý tạo tài khoản user từ số điện thoại offline.
*   Giao dịch mua sắm offline của nhãn hàng (Brand offline purchase) chỉ tạo hoặc truy vấn hồ sơ khách hàng nhãn hàng (`brand_customers`) trong phạm vi Brand module.
*   Quy trình Claim quyền lợi loyalty offline bắt buộc sử dụng endpoint `POST /api/v1/brands/claim`, không được sử dụng các endpoint auth thông thường.
*   Chính sách ẩn danh email khi tài khoản bị xoá (giải phóng email cho người dùng mới) sẽ áp dụng khi luồng xoá tài khoản tự phục vụ được thêm; hiện chưa có luồng này.
