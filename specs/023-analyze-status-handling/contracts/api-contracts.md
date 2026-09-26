# API Contracts: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Date**: 2026-09-26

Quy ước chung: auth Bearer; JSON camelCase; response bọc `APIResponse { success, message, data }` (trừ SSE). Đây là hợp đồng hướng frontend — bản đầy đủ dành cho FE nằm ở [`../frontend-guide.md`](../frontend-guide.md).

---

## 1. API MỚI

### 1.1 Xác nhận món "cần rà soát" (Wardrobe, User)

| Thuộc tính | Giá trị |
|-----------|---------|
| Method/Path | `POST /api/v1/wardrobe-items/{id}/confirm-review` |
| Role | `user` |
| Mô tả | Chuyển món từ `NeedsReview` → `InWardrobe` (sử dụng được) |

**Request**: không body.

**Response 200**
```json
{
  "success": true,
  "message": "Xác nhận trang phục thành công",
  "data": {
    "id": "uuid",
    "price": null,
    "status": 0,
    "isLocked": false,
    "fashionItem": { "id": "uuid", "category": { "id": "uuid", "name": "Áo", "slug": "ao" }, "imageUrl": "...", "color": "...", "colorHex": "#000000", "style": "..." },
    "taskId": null
  }
}
```

**Lỗi**:
| HTTP | Điều kiện |
|------|-----------|
| 403 | Không sở hữu món / món không ở trạng thái `NeedsReview` |
| 422/400 | Thiếu danh mục hợp lệ (chưa chọn hoặc category không thể phối đồ) |
| 404 | Không tìm thấy món |

### 1.2 Thử lại phân tích cho sản phẩm nhãn hàng (Brand Portal, Owner/Staff)

| Thuộc tính | Giá trị |
|-----------|---------|
| Method/Path | `POST /api/v1/brand-portal/brands/{brandId}/items/{itemId}/retry-analysis` |
| Role | `user`/`admin` + brand membership (Owner/Staff) |
| Mô tả | Đưa sản phẩm `failed` (hoặc `draft` cần rà soát) lại hàng đợi phân tích |

**Response 200**
```json
{
  "success": true,
  "message": "Yêu cầu phân tích lại sản phẩm đã được gửi thành công",
  "data": {
    "id": "uuid",
    "brandId": "uuid",
    "status": "draft",
    "processingErrorReason": null,
    "reviewReason": null,
    "taskId": "uuid",
    "fashionItem": { "...": "..." }
  }
}
```

**Lỗi**:
| HTTP | Điều kiện |
|------|-----------|
| 403 | Không thuộc brand / item không ở trạng thái cho phép (đang xử lý hoặc đã sử dụng) |
| 404 | Không tìm thấy brand/item |

---

## 2. API THAY ĐỔI HÀNH VI (trước → sau)

### 2.1 `POST /api/v1/wardrobe-items/{id}/retry-analysis`

| | Trước | Sau |
|---|-------|-----|
| Hành vi | Chỉ chuyển trạng thái + phát job | Thêm guard trạng thái + chống phiên trùng (claim nguyên tử); **trả thêm `taskId`** để subscribe SSE; giữ danh mục người dùng |
| Response | `WardrobeItemRes` (không có taskId) | `WardrobeItemRes` + `taskId` (mới, optional) |
| Lỗi mới | — | `retry khi trạng thái không hợp lệ` (đang xử lý / đã sử dụng) |

**Trước**
```json
{ "id": "uuid", "status": 3, "isLocked": false, "fashionItem": { } }
```
**Sau**
```json
{ "id": "uuid", "status": 3, "isLocked": false, "fashionItem": { }, "taskId": "uuid" }
```

### 2.2 `PATCH /api/v1/brand-portal/brands/{brandId}/items/{itemId}/status` (UpdateBrandItemStatus)

| | Trước | Sau |
|---|-------|-----|
| Status hợp lệ | `draft`, `active`, `archived` | `draft`, `active`, `archived` (KHÔNG cho đặt `failed`) |
| Chuyển từ `failed` | Không xử lý đặc biệt | **Từ chối** mọi thay đổi rời `failed` |
| Lỗi | `ErrBenefitInvalidStatus` | Thêm `ErrBrandItemStatusLocked` (khi item đang `failed`) |

### 2.3 `PUT /api/v1/brand-portal/brands/{brandId}/items/{itemId}` (UpdateBrandItem)

| | Trước | Sau |
|---|-------|-----|
| Validate status | Không validate (bug) | Validate status thuộc tập hợp lệ + chặn khi item đang `failed` |
| Trường `status` trong response | `draft|active|archived` | Có thể là `failed` (do hệ thống set) |

### 2.4 `GET /api/v1/brand-portal/brands/{brandId}/items` và `GET .../items/{itemId}`

| | Trước | Sau |
|---|-------|-----|
| `status` | `draft|active|archived` | Thêm `failed` |
| Lý do | Không có | Thêm `processingErrorReason`, `reviewReason` (optional) |

### 2.5 `GET /api/v1/wardrobe-items/tasks/{taskId}/sse` và brand SSE

| | Trước | Sau |
|---|-------|-----|
| `status` trong event | `completed|failed|needs_review` | Thêm `processing` |
| Replay khi reconnect | Không | Có (buffer N event/phiên) |
| Kênh truyền | In-process | Redis Pub/Sub (không đổi hợp đồng phía client) |

### 2.6 `POST /api/v1/wardrobe-items/batch-upload` (202)

| | Trước | Sau |
|---|-------|-----|
| Hành vi | Trả task + danh sách item `processing` | Không đổi hợp đồng; client dùng taskId subscribe SSE như cũ |

> Các API khác giữ nguyên. FE không cần refetch khi nhận `completed`/`needs_review` vì `data` đã đầy đủ.

---

## 3. Ma trận ảnh hưởng FE

| API | Loại | Hành động FE |
|-----|------|--------------|
| `POST /wardrobe-items/{id}/confirm-review` | Mới | Thêm nút/màn hình xác nhận (kèm chọn danh mục) |
| `POST /brand-portal/.../retry-analysis` | Mới | Thêm nút "Thử lại" cho brand item lỗi |
| `POST /wardrobe-items/{id}/retry-analysis` | Đổi | Nhận `taskId`; xử lý lỗi khi trạng thái không cho phép retry |
| `PATCH brand item status` | Đổi | Ẩn/khóa đổi status khi `failed` |
| `PUT brand item` | Đổi | Không gửi status không hợp lệ |
| `GET brand items` | Đổi | Hiển thị `failed` + reason |
| SSE | Đổi (additive) | Xử lý `processing`; xử lý replay/reconnect |
