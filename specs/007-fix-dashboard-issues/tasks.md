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

- [ ] T001 [P] Xoá `Filters` và `GroupBy` khỏi `DynamicQueryReq` trong `internal/modules/dashboard/application/dto/dynamic_query_dto.go`
- [ ] T002 [P] Xoá field `AverageItemValueVND float64` khỏi `UserWardrobeInsightsRes` trong `internal/modules/dashboard/application/dto/user_dashboard_dto.go`
- [ ] T003 Xoá mapping `AverageItemValueVND` trong `internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go` — xoá dòng `AverageItemValueVND: summary.AverageCostPerWearVND` ở cache-hit path (line 49) và cache-miss path (line 125)
- [ ] T004 Verify: `go build ./...` pass, `go vet ./...` pass

**Checkpoint**: Build sạch, DTO không còn field chết.

---

## Phase 2: MetricKey Constants (FR-005)

**Mục tiêu**: Thêm 7 MetricKey constants mới cho Brand KPI, xoá `AverageCostPerWearVND`.

**Independent Test**: `go build ./...` pass.

- [ ] T005 Xoá constant `AverageCostPerWearVND MetricKey = "average_cost_per_wear_vnd"` khỏi `internal/modules/dashboard/domain/constants/metrickey/metric_key.go`
- [ ] T006 Thêm 7 constants mới vào `internal/modules/dashboard/domain/constants/metrickey/metric_key.go`:
  - `VouchersRedeemed MetricKey = "vouchers_redeemed"`
  - `DigitalSampleVotesCount MetricKey = "digital_sample_votes_count"`
  - `DigitalSampleAvgRating MetricKey = "digital_sample_avg_rating"`
  - `OpenCSTickets MetricKey = "open_cs_tickets"`
  - `CSFirstResponseAvgSeconds MetricKey = "cs_first_response_avg_seconds"`
  - `PointsExpiring30d MetricKey = "points_expiring_30d"`
  - `PointsLiabilityValueVND MetricKey = "points_liability_value_vnd"`
- [ ] T007 Verify: `go build ./...` pass

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

- [ ] T008 Thêm hàm `aggregateByGranularity(series []dto.TimeSeriesPointDTO, gran granularity.Granularity) []dto.TimeSeriesPointDTO` trong `internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go`:
  - Phân loại metric keys (cumulative/non-cumulative/rate) dựa trên quy tắc trên
  - Gộp daily points vào periods (ISO week cho week, calendar month cho month)
  - Timestamp của aggregated point = ngày đầu kỳ (Monday cho week, 1st cho month)
- [ ] T009 Trong `ExecuteDynamicQuery`, sau khi build `series` từ daily snapshots, nếu `gran != "day"` thì gọi `aggregateByGranularity(series, gran)`
- [ ] T010 Kiểm tra auto-downgrade rules vẫn hoạt động — response `granularity` phản ánh đúng granularity thực tế
- [ ] T011 Verify: `go build ./...` pass, `go test ./internal/modules/dashboard/...` pass

**Checkpoint**: Gọi API `granularity: "month"` + 12 tháng → series có đúng 12 points.

---

## Phase 4: Subscription Reader + Fallback (FR-003 — Cross-module)

**Mục tiêu**: Inject `ISubscriptionDashboardReader` vào `AdminDashboardUseCase` để fallback path populate `subscriptionRevenueVnd` và `payOSSuccessCount`.

**Independent Test**: Gọi `GET /api/v1/admin/dashboard/kpi-cards` khi bảng `dashboard_admin_daily_snapshots` trống → trả về `subscriptionRevenueVnd` và `payOSSuccessCount` từ live data.

**⚠️ Cross-module**: Dashboard module nhận dependency mới từ **subscription** module.

