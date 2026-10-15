# Admin Billing API Specs

Tài liệu thiết kế các API quản trị viên liên quan đến quản lý giao dịch thanh toán và đối soát. Các giá trị hằng số nghiệp vụ chung tham chiếu tại [constants.md](constants.md); riêng hợp đồng `status` của API admin là bảng 11 giá trị trong tài liệu này (khác với bảng nhãn cũ trong constants.md).

Hợp đồng chi tiết của các thay đổi 029: `specs/029-admin-observability/contracts/admin-api-delta.md` §2.

> **Before→after của 029 (tóm tắt):**
> *   `status` trong mọi phản hồi admin của giao dịch: **số** → **chuỗi** đủ 11 giá trị (§Trạng thái bên dưới).
> *   `GET /transactions/{id}`: **entity thô** (tên trường Go: `ID`, `UserID`, `OrderCode`, `Status` dạng số) → **`DepositTransactionRes` camelCase**, đúng bộ trường như item trong danh sách.
> *   Kỳ ngày của danh sách: **nửa đêm UTC + `toDate <=`** (bỏ sót gần hết ngày cuối, lệch 7h; ngày sai bị **bỏ qua im lặng**) → **ngày lịch Việt Nam, cận nửa mở `[from, to)`**, ngày sai định dạng hoặc `fromDate > toDate` → **400** với thông báo tiếng Việt.
> *   Bộ lọc `status` nhận thêm `failed_legacy` (khứ hồi đủ 11 giá trị vào/ra).
> *   Phân trang, các filter còn lại và 4 endpoint resolve/retry: **không đổi**.

---

## Trạng thái giao dịch nạp tiền trong API admin (11 giá trị)

Trường `status` của mọi phản hồi admin của giao dịch là **chuỗi**. Bảng giá trị (data-model §2):

| Chuỗi xuất ra | Nhận ở filter `status` |
|---|---|
| `pending` | có |
| `success` | có |
| `failed_legacy` | **có (mới)** |
| `creating` | có |
| `reconciliation_required` | có |
| `reconciling` | có |
| `creation_failed` | có |
| `cancelled` | có |
| `expired` | có |
| `investigation_required` | có |
| `refunded` | có |

Bất biến: mọi giá trị xuất ra dùng lại được làm đầu vào filter (khứ hồi một vòng). Giá trị lạ (ngoài 11 giá trị, chỉ có thể phát sinh do dữ liệu bị sửa tay) được trả là `unknown` và không phải đầu vào hợp lệ của filter.

---

## Danh sách giao dịch thanh toán

### 1. Lấy danh sách giao dịch thanh toán
*   **Endpoint:** `GET /api/v1/admin/subscriptions/transactions`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc danh sách bản ghi giao dịch nạp tiền `deposit_transactions`.
*   **Mô tả:** Trả về danh sách giao dịch nạp tiền/đăng ký, hỗ trợ lọc theo trạng thái và khoảng ngày.
*   **Query Parameters:**
    *   `status` (string, optional): Trạng thái giao dịch — một trong 11 giá trị ở bảng trên. Giá trị khác bị bỏ qua (không lọc).
    *   `fromDate` (string, optional): Ngày bắt đầu (`YYYY-MM-DD`, ngày lịch Việt Nam, bao gồm trọn ngày).
    *   `toDate` (string, optional): Ngày kết thúc (`YYYY-MM-DD`, ngày lịch Việt Nam, bao gồm trọn ngày).
    *   `page` (integer, optional): Trang.
    *   `limit` (integer, optional): Số bản ghi/trang.
*   **Quy tắc kỳ ngày:** gửi một đầu thì đầu còn lại để mở (chỉ `fromDate` = từ ngày đó trở đi; chỉ `toDate` = đến hết ngày đó). Sai định dạng hoặc `fromDate > toDate` (khi gửi cả hai) → 400.
*   **Response:**
    *   `200 OK`: Trả về danh sách giao dịch thanh toán (phân trang).

```json
{
  "message": "Lấy danh sách giao dịch thành công",
  "data": {
    "items": [
      {
        "id": "9f0b6a3e-...",
        "userId": "1c2d3e4f-...",
        "orderCode": 1234567890,
        "amount": 250000,
        "currency": "VND",
        "status": "reconciliation_required",
        "transactionType": "direct_purchase",
        "targetPlanId": "7a1b...",
        "paymentUrl": "https://pay.payos.vn/web/abc",
        "reference": "REF-20261015-0001",
        "paymentLinkId": "pay_link_123",
        "failureReason": "gateway_timeout",
        "cancelledAt": "2026-10-15T10:30:00Z",
        "nextReconciliationAt": "2026-10-15T11:00:00Z",
        "createdAt": "2026-10-15T08:00:00Z",
        "updatedAt": "2026-10-15T09:00:00Z"
      }
    ],
    "metadata": { "page": 1, "limit": 20, "totalItems": 1, "totalPages": 1 }
  }
}
```
*   **Mã lỗi:**
    *   `400 Bad Request`: `fromDate`/`toDate` sai định dạng hoặc `fromDate > toDate` (thông báo tiếng Việt).

### 2. Lấy chi tiết giao dịch thanh toán
*   **Endpoint:** `GET /api/v1/admin/subscriptions/transactions/{id}`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc bản ghi giao dịch nạp tiền `deposit_transactions`.
*   **Mô tả:** Trả về chi tiết một giao dịch thanh toán theo ID. **Thay đổi 029:** thay vì trả entity thô, phản hồi là `DepositTransactionRes` camelCase — đúng bộ trường như một item trong danh sách (kể cả `status` dạng chuỗi).
*   **Path Parameters:**
    *   `id` (string, required): ID giao dịch (UUID).
*   **Response:**
    *   `200 OK`: Trả về thông tin chi tiết giao dịch, cùng shape với item trong danh sách.
    *   `404 Not Found`: Không tìm thấy giao dịch theo `id`.

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
