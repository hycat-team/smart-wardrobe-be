# Contract: Swagger Module Mapping (Đợt 2)

**Branch**: `010-module-ownership-refactor` | **Date**: 2026-08-01

## Mục đích

Sinh 6 spec swagger per-module từ 1 master spec (FR-013) và bật dropdown module trong Swagger UI (FR-014). Theo Q1 Option B: giữ tag nhóm, mỗi tag thuộc đúng 1 module.

## Mapping tag → module (sau realign)

| Module | Tags |
|---|---|
| identity | `Auth`, `Me`, `Admin Identity` |
| subscription | `Subscription`, `Billing` |
| wardrobe | `Wardrobe`, `Outfits`, `Category`, `Admin Wardrobe` |
| dashboard | `Dashboard Analytics`, `Dashboard User`, `Dashboard Brand`, `Dashboard Admin` |
| brand | `Brand`, `Brand Portal`, `Brand Chat`, `Brand Loyalty`, `Brand Item`, `Brand Member`, `Brand Benefit`, `Admin Brand` |
| fashion | `Wardrobe AI` |

## Thay đổi annotation cần thực hiện

1. `@Tags Admin` trong handler **identity** (admin users) → `@Tags Admin Identity`
2. `@Tags Admin` trong handler **wardrobe** (admin wardrobe-items, categories) → `@Tags Admin Wardrobe`
3. `@Tags Admin` trong handler **brand** (admin brands) → `@Tags Admin Brand`

Các tag còn lại đã thuộc đúng 1 module → không đổi.

## File mapping (input cho script split)

Bản **runtime** đặt tại `api/swagger/swagger-module-mapping.json` (copy từ `contracts/swagger-module-mapping.json` trong specs — file specs này là tài liệu tham chiếu, không phải nguồn runtime). Script `internal/tools/swagger-split` đọc `api/swagger/swagger-module-mapping.json`.

```json
{
  "identity": ["Auth", "Me", "Admin Identity"],
  "subscription": ["Subscription", "Billing"],
  "wardrobe": ["Wardrobe", "Outfits", "Category", "Admin Wardrobe"],
  "dashboard": ["Dashboard Analytics", "Dashboard User", "Dashboard Brand", "Dashboard Admin"],
  "brand": ["Brand", "Brand Portal", "Brand Chat", "Brand Loyalty", "Brand Item", "Brand Member", "Brand Benefit", "Admin Brand"],
  "fashion": ["Wardrobe AI"]
}
```

## Thuật toán split (Go script `internal/tools/swagger-split`)

1. Đọc master `api/swagger/swagger.json`.
2. Đọc mapping (JSON).
3. Với mỗi module: lọc `paths` — operation thuộc module nếu **mọi tag** của operation nằm trong danh sách module đó.
4. Giữ nguyên `info`, `securityDefinitions`, `definitions` cần thiết.
5. Ghi `api/swagger/modules/<name>/swagger.json`.

## Kiểm tra toàn vẹn

- Tổng paths của 6 spec module == tổng paths master (FR-012 acceptance 2, spec Edge Cases "script split bỏ sót path").
- Script fail nếu có operation không map được vào module nào (tag không có trong mapping).

## Dropdown Swagger UI

Sửa `api/swagger/index.html` — mở rộng mảng `urls` trong `SwaggerUIBundle`:

```js
urls: [
  { url: "/api-docs/swagger.json", name: "v1 - Tất cả" },
  { url: "/api-docs/modules/identity/swagger.json", name: "identity" },
  { url: "/api-docs/modules/subscription/swagger.json", name: "subscription" },
  { url: "/api-docs/modules/wardrobe/swagger.json", name: "wardrobe" },
  { url: "/api-docs/modules/dashboard/swagger.json", name: "dashboard" },
  { url: "/api-docs/modules/brand/swagger.json", name: "brand" },
  { url: "/api-docs/modules/fashion/swagger.json", name: "fashion" },
]
```

- `router.go:52` dùng `Static("/api-docs", "./api/swagger")` → thư mục `modules/` được serve tự động, không cần đổi route.
- SwaggerLockMiddleware (production) áp dụng cho cả thư mục con.

## Makefile

- `make swagger`: thêm bước chạy `go run ./internal/tools/swagger-split` sau `swag init`.
- `make generate-check`: `git diff --exit-code -- $(WIRE_PATH) $(SWAG_OUTPUT_DIR)` đã bao phủ `api/swagger` (gồm `modules/`) → tự động kiểm tra spec per-module có thay đổi.

## Ràng buộc

- Không đổi master spec (`swagger.json`) — vẫn là nguồn sự thật.
- Đổi tên tag hiển thị (`Admin` → `Admin Identity`...) là thay đổi docs, cần thông báo trong PR/changelog.
