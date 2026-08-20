# Admin Billing API Specs

Tài liệu thiết kế các API quản trị viên liên quan đến quản lý giao dịch thanh toán và đối soát. Tất cả các giá trị hằng số sử dụng tham chiếu tại [constants.md](constants.md).

---

## Danh sách giao dịch thanh toán

### 1. Lấy danh sách giao dịch thanh toán
*   **Endpoint:** `GET /api/v1/admin/subscriptions/transactions`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc danh sách bản ghi giao dịch nạp tiền `deposit_transactions`.
*   **Mô tả:** Trả về danh sách giao dịch nạp tiền/đăng ký, hỗ trợ lọc theo trạng thái và thời gian.
*   **Query Parameters:**
    *   `status` (string, optional): Trạng thái giao dịch (tham chiếu tại [constants.md:DepositStatus](constants.md#1-trang-thai-giao-dich-nap-tien-depositstatus)).
    *   `fromDate` (string, optional): Ngày bắt đầu (định dạng `YYYY-MM-DD`).
    *   `toDate` (string, optional): Ngày kết thúc (định dạng `YYYY-MM-DD`).
    *   `page` (integer, optional): Trang.
    *   `limit` (integer, optional): Số bản ghi/trang.
*   **Response:**
    *   `200 OK`: Trả về danh sách giao dịch thanh toán (phân trang).

### 2. Lấy chi tiết giao dịch thanh toán
*   **Endpoint:** `GET /api/v1/admin/subscriptions/transactions/{id}`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc bản ghi giao dịch nạp tiền `deposit_transactions`.
*   **Mô tả:** Trả về chi tiết một giao dịch thanh toán theo ID.
*   **Path Parameters:**
    *   `id` (string, required): ID giao dịch.
*   **Response:**
    *   `200 OK`: Trả về thông tin chi tiết giao dịch.

---

## Đối soát và xử lý giao dịch

### 3. Xác nhận giao dịch đã thanh toán
*   **Endpoint:** `POST /api/v1/admin/subscriptions/transactions/{orderCode}/resolve-paid`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái giao dịch `deposit_transactions`, cộng số dư ví hoặc kích hoạt gói Premium cho người dùng.
*   **Mô tả:** Xác nhận đã thanh toán. Bắt buộc có bằng chứng (reference + paymentLinkId).
*   **Path Parameters:**
    *   `orderCode` (integer, required): Mã đơn hàng.
*   **Request Body:**
    ```json
    {
      "reason": "Khách hàng đã chuyển khoản thành công",
      "reference": "123456789",
      "paymentLinkId": "pay_xxx"
    }
    ```
*   **Response:**
    *   `200 OK`: Xác nhận giao dịch thành công.

### 4. Xác nhận hủy giao dịch
*   **Endpoint:** `POST /api/v1/admin/subscriptions/transactions/{orderCode}/resolve-cancelled`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái giao dịch `deposit_transactions`.
*   **Mô tả:** Xác nhận giao dịch đã bị hủy.
*   **Path Parameters:**
    *   `orderCode` (integer, required): Mã đơn hàng.
*   **Request Body:**
    ```json
    {
      "reason": "Khách hàng yêu cầu hủy giao dịch"
    }
    ```
*   **Response:**
    *   `200 OK`: Xác nhận hủy giao dịch thành công.

### 5. Đánh dấu tạo link thanh toán thất bại
*   **Endpoint:** `POST /api/v1/admin/subscriptions/transactions/{orderCode}/resolve-creation-failed`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái giao dịch `deposit_transactions`.
*   **Mô tả:** Đánh dấu giao dịch là tạo link thanh toán thất bại.
*   **Path Parameters:**
    *   `orderCode` (integer, required): Mã đơn hàng.
*   **Request Body:**
    ```json
    {
      "reason": "Cổng thanh toán PayOS gặp lỗi khi tạo link"
    }
    ```
*   **Response:**
    *   `200 OK`: Đánh dấu trạng thái thành công.

### 6. Gửi lại yêu cầu đối soát giao dịch
*   **Endpoint:** `POST /api/v1/admin/subscriptions/transactions/{orderCode}/retry-investigation`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái giao dịch `deposit_transactions`.
*   **Mô tả:** Chuyển giao dịch về trạng thái cần đối soát lại.
*   **Path Parameters:**
    *   `orderCode` (integer, required): Mã đơn hàng.
*   **Request Body:**
    ```json
    {
      "reason": "Cần đối soát lại với cổng thanh toán"
    }
    ```
*   **Response:**
    *   `200 OK`: Gửi lại yêu cầu đối soát thành công.
