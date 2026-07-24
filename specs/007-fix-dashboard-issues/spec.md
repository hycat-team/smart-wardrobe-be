# Đặc Tả Tính Năng: Sửa Lỗi Dashboard

**Feature Branch**: `007-fix-dashboard-issues`

**Ngày tạo**: 2026-07-24

**Trạng thái**: Bản nháp

**Đầu vào**: Kết quả audit toàn bộ Dashboard APIs (Feature 006) phát hiện nhiều field trong DTO được khai báo nhưng không bao giờ xử lý, không được populate trong fallback path, MetricKey constants không được resolve, và lỗi kiến trúc.

---

## Tổng Quan

Feature 006 (Dashboard & Analytics) đã triển khai 9 phase với 55 tasks. Một cuộc audit toàn diện 24 DTOs (114 fields trên admin, brand, user, dynamic query APIs) xác định 7 nhóm vấn đề:

- **P0 (Sai thiết kế)**: Granularity, Filters, GroupBy trong Dynamic Query Engine được khai báo nhưng không bao giờ xử lý — dữ liệu luôn trả về daily granularity bất kể request.
- **P1 (Thiếu dữ liệu ở edge case)**: 4 response fields không được populate trong fallback path (khi không có snapshot), trả về zero/nil âm thầm.
- **P2 (Bao phủ metric không đầy đủ)**: 3 MetricKey constants tồn tại nhưng Dynamic Query Engine không resolve; 7 brand KPI fields không có MetricKey.
- **P3 (Lỗi kiến trúc)**: AdminOverview nuốt lỗi trả về HTTP 200 với dữ liệu null; `averageItemValueVnd` trong UserWardrobeInsightsRes là field vô dụng (không tính được CPW).

Feature này sửa tất cả các lỗi phát hiện được để Dashboard APIs hoạt động đúng cho frontend integration.

---

## Kịch Bản Người Dùng & Kiểm Thử

### Câu Chuyện 1 — Admin Xem Xu Hướng 12 Tháng Qua Dynamic Query Engine (P0)

Là một System Admin, tôi muốn xem biểu đồ xu hướng 12 tháng của tổng người dùng và doanh thu subscription, để đánh giá tăng trưởng theo năm.

**Tại sao ưu tiên này**: Đây là use case phổ biến nhất của dashboard analytics — biểu đồ xu hướng theo tháng. Hiện tại `granularity="month"` không hoạt động.

**Kiểm thử độc lập**: Gọi `POST /api/v1/dashboard/analytics/query` với `granularity: "month"` và timeframe 12 tháng, kiểm tra `series` trả về đúng 12 points.

**Kịch bản chấp nhận**:
1. **Given** Admin gọi Dynamic Query với `granularity: "month"` và timeframe 12 tháng, **When** hệ thống xử lý, **Then** series trả về đúng 12 data points, mỗi point đại diện 1 tháng (giá trị là tổng/tháng).
2. **Given** Admin gọi với `granularity: "week"` và timeframe 4 tuần, **When** hệ thống xử lý, **Then** series trả về 4 data points, mỗi point đại diện 1 tuần.
3. **Given** Admin gọi với `granularity: "hour"` và timeframe > 365 ngày, **When** hệ thống xử lý, **Then** tự động downgrade thành `"day"` (cơ chế hiện tại) và response thông báo granularity thực tế.
4. **Given** Admin gửi request có chứa `filters` hoặc `groupBy` (các field đã bị xoá khỏi DTO), **When** hệ thống parse request, **Then** các field này bị ignore hoặc DTO không còn chấp nhận chúng.

### Câu Chuyện 2 — Admin Fallback Khi Thiếu Snapshot (P1)

Là một System Admin, tôi muốn dashboard vẫn hiển thị dữ liệu đầy đủ ngay cả khi snapshot worker chưa kịp chạy, để không gây nhầm lẫn khi xem báo cáo.

**Tại sao ưu tiên này**: Fallback path hiện tại trả về `subscriptionRevenueVnd=0` và `payOSSuccessCount=0` — gây hiểu lầm là không có doanh thu/giao dịch.

**Kiểm thử độc lập**: Xoá admin snapshot trong DB, gọi `GET /api/v1/admin/dashboard/kpi-cards`, kiểm tra `subscriptionRevenueVnd` và `payOSSuccessCount` có giá trị > 0 từ live data.

