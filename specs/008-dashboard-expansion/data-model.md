# Data Model: Dashboard Expansion

**Phase**: 1 — Data Model & Contracts
**Date**: 2026-07-24

---

## Admin APIs — Data Sources

### 1. Subscription Distribution

**Source table**: `user_subscriptions`

| Column | Type | Description |
|--------|------|-------------|
| `current_plan_code` | VARCHAR(100) | Mã gói subscription (FREE, PREMIUM_MONTHLY, PREMIUM_YEARLY, ...) |
| `current_tier_rank` | INT | Thứ hạng tier |
| `user_id` | UUID (PK) | User |

**Query**: `SELECT current_plan_code, COUNT(*) as user_count FROM user_subscriptions GROUP BY current_plan_code ORDER BY user_count DESC`

**Response**:
```json
{
  "distribution": [
    { "planCode": "FREE", "userCount": 800, "percentage": 53.33 },
    { "planCode": "PREMIUM_MONTHLY", "userCount": 500, "percentage": 33.33 },
    { "planCode": "PREMIUM_YEARLY", "userCount": 200, "percentage": 13.33 }
  ],
  "totalUsers": 1500
}
```

---

### 2. Revenue Breakdown

**Source table**: `deposit_transactions`

| Column | Type | Description |
|--------|------|-------------|
| `plan_code` | VARCHAR(100) | Mã gói |
| `amount` | NUMERIC(12,2) | Số tiền |
| `status` | INT2 | 1 = Success |
| `transaction_type` | VARCHAR(50) | 'DIRECT_PURCHASE' |
| `created_at` | TIMESTAMPTZ | Thời gian |

**Query**: 
```sql
SELECT plan_code, 
       DATE_TRUNC('month', created_at) as month,
       COUNT(*) as transaction_count,
       SUM(amount) as revenue
FROM deposit_transactions
WHERE status = 1 
  AND transaction_type = 'DIRECT_PURCHASE'
  AND created_at BETWEEN @fromDate AND @toDate
GROUP BY plan_code, DATE_TRUNC('month', created_at)
ORDER BY month, plan_code
```

---

### 3. AI Usage by Operation

**Source table**: `ai_usage_events`

| Column | Type | Description |
|--------|------|-------------|
| `operation` | VARCHAR(30) | Tác vụ AI (outfit-recommend, chat, styling, ...) |
| `logical_route` | VARCHAR(50) | Route xử lý (normal, reduced, free) |
| `actual_cost_micro_vnd` | INT8 | Chi phí thực tế (micro VND) |
| `prompt_tokens` | INT8 | Token đầu vào |
| `output_tokens` | INT8 | Token đầu ra |
| `created_at` | TIMESTAMPTZ | Thời gian |

---

### 4. AI Error Breakdown

**Source table**: `ai_usage_events`

| Column | Type | Description |
|--------|------|-------------|
| `error_code` | VARCHAR(100) | Mã lỗi |
| `provider` | VARCHAR(50) | Nhà cung cấp AI |
| `model` | VARCHAR(150) | Model AI |
| `status` | VARCHAR(30) | Trạng thái (FAILED, RELEASED, EXPIRED) |

---

### 5. Renewal Stats

**Source table**: `subscription_renewal_attempts`

| Column | Type | Description |
|--------|------|-------------|
| `status` | VARCHAR(50) | SUCCESS, FAILED, PENDING |
| `attempt_count` | INT4 | Số lần thử |
| `last_error_code` | VARCHAR(100) | Mã lỗi cuối |
| `completed_at` | TIMESTAMPTZ | Thời gian hoàn thành |

---

## Brand APIs — Data Sources

### 6. Customer Acquisition

**Source table**: `brand_customers`

| Column | Type | Description |
|--------|------|-------------|
| `brand_id` | UUID | Brand |
| `joined_source` | VARCHAR(50) | Kênh (walk_in, referral, online, QR, ...) |
| `status` | VARCHAR(50) | ACTIVE, INACTIVE |

---

### 7. Loyalty Tier Distribution

**Source tables**: `loyalty_accounts` + `loyalty_tiers`

| Column | Type | Source |
|--------|------|--------|
| `current_tier_id` | UUID | loyalty_accounts |
| `brand_id` | UUID | loyalty_accounts |
| `name` | VARCHAR(255) | loyalty_tiers |
| `rank` | INT4 | loyalty_tiers |

---

### 8. Point Expiry Forecast

