# Frontend Integration Guide — Xử lý trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Trạng thái**: Hoàn thiện (feature 023 đã triển khai).
**Đối tượng đọc**: Lập trình viên frontend (web/mobile).
**Mục tiêu**: FE chỉ cần đọc tài liệu này để triển khai, KHÔNG cần đọc code backend hay Swagger.

> Quy ước: mọi trường JSON dùng camelCase. Nội dung này mô tả hành vi API ở mức hợp đồng; chi tiết nội bộ backend không nằm trong phạm vi.

---

## 0. Tóm tắt nhanh (TL;DR)

- Một món đồ khi tải lên đi qua các trạng thái phân tích: `processing → completed | failed | needs_review`.
- Có thêm trạng thái realtime `processing` (mới) để FE hiển thị "đang xử lý".
- Món `needs_review` PHẢI được người dùng xử lý bằng cách chọn **category hợp lệ** rồi gọi lại `retry-analysis` kèm `categoryId` mới trở thành "sử dụng được". `needs_review` chỉ dành cho ảnh **đúng một món** nhưng AI chưa chắc category.
- Ảnh **không hợp lệ** (đa món/toàn thân/không phải trang phục) ⇒ trạng thái `failed` kèm mã lý do; **không** retry — FE hướng dẫn tải ảnh khác.
- Lý do (reason) là **mã** và chỉ nằm trong `data.fashionItem` (`reviewReason`/`processingErrorReason`); frontend tự đặt text hiển thị.
- Sản phẩm nhãn hàng có thêm trạng thái `failed`; đây là trạng thái cuối với thao tác thủ công (chỉ retry/hệ thống mới đưa ra khỏi `failed`).
- Thao tác retry chỉ khả dụng khi món/sản phẩm đang ở trạng thái lỗi hoặc cần rà soát; giữ nguyên category người dùng đã sửa. Với `needs_review`, retry **bắt buộc** kèm `categoryId`.
- API `confirm-review` đã **bị gỡ** (breaking): gọi sẽ nhận `404`; FE chuyển sang `retry-analysis` kèm danh mục.
- Kết nối realtime nếu đứt sẽ được phát lại các sự kiện gần nhất khi kết nối lại.

---

## 0.1 Delta 2026-10-01 — mã lý do `no_fashion_item_detected` + kiểm tra ảnh cho nhánh chọn danh mục

> **FE đã apply bản 2026-09-27 (mục 7 changelog). Chỉ áp dụng phần này — các phần còn lại của tài liệu KHÔNG đổi.**
> Nguồn chuẩn cho FE. Các file spec khác của feature 023 là bản ghi thiết kế tại thời điểm đóng feature nên **không** phản ánh delta này.

### Behavior changes cần nắm

| # | Thay đổi | Ảnh hưởng |
|---|---|---|
| **BC-1** | Ảnh không có trang phục nào (chuột laptop, điện thoại, sách, đồ ăn, ảnh trắng/mờ) giờ trả mã mới `no_fashion_item_detected`. Trước đây nhóm này bị báo nhầm thành `multiple_items_detected`. | Toàn bộ user |
| **BC-2** | Payload cũ dùng key `"error"` từ AI (còn sót trong prompt cache sau khi deploy) giờ được chuẩn hoá thành `no_fashion_item_detected` thay vì `analysis_temporary_error`. **Hệ quả: nhóm này mất nút "Thử lại".** Trước đây retry được nhưng vô hạn và luôn fail. | Thay đổi thuận — sửa bug retry vô hạn, nhưng FE phải bỏ nút Thử lại cho nhóm này |
| **BC-3** | **Nhánh "chọn danh mục → phân tích lại" KHÔNG còn luôn thành công.** Có thể trả `failed` với `multiple_items_detected` / `full_body_outfit_detected` / `no_fashion_item_detected`. | Luồng `needs_review` |
| **BC-4** | Thứ tự kiểm tra trong prompt đảo: "đa món / toàn thân" kiểm **trước** "không phải trang phục". Ảnh có vật lạ bên cạnh **một món trang phục** vẫn được phân loại như một món. | Ảnh bối cảnh lộn xộn |
| **BC-5** | Message 400 của `retry-analysis` đổi: *"không phải trang phục, nhiều món hoặc ảnh toàn thân"*. | FE hiển thị |

