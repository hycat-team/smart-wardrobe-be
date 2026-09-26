# Frontend Integration Guide — Xử lý trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Trạng thái**: Khung tài liệu — nội dung chi tiết sẽ được hoàn thiện trong quá trình triển khai (plan/tasks/implement).
**Đối tượng đọc**: Lập trình viên frontend (web/mobile).
**Mục tiêu**: FE chỉ cần đọc tài liệu này để triển khai, KHÔNG cần đọc code backend hay Swagger.

> Quy ước: mọi trường JSON dùng camelCase. Nội dung này mô tả hành vi API ở mức hợp đồng; chi tiết nội bộ backend không nằm trong phạm vi.

---

## 0. Tóm tắt nhanh (TL;DR)

- Một món đồ khi tải lên đi qua các trạng thái phân tích: `processing → completed | failed | needs_review`.
- Có thêm trạng thái realtime `processing` (mới) để FE hiển thị "đang xử lý".
- Món `needs_review` PHẢI được người dùng xác nhận (và PHẢI có danh mục hợp lệ) mới trở thành "sử dụng được".
- Sản phẩm nhãn hàng có thêm trạng thái `failed`; đây là trạng thái cuối với thao tác thủ công (chỉ retry/hệ thống mới đưa ra khỏi `failed`).
- Thao tác retry chỉ khả dụng khi món/sản phẩm đang ở trạng thái lỗi (hoặc cần rà soát); giữ nguyên danh mục người dùng đã sửa.
- Kết nối realtime nếu đứt sẽ được phát lại các sự kiện gần nhất khi kết nối lại.

---

## 1. Tổng quan trạng thái

### 1.1 State machine (món đồ cá nhân — WardrobeItem)

```text
                batch-upload / retry-analysis
        (create)          (retry)
           │                 │
           ▼                 │
      ┌──────────┐  retry   │
      │processing│◄─────────┘
      └────┬─────┘
           │
   ┌───────┼────────────────┐
   ▼       ▼                ▼
completed failed        needs_review
 (dùng   (lỗi)          (cần rà soát)
  được)   │                 │
          │ retry           │ confirm-review (cần danh mục hợp lệ)
          └──────► processing ◄──────┘
```

### 1.2 Trạng thái phân tích (`status` trong sự kiện realtime)

| `status` | Ý nghĩa | Client cần làm |
|----------|---------|----------------|
| `processing` | Món bắt đầu được xử lý | Hiển thị trạng thái "đang xử lý"; chờ sự kiện kết thúc |
| `completed` | Phân tích xong, dùng được | Render món bằng `data` (không cần gọi lại API) |
| `needs_review` | AI không chắc / không phân loại được | Điều hướng tới màn hình rà soát; cho sửa danh mục rồi gọi xác nhận |
| `failed` | Phân tích thất bại | Hiển thị `error` thân thiện; gợi ý "Thử lại" |

### 1.3 Trạng thái nghiệp vụ trả trong `data.status`

| Đối tượng | Giá trị | Ý nghĩa |
|-----------|---------|---------|
| WardrobeItem | `inWardrobe` (0) | Sử dụng được |
| WardrobeItem | `processing` (3) | Đang xử lý |
| WardrobeItem | `failed` (4) | Lỗi phân tích |
| WardrobeItem | `needsReview` (5) | Cần rà soát |
| BrandItem | `active` | Đang hoạt động |
| BrandItem | `draft` | Nháp / chờ duyệt |
| BrandItem | `failed` | Lỗi phân tích (trạng thái cuối với thao tác thủ công) |
| BrandItem | `archived` | Lưu trữ |

> (Giá trị số/chuỗi thực tế sẽ chốt khi triển khai; FE nên xử lý theo tên trạng thái, không hardcode số.)

---

## 2. Kịch bản xử lý theo từng trạng thái

### 2.1 `completed`
- Điều kiện: AI phân tích thành công.
- Sự kiện realtime: `status: "completed"`, có `data` đầy đủ (món + metadata danh mục/ảnh/màu/phong cách).
- FE: cập nhật danh sách bằng `data` trực tiếp, KHÔNG cần refetch.

### 2.2 `failed`
- Điều kiện: lỗi đọc ảnh / lỗi AI vĩnh viễn / hết retry hệ thống.
- Sự kiện realtime: `status: "failed"`, có `error` (thông báo thân thiện).
- FE: hiển thị lỗi + nút "Thử lại"; gọi API retry (mục 3).
- Lưu ý: nút "Thử lại" chỉ bật khi món đang ở trạng thái lỗi; khi đang xử lý hoặc đã dùng được thì không cho retry (backend từ chối nếu gọi sai).

### 2.3 `needs_review`
- Điều kiện: AI không đủ tự tin hoặc không phân loại được.
- Sự kiện realtime: `status: "needs_review"`, có `data` (có thể thiếu metadata).
- FE:
  1. Hiển thị màn hình rà soát.
  2. Cho người dùng chọn **danh mục hợp lệ** (bắt buộc) trước khi xác nhận.
  3. Gọi API xác nhận (mục 3) để đưa món vào sử dụng.
  4. (Tùy chọn) Cho phép "Thử lại AI" nếu người dùng muốn AI phân tích lại.