**Source table**: `loyalty_point_lots`

| Column | Type | Description |
|--------|------|-------------|
| `brand_id` | UUID | Brand |
| `expires_at` | TIMESTAMPTZ | Ngày hết hạn |
| `remaining_points` | INT4 | Điểm còn lại |
| `status` | VARCHAR(50) | ACTIVE |
| `user_id` | UUID | User |

---

### 9. Benefit Redemption Analytics

**Source tables**: `benefit_redemptions` + `brand_benefits`

| Column | Type | Source |
|--------|------|--------|
| `benefit_id` | UUID | benefit_redemptions |
| `brand_id` | UUID | benefit_redemptions |
| `name` | VARCHAR(255) | brand_benefits |
| `benefit_type` | VARCHAR(50) | brand_benefits |
| `redeemed_at` | TIMESTAMPTZ | benefit_redemptions |
| `status` | VARCHAR(50) | benefit_redemptions |

---

### 10. Catalog Stats

**Source table**: `brand_items`

| Column | Type | Description |
|--------|------|-------------|
| `brand_id` | UUID | Brand |
| `status` | VARCHAR(50) | ACTIVE, INACTIVE, DRAFT, ... |
| `item_type` | VARCHAR(50) | clothing, accessory, ... |
| `price` | NUMERIC(12,2) | Giá |

---

### 11. Customer Spend Segmentation

**Source table**: `loyalty_accounts`

| Column | Type | Description |
|--------|------|-------------|
| `brand_id` | UUID | Brand |
| `total_spend` | NUMERIC(12,2) | Tổng chi tiêu |

**Segments**:
| Segment | Range |
|---------|-------|
| low | `total_spend BETWEEN 0 AND 1000000` |
| medium | `total_spend BETWEEN 1000000 AND 5000000` |
| high | `total_spend BETWEEN 5000000 AND 10000000` |
| premium | `total_spend > 10000000` |

---

## User API — Data Source

### 12. Wardrobe Category Distribution

**Source tables**: `wardrobe_items` + `fashion_items` + `categories`

```sql
SELECT c.id as category_id,
       c.name as category_name,
       COUNT(*) as item_count
FROM wardrobe_items wi
JOIN fashion_items fi ON fi.id = wi.fashion_item_id
JOIN categories c ON c.id = fi.category_id
WHERE wi.user_id = @userID
  AND wi.is_deleted = false
GROUP BY c.id, c.name
ORDER BY item_count DESC
```

---

## Dynamic Query Metrics

### Admin — từ DashboardAdminDailySnapshot (đã có field)

| Metric Key | Entity Field | Type | Công thức |
|------------|-------------|------|-----------|
| `new_users` | `NewUsers` | int | `s.NewUsers` |
| `ai_error_count` | `AIErrorCount` | int64 | `s.AIErrorCount` |
| `ai_error_rate` | computed | float64 | `s.AIErrorCount / s.AIRequestsCount * 100` |

Cần thêm vào `metrickey/metric_key.go`:
```go
NewUsers       MetricKey = "new_users"
AIErrorCount   MetricKey = "ai_error_count"
AIErrorRate    MetricKey = "ai_error_rate"
```

### Brand — Live query từ bảng nguồn

| Metric Key | Source Table | Query Logic |
|------------|-------------|-------------|
| `new_customers` | `brand_customers` | `COUNT(*) WHERE brand_id=? AND created_at BETWEEN ? AND ?` |
| `customer_claim_rate` | `brand_customers` | `SUM(CASE WHEN claimed_at IS NOT NULL THEN 1 ELSE 0 END) / COUNT(*)` |
| `benefit_redemption_count` | `benefit_redemptions` | `COUNT(*) WHERE brand_id=? AND redeemed_at BETWEEN ? AND ?` |
| `brand_item_count` | `brand_items` | `COUNT(*) WHERE brand_id=?` |
| `brand_item_count_by_type` | `brand_items` | `SELECT item_type, COUNT(*) FROM brand_items WHERE brand_id=? GROUP BY item_type` |

Cần thêm vào `metrickey/metric_key.go`:
```go
NewCustomers           MetricKey = "new_customers"
CustomerClaimRate      MetricKey = "customer_claim_rate"
BenefitRedemptionCount MetricKey = "benefit_redemption_count"
BrandItemCount         MetricKey = "brand_item_count"
BrandItemCountByType   MetricKey = "brand_item_count_by_type"
```
