# Tasks: Sửa Lỗi Dashboard

**Input**: Design documents từ `/specs/007-fix-dashboard-issues/`

**Quy tắc implement**:
- Mỗi phase implement xong → dừng lại → user verify → mới implement phase tiếp theo
- Tuyệt đối **cấm zero value** — nếu không có dữ liệu phải trả về lỗi, không silent zero
- **Cấm bỏ qua value** vì thiếu hàm từ module khác — phải implement đủ tất cả contract methods
- Tuân thủ Hiến pháp: không preload entity chéo module, giao tiếp qua contract interfaces
- **Migration rule**: Nếu cần thay đổi DB schema, dùng Goose migration (`make migration-create`, `make migration-up`). Không sửa file init-db trực tiếp.

---

## Phase 1: DTO Cleanup (FR-002, FR-007)

**Mục tiêu**: Xoá các field vô dụng khỏi DTO — không thay đổi logic use case.

**Independent Test**: `go build ./...` pass, `go vet ./...` pass.

**⚠️ Lưu ý**: Phase này chỉ sửa DTO structs. MetricKey constants và use case sẽ được xử lý ở phase sau.

- [x] T001 [P] Xoá `Filters` và `GroupBy` khỏi `DynamicQueryReq` trong `internal/modules/dashboard/application/dto/dynamic_query_dto.go`
- [x] T002 [P] Xoá field `AverageItemValueVND float64` khỏi `UserWardrobeInsightsRes` trong `internal/modules/dashboard/application/dto/user_dashboard_dto.go`
- [x] T003 Xoá mapping `AverageItemValueVND` trong `internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go` — xoá dòng `AverageItemValueVND: summary.AverageCostPerWearVND` ở cache-hit path và cache-miss path
- [x] T004 Verify: `go build ./...` pass, `go vet ./...` pass

**Checkpoint**: Build sạch, DTO không còn field chết.

---

## Phase 2: MetricKey Constants (FR-005)

**Mục tiêu**: Thêm 7 MetricKey constants mới cho Brand KPI, xoá `AverageCostPerWearVND`.

**Independent Test**: `go build ./...` pass.

- [x] T005 Xoá constant `AverageCostPerWearVND MetricKey = "average_cost_per_wear_vnd"` khỏi `internal/modules/dashboard/domain/constants/metrickey/metric_key.go`
- [x] T006 Thêm 7 constants mới vào `internal/modules/dashboard/domain/constants/metrickey/metric_key.go`:
  - `VouchersRedeemed MetricKey = "vouchers_redeemed"`
  - `DigitalSampleVotesCount MetricKey = "digital_sample_votes_count"`
  - `DigitalSampleAvgRating MetricKey = "digital_sample_avg_rating"`
  - `OpenCSTickets MetricKey = "open_cs_tickets"`
  - `CSFirstResponseAvgSeconds MetricKey = "cs_first_response_avg_seconds"`
  - `PointsExpiring30d MetricKey = "points_expiring_30d"`
  - `PointsLiabilityValueVND MetricKey = "points_liability_value_vnd"`
- [x] T007 Verify: `go build ./...` pass

**Checkpoint**: Build sạch, 7 brand MetricKeys sẵn sàng.

---

## Phase 3: Granularity Aggregation (FR-001)

**Mục tiêu**: Dynamic Query Engine aggregate daily snapshots theo `granularity` (week/month).

**Independent Test**: `go test ./internal/modules/dashboard/application/usecase/...` pass.

**Chi tiết aggregation logic:**

| Metric type | Cách tính cho weekly/monthly |
|---|---|
| Cumulative (total_users, total_brands, total_members, active_vouchers) | **Last value** của kỳ |
| Non-cumulative (ai_requests_count, subscription_revenue_vnd, points_issued, points_redeemed, vouchers_redeemed, digital_sample_votes_count, open_cs_tickets) | **SUM** trong kỳ |
| Rate (digital_sample_avg_rating, cs_first_response_avg_seconds, net_ai_margin_vnd) | **AVERAGE** (mean) của daily values |

