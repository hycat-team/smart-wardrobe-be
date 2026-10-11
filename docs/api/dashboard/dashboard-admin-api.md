# Dashboard Admin API

Tài liệu đặc tả các API Dashboard dành cho **System Admin**. Bao gồm các endpoints từ dashboard-api.md và dashboard-api-v2.md.

Xem thêm hằng số tại [constants.md](constants.md).

> **Quy ước ngày:** mọi mốc "ngày" trong dashboard admin đều tính theo **lịch Việt Nam (+07:00)**, không phải UTC. Snapshot `snapshot_date` cũng lưu ngày VN.

---

## 1. Lấy thẻ chỉ số KPI tổng quan

- **Endpoint:** `GET /api/v1/admin/dashboard/kpi-cards`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Dữ liệu từ bảng `dashboard_admin_daily_snapshots` (snapshot mới nhất), nếu chưa có snapshot thì fallback query trực tiếp từ domain reader.
- **Response struct:** `AdminKPICardsRes`

| Field                  | JSON                     | Kiểu Go   | Ý nghĩa                                                         |
| ---------------------- | ------------------------ | --------- | --------------------------------------------------------------- |
| TotalUsers             | `totalUsers`             | int       | Tổng số người dùng đã đăng ký                                   |
| ActiveUsersDAU         | `activeUsersDau`         | int       | Số người dùng hoạt động trong ngày (Daily Active Users) = số user phân biệt có sự kiện login/refresh trong ngày (bảng `user_activity_events`), **không** dựa trên `users.updated_at` |
| TotalBrands            | `totalBrands`            | int       | Tổng số nhãn hàng đang hoạt động                                |
| PendingReviewBrands    | `pendingReviewBrands`    | int       | Số nhãn hàng đang chờ phê duyệt                                 |
| SubscriptionRevenueVND | `subscriptionRevenueVnd` | float64   | Tổng doanh thu gói Premium (VND) = `SUM(amount)` các giao dịch `direct_purchase` có `status=Success` trong ngày (bỏ qua pending/failed/cancelled/expired/refunded) |
| NetAIMarginVND         | `netAiMarginVnd`         | float64   | Biên lợi nhuận AI = doanh thu - chi phí token (VND)             |
| PayOSSuccessCount      | `payOSSuccessCount`      | int       | Số giao dịch PayOS thành công trong ngày                        |
| UpdatedAt              | `updatedAt`              | time.Time | Thời điểm cập nhật gần nhất                                     |

**Ví dụ response:**

```json
{
    "success": true,
    "data": {
        "totalUsers": 1500,
        "activeUsersDau": 720,
        "totalBrands": 45,
        "pendingReviewBrands": 3,
        "subscriptionRevenueVnd": 125000000.0,
        "netAiMarginVnd": 87500000.0,
        "payOSSuccessCount": 98,
        "updatedAt": "2026-07-23T10:00:00Z"
    }
}
```

---

## 2. Phân tích biên lợi nhuận AI Token

- **Endpoint:** `GET /api/v1/admin/dashboard/ai-margin`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Query Parameters:**
    - `fromDate` (string, optional) — Ngày bắt đầu, định dạng `YYYY-MM-DD`.
    - `toDate` (string, optional) — Ngày kết thúc, định dạng `YYYY-MM-DD`.
- **Response struct:** `AIMarginAnalyticsRes`

| Field                   | JSON                      | Kiểu Go   | Ý nghĩa                                                                             |
| ----------------------- | ------------------------- | --------- | ----------------------------------------------------------------------------------- |
| FromDate                | `fromDate`                | time.Time | Ngày bắt đầu kỳ phân tích                                                           |
| ToDate                  | `toDate`                  | time.Time | Ngày kết thúc kỳ phân tích                                                          |
| TotalSubscriptionRevVND | `totalSubscriptionRevVnd` | float64   | Tổng doanh thu subscription trong kỳ (VND)                                          |
| TotalAIActualCostVND    | `totalAiActualCostVnd`    | float64   | Tổng chi phí AI token thực tế (VND). Quy đổi từ `actual_cost_micro_vnd / 1_000_000` |
| NetAIMarginVND          | `netAiMarginVnd`          | float64   | Lợi nhuận ròng = doanh thu - chi phí (VND)                                          |
| MarginPercentage        | `marginPercentage`        | float64   | Tỷ lệ biên lợi nhuận = `(netMargin / revenue) * 100`. `0.0` nếu không có doanh thu  |
| TotalAIPaidTokens       | `totalAiPaidTokens`       | int64     | Tổng số token AI đã tiêu thụ (paid tokens)                                          |
| AIRequestsCount         | `aiRequestsCount`         | int64     | Tổng số request AI đã gửi trong kỳ                                                  |
| AIErrorCount            | `aiErrorCount`            | int64     | Tổng số request AI bị lỗi trong kỳ                                                  |

