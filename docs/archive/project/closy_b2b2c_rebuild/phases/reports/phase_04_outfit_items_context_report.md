# Phase 04 - outfit_items dùng fashion_item_id và item_context

## Trạng thái

Đã hoàn thành phần code, migration và Swagger cho Phase 04 theo quyết định mới. Chưa chạy `migrate up`.

## Quyết định đã chốt

- Primary key mới của `outfit_items`: `(outfit_id, fashion_item_id, item_context)`.
- FE chỉ gửi `fashionItemId` trong payload item của outfit.
- FE không gửi `itemContext`.
- Backend tự detect context. Trong MVP hiện tại, nếu `fashionItemId` thuộc một `wardrobe_items` của user thì backend ghi `item_context = USER_WARDROBE`.
- `BRAND_ITEM` được giữ ở tầng schema/context để mở rộng khi module brand item có nguồn dữ liệu và eligibility check đầy đủ.

## Files changed

- `internal/shared/domain/entities/wardrobe_entities.go`
- `internal/shared/domain/constants/outfititemcontext/outfit_item_context.go`
- `internal/modules/wardrobe/application/dto/outfit.go`
- `internal/modules/wardrobe/application/usecase/outfit/outfit_uc.go`
- `internal/modules/wardrobe/application/mapper/outfit.go`
- `internal/modules/wardrobe/domain/repositories/interfaces.go`
- `internal/modules/wardrobe/infrastructure/persistence/outfit_repo.go`
- `internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`
- `api/swagger/docs.go`
- `api/swagger/swagger.json`
- `api/swagger/swagger.yaml`

## Migration added

- `migrations/20260628123858_outfit_items_fashion_item_context.sql`

Migration thực hiện:

- Thêm `outfit_items.fashion_item_id`.
- Thêm `outfit_items.item_context`.
- Backfill từ `outfit_items.item_id -> wardrobe_items.fashion_item_id`.
- Set `item_context = USER_WARDROBE` cho dữ liệu cũ.
- Validate không còn NULL và không trỏ thiếu `fashion_items`.
- Validate không có duplicate `(outfit_id, fashion_item_id, item_context)` trước khi đổi primary key.
- Drop constraint/FK cũ liên quan `item_id`.
- Tạo FK mới tới `fashion_items(id)`.
- Tạo composite key mới `(outfit_id, fashion_item_id, item_context)`.
- Drop cột `item_id`.

## API changed

- `SaveOutfitItemReq` bỏ `wardrobeItemId`.
- `SaveOutfitItemReq` bỏ input `itemContext`.
- `SaveOutfitItemReq` dùng required field `fashionItemId` với JSON camelCase đúng cho FE.
- `OutfitItemRes` trả:
  - `id`: hiện map bằng `fashionItemId` để giữ ID item ổn định cho canvas.
  - `fashionItemId`
  - `itemContext`
  - `wardrobeItem` khi context là `USER_WARDROBE`.

## Code behavior

- Save/Update outfit nhận `fashionItemId` từ FE.
- Backend resolve `fashionItemId` sang `wardrobe_items` thuộc user bằng `user_id + fashion_item_id`.
- Nếu resolve được user wardrobe item hợp lệ, backend ghi:
  - `fashion_item_id = input.fashionItemId`
  - `item_context = USER_WARDROBE`
- Nếu fashion item không thuộc wardrobe của user, backend trả lỗi invalid/forbidden trong MVP.
- `last_used_at` vẫn update trên `wardrobe_items.id` thật sau khi backend resolve.
- Outfit detail load metadata qua `fashion_items`.
- Với `USER_WARDROBE`, repository gắn đúng `wardrobe_items` theo `outfit.user_id + fashion_item_id` để tránh nhầm khi nhiều wardrobe wrapper cùng dùng một fashion item.

## Verification

- `go test ./...`: pass.
- `make swagger`: pass.
- `make build`: pass.

## Manual verification steps

1. Chạy migration trong môi trường DB test/staging.
2. Kiểm tra validation:
   - `SELECT COUNT(*) FROM outfit_items WHERE fashion_item_id IS NULL;`
   - `SELECT COUNT(*) FROM outfit_items WHERE item_context IS NULL;`
3. Tạo outfit bằng payload mới chỉ có `fashionItemId`.
4. Gọi detail outfit và xác nhận item trả về có `fashionItemId`, `itemContext = USER_WARDROBE`, và `wardrobeItem` đầy đủ metadata.

## Known limitations

- Chưa hỗ trợ save `BRAND_ITEM` từ public API trong Phase 04 vì chưa có nguồn brand item và eligibility check.
- Nếu dữ liệu cũ có duplicate `(outfit_id, fashion_item_id, USER_WARDROBE)`, migration sẽ fail rõ ràng để tránh mất item.
- Chưa chạy `make migration-up`.
