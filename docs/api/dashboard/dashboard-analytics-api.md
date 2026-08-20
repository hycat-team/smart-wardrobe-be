# Dashboard Analytics API

Tài liệu đặc tả các API Analytics dành cho **End User (B2C)** và **Dynamic Aggregation Query Engine**. Bao gồm các endpoints từ dashboard-api.md.

Xem thêm hằng số tại [constants.md](constants.md).

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
| ExpiringVouchersCount   | `expiringVouchersCount`   | int                    | Số voucher sắp hết hạn             | Live query từ `benefit_redemptions` (cache miss path), hoặc từ snapshot (cache hit)                                                                                     |

**Cấu trúc `UnderutilizedItemDTO`:**

| Field            | JSON               | Kiểu Go   | Ý nghĩa                                                                       |
| ---------------- | ------------------ | --------- | ----------------------------------------------------------------------------- |
| ItemID           | `itemId`           | uuid.UUID | ID của món đồ                                                                 |
| Name             | `name`             | string    | Tên món đồ (từ fashion_item.style hoặc category.name)                         |
| CategoryName     | `categoryName`     | string    | Tên danh mục (từ fashion_item.category.name)                                  |
| PurchasePriceVND | `purchasePriceVnd` | float64   | Giá mua (VND)                                                                 |
| LastWornDaysAgo  | `lastWornDaysAgo`  | int       | Số ngày từ lần cuối mặc (`now - last_used_at`, mặc định 30 nếu chưa từng mặc) |
| ImageURL         | `imageUrl`         | string    | URL ảnh (Cloudinary, từ fashion_item.image_url)                               |

### 2. Lấy phân bổ danh mục tủ đồ

- **Endpoint:** `GET /api/v1/me/dashboard/wardrobe/category-distribution`
- **Tác nhân (Actor):** Người dùng B2C (`roleslug.User`).
- **Mô tả:** Thống kê số lượng và tỷ lệ phần trăm các món đồ trong tủ của người dùng, phân nhóm theo danh mục thời trang (Áo Sơ Mi, Quần Jean, Giày, v.v.). Scope tự động theo user từ JWT.
- **Response struct:** `WardrobeCategoryDistributionRes`
- **Nguồn dữ liệu:** Join 3 bảng `wardrobe_items` → `fashion_items` → `categories`, `WHERE user_id=? AND is_deleted=false`, `GROUP BY category_id`.

| Field          | JSON           | Kiểu                      | Ý nghĩa                                   |
| -------------- | -------------- | ------------------------- | ----------------------------------------- |
| TotalItems     | `totalItems`   | int                       | Tổng số món đồ trong tủ                   |
| Categories     | `categories`   | []CategoryDistributionDTO | Phân bổ theo danh mục                     |
| ├ CategoryID   | `categoryId`   | uuid.UUID                 | UUID của danh mục                         |
| ├ CategoryName | `categoryName` | string                    | Tên danh mục (ví dụ: Áo Sơ Mi, Quần Jean) |
| ├ ItemCount    | `itemCount`    | int                       | Số lượng món đồ thuộc danh mục            |
| └ Percentage   | `percentage`   | float64                   | Tỷ lệ phần trăm so với tổng               |

---

## Flow 4: Dynamic Aggregation Query Engine

### 1. Truy vấn tổng hợp linh hoạt

- **Endpoint:** `POST /api/v1/dashboard/analytics/query`
- **Tác nhân (Actor):** Nhiều role (Admin, Brand Member, User) — RBAC tự động theo `targetDashboard`.
- **Mô tả:** API truy vấn tổng hợp đa chiều. Backend aggregate daily snapshots theo granularity yêu cầu. Tự động điều chỉnh:
    - Granularity không hợp lệ → request validation trả 400
    - Date range > 730 ngày → request validation trả 400

    **RBAC:**
    - `"admin"`: chỉ Admin
    - `"brand"`: kiểm tra quyền user với brandId qua JWT
    - `"user"`: scope theo user hiện tại

- **Request struct:** `DynamicQueryReq`