- [ ] T012 Thêm field `subscriptionReader subscriptionContract.ISubscriptionDashboardReader` vào struct `AdminDashboardUseCase` trong `internal/modules/dashboard/application/usecase/admin_dashboard_uc_impl.go`
- [ ] T013 Sửa `NewAdminDashboardUseCase` — thêm param `subscriptionReader subscriptionContract.ISubscriptionDashboardReader`
- [ ] T014 Sửa fallback path trong `GetAdminKPICards` (khi `latest == nil`):
  - Gọi `uc.subscriptionReader.GetAdminSubscriptionStats(ctx, now)`
  - Nếu subscriptionStats != nil: set `SubscriptionRevenueVND` (convert decimal → float64), `PayOSSuccessCount`, `PayOSTransactionCount`
  - Nếu subscriptionStats == nil: **không set mặc định 0** — return lỗi
- [ ] T015 Sửa `internal/modules/dashboard/provider.go` — thêm `subscriptionContract.NewSubscriptionDashboardReader` vào ProviderSet, inject vào `NewAdminDashboardUseCase`
- [ ] T016 Sửa `internal/modules/dashboard/application/usecase/admin_dashboard_uc_test.go` — thêm mock subscription reader, update `NewAdminDashboardUseCase` calls
- [ ] T017 Verify: `go build ./...` pass, `go test ./internal/modules/dashboard/application/usecase/...` pass

**Checkpoint**: Fallback KPI cards trả về subscription + PayOS data từ live subscription reader.

---

## Phase 5: Brand Item Lazy Load (FR-004 — Cross-module)

**Mục tiêu**: Chuyển `DashboardBrandItemDailySnapshot` từ worker-based → lazy load on-demand. `GetDigitalSampleLabAnalytics` tính `feedbackSentiment` từ `digital_sample_responses.rating` + `topPairedCategories` từ outfit pairing data qua 3 contracts.

**Independent Test**: Gọi `GET /api/v1/brand/dashboard/operations/digital-sample-lab/{itemId}` → `feedbackSentiment` có giá trị (không ""), `topPairedCategories` có data (không null).

**⚠️ Cross-module changes (3 modules)**:

### Step A: Brand module — GetItemFeedbackSummary

**Mục đích**: Brand module expose per-item feedback + outfit IDs.

- [ ] T018 Thêm DTO `ItemFeedbackSummaryDTO` vào `internal/modules/brand/application/dto/dashboard_dto.go`:
  ```go
  type ItemFeedbackSummaryDTO struct {
      VotesCount        int
      AvgRating         float64
      FeedbackSentiment string
      OutfitIDs         []uuid.UUID // outfit_ids từ digital_sample_responses có outfit context
  }
  ```
- [ ] T019 Thêm `GetItemFeedbackSummary(ctx context.Context, itemID uuid.UUID) (*dto.ItemFeedbackSummaryDTO, error)` vào `IBrandUseCase` trong `internal/modules/brand/application/interface/usecase/brand_uc.go`
- [ ] T020 Implement `GetItemFeedbackSummary` trong brand use case impl:
  - Query `digital_sample_responses` WHERE `brand_item_id = ?`
  - `VotesCount = COUNT(*)`
  - `AvgRating = AVG(rating)` (nullable → 0 nếu không có)
  - `FeedbackSentiment`: AVG(rating) >= 4 → "positive", = 3 → "neutral", <= 2 → "negative"
  - `OutfitIDs`: DISTINCT outfit_id WHERE outfit_id IS NOT NULL
  - **Cấm zero value**: nếu không có response nào → return error (không trả DTO rỗng)
- [ ] T021 Thêm `GetItemFeedbackSummary` vào `BrandDashboardReader` trong `internal/modules/brand/presentation/contract/dashboard_reader.go` — delegate tới `brandUC.GetItemFeedbackSummary`

### Step B: Wardrobe module — GetOutfitItemFashionItemIDs

**Mục đích**: Wardrobe module trả về fashion_item_ids từ outfit_ids.

- [ ] T022 Thêm DTO `OutfitItemFashionDTO` vào `internal/modules/wardrobe/application/dto/dashboard_dto.go`:
  ```go
  type OutfitItemFashionDTO struct {
      FashionItemID uuid.UUID
  }
  ```
  (hoặc đơn giản: method trả về `[]uuid.UUID`)
