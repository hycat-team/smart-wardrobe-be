# Hợp đồng HTTP API — delta của 029

> Tài liệu này là hợp đồng chính xác cho implementation. Bản dành cho frontend nằm ở
> [`../frontend-guide.md`](../frontend-guide.md). Auth toàn bộ: Bearer JWT + role admin.
> Response bọc `APIResponse { message, data }` (trừ ghi chú khác).

## 1. MỚI — `GET /api/v1/admin/activity-events` (account service, identity)

Query params:

| Param | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `userId` | uuid | không | sai định dạng → 400 |
| `activityType` | string | không | chỉ `login` \| `refresh`; giá trị khác → 400 |
| `fromDate` | `YYYY-MM-DD` | không | ngày lịch VN, bao gồm trọn ngày |
| `toDate` | `YYYY-MM-DD` | không | `fromDate > toDate` → 400 |
| `page` | int | không | mặc định 1 |
| `limit` | int | không | mặc định 20, trần 100 |

Response `data` (`PaginationResult[*ActivityEventRes]`):

```json
{
  "items": [
    {
      "id": "9f0b6a3e-...",
      "userId": "1c2d3e4f-...",
      "activityType": "login",
      "occurredAt": "2026-10-15T08:31:02+07:00",
      "createdAt": "2026-10-15T08:31:02+07:00",
      "userEmail": "user@example.com",
      "userFirstName": "An",
      "userLastName": "Nguyễn"
    }
  ],
  "metadata": { "page": 1, "limit": 20, "totalItems": 1, "totalPages": 1 }
}
```

- Sắp xếp `occurredAt` giảm dần (tie-break `id`).
- `userFirstName`/`userLastName` có thể vắng mặt khi NULL; `userEmail` luôn có.
- Kỳ ngày: gửi một đầu thì đầu còn lại để mở (chỉ `fromDate` = từ ngày đó trở đi; chỉ `toDate` = đến hết ngày đó); gửi cả hai và `fromDate > toDate` → 400 (quy tắc chung, xem `../data-model.md` §3).
- HTTP status: 200; 400 (filter/kỳ sai); 401/403 (chuẩn middleware).
- Chỉ đọc; không có endpoint ghi.

## 2. THAY ĐỔI — nhóm giao dịch admin (account service, subscription)

### 2.1 `GET /api/v1/admin/subscriptions/transactions/{id}` — CHANGED (shape)

| | Trước | Sau |
|---|---|---|
| Response `data` | entity thô, tên trường Go (`ID`, `UserID`, `OrderCode`, `Status` dạng số, ...) | `DepositTransactionRes` camelCase — đúng bộ trường như item trong danh sách |
| `status` | số int16 | chuỗi (11 giá trị, xem §2.3) |

### 2.2 `GET /api/v1/admin/subscriptions/transactions` — CHANGED (status + kỳ)

- `items[].status`: số → chuỗi (11 giá trị). Các trường khác không đổi.
- `fromDate`/`toDate`: **trước** parse nửa đêm UTC, `toDate` bị `<=` nên bỏ sót gần hết ngày cuối
  (lệch 7h); ngày sai bị **bỏ qua im lặng**. **Sau**: ngày lịch VN, bao gồm trọn ngày; ngày sai
  định dạng hoặc `fromDate > toDate` → 400 với thông báo tiếng Việt.
- `status` filter nhận thêm `failed_legacy` (tổng 11 giá trị vào/ra khứ hồi).
- Phân trang, các filter còn lại, 4 endpoint resolve/retry: không đổi.

### 2.3 Bảng trạng thái chuỗi (11 giá trị)

`pending`, `success`, `failed_legacy`, `creating`, `reconciliation_required`, `reconciling`,
`creation_failed`, `cancelled`, `expired`, `investigation_required`, `refunded`.

## 3. THAY ĐỔI — dashboard admin (commerce service)

### 3.1 Nhóm endpoint nhận kỳ (mới thêm tham số)

