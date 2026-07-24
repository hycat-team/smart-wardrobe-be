# Implementation Plan: Mở Rộng Dashboard

**Branch**: `008-dashboard-expansion` | **Date**: 2026-07-24 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/008-dashboard-expansion/spec.md`

---

## Summary

Mở rộng hệ thống Dashboard với 12 API endpoints mới (5 Admin + 6 Brand + 1 User) và 8 Dynamic Query metrics mới (3 Admin + 5 Brand). Các API query trực tiếp từ bảng nghiệp vụ real-time. Các metric dynamic query tận dụng field snapshot đã có + tính toán runtime. Kết thúc plan tạo file `docs/api/dashboard-api-v2.md`.

---

## Technical Context

**Language/Version**: Go 1.22+

**Framework**: Gin v1.x (HTTP), GORM v2.x (ORM), Google Wire (DI)

**Storage**: PostgreSQL 16 (bảng nghiệp vụ + dashboard snapshots), pgvector

**Testing**: `make test` (Go testing + testify), `make check` (lint + format)

**Target Platform**: Linux server (Docker), REST API

**Project Type**: Modular monolith backend (Gin-based HTTP service)

**Performance Goals**: API response < 2s cho dataset < 100k records

**Constraints**: Tuân thủ module boundaries (không cross-entity Preload/Join), handler → use case → repository pattern

**Scale/Scope**: ~10k users, ~100 brands, ~500k transactions

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Không tìm thấy file `.specify/memory/constitution.md`. Mặc định tuân thủ:
- **Module isolation**: Không import entity/repository cross-module
- **Layer separation**: presentation → application → infrastructure (handler → use case → repository)
- **Contract pattern**: Cross-module qua `presentation/contract/`
- **Handler pattern**: Error-returning handler + `shared_pres.WrapHandler`
- **Response helpers**: `shared_pres.Success()`, validation via `validation.BindQuery`
- **Error pattern**: `apperror` types, không raw DB error

**Gate result: PASS** — không có vi phạm.

---

## Project Structure

### Documentation (this feature)

```text
specs/008-dashboard-expansion/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── admin-apis.md
│   ├── brand-apis.md
│   └── user-apis.md
└── tasks.md             # (created by /speckit-tasks)
```

### Source Code (repository root)

```text
internal/
├── modules/
│   └── dashboard/
│       ├── application/
│       │   ├── dto/
│       │   │   ├── admin_dashboard_dto.go        # + 5 response DTOs mới
│       │   │   ├── brand_dashboard_dto.go         # + 6 response DTOs mới
│       │   │   └── user_dashboard_dto.go          # + 1 response DTO mới
│       │   ├── interface/usecase/
│       │   │   ├── admin_dashboard_uc.go          # + 5 methods mới
│       │   │   ├── brand_dashboard_uc.go          # + 6 methods mới
│       │   │   └── user_dashboard_uc.go           # + 1 method mới
│       │   └── usecase/
│       │       ├── admin_dashboard_uc_impl.go     # + 5 use case methods
│       │       ├── brand_dashboard_uc_impl.go     # + 6 use case methods
│       │       ├── user_dashboard_uc_impl.go      # + 1 use case method
│       │       └── dynamic_query_uc_impl.go       # + 8 metric mappings
│       ├── domain/
│       │   ├── entities/
│       │   │   └── dashboard_snapshots.go          # Giữ nguyên (fields đã có)
│       │   └── repositories/
│       │       └── dashboard_repository.go         # + 12 repository methods
│       ├── infrastructure/persistence/
│       │   ├── dashboard_admin_repo.go            # + methods cho admin APIs
│       │   ├── dashboard_brand_repo.go            # + methods cho brand APIs
│       │   └── dashboard_user_repo.go             # + methods cho user API
│       └── presentation/handler/
│           ├── admin_dashboard_handler.go          # + 5 handlers
│           ├── brand_dashboard_handler.go          # + 6 handlers
│           └── user_dashboard_handler.go           # + 1 handler
├── api/
│   └── routes/
│       └── admin/router.go                        # + 5 routes
│       └── brand/router.go                        # + 6 routes
│       └── wardrobe/router.go                     # + 1 route

docs/
└── api/
    └── dashboard-api-v2.md                        # NEW: full API spec
