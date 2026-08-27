# Data Model: Pure Contract Refactor

**Feature**: `015-contract-purity-refactor`
**Ngày**: 2026-08-27 (cập nhật tránh overengineering)

> Feature này KHÔNG thêm entity/DB migration, KHÔNG thêm interface `IPure*`. Chỉ sửa trực tiếp contract usecase có sẵn cho thuần.

---

## 1. Thực thể liên quan (không đổi)

| Entity | Module | Mô tả |
|---|---|---|
| `WardrobeItem` | wardrobe | Item tủ đồ, có `FashionItemID` FK. |
| `FashionItem` | fashion | Item chuẩn hóa (AI). |
| `BrandItem` | brand | Item nhãn hàng. |
| `UserSubscription` | subscription | Gói subscription. |
| `Category` | fashion | Danh mục. |

Không có migration.

---

## 2. Sửa trực tiếp cho thuần (không tạo mới)

### 2.1. `WardrobeContractUseCase` + DTO `WardrobeItemRes` (wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go + dto/wardrobe.go)

- **Hiện trạng**: Struct `WardrobeContractUseCase` có 3 field `wardrobeRepo`, `userSubContract`, `fashionContract`. 7/15 method leaky do gọi 2 contract này để enrich. DTO `WardrobeItemRes` hiện chỉ có `FashionItem *FashionItemRes` (không có `FashionItemID` top-level).
- **Sửa DTO (D1 — Hiến pháp Code Style)**: Thêm field `FashionItemID *uuid.UUID \`json:"fashionItemId,omitempty"\`` vào `WardrobeItemRes` (`dto/wardrobe.go:54-67`) — **camelCase `fashionItemId`** (tuân Hiến pháp Code Style JSON). Cập nhật `mapper.MapToWardrobeItemRes` (`mapper/wardrobe.go:39-55`) để map `FashionItemID` từ `item.FashionItemID` (entity) vào DTO, kể cả khi `FashionItem == nil`.
- **Sửa Usecase**: Bỏ field `userSubContract` + `fashionContract` khỏi struct và constructor. 4 method alive leaky sửa trực tiếp (3 deadcode sẽ xóa ở §3):
  - `CopyItemToUser:69` — deadcode, xóa ở §3.
  - `VerifyItemsForPost:92` — deadcode, xóa ở §3.
  - `GetItemsByIDs:230` — deadcode, xóa ở §3.
  - `GetWardrobeItemsByUserID:164` — bỏ enrich fashion, trả `FashionItem == nil` + `FashionItemID` mới.
  - `GetActiveWardrobeItemsByUserID:198` — tương tự.
  - `GetFailedItemsForCleanup:262` — tương tự.
  - `GetStaleProcessingItems:299+309` — bỏ 2 call `GetStaleProcessingInfo` + `ListFashionItems`, chỉ filter theo wardrobe stale.
- **8 method pure giữ nguyên**: `UpdateItemStatus`, `GetUserFashionItemIDs`, `SetFashionItemID`, `DeleteWardrobeItem`, `MarkProcessingFailed/NeedsReview/CompleteProcessingSuccess`, `CountItemsByFashionIDs`.

### 2.2. `BrandItemUseCase` 5 method leaky (brand/application/usecase/item/usecase.go + helper.go)

- **Hiện trạng**: 5 method qua helper gọi `fashionContract` để enrich `FashionItem`.
- **Sửa trực tiếp** 5 method + helper cho thuần (bỏ `fashionContract`/`userBriefContract`):
  - `ListEligibleBrandItemsForStyling:87+157` — bỏ `userBriefContract.GetByID` + `fashionContract.ListFashionItems`.
  - `CheckBrandItemEligibility:185+227` — bỏ `userBriefContract` + `fashionContract.GetFashionItem`.
  - `GetBrandItemByIDDirect:570-579` → `helper mapToRes:21` bỏ `fashionContract.GetFashionItem`.
  - `GetBrandItemsByFashionItemIDs:581-597` → `helper mapSliceToBriefRes:33` bỏ `ListFashionItemsBrief`.
  - `GetBrandItemsByIDs:599-608` → `helper mapSliceToRes:54` bỏ `ListFashionItems`.
- **4 method pure giữ nguyên**: `CheckBrandFeatureAccess` (benefit), `ListActiveBrandItemFashionIDs`, `SetFashionItemID`, `UpdateBrandItemStatusDirect`.

### 2.3. `CategoryUseCase` / `SubscriptionUseCase` — KHÔNG sửa

- Verify method-level: `CategoryContract.GetCategory` chỉ `categoryRepo` → pure, không sửa. `DeleteCategory` có gọi `wardrobeContract` nhưng không qua contract nên không tính.
- `Subscription` 8 delegate (3 `UserSubscription`, 2 `Plan`, 3 `Quota`) đều pure → không sửa, không tạo `IUserSubscriptionPureUseCase`.

---

## 3. Contract Interface thu hẹp (xóa deadcode)

| Contract | File | Method xóa | Ghi chú |
|---|---|---|---|
| `IWardrobeContract` | `wardrobe/presentation/contract/wardrobe_contract.go` | `CopyItemToUser`, `VerifyItemsForPost`, `GetItemsByIDs`, `GetUserFashionItemIDs`, `MarkProcessingFailed`, `MarkProcessingNeedsReview`, `CompleteProcessingSuccess` (7) | 3 trong số đó trùng leaky (CopyItemToUser, VerifyItemsForPost, GetItemsByIDs) nên xóa luôn giải quyết cả deadcode + leaky |
| `IBrandContract` | `brand/presentation/contract/brand_contract.go` | `CheckBrandFeatureAccess`, `UpdateBrandItemStatus` (2) | Pure nhưng deadcode → xóa |
| `IUserSubscriptionContract` | `subscription/presentation/contract/subscription_contract.go` | `GetUserSubscription` (1) | Pure nhưng deadcode |
| `ISubscriptionPlanContract` | `subscription/presentation/contract/subscription_contract.go` | `GetDefaultSubscriptionPlanID`, `IsPremiumPlan` (2) | Pure nhưng deadcode, nếu rỗng xóa cả contract |
| `IFashionContract`/`ICategoryContract` | — | 0 | Đều alive, không xóa |

---

## 4. Dead Wiring

| Struct | File | Field xóa |
|---|---|---|
| `WardrobeItemUseCase` | `wardrobe/application/usecase/wardrobe/item/item_uc_read.go:41,57` | `aiService aiexecutor_contract.IAIOrchestrator` |

---

## 5. Caller phải tự enrich (nil risk)

Sau thuần hóa, `WardrobeItemRes.FashionItem` và `BrandItemRes.FashionItem` sẽ `nil`. 7 caller fashion phải tự gọi:

```go
items, _ := wardrobeContract.GetWardrobeItemsByUserID(ctx, userID) // pure, FashionItem == nil
ids := extractFashionIDs(items)
fashionItems, _ := fashionContract.ListFashionItems(ctx, ids)
byID := sliceutils.IndexBy(fashionItems, func(f *dto.FashionItemRes) uuid.UUID { return f.ID })
// stitch vào res trước khi trả
```

Nếu sót, UI thiếu ảnh/category/style.

---

## 6. Tài liệu tham khảo

- `docs/temp/detour-call-audit/02_wardrobe_fashion.md` — alive/deadcode
- Audit method-level 2026-08-27 — brand/wardrobe leaky chi tiết
