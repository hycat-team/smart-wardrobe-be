# Dashboard & Analytics API Specs

Tài liệu đặc tả các API của Hệ thống Dashboard & Analytics cho 3 nhóm đối tượng: **System Admin**, **Brand Manager (B2B)**, và **End User (B2C)**. Tài liệu này được viết dựa trên code tại `internal/modules/dashboard/`.

Xem thêm hằng số tại [constants/dashboard.md](constants/dashboard.md).

---

## Flow 1: System Admin Dashboard

### 1. Lấy thẻ chỉ số KPI tổng quan

- **Endpoint:** `GET /api/v1/admin/dashboard/kpi-cards`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Dữ liệu từ bảng `dashboard_admin_daily_snapshots` (snapshot mới nhất), nếu chưa có snapshot thì fallback query trực tiếp từ domain reader.
- **Response struct:** `AdminKPICardsRes`

| Field                  | JSON                     | Kiểu Go   | Ý nghĩa                                                         |
| ---------------------- | ------------------------ | --------- | --------------------------------------------------------------- |
| TotalUsers             | `totalUsers`             | int       | Tổng số người dùng đã đăng ký                                   |
| ActiveUsersDAU         | `activeUsersDau`         | int       | Số người dùng hoạt động trong ngày (Daily Active Users)         |
| TotalBrands            | `totalBrands`            | int       | Tổng số nhãn hàng đang hoạt động                                |
| PendingReviewBrands    | `pendingReviewBrands`    | int       | Số nhãn hàng đang chờ phê duyệt                                 |
| SubscriptionRevenueVND | `subscriptionRevenueVnd` | float64   | Tổng doanh thu gói Premium (VND), lấy từ `deposit_transactions` |
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

### 2. Phân tích biên lợi nhuận AI Token

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

### 3. Thống kê đối soát giao dịch PayOS

- **Endpoint:** `GET /api/v1/admin/dashboard/payos-reconciliation`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Response struct:** `PayOSReconciliationRes`

| Field             | JSON                | Kiểu Go | Ý nghĩa                                                                     |
| ----------------- | ------------------- | ------- | --------------------------------------------------------------------------- |
| TotalTransactions | `totalTransactions` | int     | Tổng số giao dịch PayOS trong snapshot gần nhất                             |
| SuccessCount      | `successCount`      | int     | Số giao dịch thành công                                                     |
| FailedCount       | `failedCount`       | int     | Số giao dịch thất bại                                                       |
| TotalSuccessValue | `totalSuccessValue` | float64 | Tổng giá trị VND của giao dịch thành công (lấy từ `subscriptionRevenueVND`) |

---

### 4. Báo cáo tổng quan hệ thống

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

## Flow 2: Brand Manager Dashboard (B2B Portal)

### 1. Lấy thẻ KPI Brand

- **Endpoint:** `GET /api/v1/brand/dashboard/kpi-cards`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Query Parameters:**
    - `brandId` (string, **required**) — UUID của nhãn hàng.
- **Response struct:** `BrandKPICardsRes`

| Field                     | JSON                        | Kiểu Go   | Ý nghĩa                                       |
| ------------------------- | --------------------------- | --------- | --------------------------------------------- |
| BrandID                   | `brandId`                   | uuid.UUID | ID của nhãn hàng                              |
| TotalMembers              | `totalMembers`              | int       | Tổng số hội viên loyalty                      |
| PointsIssued              | `pointsIssued`              | int       | Số điểm đã phát hành                          |
| PointsRedeemed            | `pointsRedeemed`            | int       | Số điểm đã quy đổi                            |
| ActiveVouchers            | `activeVouchers`            | int       | Số voucher đang hoạt động                     |
| VouchersRedeemed          | `vouchersRedeemed`          | int       | Số voucher đã quy đổi                         |
| DigitalSampleVotesCount   | `digitalSampleVotesCount`   | int       | Số lượt vote mẫu thử Digital Sample           |
| DigitalSampleAvgRating    | `digitalSampleAvgRating`    | float64   | Điểm đánh giá trung bình (thang 0–5)          |
| OpenCSTickets             | `openCsTickets`             | int       | Số ticket CS đang mở                          |
| CSFirstResponseAvgSeconds | `csFirstResponseAvgSeconds` | int       | Thời gian phản hồi đầu tiên trung bình (giây) |
| UpdatedAt                 | `updatedAt`                 | time.Time | Thời điểm cập nhật gần nhất                   |

