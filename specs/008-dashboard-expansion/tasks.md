# Tasks: Mở Rộng Dashboard

**Input**: Design documents from `specs/008-dashboard-expansion/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

---

## Format

`- [ ] [ID] [P?] [Story?] Description with file path`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to

---

## Phase 1: Foundational — Constants & DTOs

**Purpose**: Định nghĩa MetricKey constants mới và tất cả response DTOs

- [x] T001 [P] Thêm 8 MetricKey constants mới (`NewUsers`, `AIErrorCount`, `AIErrorRate`, `NewCustomers`, `CustomerClaimRate`, `BenefitRedemptionCount`, `BrandItemCount`, `BrandItemCountByType`) trong `internal/modules/dashboard/domain/constants/metrickey/metric_key.go`
- [x] T002 [P] Thêm DTOs cho Admin APIs trong `internal/modules/dashboard/application/dto/admin_dashboard_dto.go`:
  - `SubscriptionDistributionDTO`, `SubscriptionDistributionRes`
  - `RevenueBreakdownDTO`, `RevenueBreakdownRes`
  - `AIUsageByOperationDTO`, `AIUsageByOperationRes`
  - `AIErrorBreakdownDTO`, `AIErrorBreakdownRes`
  - `RenewalStatsRes`
- [x] T003 [P] Thêm DTOs cho Brand APIs trong `internal/modules/dashboard/application/dto/brand_dashboard_dto.go`:
  - `CustomerAcquisitionDTO`, `CustomerAcquisitionRes`
  - `TierDistributionDTO`, `TierDistributionRes`
  - `PointExpiryForecastDTO`, `PointExpiryForecastRes`
  - `RedemptionTrendDTO`, `BenefitRedemptionDTO`, `BenefitRedemptionAnalyticsRes`
  - `CatalogStatGroupDTO`, `CatalogStatsRes`
  - `SpendSegmentDTO`, `SpendSegmentationRes`
- [x] T004 [P] Thêm DTO cho User API trong `internal/modules/dashboard/application/dto/user_dashboard_dto.go`:
  - `CategoryDistributionDTO`, `WardrobeCategoryDistributionRes`

**Checkpoint**: MetricKey constants và DTOs hoàn tất. Review trước khi sang Phase 2.

---

## Phase 2: Foundational — Interfaces

**Purpose**: Định nghĩa tất cả repository methods và use case methods mới

- [x] T005 Thêm 5 method vào `IAdminDashboardUseCase` trong `internal/modules/dashboard/application/interface/usecase/admin_dashboard_uc.go`:
  - `GetSubscriptionDistribution(ctx) (*dto.SubscriptionDistributionRes, error)`
  - `GetRevenueBreakdown(ctx, fromDate, toDate time.Time) (*dto.RevenueBreakdownRes, error)`
  - `GetAIUsageByOperation(ctx, fromDate, toDate time.Time) (*dto.AIUsageByOperationRes, error)`
  - `GetAIErrorBreakdown(ctx) (*dto.AIErrorBreakdownRes, error)`
  - `GetRenewalStats(ctx) (*dto.RenewalStatsRes, error)`
- [x] T006 Thêm 6 method vào `IBrandDashboardUseCase` trong `internal/modules/dashboard/application/interface/usecase/brand_dashboard_uc.go`:
  - `GetCustomerAcquisition(ctx, brandID uuid.UUID) (*dto.CustomerAcquisitionRes, error)`
  - `GetLoyaltyTierDistribution(ctx, brandID uuid.UUID) (*dto.TierDistributionRes, error)`
  - `GetPointExpiryForecast(ctx, brandID uuid.UUID) (*dto.PointExpiryForecastRes, error)`
  - `GetBenefitRedemptionAnalytics(ctx, brandID uuid.UUID) (*dto.BenefitRedemptionAnalyticsRes, error)`
  - `GetCatalogStats(ctx, brandID uuid.UUID) (*dto.CatalogStatsRes, error)`
  - `GetCustomerSpendSegmentation(ctx, brandID uuid.UUID) (*dto.SpendSegmentationRes, error)`
- [x] T007 Thêm 1 method vào `IUserDashboardUseCase` trong `internal/modules/dashboard/application/interface/usecase/user_dashboard_uc.go`:
  - `GetWardrobeCategoryDistribution(ctx, userID uuid.UUID) (*dto.WardrobeCategoryDistributionRes, error)`
- [x] T008 Thêm 12 method vào `IDashboardRepository` trong `internal/modules/dashboard/domain/repositories/dashboard_repository.go`:
  - Admin: `GetSubscriptionDistribution`, `GetRevenueBreakdown`, `GetAIUsageByOperation`, `GetAIErrorBreakdown`, `GetRenewalStats`
  - Brand: `GetCustomerAcquisition`, `GetLoyaltyTierDistribution`, `GetPointExpiryForecast`, `GetBenefitRedemptionAnalytics`, `GetCatalogStats`, `GetCustomerSpendSegmentation`
  - User: `GetWardrobeCategoryDistribution`

**Checkpoint**: Interfaces hoàn tất. Review trước khi sang Phase 3.

---

## Phase 3: [US1] Admin Subscription APIs

**Goal**: FR-001 (Subscription Distribution) + FR-002 (Revenue Breakdown)

**Independent Test**: `GET /admin/dashboard/subscriptions/distribution` trả về distribution array. `GET /admin/dashboard/subscriptions/revenue-breakdown?fromDate=...&toDate=...` trả về revenue breakdown.

- [x] T009 [US1] Implement `GetSubscriptionDistribution` trong `AdminDashboardUseCase` tại `internal/modules/dashboard/application/usecase/admin_dashboard_uc_impl.go` — gọi `uc.dashboardRepo.GetSubscriptionDistribution(ctx)`, tính percentage
- [x] T010 [US1] Implement `GetRevenueBreakdown` trong `AdminDashboardUseCase` tại `internal/modules/dashboard/application/usecase/admin_dashboard_uc_impl.go` — gọi `uc.dashboardRepo.GetRevenueBreakdown(ctx, fromDate, toDate)`, tính total
- [x] T011 [P] [US1] Implement repository methods trong `internal/modules/dashboard/infrastructure/persistence/dashboard_admin_repo.go`:
  - `GetSubscriptionDistribution(ctx)` → query `user_subscriptions` group by `current_plan_code`
  - `GetRevenueBreakdown(ctx, fromDate, toDate)` → query `deposit_transactions` where `status=1 AND transaction_type='DIRECT_PURCHASE'`, group by `plan_code` + month
- [x] T012 [US1] Thêm handler `GetSubscriptionDistribution` và `GetRevenueBreakdown` trong `internal/modules/dashboard/presentation/handler/admin_dashboard_handler.go` — bind query params, call use case
- [x] T013 [US1] Đăng ký routes trong `internal/api/routes/admin/router.go`:
  - `adminDashboard.GET("/subscriptions/distribution", ...)`
  - `adminDashboard.GET("/subscriptions/revenue-breakdown", ...)`

**Checkpoint**: US1 hoàn tất — 2 Admin Subscription APIs hoạt động. Review trước khi sang Phase 4.

---

## Phase 4: [US2] Admin AI APIs

**Goal**: FR-003 (AI Usage by Operation) + FR-004 (AI Error Breakdown)

**Independent Test**: `GET /admin/dashboard/ai/usage-by-operation` trả về operations array. `GET /admin/dashboard/ai/error-breakdown` trả về error breakdown.

- [x] T014 [US2] Implement `GetAIUsageByOperation` trong `AdminDashboardUseCase` tại `admin_dashboard_uc_impl.go` — gọi `uc.dashboardRepo.GetAIUsageByOperation`, convert cost từ micro VND sang VND
- [x] T015 [US2] Implement `GetAIErrorBreakdown` trong `AdminDashboardUseCase` tại `admin_dashboard_uc_impl.go` — gọi `uc.dashboardRepo.GetAIErrorBreakdown`, tính error rate
- [x] T016 [P] [US2] Implement repository methods trong `dashboard_admin_repo.go`:
  - `GetAIUsageByOperation(ctx, fromDate, toDate)` → query `ai_usage_events` group by `operation`, `logical_route`
  - `GetAIErrorBreakdown(ctx)` → query `ai_usage_events` where status là lỗi, group by `error_code`, `provider`, `model`
- [x] T017 [US2] Thêm handler `GetAIUsageByOperation` và `GetAIErrorBreakdown` trong `admin_dashboard_handler.go`
- [x] T018 [US2] Đăng ký routes trong `admin/router.go`:
  - `adminDashboard.GET("/ai/usage-by-operation", ...)`
  - `adminDashboard.GET("/ai/error-breakdown", ...)`

**Checkpoint**: US2 hoàn tất. Review trước khi sang Phase 5.

---

## Phase 5: [US3] Admin Renewal Stats API

**Goal**: FR-005 (Renewal Stats)

**Independent Test**: `GET /admin/dashboard/subscriptions/renewal-stats` trả về totalAttempts, successCount, failedCount, successRate, avgRetryCount.

- [x] T019 [US3] Implement `GetRenewalStats` trong `AdminDashboardUseCase` tại `admin_dashboard_uc_impl.go` — gọi repo, tính success rate + avg retry
- [x] T020 [US3] Implement repository method `GetRenewalStats(ctx)` trong `dashboard_admin_repo.go` — query `subscription_renewal_attempts` count, sum success, avg attempt_count
- [x] T021 [US3] Thêm handler `GetRenewalStats` trong `admin_dashboard_handler.go`
- [x] T022 [US3] Đăng ký route trong `admin/router.go`: `adminDashboard.GET("/subscriptions/renewal-stats", ...)`

**Checkpoint**: US3 hoàn tất. Review trước khi sang Phase 6.

---

## Phase 6: [US4] Brand Customer APIs

**Goal**: FR-007 (Customer Acquisition) + FR-008 (Loyalty Tier Distribution)

**Independent Test**: `GET /brand/dashboard/customers/acquisition?brandId=<uuid>` trả về sources array. `GET /brand/dashboard/loyalty/tier-distribution?brandId=<uuid>` trả về tiers array.

- [x] T023 [US4] Implement `GetCustomerAcquisition` trong `BrandDashboardUseCase` tại `internal/modules/dashboard/application/usecase/brand_dashboard_uc_impl.go` — gọi repo, tính percentage
- [x] T024 [US4] Implement `GetLoyaltyTierDistribution` trong `BrandDashboardUseCase` tại `brand_dashboard_uc_impl.go` — gọi repo, map tier name
- [x] T025 [P] [US4] Implement repository methods trong `internal/modules/dashboard/infrastructure/persistence/dashboard_brand_repo.go`:
  - `GetCustomerAcquisition(ctx, brandID)` → query `brand_customers` group by `joined_source`
  - `GetLoyaltyTierDistribution(ctx, brandID)` → query `loyalty_accounts` join `loyalty_tiers`, group by `current_tier_id`
- [x] T026 [US4] Thêm handler `GetCustomerAcquisition` và `GetLoyaltyTierDistribution` trong `internal/modules/dashboard/presentation/handler/brand_dashboard_handler.go` — bind brandId, call use case
- [x] T027 [US4] Đăng ký routes trong `internal/api/routes/brand/router.go`:
  - `brandDashboard.GET("/customers/acquisition", ...)`
  - `brandDashboard.GET("/loyalty/tier-distribution", ...)`

**Checkpoint**: US4 hoàn tất. Review trước khi sang Phase 7.

---

## Phase 7: [US5] Brand Loyalty APIs

**Goal**: FR-009 (Point Expiry Forecast) + FR-010 (Benefit Redemption Analytics)

**Independent Test**: `GET /brand/dashboard/loyalty/point-expiry-forecast?brandId=<uuid>` trả về forecast array theo tháng. `GET /brand/dashboard/benefits/redemption-analytics?brandId=<uuid>` trả về top benefits + trend.

- [x] T028 [US5] Implement `GetPointExpiryForecast` trong `BrandDashboardUseCase` tại `brand_dashboard_uc_impl.go` — gọi repo, group tháng, limit 3+ tháng
- [x] T029 [US5] Implement `GetBenefitRedemptionAnalytics` trong `BrandDashboardUseCase` tại `brand_dashboard_uc_impl.go` — gọi repo, tính top benefits + trend
- [x] T030 [P] [US5] Implement repository methods trong `dashboard_brand_repo.go`:
  - `GetPointExpiryForecast(ctx, brandID)` → query `loyalty_point_lots` where ACTIVE + remaining > 0, group by month
  - `GetBenefitRedemptionAnalytics(ctx, brandID)` → query `benefit_redemptions` join `brand_benefits`, group by benefit_id
- [x] T031 [US5] Thêm handler `GetPointExpiryForecast` và `GetBenefitRedemptionAnalytics` trong `brand_dashboard_handler.go`
- [x] T032 [US5] Đăng ký routes trong `brand/router.go`:
  - `brandDashboard.GET("/loyalty/point-expiry-forecast", ...)`
  - `brandDashboard.GET("/benefits/redemption-analytics", ...)`

**Checkpoint**: US5 hoàn tất. Review trước khi sang Phase 8.

---

## Phase 8: [US6] Brand Catalog APIs

**Goal**: FR-011 (Catalog Stats) + FR-012 (Customer Spend Segmentation)

**Independent Test**: `GET /brand/dashboard/operations/catalog-stats?brandId=<uuid>` trả về byStatus + byType. `GET /brand/dashboard/customers/spend-segmentation?brandId=<uuid>` trả về segments.

- [x] T033 [US6] Implement `GetCatalogStats` trong `BrandDashboardUseCase` tại `brand_dashboard_uc_impl.go` — gọi repo, tính percentage cho byStatus và byType
- [x] T034 [US6] Implement `GetCustomerSpendSegmentation` trong `BrandDashboardUseCase` tại `brand_dashboard_uc_impl.go` — gọi repo, map segment names + percentage
- [x] T035 [P] [US6] Implement repository methods trong `dashboard_brand_repo.go`:
  - `GetCatalogStats(ctx, brandID)` → 2 queries: group by `status`, group by `item_type`
  - `GetCustomerSpendSegmentation(ctx, brandID)` → query `loyalty_accounts` with CASE WHEN for spend ranges
- [x] T036 [US6] Thêm handler `GetCatalogStats` và `GetCustomerSpendSegmentation` trong `brand_dashboard_handler.go`
- [x] T037 [US6] Đăng ký routes trong `brand/router.go`:
  - `brandDashboard.GET("/operations/catalog-stats", ...)`
  - `brandDashboard.GET("/customers/spend-segmentation", ...)`

**Checkpoint**: US6 hoàn tất. Review trước khi sang Phase 9.

---

## Phase 9: [US7] User Wardrobe Category Distribution API

**Goal**: FR-014 (Wardrobe Category Distribution)

**Independent Test**: `GET /me/dashboard/wardrobe/category-distribution` trả về categories array với itemCount và percentage.

- [x] T038 [US7] Implement `GetWardrobeCategoryDistribution` trong `UserDashboardUseCase` tại `internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go` — gọi repo, tính percentage
- [x] T039 [US7] Implement repository method `GetWardrobeCategoryDistribution(ctx, userID)` trong `internal/modules/dashboard/infrastructure/persistence/dashboard_user_repo.go` — join `wardrobe_items` → `fashion_items` → `categories`, group by category
- [x] T040 [US7] Thêm handler `GetWardrobeCategoryDistribution` trong `internal/modules/dashboard/presentation/handler/user_dashboard_handler.go` — get userId từ context
- [x] T041 [US7] Đăng ký route trong `internal/api/routes/wardrobe/router.go`: `userDashboard.GET("/wardrobe/category-distribution", ...)`

**Checkpoint**: US7 hoàn tất. Review trước khi sang Phase 10.

---

## Phase 10: Dynamic Query Metrics & Documentation

**Goal**: FR-006 (Admin Dynamic Query Metrics) + FR-013 (Brand Dynamic Query Metrics) + FR-017 (Documentation)

**Independent Test**: `POST /dashboard/analytics/query` với target=admin và metrics `["new_users", "ai_error_count", "ai_error_rate"]` trả về dữ liệu. Tương tự target=brand với metrics mới.

- [x] T042 Thêm mapping cho admin metrics trong `internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go`:
  - `case metrickey.NewUsers`: `metricValues[m] = float64(s.NewUsers)`
  - `case metrickey.AIErrorCount`: `metricValues[m] = float64(s.AIErrorCount)`
  - `case metrickey.AIErrorRate`: computed = `float64(s.AIErrorCount) / float64(s.AIRequestsCount) * 100`
- [x] T043 Cập nhật `metricValueFromAdminSnapshot` trong `dynamic_query_uc_impl.go` để xử lý `NewUsers` và `AIErrorCount`
- [x] T044 Cập nhật `getMetricAggType` trong `dynamic_query_uc_impl.go` cho các metric mới
- [x] T045 Thêm các brand metrics vào dynamic query target=brand trong `dynamic_query_uc_impl.go` — gọi repository methods live query cho `new_customers`, `customer_claim_rate`, `benefit_redemption_count`, `brand_item_count`, `brand_item_count_by_type`
- [x] T046 [P] Thêm các repository method cho brand dynamic query metrics trong `dashboard_brand_repo.go`:
  - `GetBrandNewCustomerCount(ctx, brandID, fromDate, toDate)`
  - `GetBrandCustomerClaimRate(ctx, brandID)`
  - `GetBrandRedemptionCount(ctx, brandID, fromDate, toDate)`
  - `GetBrandItemCount(ctx, brandID)`
  - `GetBrandItemCountByType(ctx, brandID)`
- [x] T047 Tạo file `docs/api/dashboard-api-v2.md` — copy nội dung từ `dashboard-api.md` hiện tại + thêm spec cho 12 API mới (dựa vào contracts/ đã tạo)
- [x] T048 Build thử và kiểm tra: `make check`

**Checkpoint**: All tasks complete. Run `make test` để verify.

---

## Dependencies & Execution Order

### Phase Dependencies
```
Phase 1 (Constants & DTOs)
    ↓
