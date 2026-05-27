# 🚀 SmartWardrobe Backend - Project Features Overview

Tài liệu này tổng hợp toàn bộ các tính năng (Features), thành phần kỹ thuật (Technical Components), và cơ chế bảo mật (Security Mechanisms) hiện có của hệ thống **SmartWardrobe Backend**. Hệ thống được xây dựng trên ngôn ngữ **Golang**, áp dụng mô hình kiến trúc **Modular Monolith** kết hợp **Clean Architecture** chuẩn chỉnh, hiệu năng cao và bảo mật nghiêm ngặt.

---

## 📌 1. Module Quản lý Định danh & Tài khoản (Identity Module)

Tập trung vào toàn bộ các quy trình nghiệp vụ liên quan đến tài khoản người dùng, phân quyền, xác thực và bảo mật thông tin cá nhân.

### A. Hệ thống Xác thực & Phân quyền (Authentication & Authorization)

| Tính năng | Phương thức & Endpoint | Chi tiết kỹ thuật & Cơ chế |
| :--- | :--- | :--- |
| **Đăng ký tài khoản** | `POST /api/v1/auth/register` | Người dùng nhập thông tin đăng ký. Hệ thống tự động tạo mã OTP ngẫu nhiên, lưu vào **Redis** và kích hoạt hàng đợi gửi email xác thực thông qua dịch vụ **Gmail SMTP**. |
| **Xác thực OTP Đăng ký** | `POST /api/v1/auth/register/confirm-otp` | Xác nhận mã OTP được gửi vào hòm thư. Hệ thống đối chiếu dữ liệu trong **Redis**, kích hoạt trạng thái tài khoản trong Postgres DB từ `Pending` sang `Active`. |
| **Đăng nhập hệ thống** | `POST /api/v1/auth/login` | Hỗ trợ đăng nhập bằng `Username` hoặc `Email` cùng `Password` (đã mã hóa bcrypt). Trả về Access Token và Refresh Token được đóng gói an toàn trong **Secure HttpOnly Cookies** với cấu hình nghiêm ngặt (`Strict SameSite`, `Secure`, `HttpOnly`). |
| **Đăng xuất hệ thống** | `POST /api/v1/auth/logout` | Vô hiệu hóa phiên đăng nhập hiện tại: Tiến hành đưa Access Token vào danh sách đen (**Blacklist**) trong **Redis** với thời gian hết hạn tương ứng, đồng thời xóa toàn bộ cookie lưu ở trình duyệt của client. |
| **Xoay vòng Token (Refresh)**| `POST /api/v1/auth/refresh-token` | Triển khai cơ chế **Refresh Token Rotation (RTR)**. Client gửi Refresh Token cũ qua Cookie để nhận về cặp Access & Refresh Token mới, ngăn ngừa tối đa rủi ro chiếm đoạt session. |
| **Yêu cầu Quên mật khẩu**| `POST /api/v1/auth/forgot-password` | Gửi yêu cầu khôi phục mật khẩu. Hệ thống tạo mã OTP xác nhận khôi phục mật khẩu gửi qua email và lưu trữ tạm thời tại **Redis**. |
| **Xác thực OTP Quên mật khẩu**| `POST /api/v1/auth/forgot-password/confirm-otp` | Xác thực mã OTP quên mật khẩu. Nếu hợp lệ, hệ thống trả về và lưu trữ một `forgot_password_token` tạm thời trong **Secure HttpOnly Cookie** phục vụ cho bước đổi mật khẩu tiếp theo. |
| **Đặt lại mật khẩu mới** | `POST /api/v1/auth/reset-password` | Nhận mật khẩu mới từ người dùng, kiểm tra tính hợp lệ của `forgot_password_token` từ cookie để tiến hành cập nhật mật khẩu mới vào cơ sở dữ liệu và xóa cookie tạm thời. |

### B. Quản lý Hồ sơ Người dùng (User Profile Management)

| Tính năng | Phương thức & Endpoint | Chi tiết kỹ thuật & Cơ chế |
| :--- | :--- | :--- |
| **Xem hồ sơ cá nhân** | `GET /api/v1/me` | Lấy chi tiết thông tin của người dùng đang đăng nhập (Họ tên, Email, Số điện thoại, Giới tính, Ngày sinh, Ảnh đại diện, Trạng thái, Vai trò). |
| **Cập nhật hồ sơ** | `PUT /api/v1/me` | Cho phép người dùng thay đổi các thông tin cá nhân cơ bản và cập nhật trực tiếp vào cơ sở dữ liệu Postgres. |
| **Đổi mật khẩu** | `PUT /api/v1/me/change-password` | Người dùng đã đăng nhập có thể thực hiện đổi mật khẩu bằng cách cung cấp mật khẩu cũ (để xác thực lại) và nhập mật khẩu mới. |

