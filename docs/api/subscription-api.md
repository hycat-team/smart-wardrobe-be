# Subscription & Billing API Specs

Tài liệu thiết kế các API liên quan đến gói dịch vụ đăng ký thành viên (subscription), hạn mức sử dụng AI (quota AI), ví điện tử nội bộ (internal wallet) và quy trình thanh toán. Tất cả các giá trị hằng số sử dụng tham chiếu tại [constants/subscription.md](constants/subscription.md).

---

## Flow 1: Xem gói subscription và quản lý hạn mức AI (Quota)

Người dùng tham khảo các gói dịch vụ Premium hiện có và theo dõi mức độ tiêu thụ lượng gọi phối đồ AI của cá nhân trong ngày.

### 1. Lấy danh sách các gói Premium hiện có trên Closy
*   **Endpoint:** `GET /api/v1/subscriptions/plans`
*   **Tác nhân (Actor):** Khách hàng / Khách vãng lai (Public/Customer).
*   **Đối tượng ảnh hưởng:** Đọc danh sách gói dịch vụ `subscription_plans`.
*   **Mô tả:** Trả về danh sách các gói cước đang được cấu hình hoạt động trên hệ thống. Loại gói cước `planKind` tham chiếu tại [constants/subscription.md:PlanKind](constants/subscription.md#3-phan-loai-goi-cuoc-plankind).
*   **Response:**
    *   `200 OK`: Trả về danh sách các gói cước.

### 2. Xem tổng quan tình trạng gói cước của cá nhân
*   **Endpoint:** `GET /api/v1/subscriptions/me`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc bản ghi gói cước đăng ký đang hoạt động của người dùng hiện tại (current user).
*   **Mô tả:** Trả về thông tin chi tiết gói cước người dùng đang sử dụng, thời hạn hết hạn gói, trạng thái tự động gia hạn (auto-renew) và các thông số giới hạn quota kèm theo.
*   **Response:**
    *   `200 OK`: Trả về thông tin tổng quan gói cước.

### 3. Lấy thông tin hạn mức gọi AI hàng ngày
*   **Endpoint:** `GET /api/v1/subscriptions/me/daily-quota`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc hạn mức sử dụng AI hàng ngày của current user.
*   **Mô tả:** Trả về giới hạn số lần gọi AI gợi ý phối đồ tối đa được cấp trong ngày và lượng đã tiêu dùng thực tế của người dùng tính đến thời điểm hiện tại.
*   **Response:**
    *   `200 OK`: Trả về thông tin quota trong ngày.

### 4. Bật hoặc tắt tính năng tự động gia hạn gói cước
*   **Endpoint:** `PUT /api/v1/subscriptions/me/auto-renew`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái tự động gia hạn của bản ghi đăng ký gói của current user.
*   **Mô tả:** Thiết lập bật hoặc tắt cơ chế tự động trừ tiền từ ví nội bộ để duy trì gói Premium khi đến ngày hết hạn chu kỳ.
*   **Response:**
    *   `200 OK`: Trả về thông tin đăng ký gói sau cập nhật.

---

## Flow 2: Quản lý số dư và biến động ví nội bộ

Người dùng nạp tiền vào ví cá nhân và theo dõi lịch sử biến động số dư.

### 1. Lấy thông tin số dư ví hiện tại
*   **Endpoint:** `GET /api/v1/subscriptions/me/wallet`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc số dư tài khoản ví của current user.
*   **Mô tả:** Trả về số dư khả dụng hiện tại trong ví nội bộ của khách hàng.
*   **Response:**
    *   `200 OK`: Trả về thông tin ví.

### 2. Xem lịch sử biến động số dư ví
*   **Endpoint:** `GET /api/v1/subscriptions/me/wallet/statements`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc lịch sử biến động ví `wallet_statements`.
*   **Mô tả:** Trả về danh sách chi tiết các lần cộng/trừ tiền trong ví. Loại biến động ví `statementType` tham chiếu tại [constants/subscription.md:WalletStatementType](constants/subscription.md#8-phan-loai-bien-dong-vi-walletstatementtype).
*   **Response:**
    *   `200 OK`: Trả về danh sách lịch sử biến động.

### 3. Tạo yêu cầu nạp tiền vào ví cá nhân
*   **Endpoint:** `POST /api/v1/subscriptions/me/wallet/topup`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi giao dịch nạp tiền `deposit_transactions`.
*   **Mô tả:** Khởi tạo yêu cầu nạp tiền bằng cổng thanh toán PayOS/VietQR để cộng số dư ví. Trạng thái giao dịch nạp tiền `depositStatus` và phân loại giao dịch nạp tiền `depositTransactionType` tham chiếu chi tiết tại [constants/subscription.md](constants/subscription.md).
*   **Request Body:**
    ```json
    {
      "amount": 100000
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin mã giao dịch nạp tiền kèm theo liên kết chuyển hướng thanh toán (payment link).

---

## Flow 3: Quy trình mua gói subscription Premium

### 1. Thực hiện mua gói Premium trực tiếp bằng tài khoản ngân hàng (qua PayOS)
*   **Endpoint:** `POST /api/v1/subscriptions/me/purchase`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi thanh toán giao dịch cho yêu cầu mua gói Premium trực tiếp.
*   **Mô tả:** Tạo liên kết thanh toán PayOS/VietQR để người dùng chuyển khoản trực tiếp mua gói Premium mà không cần thông qua bước nạp ví nội bộ.
*   **Request Body:**
    ```json
    {
      "planId": "38a68d7f-5f47-46c4-8ad4-b98f28130222"
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về liên kết thanh toán trực tiếp và mã giao dịch.

### 2. Thực hiện mua gói Premium sử dụng số dư ví nội bộ
*   **Endpoint:** `POST /api/v1/subscriptions/me/purchase-with-wallet`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Trừ số dư tiền trong ví cá nhân, tạo bản ghi biến động ví (wallet statement) và kích hoạt trực tiếp gói cước đăng ký mới cho người dùng.
*   **Mô tả:** Thực hiện trừ tiền từ số dư khả dụng trong ví nội bộ để nâng cấp tài khoản lên gói Premium trực tiếp.
*   **Request Body:**
    ```json
    {
      "planId": "38a68d7f-5f47-46c4-8ad4-b98f28130222"
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin gói đăng ký vừa mua thành công.

---

## Flow 4: Nhận thông báo trạng thái thanh toán (Payment webhook)

### 1. Cổng thanh toán PayOS gửi thông báo trạng thái giao dịch (Webhook)
*   **Endpoint:** `POST /api/v1/subscriptions/payos-webhook`
*   **Tác nhân (Actor):** Cổng thanh toán trung gian PayOS.
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái các giao dịch nạp tiền/mua gói tương ứng, cộng số dư ví hoặc kích hoạt gói Premium cho người dùng.
*   **Mô tả:** Điểm nhận thông báo IPN tự động từ cổng PayOS khi khách hàng chuyển khoản thành công. Phía backend thực hiện xác thực chữ ký (validate signature) bảo mật và đảm bảo tính idempotent của giao dịch.
*   **Response:**
    *   `200 OK`: Ghi nhận xử lý webhook thành công.
