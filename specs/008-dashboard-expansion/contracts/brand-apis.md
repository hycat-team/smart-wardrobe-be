# Brand Dashboard API Contracts

**Base path**: `/api/v1/brand/dashboard`
**Auth**: JWT + brand membership check
**Query params (all)**: `brandId` (UUID, required)

---

## 6. GET /customers/acquisition

**Response 200**:
```json
{
  "success": true,
  "data": {
    "brandId": "uuid",
    "totalCustomers": 250,
    "sources": [
      { "source": "walk_in", "customerCount": 100, "percentage": 40.0 },
      { "source": "referral", "customerCount": 80, "percentage": 32.0 },
      { "source": "online", "customerCount": 50, "percentage": 20.0 },
      { "source": "QR", "customerCount": 20, "percentage": 8.0 }
    ]
  }
}
```

**DTOs**:
```go
type CustomerAcquisitionDTO struct {
    Source         string `json:"source"`
    CustomerCount  int    `json:"customerCount"`
    Percentage     float64 `json:"percentage"`
}

type CustomerAcquisitionRes struct {
    BrandID        uuid.UUID               `json:"brandId"`
    TotalCustomers int                     `json:"totalCustomers"`
    Sources        []CustomerAcquisitionDTO `json:"sources"`
}
```

---

## 7. GET /loyalty/tier-distribution

**Response 200**:
```json
{
  "success": true,
  "data": {
    "brandId": "uuid",
    "totalMembers": 250,
    "tiers": [
      { "tierId": "uuid", "tierName": "Bronze", "tierRank": 1, "memberCount": 150, "percentage": 60.0 },
      { "tierId": "uuid", "tierName": "Silver", "tierRank": 2, "memberCount": 70, "percentage": 28.0 },
      { "tierId": "uuid", "tierName": "Gold", "tierRank": 3, "memberCount": 30, "percentage": 12.0 }
    ]
  }
}
```

**DTOs**:
```go
type TierDistributionDTO struct {
    TierID       uuid.UUID `json:"tierId"`
    TierName     string    `json:"tierName"`
    TierRank     int       `json:"tierRank"`
    MemberCount  int       `json:"memberCount"`
    Percentage   float64   `json:"percentage"`
}

type TierDistributionRes struct {
    BrandID      uuid.UUID             `json:"brandId"`
    TotalMembers int                   `json:"totalMembers"`
    Tiers        []TierDistributionDTO `json:"tiers"`
}
```

---

## 8. GET /loyalty/point-expiry-forecast

**Response 200**:
```json
{
  "success": true,
  "data": {
    "brandId": "uuid",
    "forecast": [
      { "month": "2026-08", "pointsExpiring": 5000 },
      { "month": "2026-09", "pointsExpiring": 12000 },
      { "month": "2026-10", "pointsExpiring": 3000 }
    ]
  }
}
```

**DTOs**:
```go
type PointExpiryForecastDTO struct {
    Month           string `json:"month"`
    PointsExpiring  int    `json:"pointsExpiring"`
}

type PointExpiryForecastRes struct {
    BrandID  uuid.UUID                `json:"brandId"`
    Forecast []PointExpiryForecastDTO `json:"forecast"`
}
```

---

## 9. GET /benefits/redemption-analytics

**Response 200**:
```json
{
  "success": true,
  "data": {
    "brandId": "uuid",
    "totalRedemptions": 120,
    "trend": [
      { "month": "2026-07", "redemptionCount": 45 },
      { "month": "2026-06", "redemptionCount": 30 }
    ],
    "topBenefits": [
      { "benefitId": "uuid", "benefitName": "Giảm 10%", "benefitType": "discount", "redemptionCount": 50, "percentage": 41.67 },
      { "benefitId": "uuid", "benefitName": "FreeShip", "benefitType": "shipping", "redemptionCount": 30, "percentage": 25.0 }
    ]
  }
}
```

**DTOs**:
```go
type RedemptionTrendDTO struct {
    Month            string `json:"month"`
    RedemptionCount  int    `json:"redemptionCount"`
}

type BenefitRedemptionDTO struct {
    BenefitID        uuid.UUID `json:"benefitId"`
    BenefitName      string    `json:"benefitName"`
    BenefitType      string    `json:"benefitType"`
    RedemptionCount  int       `json:"redemptionCount"`
    Percentage       float64   `json:"percentage"`
}

type BenefitRedemptionAnalyticsRes struct {
    BrandID         uuid.UUID              `json:"brandId"`
    TotalRedemptions int                   `json:"totalRedemptions"`
    Trend           []RedemptionTrendDTO   `json:"trend"`
    TopBenefits     []BenefitRedemptionDTO `json:"topBenefits"`
}
```

---

## 10. GET /operations/catalog-stats

**Response 200**:
```json
{
  "success": true,
  "data": {
    "brandId": "uuid",
    "totalItems": 150,
    "byStatus": [
      { "status": "ACTIVE", "count": 120, "percentage": 80.0 },
      { "status": "DRAFT", "count": 20, "percentage": 13.33 },
      { "status": "INACTIVE", "count": 10, "percentage": 6.67 }
    ],
    "byType": [
      { "itemType": "clothing", "count": 100, "percentage": 66.67 },
      { "itemType": "accessory", "count": 50, "percentage": 33.33 }
    ]
  }
}
```

**DTOs**:
```go
type CatalogStatGroupDTO struct {
    Label      string  `json:"label"`
    Count      int     `json:"count"`
    Percentage float64 `json:"percentage"`
}

type CatalogStatsRes struct {
    BrandID    uuid.UUID            `json:"brandId"`
    TotalItems int                  `json:"totalItems"`
    ByStatus   []CatalogStatGroupDTO `json:"byStatus"`
    ByType     []CatalogStatGroupDTO `json:"byType"`
}
```

---

## 11. GET /customers/spend-segmentation

**Response 200**:
```json
{
  "success": true,
  "data": {
    "brandId": "uuid",
    "totalCustomers": 250,
    "segments": [
      { "segment": "low", "minVnd": 0, "maxVnd": 1000000, "customerCount": 100, "percentage": 40.0 },
      { "segment": "medium", "minVnd": 1000000, "maxVnd": 5000000, "customerCount": 80, "percentage": 32.0 },
      { "segment": "high", "minVnd": 5000000, "maxVnd": 10000000, "customerCount": 40, "percentage": 16.0 },
      { "segment": "premium", "minVnd": 10000000, "maxVnd": null, "customerCount": 30, "percentage": 12.0 }
    ]
  }
}
```

**DTOs**:
```go
type SpendSegmentDTO struct {
    Segment        string   `json:"segment"`
    MinVND         float64  `json:"minVnd"`
    MaxVND         *float64 `json:"maxVnd,omitempty"`
    CustomerCount  int      `json:"customerCount"`
    Percentage     float64  `json:"percentage"`
}

type SpendSegmentationRes struct {
    BrandID       uuid.UUID         `json:"brandId"`
    TotalCustomers int              `json:"totalCustomers"`
    Segments      []SpendSegmentDTO `json:"segments"`
}
```
