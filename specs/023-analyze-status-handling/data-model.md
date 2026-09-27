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
NeedsReview ──(retry-analysis kèm category hợp lệ, chế độ danh mục cố định)──► Processing ──► InWardrobe
Failed | NeedsReview ──(retry-analysis, trạng thái cho phép)──► Processing
```

Quy tắc:
- `needs_review` xử lý bằng `retry-analysis` kèm danh mục hợp lệ (chế độ danh mục cố định, giữ danh mục); KHÔNG còn `confirm-review` (xem §10).
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
  "error": "mã lý do (chỉ khi failed)"
}
```

### 5.3 Redis Pub/Sub channel

- Channel: `sse:task:{taskId}`; relay subscribe pattern `sse:task:*`.
- Payload: chính là bytes JSON của `SSEEventPayload`.

---

## 6. Ràng buộc & validation tổng hợp

- `retry-analysis`: item thuộc user/brand hợp lệ, status ∈ {Failed, NeedsReview} (wardrobe) hoặc {Failed, Draft-review} (brand); claim nguyên tử chuyển trạng thái (không thêm ràng buộc count/thời gian). Với `NeedsReview`/`Draft-review` yêu cầu danh mục hợp lệ (`IsWearableCategory == true`, từ request hoặc có sẵn) và chạy chế độ danh mục cố định (xem §10).
- `UpdateBrandItemStatus`: status đích ∈ {draft, active, archived}; nếu status hiện tại là `failed` → từ chối.
- `UpdateBrandItem`: validate status và chặn khi đang `failed`.
- Idempotency: consumer bỏ update DB nếu status đã ở trạng thái đích; vẫn phát SSE nếu kênh mở.

---

## 7. Thay đổi chữ ký repository (G8 — KHÔNG đổi schema/dữ liệu)

> US5 không thêm cột, không migration. Chỉ thay đổi hợp đồng nội bộ module `wardrobe` để mang "ý định lọc" tường minh.

### 7.1 Interface `WardrobeItemRepository` (domain/repositories/interfaces.go)

| Method | Trước | Sau |
|--------|-------|-----|
| `CountByUserIDAndFilters` | `(ctx, userID, fashionItemIDs []uuid.UUID, statuses)` | `+ filterByFashionItems bool` |
| `GetByUserIDAndFiltersPaginated` | `(ctx, userID, fashionItemIDs, statuses, pagination)` | `+ filterByFashionItems bool` |
| `CountItems` | `(ctx, fashionItemIDs, itemType)` | `+ filterByFashionItems bool` |
| `GetItems` / `GetItemsPaginated` | `(ctx, fashionItemIDs, itemType[, pagination])` | `+ filterByFashionItems bool` |

### 7.2 Ngữ nghĩa

| `filterByFashionItems` | `fashionItemIDs` | Hành vi |
|------------------------|------------------|---------|
| `false` | (bất kỳ) | Không lọc ⇒ trả tất cả món (kể cả `fashion_item_id` NULL) |
| `true` | rỗng | Áp bộ lọc rỗng ⇒ 0 kết quả (usecase short-circuit, không query) |
| `true` | có phần tử | `fashion_item_id IN (...)` |

### 7.3 Call site

- `GetWardrobeItems`: `filterByFashionItems = (categorySlug != nil)`.
- `GetSystemCatalogWardrobeItems` (fallback DB search) và `catalog_uc.go`: `filterByFashionItems = (queryStr != nil || categorySlug != nil)`.
- Các caller cũ truyền `nil` với ý nghĩa "không lọc" (`GetByUserID` cho locked-map/outfit) giữ hành vi trả tất cả.

---

## 8. Tín hiệu quan sát realtime (G9 — KHÔNG đổi schema/dữ liệu)

> Best-effort (FR-025) + observability (FR-026). Không thêm bảng/cột; chỉ counter/gauges in-memory theo pattern `shared/infrastructure/messaging/metrics.go`.

### 8.1 Metrics (package `shared/application/sse`, file `realtime_metrics.go`)