---

### 2. Phân tích nợ điểm hết hạn

- **Endpoint:** `GET /api/v1/brand/dashboard/financials/points-liability`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Query Parameters:**
    - `brandId` (string, **required**) — UUID của nhãn hàng.
- **Cơ chế Column-Level RBAC:** Tài khoản `Staff` bị redact trường `pointsLiabilityValueVnd` (trả về `0`).
- **Response struct:** `BrandPointsLiabilityRes`

| Field                   | JSON                      | Kiểu Go   | Ý nghĩa                                        |
| ----------------------- | ------------------------- | --------- | ---------------------------------------------- |
| BrandID                 | `brandId`                 | uuid.UUID | ID của nhãn hàng                               |
| PointsExpiring30d       | `pointsExpiring30d`       | int       | Số điểm sắp hết hạn trong 30 ngày              |
| PointsLiabilityValueVND | `pointsLiabilityValueVnd` | float64   | Giá trị quy đổi (VND). **Staff luôn thấy `0`** |

---

### 3. Phân tích đánh giá mẫu thử Digital Sample Lab

- **Endpoint:** `GET /api/v1/brand/dashboard/operations/digital-sample-lab/:itemId`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Path Parameters:**
    - `itemId` (string, **required**) — UUID sản phẩm mẫu thử.
- **Query Parameters:**
    - `brandId` (string, **required**) — UUID của nhãn hàng.
- **Response struct:** `DigitalSampleLabAnalyticsRes`

| Field               | JSON                  | Kiểu Go              | Ý nghĩa                                            |
| ------------------- | --------------------- | -------------------- | -------------------------------------------------- |
| ItemID              | `itemId`              | uuid.UUID            | ID của mẫu thử                                     |
| VotesCount          | `votesCount`          | int                  | Số lượt vote                                       |
| AvgRating           | `avgRating`           | float64              | Điểm rating trung bình (0–5)                       |
| FeedbackSentiment   | `feedbackSentiment`   | FeedbackSentiment    | Sentiment: `"positive"`, `"neutral"`, `"negative"` |
| TopPairedCategories | `topPairedCategories` | []PairedCategoryStat | Danh sách danh mục thường phối kèm                 |

**Cấu trúc `PairedCategoryStat`:**

| Field        | JSON           | Kiểu Go | Ý nghĩa                                   |
| ------------ | -------------- | ------- | ----------------------------------------- |
| CategoryName | `categoryName` | string  | Tên danh mục thời trang (VD: "Quần Jean") |
| PairCount    | `pairCount`    | int     | Số lần xuất hiện trong các bộ phối đồ     |

---

## Flow 3: End User Wardrobe Utility (B2C)

### 1. Thống kê phân tích tủ đồ cá nhân

- **Endpoint:** `GET /api/v1/me/dashboard/wardrobe-insights`
- **Tác nhân (Actor):** Người dùng B2C (`roleslug.User`).
- **Mô tả:** Cơ chế **lazy cache** — snapshot < 1h thì đọc từ cache, nếu không tính real-time và write-back.
- **Response struct:** `UserWardrobeInsightsRes`