- [ ] T023 Thêm `GetOutfitItemFashionItemIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error)` vào `IWardrobeDashboardReader` trong `internal/modules/wardrobe/presentation/contract/dashboard_reader.go`
- [ ] T024 Implement trong wardrobe use case impl:
  - Query `outfit_items` WHERE `outfit_id IN (?)`
  - Return DISTINCT fashion_item_ids
- [ ] T025 Implement `WardrobeDashboardReader` method delegate tới use case

### Step C: Dashboard module — Lazy load orchestration

- [ ] T026 Mở rộng struct `BrandDashboardUseCase` trong `internal/modules/dashboard/application/usecase/brand_dashboard_uc_impl.go`:
  - Thêm field `wardrobeReader wardrobeContract.IWardrobeDashboardReader`
  - Thêm field `fashionContract fashionContract.IFashionContract`
  - Update `NewBrandDashboardUseCase` constructor
- [ ] T027 Sửa `GetDigitalSampleLabAnalytics` — lazy load logic:
  - **Bỏ** gọi `GetLatestBrandItemSnapshot` — không dùng snapshot nữa
  - Gọi `uc.brandReader.GetItemFeedbackSummary(ctx, itemID)` → lấy VotesCount, AvgRating, FeedbackSentiment, OutfitIDs
  - Nếu có OutfitIDs > 0:
    - Gọi `uc.wardrobeReader.GetOutfitItemFashionItemIDs(ctx, outfitIDs)` → []fashionItemIDs
    - Nếu có fashionItemIDs > 0:
      - Gọi `fashionContract.ListFashionItems(ctx, fashionItemIDs)` → []FashionItemRes (có CategoryID)
      - Collect categoryIDs từ kết quả
      - Gọi `fashionContract.ListCategories(ctx, categoryIDs)` → []CategoryRes
      - Aggregate: count categoryName → sort DESC → top 5 → `[]PairedCategoryStat`
    - Nếu không có fashionItemIDs: `topPairedCategories = []`
  - Nếu không có OutfitIDs: `topPairedCategories = []`
  - Return `DigitalSampleLabAnalyticsRes` với đầy đủ data
  - **Cấm zero value**: nếu `GetItemFeedbackSummary` trả error → return error
- [ ] T028 Sửa `internal/modules/dashboard/provider.go`:
  - Inject `wardrobeReader` và `fashionContract` vào `NewBrandDashboardUseCase`
- [ ] T029 Verify: `go build ./...` pass, `go test ./internal/...` pass

**Checkpoint**: Digital sample lab API trả về feedbackSentiment + topPairedCategories đầy đủ từ live data.

---

## Phase 5b: Fix User Dashboard Stale Data (Cross-module)

**Mục tiêu**: Xoá `aiQuotaRemaining` khỏi response; refresh `expiringVouchersCount` từ brand reader khi cache miss.

**Independent Test**: Gọi `GET /api/v1/me/dashboard/wardrobe-insights` → response không còn `aiQuotaRemaining`. Xoá snapshot user → gọi lại → `expiringVouchersCount` vẫn có giá trị > 0 từ live data.

### Part A: Xoá aiQuotaRemaining

- [ ] T030 Xoá field `AIQuotaRemaining int` khỏi `UserWardrobeInsightsRes` trong `internal/modules/dashboard/application/dto/user_dashboard_dto.go`
- [ ] T031 Xoá tất cả logic liên quan `AIQuotaRemaining` trong `internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go`:
  - Cache-hit path: xoá dòng `AIQuotaRemaining: summary.AIQuotaRemaining`
  - Cache-miss path: xoá block đọc stale `summary.AIQuotaRemaining` (line 101-107)
  - Xoá `aiQuota` variable và logic gán
- [ ] T032 Verify: `go build ./...` pass

### Part B: Fix expiringVouchersCount từ brand reader