**Kịch bản chấp nhận**:
1. **Given** Chưa có admin snapshot nào (worker chưa chạy hoặc bảng trống), **When** Admin gọi `GET /api/v1/admin/dashboard/kpi-cards`, **Then** `subscriptionRevenueVnd` và `payOSSuccessCount` phải được populate từ live data (subscription reader), không phải mặc định 0.
2. **Given** Chưa có brand item snapshot nào cho item cụ thể, **When** Brand Owner gọi `GET /api/v1/brand/dashboard/operations/digital-sample-lab/{itemId}`, **Then** `feedbackSentiment` và `topPairedCategories` phải được populate từ live data (brand reader), không phải null/empty.

### Câu Chuyện 3 — Bao Phủ Metric Đầy Đủ Trong Dynamic Query (P2)

Là một System Admin hoặc Brand Manager, tôi muốn tất cả KPI hiển thị trong dashboard widget đều có thể query qua Dynamic Query Engine, để xây dựng biểu đồ tùy chỉnh.

**Tại sao ưu tiên này**: 3 MetricKey constants được định nghĩa nhưng dynamic query không xử lý, và 7 brand KPI fields không có MetricKey — gây confusion cho FE.

**Kiểm thử độc lập**: Gọi Dynamic Query với từng metric mới, kiểm tra response trả về giá trị đúng.

**Kịch bản chấp nhận**:
1. **Given** Admin gọi dynamic query với metric `total_brands`, **When** hệ thống xử lý, **Then** trả về giá trị tương ứng từ admin snapshot.
2. **Given** Admin gọi với metric `pending_review_brands`, **When** hệ thống xử lý, **Then** trả về giá trị tương ứng.
3. **Given** Brand Manager muốn query `vouchers_redeemed` hoặc `digital_sample_votes_count` qua dynamic query, **When** hệ thống xử lý, **Then** các metric này có MetricKey constant và được resolve trong switch.

### Câu Chuyện 4 — AdminOverview Xử Lý Lỗi Đúng (P3)

Là một System Admin, tôi muốn nếu một phần của overview API bị lỗi, tôi nhận được thông báo lỗi rõ ràng thay vì thấy dữ liệu null.

**Tại sao ưu tiên này**: `GetSystemOverview` nuốt lỗi trả về 200 OK với sub-response = nil, gây khó debug cho cả backend và frontend.

**Kiểm thử độc lập**: Mock PayOS service trả về lỗi, gọi `GET /api/v1/admin/dashboard/overview`, kiểm tra HTTP status là 5xx.

**Kịch bản chấp nhận**:
1. **Given** PayOS reconciliation service tạm thời không khả dụng, **When** Admin gọi `GET /api/v1/admin/dashboard/overview`, **Then** response trả về HTTP 500 thay vì 200 với `payosReconcile=null`.

---

## Yêu Cầu Chức Năng

### FR-001: Aggregation Dữ Liệu Theo Granularity Trong Dynamic Query Engine (P0)

Dynamic Query Engine phải aggregate dữ liệu snapshot daily thành granularity được yêu cầu:

| Granularity | Phương pháp Aggregation | Ví dụ (query 12 tháng) |
|-------------|------------------------|------------------------|
| `hour` | 🗑️ Đã xoá — backend không support hourly | N/A |
| `day` | Trả raw daily data (hành vi hiện tại, giữ nguyên) | 365 points |
| `week` | Gộp daily → weekly (SUM cho metric tích luỹ, AVG cho metric tỷ lệ, ISO week) | ~52 points |
| `month` | Gộp daily → monthly (SUM cho metric tích luỹ, AVG cho metric tỷ lệ, calendar month) | 12 points |

- Metric **cumulative** (total_users, total_brands): dùng **giá trị cuối kỳ** (last day value of the period).
- Metric **non-cumulative** (ai_requests_count, subscription_revenue_vnd): dùng **SUM theo kỳ**.
- Metric **tỷ lệ** (margin_percentage, avg_rating): dùng **AVERAGE theo kỳ**.
- Giữ nguyên auto-downgrade rules hiện tại — response field `granularity` phải phản ánh granularity thực tế.

### FR-002: Xoá Filters/GroupBy Khỏi DTO (P0)