| Tên metric | Kiểu | Ý nghĩa | Tăng/giảm tại |
|------------|------|---------|----------------|
| `realtime_publish_total` | counter | Số lần publish sự kiện realtime | `RedisRealtimePublisher.Publish` (mọi lần) |
| `realtime_publish_failed` | counter | Số lần publish lỗi (Redis down/timeout) | nhánh lỗi publish (kèm log warn + taskId) |
| `realtime_relay_total` | counter | Số sự kiện relay nhận từ Pub/Sub và forward | `RealtimeRelay` khi forward thành công |
| `realtime_relay_failed` | counter | Số lần relay lỗi/hủy forward | nhánh lỗi relay |
| `realtime_subscribers` | gauge | Tổng số subscriber đang mở (theo phiên) | `SSEManager.Subscribe`/`Unsubscribe` (atomic +1/−1) |

- Hàm `GetRealtimeMetrics() map[string]int64` phơi qua health payload `GET /api/v1/commerce/health` (khóa `"realtime"`).

### 8.2 Quy tắc hành vi

- Lỗi publish/relay: log `warn` kèm `taskId`; **không** trả lỗi lên usecase/worker; trạng thái DB vẫn được ghi.
- Gauge `realtime_subscribers` phản ánh số kênh đang mở, không cần chính xác tuyệt đối theo thời điểm.
- Không thêm config/dependency mới cho metrics.

---

## 9. Phân loại ảnh không hợp lệ + mã hóa lý do (G10 — KHÔNG đổi schema)

> US6/FR-027..FR-036. Không thêm cột/bảng; reason đã ở `fashion_items` (`processing_error_reason`, `review_reason`). Không thêm status mới.

### 9.1 Catalog mã reason (hằng số `fashion/domain/constants/analysisreason`)

| Nhóm | Mã | Ý nghĩa | Trạng thái |
|------|----|---------|-----------|
| Review | `uncertain_category` | Ảnh một món nhưng AI chưa chắc category | `needs_review` |
| Error | `multiple_items_detected` | Ảnh chứa nhiều món | `failed` |
| Error | `full_body_outfit_detected` | Ảnh toàn thân/outfit | `failed` |
| Error | `analysis_temporary_error` | Lỗi tạm thời khi phân tích | `failed` |
| Error | `auto_retry_exceeded` | Hết retry tự động | `failed` |

### 9.2 Định tuyến AI → trạng thái

| Tín hiệu AI | Trạng thái | Reason |
|-------------|-----------|--------|
| `is_single_item = false` | `failed` | `multiple_items_detected` / `full_body_outfit_detected` |
| `is_single_item = true` && `category_confident = false` | `needs_review` | `uncertain_category` |
| `is_single_item = true` && `category_confident = true` | `completed` | (xóa reason) |

- `FashionMetadataResult` thêm `CategoryConfident bool`. Mặc định an toàn:
  - thiếu/không xác định `is_single_item` ⇒ ảnh không hợp lệ ⇒ `failed`.
  - `is_single_item=true` nhưng thiếu/null `category_confident` ⇒ mặc định **false** ⇒ `needs_review` (`uncertain_category`).
  - mâu thuẫn `is_single_item=false` + `review_reason=uncertain_category` ⇒ ưu tiên ảnh không hợp lệ ⇒ `failed` + `multiple_items_detected`.

### 9.3 State machine cập nhật (WardrobeItem / BrandItem)

```text
Processing ──(AI ok)──► InWardrobe / active
Processing ──(AI: ảnh không hợp lệ)──► Failed (+ multiple_items_detected|full_body_outfit_detected)   [không retry]
Processing ──(AI: một món, chưa chắc category)──► NeedsReview / draft (+ uncertain_category)         [retry kèm category]
Processing ──(AI fail / exhausted)──► Failed (+ analysis_temporary_error|auto_retry_exceeded)        [retry được]
```

### 9.4 Hiển thị reason (chỉ nested trong `fashionItem`)

- Bỏ `ReviewReason`/`ProcessingErrorReason` ở root `BrandItemRes`/`BrandItemBriefRes`.
- `FashionItemContractDTO` (camelCase) và `FashionItemBriefRes` (`reviewReason`, `processingErrorReason`); `FashionItemRes` (wardrobe projection) thêm 2 field.
- Không thêm field reason ở root `WardrobeItemRes`/`WardrobeItemBriefRes`.

### 9.5 Payload `processing-exhausted` (đổi)