---

## 3. Thống kê đối soát giao dịch PayOS

- **Endpoint:** `GET /api/v1/admin/dashboard/payos-reconciliation`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Response struct:** `PayOSReconciliationRes`

| Field             | JSON                | Kiểu Go | Ý nghĩa                                                                     |
| ----------------- | ------------------- | ------- | --------------------------------------------------------------------------- |
| TotalTransactions | `totalTransactions` | int     | Tổng số giao dịch PayOS trong snapshot gần nhất                             |
| SuccessCount      | `successCount`      | int     | Số giao dịch thành công (`status=Success`)                                  |
| FailedCount       | `failedCount`       | int     | Số giao dịch **thất bại thật** (`FailedLegacy`, `CreationFailed`, `InvestigationRequired`) — KHÔNG còn tính bằng `total − success` |
| PendingCount      | `pendingCount`      | int     | Số giao dịch đang xử lý (`Pending`, `Creating`, `ReconciliationRequired`, `Reconciling`) |
| VoidedCount       | `voidedCount`       | int     | Số giao dịch đóng mà không thu tiền (hủy `Cancelled`, hết hạn `Expired`, hoàn tiền `Refunded`) |
| TotalSuccessValue | `totalSuccessValue` | float64 | Tổng giá trị VND của giao dịch thành công (lấy từ `subscriptionRevenueVND`) |

---

## 4. Báo cáo tổng quan hệ thống

- **Endpoint:** `GET /api/v1/admin/dashboard/overview`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Response struct:** `AdminOverviewRes` — composite từ 3 response trên

| Field          | JSON             | Kiểu Go                  | Ý nghĩa             |
| -------------- | ---------------- | ------------------------ | ------------------- |
| KPICards       | `kpiCards`       | \*AdminKPICardsRes       | Thẻ KPI tổng quan   |
| AIMargin       | `aiMargin`       | \*AIMarginAnalyticsRes   | Phân tích AI margin |
| PayOSReconcile | `payosReconcile` | \*PayOSReconciliationRes | Đối soát PayOS      |

Các trường luôn có dữ liệu. Nếu sub-component lỗi, API trả về HTTP 500 với thông báo lỗi.

---

## 5. Lấy phân bổ người dùng theo gói subscription

- **Endpoint:** `GET /api/v1/admin/dashboard/subscriptions/distribution`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê số lượng người dùng đang hoạt động theo từng gói subscription (`free`, `premium-monthly`, v.v.), kèm tỷ lệ phần trăm. Chỉ đếm subscription còn hiệu lực (`expires_at IS NULL OR expires_at > now()`); gói đã hết hạn không được tính.
- **Response struct:** `SubscriptionDistributionRes`
- **Nguồn dữ liệu:** Bảng `user_subscriptions`, query `SELECT current_plan_code, COUNT(*) WHERE expires_at IS NULL OR expires_at > now() GROUP BY current_plan_code ORDER BY COUNT(*) DESC`.

| Field        | JSON           | Kiểu                          | Ý nghĩa                                          |
| ------------ | -------------- | ----------------------------- | ------------------------------------------------ |
| Distribution | `distribution` | []SubscriptionDistributionDTO | Danh sách phân bổ theo plan_code                 |
| ├ PlanCode   | `planCode`     | string                        | Mã gói subscription (`free`, `premium-monthly`, ...) |
| ├ UserCount  | `userCount`    | int                           | Số người dùng thuộc gói này                      |
| └ Percentage | `percentage`   | float64                       | Tỷ lệ phần trăm so với tổng user                 |
| TotalUsers   | `totalUsers`   | int                           | Tổng số người dùng có subscription               |

---

## 6. Lấy doanh thu subscription theo gói và thời gian