```

---

## Complexity Tracking

Không có vi phạm — không cần justification.

---

## Phase 0: Research & Design Decisions

Xem [research.md](research.md) cho chi tiết.

### Key Decisions:
1. **Brand dynamic query metrics**: Các metric mới cho brand (`new_customers`, `customer_claim_rate`, `benefit_redemption_count`, `brand_item_count`, `brand_item_count_by_type`) sẽ được query live từ bảng nguồn thay vì snapshot, vì cần group-by (`item_type`) và tính toán tỷ lệ real-time.
2. **Repository pattern**: Tách riêng file repository cho từng nhóm (admin/brand/user) để tránh một file quá lớn.
3. **Không schema migration**: Các API mới query từ bảng có sẵn, không cần migration.

---

## Phase 1: Design Artifacts

Xem:
- [data-model.md](data-model.md) — entity fields và query logic
- [contracts/admin-apis.md](contracts/admin-apis.md) — Admin API contracts
- [contracts/brand-apis.md](contracts/brand-apis.md) — Brand API contracts
- [contracts/user-apis.md](contracts/user-apis.md) — User API contract
- [quickstart.md](quickstart.md) — validation guide

---

## API Call Chains

### Admin APIs

#### 1. GET /api/v1/admin/dashboard/subscriptions/distribution
```
Router (admin/router.go)
  → shared_pres.WrapHandler(h.GetSubscriptionDistribution)
    → AdminDashboardHandler.GetSubscriptionDistribution (presentation/handler/admin_dashboard_handler.go)
      → validation.BindQuery(c, &req)
      → h.adminUC.GetSubscriptionDistribution(ctx)
        → IAdminDashboardUseCase.GetSubscriptionDistribution (application/interface/usecase/admin_dashboard_uc.go)
          → AdminDashboardUseCase.GetSubscriptionDistribution (application/usecase/admin_dashboard_uc_impl.go)
            → uc.dashboardRepo.GetSubscriptionDistribution(ctx)
              → IDashboardRepository.GetSubscriptionDistribution (domain/repositories/dashboard_repository.go)
                → DashboardRepository.GetSubscriptionDistribution (infrastructure/persistence/dashboard_admin_repo.go)
                  → db.Model(&user_subscriptions{}).Select(...).Group(...).Find(...)
      → shared_pres.Success(c, msg, res)
```

#### 2. GET /api/v1/admin/dashboard/subscriptions/revenue-breakdown
```
Router → WrapHandler(h.GetRevenueBreakdown)
  → Handler (admin_dashboard_handler.go)
    → validation.BindQuery(c, &req)  // fromDate, toDate
    → h.adminUC.GetRevenueBreakdown(ctx, fromDate, toDate)
      → UseCase Interface → IAdminDashboardUseCase.GetRevenueBreakdown
        → UseCase Impl (admin_dashboard_uc_impl.go)
          → uc.dashboardRepo.GetRevenueBreakdown(ctx, fromDate, toDate)
            → Repository Interface → IDashboardRepository.GetRevenueBreakdown
              → Repository Impl (dashboard_admin_repo.go)
                → db.Model(&deposit_transactions{}).Where(status=1, type=DIRECT_PURCHASE).Select(...).Group(plan_code).Find(...)
      → shared_pres.Success(c, msg, res)
```

#### 3. GET /api/v1/admin/dashboard/ai/usage-by-operation
```
Router → WrapHandler(h.GetAIUsageByOperation)
  → Handler → validation.BindQuery(c, &req)  // fromDate, toDate
    → h.adminUC.GetAIUsageByOperation(ctx, fromDate, toDate)
      → UseCase Interface → IAdminDashboardUseCase.GetAIUsageByOperation
        → UseCase Impl
          → uc.dashboardRepo.GetAIUsageByOperation(ctx, fromDate, toDate)
            → Repository → IDashboardRepository.GetAIUsageByOperation
              → Impl (dashboard_admin_repo.go)
                → db.Model(&ai_usage_events{}).Select(operation, logical_route, count, sum(cost)).Group(operation, logical_route).Find(...)
      → shared_pres.Success(c, msg, res)
```

#### 4. GET /api/v1/admin/dashboard/ai/error-breakdown
```
Router → WrapHandler(h.GetAIErrorBreakdown)
  → Handler → h.adminUC.GetAIErrorBreakdown(ctx)
    → UseCase Interface → IAdminDashboardUseCase.GetAIErrorBreakdown
      → UseCase Impl
        → uc.dashboardRepo.GetAIErrorBreakdown(ctx)
          → Repository → IDashboardRepository.GetAIErrorBreakdown
            → Impl (dashboard_admin_repo.go)
              → db.Model(&ai_usage_events{}).Where(status in (FAILED, ERROR, ...)).Select(error_code, provider, model, count).Group(error_code, provider, model).Find(...)
      → shared_pres.Success(c, msg, res)
