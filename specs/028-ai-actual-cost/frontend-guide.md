# Frontend Guide: Chi phí AI thật phục vụ báo cáo lợi nhuận (028)

> Đối chiếu lần cuối với mã nguồn sau implement. Mọi con số đếm dưới đây đều
> từ đối chiếu trực tiếp (xem mục Đối chiếu).

## Endpoint thay đổi: 1 (changed, additive — tương thích ngược)

### `GET /api/v1/admin/dashboard/drilldown/ai-events` — CHANGED (thêm field)

- Handler: `services/commerce/internal/modules/dashboard/presentation/handler/analytics_handler.go:76` (`GetAIEventsDrillDown`), route đăng ký tại `services/commerce/internal/modules/dashboard/presentation/router/router.go:87`, yêu cầu quyền admin.
- Query: `status` (lọc theo trạng thái, để trống = tất cả) + phân trang chuẩn (`page`, `limit`).
- Response `data.items[]` (`DrillDownAIEventDTO`, `dynamic_query_dto.go:54-62`) — 7 field JSON, đã đếm đủ trong code:

| Field JSON | Trước | Sau |
|---|---|---|
| `requestId` | có | giữ nguyên |
| `provider` | có | giữ nguyên |
| `model` | có | giữ nguyên |
| `statusCode` | có | giữ nguyên |
| `errorMessage` | có | giữ nguyên |
| `errorCode` | **không có** | **MỚI — string, rỗng khi không có** |
| `createdAt` | có | giữ nguyên |

- Giá trị `errorCode` mới do tính năng này sinh (chỉ khi `status=confirmed`):
  `pricing_config_missing` (thiếu giá model — kể cả provider/model rỗng),
  `usage_unreported` (có giá nhưng không có số token — hiện tại: embedding).
  Các mã cũ (lỗi provider ở `status=released`) giữ nguyên.
- Giá trị `status` hợp lệ (6 giá trị, đối chiếu 2 chiều với
  `aiusageeventstatus.go:6-12`): `reserved`, `in_flight`, `confirmed`,
  `released`, `unknown_usage`, `expired_unverified`.
- HTTP status handler có thể trả: 200 thành công; 400 khi query không hợp lệ;
  401/403 khi thiếu/không đủ quyền (chuẩn middleware `WrapHandler` + admin guard).

## Endpoint KHÔNG đổi shape (giá trị trả về trở nên đúng)

| Endpoint | Trạng thái | Ghi chú cho frontend |
|---|---|---|
| `GET /api/v1/admin/dashboard/ai/usage-by-operation` | unchanged | `totalCostVnd`/token giờ là số thật: vision > 0, embedding 0 kèm mã. Không cần sửa mapping, nhưng tooltip "0đ" cũ nên diễn giải lại khi có `errorCode`. |
| `GET /api/v1/admin/dashboard/kpi-cards`, `overview` | unchanged | `netAiMarginVnd` tự đúng. Không đổi field nào. |
| gRPC `GetAIErrorBreakdown` | unchanged | Vẫn chỉ đếm `status=released` — hai mã mới không lọt vào đây (đúng thiết kế). |

## Không có endpoint/event nào mới hay bị xoá.

## Đối chiếu (đếm số)

- DTO drill-down: 7/7 field JSON trong code đều có trong bảng trên.
- Enum status: 6/6 giá trị Go ↔ guide khớp cả hai chiều.
- Route: 1 route GET liệt kê đúng dạng OpenAPI (`/api/v1/admin/dashboard/drilldown/ai-events`, khớp `@Router` trong handler).
- HTTP status: 200/400/401/403 — đúng các nhánh handler + middleware có thể sinh.

## Sync cuối (Loop C, 11/10/2026)

- Không có code production nào đổi sau khi guide được viết (Phase 4–6 chỉ tick tasks + docs) → đối chiếu trên vẫn đúng, không sửa gì thêm.
