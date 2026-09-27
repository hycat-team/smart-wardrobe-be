# API Contracts: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Date**: 2026-09-26

Quy ước chung: auth Bearer; JSON camelCase; response bọc `APIResponse { success, message, data }` (trừ SSE). Đây là hợp đồng hướng frontend — bản đầy đủ dành cho FE nằm ở [`../frontend-guide.md`](../frontend-guide.md).

---

## 1. API MỚI

### 1.1 ĐÃ GỠ — `POST /api/v1/wardrobe-items/{id}/confirm-review`

Endpoint xác nhận đồng bộ `needs_review → inWardrobe` **đã bị gỡ bỏ** (breaking). Lý do: với G10 không lưu metadata ở `needs_review`, endpoint này chỉ set danh mục + trạng thái mà fashion item rỗng metadata/embedding. `needs_review` giờ xử lý bằng `retry-analysis` kèm `categoryId` (xem §2.1). Client cũ gọi endpoint này sẽ nhận `404`.

### 1.2 Thử lại phân tích cho sản phẩm nhãn hàng (Brand Portal, Owner/Staff)

| Thuộc tính | Giá trị |
|-----------|---------|
| Method/Path | `POST /api/v1/brand-portal/brands/{brandId}/items/{itemId}/retry-analysis` |
| Role | `user`/`admin` + brand membership (Owner/Staff) |
| Mô tả | Đưa sản phẩm `failed` (hoặc `draft` cần rà soát) lại hàng đợi phân tích; với `draft` cần rà soát **bắt buộc** kèm `categoryId` |

**Request** (body tùy chọn):
```json
{ "categoryId": "uuid" }
```

**Response 200**
```json
{
  "success": true,
  "message": "Yêu cầu phân tích lại sản phẩm đã được gửi thành công",
  "data": {
    "id": "uuid",
    "brandId": "uuid",
    "status": "draft",
    "taskId": "uuid",
    "fashionItem": {
      "id": "uuid",
      "reviewReason": "uncertain_category",
      "processingErrorReason": null
    }
  }
}
```

**Lỗi**:
| HTTP | Điều kiện |
|------|-----------|
| 403 | Không thuộc brand / item không ở trạng thái cho phép (đang xử lý hoặc đã sử dụng) |
| 400 | `draft` cần rà soát mà thiếu/không hợp lệ `categoryId` |
| 404 | Không tìm thấy brand/item |

---

## 2. API THAY ĐỔI HÀNH VI (trước → sau)

### 2.1 `POST /api/v1/wardrobe-items/{id}/retry-analysis`

| | Trước | Sau |
|---|-------|-----|
| Hành vi | Chỉ chuyển trạng thái + phát job | Thêm guard trạng thái + chống phiên trùng (claim nguyên tử); **trả thêm `taskId`** để subscribe SSE; giữ danh mục người dùng |
| Request | không body | body tùy chọn `{ "categoryId": "uuid" }`; với `needsReview` (`uncertain_category`) **bắt buộc** `categoryId` (hoặc danh mục có sẵn) |
| Phân tích | Phân tích thường (AI tự phân loại danh mục) | Có danh mục ⇒ chạy **danh mục cố định** (metadata-only, không tái sinh `needs_review`, giữ danh mục) |
| Response | `WardrobeItemRes` (không có taskId) | `WardrobeItemRes` + `taskId` (mới, optional) |
| Lỗi mới | — | `retry khi trạng thái không hợp lệ` (đang xử lý / đã sử dụng); `needsReview` thiếu/không hợp lệ `categoryId` |

**Sau** (có danh mục)
```json
{ "id": "uuid", "status": 3, "isLocked": false, "fashionItem": { }, "taskId": "uuid" }
```
Row `needs_review` điển hình: `status` 3 (Processing) → khi AI xong `status` 0 (inWardrobe) với metadata đầy đủ và danh mục đã chọn được giữ.

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

