# Billing API Specs

Tài liệu thiết kế các API liên quan đến ví điện tử nội bộ (internal wallet) và quy trình thanh toán mua gói subscription. Tất cả các giá trị hằng số sử dụng tham chiếu tại [constants.md](constants.md).

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
*   **Mô tả:** Trả về danh sách chi tiết các lần cộng/trừ tiền trong ví. Loại biến động ví `statementType` tham chiếu tại [constants.md:WalletStatementType](constants.md#8-phan-loai-bien-dong-vi-walletstatementtype).
*   **Response:**
    *   `200 OK`: Trả về danh sách lịch sử biến động.

### 3. Tạo yêu cầu nạp tiền vào ví cá nhân
*   **Endpoint:** `POST /api/v1/subscriptions/me/wallet/topup`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi giao dịch nạp tiền `deposit_transactions`.
*   **Mô tả:** Khởi tạo yêu cầu nạp tiền bằng cổng thanh toán PayOS/VietQR để cộng số dư ví. Trạng thái giao dịch nạp tiền `depositStatus` và phân loại giao dịch nạp tiền `depositTransactionType` tham chiếu chi tiết tại [constants.md](constants.md).
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
*   **Lưu ý về trạng thái:** Payload webhook của PayOS **không chứa trạng thái payment link** (chỉ có `orderCode`, `amount`, `reference`, `paymentLinkId`, `data.code`, `data.desc`). Vì vậy:
    *   Webhook thành công (`code = "00"`) được áp dụng trực tiếp.
    *   Webhook không thành công: backend gọi `GET /v2/payment-requests/{orderCode}` để lấy trạng thái thật và xử lý theo bảng dưới đây. Nếu gọi provider lỗi, backend trả lỗi để PayOS retry.
    *   Hủy thanh toán phía người dùng được PayOS trả về qua `cancelUrl` (không qua webhook); backend vẫn ghi nhận nhờ đối soát định kỳ gọi API lấy trạng thái link.

| Trạng thái PayOS | Trạng thái nội bộ |
| --- | --- |
| `PENDING`, `PROCESSING` | `pending` (chờ thanh toán) |
| `PAID` | `success` |
| `CANCELLED` | `cancelled` (kèm `cancelledAt`, lý do hủy) |
| `EXPIRED` | `expired` |
| `UNDERPAID` | `investigation_required` |
| `FAILED` | `investigation_required` |
| Không xác định | `reconciliation_required` → đối soát lại |

*   **Response:**
    *   `200 OK`: Ghi nhận xử lý webhook thành công.
