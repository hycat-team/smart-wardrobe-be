# Technical Debt: Dashboard Cross-Module Direct Table Queries

**Phát hiện**: Sprint 2 (tech-debt remediation)  
**Trạng thái**: ĐANG XỬ LÝ trong Sprint 1 (feature `011-assess-schema-separation`)  
**Mức độ**: Thấp — dashboard là read-only reporting, risk thấp

## Vấn đề

Dashboard module (`internal/modules/dashboard/infrastructure/persistence/`)
đang query trực tiếp table của các module khác thay vì đi qua contract layer:

### File `dashboard_brand_repo.go`

| Bảng | Module sở hữu | Function |
|------|---------------|----------|
| `brand_customers` | brand | `GetCustomerAcquisition`, `GetBrandNewCustomerCount`, `GetBrandCustomerClaimRate` |
| `loyalty_accounts` | brand | `GetLoyaltyTierDistribution`, `GetCustomerSpendSegmentation` |
| `loyalty_tiers` | brand | `GetLoyaltyTierDistribution` (JOIN) |
| `loyalty_point_lots` | brand | `GetPointExpiryForecast` |
| `benefit_redemptions` | brand | `GetBenefitRedemptionAnalytics`, `GetBrandRedemptionCount` |
| `brand_benefits` | brand | `GetBenefitRedemptionAnalytics` (JOIN) |
| `brand_items` | brand | `GetCatalogStats`, `GetBrandItemCount`, `GetBrandItemCountByType` |

### File `dashboard_admin_repo.go`

| Bảng | Module sở hữu | Function |
|------|---------------|----------|
| `user_subscriptions` | subscription | `GetSubscriptionDistribution` |
| `deposit_transactions` | subscription | `GetRevenueBreakdown` |
| `subscription_renewal_attempts` | subscription | `GetRenewalStats` |
| `ai_usage_events` | aiexecutor | `GetAIUsageByOperation`, `GetTotalAIRequestCount`, `GetAIErrorBreakdown` |

### File `dashboard_user_repo.go`

| Bảng | Module sở hữu | Function |
|------|---------------|----------|
| `wardrobe_items` | wardrobe | `GetWardrobeCategoryDistribution` |
| `fashion_items` | fashion | `GetWardrobeCategoryDistribution` (JOIN) |
| `categories` | fashion | `GetWardrobeCategoryDistribution` (JOIN) |

## Import domain constants cross-module

Import `domain/constants/` của module khác được Hiến pháp II cho phép (exception), NHƯNG phải dùng symbol (không hardcode giá trị):

| File | Import | Ghi chú |
|------|--------|---------|
| `dashboard_brand_repo.go` | `brand/domain/constants/loyaltypointlotstatus` | Hợp lệ (dùng symbol) |
| `dashboard_admin_repo.go` | `aiexecutor/domain/constants/aiusageeventstatus` | Hợp lệ (dùng symbol) |
| `dashboard_admin_repo.go` | `subscription/domain/constants/depositstatus`, `deposittransactiontype` | Hợp lệ (dùng symbol) |

## Tác động

- Vi phạm kiến trúc cross-module (architecture.md §III, §VI)
- Nếu schema table thay đổi, dashboard sẽ silent fail
- Không tận dụng được business logic từ module sở hữu

## Hướng fix (đã chốt — feature 011)

Với mỗi query, thêm 1 method vào contract interface của module sở hữu
(vd: `IBrandDashboardReader`, `ISubscriptionDashboardReader`, `IAICostDashboardReader`,
`IWardrobeDashboardReader`, `IFashionDashboardReader`),
dashboard gọi contract thay vì query thẳng DB:
- `IBrandDashboardReader`: `GetCustomerAcquisition`, `GetLoyaltyTierDistribution`, `GetPointExpiryForecast`, `GetBenefitRedemptionAnalytics`, `GetCatalogStats`, `GetCustomerSpendSegmentation`, `GetBrandNewCustomerCount`, `GetBrandCustomerClaimRate`, `GetBrandRedemptionCount`, `GetBrandItemCount`, `GetBrandItemCountByType`
- `ISubscriptionDashboardReader`: `GetSubscriptionDistribution`, `GetRevenueBreakdown`, `GetRenewalStats`
- `IAICostDashboardReader`: `GetAIUsageByOperation`, `GetTotalAIRequestCount`, `GetAIErrorBreakdown`
- `IWardrobeDashboardReader`: `GetWardrobeCategoryDistribution`
- `IFashionDashboardReader` (mới): `GetCategoriesByIDs`, `GetCategoryIDsByFashionItemIDs`