Phase 2 (Interfaces)
    ↓
Phase 3 [US1] → Phase 4 [US2] → Phase 5 [US3]   (Admin - tuần tự)
Phase 6 [US4] → Phase 7 [US5] → Phase 8 [US6]   (Brand - tuần tự)
Phase 9 [US7]                                      (User)
    ↓ (có thể chạy song song các nhóm)
Phase 10 (Dynamic Query + Docs)
```

### Parallel Opportunities
- Các task [P] trong cùng phase có thể chạy song song
- Phase 3-5 (Admin) và Phase 6-8 (Brand) và Phase 9 (User) độc lập, có thể làm song song nếu có nhiều dev

### Implementation Strategy (MVP)
1. Phase 1 + 2 → Foundation ready
2. Phase 3 [US1] → Admin Subscription APIs → Test → **MVP complete**
3. Phase 4 → Add Admin AI APIs
4. Phase 5 → Add Admin Renewal API
5. Phase 6 → Add Brand Customer APIs
6. Phase 7 → Add Brand Loyalty APIs
7. Phase 8 → Add Brand Catalog APIs
8. Phase 9 → Add User API
9. Phase 10 → Dynamic Query + Docs

---

---

## Phase 11: Convergence (Remaining Fixes)

**Purpose**: Fix gaps found during codebase assessment after initial implementation

### T049 — [HIGH] Sửa `GetAIErrorBreakdown`: bổ sung query tổng số request AI để tính `TotalRequests` và `ErrorRate` chính xác

- [x] T049 [P] Thêm repository method `GetTotalAIRequestCount(ctx, fromDate, toDate)` trong `dashboard_admin_repo.go` — `SELECT COUNT(*) FROM ai_usage_events WHERE created_at BETWEEN ? AND ?`
- [x] T050 Cập nhật `GetAIErrorBreakdown` trong `admin_dashboard_uc_impl.go` — gọi `GetTotalAIRequestCount` với `last 30 days` (default), populate `totalRequests` và tính `errorRate`

### T051 — [LOW] Cleanup `BrandItemCountByType` dead code trong dynamic query

- [x] T051 Xoá dead loop variable trong `dynamic_query_uc_impl.go` — bỏ `if i == 0 { _ = itemType }` tại dòng 199-201

### T052 — Verify

- [x] T052 Build và kiểm tra: `go build ./...`

---

## Implementation Notes

- Tuân thủ pattern: presenter → application → infrastructure
- Handler return error, dùng `shared_pres.Success()` + `shared_pres.WrapHandler()`
- Validation dùng `validation.BindQuery()` từ `internal/pkg/utils/validation`
- Error dùng `apperror` types
- Message constants khai báo đầu file handler
- Cost micro VND → VND: chia cho 1,000,000
- Phân khúc chi tiêu: CASE WHEN trong SQL
