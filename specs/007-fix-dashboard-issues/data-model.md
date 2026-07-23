# Data Model Changes: Sửa Lỗi Dashboard

## Tổng quan

Không có entity DB mới. Các thay đổi chỉ ở tầng DTO và constants.

---

## 1. DTO Changes

### 1.1 DynamicQueryReq — Xoá Filters/GroupBy (FR-002)

**File:** `internal/modules/dashboard/application/dto/dynamic_query_dto.go`

| Field | Type | Action |
|-------|------|--------|
| `TargetDashboard` | `TargetDashboard` | Giữ nguyên |
| `BrandID` | `*uuid.UUID` | Giữ nguyên |
| `Metrics` | `[]MetricKey` | Giữ nguyên |
| `Timeframe` | `DynamicQueryTimeframeDTO` | Giữ nguyên |
| `Granularity` | `Granularity` | Giữ nguyên |
| **`Filters`** | **`map[string]string`** | **🗑️ Xoá** |
| **`GroupBy`** | **`[]string`** | **🗑️ Xoá** |
| `CompareWithPrevious` | `bool` | Giữ nguyên |

### 1.2 UserWardrobeInsightsRes — Xoá averageItemValueVnd (FR-007)

**File:** `internal/modules/dashboard/application/dto/user_dashboard_dto.go`

| Field | Type | Action |
|-------|------|--------|
| `TotalItems` | `int` | Giữ nguyên |
| `TotalWardrobeValueVND` | `float64` | Giữ nguyên |
| **`AverageItemValueVND`** | **`float64`** | **🗑️ Xoá** |
| `UnderutilizedItemsCount` | `int` | Giữ nguyên |
| `UnderutilizedItems` | `[]UnderutilizedItemDTO` | Giữ nguyên |
| `AIQuotaRemaining` | `int` | Giữ nguyên |
| `ExpiringVouchersCount` | `int` | Giữ nguyên |

**Side effect:** DTO struct `UnderutilizedItemDTO` không đổi.

### 1.3 Không thay đổi

Các DTO sau giữ nguyên 100%:
- `AdminKPICardsRes` (8 fields)
- `AIMarginAnalyticsRes` (9 fields)
- `PayOSReconciliationRes` (4 fields)
- `AdminOverviewRes` (3 sub-fields)
- `BrandKPICardsRes` (11 fields)
- `BrandPointsLiabilityRes` (3 fields)
- `DigitalSampleLabAnalyticsRes` (5 fields)
- `DynamicQueryTimeframeDTO`
- `TimeSeriesPointDTO`
- `ComparisonDTO`
- `DynamicQueryRes`
- `DrillDownAIEventDTO` / `DrillDownAIEventListRes`
- `DrillDownSampleFeedbackDTO` / `DrillDownSampleFeedbackListRes`

---

## 2. MetricKey Constants Changes

**File:** `internal/modules/dashboard/domain/constants/metrickey/metric_key.go`

### Xoá

| Constant | Giá trị | Lý do |
|----------|---------|-------|
| `AverageCostPerWearVND` | `"average_cost_per_wear_vnd"` | Field DTO bị xoá, vô dụng |

### Thêm (7 constants mới cho Brand target)

| Constant | Giá trị | Map từ entity field |
|----------|---------|-------------------|
| `VouchersRedeemed` | `"vouchers_redeemed"` | `snapshot.VouchersRedeemed` |
| `DigitalSampleVotesCount` | `"digital_sample_votes_count"` | `snapshot.DigitalSampleVotesCount` |
| `DigitalSampleAvgRating` | `"digital_sample_avg_rating"` | `snapshot.DigitalSampleAvgRating` |
| `OpenCSTickets` | `"open_cs_tickets"` | `snapshot.OpenCSTickets` |
| `CSFirstResponseAvgSeconds` | `"cs_first_response_avg_seconds"` | `snapshot.CSFirstResponseAvgSeconds` |
| `PointsExpiring30d` | `"points_expiring_30d"` | `snapshot.PointsExpiring30d` |
| `PointsLiabilityValueVND` | `"points_liability_value_vnd"` | `snapshot.PointsLiabilityValueVND` |

---

## 3. Entity DB — Không đổi

| Entity | Table | Action |
|--------|-------|--------|
| `DashboardAdminDailySnapshot` | `dashboard_admin_daily_snapshots` | Giữ nguyên |
| `DashboardBrandDailySnapshot` | `dashboard_brand_daily_snapshots` | Giữ nguyên |
| `DashboardBrandItemDailySnapshot` | `dashboard_brand_item_daily_snapshots` | Giữ nguyên |
| `DashboardUserWardrobeSummary` | `dashboard_user_wardrobe_summaries` | Giữ nguyên (field `AverageCostPerWearVND` không dùng nữa nhưng giữ trong DB) |

---

## 4. Granularity Aggregation Logic (FR-001)

Không thay đổi data model. Logic aggregation thêm trong `dynamic_query_uc_impl.go`:

```text
Input: []DailySnapshot (from DB)
Params: granularity (week|month), metric type (cumulative|non-cumulative|rate)
Algorithm:
1. Group daily snapshots by period (ISO week hoặc calendar month)
2. For each period:
   - Cumulative metric → last value in period
   - Non-cumulative metric → SUM over period
   - Rate metric → AVG over period
3. Return []TimeSeriesPointDTO với Timestamp là start of period
```
