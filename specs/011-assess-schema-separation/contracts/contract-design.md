# Contracts: Xử lý tech debt cross-module & tách schema theo module

**Phase 1 output** — thiết kế contract cho Sprint 1 (Sprint 2 không thêm contract mới).

## 1. Wardrobe ↔ Fashion contracts

### 1.1 `IWardrobeContract` — thêm method

File: `internal/modules/wardrobe/presentation/contract/wardrobe_contract.go`

```go
// GetUserFashionItemIDs trả về danh sách fashion_item_id thuộc tủ của user
// (chỉ SELECT cột, không trả entity wardrobe). Dùng bởi fashion module
// cho vector search trên bảng fashion_items của chính mình.
GetUserFashionItemIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
```

### 1.2 `IFashionContract` — thêm method

File: `internal/modules/fashion/presentation/contract/fashion_contract.go`

> **Phân định (I3)**: Chỉ các thao tác ĐỒNG BỘ được thêm vào `IFashionContract`. Các thao tác claim retry / processing state trên `fashion_items` (claim_manual_retry, claim_stale_retry, mark_failed, mark_needs_review, complete_success) KHÔNG đi qua contract đồng bộ — chuyển sang outbox event `wardrobe.processing.state` (xem §3) do fashion consumer xử lý. `GetStaleProcessingItems` thuộc `IWardrobeContract` (operate trên `wardrobe_items`) được giữ nguyên, KHÔNG trùng lặp sang `IFashionContract`.

```go
// GetStaleProcessingInfo trả thông tin stale (last_processing_attempt_at,
// processing_started_at, processing_version) của các fashion_item để wardrobe
// xác định item stale mà không cần JOIN bảng fashion_items.
GetStaleProcessingInfo(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.FashionProcessingInfoDTO, error)

// ReassignCategoryForCatalogItems cập nhật category_id cho fashion_items
// (admin bulk operation, giữ đồng bộ qua contract).
ReassignCategoryForCatalogItems(ctx context.Context, fromCategoryID uuid.UUID, toCategoryID uuid.UUID, wardrobeItemIDs []uuid.UUID) error

// ResetProcessingState reset các processing fields trên fashion_items
// (review_reason, processing_error_reason, processing_retry_count, processing_started_at,
// last_processing_attempt_at, increment processing_version) — dùng bởi wardrobe
// trong RetryWardrobeAnalysis khi user claim manual retry.
ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error
```

## 2. Dashboard Reader contracts

### 2.1 `IBrandDashboardReader` — thêm method

File: `internal/modules/brand/presentation/contract/dashboard_reader.go`

```go
GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]dto.DashboardCustomerAcquisitionDTO, error)
GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]dto.DashboardTierDistributionDTO, error)
GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]dto.DashboardPointExpiryDTO, error)
GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]dto.DashboardBenefitRedemptionDTO, error)
GetCatalogStats(ctx context.Context, brandID uuid.UUID) (*dto.DashboardCatalogStatsDTO, error)
GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]dto.DashboardSpendSegmentDTO, error)
GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error)
GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error)
GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error)
GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (int64, error)
GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]dto.DashboardItemTypeCountDTO, error)
```

### 2.2 `ISubscriptionDashboardReader` — thêm method

File: `internal/modules/subscription/presentation/contract/dashboard_reader.go`

```go
GetSubscriptionDistribution(ctx context.Context) ([]dto.DashboardSubscriptionDistributionDTO, error)
GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.DashboardRevenueBreakdownDTO, error)
GetRenewalStats(ctx context.Context) (*dto.DashboardRenewalStatsDTO, error)
```

### 2.3 `IAICostDashboardReader` — thêm method

File: `internal/modules/aiexecutor/presentation/contract/dashboard_reader.go`

```go
GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]dto.DashboardAIUsageByOperationDTO, error)
GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error)
GetAIErrorBreakdown(ctx context.Context) ([]dto.DashboardAIErrorBreakdownDTO, error)
```

### 2.4 `IWardrobeDashboardReader` — thêm method

File: `internal/modules/wardrobe/presentation/contract/dashboard_reader.go`

```go
GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) ([]dto.DashboardWardrobeCategoryDTO, error)
```

### 2.5 `IFashionDashboardReader` — tạo MỚI

File: `internal/modules/fashion/presentation/contract/dashboard_reader.go`

```go
// GetCategoryByIDs trả thông tin category (id, name) theo danh sách category_id
// — phục vụ dashboard stitch thông tin category cho wardrobe items mà không
// cần JOIN trực tiếp bảng categories (fashion module sở hữu categories).
GetCategoriesByIDs(ctx context.Context, categoryIDs []uuid.UUID) ([]dto.DashboardCategoryDTO, error)

// GetCategoryIDsByFashionItemIDs trả mapping fashion_item_id → category_id
// cho danh sách fashion_item_ids (phục vụ dashboard user tính category distribution).
GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.DashboardFashionItemCategoryDTO, error)
```

> **Ghi chú**: `GetWardrobeCategoryDistribution` (dashboard_user_repo.go:26) join `wardrobe_items` + `fashion_items` + `categories`. Phân bổ refactor: wardrobe module trả wardrobe items thuộc user (qua `IWardrobeDashboardReader`), fashion module trả mapping fashion_item_id → category (qua `IFashionDashboardReader`), dashboard stitch bằng `sliceutils.IndexBy`/`GroupBy`.

## 3. Outbox events (Sprint 1 — critical path worker)

Topic: `wardrobe.processing.state`

Payload (JSON):
```json
{
  "item_id": "uuid",
  "fashion_item_id": "uuid",
  "action": "claim_manual_retry | claim_stale_retry | mark_failed | mark_needs_review | complete_success",
  "processing_version": 3,
  "reason": "optional",
  "review_reason": "optional",
  "occurred_at": "RFC3339"
}
```

Consumer: fashion module worker — update `fashion_items` tương ứng (idempotent theo processing_version).

## 4. Nguyên tắc chung

- Mọi contract method dùng DTO, không dùng entity module khác (Hiến pháp II).
- Cross-module stitching dùng `sliceutils.IndexBy`/`GroupBy` (không nested loop).
- DTO response JSON cho frontend dùng camelCase; DTO nội bộ event/search có thể snake_case.
