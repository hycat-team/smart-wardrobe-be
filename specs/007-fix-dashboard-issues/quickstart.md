# Quickstart Validation — Sửa Lỗi Dashboard

## Môi trường

- Backend đang chạy (`make dev`)
- DB đã có seed data từ Feature 006
- JWT token cho từng role (Admin, Brand Owner, Staff, User)

## Các kịch bản kiểm thử

### TC01: Granularity month trả về 12 points

```bash
curl -s -X POST http://localhost:8080/api/v1/dashboard/analytics/query \
  -H "Authorization: Bearer $TOKEN_ADMIN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetDashboard": "admin",
    "metrics": ["total_users", "subscription_revenue_vnd"],
    "timeframe": { "fromDate": "2025-07-24", "toDate": "2026-07-24" },
    "granularity": "month"
  }' | jq '.data.series | length'
```

**Kỳ vọng:** `12` (12 data points, mỗi point đại diện 1 tháng)

---

### TC02: Granularity week trả về đúng số tuần

```bash
curl -s -X POST http://localhost:8080/api/v1/dashboard/analytics/query \
  -H "Authorization: Bearer $TOKEN_ADMIN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetDashboard": "admin",
    "metrics": ["total_users"],
    "timeframe": { "fromDate": "2026-06-26", "toDate": "2026-07-24" },
    "granularity": "week"
  }' | jq '.data.series | length'
```

**Kỳ vọng:** `4` (4 weeks)

---

### TC03: Fallback admin KPI khi không có snapshot

> Cần xoá hoặc truncate bảng `dashboard_admin_daily_snapshots` trước.

```bash
curl -s http://localhost:8080/api/v1/admin/dashboard/kpi-cards \
  -H "Authorization: Bearer $TOKEN_ADMIN" | jq '.data'
```

**Kỳ vọng:** `subscriptionRevenueVnd` > 0 và `payOSSuccessCount` > 0 (từ live data)

---

### TC04: Digital sample lab fallback (không item snapshot)

```bash
curl -s "http://localhost:8080/api/v1/brand/dashboard/operations/digital-sample-lab/b1000001-0000-0000-0000-000000000001?brandId=e7b10001-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer $TOKEN_OWNER" | jq '.data'
```

**Kỳ vọng:** `feedbackSentiment` và `topPairedCategories` có giá trị (không null/empty)

---

### TC05: Dynamic query với metric mới `total_brands`

```bash
curl -s -X POST http://localhost:8080/api/v1/dashboard/analytics/query \
  -H "Authorization: Bearer $TOKEN_ADMIN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetDashboard": "admin",
    "metrics": ["total_brands", "pending_review_brands"],
    "timeframe": { "fromDate": "2026-07-17", "toDate": "2026-07-24" },
    "granularity": "day"
  }' | jq '.data.series[0].metrics'
```

**Kỳ vọng:** `total_brands` và `pending_review_brands` có giá trị > 0

---

### TC06: Dynamic query với brand metric mới

```bash
curl -s -X POST http://localhost:8080/api/v1/dashboard/analytics/query \
  -H "Authorization: Bearer $TOKEN_OWNER" \
  -H "Content-Type: application/json" \
  -d '{
    "targetDashboard": "brand",
    "brandId": "e7b10001-0000-0000-0000-000000000001",
    "metrics": ["vouchers_redeemed", "digital_sample_votes_count", "digital_sample_avg_rating"],
    "timeframe": { "fromDate": "2026-07-17", "toDate": "2026-07-24" },
    "granularity": "day"
  }' | jq '.data.series[0].metrics'
```

**Kỳ vọng:** Các metric mới được resolve, không trả về 0

---

### TC07: Dynamic query không còn Filters/GroupBy

```bash
curl -s -X POST http://localhost:8080/api/v1/dashboard/analytics/query \
  -H "Authorization: Bearer $TOKEN_ADMIN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetDashboard": "admin",
    "metrics": ["total_users"],
    "timeframe": { "fromDate": "2026-07-17", "toDate": "2026-07-24" },
    "granularity": "day",
    "filters": {"status": "active"},
    "groupBy": ["region"]
  }' 2>&1 | head -5
```

**Kỳ vọng:** Request không bị parse lỗi nhưng 2 field bị ignore (hoặc lỗi 400 nếu DTO reject strict)

---

### TC08: AdminOverview trả 500 khi sub-component lỗi

> Cần mock PayOS service offline.

```bash
curl -s -w "\nHTTP_STATUS: %{http_code}\n" http://localhost:8080/api/v1/admin/dashboard/overview \
  -H "Authorization: Bearer $TOKEN_ADMIN"
```

**Kỳ vọng:** HTTP 500, không phải 200 với `payosReconcile=null`

---

### TC09: User wardrobe insights không còn averageItemValueVnd

```bash
curl -s http://localhost:8080/api/v1/me/dashboard/wardrobe-insights \
  -H "Authorization: Bearer $TOKEN_USER" | jq '.data | keys'
```

**Kỳ vọng:** `averageItemValueVnd` không còn trong keys

---

### TC10: Zero regression — unit tests

```bash
make test
```

**Kỳ vọng:** Tất cả test pass (34/34 packages, 0 FAIL)