```

#### 5. GET /api/v1/admin/dashboard/subscriptions/renewal-stats
```
Router → WrapHandler(h.GetRenewalStats)
  → Handler → h.adminUC.GetRenewalStats(ctx)
    → UseCase Interface → IAdminDashboardUseCase.GetRenewalStats
      → UseCase Impl
        → uc.dashboardRepo.GetRenewalStats(ctx)
          → Repository → IDashboardRepository.GetRenewalStats
            → Impl (dashboard_admin_repo.go)
              → db.Model(&subscription_renewal_attempts{}).Select(count, sum(case when status=SUCCESS), avg(attempt_count)).Find(...)
      → shared_pres.Success(c, msg, res)
```

### Brand APIs

#### 6. GET /api/v1/brand/dashboard/customers/acquisition?brandId=<uuid>
```
Router (brand/router.go)
  → WrapHandler(h.GetCustomerAcquisition)
    → BrandDashboardHandler.GetCustomerAcquisition (brand_dashboard_handler.go)
      → validation.BindQuery(c, &req)  // brandId
      → h.brandUC.GetCustomerAcquisition(ctx, brandID)
        → IBrandDashboardUseCase.GetCustomerAcquisition (brand_dashboard_uc.go)
          → BrandDashboardUseCase.GetCustomerAcquisition (brand_dashboard_uc_impl.go)
            → uc.dashboardRepo.GetCustomerAcquisition(ctx, brandID)
              → IDashboardRepository.GetCustomerAcquisition (dashboard_repository.go)
                → DashboardRepository.GetCustomerAcquisition (dashboard_brand_repo.go)
                  → db.Model(&brand_customers{}).Where(brand_id).Select(joined_source, count).Group(joined_source).Find(...)
      → shared_pres.Success(c, msg, res)
```

#### 7. GET /api/v1/brand/dashboard/loyalty/tier-distribution?brandId=<uuid>
```
Router → WrapHandler(h.GetLoyaltyTierDistribution)
  → Handler → BindQuery(brandId)
    → h.brandUC.GetLoyaltyTierDistribution(ctx, brandID)
      → UseCase Interface → IBrandDashboardUseCase.GetLoyaltyTierDistribution
        → UseCase Impl
          → uc.dashboardRepo.GetLoyaltyTierDistribution(ctx, brandID)
            → Repository → IDashboardRepository.GetLoyaltyTierDistribution
              → Impl (dashboard_brand_repo.go)
                → db.Model(&loyalty_accounts{}).Joins("loyalty_tiers").Where(brand_id).Select(tier_id, tier_name, count).Group(tier_id).Find(...)
      → shared_pres.Success(c, msg, res)
```

#### 8. GET /api/v1/brand/dashboard/loyalty/point-expiry-forecast?brandId=<uuid>
```
Router → WrapHandler(h.GetPointExpiryForecast)
  → Handler → BindQuery(brandId)
    → h.brandUC.GetPointExpiryForecast(ctx, brandID)
      → UseCase Interface → IBrandDashboardUseCase.GetPointExpiryForecast
        → UseCase Impl
          → uc.dashboardRepo.GetPointExpiryForecast(ctx, brandID)
            → Repository → IDashboardRepository.GetPointExpiryForecast
              → Impl (dashboard_brand_repo.go)
                → db.Model(&loyalty_point_lots{}).Where(brand_id, ACTIVE, remaining>0).Select(date_trunc('month', expires_at), sum(remaining_points)).Group(1).Order(1).Find(...)
      → shared_pres.Success(c, msg, res)
```

#### 9. GET /api/v1/brand/dashboard/benefits/redemption-analytics?brandId=<uuid>
```
Router → WrapHandler(h.GetBenefitRedemptionAnalytics)
  → Handler → BindQuery(brandId)
    → h.brandUC.GetBenefitRedemptionAnalytics(ctx, brandID)
      → UseCase Interface → IBrandDashboardUseCase.GetBenefitRedemptionAnalytics
        → UseCase Impl
          → uc.dashboardRepo.GetBenefitRedemptionAnalytics(ctx, brandID)
            → Repository → IDashboardRepository.GetBenefitRedemptionAnalytics
              → Impl (dashboard_brand_repo.go)
                → db.Model(&benefit_redemptions{}).Joins("brand_benefits").Where(brand_id).Select(benefit_id, name, count).Group(benefit_id, name).Order(count DESC).Limit(10).Find(...)
      → shared_pres.Success(c, msg, res)
```

#### 10. GET /api/v1/brand/dashboard/operations/catalog-stats?brandId=<uuid>
```
Router → WrapHandler(h.GetCatalogStats)
  → Handler → BindQuery(brandId)
    → h.brandUC.GetCatalogStats(ctx, brandID)
      → UseCase Interface → IBrandDashboardUseCase.GetCatalogStats
        → UseCase Impl
          → uc.dashboardRepo.GetCatalogStats(ctx, brandID)
            → Repository → IDashboardRepository.GetCatalogStats
              → Impl (dashboard_brand_repo.go)
                → db.Model(&brand_items{}).Where(brand_id).Select(status, count).Group(status).Find(...)
                + select item_type, count group by item_type
      → shared_pres.Success(c, msg, res)
