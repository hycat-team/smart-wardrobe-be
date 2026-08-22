# Báo cáo Audit Response DTOs — Module `dashboard`

**Branch**: `013-briefres-audit`
**Ngày**: 2026-08-22
**Sub-agent**: audit dashboard
**Phạm vi**: `internal/modules/dashboard/application/dto/` + `internal/modules/dashboard/presentation/handler/`

---

## 1. Tổng quan

Module **dashboard** gồm 3 nhóm context khác nhau rõ rệt:

| Nhóm | Handler | Trạng thái audit |
|---|---|---|
| Brand dashboard (9 endpoint) | `brand_dashboard_handler.go` | Được đánh giá over/underdelivery |
| User dashboard (2 endpoint) | `user_dashboard_handler.go` | Được đánh giá over/underdelivery |
| Admin dashboard (9 endpoint) + drill-down Admin | `admin_dashboard_handler.go`, `analytics_handler.go` | **Chỉ ghi nhận — không đề xuất refactor** |

**Nhận xét chung**: Đại đa số struct trong module này là **struct aggregation (thống kê)** — mỗi field đều là một metric hiển thị trên card/chart cụ thể, nên KHÔNG bị overdelivery theo nghĩa cổ điển (trả entity lớn). Vấn đề chính của module nằm ở **underdelivery**: một số response trả **bare UUID** (`itemId`, `userId`) mà frontend bắt buộc phải hiển thị thông tin liên quan (tên sản phẩm, ảnh sản phẩm, tên/avatar user) nhưng không có sẵn → phải gọi thêm API.

---

## 2. Danh sách struct

### 2.1 Brand dashboard — `brand_dashboard_dto.go`

| # | Struct | Fields | Ghi chú |
|---|---|---|---|
| 1 | `BrandKPICardsRes` | BrandID, TotalMembers, PointsIssued, PointsRedeemed, ActiveVouchers, VouchersRedeemed, DigitalSampleVotesCount, DigitalSampleAvgRating, OpenCSTickets, UpdatedAt | 10 fields — aggregation KPI |
| 2 | `BrandPointsLiabilityRes` | BrandID, PointsExpiring30d, PointsLiabilityValueVND | 3 fields — OK |
| 3 | `CustomerAcquisitionRes` | BrandID, TotalCustomers, Sources[CustomerAcquisitionDTO] | aggregation — OK |
| 4 | `CustomerAcquisitionDTO` | Source, CustomerCount, Percentage | 3 fields — OK |
| 5 | `TierDistributionRes` | BrandID, TotalMembers, Tiers[TierDistributionDTO] | aggregation — OK |
| 6 | `TierDistributionDTO` | TierID, TierName, TierRank, MemberCount, Percentage | 5 fields — OK |
| 7 | `PointExpiryForecastRes` | BrandID, Forecast[PointExpiryForecastDTO] | aggregation — OK |
| 8 | `PointExpiryForecastDTO` | Month, PointsExpiring | 2 fields — OK |
| 9 | `BenefitRedemptionAnalyticsRes` | BrandID, TotalRedemptions, Trend, TopBenefits | aggregation — OK |
| 10 | `BenefitRedemptionDTO` | BenefitID, BenefitName, BenefitType, RedemptionCount, Percentage | Đã có name/type → **đã là brief sẵn** |
| 11 | `RedemptionTrendDTO` | Month, RedemptionCount | 2 fields — OK |
| 12 | `CatalogStatsRes` | BrandID, TotalItems, ByStatus, ByType | aggregation — OK |
| 13 | `CatalogStatGroupDTO` | Label, Count, Percentage | 3 fields — OK |
| 14 | `SpendSegmentationRes` | BrandID, TotalCustomers, Segments | aggregation — OK |
| 15 | `SpendSegmentDTO` | Segment, MinVND, MaxVND, CustomerCount, Percentage | 5 fields — OK |
| 16 | `DigitalSampleLabAnalyticsRes` | **ItemID**, VotesCount, AvgRating, FeedbackSentiment, TopPairedCategories | **Underdelivery** — bare ItemID |
| 17 | `PairedCategoryStat` | CategoryName, PairCount | **Thiếu CategoryID** |

### 2.2 User dashboard — `user_dashboard_dto.go`