### Danh sách thay đổi trong tài liệu này

| Vị trí | Loại | Nội dung |
|---|---|---|
| Mục 0 (TL;DR, dòng "Ảnh **không hợp lệ**") | **SỬA** | Thêm "**không phải trang phục**" vào ngoặc |
| Mục 1.2 — bảng trạng thái, dòng `failed` | **SỬA** | Mô tả thêm "**không phải trang phục**" |
| Mục 2.3 — `#### failed`, dòng mô tả | **SỬA** | Thêm "**hoặc không phải trang phục**" |
| Mục 2.3 — `#### failed`, nhóm mã "ảnh không hợp lệ" | **SỬA** | Thêm `no_fashion_item_detected` vào nhóm hiện có (cùng hành vi: tải ảnh khác, KHÔNG hiện nút Thử lại) |
| Mục 2.5 — bảng mã lý do | **THÊM 1 dòng** | xem bảng dưới |
| Mục 2.5 — quy tắc gate retry | **THÊM** | Thêm `no_fashion_item_detected` vào danh sách retry bị chặn |
| Mục 5 — mã lỗi 400 | **SỬA** | Thêm mã vào danh sách retry bị từ chối |
| Mục 6 — checklist | **THÊM 1 mục** | Ca mới |
| Mục 7 — changelog | **THÊM 1 dòng** | Bản ghi thay đổi |

Tại các dòng sửa tại chỗ đã gắn comment `<!-- delta-2026-10-01: bổ sung -->` để FE diff dễ.

### Bảng mã lý do (thay thế mục 2.5)

| Mã | Loại | Status | Retry được? | Text FE nên hiển thị |
|---|---|---|---|---|
| `uncertain_category` | review | `needs_review` | Có (bắt buộc kèm `categoryId`) | "AI chưa chắc category - chọn category rồi gọi phân tích lại" |
| `multiple_items_detected` | error | `failed` | Không | "Ảnh có nhiều món - tải ảnh khác" |
| `full_body_outfit_detected` | error | `failed` | Không | "Ảnh toàn thân - tải ảnh cận một món" |
| **`no_fashion_item_detected`** | **error** | **`failed`** | **Không** | **"Ảnh không phải trang phục - hãy tải ảnh đúng món đồ"** |
| `analysis_temporary_error` | error | `failed` | Có | "Lỗi tạm thời - thử lại" |
| `auto_retry_exceeded` | error | `failed` | Có | "Đã thử nhiều lần - thử lại" |

### Mở rộng: nhánh chọn danh mục có thể thất bại (BC-3)

Luồng `needs_review` → chọn `categoryId` → `retry-analysis` **trước đây luôn thành công**, vì backend bỏ qua toàn bộ kiểm tra ảnh ở nhánh này. Nay backend kiểm tra lại:

```
chọn category → retry → AI kiểm tra ảnh
  ├─ ảnh hợp lệ (đúng 1 món)  → completed, ghi metadata như cũ
  └─ ảnh không dùng được       → failed + 1 trong 3 mã ảnh không hợp lệ
                                 → KHÔNG có nút "Thử lại"
                                 → FE hiển thị "tải ảnh khác"
```

**FE cần làm gì thêm cho luồng này:** sau khi bấm "Phân tích lại", **phải chờ kết quả qua SSE** thay vì điều hướng thẳng sang màn danh sách như đang làm. Nếu kết quả là `failed`, hiển thị hướng dẫn tải ảnh khác thay vì coi như xong.

**Vì sao không cho sửa tay:** `PATCH /wardrobe-items/:id` chỉ nhận `price` và `categoryId`. Các trường AI sinh ra (`color`, `style`, `material`, `pattern`, `fit`, `seasonality`, `description`) **không sửa được qua API** — lối thoát duy nhất là xoá món và tải lại ảnh. Vì vậy backend chọn báo lỗi sớm còn hơn đẩy một món sai vào tủ (món sai sẽ đi vào embedding và làm nhiễu gợi ý phối đồ về sau).