Xoá hoàn toàn 2 field `Filters` (`map[string]string`) và `GroupBy` (`[]string`) khỏi `DynamicQueryReq` DTO. Contract chỉ giữ lại các field có implement thực tế: `targetDashboard`, `brandId`, `metrics`, `timeframe`, `granularity`, `compareWithPrevious`.

### FR-003: Populate Fallback Đầy Đủ Cho Admin KPIs (P1)

Khi không có admin daily snapshot (`GetLatestAdminSnapshot` trả về nil), fallback path phải:
- Đọc subscription revenue từ `subscriptionContract.ISubscriptionDashboardReader` (mở rộng contract của subscription module) → populate `subscriptionRevenueVnd`.
- Đọc PayOS transaction count/success từ reader tương tự → populate `payOSSuccessCount`.

### FR-004: Lazy Load Digital Sample Lab Analytics Từ Live Data (P1)

Chuyển `GetDigitalSampleLabAnalytics` từ worker-based snapshot → lazy load on-demand:
- Bỏ hoàn toàn đọc `DashboardBrandItemDailySnapshot`.
- Đọc `feedbackSentiment` từ brand reader — query `digital_sample_responses` per-item, tính `AVG(rating)` → sentiment classification (≥4 "positive", =3 "neutral", ≤2 "negative").
- Đọc `topPairedCategories` từ outfit pairing data qua wardrobe + fashion contracts: lấy `outfit_ids` từ brand reader → `fashion_item_ids` từ wardrobe reader → `category_ids` từ fashion contract → aggregate top 5 paired categories.
- **Cấm zero value**: nếu item không có response nào → return error, không trả DTO rỗng.

### FR-005: Thêm MetricKey Còn Thiếu Vào Dynamic Query Engine (P2)

Thêm các case metric sau vào switch-case của Dynamic Query Engine:

**Admin target** (constant đã tồn tại nhưng chưa có case trong switch):
- `total_brands` → `snapshot.TotalBrands`
- `pending_review_brands` → `snapshot.PendingReviewBrands`

**Brand target** (tạo MetricKey constants mới):
- `vouchers_redeemed` → `snapshot.VouchersRedeemed`
- `digital_sample_votes_count` → `snapshot.DigitalSampleVotesCount`
- `digital_sample_avg_rating` → `snapshot.DigitalSampleAvgRating`
- `open_cs_tickets` → `snapshot.OpenCSTickets`
- `cs_first_response_avg_seconds` → `snapshot.CSFirstResponseAvgSeconds`
- `points_expiring_30d` → `snapshot.PointsExpiring30d`
- `points_liability_value_vnd` → `snapshot.PointsLiabilityValueVND`

### FR-006: AdminOverview Lan Truyền Lỗi Theo Chuẩn Dự Án (P3)

`GetSystemOverview` không được nuốt lỗi từ sub-calls:
- Nếu `GetAIMarginAnalytics` hoặc `GetPayOSReconciliation` thất bại → trả về lỗi ngay lập tức bằng `apperror.NewInternalError("message")`.
- Handler nhận error và trả về HTTP 500 qua `shared_pres.WrapHandler`, không trả về 200 OK với dữ liệu null.
- Message lỗi bằng tiếng Việt, mô tả component nào bị lỗi (vd: "Không thể truy vấn dữ liệu đối soát PayOS. Vui lòng thử lại sau.").

### FR-007: Xoá averageItemValueVnd Khỏi UserWardrobeInsightsRes (P3)

`UserWardrobeInsightsRes.averageItemValueVnd` hiện đang lấy dữ liệu từ entity field `AverageCostPerWearVND` (CPW). Do backend chưa có cơ chế tính CPW chính xác, field này vô dụng và gây hiểu lầm.

**Quyết định**: Xoá hoàn toàn field `averageItemValueVnd` khỏi DTO `UserWardrobeInsightsRes`. Entity `DashboardUserWardrobeSummary.AverageCostPerWearVND` giữ nguyên trong DB (do worker vẫn populate), nhưng không expose ra API.
- **MetricKey `average_cost_per_wear_vnd`** cũng bị xoá — không có DTO field nào cần nó, tránh gây confusion cho FE.
- Entity field `AverageCostPerWearVND` trong DB giữ nguyên (có thể dùng sau này khi có CPW thực tế).