- [x] T008 Thêm hàm `aggregateByGranularity(series []dto.TimeSeriesPointDTO, gran granularity.Granularity) []dto.TimeSeriesPointDTO` trong `internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go`:
  - Phân loại metric keys (cumulative/non-cumulative/rate) dựa trên quy tắc trên
  - Gộp daily points vào periods (ISO week cho week, calendar month cho month)
  - Timestamp của aggregated point = ngày đầu kỳ (Monday cho week, 1st cho month)
- [x] T009 Trong `ExecuteDynamicQuery`, sau khi build `series` từ daily snapshots, nếu effective granularity là week/month thì gọi `aggregateByGranularity(series, gran)`
- [x] T010 Kiểm tra auto-downgrade rules vẫn hoạt động — response `granularity` phản ánh đúng granularity thực tế
- [x] T011 Thêm switch cases còn thiếu trong `ExecuteDynamicQuery` (`internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go`):
  - **Admin target**: thêm case `metrickey.TotalBrands` → `snapshot.TotalBrands`, `metrickey.PendingReviewBrands` → `snapshot.PendingReviewBrands`
  - **Brand target**: thêm 7 case mới: `VouchersRedeemed`, `DigitalSampleVotesCount`, `DigitalSampleAvgRating`, `OpenCSTickets`, `CSFirstResponseAvgSeconds`, `PointsExpiring30d`, `PointsLiabilityValueVND` → snapshot fields tương ứng
  - Đồng bộ `sumPreviousPeriod` với các metric mới tương tự
- [x] T012 Verify: `go build ./...` pass, `go test ./internal/modules/dashboard/...` pass

**Checkpoint**: Gọi API `granularity: "month"` + 12 tháng → series có đúng 12 points.

---

## Phase 4: Subscription Reader + Fallback (FR-003 — Cross-module)

**Mục tiêu**: Inject `ISubscriptionDashboardReader` vào `AdminDashboardUseCase` để fallback path populate `subscriptionRevenueVnd` và `payOSSuccessCount`.

**Independent Test**: Gọi `GET /api/v1/admin/dashboard/kpi-cards` khi bảng `dashboard_admin_daily_snapshots` trống → trả về `subscriptionRevenueVnd` và `payOSSuccessCount` từ live data.

**⚠️ Cross-module**: Dashboard module nhận dependency mới từ **subscription** module.

**Contract**: tạo `internal/modules/subscription/presentation/contract/dashboard_reader.go`:
```go
type ISubscriptionDashboardReader interface {
    GetAdminSubscriptionStats(ctx context.Context, asOfDate time.Time) (*dto.SubscriptionStatsDTO, error)
}
```

- [x] T013 DTO `AdminSubscriptionStatsDTO` đã tồn tại trong subscription module — không cần tạo
- [x] T014 `ISubscriptionDashboardReader` + `dashboard_reader.go` đã tồn tại — không cần tạo
- [x] T015 `SubscriptionDashboardReader` đã implement — không cần làm
- [x] T016 Thêm field `subscriptionReader subscriptionContract.ISubscriptionDashboardReader` vào struct `AdminDashboardUseCase`
- [x] T017 Sửa `NewAdminDashboardUseCase` — thêm param `subscriptionReader`
- [x] T018 Sửa fallback path trong `GetAdminKPICards` — dùng subscriptionReader populate `SubscriptionRevenueVND` + `PayOSSuccessCount`
- [x] T019 Chạy `wire` generator — tự động inject subscriptionReader vào DI
- [x] T020 Sửa test file — thêm mock subscription reader, update `NewAdminDashboardUseCase` calls
- [x] T021 Verify: `go build ./...` pass, `go test ./internal/modules/dashboard/application/usecase/...` pass

**Checkpoint**: Fallback KPI cards trả về subscription + PayOS data từ live subscription reader.

---

## Phase 5: Brand Item Lazy Load (FR-004 — Cross-module)

**Mục tiêu**: Chuyển `DashboardBrandItemDailySnapshot` từ worker-based → lazy load on-demand. `GetDigitalSampleLabAnalytics` tính `feedbackSentiment` từ `digital_sample_responses.rating` + `topPairedCategories` từ outfit pairing data qua 3 contracts.