```

#### 11. GET /api/v1/brand/dashboard/customers/spend-segmentation?brandId=<uuid>
```
Router → WrapHandler(h.GetCustomerSpendSegmentation)
  → Handler → BindQuery(brandId)
    → h.brandUC.GetCustomerSpendSegmentation(ctx, brandID)
      → UseCase Interface → IBrandDashboardUseCase.GetCustomerSpendSegmentation
        → UseCase Impl
          → uc.dashboardRepo.GetCustomerSpendSegmentation(ctx, brandID)
            → Repository → IDashboardRepository.GetCustomerSpendSegmentation
              → Impl (dashboard_brand_repo.go)
                → db.Model(&loyalty_accounts{}).Where(brand_id).
                    Select(CASE WHEN total_spend ... END as segment, count(*)).
                    Group(segment).Find(...)
      → shared_pres.Success(c, msg, res)
```

### User API

#### 12. GET /api/v1/me/dashboard/wardrobe/category-distribution
```
Router (wardrobe/router.go)
  → WrapHandler(h.GetWardrobeCategoryDistribution)
    → UserDashboardHandler.GetWardrobeCategoryDistribution (user_dashboard_handler.go)
      → userID := contextutils.GetUserId(c)
      → h.userUC.GetWardrobeCategoryDistribution(ctx, userID)
        → IUserDashboardUseCase.GetWardrobeCategoryDistribution (user_dashboard_uc.go)
          → UserDashboardUseCase.GetWardrobeCategoryDistribution (user_dashboard_uc_impl.go)
            → uc.dashboardRepo.GetWardrobeCategoryDistribution(ctx, userID)
              → IDashboardRepository.GetWardrobeCategoryDistribution (dashboard_repository.go)
                → DashboardRepository.GetWardrobeCategoryDistribution (dashboard_user_repo.go)
                  → db.Model(&wardrobe_items{}).
                      Joins("fashion_items").Joins("categories").
                      Where(user_id, is_deleted=false).
                      Select(category_id, category_name, count(*)).
                      Group(category_id, category_name).Find(...)
      → shared_pres.Success(c, msg, res)
```

### Dynamic Query Metrics

#### Admin Metrics (dynamic_query_uc_impl.go)

```go
// Trong switch m của hàm ExecuteDynamicQuery, thêm:
case metrickey.NewUsers:
    metricValues[m] = float64(s.NewUsers)
case metrickey.AIErrorCount:
    metricValues[m] = float64(s.AIErrorCount)
case metrickey.AIErrorRate:
    // computed: ai_error_count / ai_requests_count * 100
    if s.AIRequestsCount > 0 {
        metricValues[m] = float64(s.AIErrorCount) / float64(s.AIRequestsCount) * 100.0
    } else {
        metricValues[m] = 0
    }
```

Đồng thời update `metricValueFromAdminSnapshot` và `getMetricAggType`.

#### Brand Metrics — Live Query Approach

Các brand metric không có sẵn trong snapshot, cần query live:

```go
// dynamic_query_uc_impl.go - thêm case targetdashboard.Brand:
case metrickey.NewCustomers:
    val, _ := uc.dashboardRepo.GetBrandNewCustomerCount(ctx, *req.BrandID, fromDate, toDate)
    metricValues[m] = val
case metrickey.CustomerClaimRate:
    rate, _ := uc.dashboardRepo.GetBrandCustomerClaimRate(ctx, *req.BrandID)
    metricValues[m] = rate
// ... tương tự cho benefit_redemption_count, brand_item_count, brand_item_count_by_type
```

Thêm các repository method tương ứng trong `dashboard_brand_repo.go`.

---

## Complexity Tracking

Không có vi phạm.

---

## Output Artifacts

| Artifact | Path | Description |
|----------|------|-------------|
| Plan | `specs/008-dashboard-expansion/plan.md` | This file |
| Research | `specs/008-dashboard-expansion/research.md` | Design decisions |
| Data Model | `specs/008-dashboard-expansion/data-model.md` | Entity fields & queries |
| Contracts | `specs/008-dashboard-expansion/contracts/*.md` | API contracts |
| Quickstart | `specs/008-dashboard-expansion/quickstart.md` | Validation guide |
| API Docs v2 | `docs/api/dashboard-api-v2.md` | Full API documentation (tạo sau implement) |
