# Data Model: Khả năng quan sát cho admin (029)

> Không có migration và không có bảng/cột mới (R12). Tài liệu này chốt ngữ nghĩa đọc,
> hợp đồng serialize và quy tắc validation mà implementation phải tuân theo.

## 1. `identity.user_activity_events` — mô hình đọc mới

Bảng hiện có (append-only, không đổi):

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | uuid PK | |
| `user_id` | uuid NOT NULL, FK → `identity.users(id)` ON DELETE CASCADE | |
| `activity_type` | varchar(20) NOT NULL | Tập giá trị hiện có: `login`, `refresh` (`useractivitytype`) |
| `occurred_at` | timestamptz NOT NULL | Thời điểm phát sinh — khoá sắp xếp của API |
| `created_at` | timestamptz NOT NULL | Thời điểm ghi nhận |

Read model mới (domain repositories):

```
AdminActivityEventRow {
  ID           uuid.UUID
  UserID       uuid.UUID
  ActivityType string
  OccurredAt   time.Time
  CreatedAt    time.Time
  Email        string        // từ LEFT JOIN users
  FirstName    *string       // NULL khi chưa đặt
  LastName     *string
}
```

Quy tắc truy vấn:

- `LEFT JOIN users u ON u.id = e.user_id` — sự kiện của user soft-deleted vẫn hiển thị (danh
  tính đọc được vì soft delete giữ dòng); hard delete đã cascade xoá sự kiện.
- Sắp xếp `occurred_at DESC, id DESC` (tie-breaker ổn định cho phân trang).
- Lọc: `user_id = ?` (tuỳ chọn), `activity_type = ?` (tuỳ chọn), `occurred_at >= from AND
  occurred_at < to` (tuỳ chọn, cận nửa mở VN — R1).
- COUNT dùng đúng tập điều kiện của SELECT.

## 2. Trạng thái giao dịch nạp tiền — hợp đồng chuỗi (11 giá trị)

Type Go `depositstatus.DepositStatus` (int16) **không đổi**. Chỉ DTO admin serialize thành chuỗi
(R4); `PaymentLinkDTO.PaymentStatus` (user-facing) giữ số nguyên.

| Giá trị số | Chuỗi xuất ra | Nhận ở filter |
|---|---|---|
| 0 | `pending` | có |
| 1 | `success` | có |
| 2 | `failed_legacy` | **có (mới)** |
| 3 | `creating` | có |
| 4 | `reconciliation_required` | có |
| 5 | `reconciling` | có |
| 6 | `creation_failed` | có |
| 7 | `cancelled` | có |
| 8 | `expired` | có |
| 9 | `investigation_required` | có |
| 10 | `refunded` | có |

Bất biến (FR-008): mọi giá trị xuất ra dùng lại được làm đầu vào filter — khứ hồi đủ 11 giá trị.

`DepositTransactionRes` (admin): trường `status` đổi kiểu `DepositStatus` → `string`; các trường
còn lại không đổi (`id`, `userId`, `orderCode`, `amount`, `currency`, `transactionType`,
`targetPlanId?`, `paymentUrl?`, `reference?`, `paymentLinkId?`, `failureReason?`, `cancelledAt?`,
`nextReconciliationAt?`, `createdAt`, `updatedAt`). `GET /transactions/{id}` trả đúng shape này.

## 3. Kỳ dữ liệu — quy tắc chuẩn (R1)

- Tham số HTTP: `fromDate`, `toDate` — `YYYY-MM-DD`, ngày lịch VN, bao gồm trọn hai đầu.
- Biểu diễn nội bộ: `[from, to)` với `from = StartOfDayInVN(fromDate)`,
  `to = EndOfDayInVN(toDate)`; mọi SQL `>= from AND < to`.
- Gửi một đầu: đầu còn lại để mở (chỉ `fromDate` = từ ngày đó trở đi; chỉ `toDate` = đến hết
  ngày đó); helper trả zero-time cho đầu thiếu, repo bỏ điều kiện tương ứng. `400` chỉ khi định
  dạng sai hoặc khi gửi **cả hai** mà `fromDate > toDate`.
- Validation chung (`shared/pkg/utils/validation`): 400 tiếng Việt khi định dạng sai hoặc
  `fromDate > toDate` (khi cả hai được gửi).
