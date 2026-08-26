# Hợp đồng Sự kiện & Hàng đợi (Event & Queue Contracts)

**Feature**: [spec.md](../spec.md) | **Date**: 2026-08-26

---

## 1. RabbitMQ Exchanges & Topics

* **Exchange**: `smartwardrobe.topic` (Topic Exchange)
* **Topics liên quan**:
  * `fashion.event.init`: Wardrobe / Brand đẩy job phân tích cho Fashion AI Worker.
  * `fashion.event.analyzed`: Fashion AI Worker phát khi phân tích hoàn tất (thành công hoặc cần review).
  * `fashion.event.failed`: Fashion AI Worker phát khi xử lý thất bại.

---

## 2. Queues & Routing Keys

| Queue Name | Routing Key Pattern | Consumer Handler | Chức năng |
| :--- | :--- | :--- | :--- |
| `wardrobe.fashion.analyzed.queue` | `fashion.event.*` | `WardrobeAnalyzeConsumer` | Lắng nghe kết quả AI cho wardrobe items, update DB wardrobe & bắn SSE |
| `brand.fashion.analyzed.queue` | `fashion.event.*` | `BrandAnalyzeConsumer` | Lắng nghe kết quả AI cho brand items, update DB brand & bắn SSE |

---

## 3. Quy trình Chuyển giao Sự kiện (Event Flow Lifecycle)

1. **Khởi tạo**:
   * Client $\rightarrow$ `POST /api/v1/wardrobe-items/batch-upload` $\rightarrow$ Wardrobe lưu `wardrobe_items` (status: `pending`) $\rightarrow$ Ghi Outbox `fashion.event.init`.
2. **AI Xử lý**:
   * Fashion Worker nhận message $\rightarrow$ Gọi Vision AI $\rightarrow$ Lưu `fashion_items` $\rightarrow$ Ghi Outbox `fashion.event.analyzed` (hoặc `fashion.event.failed`).
3. **Phản hồi Nghiệp vụ & Realtime**:
   * `WardrobeAnalyzeConsumer` nhận `fashion.event.analyzed` $\rightarrow$ Cập nhật `wardrobe_item.status = in_wardrobe` $\rightarrow$ Tạo `WardrobeItemBriefRes` $\rightarrow$ Bắn vào `sseManager.Publish(taskId, payload)`.
   * Client đang mở stream SSE `/api/v1/wardrobe-items/tasks/{taskId}/sse` nhận được event `message` và hiển thị UI.
