# Dashboard & Analytics — Hằng số (Constants)

Tài liệu các hằng số (constants) được sử dụng trong Hệ thống Dashboard & Analytics. Giá trị lấy từ code tại `internal/modules/dashboard/domain/constants/`.

---

## 1. Feedback Sentiment

**Source:** `internal/modules/dashboard/domain/constants/feedbacksentiment`

| Tên hằng | Giá trị (string) | JSON | Ý nghĩa |
|----------|-----------------|------|---------|
| `Positive` | `"positive"` | `"positive"` | Phản hồi tích cực |
| `Neutral` | `"neutral"` | `"neutral"` | Phản hồi trung tính |
| `Negative` | `"negative"` | `"negative"` | Phản hồi tiêu cực |

---

## 2. Target Dashboard

**Source:** `internal/modules/dashboard/domain/constants/targetdashboard`

Giá trị của field `targetDashboard` trong request body `DynamicQueryReq`.

| Tên hằng | Giá trị (string) | Phạm vi RBAC |
|----------|-----------------|-------------|
| `Admin` | `"admin"` | Chỉ `roleslug.Admin` |
| `Brand` | `"brand"` | `brandmemberrole.Owner`, `Staff` |
| `User` | `"user"` | `roleslug.User` (scope tự động) |

---

## 3. Granularity

**Source:** `internal/modules/dashboard/domain/constants/granularity`

Giá trị của field `granularity` trong `DynamicQueryReq`.

| Tên hằng | Giá trị (string) | JSON | Ý nghĩa |
|----------|-----------------|------|---------|
| `Hour` | `"hour"` | `"hour"` | Theo giờ (trả raw daily nếu không có hourly snapshot) |
| `Day` | `"day"` | `"day"` | Theo ngày (mặc định nếu không gửi) |
| `Week` | `"week"` | `"week"` | Theo tuần — backend aggregate daily → weekly |
| `Month` | `"month"` | `"month"` | Theo tháng — backend aggregate daily → monthly |

**Tự động điều chỉnh (auto-downgrade) khi date range quá lớn:**
- `"hour"` + >365 ngày → tự động set thành `"day"`
- (`"hour"` hoặc `"day"`) + >730 ngày → tự động set thành `"month"`

---

## 4. Metric Key

**Source:** `internal/modules/dashboard/domain/constants/metrickey`

Danh sách Metric Key hợp lệ trong field `metrics` của `DynamicQueryReq`.

| Nhóm | Tên hằng Go | Giá trị (string) | JSON |
|------|------------|-----------------|------|
| Admin | `TotalUsers` | `"total_users"` | `"total_users"` |
| Admin | `ActiveUsersDau` | `"active_users_dau"` | `"active_users_dau"` |
| Admin | `TotalBrands` | `"total_brands"` | `"total_brands"` |
| Admin | `PendingReviewBrands` | `"pending_review_brands"` | `"pending_review_brands"` |
| Admin | `AIRequestsCount` | `"ai_requests_count"` | `"ai_requests_count"` |
| Admin | `SubscriptionRevenueVND` | `"subscription_revenue_vnd"` | `"subscription_revenue_vnd"` |
| Admin | `NetAIMarginVND` | `"net_ai_margin_vnd"` | `"net_ai_margin_vnd"` |
| Brand | `TotalMembers` | `"total_members"` | `"total_members"` |
| Brand | `PointsIssued` | `"points_issued"` | `"points_issued"` |
| Brand | `PointsRedeemed` | `"points_redeemed"` | `"points_redeemed"` |
| Brand | `ActiveVouchers` | `"active_vouchers"` | `"active_vouchers"` |
| Brand | `VouchersRedeemed` | `"vouchers_redeemed"` | `"vouchers_redeemed"` |
| Brand | `DigitalSampleVotesCount` | `"digital_sample_votes_count"` | `"digital_sample_votes_count"` |
| Brand | `DigitalSampleAvgRating` | `"digital_sample_avg_rating"` | `"digital_sample_avg_rating"` |
| Brand | `OpenCSTickets` | `"open_cs_tickets"` | `"open_cs_tickets"` |
| Brand | `CSFirstResponseAvgSeconds` | `"cs_first_response_avg_seconds"` | `"cs_first_response_avg_seconds"` |
| Brand | `PointsExpiring30d` | `"points_expiring_30d"` | `"points_expiring_30d"` |
| Brand | `PointsLiabilityValueVND` | `"points_liability_value_vnd"` | `"points_liability_value_vnd"` |
| User | `TotalWardrobeItems` | `"total_wardrobe_items"` | `"total_wardrobe_items"` |
| User | `TotalWardrobeValueVND` | `"total_wardrobe_value_vnd"` | `"total_wardrobe_value_vnd"` |

---

## 5. Hằng số ứng dụng

**Source:** `internal/modules/dashboard/application`

| Tên hằng | Giá trị | Mục đích |
|----------|---------|----------|
| `DashboardCacheTTL` | `1 * time.Hour` | Thời gian sống của cache snapshot User Wardrobe |
| `AggregationInterval` | `15 * time.Minute` | Chu kỳ chạy Aggregator Worker |
| `DefaultLookbackDays` | `30` | Số ngày mặc định khi không gửi `fromDate` |
| `UnderutilizedItemsLimit` | `10` | Số món đồ lãng phí tối đa trả về |
| `MaxDateRangeDays` | `730` | Giới hạn tối đa số ngày cho Dynamic Query |

---

## 6. Lỗi (Error Messages)

**Source:** `internal/modules/dashboard/application/errors`

| Hàm | HTTP Status | Message (tiếng Việt) |
|-----|-------------|---------------------|
| `ErrDashboardDataNotFound()` | 404 | "Không tìm thấy dữ liệu thống kê dashboard." |
| `ErrBrandAnalyticsNotFound()` | 404 | "Không tìm thấy dữ liệu thống kê cho thương hiệu này." |