- [ ] T033 Thêm `GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error)` vào `IBrandUseCase` trong `internal/modules/brand/application/interface/usecase/brand_uc.go`
- [ ] T034 Implement `GetUserExpiringBenefitsCount` trong brand use case impl:
  - Query: `SELECT COUNT(*) FROM benefit_redemptions WHERE user_id = ? AND status = 'REDEEMED' AND expires_at BETWEEN now() AND now() + interval '30 days'`
- [ ] T035 Thêm `GetUserExpiringBenefitsCount` vào `IBrandDashboardReader` trong `internal/modules/brand/presentation/contract/dashboard_reader.go` — delegate tới `brandUC.GetUserExpiringBenefitsCount`
- [ ] T036 Mở rộng struct `UserDashboardUseCase` trong `internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go`:
  - Thêm field `brandReader brandContract.IBrandDashboardReader`
  - Update `NewUserDashboardUseCase` constructor
- [ ] T037 Sửa cache-miss path trong `GetWardrobeInsights`:
  - Gọi `uc.brandReader.GetUserExpiringBenefitsCount(ctx, userID)` → populate `ExpiringVouchersCount`
  - Nếu reader trả error → **không silent zero** → return error
- [ ] T038 Sửa `internal/modules/dashboard/provider.go` — inject `brandReader` vào `NewUserDashboardUseCase`
- [ ] T039 Sửa test file `internal/modules/dashboard/application/usecase/user_dashboard_uc_test.go` — thêm mock brand reader
- [ ] T040 Verify: `go build ./...` pass, `go test ./internal/...` pass

**Checkpoint**: Response không còn aiQuotaRemaining. ExpiringVouchersCount luôn fresh sau mỗi cache miss.

---

## Phase 6: AdminOverview Error Handling (FR-006)

**Mục tiêu**: `GetSystemOverview` không nuốt lỗi — trả `apperror.NewInternalError` ngay khi sub-call fail.

**Independent Test**: Mock PayOS service unavailable → HTTP 500, không phải 200 với `payosReconcile=null`.

- [ ] T041 Sửa `GetSystemOverview` trong `internal/modules/dashboard/application/usecase/admin_dashboard_uc_impl.go`:
  - Dòng 199-201: thay `aiMargin = nil` bằng `return nil, apperror.NewInternalError("Không thể truy vấn dữ liệu biên lợi nhuận AI. Vui lòng thử lại sau.")`
  - Dòng 203-206: thay `payos = nil` bằng `return nil, apperror.NewInternalError("Không thể truy vấn dữ liệu đối soát PayOS. Vui lòng thử lại sau.")`
- [ ] T042 Verify: `go build ./...` pass, `go test ./internal/modules/dashboard/...` pass
- [ ] T043 Run `go test ./internal/...` — zero regression

**Checkpoint**: Overview API trả 500 đúng lúc.

---

## Phase 7: Documentation Sync

**Mục tiêu**: Đồng bộ `docs/api/dashboard-api.md` và `docs/api/constants/dashboard.md`.

- [ ] T044 Đồng bộ `docs/api/dashboard-api.md`:
  - Xoá Filters/GroupBy khỏi DynamicQueryReq table
  - Xoá averageItemValueVnd khỏi UserWardrobeInsightsRes
  - Xoá aiQuotaRemaining khỏi UserWardrobeInsightsRes
  - Thêm total_brands, pending_review_brands vào Admin metric table
  - Thêm 7 brand MetricKeys vào Brand metric table
  - Update Overview section: không còn null — trả 500 nếu lỗi
  - Update Dynamic Query trace table trong Phụ lục: thêm 9 metric keys, xoá average_cost_per_wear_vnd
  - Update Granularity description: backend thực sự aggregate
  - Update User Wardrobe Insights trace: xoá averageItemValueVnd dòng
- [ ] T045 Đồng bộ `docs/api/constants/dashboard.md`:
  - Xoá `AverageCostPerWearVND` khỏi MetricKey table
  - Thêm 7 brand MetricKeys mới
  - Update Granularity notes: week/month được aggregate

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