**Independent Test**: Gọi `GET /api/v1/brand/dashboard/operations/digital-sample-lab/{itemId}` → `feedbackSentiment` có giá trị (không ""), `topPairedCategories` có data (không null).

**⚠️ Cross-module changes (3 modules)**:

### Step A: Brand module — GetItemFeedbackSummary

**Mục đích**: Brand module expose per-item feedback + outfit IDs.

- [x] T022 Thêm DTO `ItemFeedbackSummaryDTO` vào `internal/modules/brand/application/dto/dashboard_dto.go`
- [x] T023 Thêm `GetItemFeedbackSummary` vào `IBrandUseCase` interface
- [x] T024 Implement `GetItemFeedbackSummary` trong `BrandUseCase` — thêm `feedbackRepo`, query `digital_sample_responses`, tính VotesCount/AvgRating/Sentiment/OutfitIDs
- [x] T025 Thêm `GetItemFeedbackSummary` vào `BrandDashboardReader` — delegate tới `brandUC`

### Step B: Wardrobe module — GetOutfitItemFashionItemIDs

- [x] T026 Không cần DTO riêng — method trả về `[]uuid.UUID`
- [x] T027 Thêm `GetOutfitItemFashionItemIDs` vào `IWardrobeDashboardReader`
- [x] T028 Implement: thêm `GetFashionItemIDsByOutfitIDs` vào `IOutfitRepository` + outfit repo impl; thêm method vào `WardrobeItemUseCase`
- [x] T029 Implement `WardrobeDashboardReader` method delegate tới `wardrobeUC`

### Step C: Dashboard module — Lazy load orchestration

- [x] T030 Mở rộng `BrandDashboardUseCase` — thêm `wardrobeReader` + `fashionContract` + update constructor
- [x] T031 `GetDigitalSampleLabAnalytics` rewritten: lazy load feedback từ brand → pairedCategories từ wardrobe + fashion chain; giữ nguyên 2 field còn lại
- [x] T032 Wire tự động inject `wardrobeReader` và `fashionContract` — chỉ cần chạy `wire` generator
- [x] T033 Verify: `go build ./...` pass, `go test ./internal/...` pass (zero regression)

**Checkpoint**: Digital sample lab API trả về feedbackSentiment + topPairedCategories đầy đủ từ live data.

---

## Phase 5b: Fix User Dashboard Stale Data (Cross-module)

**Mục tiêu**: Xoá `aiQuotaRemaining` khỏi response; refresh `expiringVouchersCount` từ brand reader khi cache miss.

**Independent Test**: Gọi `GET /api/v1/me/dashboard/wardrobe-insights` → response không còn `aiQuotaRemaining`. Xoá snapshot user → gọi lại → `expiringVouchersCount` vẫn có giá trị > 0 từ live data.

### Part A: Xoá aiQuotaRemaining

- [x] T034 Xoá field `AIQuotaRemaining int` khỏi `UserWardrobeInsightsRes`
- [x] T035 Xoá tất cả logic liên quan `AIQuotaRemaining` trong user_dashboard_uc_impl.go (cache-hit, cache-miss, write-back)
- [x] T036 Verify: `go build ./...` pass

### Part B: Fix expiringVouchersCount từ brand reader

- [x] T037 Thêm `GetUserExpiringBenefitsCount` vào `IBrandUseCase`
- [x] T038 Implement: thêm `GetUserExpiringBenefitsCount` vào `IBenefitRedemptionRepository` + `BenefitRedemptionRepository` + `BrandUseCase`
- [x] T039 Thêm `GetUserExpiringBenefitsCount` vào `IBrandDashboardReader`
- [x] T040 Mở rộng `UserDashboardUseCase` — thêm `brandReader` + update constructor
- [x] T041 Cache-miss path: gọi `brandReader.GetUserExpiringBenefitsCount` để populate `ExpiringVouchersCount`
- [x] T042 Wire tự động inject `brandReader` — không cần sửa provider.go
- [x] T043 Sửa test file — thêm mock brand reader + update calls + remove AIQuotaRemaining assertions
- [x] T044 Verify: `go build ./...` pass, `go test ./internal/...` pass

**Checkpoint**: Response không còn aiQuotaRemaining. ExpiringVouchersCount luôn fresh sau mỗi cache miss.