| Field                   | JSON                      | Kiểu Go                | Ý nghĩa                            | Cách tính                                                                                                                                                               |
| ----------------------- | ------------------------- | ---------------------- | ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TotalItems              | `totalItems`              | int                    | Tổng số món đồ trong tủ            | `SELECT COUNT(*) FROM wardrobe_items WHERE user_id=? AND is_deleted=false`                                                                                              |
| TotalWardrobeValueVND   | `totalWardrobeValueVnd`   | float64                | Tổng giá trị tài chính tủ đồ (VND) | `SELECT COALESCE(SUM(purchase_price),0) FROM wardrobe_items WHERE purchase_price IS NOT NULL`                                                                           |
| UnderutilizedItemsCount | `underutilizedItemsCount` | int                    | Số món đồ không mặc >30 ngày       | `SELECT COUNT(*) FROM wardrobe_items WHERE last_used_at IS NULL OR last_used_at < 30 ngày trước`                                                                        |
| UnderutilizedItems      | `underutilizedItems`      | []UnderutilizedItemDTO | Danh sách đồ lãng phí              | Cùng điều kiện, lấy `last_used_at ASC`, giới hạn 10 món                                                                                                                 |
| AIQuotaRemaining        | `aiQuotaRemaining`        | int                    | Số lượt AI Stylist còn lại         | Từ snapshot entity (nếu có stale), mặc định 0                                                                                                                           |
| ExpiringVouchersCount   | `expiringVouchersCount`   | int                    | Số voucher sắp hết hạn             | Từ snapshot entity (nếu có stale), mặc định 0                                                                                                                           |

**Cấu trúc `UnderutilizedItemDTO`:**

| Field            | JSON               | Kiểu Go   | Ý nghĩa                                                                       |
| ---------------- | ------------------ | --------- | ----------------------------------------------------------------------------- |
| ItemID           | `itemId`           | uuid.UUID | ID của món đồ                                                                 |
| Name             | `name`             | string    | Tên món đồ (từ fashion_item.style hoặc category.name)                         |
| CategoryName     | `categoryName`     | string    | Tên danh mục (từ fashion_item.category.name)                                  |
| PurchasePriceVND | `purchasePriceVnd` | float64   | Giá mua (VND)                                                                 |
| LastWornDaysAgo  | `lastWornDaysAgo`  | int       | Số ngày từ lần cuối mặc (`now - last_used_at`, mặc định 30 nếu chưa từng mặc) |
| ImageURL         | `imageUrl`         | string    | URL ảnh (Cloudinary, từ fashion_item.image_url)                               |

---

## Flow 4: Dynamic Aggregation Query Engine

### 1. Truy vấn tổng hợp linh hoạt

- **Endpoint:** `POST /api/v1/dashboard/analytics/query`
- **Tác nhân (Actor):** Nhiều role (Admin, Brand Member, User) — RBAC tự động theo `targetDashboard`.
- **Mô tả:** API truy vấn tổng hợp đa chiều. Backend aggregate daily snapshots theo granularity yêu cầu. Tự động điều chỉnh:
    - `granularity = "hour"` + date range > 365 ngày → tự động set thành `"day"`
    - date range > 730 ngày + `"hour"`/`"day"` → set thành `"month"`
    - Date range bị giới hạn tối đa 730 ngày

    **RBAC:**
    - `"admin"`: chỉ Admin
    - `"brand"`: kiểm tra quyền user với brandId qua JWT
    - `"user"`: scope theo user hiện tại

- **Request struct:** `DynamicQueryReq`

