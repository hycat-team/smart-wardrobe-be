# Phase 06 - Brand Items and Sample Feedback

## Mục tiêu

Thêm `brand_items` để brand quản lý sản phẩm thật và sample thử nghiệm, đồng thời thêm feedback/vote cho sample.

> [!IMPORTANT]
> **Tái cấu trúc AI Vision và Fashion Item (Fashion Module Refactor)**:
> 1. Toàn bộ thực thể `FashionItem` và logic xử lý AI Vision phân tích hình ảnh (async metadata extraction) được chuyển giao hoàn toàn sang module `fashion` quản lý.
> 2. Các module `wardrobe` và `brand` khi tạo item mới sẽ không tự thao tác trực tiếp với repo của `FashionItem`, mà phải gọi thông qua Service Contract của module `fashion`.
> 3. Khi một item mới được tạo, module `wardrobe` và `brand` chịu trách nhiệm push event tương ứng vào RabbitMQ (sử dụng topic mới thay thế hoàn toàn topic cũ: `fashion.event.analyze_item`). Module `fashion` sẽ chạy worker để lắng nghe queue tương ứng, gọi AI Vision (OpenAI) để phân tích ảnh và sinh metadata tự động.
> 4. Về các API sửa thủ công (nhập tay metadata): Cả `wardrobe` (dành cho tủ đồ cá nhân) và `brand` (dành cho brand portal) sẽ tự định nghĩa/expose API của chính mình, sau đó gọi ngầm sang Service Contract của `fashion` module để thực hiện cập nhật `FashionItem`.
> 5. Giữ nguyên toàn bộ database schema dùng chung (không migrate các bảng `fashion_items` sang schema riêng biệt), chỉ thay đổi tầng code quản lý (Repositories/UseCases) quy về module `fashion`.

## Không làm trong phase này

```text
- Không tạo digital_sample_variants.
- Không tạo sample_outfit_trials.
- Không tạo sample_trial_items.
- Không tạo brand_orders.
- Không tự động thêm brand product vào wardrobe khi mua vì chưa có order flow.
- Không tích hợp AI recommendation trong phase này; phase 07 làm.
```

## Schema target

### brand_items

```text
id UUID PK
brand_id UUID FK brands(id)
fashion_item_id UUID FK fashion_items(id)
product_code VARCHAR(100) NULL
name VARCHAR(255)
description TEXT NULL
price DECIMAL(12,2) NULL
item_type VARCHAR(50)
status VARCHAR(50)
created_at
updated_at
```

Item type:

```text
PRODUCT
SAMPLE
```

Status:

```text
DRAFT
ACTIVE
ARCHIVED
```

Recommended constraints:

```text
unique(fashion_item_id) if one fashion item can only be wrapped by one brand item.
unique(brand_id, product_code) where product_code is not null.
```

Do not create global unique on product_code alone.

### digital_sample_responses

```text
id UUID PK
brand_item_id UUID FK brand_items(id)
user_id UUID FK users(id)
outfit_id UUID FK outfits(id) NULL
vote_type VARCHAR(50) NULL
rating INT NULL
feedback_text TEXT NULL
created_at TIMESTAMP
```

Vote type:

```text
LIKE
DISLIKE
WOULD_BUY
NOT_INTERESTED
```

> [!NOTE]
> Không giới hạn unique constraint trên (user_id, brand_item_id) để người dùng có thể gửi phản hồi nhiều lần cho cùng một mẫu thử (SAMPLE).

## Rules

User-facing chat/feedback:
- User must be ACTIVE.
- Brand item must be ACTIVE.
- brand_item_id must reference brand_items.item_type = SAMPLE.

Postgres cannot enforce conditional FK on item_type directly. Enforce in usecase/repository.

## Brand Portal APIs

```text
GET /api/v1/brand-portal/brands/:brandId/brand-items
POST /api/v1/brand-portal/brands/:brandId/brand-items
PATCH /api/v1/brand-portal/brand-items/:itemId
```

Permissions:

```text
OWNER/MANAGER/MARKETER can manage brand_items.
SUPPORT_STAFF cannot manage brand_items by default.
```

Create brand item flow:

```text
1. Check brand permission.
2. Validate item_type PRODUCT/SAMPLE.
3. Gọi Fashion Module Contract để tạo fashion_item cơ bản (chứa hình ảnh và danh mục).
4. Tạo bản ghi brand_item liên kết với fashion_item_id vừa tạo.
5. Push event phân tích ảnh lên RabbitMQ topic `fashion.event.analyze_item` để Fashion Module tự động chạy AI Vision bất đồng bộ phân tích ảnh và hoàn thiện metadata.
```

## User-facing sample response API

```text
POST /api/v1/brand-items/:itemId/responses
```

Body:

```json
{
  "vote_type": "LIKE",
  "rating": 4,
  "feedback_text": "Looks good",
  "outfit_id": "nullable uuid"
}
```

Rules:
- User must be ACTIVE.
- itemId is brand_items.id, not fashion_items.id.
- brand_items.item_type must be SAMPLE.
- brand_items.status must be ACTIVE.
- If outfit_id provided, outfit must belong to user and contain this brand item's fashion_item_id with item_context BRAND_ITEM.

## Brand item DTO for fashion

For Phase 07, prepare DTO:
- brand_item_id
- brand_id
- fashion_item_id
- item_type PRODUCT|SAMPLE
- name
- description
- price
- metadata from fashion_items (được lấy từ fashion contract)
- feature requirements if needed

## Tests

- Brand staff creates PRODUCT brand item -> fashion_item (via fashion) + brand_item created.
- Brand staff creates SAMPLE -> fashion_item (via fashion) + brand_item created.
- SAMPLE does not create wardrobe_item.
- User cannot response to PRODUCT via sample response API.
- User response to SAMPLE saved successfully.
- User can send multiple responses to the same SAMPLE.
- User cannot attach outfit_id not owned by them.
- Brand A cannot update Brand B item.

## Acceptance checklist

- [ ] brand_items table exists.
- [ ] digital_sample_responses table exists.
- [ ] Brand item create wraps fashion_item using Fashion Module Contract.
- [ ] Sample feedback works only for SAMPLE.
- [ ] Gửi sự kiện thành công lên RabbitMQ qua topic `fashion.event.analyze_item` khi tạo item mới.
- [ ] No sample trial tables.
- [ ] No brand order flow.