---

## Phase 6: AdminOverview Error Handling (FR-006)

**Mục tiêu**: `GetSystemOverview` không nuốt lỗi — trả `apperror.NewInternalError` ngay khi sub-call fail.

**Independent Test**: Mock PayOS service unavailable → HTTP 500, không phải 200 với `payosReconcile=null`.

- [x] T045 Sửa `GetSystemOverview` — thay `aiMargin = nil` và `payos = nil` bằng `return nil, apperror.NewInternalError(...)`
- [x] T046 Verify: `go build ./...` pass, `go test ./internal/modules/dashboard/...` pass
- [x] T047 Run `go test ./internal/...` — zero regression

**Checkpoint**: Overview API trả 500 đúng lúc.

---

## Phase 7: Documentation Sync

**Mục tiêu**: Đồng bộ `docs/api/dashboard-api.md` và `docs/api/constants/dashboard.md`.

- [x] T048 Đồng bộ `docs/api/dashboard-api.md`:
  - DynamicQueryReq: xoá Filters/GroupBy, xoá `"hour"` khỏi Granularity
  - UserWardrobeInsightsRes: xoá `AIQuotaRemaining`
  - Granularity description: cập nhật auto-downgrade rules
  - User Wardrobe Insights trace: xoá aiQuotaRemaining, update expiringVouchersCount
  - Các phần còn lại (metric tables, overview) đã đúng
- [x] T049 Đồng bộ `docs/api/constants/dashboard.md`:
  - Xoá `Hour` khỏi Granularity table + cập nhật auto-downgrade
  - MetricKey table đã có đủ 7 brand keys + không có AverageCostPerWearVND

**Checkpoint**: Docs phản ánh chính xác code hiện tại.

---

## Dependencies & Execution Order

### Phase Dependencies

| Phase | Phụ thuộc | Ghi chú |
|-------|-----------|---------|
| 1 DTO Cleanup | Không | Bắt đầu ngay |
| 2 MetricKey | Không | Bắt đầu ngay |
| 3 Granularity | 1, 2 | Dùng DTO + MetricKey đã sửa |
| 4 Subscription Reader | 1 | Dùng DTO đã sửa |
| 5 Brand Lazy Load | 1, 2 | Dùng DTO + MetricKey |
| 5b User Stale Data | 1 | Dùng DTO đã sửa |
| 6 Error Handling | 1 | Dùng DTO đã sửa |
| 7 Docs Sync | 1, 2, 3, 4, 5, 5b, 6 | Phải implement xong mới docs |

### Execution Order

```text
Phase 1 ──→ Phase 2 ──→ Phase 3 ──→ Phase 4 ──→ Phase 5 ──→ Phase 5b ──→ Phase 6 ──→ Phase 7
            (tuần tự, mỗi phase verify xong mới qua phase tiếp theo)
```

---

## Implementation Strategy

1. Implement **Phase 1** → Build + test → **STOP, user verify**
2. Implement **Phase 2** → Build + test → **STOP, user verify**
3. Implement **Phase 3** → Build + test → **STOP, user verify**
4. Implement **Phase 4** → Build + test → **STOP, user verify**
5. Implement **Phase 5** → Build + test → **STOP, user verify**
6. Implement **Phase 5b** → Build + test → **STOP, user verify**
7. Implement **Phase 6** → Build + test → **STOP, user verify**
8. Implement **Phase 7** → Docs sync → **DONE**

Mỗi phase implement xong, agent **phải dừng lại** và đợi user xác nhận trước khi implement phase tiếp theo.

---

## Notes

- [P] tasks = khác file, không phụ thuộc — có thể parallel trong cùng phase
- **Cấm zero value**: không có dữ liệu → return error, không silent zero
- **Cross-module**: Phase 4 (subscription), Phase 5 (brand + wardrobe + fashion), Phase 5b (brand)
- Không cần DB migration: `dashboard_brand_item_daily_snapshots` giữ nguyên (không dùng), `dashboard_user_wardrobe_summaries.ai_quota_remaining` giữ nguyên (không xoá column)
- Mọi error message bằng tiếng Việt
- Commit sau mỗi phase khi user verify xong