### Hạn chế còn lại

Nhánh `needs_review` chỉ vào được khi AI ở lần phân tích đầu đã nói "đúng một món". Với ảnh toàn thân bị AI nhầm là một món, người dùng có thể chọn nhầm danh mục và nhận metadata từ ảnh đó. Đây là hạn chế đã biết, chưa xử lý — việc này cần quyết định sản phẩm về luồng `needs_review` nên tách ticket riêng.

---

## 1. Tổng quan trạng thái

### 1.1 State machine (món đồ cá nhân — WardrobeItem)

```text
                batch-upload / retry-analysis
        (create)          (retry, kèm categoryId khi needs_review)
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
          │ retry           │ retry-analysis (bắt buộc categoryId)
          └──────► processing ◄──────┘
```

### 1.2 Trạng thái phân tích (`status` trong sự kiện realtime)

| `status` | Ý nghĩa | Client cần làm |
|----------|---------|----------------|
| `processing` | Món bắt đầu được xử lý | Hiển thị trạng thái "đang xử lý"; chờ sự kiện kết thúc |
| `completed` | Phân tích xong, dùng được | Render món bằng `data` (không cần gọi lại API) |
| `needs_review` | Ảnh **đúng 1 món** nhưng AI chưa chắc category (`uncertain_category`) | Điều hướng màn hình rà soát; chọn category rồi gọi `retry-analysis` kèm `categoryId` |
| `failed` | Ảnh **không hợp lệ** (đa món/toàn thân/không phải trang phục) hoặc lỗi tạm thời/hết retry <!-- delta-2026-10-01: bổ sung --> | Hiển thị `error` (mã lý do); với ảnh không hợp lệ ⇒ yêu cầu tải ảnh khác; với lỗi tạm thời ⇒ gợi ý "Thử lại" |

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

> Wardrobe `status` là số nguyên (`NeedsReview = 5`); Brand `status` là chuỗi (`"failed"`). FE nên xử lý theo tên/ý nghĩa trạng thái, không hardcode số.

---

## 2. Case nào sinh ra status nào & hành vi kỳ vọng

### 2.1 Ma trận case → status (WardrobeItem)

| Case (thao tác) | Status realtime xuất hiện | Status nghiệp vụ cuối (`data.status`) |
|-----------------|---------------------------|----------------------------------------|
| `batch-upload` (tải ảnh mới) | `processing` → **đúng một** trong `completed` / `needs_review` / `failed` | `inWardrobe` (0) / `needsReview` (5) / `failed` (4) |
| `retry-analysis` (món `failed`) | `processing` → **đúng một** trong `completed` / `needs_review` / `failed` | `inWardrobe` (0) / `needsReview` (5) / `failed` (4) |
| `retry-analysis` (món `needsReview`, **bắt buộc `categoryId`**) | `processing` → `completed` (danh mục cố định) hoặc `failed` | `inWardrobe` (0) / `failed` (4) |

> Mỗi món luôn nhận `processing` ngay khi bắt đầu; sau đó đúng MỘT trong `completed` / `needs_review` / `failed`. Không có event nào khác.

### 2.2 Ma trận case → status (BrandItem)

| Case (thao tác) | Status realtime xuất hiện | Status nghiệp vụ cuối (`data.status`) |
|-----------------|---------------------------|----------------------------------------|
| Tạo/upload sản phẩm | `processing` → **đúng một** trong `completed` / `needs_review` / `failed` | `active` / `draft` (+`reviewReason`) / `failed` (+`processingErrorReason`) |
| `retry-analysis` (`failed` hoặc `draft` có `reviewReason`) | `processing` → **đúng một** trong `completed` / `needs_review` / `failed` | `active` / `draft` / `failed` |
| `PATCH .../status`, `PUT .../items/{id}` | (không phát sinh) | giữ nguyên; bị từ chối nếu sản phẩm đang `failed` |

### 2.3 Hành vi kỳ vọng theo từng status realtime

