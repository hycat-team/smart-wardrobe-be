# REST API Schema Contracts: Dashboard & Analytics System

**Feature**: `006-dashboard-analytics`  
**Date**: 2026-07-23  

---

## 1. Dynamic Query Endpoint

### `POST /api/v1/dashboard/analytics/query`

Cho phép Frontend gửi truy vấn tổng hợp linh hoạt động cho bất kỳ Widget hoặc biểu đồ nào.

#### Request Body
```json
{
  "target_dashboard": "ADMIN", // "ADMIN" | "BRAND" | "USER"
  "brand_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6", // Optional, required for BRAND if not in JWT
  "metrics": ["ai_requests_count", "net_ai_margin_vnd", "subscription_revenue_vnd"],
  "timeframe": {
    "from_date": "2026-07-01",
    "to_date": "2026-07-23"
  },
  "granularity": "day", // "hour" | "day" | "week" | "month"
  "filters": {
    "provider": "google",
    "status": "SUCCESS"
  },
  "group_by": ["provider"],
  "compare_with_previous": true
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "data": {
    "target_dashboard": "ADMIN",
    "granularity": "day",
    "series": [
      {
        "timestamp": "2026-07-22T00:00:00Z",
        "metrics": {
          "ai_requests_count": 1250,
          "subscription_revenue_vnd": 1500000.00,
          "net_ai_margin_vnd": 1120000.00
        }
      },
      {
        "timestamp": "2026-07-23T00:00:00Z",
        "metrics": {
          "ai_requests_count": 1420,
          "subscription_revenue_vnd": 1800000.00,
          "net_ai_margin_vnd": 1350000.00
        }
      }
    ],
    "comparison": {
      "previous_period_from": "2026-07-01",
      "previous_period_to": "2026-07-21",
      "metric_delta_percentages": {
        "ai_requests_count": 13.6,
        "net_ai_margin_vnd": 20.5
      }
    }
  }
}
```

---

## 2. Granular Widget Endpoints

### 2.1. System Admin KPIs
- **`GET /api/v1/admin/dashboard/kpi-cards`**
  - **Auth**: Bearer JWT (`roleslug.Admin`)
  - **Response**: Thẻ KPI tổng quan: Total users, DAU, Active Brands, Total Revenue, Net AI Margin (VND).

- **`GET /api/v1/admin/dashboard/ai-margin`**
  - **Auth**: Bearer JWT (`roleslug.Admin`)
  - **Response**: Chi tiết Doanh thu Subscription vs Chi phí AI Token từ `ai_usage_period_ledgers`.

---

### 2.2. B2B Brand Manager Portal
- **`GET /api/v1/brand/dashboard/kpi-cards`**
  - **Auth**: Bearer JWT (`brandmemberrole.Owner`, `brandmemberrole.Staff`)
  - **Response**: Số hội viên, điểm phát hành, điểm quy đổi, số voucher active.

- **`GET /api/v1/brand/dashboard/financials/points-liability`**
  - **Auth**: Bearer JWT (`brandmemberrole.Owner` ONLY)
  - **Response**: Dự báo điểm hết hạn trong 30 ngày & giá trị tiền mặt quy đổi tương ứng. *(Tài khoản Staff gọi API này trả về 403 Forbidden)*.

- **`GET /api/v1/brand/dashboard/operations/digital-sample-lab/{itemId}`**
  - **Auth**: Bearer JWT (`brandmemberrole.Owner`, `brandmemberrole.Staff`)
  - **Response**: Số lượt vote, rating trung bình, feedback sentiment, và danh sách trang phục thường phối kèm.

---

### 2.3. End User Personal Overview (B2C)
- **`GET /api/v1/user/dashboard/wardrobe-insights`**
  - **Auth**: Bearer JWT (`roleslug.User`)
  - **Response**:
```json
{
  "success": true,
  "data": {
    "total_items": 42,
    "total_wardrobe_value_vnd": 12500000.00,
    "average_cost_per_wear_vnd": 45000.00,
    "ai_quota_remaining": 8,
    "underutilized_items": [
      {
        "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "name": "Áo khoác Blazer đen",
        "purchase_price": 1200000.00,
        "last_used_at": "2026-05-10T10:00:00Z",
        "days_unused": 74
      }
    ]
  }
}
```

---

## 3. Drill-Down Endpoints (Truy vết Chi tiết)

- **`GET /api/v1/admin/dashboard/drilldown/ai-events?status=FAILED&page=1&limit=20`**
  - Trả về danh sách chi tiết các request AI bị lỗi kèm `request_id`, `error_code`, `provider`, `model` để Admin điều tra.

- **`GET /api/v1/brand/dashboard/drilldown/sample-lab-feedbacks?item_id=...&page=1&limit=20`**
  - Trả về danh sách chi tiết các đánh giá và ý kiến phản hồi của người dùng cho mẫu thử 3D.
