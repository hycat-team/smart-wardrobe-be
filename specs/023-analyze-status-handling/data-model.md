# Phase 1 Data Model: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Date**: 2026-09-26

Tài liệu mô tả thay đổi dữ liệu/kiểu dữ liệu (KHÔNG bao gồm code triển khai). Tham chiếu entity hiện có tại module tương ứng.

---

## 1. Thay đổi schema (Goose migration — service `commerce`)

> Quyết định (xem `research.md` R4): KHÔNG thêm cột đếm/thời gian cho retry thủ công — chỉ kiểm soát bằng trạng thái. Do đó migration chỉ cần cập nhật CHECK constraint của brand.

### 1.1 `wardrobe.wardrobe_items`
Không thay đổi schema. `status` giữ nguyên `SMALLINT` (`wardrobestatus`: InWardrobe=0, Selling=1, Sold=2, Processing=3, Failed=4, NeedsReview=5). Không có CHECK constraint.

### 1.2 `brand.brand_items` (cập nhật CHECK)

**Constraint cập nhật**:
```sql
-- drop + add
chk_brand_items_status CHECK (status IN ('draft','active','archived','failed'))
```

### 1.3 Migration template

```sql
-- +goose Up
ALTER TABLE brand.brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_status;
ALTER TABLE brand.brand_items
    ADD CONSTRAINT chk_brand_items_status
    CHECK (status IN ('draft','active','archived','failed'));

-- +goose Down
ALTER TABLE brand.brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_status;
ALTER TABLE brand.brand_items
    ADD CONSTRAINT chk_brand_items_status
    CHECK (status IN ('draft','active','archived'));
```

> Lệnh: `make migrate-create svc=commerce name=add_brand_item_failed_status` → `make migrate-up svc=commerce`.

---

## 2. Thay đổi entity/DTO (Go)

### 2.1 `WardrobeItem` (entity)
Không thay đổi field.

### 2.2 `BrandItem` (entity)
Không thay đổi field.

> Lưu ý: KHÔNG dựa vào tag `default:DRAFT` của `Status` (không khớp DB); luôn set status tường minh.

### 2.3 `wardrobestatus` / `branditemstatus`
- `wardrobestatus`: không đổi.
- `branditemstatus`: thêm `Failed BrandItemStatus = "failed"`.

### 2.4 `analyzetaskstatus`
Thêm `Processing AnalyzeTaskStatus = "processing"`.

### 2.5 `FashionAnalyzeJobDTO` (event/job nội bộ)
Thêm `PreserveCategory bool`.

### 2.6 `FashionItemContractDTO` (contract fashion → ngoài)
Thêm (scalar, không nhúng entity):
- `ProcessingErrorReason *string`
- `ReviewReason *string`

Populate tại mapper fashion; dùng cho brand hiển thị lý do.

### 2.7 `BrandItemRes` / `BrandItemBriefRes` (portal DTO)
Thêm:
- `ProcessingErrorReason *string` (`json:"processingErrorReason,omitempty"`)
- `ReviewReason *string` (`json:"reviewReason,omitempty"`)

### 2.8 `FashionAnalyzedEventPayload` (dùng lại cho processing)
Giữ nguyên cấu trúc; khi `Status = "processing"` thì `FashionBrief`/`Error` để trống. Không tạo payload mới.

### 2.9 `WardrobeItemRes` (retry response)
Thêm `TaskID *string` (`json:"taskId,omitempty"`) — mã phiên để client subscribe SSE sau khi khởi tạo thử lại (FR-009). Cập nhật mapper tại `wardrobe/application/mapper/wardrobe.go`.

---

## 3. State transitions

### 3.1 WardrobeItem

```text
Processing ──(AI ok)──► InWardrobe
Processing ──(AI fail / exhausted)──► Failed
Processing ──(AI uncertain)──► NeedsReview
NeedsReview ──(confirm-review, có category hợp lệ)──► InWardrobe
Failed | NeedsReview ──(retry-analysis, trạng thái cho phép)──► Processing
```

