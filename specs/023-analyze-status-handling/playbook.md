# Playbook vận hành: xử lý trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Áp dụng cho**: B2C (tủ đồ cá nhân) và B2B (Brand Portal).

Tài liệu này mô tả kịch bản xử lý cho từng trạng thái phân tích, dành cho đội vận hành và hỗ trợ.

---

## 1. Trạng thái phân tích (AnalyzeTaskStatus)

| Trạng thái | Ý nghĩa | Kênh realtime |
|-----------|---------|----------------|
| `processing` | AI đang phân tích | SSE event `processing` |
| `completed` | Phân tích xong, món dùng được | SSE `completed` + `data` tóm tắt |
| `needs_review` | AI không đủ tự tin, cần người xác nhận | SSE `needs_review` + lý do |
| `failed` | Lỗi hệ thống | SSE `failed` + thông báo lỗi |

## 2. B2C — Tủ đồ cá nhân

### 2.1 `needs_review`
1. Người dùng mở món cần rà soát, chọn **category hợp lệ** (có thể phối đồ).
2. Gọi `POST /api/v1/wardrobe-items/{id}/retry-analysis` với body `{ "categoryId": "<uuid>" }` (bắt buộc).
3. Hệ thống chạy lại AI ở chế độ **danh mục cố định** (chỉ sinh metadata, giữ danh mục) → món chuyển sang `inWardrobe`, tham gia gợi ý phối đồ. Nếu thiếu `categoryId` hoặc category không hợp lệ → từ chối (yêu cầu chọn category).
4. API `confirm-review` đã bị gỡ (gọi sẽ nhận `404`).

### 2.2 `failed`
1. Người dùng gọi `POST /api/v1/wardrobe-items/{id}/retry-analysis`.
2. Chỉ chấp nhận khi trạng thái là `failed` hoặc `needs_review` (đang xử lý/đã dùng được → từ chối).
3. Danh mục người dùng đã sửa được **giữ nguyên**; response trả `taskId` để theo dõi realtime.

## 3. B2B — Brand Portal

### 3.1 Phân biệt `failed` vs `draft`
- `failed`: lỗi hệ thống, kèm `processingErrorReason`.
- `draft`: AI không chắc, kèm `reviewReason`.
- Nhân viên **không thể** tự đổi trạng thái rời khỏi `failed`; chỉ `retry-analysis` hoặc hệ thống mới đưa ra.

### 3.2 Thử lại
1. `POST /api/v1/brand-portal/brands/{brandId}/items/{itemId}/retry-analysis` (body tùy chọn `{ "categoryId": "<uuid>" }`).
2. Chấp nhận khi `failed`, hoặc `draft` có `reviewReason`; nháp thường bị từ chối.
3. Với `draft` chờ duyệt, `categoryId` là **bắt buộc** (chạy chế độ danh mục cố định).
4. Trả `taskId` để subscribe SSE.

## 4. Realtime best-effort (quan trọng cho vận hành)

- Sự kiện realtime (SSE) là **best-effort**: lỗi phát sự kiện (Redis Pub/Sub gián đoạn) chỉ ghi log cảnh báo và **không** làm hỏng nghiệp vụ; DB là nguồn sự thật.
- Khi client không nhận được sự kiện: mở lại màn hình hoặc refetch danh sách (FE dự phòng).
- Tín hiệu quan sát tại `GET /api/v1/commerce/health` (khóa `realtime`):
  - `realtime_publish_total` / `realtime_publish_failed`
  - `realtime_relay_total` / `realtime_relay_failed`
  - `realtime_subscribers`

## 5. Checklist sự cố

- [ ] Sự kiện realtime không tới client → kiểm tra Redis kết nối và `realtime_*_failed`; xác nhận nghiệp vụ (DB) vẫn đúng.
- [ ] Món kẹt ở `needs_review` → hướng dẫn chọn category rồi `retry-analysis` kèm `categoryId` (không còn confirm-review).
- [ ] Sản phẩm brand kẹt ở `failed` → dùng retry-analysis (không đổi trạng thái thủ công).
- [ ] Nghi ngờ mất sự kiện âm thầm → đối chiếu `realtime_publish_total` vs `realtime_relay_total` và log `warn` kèm `taskId`.

## 6. Mã lý do & phân loại ảnh (G10)

### 6.1 Bảng mã lý do

| Mã | Nhóm | Trạng thái | Retry | Xử lý |
|----|------|-----------|-------|-------|
| `uncertain_category` | review | `needs_review` | ✅ (bắt buộc `categoryId`) | Người dùng chọn category + `retry-analysis` (danh mục cố định) |
| `multiple_items_detected` | error | `failed` | ❌ | Yêu cầu tải ảnh khác |
| `full_body_outfit_detected` | error | `failed` | ❌ | Yêu cầu ảnh cận một món |
| `analysis_temporary_error` | error | `failed` | ✅ | Cho thử lại |
| `auto_retry_exceeded` | error | `failed` | ✅ | Cho thử lại |

- Reason là **mã**, chỉ hiển thị trong `fashionItem` (nested); không có ở root.

### 6.2 Ảnh không hợp lệ

- Ảnh đa món/toàn thân ⇒ `failed` + mã lý do tương ứng; **không** retry (retry cùng ảnh vô nghĩa).
- Người dùng (wardrobe) hoặc nhân viên (brand) cần **tải ảnh khác**; với brand là **tạo sản phẩm mới**.

### 6.3 Brand hết retry tự động

- Recovery quét theo fashion item (fashion sở hữu trạng thái xử lý) và áp dụng cho cả wardrobe lẫn brand.
- Brand nhận event `processing-exhausted` ⇒ chuyển `failed` + `auto_retry_exceeded` (không kẹt `draft`).

### 6.4 Checklist sự cố (G10)

- [ ] Món/sản phẩm ở `failed` với lý do `multiple_items_detected`/`full_body_outfit_detected` → hướng dẫn tải ảnh khác (không retry).
- [ ] Brand kẹt `draft` sau nhiều lần xử lý → kiểm tra recovery worker + event `processing-exhausted`; kỳ vọng chuyển `failed` + `auto_retry_exceeded`.
- [ ] Kiểm tra reason luôn là mã và nằm trong `fashionItem` (không ở root).