- Mặc định khi không gửi (từng endpoint, theo FR-012):

| Endpoint | Mặc định |
|---|---|
| `revenue-breakdown` | 6 tháng gần nhất → nay (đổi mốc sang biên ngày VN) |
| `distribution` | không kỳ = hiện trạng (không đổi) |
| `renewal-stats` | không kỳ = toàn thời gian (không đổi) |
| `ai/error-breakdown` | 30 ngày gần nhất (đổi chủ đích — cùng kỳ tử/mẫu) |
| `payos-reconciliation` | hôm nay VN (đổi nguồn: truy vấn theo kỳ thay vì ảnh chụp) |
| `ai-margin`, `ai/usage-by-operation` | giữ nguyên (30 ngày / giá trị zero → 30 ngày) |

## 4. Thống kê payOS — phạm vi và giá trị (R6)

`AdminSubscriptionStats` sau thay đổi (account service):

| Trường | Trước | Sau |
|---|---|---|
| `TotalSubscriptionRevenue` | SUM(amount) status=success, type=direct_purchase | **không đổi** (doanh thu gói cước) |
| `PayOSTransactionCount` | COUNT type=direct_purchase | COUNT **cả hai type** |
| `PayOSSuccessCount` / `PendingCount` / `FailedCount` / `VoidedCount` | theo direct_purchase | theo **cả hai type** |
| `TotalPayOSSuccessValue` (mới) | — | SUM(amount) status=success trên **cả hai type** |

Bucket trạng thái không đổi: pending = ActivePaymentStatuses; failed = FailedLegacy +
CreationFailed + InvestigationRequired; voided = Cancelled + Expired + Refunded.

`PayOSReconciliationRes.TotalSuccessValue` (commerce) lấy từ `TotalPayOSSuccessValue` — không còn
mượn `SubscriptionRevenueVND`.

Bảng `dashboard_admin_daily_snapshots`: **không đổi cấu trúc**; cột `payos_*_count` trong các
snapshot mới mang phạm vi hai loại (worker dùng cùng truy vấn với `VNDayRange`), nên
`kpi-cards.payOSSuccessCount` cũng đổi giá trị theo (shape không đổi — FR-017).

## 5. Phân bổ theo gói — hai ngữ nghĩa theo tham số (R7)

| Trường hợp | Điều kiện | Nghĩa |
|---|---|---|
| Không gửi kỳ | `expires_at IS NULL OR expires_at > now()` | Hiện trạng (không đổi) |
| Có kỳ | `started_at < to AND (expires_at IS NULL OR expires_at >= from)` | Giao cửa sổ gói hiện tại với kỳ |

Nhóm theo `current_plan_code`. Giới hạn chấp nhận: lịch sử đổi gói không tái dựng; FREE
(`expires_at IS NULL`) hiện diện mọi kỳ.

## 6. Hiệu quả gia hạn — lọc theo `created_at` (R8)

`subscription_renewal_attempts`: giữ điều kiện `status <> 'processing'`; thêm
`created_at >= from AND created_at < to` khi có kỳ. `successRate` giữ công thức hiện có
(success / (success + failed)).

## 7. Lỗi AI — cùng kỳ (R9)

`AIErrorBreakdownRes` giữ shape (`totalErrors`, `totalRequests`, `errorRate`, `breakdown[]`);
ngữ nghĩa đổi:

- `breakdown[]` và `totalErrors` cùng giới hạn trong kỳ (Σ breakdown = totalErrors).
- `totalRequests` cùng kỳ.
- `errorRate` = totalErrors/totalRequests×100; 0 khi mẫu 0; không vượt 100% khi E ≤ R.

## 8. Quy tắc validation tổng hợp (cho implementation)

- Mọi endpoint nhận kỳ: 400 khi định dạng ngày sai hoặc `fromDate > toDate` (helper R1).
- `activity-events`: `userId` sai UUID → 400; `activityType` ngoài `{login, refresh}` → 400;
  `page`/`limit` chuẩn hoá theo `PaginationQuery` (limit trần 100).
- Danh sách giao dịch: ngày sai/đảo → 400 (thay đổi before→after so với hành vi nuốt lỗi cũ).
- Giao dịch không tồn tại ở `GET /transactions/{id}`: 404 theo `apperror` hiện có.
