# Admin Dashboard API Contracts

**Base path**: `/api/v1/admin/dashboard`
**Auth**: JWT + `roleslug.Admin`

---

## 1. GET /subscriptions/distribution

**Description**: Phân bổ user theo gói subscription

**Response 200**:
```json
{
  "success": true,
  "data": {
    "distribution": [
      { "planCode": "FREE", "userCount": 800, "percentage": 53.33 },
      { "planCode": "PREMIUM_MONTHLY", "userCount": 500, "percentage": 33.33 }
    ],
    "totalUsers": 1500
  }
}
```

**DTOs**:
```go
type SubscriptionDistributionDTO struct {
    PlanCode    string  `json:"planCode"`
    UserCount   int     `json:"userCount"`
    Percentage  float64 `json:"percentage"`
}

type SubscriptionDistributionRes struct {
    Distribution []SubscriptionDistributionDTO `json:"distribution"`
    TotalUsers   int                           `json:"totalUsers"`
}
```

---

## 2. GET /subscriptions/revenue-breakdown

**Query params**: `fromDate` (YYYY-MM-DD, optional), `toDate` (YYYY-MM-DD, optional)

**Response 200**:
```json
{
  "success": true,
  "data": {
    "fromDate": "2026-01-01",
    "toDate": "2026-07-24",
    "breakdown": [
      { "planCode": "PREMIUM_MONTHLY", "month": "2026-01", "transactionCount": 45, "revenueVnd": 22500000 },
      { "planCode": "PREMIUM_YEARLY", "month": "2026-06", "transactionCount": 10, "revenueVnd": 50000000 }
    ],
    "totalRevenueVnd": 125000000
  }
}
```

**DTOs**:
```go
type RevenueBreakdownDTO struct {
    PlanCode         string  `json:"planCode"`
    Month            string  `json:"month"`
    TransactionCount int     `json:"transactionCount"`
    RevenueVND       float64 `json:"revenueVnd"`
}

type RevenueBreakdownRes struct {
    FromDate        string               `json:"fromDate"`
    ToDate          string               `json:"toDate"`
    Breakdown       []RevenueBreakdownDTO `json:"breakdown"`
    TotalRevenueVND float64              `json:"totalRevenueVnd"`
}
```

---

## 3. GET /ai/usage-by-operation

**Query params**: `fromDate` (YYYY-MM-DD, optional), `toDate` (YYYY-MM-DD, optional)

**Response 200**:
```json
{
  "success": true,
  "data": {
    "fromDate": "2026-07-01",
    "toDate": "2026-07-24",
    "operations": [
      {
        "operation": "outfit-recommend",
        "logicalRoute": "normal",
        "requestCount": 1200,
        "totalCostVnd": 2500000,
        "totalPromptTokens": 150000,
        "totalOutputTokens": 80000
      }
    ]
  }
}
```

**DTOs**:
```go
type AIUsageByOperationDTO struct {
    Operation         string  `json:"operation"`
    LogicalRoute      string  `json:"logicalRoute"`
    RequestCount      int64   `json:"requestCount"`
    TotalCostVND      float64 `json:"totalCostVnd"`
    TotalPromptTokens int64   `json:"totalPromptTokens"`
    TotalOutputTokens int64   `json:"totalOutputTokens"`
}

type AIUsageByOperationRes struct {
    FromDate   string                   `json:"fromDate"`
    ToDate     string                   `json:"toDate"`
    Operations []AIUsageByOperationDTO  `json:"operations"`
}
```

---

## 4. GET /ai/error-breakdown

**Response 200**:
```json
{
  "success": true,
  "data": {
    "totalErrors": 45,
    "totalRequests": 3200,
    "errorRate": 1.41,
    "breakdown": [
      { "errorCode": "rate_limit_exceeded", "provider": "openai", "model": "gpt-4", "errorCount": 20 },
      { "errorCode": "timeout", "provider": "anthropic", "model": "claude-3", "errorCount": 15 }
    ]
  }
}
```

**DTOs**:
```go
type AIErrorBreakdownDTO struct {
    ErrorCode  string `json:"errorCode"`
    Provider   string `json:"provider"`
    Model      string `json:"model"`
    ErrorCount int64  `json:"errorCount"`
}

type AIErrorBreakdownRes struct {
    TotalErrors   int64                `json:"totalErrors"`
    TotalRequests int64                `json:"totalRequests"`
    ErrorRate     float64              `json:"errorRate"`
    Breakdown     []AIErrorBreakdownDTO `json:"breakdown"`
}
```

---

## 5. GET /subscriptions/renewal-stats

**Response 200**:
```json
{
  "success": true,
  "data": {
    "totalAttempts": 120,
    "successCount": 98,
    "failedCount": 22,
    "successRate": 81.67,
    "avgRetryCount": 1.3
  }
}
```

**DTOs**:
```go
type RenewalStatsRes struct {
    TotalAttempts int     `json:"totalAttempts"`
    SuccessCount  int     `json:"successCount"`
    FailedCount   int     `json:"failedCount"`
    SuccessRate   float64 `json:"successRate"`
    AvgRetryCount float64 `json:"avgRetryCount"`
}
```
