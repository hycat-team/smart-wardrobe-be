# Quickstart & Verification Guide: Dashboard & Analytics System

**Feature**: `006-dashboard-analytics`  
**Date**: 2026-07-23  

---

## 1. Prerequisites

- Go v1.25+ installed
- PostgreSQL and Redis servers running locally (hoặc qua `make dev` / Docker Compose)
- DB Goose Migrations up-to-date (`make migration-up`)

---

## 2. Running Goose Migrations

Tạo các bảng Read Model Snapshots cho Dashboard:

```bash
make migration-up
```

Kiểm tra DB đã có 3 bảng:
- `dashboard_admin_daily_snapshots`
- `dashboard_brand_daily_snapshots`
- `dashboard_user_wardrobe_summaries`

---

## 3. Wire Dependency Injection Generation

Sau khi bổ sung `internal/modules/dashboard/provider.go`, chạy Wire injector:

```bash
make wire
```

---

## 4. Manual / Automated Verification Scenarios

### Test 1: Admin Dynamic Query API
```bash
curl -X POST http://localhost:8080/api/v1/dashboard/analytics/query \
  -H "Authorization: Bearer <ADMIN_JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "target_dashboard": "ADMIN",
    "metrics": ["ai_requests_count", "net_ai_margin_vnd"],
    "timeframe": { "from_date": "2026-07-01", "to_date": "2026-07-23" },
    "granularity": "day"
  }'
```
- **Expected**: Phản hồi HTTP 200 OK dưới 150ms với chuỗi dữ liệu Time-Series theo từng ngày.

---

### Test 2: Brand Staff Column-Level RBAC Redaction Test
```bash
# Call financial endpoint as BRAND STAFF
curl -X GET http://localhost:8080/api/v1/brand/dashboard/financials/points-liability \
  -H "Authorization: Bearer <BRAND_STAFF_JWT_TOKEN>"
```
- **Expected**: Phản hồi HTTP 403 Forbidden (chỉ cho phép `brandmemberrole.Owner`).

---

### Test 3: User Cost Per Wear (CPW) Summary Test
```bash
curl -X GET http://localhost:8080/api/v1/user/dashboard/wardrobe-insights \
  -H "Authorization: Bearer <USER_JWT_TOKEN>"
```
- **Expected**: HTTP 200 OK trả về tổng giá trị tủ đồ, chỉ số CPW và danh sách trang phục chưa mặc > 30 ngày.