#### `processing`
- Xuất hiện: khi món/sản phẩm bắt đầu (hoặc được đưa lại) hàng đợi phân tích.
- Payload: `status: "processing"`, `itemId`, `index`/`total`; `data`/`error` rỗng.
- FE: hiển thị "đang xử lý" cho đúng `itemId`; cập nhật tiến trình `index/total`; tắt nút "Thử lại".

#### `completed`
- Xuất hiện: AI phân tích thành công.
- Payload: `data` đầy đủ (món + metadata category/ảnh/màu/phong cách).
- FE: render trực tiếp bằng `data`, KHÔNG refetch. `data.status` = `inWardrobe` (wardrobe) / `active` (brand).

#### `needs_review`
- Xuất hiện: ảnh **đúng một món** nhưng AI chưa chắc category.
- Payload: `data` (có thể thiếu metadata); lý do **luôn** là `uncertain_category` (đọc tại `data.fashionItem.reviewReason`).
- FE:
  1. Điều hướng màn hình rà soát.
  2. Bắt buộc chọn **category hợp lệ** (thuộc nhóm có thể phối đồ).
  3. Wardrobe: gọi `retry-analysis` với body `{ "categoryId": "<uuid>" }` → nhận `taskId`, subscribe SSE; hệ thống chạy phân tích ở chế độ **danh mục cố định** (chỉ sinh metadata, giữ danh mục) → `data.status` = `inWardrobe`. Brand: `data.status` = `draft` cho tới khi có kết quả; nhân viên xử lý hoặc "Thử lại" kèm `categoryId`.
  4. Không gửi `categoryId` cho món `needsReview` ⇒ backend từ chối (400).

#### `failed`
- Xuất hiện: **Ảnh không hợp lệ** (đa món/toàn thân **hoặc không phải trang phục** <!-- delta-2026-10-01: bổ sung -->) HOẶC lỗi tạm thời / hết retry tự động.
- Payload: `error` là **mã lý do**; đọc thêm `data.fashionItem.processingErrorReason` (nếu có).
- FE theo mã lý do:
  - `multiple_items_detected`, `full_body_outfit_detected`, `no_fashion_item_detected` <!-- delta-2026-10-01: bổ sung --> ⇒ **ảnh không hợp lệ**: hiển thị "vui lòng tải ảnh khác"; **KHÔNG** hiện nút "Thử lại" (retry sẽ bị từ chối).
  - `analysis_temporary_error`, `auto_retry_exceeded` ⇒ hiện nút "Thử lại"; gọi `retry-analysis` (mục 3).
- Lưu ý: đang `processing` hoặc đã dùng được ⇒ không cho retry (backend từ chối nếu gọi sai).
- **Lưu ý BC-3 (2026-10-01):** món `needsReview` sau khi chọn danh mục và gọi `retry-analysis` **có thể** vẫn ra `failed` với 3 mã ảnh không hợp lệ trên. FE phải chờ kết quả qua SSE thay vì coi như xong ngay. Xem mục 0.1.

### 2.4 Hành vi kỳ vọng theo status nghiệp vụ (nhãn UI & hành động cho phép)

| Đối tượng | `data.status` | Nhãn UI | Hành động cho phép |
|-----------|---------------|---------|--------------------|
| WardrobeItem | `inWardrobe` (0) | Sử dụng được | Dùng để phối đồ; không retry |
| WardrobeItem | `processing` (3) | Đang xử lý | Chờ realtime; không retry |
| WardrobeItem | `failed` (4) | Lỗi | Nếu lý do là ảnh không hợp lệ ⇒ tải ảnh khác; nếu lỗi tạm thời ⇒ "Thử lại" |
| WardrobeItem | `needsReview` (5) | Cần rà soát | Chọn category + "Gửi phân tích lại" (retry-analysis kèm `categoryId`) |
| BrandItem | `active` | Đang hoạt động | — |
| BrandItem | `draft` + `fashionItem.reviewReason=uncertain_category` | Chờ duyệt | Cho "Thử lại" |
| BrandItem | `draft` (không lý do) | Nháp | Không retry |
| BrandItem | `failed` | Lỗi | Ảnh không hợp lệ ⇒ tạo sản phẩm mới; lỗi tạm thời ⇒ "Thử lại"; chặn đổi status thủ công |
| BrandItem | `archived` | Lưu trữ | — |