- **Endpoint:** `GET /api/v1/admin/dashboard/subscriptions/revenue-breakdown`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê doanh thu subscription theo từng plan_code và theo tháng, chỉ tính các giao dịch `direct_purchase` thành công. Hỗ trợ lọc khoảng thời gian.
- **Query Parameters:**
    - `fromDate` (string, optional) — Ngày bắt đầu, định dạng `YYYY-MM-DD`. Mặc định: 6 tháng trước.
    - `toDate` (string, optional) — Ngày kết thúc, định dạng `YYYY-MM-DD`. Mặc định: hôm nay.
- **Response struct:** `RevenueBreakdownRes`
- **Nguồn dữ liệu:** Bảng `deposit_transactions`, `WHERE status=1 AND transaction_type='direct_purchase'`, `GROUP BY plan_code, DATE_TRUNC('month', created_at)`. Khoảng ngày bao cả 2 đầu: `[fromDate, toDate+1d)` nên `toDate` ở mốc 00:00 vẫn tính trọn ngày đó.

| Field              | JSON               | Kiểu                  | Ý nghĩa                               |
| ------------------ | ------------------ | --------------------- | ------------------------------------- |
| FromDate           | `fromDate`         | string                | Ngày bắt đầu kỳ thống kê              |
| ToDate             | `toDate`           | string                | Ngày kết thúc kỳ thống kê             |
| Breakdown          | `breakdown`        | []RevenueBreakdownDTO | Danh sách doanh thu theo plan + tháng |
| ├ PlanCode         | `planCode`         | string                | Mã gói subscription                   |
| ├ Month            | `month`            | string                | Tháng (YYYY-MM)                       |
| ├ TransactionCount | `transactionCount` | int                   | Số giao dịch thành công               |
| └ RevenueVND       | `revenueVnd`       | float64               | Tổng doanh thu (VND)                  |
| TotalRevenueVND    | `totalRevenueVnd`  | float64               | Tổng doanh thu tất cả gói             |

---

## 7. Lấy thống kê sử dụng AI theo tác vụ

- **Endpoint:** `GET /api/v1/admin/dashboard/ai/usage-by-operation`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê số lượng request, chi phí (VND), và token AI theo từng loại tác vụ (operation) và tuyến xử lý (logical_route). Chi phí được quy đổi từ micro VND sang VND.
- **Query Parameters:**
    - `fromDate` (string, optional) — Ngày bắt đầu, định dạng `YYYY-MM-DD`. Mặc định: 30 ngày trước.
    - `toDate` (string, optional) — Ngày kết thúc, định dạng `YYYY-MM-DD`. Mặc định: hôm nay.
- **Response struct:** `AIUsageByOperationRes`
- **Nguồn dữ liệu:** Bảng `ai_usage_events`, `GROUP BY operation, logical_route`.

| Field               | JSON                | Kiểu                    | Ý nghĩa                                                |
| ------------------- | ------------------- | ----------------------- | ------------------------------------------------------ |
| FromDate            | `fromDate`          | string                  | Ngày bắt đầu kỳ thống kê                               |
| ToDate              | `toDate`            | string                  | Ngày kết thúc kỳ thống kê                              |
| Operations          | `operations`        | []AIUsageByOperationDTO | Danh sách thống kê theo operation                      |
| ├ Operation         | `operation`         | string                  | Loại tác vụ (outfit-recommend, chat, ...)              |
| ├ LogicalRoute      | `logicalRoute`      | string                  | Tuyến xử lý (normal, reduced, free)                    |
| ├ RequestCount      | `requestCount`      | int64                   | Số lượng request                                       |
| ├ TotalCostVND      | `totalCostVnd`      | float64                 | Tổng chi phí (VND) = actual_cost_micro_vnd / 1,000,000 |
| ├ TotalPromptTokens | `totalPromptTokens` | int64                   | Tổng token đầu vào                                     |
| └ TotalOutputTokens | `totalOutputTokens` | int64                   | Tổng token đầu ra                                      |

---

## 8. Lấy thống kê lỗi AI