---

## 💳 2. Module Đăng ký Gói thành viên (Subscription Module)

Được thiết kế dưới dạng một Module độc lập với ranh giới rõ ràng trong kiến trúc Modular Monolith để chuẩn bị cho việc mở rộng các mô hình kinh doanh Premium của tủ đồ thông minh.

* **Subscription Plan Repository**: Cung cấp các phương thức truy xuất, quản lý các gói dịch vụ và các điều khoản đăng ký trong cơ sở dữ liệu.
* **Loose Coupling Integration**: Tương tác xuyên module được đảm bảo thông qua giao diện hợp đồng chung `ISubscriptionModuleContract` tại tầng Shared Application, loại bỏ hoàn toàn sự phụ thuộc trực tiếp (tight coupling) giữa Identity và Subscription.

---

## 🛠️ 3. Hạ tầng hệ thống & Cơ chế bổ trợ (System Infrastructure)

Các tính năng bổ trợ hệ thống được tích hợp dưới dạng Middleware hoặc Shared Services toàn cục nhằm tối ưu hóa hiệu năng, tính ổn định và tính mở rộng cao của ứng dụng:

### A. Cơ chế Middleware Động (Dynamic Middleware)
Hệ thống không fix cứng thông số mà tự động nạp cấu hình tối ưu từ tệp môi trường `.env` thông qua bộ quản lý cấu hình tập trung (`config.Config`):

* **Global Rate Limiting Middleware (`ratelimit.go`)**: 
  * Bảo vệ hệ thống khỏi các cuộc tấn công Brute-Force hoặc DDoS bằng thuật toán **Token Bucket** dựa trên hạ tầng **Redis**.
  * Cấu hình động linh hoạt thông qua `.env`: `RATE_LIMIT_TOKEN_LIMIT` (giới hạn token tối đa), `RATE_LIMIT_TOKENS_PER_PERIOD` (số lượng token hồi phục mỗi chu kỳ), và `RATE_LIMIT_REPLENISHMENT_SECONDS` (thời gian chu kỳ hồi phục).
* **Global Timeout Middleware (`timeout.go`)**:
  * Tự động ngắt các request xử lý quá lâu bằng context deadline nhằm giải phóng tài nguyên server.
  * Cấu hình linh hoạt thông qua biến môi trường `REQUEST_TIMEOUT_SECONDS`.

### B. Cơ chế Xử lý lỗi toàn cục & Bảo mật Token
* **Centralized Error Handling (`errorcode`)**:
  * Chuyển đổi toàn bộ các lỗi nghiệp vụ trong hệ thống thành chuẩn phản hồi JSON thống nhất chứa mã lỗi ứng dụng (`app_error_code`), HTTP Status tương ứng và thông điệp dịch nghĩa thân thiện với client.
* **Token Blacklisting Service (`security`)**:
  * Tích hợp **Redis Blacklist** cho các JWT Access Token đã đăng xuất hoặc bị thu hồi để đảm bảo chúng không thể tái sử dụng ngay cả khi chưa hết hạn.

### C. Tài liệu API Swagger UI Chuyên nghiệp (`/swagger`)
Hệ thống tích hợp công cụ tự động tạo tài liệu API Swagger UI được tinh chỉnh giao diện cao cấp:
* **Mặc định Theme Light**: Ép hiển thị mặc định chế độ sáng chuyên nghiệp (`Light Mode`) trên mọi thiết bị và hệ điều hành thông qua việc ghi đè thông minh phương thức `window.matchMedia` và các thẻ meta chuyên dụng.
* **Dropdown Chọn Phiên bản**: Khôi phục lại thanh chọn phiên bản tài liệu API gốc (dropdown selector) bằng cách cấu hình danh sách mảng `urls` động cho phép chuyển đổi mượt mà giữa các phiên bản API.

---

## 🏗️ 4. Kiến trúc Dependency Injection tối ưu (Google Wire)

Dự án áp dụng công nghệ Dependency Injection compile-time chuyên nghiệp từ Google (**Wire**):
* Mỗi Layer con trong từng module (`presentation`, `application`, `infrastructure`) sở hữu riêng một tệp `provider.go` để đóng gói toàn bộ các instance/service thuộc layer đó.
* Tệp `provider.go` tại thư mục gốc của Module đóng vai trò gom nhóm các `ProviderSet` của từng Layer thành một bộ duy nhất để xuất ra ngoài.
* `internal/di/wire.go` đóng vai trò kết nối toàn cục toàn bộ hệ thống giúp quá trình khởi động ứng dụng luôn nhất quán, tối ưu hiệu năng và phát hiện sớm các lỗi thiếu phụ thuộc ngay tại thời điểm biên dịch.