### 2.5 Bảng mã lý do (reason codes) & quy tắc gate

Lý do là **mã** (FE tự đặt text hiển thị), **chỉ nằm trong `data.fashionItem`** (`reviewReason` hoặc `processingErrorReason`). Không có ở cấp gốc của wardrobe/brand item.

| Mã | Nhóm | Trạng thái | Retry? | Gợi ý UI |
|----|------|-----------|--------|----------|
| `uncertain_category` | review | `needs_review` | ✅ (bắt buộc kèm `categoryId`) | "AI chưa chắc category — chọn category rồi gửi phân tích lại" |
| `multiple_items_detected` | error | `failed` | ❌ | "Ảnh có nhiều món — tải ảnh khác" |
| `full_body_outfit_detected` | error | `failed` | ❌ | "Ảnh toàn thân — tải ảnh cận một món" |
| `no_fashion_item_detected` <!-- delta-2026-10-01: bổ sung --> | error | `failed` | ❌ | "Ảnh không phải trang phục — hãy tải ảnh đúng món đồ" |
| `analysis_temporary_error` | error | `failed` | ✅ | "Lỗi tạm thời — thử lại" |
| `auto_retry_exceeded` | error | `failed` | ✅ | "Đã thử nhiều lần — thử lại" |

- **Retry-analysis** với món `needs_review` (`uncertain_category`) **bắt buộc** kèm `categoryId` hợp lệ; backend chạy lại AI ở chế độ danh mục cố định (không sinh lại `needs_review`). **Lưu ý BC-3: lần chạy đó vẫn có thể `failed` nếu ảnh thực sự không dùng được.**
- **Retry-analysis** bị **từ chối** với `multiple_items_detected`/`full_body_outfit_detected`/`no_fashion_item_detected` <!-- delta-2026-10-01: bổ sung --> (retry cùng ảnh vô nghĩa) → hướng tải ảnh khác.
- Mã lạ (ngoài bảng): hiển thị thông báo mặc định, không so khớp chuỗi.

---

## 3. API: mới & thay đổi hành vi (trước → sau)

> Mỗi API đổi hành vi ghi rõ "trước → sau"; response bọc `APIResponse { success, message, data }` (trừ SSE). Chi tiết schema đầy đủ xem `contracts/api-contracts.md`.

### 3.1 API mới

| API | Mô tả | Ai dùng |
|-----|-------|---------|
| `POST /brand-portal/brands/{brandId}/items/{itemId}/retry-analysis` | Thử lại phân tích cho sản phẩm nhãn hàng lỗi/cần rà soát (body tùy chọn `categoryId`) | Brand staff |

> API cũ `POST /wardrobe-items/{id}/confirm-review` đã **bị gỡ** — gọi sẽ nhận `404`. Xem mục 1.1 TL;DR.

### 3.2 API thay đổi hành vi

| API | Trước | Sau | Ảnh hưởng FE |
|-----|-------|-----|--------------|
| `POST /wardrobe-items/{id}/retry-analysis` | Không nhận body | Nhận body tùy chọn `{ "categoryId": "<uuid>" }`; **bắt buộc** khi món `needsReview`; trả kèm `taskId`; chỉ cho phép khi trạng thái lỗi/cần rà soát | Thêm ô chọn category cho luồng `needsReview`; cập nhật luồng retry |
| `PATCH /wardrobe-items/{id}` | Chỉ sửa dữ liệu | Cho phép cập nhật category/giá; category đã sửa được bảo toàn khi phân tích lại | Cho phép chọn category trước khi retry |
| `PUT/PATCH` brand item/status | Gộp lỗi và cần duyệt | Phân biệt `failed` vs `draft`; chặn đổi khỏi `failed` | Cập nhật UI trạng thái |
| `GET /wardrobe-items?categorySlug=...` | Category không có món ⇒ trả toàn bộ tủ đồ (bug) | Category không có món ⇒ trả rỗng (`data: []`, `total: 0`) | Bỏ workaround lọc phía client; hiển thị đúng trạng thái rỗng |
| `GET /wardrobe-items/catalog?categorySlug|query` | Category/từ khóa không khớp ⇒ trả toàn bộ catalog (bug) | Không khớp ⇒ trả rỗng | Hiển thị đúng "không tìm thấy kết quả" |

