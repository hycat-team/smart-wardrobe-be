# Contracts — Sửa Lỗi Dashboard

## Các interface liên quan

### 1. Subscription/PayOS Dashboard Reader (FR-003)

**File đích:** `internal/modules/subscription/presentation/contract/dashboard_reader.go`

**Interface mới cần tạo:**
```go
type ISubscriptionDashboardReader interface {
    GetAdminSubscriptionStats(ctx context.Context, asOfDate time.Time) (*dto.SubscriptionStatsDTO, error)
}
```

**DTO mới** trong `internal/modules/subscription/application/dto/`:
```go
type SubscriptionStatsDTO struct {
    TotalSubscriptionRevenueVND float64
    PayOSTransactionCount       int
    PayOSSuccessCount           int
}
```

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
    OutfitIDs          []uuid.UUID                     // outfit_ids từ digital_sample_responses
}
```

### 3. Wardrobe Dashboard Reader — Mở rộng (FR-004)

**File đích:** `internal/modules/wardrobe/presentation/contract/dashboard_reader.go`

**Method mới:**
```go
type IWardrobeDashboardReader interface {
    // ... existing methods ...
    
    GetOutfitItemFashionItemIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error)
}
```

### 4. Dynamic Query Use Case — Không đổi interface

**File:** `internal/modules/dashboard/application/interface/usecase/dynamic_query_uc.go`

Interface `IDynamicQueryUseCase` giữ nguyên. Chỉ thay đổi implementation.