- **Endpoint:** `GET /api/v1/admin/dashboard/ai/error-breakdown`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê số lượng lỗi AI, tổng số request trong 30 ngày qua, và tỷ lệ lỗi. Chi tiết lỗi được phân nhóm theo error_code, provider, và model.
- **Response struct:** `AIErrorBreakdownRes`
- **Nguồn dữ liệu:** Bảng `ai_usage_events`, lấy các status thuộc nhóm lỗi: `released` (gọi provider thất bại), `unknown_usage` (không quy được usage), `expired_unverified` (reservation hết hạn chưa xác nhận). Tổng request lấy từ `COUNT(*)` trong 30 ngày gần nhất.

| Field         | JSON            | Kiểu                  | Ý nghĩa                                        |
| ------------- | --------------- | --------------------- | ---------------------------------------------- |
| TotalErrors   | `totalErrors`   | int64                 | Tổng số request bị lỗi                         |
| TotalRequests | `totalRequests` | int64                 | Tổng số request (30 ngày qua)                  |
| ErrorRate     | `errorRate`     | float64               | Tỷ lệ lỗi = totalErrors / totalRequests \* 100 |
| Breakdown     | `breakdown`     | []AIErrorBreakdownDTO | Chi tiết lỗi theo nhóm                         |
| ├ ErrorCode   | `errorCode`     | string                | Mã lỗi (rate_limit_exceeded, timeout, ...)     |
| ├ Provider    | `provider`      | string                | Nhà cung cấp AI (openai, anthropic, ...)       |
| ├ Model       | `model`         | string                | Model AI (gpt-4, claude-3, ...)                |
| └ ErrorCount  | `errorCount`    | int64                 | Số lần xuất hiện lỗi này                       |

---

## 9. Lấy thống kê gia hạn subscription

- **Endpoint:** `GET /api/v1/admin/dashboard/subscriptions/renewal-stats`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê tổng quan về tình trạng gia hạn subscription: tổng số lần gia hạn, số thành công, số thất bại, tỷ lệ thành công, và số lần thử lại trung bình.
- **Response struct:** `RenewalStatsRes`
- **Nguồn dữ liệu:** Bảng `subscription_renewal_attempts`.

| Field         | JSON            | Kiểu    | Ý nghĩa                                                |
| ------------- | --------------- | ------- | ------------------------------------------------------ |
| TotalAttempts | `totalAttempts` | int     | Tổng số lần gia hạn **đã kết thúc** (loại trạng thái đang chạy `processing`) |
| SuccessCount  | `successCount`  | int     | Số lần gia hạn thành công (status = 'succeeded')       |
| FailedCount   | `failedCount`   | int     | Số lần gia hạn thất bại (status = 'failed')            |
| SuccessRate   | `successRate`   | float64 | Tỷ lệ thành công = successCount / (successCount + failedCount) \* 100 — các kết quả phi-thanh-toán (`skipped`, `downgraded`, `renewed`) không bị tính là thất bại |
| AvgRetryCount | `avgRetryCount` | float64 | Số lần thử lại trung bình (AVG của attempt_count trên các lần đã kết thúc) |

---

## 10. Truy vết sự kiện lỗi AI (Admin)

- **Endpoint:** `GET /api/v1/admin/dashboard/drilldown/ai-events`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Query Parameters:**
    - `status` (string, optional) — Lọc theo trạng thái (VD: `"FAILED"`).
    - `page` (int, optional, default=1) — Số trang.
    - `limit` (int, optional, default=20, max=100) — Số bản ghi mỗi trang.
- **Response struct:** `DrillDownAIEventListRes` (alias của `PaginationResult[DrillDownAIEventDTO]`)

**Cấu trúc `DrillDownAIEventDTO`:**

| Field        | JSON           | Kiểu Go   | Ý nghĩa                                          |
| ------------ | -------------- | --------- | ------------------------------------------------ |
| RequestID    | `requestId`    | string    | ID request AI                                    |
| Provider     | `provider`     | string    | Nhà cung cấp AI (`"openai"`, `"anthropic"`, ...) |
| Model        | `model`        | string    | Model AI (`"gpt-4"`, `"claude-3"`, ...)          |
| StatusCode   | `statusCode`   | int       | Mã HTTP response từ provider                     |
| ErrorMessage | `errorMessage` | string    | Thông báo lỗi                                    |
| CreatedAt    | `createdAt`    | time.Time | Thời điểm request                                |

**Cấu trúc `PaginationMetadata`:**