**Ví dụ — Wardrobe retry-analysis (món `needsReview`, bắt buộc category)**

Request: `POST /api/v1/wardrobe-items/{id}/retry-analysis`

```json
{ "categoryId": "uuid-danh-muc-da-chon" }
```

```json
{
  "success": true,
  "message": "Yêu cầu phân tích lại trang phục đã được gửi thành công",
  "data": {
    "id": "uuid",
    "status": 3,
    "isLocked": false,
    "taskId": "uuid",
    "fashionItem": { "id": "uuid", "category": { "id": "uuid", "name": "Áo", "slug": "ao" }, "imageUrl": "...", "reviewReason": null }
  }
}
```

> `status: 3` nghĩa là `processing`; kết quả cuối nhận qua SSE (`completed` → `inWardrobe`). Với món `failed` (lỗi tạm thời), body có thể bỏ trống.

**Ví dụ — Brand retry-analysis**

Request: `POST /api/v1/brand-portal/brands/{brandId}/items/{itemId}/retry-analysis`

```json
{ "categoryId": "uuid-danh-muc-da-chon" }
```

> Body tùy chọn; với sản phẩm `draft` đang chờ duyệt, `categoryId` là **bắt buộc**.

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
      "imageUrl": "...",
      "colorHex": "#000000",
      "reviewReason": "uncertain_category",
      "processingErrorReason": null
    }
  }
}
```

**Thay đổi casing của `fashionItem` (trước → sau)**: object `fashionItem` (contract DTO) chuyển từ **snake_case** sang **camelCase**:
`category_id→categoryId`, `image_url→imageUrl`, `color_hex→colorHex`, `color_hue→colorHue`, `color_saturation→colorSaturation`, `color_lightness→colorLightness`, `processing_error_reason→processingErrorReason`, `review_reason→reviewReason`, `created_at→createdAt`, `updated_at→updatedAt`.

> Reason **không còn** ở cấp gốc của brand/wardrobe item; FE đọc tại `data.fashionItem.reviewReason` / `data.fashionItem.processingErrorReason`.

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
  "error": "mã lý do (chỉ khi failed)"
}
```

- `data` có đầy đủ thông tin tóm tắt khi `completed` / `needs_review`.
- `error` chỉ xuất hiện khi `failed` và là **mã lý do** (ví dụ `analysis_temporary_error`, `multiple_items_detected`).

### 4.3 Kết nối lại / phát lại (replay)
- Khi client kết nối lại trong lúc phiên đang chạy, backend phát lại các sự kiện gần nhất (trong giới hạn lưu trữ).
- FE nên: giữ `taskId`, khi mở lại màn hình thì subscribe lại; xử lý idempotent theo `itemId` (không tạo trùng bản ghi UI).
- **Realtime là best-effort**: nếu kênh realtime gián đoạn, backend KHÔNG đảm bảo phát đủ sự kiện nhưng nghiệp vụ vẫn thành công. FE MUST có cơ chế dự phòng — khi không nhận được sự kiện (gợi ý: mở lại màn hình, hoặc không có sự kiện mới trong ~10 giây sau khi kết nối), gọi API danh sách để refetch trạng thái thay vì treo chờ.

### 4.4 Mã phiên (`taskId`)
- Có được từ response `batch-upload` (202) và (mới) từ response `retry-analysis`.
- Dùng để subscribe SSE.

---

## 5. Mã lỗi & thông báo