| Endpoint | Tham số mới | Hành vi khi có kỳ | Mặc định khi không gửi |
|---|---|---|---|
| `GET /admin/dashboard/subscriptions/revenue-breakdown` | đã nhận `fromDate/toDate`; đổi cách diễn giải sang ngày lịch VN | chỉ giao dịch trong kỳ | 6 tháng gần nhất |
| `GET /admin/dashboard/subscriptions/distribution` | **MỚI** `fromDate/toDate` | user có gói hiệu lực giao với kỳ (R7) | hiện trạng |
| `GET /admin/dashboard/subscriptions/renewal-stats` | **MỚI** `fromDate/toDate` | attempt tạo trong kỳ | toàn thời gian |
| `GET /admin/dashboard/ai/error-breakdown` | **MỚI** `fromDate/toDate` | lỗi + request + phân nhóm cùng kỳ | 30 ngày gần nhất |
| `GET /admin/dashboard/payos-reconciliation` | **MỚI** `fromDate/toDate` | tổng hợp theo kỳ từ bảng giao dịch | hôm nay (VN) |
| `GET /admin/dashboard/ai-margin`, `ai/usage-by-operation` | đã nhận; đổi cách diễn giải sang ngày lịch VN | như cũ | như cũ |

Mọi endpoint trên: ngày sai định dạng → 400 (ai-margin/usage đã có sẵn hành vi này; các endpoint
còn lại mới thêm), `fromDate > toDate` → 400 (mới cho tất cả), và diễn giải kỳ đổi sang ngày lịch
VN (trước đây ai-margin/usage nhận mốc nửa đêm UTC). Gửi một đầu ngày: đầu còn lại để mở (quy tắc
chung tại `../data-model.md` §3).

### 3.2 `GET /admin/dashboard/payos-reconciliation` — CHANGED (phạm vi + giá trị)

| Field | Trước | Sau |
|---|---|---|
| `totalTransactions`, `successCount`, `failedCount`, `pendingCount`, `voidedCount` | chỉ `direct_purchase`, từ ảnh chụp mới nhất (15 phút/lần) | **cả `direct_purchase` + `wallet_topup`**, tổng hợp theo kỳ (mặc định hôm nay) |
| `totalSuccessValue` | = doanh thu subscription cùng phạm vi (mượn trường) | tổng tiền giao dịch thành công trên phạm vi hai loại, cùng kỳ |

- Shape không đổi; nguồn dữ liệu đổi từ snapshot sang truy vấn theo kỳ (R6).
- `GET /admin/dashboard/overview` → `payosReconcile` mang ngữ nghĩa mới, kỳ mặc định hôm nay.
- Ảnh chụp dashboard (worker dùng chung truy vấn) mang cùng phạm vi hai loại: các số đếm payOS
  trong snapshot mới — kể cả `payOSSuccessCount` của `kpi-cards` — đổi giá trị chủ đích (shape
  không đổi).

### 3.3 `GET /admin/dashboard/ai/error-breakdown` — CHANGED (cùng kỳ)

| Field | Trước | Sau |
|---|---|---|
| `totalErrors`, `breakdown[]` | toàn thời gian | trong kỳ (mặc định 30 ngày); Σ breakdown = totalErrors |
| `totalRequests` | 30 ngày | cùng kỳ |
| `errorRate` | trộn kỳ (có thể > 100%) | = totalErrors/totalRequests×100, cùng kỳ; 0 khi mẫu 0 |

### 3.4 `GET /admin/dashboard/subscriptions/distribution` — CHANGED (kỳ)

Shape không đổi (`distribution[]`, `totalUsers`). Khi có kỳ: đếm user có gói hiệu lực giao với kỳ,
nhóm theo gói hiện tại; giới hạn lịch sử đổi gói ghi trong guide.

## 4. Không đổi

- 4 endpoint resolve/retry giao dịch (`retry-investigation`, `resolve-paid`, `resolve-cancelled`,
  `resolve-creation-failed`): đường dẫn, request, nghiệp vụ giữ nguyên; phản hồi trả `data: null`
  (không chứa trạng thái giao dịch — vế điều kiện của FR-009 hiện không kích hoạt).
- `kpi-cards` shape giữ nguyên (riêng `payOSSuccessCount` đổi giá trị theo phạm vi hai loại — §3.2);
  `ai-margin`, `ai/usage-by-operation`, `drilldown/ai-events`: shape giữ nguyên.
- DTO user-facing `PaymentLinkDTO.paymentStatus` vẫn là số (ngoài phạm vi admin).