| Field      | JSON         | Kiểu Go | Ý nghĩa           |
| ---------- | ------------ | ------- | ----------------- |
| Page       | `page`       | int     | Trang hiện tại    |
| Limit      | `limit`      | int     | Số item mỗi trang |
| TotalItems | `totalItems` | int64   | Tổng số bản ghi   |
| TotalPages | `totalPages` | int     | Tổng số trang     |

---

## Phụ lục: Trace toàn bộ Metric — Admin

### Admin KPI Cards (`GET /api/v1/admin/dashboard/kpi-cards`)

| Field                    | Nguồn dữ liệu          | Câu truy vấn / Cách tính                                                                                                                                                                                                                 |
| ------------------------ | ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `totalUsers`             | `users` table          | `SELECT COUNT(*) FROM users WHERE is_deleted = false`                                                                                                                                                                                    |
| `activeUsersDau`         | `user_activity_events` | `SELECT COUNT(DISTINCT user_id) FROM user_activity_events WHERE occurred_at BETWEEN [00:00 hôm nay, 00:00 ngày mai) theo giờ VN AND user chưa bị xoá`. Nguồn sự kiện: login + refresh token (refresh ~15 phút/lần khi app đang dùng)      |
| `totalBrands`            | `brands` table         | `SELECT COUNT(*) FROM brands`                                                                                                                                                                                                            |
| `pendingReviewBrands`    | `brands` table         | `SELECT COUNT(*) FROM brands WHERE status = 'PENDING_REVIEW'`                                                                                                                                                                            |
| `subscriptionRevenueVnd` | `deposit_transactions` | `SELECT COALESCE(SUM(amount) FILTER (WHERE status=1),0) FROM deposit_transactions WHERE transaction_type='direct_purchase' AND created_at IN [ngày VN]` — lấy từ Aggregator Worker chạy 15p/lần                                                                     |
| `netAiMarginVnd`         | Tính toán              | `subscriptionRevenueVnd - (aiActualCostMicroVnd / 1_000_000)`                                                                                                                                                                            |
| `payOSSuccessCount`      | `deposit_transactions` | `SELECT COUNT(*) FROM deposit_transactions WHERE transaction_type='direct_purchase' AND status=1 (Success) AND created_at IN [ngày VN]`                                                                                                     |

### AI Margin (`GET /api/v1/admin/dashboard/ai-margin`)

| Field                     | Nguồn                             | Cách tính                                                                                                       |
| ------------------------- | --------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `totalSubscriptionRevVnd` | `dashboard_admin_daily_snapshots` | SUM(snapshot.SubscriptionRevenueVND) trong khoảng ngày                                                          |
| `totalAiActualCostVnd`    | Tính toán                         | `totalRev - totalMargin` (suy ngược)                                                                            |
| `netAiMarginVnd`          | `dashboard_admin_daily_snapshots` | SUM(snapshot.NetAIMarginVND) trong khoảng ngày                                                                  |
| `marginPercentage`        | Tính toán                         | `(totalMargin / totalRev) * 100`. `0` nếu `totalRev=0`                                                          |
| `totalAiPaidTokens`       | `ai_usage_events`                 | `SELECT COALESCE(SUM(total_tokens),0) FROM ai_usage_events WHERE created_at IN [ngày]`                          |
| `aiRequestsCount`         | `ai_usage_events`                 | `SELECT COUNT(*) FROM ai_usage_events WHERE created_at IN [ngày]`                                               |
| `aiErrorCount`            | `ai_usage_events`                 | `SELECT COUNT(*) FROM ai_usage_events WHERE created_at IN [ngày] AND status IN ('FAILED','RELEASED','EXPIRED')` |

### PayOS Reconciliation (`GET /api/v1/admin/dashboard/payos-reconciliation`)

| Field               | Nguồn                             | Cách tính                          |
| ------------------- | --------------------------------- | ---------------------------------- |
| `totalTransactions` | `dashboard_admin_daily_snapshots` | `snapshot.PayOSTransactionCount`   |
| `successCount`      | `dashboard_admin_daily_snapshots` | `snapshot.PayOSSuccessCount`       |
| `failedCount`       | `dashboard_admin_daily_snapshots` | `snapshot.PayOSFailedCount` (thất bại thật, không còn `total − success`) |
| `pendingCount`      | `dashboard_admin_daily_snapshots` | `snapshot.PayOSPendingCount`       |
| `voidedCount`       | `dashboard_admin_daily_snapshots` | `snapshot.PayOSVoidedCount`        |
| `totalSuccessValue` | `dashboard_admin_daily_snapshots` | `snapshot.SubscriptionRevenueVND` (chỉ tính giao dịch thành công) |

