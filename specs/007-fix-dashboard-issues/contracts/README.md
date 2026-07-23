# Contracts — Sửa Lỗi Dashboard

## Các interface liên quan

### 1. Subscription/PayOS Dashboard Reader (FR-003)

**File đích:** `internal/modules/deposit/presentation/contract/dashboard_reader.go` (nếu module `deposit` tồn tại) hoặc mở rộng `identityContract.IUserDashboardReader`

**Interface mới cần tạo:**
```go
type IDepositDashboardReader interface {
    GetSubscriptionStats(ctx context.Context, asOfDate time.Time) (*dto.SubscriptionStatsDTO, error)
}
```

**DTO mới:**
```go
type SubscriptionStatsDTO struct {
    TotalSubscriptionRevenueVND float64
    PayOSTransactionCount       int
    PayOSSuccessCount           int
}
```

**Kế hoạch:** Kiểm tra module `internal/modules/deposit/` có presentation/contract/ không. Nếu không có, inject trực tiếp từ repository của deposit module vào dashboard use case (tạm thời vi phạm module boundary nhẹ, cần tạo contract sau).

### 2. Brand Dashboard Reader — Mở rộng (FR-004)

**File đích:** `internal/modules/brand/presentation/contract/dashboard_reader.go`

**Method mới:**
```go
type IBrandDashboardReader interface {
    // ... existing methods ...
    
    GetItemFeedbackSummary(ctx context.Context, itemID uuid.UUID) (*dto.ItemFeedbackSummaryDTO, error)
}
```

**DTO mới trong** `internal/modules/brand/application/dto/dashboard_dto.go`:
```go
type ItemFeedbackSummaryDTO struct {
    VotesCount         int
    AvgRating          float64
    FeedbackSentiment  string                          // "positive" | "neutral" | "negative"
    TopPairedCategories []PairedCategoryStat
}
```

**Lưu ý:** DTO `PairedCategoryStat` đã tồn tại trong `brand_dashboard_dto.go`.

### 3. Dynamic Query Use Case — Không đổi interface

**File:** `internal/modules/dashboard/application/interface/usecase/dynamic_query_uc.go`

Interface `IDynamicQueryUseCase` giữ nguyên. Chỉ thay đổi implementation.