FE cần phân biệt tối thiểu (message tiếng Việt từ backend, HTTP code theo bảng):
- Retry món `needsReview` thiếu `categoryId` hoặc category không hợp lệ → 400 (từ chối, yêu cầu chọn category).
- Retry khi trạng thái không cho phép (đang xử lý / đã dùng được) → 400 (từ chối).
- Retry khi ảnh không hợp lệ (`multiple_items_detected`/`full_body_outfit_detected`/`no_fashion_item_detected` <!-- delta-2026-10-01: bổ sung -->) → 400 (từ chối; hướng tải ảnh khác).
- Brand đổi trạng thái khỏi `failed` → 403 (từ chối).
- Brand retry `draft` chờ duyệt thiếu `categoryId` → 400 (từ chối).
- Gọi API cũ `confirm-review` → 404 (đã gỡ).

---

## 6. Checklist frontend

- [ ] Xử lý 4 trạng thái realtime: `processing`, `completed`, `failed`, `needs_review`.
- [ ] Màn hình rà soát: chọn category (bắt buộc) + gửi `retry-analysis` kèm `categoryId` (chỉ cho `uncertain_category`).
- [ ] Nút "Thử lại": bật cho `analysis_temporary_error`/`auto_retry_exceeded`; với `uncertain_category` phải kèm `categoryId`; nhận `taskId` để subscribe.
- [ ] Ảnh không hợp lệ (`multiple_items_detected`/`full_body_outfit_detected`/`no_fashion_item_detected` <!-- delta-2026-10-01: bổ sung -->): ẩn nút "Thử lại", hiển thị hướng dẫn tải ảnh khác.
- [ ] `no_fashion_item_detected`: map sang text riêng ("Ảnh không phải trang phục — hãy tải ảnh đúng món đồ"), không dùng chung text của nhiều món. <!-- delta-2026-10-01: bổ sung -->
- [ ] Sau khi chọn danh mục + `retry-analysis`, **chờ kết quả SSE**; nếu ra `failed` thì hướng dẫn tải ảnh khác thay vì coi như xong. <!-- delta-2026-10-01: bổ sung (BC-3) -->
- [ ] Bỏ mọi call tới `confirm-review` (đã gỡ, trả 404).
- [ ] Đọc reason tại `data.fashionItem.reviewReason` / `data.fashionItem.processingErrorReason` (camelCase; không ở root).
- [ ] Điều hướng/nhãn trạng thái sản phẩm nhãn hàng (phân biệt `failed` vs `draft`).
- [ ] Subscribe lại + xử lý replay khi reconnect; khử trùng theo `itemId`.
- [ ] Cập nhật các call API theo bảng "trước → sau" ở mục 3 (gồm casing `fashionItem`).

---

## 7. Changelog

| Ngày | Thay đổi |
|------|----------|
| 2026-09-26 | Tạo khung tài liệu |
| 2026-09-26 | Bỏ cooldown/giới hạn retry; chỉ kiểm tra trạng thái cho phép retry |
| 2026-09-27 | Dùng thuật ngữ domain `category` (không dịch sang tiếng Việt) |
| 2026-09-27 | Làm rõ mục 2: ma trận case → status và hành vi kỳ vọng theo từng status |
| 2026-09-27 | G10: `needs_review` chỉ còn `uncertain_category`; ảnh không hợp lệ ⇒ `failed` + mã lý do (không confirm/retry); bảng mã lý do + quy tắc gate; reason chỉ nested trong `fashionItem` (camelCase); SSE `error` là mã; brand exhausted ⇒ `failed` |
| 2026-09-27 | G11: bỏ `confirm-review` (breaking, 404); `needs_review` xử lý bằng `retry-analysis` kèm `categoryId` (bắt buộc, danh mục cố định); cập nhật state machine, mã lý do, ví dụ, mã lỗi |
| 2026-10-01 | Delta (mục 0.1): thêm mã lý do `no_fashion_item_detected` cho ảnh không phải trang phục; sửa bug `error` field không bao giờ được đọc khiến nhóm này bị báo nhầm là nhiều món; kiểm tra ảnh cho cả nhánh fixed-category nên `needs_review` → chọn danh mục → retry **có thể** `failed` (BC-3); đảo thứ tự rule trong prompt (BC-4) |