### Dynamic Query — Admin Metrics (`POST /api/v1/dashboard/analytics/query`)

Backend hiện chỉ map các metric sau. Các metric khác gửi vào sẽ bị bỏ qua và nhận giá trị `0`.

| Metric                       | Mapped từ field                   | Kiểu Go gốc     |
| ---------------------------- | --------------------------------- | --------------- |
| `"total_users"`              | `snapshot.TotalUsers`             | int → float64   |
| `"active_users_dau"`         | `snapshot.ActiveUsersDau`         | int → float64   |
| `"total_brands"`             | `snapshot.TotalBrands`            | int → float64   |
| `"pending_review_brands"`    | `snapshot.PendingReviewBrands`    | int → float64   |
| `"ai_requests_count"`        | `snapshot.AIRequestsCount`        | int64 → float64 |
| `"subscription_revenue_vnd"` | `snapshot.SubscriptionRevenueVND` | float64         |
| `"net_ai_margin_vnd"`        | `snapshot.NetAIMarginVND`         | float64         |
| `"new_users"`                | `snapshot.NewUsers`               | int → float64   |
| `"ai_error_count"`           | `snapshot.AIErrorCount`           | int64 → float64 |
| `"ai_error_rate"`            | `ai_error_count / ai_requests_count * 100` | float64 |

### Dynamic Query — Metric Entity Mapping

| Metric key                       | Entity field                        | Target |
| -------------------------------- | ----------------------------------- | ------ |
| `total_users`                    | `snapshot.TotalUsers`               | admin  |
| `active_users_dau`               | `snapshot.ActiveUsersDau`           | admin  |
| `total_brands`                   | `snapshot.TotalBrands`              | admin  |
| `pending_review_brands`          | `snapshot.PendingReviewBrands`      | admin  |
| `ai_requests_count`              | `snapshot.AIRequestsCount`          | admin  |
| `subscription_revenue_vnd`       | `snapshot.SubscriptionRevenueVND`   | admin  |
| `net_ai_margin_vnd`              | `snapshot.NetAIMarginVND`           | admin  |
| `new_users`                      | `snapshot.NewUsers`                 | admin  |
| `ai_error_count`                 | `snapshot.AIErrorCount`             | admin  |
| `ai_error_rate`                  | Tính toán                           | admin  |

---

## Tổng hợp Endpoint — Admin

| Phương thức | Endpoint                                                            | Vai trò | Chức năng                                        |
| ----------- | ------------------------------------------------------------------- | ------- | ------------------------------------------------ |
| GET         | `/api/v1/admin/dashboard/kpi-cards`                                 | Admin   | Thẻ KPI tổng quan                                |
| GET         | `/api/v1/admin/dashboard/ai-margin`                                 | Admin   | Phân tích biên lợi nhuận AI                      |
| GET         | `/api/v1/admin/dashboard/payos-reconciliation`                      | Admin   | Đối soát PayOS                                   |
| GET         | `/api/v1/admin/dashboard/overview`                                  | Admin   | Báo cáo tổng quan                                |
| GET         | `/api/v1/admin/dashboard/subscriptions/distribution`                | Admin   | Phân bổ người dùng theo gói subscription         |
| GET         | `/api/v1/admin/dashboard/subscriptions/revenue-breakdown`           | Admin   | Doanh thu subscription theo gói và thời gian     |
| GET         | `/api/v1/admin/dashboard/ai/usage-by-operation`                     | Admin   | Thống kê sử dụng AI theo tác vụ                  |
| GET         | `/api/v1/admin/dashboard/ai/error-breakdown`                        | Admin   | Thống kê lỗi AI theo error_code, provider, model |
| GET         | `/api/v1/admin/dashboard/subscriptions/renewal-stats`               | Admin   | Thống kê tỷ lệ gia hạn subscription              |
| GET         | `/api/v1/admin/dashboard/drilldown/ai-events`                       | Admin   | Drill-down lỗi AI                                |