---

## Yêu Cầu Phi Chức Năng & Giới Hạn Phạm Vi

### Ngoài Phạm Vi (Explicitly Out of Scope)

- Tạo bảng snapshot monthly/hourly mới (aggregation thực hiện in-query từ daily snapshots).
- Hỗ trợ `filters` và `groupBy` như tính năng thực tế (đã xoá khỏi DTO).
- Tối ưu performance cho date range lớn (caching, pre-rollup tables).
- Thay đổi seed script hoặc hành vi snapshot worker.
- Thêm endpoint dashboard widget mới.

---

## Tiêu Chí Thành Công

1. **Granularity hoạt động end-to-end**: Gọi dynamic query với `granularity: "month"` và timeframe 12 tháng trả về chính xác 12 data points. Kiểm tra bằng automated test.
2. **Fallback đầy đủ**: Gọi `GET /api/v1/admin/dashboard/kpi-cards` khi không có snapshot trả về `subscriptionRevenueVnd` và `payOSSuccessCount` > 0 từ live data. Kiểm tra bằng integration test.
3. **Bao phủ metric đầy đủ**: 2 MetricKey còn thiếu (`total_brands`, `pending_review_brands`) và 7 brand MetricKeys mới được resolve qua dynamic query. Kiểm tra bằng unit test.
4. **Lan truyền lỗi**: `GET /api/v1/admin/dashboard/overview` trả về HTTP 500 khi sub-component fail. Kiểm tra bằng integration test.
5. **DTO sạch**: `filters`, `groupBy`, `averageItemValueVnd` bị xoá khỏi DTO. Kiểm tra bằng code review + `go build`.
6. **Không semantic mismatch**: `averageItemValueVnd` đã xoá khỏi `UserWardrobeInsightsRes`, kèm `average_cost_per_wear_vnd` MetricKey. Kiểm tra bằng code review.
7. **Zero regression**: Tất cả dashboard tests pass, `go build ./...` success, `go test ./internal/...` pass.

---

## Thực Thể Chính (Key Entities)

Không có entity mới. Thay đổi giới hạn trong:
- **DTO structs**: Xoá `filters`/`groupBy` khỏi `DynamicQueryReq`; xoá `averageItemValueVnd` khỏi `UserWardrobeInsightsRes`.
- **MetricKey constants**: Xoá `average_cost_per_wear_vnd`; thêm 7 constants mới cho brand KPI fields.
- **Use case implementations**: Thêm aggregation logic, fallback data, metric switch cases, error handling.
- **Repository interface**: Optional — thêm method subscription stats cho fallback path nếu reader chưa có.

---

## Phụ Thuộc

- Feature 006 (Dashboard & Analytics) phải được merge — feature này chỉ fix trong implementation hiện tại.
- Subscription reader (hoặc interface tương đương) phải accessible từ admin dashboard use case cho fallback path (FR-003).
- Brand reader phải expose per-item feedback sentiment và paired categories (FR-004).

---

## Giả Định

1. Monthly aggregation tính in-application (sum daily snapshots) thay vì database `GROUP BY` — đơn giản và nhất quán với repository pattern hiện tại.
2. Metric cumulative dùng last-day-of-month value cho monthly aggregation.
3. Metric tỷ lệ (rate) dùng weighted average theo số ngày trong kỳ.
4. Subscription reader interface được inject vào AdminDashboardUseCase qua Wire dependency set (không cần restructure DI).
5. Brand item fallback data (feedbackSentiment, topPairedCategories) có thể derive từ brand reader methods hiện tại.
6. API response của `UserWardrobeInsightsRes` sẽ mất field `averageItemValueVnd` — cần thông báo trước cho FE để tránh breaking change.

---

## Clarifications

### Session 2026-07-24

- Q: Filters/GroupBy nên xoá khỏi DTO hay reject với error? → A: Xoá khỏi DTO hoàn toàn.
- Q: AdminOverview error handling — trả 500 hay 200 với error field? → A: Trả 500 ngay lập tức, dùng `apperror.NewInternalError` theo chuẩn dự án.
- Q: averageItemValueVnd — rename thành averageCostPerWearVnd hay xoá? → A: Xoá khỏi DTO (không tính được CPW). Kèm MetricKey average_cost_per_wear_vnd cũng bị xoá.
