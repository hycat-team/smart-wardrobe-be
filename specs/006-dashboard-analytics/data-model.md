# Data Model & Schema Specification: Dashboard & Analytics System

**Feature**: `006-dashboard-analytics`  
**Date**: 2026-07-23  

---

## 1. Database Schema (Goose Migration)

Hệ thống bổ sung 3 bảng Read Model Snapshots trong PostgreSQL thông qua Goose Migration (`database/migrations/`).

### 1.1. Bảng `dashboard_admin_daily_snapshots`

Lưu trữ ảnh chụp số liệu tổng quan hệ thống hàng ngày phục vụ Admin Dashboard.

```sql
CREATE TABLE IF NOT EXISTS public.dashboard_admin_daily_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_date DATE NOT NULL UNIQUE,
    total_users INT NOT NULL DEFAULT 0,
    new_users INT NOT NULL DEFAULT 0,
    active_users_dau INT NOT NULL DEFAULT 0,
    total_brands INT NOT NULL DEFAULT 0,
    pending_review_brands INT NOT NULL DEFAULT 0,
    active_campaigns INT NOT NULL DEFAULT 0,
    ai_requests_count INT NOT NULL DEFAULT 0,
    ai_error_count INT NOT NULL DEFAULT 0,
    ai_paid_tokens BIGINT NOT NULL DEFAULT 0,
    ai_actual_cost_micro_vnd BIGINT NOT NULL DEFAULT 0,
    subscription_revenue_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    net_ai_margin_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    payos_transaction_count INT NOT NULL DEFAULT 0,
    payos_success_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_snapshots_date ON public.dashboard_admin_daily_snapshots(snapshot_date DESC);
```

---

### 1.2. Bảng `dashboard_brand_daily_snapshots`

Lưu trữ ảnh chụp số liệu kinh doanh hàng ngày theo từng Brand phục vụ B2B Brand Portal.

```sql
CREATE TABLE IF NOT EXISTS public.dashboard_brand_daily_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES public.brands(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    total_members INT NOT NULL DEFAULT 0,
    points_issued INT NOT NULL DEFAULT 0,
    points_redeemed INT NOT NULL DEFAULT 0,
    points_expiring_30d INT NOT NULL DEFAULT 0,
    points_liability_value_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    active_vouchers INT NOT NULL DEFAULT 0,
    vouchers_redeemed INT NOT NULL DEFAULT 0,
    digital_sample_votes_count INT NOT NULL DEFAULT 0,
    digital_sample_avg_rating NUMERIC(3, 2) NOT NULL DEFAULT 0.00,
    open_cs_tickets INT NOT NULL DEFAULT 0,
    cs_first_response_avg_seconds INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_brand_daily_snapshot UNIQUE (brand_id, snapshot_date)
);

CREATE INDEX idx_brand_snapshots_brand_date ON public.dashboard_brand_daily_snapshots(brand_id, snapshot_date DESC);
```

---

### 1.3. Bảng `dashboard_user_wardrobe_summaries`

Lưu trữ thông số tổng quan tủ đồ cá nhân B2C của từng User.

```sql
CREATE TABLE IF NOT EXISTS public.dashboard_user_wardrobe_summaries (
    user_id UUID PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    total_items INT NOT NULL DEFAULT 0,
    total_wardrobe_value_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    average_cost_per_wear_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    underutilized_items_count INT NOT NULL DEFAULT 0,
    category_distribution JSONB NOT NULL DEFAULT '{}'::jsonb,
    color_distribution JSONB NOT NULL DEFAULT '{}'::jsonb,
    ai_quota_remaining INT NOT NULL DEFAULT 0,
    expiring_vouchers_count INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_wardrobe_summaries_updated ON public.dashboard_user_wardrobe_summaries(updated_at DESC);
```

---

## 2. Go Entities (`internal/modules/dashboard/domain/entities/`)

```go
package entities

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type DashboardAdminDailySnapshot struct {
	ID                       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SnapshotDate             time.Time `gorm:"type:date;not null;uniqueIndex"`
	TotalUsers               int       `gorm:"not null;default:0"`
	NewUsers                 int       `gorm:"not null;default:0"`
	ActiveUsersDau           int       `gorm:"not null;default:0"`
	TotalBrands              int       `gorm:"not null;default:0"`
	PendingReviewBrands      int       `gorm:"not null;default:0"`
	ActiveCampaigns          int       `gorm:"not null;default:0"`
	AIRequestsCount          int       `gorm:"not null;default:0"`
	AIErrorCount             int       `gorm:"not null;default:0"`
	AIPaidTokens             int64     `gorm:"not null;default:0"`
	AIActualCostMicroVND     int64     `gorm:"not null;default:0"`
	SubscriptionRevenueVND   float64   `gorm:"type:numeric(12,2);not null;default:0"`
	NetAIMarginVND           float64   `gorm:"type:numeric(12,2);not null;default:0"`
	PayOSTransactionCount    int       `gorm:"not null;default:0"`
	PayOSSuccessCount        int       `gorm:"not null;default:0"`
	CreatedAt                time.Time `gorm:"not null;default:now()"`
	UpdatedAt                time.Time `gorm:"not null;default:now()"`
}

type DashboardBrandDailySnapshot struct {
	ID                         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BrandID                    uuid.UUID `gorm:"type:uuid;not null;index"`
	SnapshotDate               time.Time `gorm:"type:date;not null"`
	TotalMembers               int       `gorm:"not null;default:0"`
	PointsIssued               int       `gorm:"not null;default:0"`
	PointsRedeemed             int       `gorm:"not null;default:0"`
	PointsExpiring30d          int       `gorm:"not null;default:0"`
	PointsLiabilityValueVND    float64   `gorm:"type:numeric(12,2);not null;default:0"`
	ActiveVouchers             int       `gorm:"not null;default:0"`
	VouchersRedeemed           int       `gorm:"not null;default:0"`
	DigitalSampleVotesCount    int       `gorm:"not null;default:0"`
	DigitalSampleAvgRating     float64   `gorm:"type:numeric(3,2);not null;default:0"`
	OpenCSTickets              int       `gorm:"not null;default:0"`
	CSFirstResponseAvgSeconds  int       `gorm:"not null;default:0"`
	CreatedAt                  time.Time `gorm:"not null;default:now()"`
	UpdatedAt                  time.Time `gorm:"not null;default:now()"`
}

type DashboardUserWardrobeSummary struct {
	UserID                   uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TotalItems               int            `gorm:"not null;default:0"`
	TotalWardrobeValueVND    float64        `gorm:"type:numeric(12,2);not null;default:0"`
	AverageCostPerWearVND    float64        `gorm:"type:numeric(12,2);not null;default:0"`
	UnderutilizedItemsCount  int            `gorm:"not null;default:0"`
	CategoryDistribution     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	ColorDistribution        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	AIQuotaRemaining         int            `gorm:"not null;default:0"`
	ExpiringVouchersCount    int            `gorm:"not null;default:0"`
	UpdatedAt                time.Time      `gorm:"not null;default:now()"`
}
```
