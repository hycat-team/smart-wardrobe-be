# Phase 03 - Tách fashion_items khỏi wardrobe_items

## Trạng thái

Đã hoàn thành phần code và migration chuẩn bị cho Phase 03 theo hướng strong migration. Chưa chạy `migrate up`.

## Thay đổi chính

- Tạo migration `20260628073348_create_fashion_items_and_backfill.sql`.
- Thêm bảng `fashion_items` để giữ metadata thời trang dùng chung: category, ảnh, màu sắc, style, material, pattern, fit, seasonality, description, embedding và processing fields.
- Backfill `fashion_items.id = wardrobe_items.id` cho dữ liệu hiện tại.
- Thêm `wardrobe_items.fashion_item_id`, backfill từ `wardrobe_items.id`, đặt NOT NULL và FK tới `fashion_items`.
- Đổi mapping GORM của `WardrobeItem.Price` sang cột `purchase_price`; migration rename `price` thành `purchase_price`.
- Drop toàn bộ metadata thời trang khỏi `wardrobe_items` ngay trong Phase 03 sau khi backfill sang `fashion_items`.
- Sau Phase 03, `wardrobe_items` chỉ còn wrapper fields: `id`, `user_id`, `fashion_item_id`, `purchase_price`, `status`, `item_type`, `last_used_at`, `is_deleted`, `created_at`, `updated_at`.

## Thay đổi code

- Thêm entity `FashionItem` và quan hệ `WardrobeItem.FashionItem`.
- Rút gọn entity `WardrobeItem` thành wrapper, không còn field metadata thời trang.
- Repository wardrobe tự tạo `fashion_items` khi tạo item mới.
- Các query lọc/tìm kiếm theo category, metadata, lexical search và vector search đã chuyển sang join qua `fashion_items`.
- Các luồng clone/copy/catalog-init dùng chung `fashion_item_id` thay vì tạo metadata thời trang mới.
- Worker processing/retry/success/fail cập nhật processing fields trên `fashion_items`, còn `wardrobe_items` giữ status/ownership.
- Search index thêm `fashion_item_id` và tiếp tục denormalize metadata để giữ khả năng search hiện tại.
- Category repo đếm và reassign system catalog qua `fashion_items.category_id`.
- Mapper, AI chat, outfit recommendation, fallback ranking và community compile path đã chuyển sang đọc metadata từ `FashionItem`.

## Kiểm tra

- `go test ./...`: pass.
- `make build`: pass.

## Ghi chú

- Chưa chạy `make migration-up` hoặc bất kỳ lệnh migrate up nào.
- Chưa chạy `make swagger` vì Phase 03 không thay đổi contract public của API response.
- `outfit_items` vẫn giữ composite key và tiếp tục reference `wardrobe_items.id`.
