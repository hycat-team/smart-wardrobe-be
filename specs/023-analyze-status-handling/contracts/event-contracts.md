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
| `wardrobe.event.created` | confirm-review publish | Tái dùng để index ES |
| `fashion.event.analyze` | brand/wardrobe retry job | Job có `preserveCategory` |

---

## 3. Job DTO `FashionAnalyzeJobDTO` (bổ sung trường)

| Field | Kiểu | Ghi chú |
|-------|------|---------|
| `preserveCategory` | bool | **[MỚI]** Khi true, worker không ghi đè `category_id` từ AI |

---

## 4. SSE Event Payload (xuất ra client — không đổi cấu trúc)

```json
{
  "itemId": "uuid",
  "status": "processing | completed | failed | needs_review",
  "total": 3,
  "index": 0,
  "data": { "...": "WardrobeItemBriefRes | BrandItemBriefRes" },
  "error": "thông báo lỗi khi failed"
}
```

Bổ sung giá trị `processing` (additive). `data` vẫn đầy đủ cho `completed`/`needs_review`; `error` cho `failed`.

---

## 5. Checklist đăng ký topology (theo messaging-guide)

- [ ] `eventconstants.go`: `TopicFashionProcessingWardrobe`, `TopicFashionProcessingBrand`.
- [ ] `rabbitmq_constants.go`: routing key alias tương ứng.
- [ ] `topology.go`: thêm routing key vào `RoutingKeys` của 2 binding analyzed hiện có (không tạo binding/queue/DLQ mới).
- [ ] Consumer handlers: `HandleFashionProcessing` (wardrobe + brand) + dispatch.
- [ ] Producer: publish trong `analyze_uc.go`.
- [ ] Không chỉnh `rabbitmq_connection.go` (registry-driven).