Quy tắc:
- `confirm-review` chỉ hợp lệ từ `NeedsReview`, cần danh mục hợp lệ.
- `retry-analysis` chỉ từ `Failed`/`NeedsReview`; chặn khi đang `Processing` hoặc trạng thái khác; dùng chuyển trạng thái nguyên tử để tránh tạo phiên trùng.
- Item `InWardrobe` mới tham gia gợi ý phối đồ.

### 3.2 BrandItem

```text
(draft) ──(AI ok)──► active
(draft) ──(AI uncertain: needs_review)──► draft
(draft) ──(AI fail)──► failed
failed ──(retry-analysis / system)──► draft ──(AI ok)──► active
```

Quy tắc:
- `failed` là trạng thái cuối với thao tác thủ công: từ chối mọi `UpdateBrandItem*` đổi trạng thái rời khỏi `failed`, và từ chối đặt thủ công `failed`.
- Chỉ `retry-analysis` (hoặc hệ thống qua event) mới đưa item khỏi `failed`.

---

## 4. Config mới

| Khóa | Kiểu | Default | Env | Ý nghĩa |
|------|------|---------|-----|---------|
| `server.sse.replay_buffer_size` | int | 100 | `SSE_REPLAY_BUFFER_SIZE` | Số event giữ lại mỗi phiên để phát lại |
| `server.sse.replay_buffer_ttl_seconds` | int | 300 | `SSE_REPLAY_BUFFER_TTL_SECONDS` | Thời gian giữ buffer để phát lại cho client reconnect muộn |

- Thêm struct field tương ứng, block YAML trong `configs/config.yaml` + `configs/config.production.yaml`, override env trong `loader.go`, và validation (`> 0`).
- Buffer bị loại theo size (vượt N → bỏ cũ nhất) hoặc TTL; KHÔNG xóa ngay khi subscriber cuối cùng ngắt kết nối.
- Không có config cho retry (đã bỏ chặn theo count/cooldown).

---

## 5. Event/Realtime data

### 5.1 Event mới trên RabbitMQ

| Thuộc tính | Giá trị |
|-----------|---------|
| Routing key (wardrobe) | `fashion.event.processing.wardrobe` |
| Routing key (brand) | `fashion.event.processing.brand` |
| Queue (tái dùng) | `wardrobe_fashion_analyzed_queue` / `brand_fashion_analyzed_queue` |
| Payload | `FashionAnalyzedEventPayload` với `status="processing"` |
| Producer | `analyze_uc.ProcessBackgroundBatchUploadJob` (đầu luồng, trong outbox/uow) |

### 5.2 SSE event payload (giữ nguyên cấu trúc)

```json
{
  "itemId": "uuid",
  "status": "processing | completed | failed | needs_review",
  "total": 3,
  "index": 0,
  "data": { "...": "khi completed/needs_review" },
  "error": "khi failed"
}
```

### 5.3 Redis Pub/Sub channel

- Channel: `sse:task:{taskId}`; relay subscribe pattern `sse:task:*`.
- Payload: chính là bytes JSON của `SSEEventPayload`.

---

## 6. Ràng buộc & validation tổng hợp

- `confirm-review`: item thuộc user, status `NeedsReview`, category tồn tại và `IsWearableCategory == true`.
- `retry-analysis`: item thuộc user/brand hợp lệ, status ∈ {Failed, NeedsReview} (wardrobe) hoặc {Failed, Draft-review} (brand); claim nguyên tử chuyển trạng thái (không thêm ràng buộc count/thời gian).
- `UpdateBrandItemStatus`: status đích ∈ {draft, active, archived}; nếu status hiện tại là `failed` → từ chối.
- `UpdateBrandItem`: validate status và chặn khi đang `failed`.
- Idempotency: consumer bỏ update DB nếu status đã ở trạng thái đích; vẫn phát SSE nếu kênh mở.