| Trước | Sau |
|-------|-----|
| `{ wardrobeItemId, fashionItemId, reason, userId? }` | `{ itemId, itemType ("wardrobe"\|"brand"), fashionItemId, reason, userId? }` |

- Consumer mới cho brand (`QueueBrandProcessingExhausted`) → set brand item `failed`.

### 9.6 Quy tắc gate

- **retry-analysis** (wardrobe/brand): chặn khi `failed` với reason ∈ {`multiple_items_detected`, `full_body_outfit_detected`}; cho phép với `analysis_temporary_error`, `auto_retry_exceeded`, hoặc `needs_review`/`draft-review` (khi đó **bắt buộc** kèm danh mục hợp lệ — xem §10).
- SSE `error` (event `failed`) = mã reason.
- Thử lại thất bại ở bước publish job ⇒ MUST ghi `processingErrorReason=analysis_temporary_error` (qua fashion item) cùng lúc set `failed` (FR-036).

---

## 10. Bỏ confirm-review — gộp vào retry kèm danh mục (G11 — KHÔNG đổi schema)

> US1 (viết lại)/FR-001..003/037..041. Không thêm cột/bảng; không thêm status.

### 10.1 API

- **Gỡ** `POST /wardrobe-items/{id}/confirm-review`.
- `POST /wardrobe-items/{id}/retry-analysis` nhận body tùy chọn `{ "categoryId": "uuid" }`.
- `POST /brand-portal/brands/{brandId}/items/{itemId}/retry-analysis` nhận body tùy chọn `{ "categoryId": "uuid" }`.

### 10.2 Quy tắc retry theo trạng thái

| Trạng thái | categoryId | Hành vi |
|-----------|-----------|---------|
| `NeedsReview` (`uncertain_category`) | **bắt buộc** (request, fallback danh mục có sẵn) | validate wearable → persist danh mục → chạy lại AI **danh mục cố định** |
| `Failed` (transient) | tùy chọn | nếu có danh mục ⇒ danh mục cố định; nếu không ⇒ phân tích thường |
| `Failed` (ảnh không hợp lệ) | — | **từ chối** (hướng tải ảnh khác) |

### 10.3 Chế độ phân tích "danh mục cố định"

- `FashionAnalyzeJobDTO` thêm `FixedCategory bool`.
- Worker (`ProcessBackgroundBatchUploadJob`): khi `FixedCategory` ⇒ dùng prompt **metadata-only** theo danh mục đã biết; **bỏ** gate `is_single_item`/`category_confident`; `PreserveCategory=true` ⇒ không ghi đè danh mục; chỉ ghi metadata + embedding.
- Kết quả: `NeedsReview → Processing → InWardrobe` (wardrobe) / `active` (brand); lỗi tạm thời ⇒ `Failed`; **không** quay lại `needs_review`.

### 10.4 Prompt fail-fast (initial analyze)

- Thứ tự quyết định và shape tối giản:
  - (a) Không có món thời trang ⇒ `{ "error": "No fashion item detected" }`.
  - (b) Ảnh không hợp lệ (đa món/toàn thân) ⇒ `{ "is_single_item": false, "review_reason": "multiple_items_detected" | "full_body_outfit_detected" }`.
  - (c) Một món nhưng chưa chắc danh mục ⇒ `{ "is_single_item": true, "category_confident": false, "review_reason": "uncertain_category" }`.
  - (d) Còn lại ⇒ JSON đầy đủ (`category_slug` + metadata + `category_confident: true`).
- Giữ **1 call, KHÔNG `ResponseSchema`** để fail-fast có tác dụng.
- Backend vẫn áp default an toàn khi model trả thiếu field (xem §9.2).

### 10.5 State machine cập nhật (WardrobeItem / BrandItem)

```text
Processing ──(AI ok)──► InWardrobe / active
Processing ──(AI: ảnh không hợp lệ)──► Failed (+ multiple_items_detected|full_body_outfit_detected)   [không retry]
Processing ──(AI: một món, chưa chắc category)──► NeedsReview / draft (+ uncertain_category)
NeedsReview / draft ──(retry + categoryId hợp lệ)──► Processing ──(AI danh mục cố định ok)──► InWardrobe / active
NeedsReview / draft ──(retry + categoryId, AI lỗi tạm thời)──► Failed (+ analysis_temporary_error)   [retryable]
```
