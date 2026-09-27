# Event Contracts: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Date**: 2026-09-26

Mô tả event mới trên RabbitMQ. Tuân thủ `docs/development/messaging-guide.md` (đăng ký binding trong `topology.go`, không sửa `rabbitmq_connection.go`).

---

## 1. Event mới: `fashion.event.processing`

Tái dùng payload `FashionAnalyzedEventPayload` với `status="processing"` (không tạo DTO mới).

### 1.1 Định danh

| Thuộc tính | Wardrobe | Brand |
|-----------|----------|-------|
| Routing key | `fashion.event.processing.wardrobe` | `fashion.event.processing.brand` |
| Queue (tái dùng) | `wardrobe_fashion_analyzed_queue` | `brand_fashion_analyzed_queue` |
| Exchange | `smart_wardrobe_exchange` (topic) | `smart_wardrobe_exchange` (topic) |
| DLQ | `.dlq` của queue tương ứng (đã có) | `.dlq` của queue tương ứng (đã có) |

> Chỉ **bind thêm routing key** vào queue analyzed hiện có; không tạo queue/DLQ mới.

### 1.2 Payload

```json
{
  "taskId": "uuid",
  "itemId": "uuid",
  "itemType": "wardrobe | brand",
  "fashionItemId": "uuid",
  "status": "processing",
  "total": 3,
  "index": 0,
  "fashionBrief": null,
  "error": ""
}
```

### 1.3 Ai publish
`FashionWorkerUseCase.ProcessBackgroundBatchUploadJob` — ngay đầu luồng, **trước khi gọi AI**, trong `uow.Execute` (outbox). Chọn routing key theo `job.ItemType`.

### 1.4 Ai consume
`WardrobeFashionEventWorker` / `BrandFashionEventWorker` (queue analyzed hiện có). `HandleFashionEvent` dispatch thêm nhánh:
- `status == processing` → `HandleFashionProcessing` (chỉ phát SSE `processing`, không đổi DB status vì đã là `Processing`).

### 1.5 Tương thích
- Client cũ bỏ qua `status: processing`.
- Không đụng `fashion.event.processing-exhausted` (word sequence khác).

---

## 2. Event hiện có được tái sử dụng (không đổi)

| Event | Dùng cho | Ghi chú |
|-------|----------|---------|
| `fashion.event.analyzed.wardrobe/.brand` | completed/needs_review | Không đổi |
| `fashion.event.failed.wardrobe/.brand` | failed | Brand giờ set `failed` thay vì `draft` |
| `wardrobe.event.created` | Phân tích hoàn tất (fashion.event.analyzed.wardrobe) | Tái dùng để index ES; trước đây do confirm-review phát, nay do phân tích lại hoàn tất |
| `fashion.event.analyze` | brand/wardrobe retry job | Job có `preserveCategory`/`fixedCategory` |

---

## 3. Job DTO `FashionAnalyzeJobDTO` (bổ sung trường)

| Field | Kiểu | Ghi chú |
|-------|------|---------|
| `preserveCategory` | bool | **[MỚI]** Khi true, worker không ghi đè `category_id` từ AI |
| `fixedCategory` | bool | **[MỚI]** Khi true, worker chạy prompt metadata-only theo danh mục đã biết, bỏ gate `is_single_item`/`category_confident` và giữ danh mục (chế độ danh mục cố định, G11) |

---

## 4. SSE Event Payload (xuất ra client — không đổi cấu trúc)

```json
{
  "itemId": "uuid",
  "status": "processing | completed | failed | needs_review",
  "total": 3,
  "index": 0,
  "data": { "...": "WardrobeItemBriefRes | BrandItemBriefRes" },
  "error": "mã lý do khi failed (ví dụ analysis_temporary_error, multiple_items_detected)"
}
```

Bổ sung giá trị `processing` (additive). `data` vẫn đầy đủ cho `completed`/`needs_review`; `error` cho `failed` và là **mã lý do** (frontend tự ánh xạ sang text).

---

## 5. Checklist đăng ký topology (theo messaging-guide)

- [ ] `eventconstants.go`: `TopicFashionProcessingWardrobe`, `TopicFashionProcessingBrand`.
- [ ] `rabbitmq_constants.go`: routing key alias tương ứng.
- [ ] `topology.go`: thêm routing key vào `RoutingKeys` của 2 binding analyzed hiện có (không tạo binding/queue/DLQ mới).
- [ ] Consumer handlers: `HandleFashionProcessing` (wardrobe + brand) + dispatch.
- [ ] Producer: publish trong `analyze_uc.go`.
- [ ] Không chỉnh `rabbitmq_connection.go` (registry-driven).

---

## 6. G10 — Cập nhật event (bổ sung)

### 6.1 `FashionAnalyzedEventPayload.Error` = mã

- Sự kiện `failed` (`fashion.event.failed.wardrobe|brand`): trường `error` chuyển từ message tiếng Việt sang **mã** (`analysis_temporary_error`, `multiple_items_detected`, `full_body_outfit_detected`). SSE `error` giữ nguyên = mã này.

### 6.2 `fashion.event.processing-exhausted` — payload đổi

| Trước | Sau |
|-------|-----|
| `{ wardrobeItemId, fashionItemId, reason, userId? }` | `{ itemId, itemType ("wardrobe"\|"brand"), fashionItemId, reason, userId? }` |

- `reason` = mã `auto_retry_exceeded`.
- Thêm binding `QueueBrandProcessingExhausted` cho `TopicFashionProcessingExhausted` (+ DLQ entry).
- Consumer brand mới: `HandleFashionProcessingExhausted` → set brand item `failed`.
- Nguồn phát hiện: quét stale theo `fashion_items` (fashion sở hữu processing state); map owner qua contract wardrobe/brand.

### 6.3 Checklist topology (G10)

- [ ] `topology.go`: thêm `QueueBrandProcessingExhausted` (bind `RoutingKeyFashionProcessingExhausted`) + entry `DLQTopology`.
- [ ] Cập nhật consumer wardrobe + ES theo payload mới (`ItemID`/`ItemType`).
- [ ] Brand worker `processing_exhausted_worker.go` + handler.
- [ ] Không chỉnh `rabbitmq_connection.go`.