### 2.4 `processing` (mới)
- Điều kiện: món bắt đầu/được đưa lại hàng đợi phân tích.
- Sự kiện realtime: `status: "processing"`.
- FE: hiển thị "đang xử lý" cho đúng món (theo `itemId`), cập nhật tiến trình theo `index/total`.

### 2.5 Sản phẩm nhãn hàng (Brand)
- `completed` → sản phẩm `active`.
- `needs_review` → sản phẩm `draft` (chờ duyệt) + lý do.
- `failed` → sản phẩm `failed` + lý do; trạng thái `failed` **không thể** đổi bằng thao tác chỉnh sửa trạng thái thủ công — chỉ có thể "Thử lại" (hoặc hệ thống).

---

## 3. API: mới & thay đổi hành vi (trước → sau)

> PHẦN NÀY SẼ ĐƯỢC ĐIỀN ĐẦY ĐỦ KHI TRIỂN KHAI. Mỗi API đổi hành vi phải ghi "trước → sau" + ví dụ payload.

### 3.1 API mới

| API | Mô tả | Ai dùng |
|-----|-------|---------|
| `POST /wardrobe-items/{id}/confirm-review` | Xác nhận món "cần rà soát" → sử dụng được (yêu cầu danh mục hợp lệ) | User |
| `POST /brand-portal/brands/{brandId}/items/{itemId}/retry-analysis` | Thử lại phân tích cho sản phẩm nhãn hàng lỗi/cần rà soát | Brand staff |

### 3.2 API thay đổi hành vi

| API | Trước | Sau | Ảnh hưởng FE |
|-----|-------|-----|--------------|
| `POST /wardrobe-items/{id}/retry-analysis` | Không trả mã phiên | Trả kèm `taskId` để theo dõi realtime; chỉ cho phép khi trạng thái lỗi/cần rà soát | Cập nhật luồng retry |
| `PATCH /wardrobe-items/{id}` | Chỉ sửa dữ liệu | (nếu có) chỉnh sửa danh mục/sản phẩm | (chốt khi triển khai) |
| `PUT/PATCH` brand item/status | Gộp lỗi và cần duyệt | Phân biệt `failed` vs `draft`; chặn đổi khỏi `failed` | Cập nhật UI trạng thái |

> Sẽ bổ sung: request/response schema, mã lỗi, ví dụ JSON cụ thể.

---

## 4. Luồng realtime (SSE)

### 4.1 Đăng ký
- Wardrobe: `GET /api/v1/wardrobe-items/tasks/{taskId}/sse`
- Brand: `GET /api/v1/brand-portal/brands/{brandId}/items/tasks/{taskId}/sse`

### 4.2 Payload mỗi sự kiện (giữ nguyên cấu trúc hiện tại)

```json
{
  "itemId": "uuid",
  "status": "processing | completed | failed | needs_review",
  "total": 3,
  "index": 0,
  "data": { },
  "error": "chỉ có khi failed"
}
```

- `data` có đầy đủ thông tin tóm tắt khi `completed` / `needs_review`.
- `error` chỉ xuất hiện khi `failed`.

### 4.3 Kết nối lại / phát lại (replay)
- Khi client kết nối lại trong lúc phiên đang chạy, backend phát lại các sự kiện gần nhất (trong giới hạn lưu trữ).
- FE nên: giữ `taskId`, khi mở lại màn hình thì subscribe lại; xử lý idempotent theo `itemId` (không tạo trùng bản ghi UI).

### 4.4 Mã phiên (`taskId`)
- Có được từ response `batch-upload` (202) và (mới) từ response `retry-analysis`.
- Dùng để subscribe SSE.

---

## 5. Mã lỗi & thông báo (dự kiến)

> Sẽ chốt khi triển khai. FE cần phân biệt tối thiểu:
- Xác nhận khi món không ở trạng thái cần rà soát → từ chối.
- Xác nhận khi thiếu danh mục hợp lệ → từ chối, yêu cầu chọn danh mục.
- Retry khi trạng thái không cho phép (đang xử lý / đã dùng được) → từ chối.
- Brand đổi trạng thái khỏi `failed` → từ chối.

---

## 6. Checklist frontend

- [ ] Xử lý 4 trạng thái realtime: `processing`, `completed`, `failed`, `needs_review`.
- [ ] Màn hình rà soát: chọn danh mục (bắt buộc) + xác nhận.
- [ ] Nút "Thử lại": chỉ bật khi trạng thái cho phép; nhận `taskId` để subscribe.
- [ ] Điều hướng/nhãn trạng thái sản phẩm nhãn hàng (phân biệt `failed` vs `draft`).
- [ ] Subscribe lại + xử lý replay khi reconnect; khử trùng theo `itemId`.
- [ ] Cập nhật các call API theo bảng "trước → sau" ở mục 3.

---

## 7. Changelog

| Ngày | Thay đổi |
|------|----------|
| 2026-09-26 | Tạo khung tài liệu |
| 2026-09-26 | Bỏ cooldown/giới hạn retry; chỉ kiểm tra trạng thái cho phép retry |