### 2.7 `GET /api/v1/wardrobe-items?categorySlug=...` và `GET /api/v1/wardrobe-items/catalog?categorySlug|query` (US5/G8)

| | Trước | Sau |
|---|-------|-----|
| Danh mục/từ khóa không khớp món nào | Trả về **toàn bộ** tủ đồ/catalog (bug hồi quy `7b1c046a`) | Trả về **rỗng** (`data: []`, `total: 0`) |
| Danh mục/từ khóa không truyền | Trả toàn bộ (kể cả món không có fashion item) | Giữ nguyên: trả toàn bộ |

> Không thay đổi hình dạng request/response — chỉ sửa kết quả trả về cho case "lọc nhưng rỗng". FE không cần đổi tích hợp, nhưng có thể bỏ mọi workaround chống hiển thị sai.

### 2.8 `GET /api/v1/commerce/health` (ops — không phải hợp đồng client)

| | Trước | Sau |
|---|-------|-----|
| Payload | `status`, `service`, `time`, `mode`, `elasticsearch` | Thêm khóa `realtime` (counters/gauges: publish/relay total–failed, subscribers) |

> Best-effort: khi Redis lỗi, thao tác nghiệp vụ vẫn thành công (FR-025); health giúp vận hành phát hiện mất sự kiện (FR-026). Client không phụ thuộc endpoint này.

> Các API khác giữ nguyên. FE không cần refetch khi nhận `completed`/`needs_review` vì `data` đã đầy đủ.

---

## 3. Ma trận ảnh hưởng FE

| API | Loại | Hành động FE |
|-----|------|--------------|
| `POST /wardrobe-items/{id}/confirm-review` | GỠ (breaking) | Chuyển sang `retry-analysis` kèm `categoryId` |
| `POST /brand-portal/.../retry-analysis` | Mới | Thêm nút "Thử lại" cho brand item lỗi |
| `POST /wardrobe-items/{id}/retry-analysis` | Đổi | Nhận `taskId`; xử lý lỗi khi trạng thái không cho phép retry |
| `PATCH brand item status` | Đổi | Ẩn/khóa đổi status khi `failed` |
| `PUT brand item` | Đổi | Không gửi status không hợp lệ |
| `GET brand items` | Đổi | Hiển thị `failed` + reason |
| `GET /wardrobe-items?categorySlug` | Đổi (bugfix) | Hiển thị đúng `total=0`/rỗng khi danh mục không có món |
| `GET /wardrobe-items/catalog` | Đổi (bugfix) | Hiển thị đúng rỗng khi `categorySlug`/`query` không khớp |
| SSE | Đổi (additive) | Xử lý `processing`; xử lý replay/reconnect |

---

## 4. G10 — Reason & phân loại ảnh không hợp lệ (bổ sung)

### 4.1 Catalog mã lý do

| Nhóm | Mã | Trạng thái | Retry |
|------|----|-----------|-------|
| Review | `uncertain_category` | `needs_review` | ✅ (bắt buộc kèm `categoryId`) |
| Error | `multiple_items_detected` | `failed` | ❌ |
| Error | `full_body_outfit_detected` | `failed` | ❌ |
| Error | `analysis_temporary_error` | `failed` | ✅ |
| Error | `auto_retry_exceeded` | `failed` | ✅ |

### 4.2 Vị trí & casing

- Reason **chỉ** nằm trong đối tượng `fashionItem` lồng nhau (camelCase: `reviewReason`, `processingErrorReason`). KHÔNG có ở root wardrobe/brand item.
- `FashionItemContractDTO` serialize **camelCase** (trước: snake_case) — đổi hành vi của object `fashionItem` trong response Brand Portal.
- SSE event `failed`: `error` là **mã** (trước: message tiếng Việt).

### 4.3 Ảnh không hợp lệ

- Ảnh đa món/toàn thân ⇒ `failed` + mã lý do; `retry-analysis` **bị từ chối**; FE hướng dẫn tải ảnh khác.
- `needs_review` chỉ còn `uncertain_category` (ảnh một món, chưa chắc category).