| # | Struct | Fields | Ghi chú |
|---|---|---|---|
| 18 | `UserWardrobeInsightsRes` | TotalItems, TotalWardrobeValueVND, UnderutilizedItemsCount, UnderutilizedItems, ExpiringVouchersCount | 5 fields — OK |
| 19 | `UnderutilizedItemDTO` | ItemID, Name, CategoryName, PurchasePriceVND, LastWornDaysAgo, ImageURL | 6 fields — đã đủ thông tin hiển thị |
| 20 | `WardrobeCategoryDistributionRes` | TotalItems, Categories | aggregation — OK |
| 21 | `CategoryDistributionDTO` | CategoryID, CategoryName, ItemCount, Percentage | 4 fields — đã có name |

### 2.3 Admin dashboard — `admin_dashboard_dto.go` *(chỉ ghi nhận)*

| # | Struct | Fields |
|---|---|---|
| 22 | `AdminKPICardsRes` | TotalUsers, ActiveUsersDAU, TotalBrands, PendingReviewBrands, SubscriptionRevenueVND, NetAIMarginVND, PayOSSuccessCount, UpdatedAt — 8 fields |
| 23 | `AIMarginAnalyticsRes` | 9 fields |
| 24 | `PayOSReconciliationRes` | 4 fields |
| 25 | `AdminOverviewRes` | composite 3 struct |
| 26 | `SubscriptionDistributionRes`/`DTO` | 2 + 3 fields |
| 27 | `RevenueBreakdownRes`/`DTO` | 4 + 4 fields |
| 28 | `AIUsageByOperationRes`/`DTO` | 3 + 6 fields |
| 29 | `AIErrorBreakdownRes`/`DTO` | 4 + 4 fields |
| 30 | `RenewalStatsRes` | 5 fields |

### 2.4 Dynamic query & Drill-down — `dynamic_query_dto.go` *(admin drill-down chỉ ghi nhận)*

| # | Struct | Fields | Ghi chú |
|---|---|---|---|
| 31 | `DynamicQueryRes` | TargetDashboard, Granularity, Series, Comparison | OK |
| 32 | `TimeSeriesPointDTO` | Timestamp, Metrics | OK |
| 33 | `ComparisonDTO` | PreviousPeriodFrom, PreviousPeriodTo, MetricDeltaPercentages | OK |
| 34 | `DrillDownAIEventDTO` | 6 fields | **Admin context — không refactor** |
| 35 | `DrillDownSampleFeedbackDTO` | FeedbackID, **ItemID**, **UserID**, Rating, Comment, CreatedAt | **Underdelivery** — bare UserID + ItemID |

---

## 3. Overdelivery

### 3.1 `BrandKPICardsRes` — Thấp

- **Context hiển thị**: Trang brand portal hiển thị 8 thẻ KPI riêng biệt → **cả 8 metric đều cần thiết**, không dư metric nào.
- **Overdelivery thực sự**: `BrandID` (frontend đã biết brand context từ URL/session) và `UpdatedAt` (timestamp snapshot internal, không cần cho display).
- **Đề xuất**:
  - **Giữ (8)**: TotalMembers, PointsIssued, PointsRedeemed, ActiveVouchers, VouchersRedeemed, DigitalSampleVotesCount, DigitalSampleAvgRating, OpenCSTickets
  - **Bỏ (2)**: `BrandID`, `UpdatedAt`

> Lưu ý: `UpdatedAt` giữ vai trò "dữ liệu có phải snapshot cũ" — nếu product cần hiển thị "cập nhật lúc X" thì cân nhắc giữ. Mặc định đề xuất bỏ.

### 3.2 Các struct aggregation khác

- `TierDistributionDTO`, `SpendSegmentDTO`, `CatalogStatGroupDTO`, `BenefitRedemptionDTO`, `CustomerAcquisitionDTO`, `CategoryDistributionDTO`, `UnderutilizedItemDTO`: đều có **3–6 fields, mỗi field một vai trò hiển thị cụ thể** → **KHÔNG overdelivery**.
- `DrillDownSampleFeedbackDTO`, `DrillDownAIEventDTO`: thuộc context admin/brand drill-down — `DrillDownAIEventDTO` là admin → không refactor; `DrillDownSampleFeedbackDTO` thuộc brand → đánh giá dưới mục underdelivery.

---

## 4. Underdelivery