| Field               | JSON                  | Kiểu Go         | Bắt buộc             | Ý nghĩa                                                                                                                                                                   |
| ------------------- | --------------------- | --------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TargetDashboard     | `targetDashboard`     | TargetDashboard | ✅                   | Giá trị: `"admin"`, `"brand"`, `"user"`. Xem [constants/dashboard.md#2-target-dashboard](constants/dashboard.md#2-target-dashboard)                                       |
| BrandID             | `brandId`             | string (UUID)   | Nếu target=`"brand"` | UUID của nhãn hàng. Bỏ qua nếu target không phải brand                                                                                                                    |
| Metrics             | `metrics`             | []string        | ✅                   | Danh sách metric key cần query. Giá trị hợp lệ xem [constants/dashboard.md#4-metric-key](constants/dashboard.md#4-metric-key). VD: `["total_users", "ai_requests_count"]` |
| Timeframe           | `timeframe`           | object          | ✅                   | `{ "fromDate": "2026-07-01", "toDate": "2026-07-23" }`. Định dạng `YYYY-MM-DD`. Tối đa 730 ngày                                                                           |
| Granularity         | `granularity`         | string          | ✅                   | `"hour"`, `"day"`, `"week"`, `"month"`. Xem [constants/dashboard.md#3-granularity](constants/dashboard.md#3-granularity). Backend thực sự aggregate data theo granularity này |
| CompareWithPrevious | `compareWithPrevious` | bool            | Optional             | `true`: tính delta % so với chu kỳ trước đó. Mặc định: `false`                                                                                                            |

- **Response struct:** `DynamicQueryRes`

| Field           | JSON              | Kiểu Go              | Ý nghĩa                                                       |
| --------------- | ----------------- | -------------------- | ------------------------------------------------------------- |
| TargetDashboard | `targetDashboard` | TargetDashboard      | Echo từ request (có thể đã lowercase)                         |
| Granularity     | `granularity`     | Granularity          | Granularity thực tế (có thể đã auto-downgrade)                |
| Series          | `series`          | []TimeSeriesPointDTO | Mảng điểm dữ liệu time-series                                 |
| Comparison      | `comparison`      | \*ComparisonDTO      | Chỉ xuất hiện nếu `compareWithPrevious=true` (có `omitempty`) |

**Cấu trúc `TimeSeriesPointDTO`:**

| Field     | JSON        | Kiểu Go               | Ý nghĩa                  |
| --------- | ----------- | --------------------- | ------------------------ |
| Timestamp | `timestamp` | time.Time             | Mốc thời gian            |
| Metrics   | `metrics`   | map[MetricKey]float64 | Map metric_key → giá trị |

**Cấu trúc `ComparisonDTO`:**

| Field                  | JSON                     | Kiểu Go               | Ý nghĩa                     |
| ---------------------- | ------------------------ | --------------------- | --------------------------- |
| PreviousPeriodFrom     | `previousPeriodFrom`     | string                | Ngày bắt đầu kỳ so sánh     |
| PreviousPeriodTo       | `previousPeriodTo`       | string                | Ngày kết thúc kỳ so sánh    |
| MetricDeltaPercentages | `metricDeltaPercentages` | map[MetricKey]float64 | Map metric_key → % thay đổi |

**MetricKey thực tế được hỗ trợ (theo từng target):**

#### Admin (`targetDashboard = "admin"`)

Backend hiện chỉ map 5 metric sau. Các metric khác gửi vào sẽ bị bỏ qua và nhận giá trị `0`.

| Metric                       | Mapped từ field                   | Kiểu Go gốc     |
| ---------------------------- | --------------------------------- | --------------- |
| `"total_users"`              | `snapshot.TotalUsers`             | int → float64   |
| `"active_users_dau"`         | `snapshot.ActiveUsersDau`         | int → float64   |
| `"total_brands"`             | `snapshot.TotalBrands`            | int → float64   |
| `"pending_review_brands"`    | `snapshot.PendingReviewBrands`    | int → float64   |
| `"ai_requests_count"`        | `snapshot.AIRequestsCount`        | int64 → float64 |
| `"subscription_revenue_vnd"` | `snapshot.SubscriptionRevenueVND` | float64         |
| `"net_ai_margin_vnd"`        | `snapshot.NetAIMarginVND`         | float64         |

#### Brand (`targetDashboard = "brand"`)

| Metric                       | Mapped từ field                    | Kiểu Go gốc   |
| ---------------------------- | ---------------------------------- | ------------- |
| `"total_members"`            | `snapshot.TotalMembers`            | int → float64 |
| `"points_issued"`            | `snapshot.PointsIssued`            | int → float64 |
| `"points_redeemed"`          | `snapshot.PointsRedeemed`          | int → float64 |
| `"active_vouchers"`          | `snapshot.ActiveVouchers`          | int → float64 |
| `"vouchers_redeemed"`        | `snapshot.VouchersRedeemed`        | int → float64 |
| `"digital_sample_votes_count"` | `snapshot.DigitalSampleVotesCount` | int → float64 |
| `"digital_sample_avg_rating"`  | `snapshot.DigitalSampleAvgRating`  | float64       |
| `"open_cs_tickets"`          | `snapshot.OpenCSTickets`           | int → float64 |
| `"cs_first_response_avg_seconds"` | `snapshot.CSFirstResponseAvgSeconds` | int → float64 |
| `"points_expiring_30d"`      | `snapshot.PointsExpiring30d`       | int → float64 |
| `"points_liability_value_vnd"` | `snapshot.PointsLiabilityValueVND` | float64       |

#### User (`targetDashboard = "user"`)

Chỉ 2 metric sau được backend xử lý. **Chỉ trả về 1 điểm dữ liệu duy nhất** (không phải time-series) vì nguồn là bảng `dashboard_user_wardrobe_summaries` dạng point-in-time.

| Metric                       | Mapped từ field                 | Kiểu Go gốc   |
| ---------------------------- | ------------------------------- | ------------- |
| `"total_wardrobe_items"`     | `summary.TotalItems`            | int → float64 |
| `"total_wardrobe_value_vnd"` | `summary.TotalWardrobeValueVND` | float64       |

---

## Flow 5: Drill-Down Traceability

### 1. Truy vết sự kiện lỗi AI (Admin)

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

### 2. Truy vết đánh giá mẫu thử (Brand)

- **Endpoint:** `GET /api/v1/brand/dashboard/drilldown/sample-lab-feedbacks`
- **Tác nhân (Actor):** Thành viên nhãn hàng / Quản trị viên.
- **Query Parameters:**
    - `itemId` (string, **required**) — UUID sản phẩm mẫu thử.
    - `page` (int, optional, default=1) — Số trang.
    - `limit` (int, optional, default=20, max=100) — Số bản ghi mỗi trang.
- **Response struct:** `DrillDownSampleFeedbackListRes` (alias của `PaginationResult[DrillDownSampleFeedbackDTO]`)

**Cấu trúc `DrillDownSampleFeedbackDTO`:**

| Field      | JSON         | Kiểu Go   | Ý nghĩa                    |
| ---------- | ------------ | --------- | -------------------------- |
| FeedbackID | `feedbackId` | string    | ID của feedback            |
| ItemID     | `itemId`     | uuid.UUID | ID sản phẩm mẫu thử        |
| UserID     | `userId`     | uuid.UUID | ID người dùng gửi feedback |
| Rating     | `rating`     | float64   | Điểm đánh giá (0–5)        |
| Comment    | `comment`    | string    | Nội dung nhận xét          |
| CreatedAt  | `createdAt`  | time.Time | Thời điểm gửi              |

---

## Phụ lục: Trace toàn bộ Metric

### Admin KPI Cards (`GET /api/v1/admin/dashboard/kpi-cards`)

| Field                    | Nguồn dữ liệu          | Câu truy vấn / Cách tính                                                                                                                                                                                                                 |
| ------------------------ | ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `totalUsers`             | `users` table          | `SELECT COUNT(*) FROM users WHERE is_deleted = false`                                                                                                                                                                                    |
| `activeUsersDau`         | `users` table          | `SELECT COUNT(*) FROM users WHERE is_deleted = false AND updated_at BETWEEN [00:00 hôm nay, 00:00 ngày mai)`. **Lưu ý:** Đây là user có `updated_at` trong ngày (đăng nhập, update profile,...), không phải DAU thực thụ từ activity log |
| `totalBrands`            | `brands` table         | `SELECT COUNT(*) FROM brands`                                                                                                                                                                                                            |
| `pendingReviewBrands`    | `brands` table         | `SELECT COUNT(*) FROM brands WHERE status = 'PENDING_REVIEW'`                                                                                                                                                                            |
| `subscriptionRevenueVnd` | `deposit_transactions` | `SELECT COALESCE(SUM(amount),0) FROM deposit_transactions WHERE transaction_type='direct_purchase' AND created_at IN [ngày]` — lấy từ Aggregator Worker chạy 15p/lần                                                                     |
| `netAiMarginVnd`         | Tính toán              | `subscriptionRevenueVnd - (aiActualCostMicroVnd / 1_000_000)`                                                                                                                                                                            |
| `payOSSuccessCount`      | `deposit_transactions` | `SELECT COUNT(*) FROM deposit_transactions WHERE transaction_type='direct_purchase' AND status=1 (Success) AND created_at IN [ngày]`                                                                                                     |

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
| `failedCount`       | Tính toán                         | `totalTransactions - successCount` |
| `totalSuccessValue` | `dashboard_admin_daily_snapshots` | `snapshot.SubscriptionRevenueVND`  |

### Brand KPI Cards (`GET /api/v1/brand/dashboard/kpi-cards`)

| Field                       | Nguồn                        | Câu truy vấn                                                                                                                                           |
| --------------------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `totalMembers`              | `brand_customers`            | `SELECT COUNT(*) FROM brand_customers WHERE brand_id=? AND status='ACTIVE'`                                                                            |
| `pointsIssued`              | `loyalty_point_transactions` | `SELECT COALESCE(SUM(points),0) FROM loyalty_point_transactions WHERE transaction_type IN ('EARN','CREDIT','ISSUED') AND brand_id=?`                   |
| `pointsRedeemed`            | `loyalty_point_transactions` | `SELECT COALESCE(SUM(points),0) FROM loyalty_point_transactions WHERE transaction_type IN ('REDEEM','DEBIT','REDEEMED') AND brand_id=?`                |
| `activeVouchers`            | `brand_benefits`             | `SELECT COUNT(*) FROM brand_benefits WHERE brand_id=? AND status IN ('ACTIVE','active')`                                                               |
| `vouchersRedeemed`          | `benefit_redemptions`        | `SELECT COUNT(*) FROM benefit_redemptions WHERE brand_id=?`                                                                                            |
| `digitalSampleVotesCount`   | `digital_sample_responses`   | `SELECT COUNT(*) FROM digital_sample_responses JOIN brand_items ON brand_items.id=digital_sample_responses.brand_item_id WHERE brand_items.brand_id=?` |
| `digitalSampleAvgRating`    | `digital_sample_responses`   | `SELECT COALESCE(AVG(rating),0) FROM digital_sample_responses [cùng join] WHERE rating IS NOT NULL`                                                    |
| `openCsTickets`             | `brand_conversations`        | `SELECT COUNT(*) FROM brand_conversations WHERE brand_id=? AND status='OPEN'`                                                                          |
| `csFirstResponseAvgSeconds` | `brand_conversations`        | `SELECT COALESCE(AVG(first_response_seconds),0) FROM brand_conversations WHERE first_response_seconds IS NOT NULL`                                     |

### Points Liability (`GET /api/v1/brand/dashboard/financials/points-liability`)

| Field                     | Nguồn                | Câu truy vấn                                                                                                                                      |
| ------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pointsExpiring30d`       | `loyalty_point_lots` | `SELECT COALESCE(SUM(remaining_points),0) FROM loyalty_point_lots WHERE brand_id=? AND status='ACTIVE' AND expires_at BETWEEN [today, today+30d]` |
| `pointsLiabilityValueVnd` | Tính toán            | `pointsExpiring30d * 1000` (giả định 1 điểm = 1000 VND)                                                                                           |

### User Wardrobe Insights (`GET /api/v1/me/dashboard/wardrobe-insights`)

| Field                     | Nguồn                              | Câu truy vấn                                                                                                                                                                |
| ------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `totalItems`              | `wardrobe_items`                   | `SELECT COUNT(*) FROM wardrobe_items WHERE user_id=? AND is_deleted=false`                                                                                                  |
| `totalWardrobeValueVnd`   | `wardrobe_items`                   | `SELECT COALESCE(SUM(purchase_price),0) FROM wardrobe_items WHERE purchase_price IS NOT NULL`                                                                               |
| `underutilizedItemsCount` | `wardrobe_items`                   | `SELECT COUNT(*) FROM wardrobe_items WHERE last_used_at IS NULL OR last_used_at < 30 ngày trước`                                                                            |
| `underutilizedItems`      | `wardrobe_items` + `fashion_items` | Query lấy items có `last_used_at IS NULL OR <30 ngày`, kết hợp `fashion_items` để lấy tên, danh mục, ảnh. Sắp xếp `last_used_at ASC`, giới hạn 10 items                     |
| `aiQuotaRemaining`        | Snapshot cache                     | Từ bảng `dashboard_user_wardrobe_summaries` (được preserve khi cache stale), mặc định `0`                                                                                   |
| `expiringVouchersCount`   | Snapshot cache                     | Từ bảng `dashboard_user_wardrobe_summaries` (được preserve khi cache stale), mặc định `0`                                                                                   |

### Dynamic Query (`POST /api/v1/dashboard/analytics/query`)

Mỗi metric key map trực tiếp vào field của snapshot entity tương ứng:

| Metric key                       | Entity field                        | Target |
| -------------------------------- | ----------------------------------- | ------ |
| `total_users`                    | `snapshot.TotalUsers`               | admin  |
| `active_users_dau`               | `snapshot.ActiveUsersDau`           | admin  |
| `total_brands`                   | `snapshot.TotalBrands`              | admin  |
| `pending_review_brands`          | `snapshot.PendingReviewBrands`      | admin  |
| `ai_requests_count`              | `snapshot.AIRequestsCount`          | admin  |
| `subscription_revenue_vnd`       | `snapshot.SubscriptionRevenueVND`   | admin  |
| `net_ai_margin_vnd`              | `snapshot.NetAIMarginVND`           | admin  |
| `total_members`                  | `snapshot.TotalMembers`             | brand  |
| `points_issued`                  | `snapshot.PointsIssued`             | brand  |
| `points_redeemed`                | `snapshot.PointsRedeemed`           | brand  |
| `active_vouchers`                | `snapshot.ActiveVouchers`           | brand  |
| `vouchers_redeemed`              | `snapshot.VouchersRedeemed`         | brand  |
| `digital_sample_votes_count`     | `snapshot.DigitalSampleVotesCount`  | brand  |
| `digital_sample_avg_rating`      | `snapshot.DigitalSampleAvgRating`   | brand  |
| `open_cs_tickets`                | `snapshot.OpenCSTickets`            | brand  |
| `cs_first_response_avg_seconds`  | `snapshot.CSFirstResponseAvgSeconds`| brand  |
| `points_expiring_30d`            | `snapshot.PointsExpiring30d`        | brand  |
| `points_liability_value_vnd`     | `snapshot.PointsLiabilityValueVND`  | brand  |
| `total_wardrobe_items`           | `summary.TotalItems`                | user   |
| `total_wardrobe_value_vnd`       | `summary.TotalWardrobeValueVND`     | user   |

---

## Tổng hợp Endpoint

| Phương thức | Endpoint                                                        | Vai trò      | Chức năng                   |
| ----------- | --------------------------------------------------------------- | ------------ | --------------------------- |
| GET         | `/api/v1/admin/dashboard/kpi-cards`                             | Admin        | Thẻ KPI tổng quan           |
| GET         | `/api/v1/admin/dashboard/ai-margin`                             | Admin        | Phân tích biên lợi nhuận AI |
| GET         | `/api/v1/admin/dashboard/payos-reconciliation`                  | Admin        | Đối soát PayOS              |
| GET         | `/api/v1/admin/dashboard/overview`                              | Admin        | Báo cáo tổng quan           |
| GET         | `/api/v1/admin/dashboard/drilldown/ai-events`                   | Admin        | Drill-down lỗi AI           |
| GET         | `/api/v1/brand/dashboard/kpi-cards`                             | Brand Member | Thẻ KPI Brand               |
| GET         | `/api/v1/brand/dashboard/financials/points-liability`           | Brand Member | Nợ điểm hết hạn (RBAC)      |
| GET         | `/api/v1/brand/dashboard/operations/digital-sample-lab/:itemId` | Brand Member | Phân tích mẫu thử           |
| GET         | `/api/v1/brand/dashboard/drilldown/sample-lab-feedbacks`        | Brand Member | Drill-down feedback         |
| GET | `/api/v1/me/dashboard/wardrobe-insights` | User | Phân tích tủ đồ cá nhân |
| POST        | `/api/v1/dashboard/analytics/query`                             | Multiple     | Dynamic Query Engine        |
