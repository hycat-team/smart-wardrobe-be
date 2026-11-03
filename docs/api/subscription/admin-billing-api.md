# Admin Billing API Specs

Tài liệu thiết kế các API quản trị viên liên quan đến quản lý giao dịch thanh toán và đối soát. Các giá trị hằng số nghiệp vụ chung tham chiếu tại [constants.md](constants.md); riêng hợp đồng `status` của API admin là bảng 11 giá trị trong tài liệu này (khác với bảng nhãn cũ trong constants.md).

Hợp đồng chi tiết của các thay đổi 029: `specs/029-admin-observability/contracts/admin-api-delta.md` §2.

> **Before→after của 029 (tóm tắt):**
> *   `status` trong mọi phản hồi admin của giao dịch: **số** → **chuỗi** đủ 11 giá trị (§Trạng thái bên dưới).
> *   `GET /transactions/{id}`: **entity thô** (tên trường Go: `ID`, `UserID`, `OrderCode`, `Status` dạng số) → **`DepositTransactionRes` camelCase**, đúng bộ trường như item trong danh sách.
> *   Kỳ ngày của danh sách: **nửa đêm UTC + `toDate <=`** (bỏ sót gần hết ngày cuối, lệch 7h; ngày sai bị **bỏ qua im lặng**) → **ngày lịch Việt Nam, cận nửa mở `[from, to)`**, ngày sai định dạng hoặc `fromDate > toDate` → **400** với thông báo tiếng Việt.
> *   Bộ lọc `status` nhận thêm `failed_legacy` (khứ hồi đủ 11 giá trị vào/ra).
> *   Phân trang, các filter còn lại và 4 endpoint resolve/retry: **không đổi**.

> **Bổ sung sau 029 — nhãn hiển thị ở chi tiết giao dịch:** `DepositTransactionRes` có thêm `user` và
> `targetPlan`. Cả hai là **`null` ở danh sách** (danh sách không gọi sang module khác) và chỉ được
> điền ở `GET /transactions/{id}`. Hai key vẫn xuất hiện ở **cả hai** phản hồi, nên FR-007 (detail
> cùng bộ key như item trong danh sách) không đổi. Lookup nhãn là best-effort: lỗi truy vấn hoặc
> không tìm thấy (tài khoản đã xoá, gói đã bị xoá khỏi danh mục, giao dịch nạp ví) đều trả `null`
> chứ không làm hỏng phản hồi chi tiết.

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
        "updatedAt": "2026-10-15T09:00:00Z",
        "user": null,
        "targetPlan": null
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
*   **Đối tượng ảnh hưởng:** Đọc bản ghi giao dịch nạp tiền `deposit_transactions`; đọc nhãn hiển thị qua contract identity (`user`) và danh mục gói của module subscription (`targetPlan`).
*   **Mô tả:** Trả về chi tiết một giao dịch thanh toán theo ID. **Thay đổi 029:** thay vì trả entity thô, phản hồi là `DepositTransactionRes` camelCase — đúng bộ trường như một item trong danh sách (kể cả `status` dạng chuỗi). **Bổ sung sau 029:** điền thêm `user` và `targetPlan`; danh sách trả hai trường này là `null`.
*   **Path Parameters:**
    *   `id` (string, required): ID giao dịch (UUID).
*   **Quy tắc nhãn hiển thị:**
    *   `user`: `id`, `username`, `email`, `firstName`, `lastName`, `avatarUrl`. Bỏ các trường nội bộ của identity (`roleSlug`, `isDeleted`, `createdByAdmin`, `gender`, `status`, `createdAt`).
    *   `targetPlan`: `id`, `slug`, `name`, `price`, `durationDays`, `planKind`, `tierRank` và các hạn mức của gói (`SubscriptionPlanBriefRes`).
    *   `null` khi: tài khoản đã bị xoá (contract identity không trả tài khoản soft-deleted), gói không còn trong danh mục, giao dịch `wallet_topup` (không có `targetPlanId`), hoặc lookup lỗi — trường hợp lỗi được ghi log, phản hồi vẫn `200`.
    *   `targetPlanId` **luôn** còn trên phản hồi, kể cả khi `targetPlan` là `null`.
*   **Response:**
    *   `200 OK`: Trả về thông tin chi tiết giao dịch, cùng shape với item trong danh sách, riêng `user`/`targetPlan` được điền.
    *   `404 Not Found`: Không tìm thấy giao dịch theo `id`.

```json
{
  "message": "Lấy chi tiết giao dịch thành công",
  "data": {
    "id": "9f0b6a3e-...",
    "userId": "1c2d3e4f-...",
    "orderCode": 1234567890,
    "amount": 250000,
    "currency": "VND",
    "status": "reconciliation_required",
    "transactionType": "direct_purchase",
    "targetPlanId": "7a1b...",
    "reference": "REF-20261015-0001",
    "createdAt": "2026-10-15T08:00:00Z",
    "updatedAt": "2026-10-15T09:00:00Z",
    "user": {
      "id": "1c2d3e4f-...",
      "username": "khachhang",
      "email": "khachhang@example.com",
      "firstName": "Nguyễn",
      "lastName": "An",
      "avatarUrl": null
    },
    "targetPlan": {
      "id": "7a1b...",
      "slug": "premium-monthly",
      "name": "Premium tháng",
      "price": 250000,
      "maxWardrobeItems": 500,
      "maxOutfits": 100,
      "aiOutfitDailyQuota": 30,
      "aiChatDailyQuota": 10,
      "durationDays": 30,
      "planKind": 1,
      "tierRank": 2
    }
  }
}
```

---

## Đối soát và xử lý giao dịch

Ma trận trạng thái được phép bắt đầu mỗi thao tác (BE là nguồn chân lý — gọi vượt ma trận trả **400** `"Không thể áp dụng quyết định xử lý thủ công vì trạng thái hoặc bằng chứng thanh toán không hợp lệ."`):

| `status` hiện tại | resolve-paid | resolve-cancelled | retry-investigation | resolve-creation-failed |
|---|:--:|:--:|:--:|:--:|
| `pending` | ✓ | ✓ | ✓ | ✗ |
| `creating` | ✓ | ✓ | ✓ | ✓ |
| `reconciling` | ✓ | ✓ | ✓ | ✗ |
| `reconciliation_required` | ✓ | ✓ | ✓ | ✓ |
| `investigation_required` | ✓ | ✓ | ✓ | ✓ |
| `cancelled` | ✓ | ✗ | ✗ | ✗ |
| `expired` | ✓ | ✗ | ✗ | ✗ |
| `failed_legacy` | ✓ | ✗ | ✗ | ✗ |
| `creation_failed` | ✗ | ✗ | ✗ | ✗ |
| `refunded` | ✗ | ✗ | ✗ | ✗ |
| `success` | ✗ | ✗ | ✗ | ✗ |
| `unknown` | ✗ | ✗ | ✗ | ✗ |

- `cancelled`/`expired`/`failed_legacy` vẫn cho `resolve-paid`: khách huỷ link nhưng vẫn chuyển khoản là trường hợp đối soát tay tồn tại để xử lý.
- `refunded` bị khoá vì tiền đã được hoàn vào ví; xác nhận thanh toán lần nữa sẽ cấp quyền thêm một lần.
- Trạng thái còn lại được kiểm tra lại **dưới row lock** khi thực thi; lệch so với lúc đọc → 400.

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