### 4.1 `DigitalSampleLabAnalyticsRes.ItemID` (bare UUID) — Cao

- **Struct**: `DigitalSampleLabAnalyticsRes` (brand_dashboard_dto.go:39)
- **Vấn đề**: Response trả `ItemID` thuần. Frontend mở trang analytics mẫu thử theo `{itemId}` từ danh sách **cần hiển thị tên + ảnh sản phẩm** ở header trang nhưng backend không trả → phải gọi thêm API lấy item detail.
- **Giải pháp**: Thêm nested `Item *BrandItemBriefRes` (tái sử dụng struct `BrandItemBriefRes` đã có ở wardrobe: ID, BrandID, BrandName, ItemType, Name, Price) **hoặc** tạo brief riêng cho sample item có thêm `ImageURL`.
- **Ghi chú**: Struct đã có `TopPairedCategories` với `CategoryName` nhưng `PairedCategoryStat` **thiếu `CategoryID`** → nếu frontend muốn drill-down/điều hướng tới category thì thiếu ID.

### 4.2 `DrillDownSampleFeedbackDTO.UserID` (bare UUID) — Cao

- **Struct**: `DrillDownSampleFeedbackDTO` (dynamic_query_dto.go:69)
- **Vấn đề**: Brand admin xem danh sách feedback mẫu thử cần biết **ai đã đánh giá** (tên/avatar user) nhưng response chỉ trả `UserID` → phải gọi thêm user API cho từng feedback (N+1 từ frontend).
- **Giải pháp**: Thêm nested `User *UserBriefRes`.

### 4.3 `DrillDownSampleFeedbackDTO.ItemID` (bare UUID) — Trung bình

- **Vấn đề**: Trang đã lọc theo `itemId` (query param), nhưng mỗi dòng feedback vẫn nên có tên/ảnh item để hiển thị trong table. Hiện tại thiếu → frontend không render được cột item mà không gọi thêm API.
- **Giải pháp**: Thêm `Item *BrandItemBriefRes` (hoặc tái sử dụng brief ở 4.1).

---

## 5. Đề xuất BriefRes

### 5.1 `UserBriefRes` *(tạo mới)* — Cao

- **Nơi định nghĩa**: Module **identity** sở hữu entity User → đặt tại `internal/modules/identity/application/dto/`, dashboard import.
- **Fields đề xuất** (5):
  ```go
  type UserBriefRes struct {
      ID        uuid.UUID `json:"id"`
      Username  string    `json:"username"`
      FirstName string    `json:"firstName"`
      LastName  string    `json:"lastName,omitempty"`
      AvatarURL *string   `json:"avatarUrl,omitempty"`
  }
  ```
- **Tham chiếu**: `BrandCustomerUserRes` (brand/customer.go:36) đã có ID/Username/FirstName/LastName/AvatarUrl — có thể tái sử dụng pattern, nhưng `UserBriefRes` chuẩn hóa đặt ở identity và dùng chung.

### 5.2 `BrandItemBriefRes` *(tái sử dụng)* — Cao

- **Struct đã tồn tại**: `internal/modules/wardrobe/application/dto/wardrobe.go:43` — ID, BrandID, BrandName, ItemType, Name, Price.
- **Lưu ý**: `DigitalSampleLabAnalyticsRes` và `DrillDownSampleFeedbackDTO` trả `ItemID` của brand item. Nếu context hiển thị cần **ảnh** thì phải mở rộng thêm `ImageURL` (hoặc tạo `SampleItemBriefRes` riêng: ID, Name, ImageURL, BrandName).

### 5.3 `PairedCategoryStat` — bổ sung field — Thấp

- **Struct**: brand_dashboard_dto.go:34
- **Đề xuất**: Thêm `CategoryID uuid.UUID` cạnh `CategoryName` để frontend có thể drill-down/track category mà không gọi thêm API.

---

## 6. Ưu tiên