| Field               | JSON                  | Kiểu Go         | Bắt buộc             | Ý nghĩa                                                                                                                                                                   |
| ------------------- | --------------------- | --------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TargetDashboard     | `targetDashboard`     | TargetDashboard | ✅                   | Giá trị: `"admin"`, `"brand"`, `"user"`. Xem [constants.md#2-target-dashboard](constants.md#2-target-dashboard)                                                           |
| BrandID             | `brandId`             | string (UUID)   | Nếu target=`"brand"` | UUID của nhãn hàng. Bỏ qua nếu target không phải brand                                                                                                                    |
| Metrics             | `metrics`             | []string        | ✅                   | Danh sách metric key cần query. Giá trị hợp lệ xem [constants.md#4-metric-key](constants.md#4-metric-key). VD: `["total_users", "ai_requests_count"]`                     |
| Timeframe           | `timeframe`           | object          | ✅                   | `{ "fromDate": "2026-07-01", "toDate": "2026-07-23" }`. Định dạng `YYYY-MM-DD`. Tối đa 730 ngày. Nếu vượt quá → 400 BadRequest                                           |
| Granularity         | `granularity`         | string          | ✅                   | `"day"`, `"week"`, `"month"`, `"year"`. Xem [constants.md#3-granularity](constants.md#3-granularity). Backend aggregate daily snapshots theo granularity yêu cầu (week/month/year) |
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

---

## Metric Key Reference

### User Metrics (`targetDashboard = "user"`)

Chỉ 2 metric sau được backend xử lý. **Chỉ trả về 1 điểm dữ liệu duy nhất** (không phải time-series) vì nguồn là bảng `dashboard_user_wardrobe_summaries` dạng point-in-time.

| Metric                       | Mapped từ field                 | Kiểu Go gốc   |
| ---------------------------- | ------------------------------- | ------------- |
| `"total_wardrobe_items"`     | `summary.TotalItems`            | int → float64 |
| `"total_wardrobe_value_vnd"` | `summary.TotalWardrobeValueVND` | float64       |

### Admin Metrics (`targetDashboard = "admin"`)

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
| `"ai_error_rate"`            | Tính toán                         | float64         |

### Brand Metrics (`targetDashboard = "brand"`)

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
| `"new_customers"`            | `brand_customers`                  | Live query    |
| `"customer_claim_rate"`      | `brand_customers`                  | Live query    |
| `"benefit_redemption_count"` | `benefit_redemptions`              | Live query    |
| `"brand_item_count"`         | `brand_items`                      | Live query    |
| `"brand_item_count_by_type"` | `brand_items`                      | Live query    |

---

## Phụ lục: Trace toàn bộ Metric — Analytics

### User Wardrobe Insights (`GET /api/v1/me/dashboard/wardrobe-insights`)

| Field                     | Nguồn                              | Câu truy vấn                                                                                                                                                                |
| ------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `totalItems`              | `wardrobe_items`                   | `SELECT COUNT(*) FROM wardrobe_items WHERE user_id=? AND is_deleted=false`                                                                                                  |
| `totalWardrobeValueVnd`   | `wardrobe_items`                   | `SELECT COALESCE(SUM(purchase_price),0) FROM wardrobe_items WHERE purchase_price IS NOT NULL`                                                                               |
| `underutilizedItemsCount` | `wardrobe_items`                   | `SELECT COUNT(*) FROM wardrobe_items WHERE last_used_at IS NULL OR last_used_at < 30 ngày trước`                                                                            |
| `underutilizedItems`      | `wardrobe_items` + `fashion_items` | Query lấy items có `last_used_at IS NULL OR <30 ngày`, kết hợp `fashion_items` để lấy tên, danh mục, ảnh. Sắp xếp `last_used_at ASC`, giới hạn 10 items                     |
| `expiringVouchersCount`   | `benefit_redemptions`              | Live query `SELECT COUNT(*) FROM benefit_redemptions WHERE user_id=? AND status='REDEEMED' AND expires_at BETWEEN now() AND now()+30d` (cache miss path)                   |

### Dynamic Query — Full Metric Entity Mapping (`POST /api/v1/dashboard/analytics/query`)

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
| `new_customers`                  | `brand_customers`                   | brand  |
| `customer_claim_rate`            | `brand_customers`                   | brand  |
| `benefit_redemption_count`       | `benefit_redemptions`               | brand  |
| `brand_item_count`               | `brand_items`                       | brand  |
| `brand_item_count_by_type`       | `brand_items`                       | brand  |
| `total_wardrobe_items`           | `summary.TotalItems`                | user   |
| `total_wardrobe_value_vnd`       | `summary.TotalWardrobeValueVND`     | user   |

---

## Tổng hợp Endpoint — Analytics

| Phương thức | Endpoint                                        | Vai trò | Chức năng                   |
| ----------- | ----------------------------------------------- | ------- | --------------------------- |
| GET         | `/api/v1/me/dashboard/wardrobe-insights`        | User    | Phân tích tủ đồ cá nhân    |
| GET         | `/api/v1/me/dashboard/wardrobe/category-distribution` | User | Phân bổ danh mục tủ đồ     |
| POST        | `/api/v1/dashboard/analytics/query`             | Multiple | Dynamic Query Engine        |
