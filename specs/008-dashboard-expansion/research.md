# Research: Dashboard Expansion

**Phase**: 0 — Research & Design Decisions
**Date**: 2026-07-24

---

## 1. Architecture Decision: Dynamic Query — Snapshot vs Live Query

### Decision
- **Admin metrics** (`new_users`, `ai_error_count`, `ai_error_rate`): Dùng snapshot — fields đã có trong `DashboardAdminDailySnapshot`, chỉ cần thêm mapping.
- **Brand metrics** (`new_customers`, `customer_claim_rate`, `benefit_redemption_count`, `brand_item_count`, `brand_item_count_by_type`): **Live query** từ bảng nguồn — vì:
  - `brand_item_count_by_type` cần GROUP BY `item_type`, không phù hợp snapshot daily aggregate
  - `customer_claim_rate` là tỷ lệ real-time, snapshot chỉ lưu số tuyệt đối
  - Các bảng nguồn có index tốt, query nhanh

### Rationale
- Giữ nguyên pattern snapshot cho admin metrics (0 effort vì field đã có)
- Brand metrics cần linh hoạt (group-by, ratio) → live query tối ưu hơn snapshot

---

## 2. Repository Organization

### Decision
Mỗi nhóm API có repository method riêng trong file tương ứng:
- `dashboard_admin_repo.go`: admin dashboard queries
- `dashboard_brand_repo.go`: brand dashboard queries
- `dashboard_user_repo.go`: user dashboard queries

### Rationale
- Nhất quán với code hiện tại (mỗi nhóm đã có file riêng)
- Tránh một file repository quá lớn

---

## 3. Query Pattern

### Decision
Sử dụng GORM `Model().Select().Group().Find()` với raw SQL fragments cho các query thống kê, không dùng subquery.

### Rationale
- Các query đơn giản (count + group by), không cần raw SQL phức tạp
- GORM Select/Group đáp ứng được
- Dễ maintain

---

## 4. Error Handling

### Decision
- Không có dữ liệu: trả về empty array / zero values, không 404
- Brand ID không hợp lệ: 400 Bad Request
- Date format sai: 400 Bad Request
- RBAC fail: 403 Forbidden

### Rationale
Nhất quán với spec FR-016 và các API dashboard hiện tại.

---

## 5. Phân khúc chi tiêu (Spend Segmentation)

### Decision
4 mốc cố định:
| Segment | Khoảng chi tiêu |
|---------|-----------------|
| low | 0 - 1,000,000 VND |
| medium | 1,000,000 - 5,000,000 VND |
| high | 5,000,000 - 10,000,000 VND |
| premium | > 10,000,000 VND |

Dùng SQL CASE WHEN để tính segment trong query.

---

## 6. Token Cost Conversion

### Decision
`ai_usage_events.actual_cost_micro_vnd` lưu ở đơn vị micro VND (1/1,000,000 VND). Khi trả về API, convert sang VND:

```
cost_vnd = actual_cost_micro_vnd / 1,000,000
```

Pattern này đã dùng trong `admin_dashboard_uc_impl.go` (hằng số `microVNDDivisor`).

---

## 7. Brand API — RBAC

### Decision
Kế thừa cơ chế hiện tại của Brand Dashboard:
- Router group dùng `authMiddleware.Handle()` + kiểm tra brand membership
- Use case nhận `brandID` từ query param, repository tự động scope theo brand_id

---

## 8. Dependencies

### Existing (không cần thêm mới)
- `github.com/gin-gonic/gin` — HTTP framework
- `gorm.io/gorm` — ORM
- `github.com/google/uuid` — UUID
- `smart-wardrobe-be/internal/shared/application/constants/apperror` — error types
- `smart-wardrobe-be/internal/shared/presentation` — response helpers
- `smart-wardrobe-be/internal/pkg/utils/validation` — binding + validation