| Ưu tiên | Struct / Endpoint | Lý do |
|---|---|---|
| **Cao** | `DigitalSampleLabAnalyticsRes` — thêm `Item` brief | Frontend không thể render trang analytics mẫu thử (header cần tên + ảnh item) nếu chỉ có `itemId`; buộc gọi thêm API item detail. |
| **Cao** | `DrillDownSampleFeedbackDTO` — thêm `User` brief | Brand admin cần hiển thị ai đánh giá trên từng dòng; trả bare `userId` gây N+1 request từ frontend. |
| **Trung bình** | `DrillDownSampleFeedbackDTO` — thêm `Item` brief | Cột item trong table không render được nếu thiếu tên/ảnh; nhưng item đã biết qua query param nên có thể defer. |
| **Thấp** | `BrandKPICardsRes` — bỏ `BrandID` + `UpdatedAt` | 8 metric KPI là cần thiết; chỉ bỏ 2 field metadata không quan trọng cho display. Đây không phải overdelivery nghiêm trọng. |
| **Thấp** | `PairedCategoryStat` — thêm `CategoryID` | Chỉ cần thiết khi product muốn drill-down vào category; hiện tại chỉ là text hiển thị. |

---

## 7. API bị ảnh hưởng

| Endpoint | Handler | Response hiện tại | Response đề xuất |
|---|---|---|---|
| `GET /api/v1/brand/dashboard/operations/digital-sample-lab/{itemId}` | `GetDigitalSampleLabAnalytics` | `DigitalSampleLabAnalyticsRes` (ItemID, VotesCount, AvgRating, FeedbackSentiment, TopPairedCategories) | Thêm `Item *BrandItemBriefRes` (hoặc `SampleItemBriefRes` có ImageURL); `PairedCategoryStat` thêm `CategoryID` |
| `GET /api/v1/brand/dashboard/drilldown/sample-lab-feedbacks` | `GetSampleLabFeedbacksDrillDown` | `DrillDownSampleFeedbackListRes` → `DrillDownSampleFeedbackDTO` (FeedbackID, ItemID, UserID, Rating, Comment, CreatedAt) | Thêm `User *UserBriefRes`; cân nhắc thêm `Item *BrandItemBriefRes` |
| `GET /api/v1/brand/dashboard/kpi-cards` | `GetKPICards` | `BrandKPICardsRes` (10 fields) | Bỏ `BrandID`, `UpdatedAt` (giữ 8 metric) |

### Admin endpoints — chỉ ghi nhận, không đề xuất refactor

- `GET /api/v1/admin/dashboard/kpi-cards` → `AdminKPICardsRes`
- `GET /api/v1/admin/dashboard/ai-margin` → `AIMarginAnalyticsRes`
- `GET /api/v1/admin/dashboard/payos-reconciliation` → `PayOSReconciliationRes`
- `GET /api/v1/admin/dashboard/overview` → `AdminOverviewRes`
- `GET /api/v1/admin/dashboard/subscriptions/distribution` → `SubscriptionDistributionRes`
- `GET /api/v1/admin/dashboard/subscriptions/revenue-breakdown` → `RevenueBreakdownRes`
- `GET /api/v1/admin/dashboard/subscriptions/renewal-stats` → `RenewalStatsRes`
- `GET /api/v1/admin/dashboard/ai/usage-by-operation` → `AIUsageByOperationRes`
- `GET /api/v1/admin/dashboard/ai/error-breakdown` → `AIErrorBreakdownRes`
- `GET /api/v1/admin/dashboard/drilldown/ai-events` → `DrillDownAIEventListRes`
- `POST /api/v1/dashboard/analytics/query` → `DynamicQueryRes` (không bị ảnh hưởng)

### User endpoints — không cần thay đổi

- `GET /api/v1/me/dashboard/wardrobe-insights` → `UserWardrobeInsightsRes` (đã đủ thông tin display)
- `GET /api/v1/me/dashboard/wardrobe/category-distribution` → `WardrobeCategoryDistributionRes` (đã có CategoryID + CategoryName)

---

## Kết luận

- **Overdelivery**: Hầu như không có — module dashboard là aggregation stats, mỗi field một metric. Duy nhất `BrandKPICardsRes` dư `BrandID`/`UpdatedAt` (ưu tiên thấp).
- **Underdelivery**: 2 struct chính cần xử lý — `DigitalSampleLabAnalyticsRes` (thiếu item brief) và `DrillDownSampleFeedbackDTO` (thiếu user brief + item brief). Cần tạo mới `UserBriefRes` (đặt ở identity) và tái sử dụng/mở rộng `BrandItemBriefRes`.
- **Admin**: Toàn bộ ghi nhận "không đề xuất refactor" — giữ full fields cho context quản trị.