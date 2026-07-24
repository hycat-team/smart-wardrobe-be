# Quickstart Validation Guide: Dashboard Expansion

## Prerequisites

- Go 1.22+ installed
- PostgreSQL 16 running
- `make deps` đã chạy (cài dependencies)
- Database đã migrate (`make migration-up`)

## Setup

```bash
# 1. Kiểm tra build
make check

# 2. Chạy test hiện tại
make test
```

## Validation Scenarios

### Scenario 1: Admin Subscription APIs

```bash
# Lấy token admin (cần có user với role_slug = 'admin')
# Gọi API subscription distribution
curl -X GET "http://localhost:8080/api/v1/admin/dashboard/subscriptions/distribution" \
  -H "Authorization: Bearer <admin_token>"

# Expected: 200 với distribution array, totalUsers > 0

# Gọi API revenue breakdown (không filter)
curl -X GET "http://localhost:8080/api/v1/admin/dashboard/subscriptions/revenue-breakdown" \
  -H "Authorization: Bearer <admin_token>"

# Gọi API revenue breakdown (có filter)
curl -X GET "http://localhost:8080/api/v1/admin/dashboard/subscriptions/revenue-breakdown?fromDate=2026-01-01&toDate=2026-07-24" \
  -H "Authorization: Bearer <admin_token>"

# Expected: 200 với revenue breakdown, totalRevenueVnd
```

### Scenario 2: Admin AI APIs

```bash
# Gọi API AI usage by operation
curl -X GET "http://localhost:8080/api/v1/admin/dashboard/ai/usage-by-operation" \
  -H "Authorization: Bearer <admin_token>"

# Expected: 200 với operations array

# Gọi API AI error breakdown
curl -X GET "http://localhost:8080/api/v1/admin/dashboard/ai/error-breakdown" \
  -H "Authorization: Bearer <admin_token>"

# Expected: 200 với error breakdown
```

### Scenario 3: Admin Renewal Stats

```bash
curl -X GET "http://localhost:8080/api/v1/admin/dashboard/subscriptions/renewal-stats" \
  -H "Authorization: Bearer <admin_token>"

# Expected: 200 với renewal stats
```

### Scenario 4: Brand APIs

```bash
# Cần brandId hợp lệ
BRAND_ID="your-brand-uuid"

# Customer acquisition
curl -X GET "http://localhost:8080/api/v1/brand/dashboard/customers/acquisition?brandId=$BRAND_ID" \
  -H "Authorization: Bearer <brand_member_token>"

# Loyalty tier distribution
curl -X GET "http://localhost:8080/api/v1/brand/dashboard/loyalty/tier-distribution?brandId=$BRAND_ID" \
  -H "Authorization: Bearer <brand_member_token>"

# Point expiry forecast
curl -X GET "http://localhost:8080/api/v1/brand/dashboard/loyalty/point-expiry-forecast?brandId=$BRAND_ID" \
  -H "Authorization: Bearer <brand_member_token>"

# Benefit redemption analytics
curl -X GET "http://localhost:8080/api/v1/brand/dashboard/benefits/redemption-analytics?brandId=$BRAND_ID" \
  -H "Authorization: Bearer <brand_member_token>"

# Catalog stats
curl -X GET "http://localhost:8080/api/v1/brand/dashboard/operations/catalog-stats?brandId=$BRAND_ID" \
  -H "Authorization: Bearer <brand_member_token>"

# Customer spend segmentation
curl -X GET "http://localhost:8080/api/v1/brand/dashboard/customers/spend-segmentation?brandId=$BRAND_ID" \
  -H "Authorization: Bearer <brand_member_token>"
```

### Scenario 5: User API

```bash
# Lấy token user
curl -X GET "http://localhost:8080/api/v1/me/dashboard/wardrobe/category-distribution" \
  -H "Authorization: Bearer <user_token>"

# Expected: 200 với categories distribution
```

### Scenario 6: Dynamic Query Metrics

```bash
# Admin: new_users, ai_error_count, ai_error_rate
curl -X POST "http://localhost:8080/api/v1/dashboard/analytics/query" \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "targetDashboard": "admin",
    "metrics": ["new_users", "ai_error_count", "ai_error_rate"],
    "timeframe": { "fromDate": "2026-07-01", "toDate": "2026-07-24" },
    "granularity": "day"
  }'

# Brand: new_customers, customer_claim_rate, brand_item_count
curl -X POST "http://localhost:8080/api/v1/dashboard/analytics/query" \
  -H "Authorization: Bearer <brand_member_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "targetDashboard": "brand",
    "brandId": "'$BRAND_ID'",
    "metrics": ["new_customers", "customer_claim_rate", "brand_item_count"],
    "timeframe": { "fromDate": "2026-07-01", "toDate": "2026-07-24" },
    "granularity": "day"
  }'
```

### Scenario 7: RBAC & Edge Cases

```bash
# User token gọi API Admin → 403
curl -X GET "http://localhost:8080/api/v1/admin/dashboard/subscriptions/distribution" \
  -H "Authorization: Bearer <user_token>"
# Expected: 403 Forbidden

# Brand API không brandId → 400
curl -X GET "http://localhost:8080/api/v1/brand/dashboard/customers/acquisition" \
  -H "Authorization: Bearer <brand_member_token>"
# Expected: 400 Bad Request

# Brand API sai brandId → 403
curl -X GET "http://localhost:8080/api/v1/brand/dashboard/customers/acquisition?brandId=00000000-0000-0000-0000-000000000000" \
  -H "Authorization: Bearer <brand_member_token>"
# Expected: 403 Forbidden
```

## Run Tests

```bash
# Chạy toàn bộ test
make test

# Chạy test dashboard module
go test ./internal/modules/dashboard/...
```
