# Contracts: Pure Contract Refactor

**Feature**: `015-contract-purity-refactor`
**Ngày**: 2026-08-27 (cập nhật tránh overengineering)

> Không thay đổi API endpoint/DTO (NFR-002). Chỉ thu hẹp contract interface và thuần hóa impl.

---

## 1. Internal Contract sau khi xóa deadcode

### 1.1. `IWardrobeContract` (wardrobe)

**File**: `internal/modules/wardrobe/presentation/contract/wardrobe_contract.go`

**Method còn lại (alive, sau khi thuần hóa 4 method)**:
- `UpdateItemStatus`, `GetWardrobeItemsByUserID` (pure), `GetActiveWardrobeItemsByUserID` (pure), `SetFashionItemID`, `GetFailedItemsForCleanup` (pure), `DeleteWardrobeItem`, `GetStaleProcessingItems` (pure), `CountItemsByFashionIDs`

**Method đã xóa** (7): `CopyItemToUser`, `VerifyItemsForPost`, `GetItemsByIDs`, `GetUserFashionItemIDs`, `MarkProcessingFailed`, `MarkProcessingNeedsReview`, `CompleteProcessingSuccess` (3 trong số pure, 3 leaky nên xóa luôn).

### 1.2. `IBrandContract` (brand)

**Còn lại**: `ListEligibleBrandItemsForStyling` (sau thuần hóa), `CheckBrandItemEligibility` (sau thuần hóa), `ListActiveBrandItemFashionIDs`, `SetFashionItemID`, `GetBrandItemByID` (sau thuần hóa), `GetBrandItemsByFashionItemIDs` (sau thuần hóa), `GetBrandItemsByIDs` (sau thuần hóa)

**Đã xóa** (2): `CheckBrandFeatureAccess`, `UpdateBrandItemStatus`.

### 1.3. Subscription — không thuần hóa usecase

- `IUserSubscriptionContract` còn `GetUserSubscriptionOverview`, `GetUserSubscriptionOverviews`; xóa `GetUserSubscription` (deadcode).
- `ISubscriptionPlanContract` xóa `GetDefaultSubscriptionPlanID`, `IsPremiumPlan` (nếu rỗng xóa cả contract).
- Không tạo `IUserSubscriptionPureUseCase`.

### 1.4. Fashion — không xóa, không thuần hóa contract

- `IFashionContract` 10 method alive giữ nguyên.
- `CategoryContract` chỉ `GetCategory` alive và đã pure → không sửa. `DeleteCategory` có gọi `wardrobeContract` nhưng không qua contract nên không tính.

---

## 2. Sửa trực tiếp cho thuần

- `WardrobeContractUseCase` 4 method alive (`GetWardrobeItemsByUserID`, `GetActive...`, `GetFailed...`, `GetStale...`) bỏ `fashionContract`/`userSubContract` enrich, chỉ `wardrobeRepo` + mapper (FashionItem = nil).
- `BrandItemUseCase` 5 method bỏ `fashionContract` enrich qua `helper mapToRes`/`mapSliceTo*`.

---

## 3. Caller Enrich Pattern (bắt buộc để tránh nil)

```go
// Sau thuần hóa:
items, _ := wardrobeContract.GetWardrobeItemsByUserID(ctx, userID) // FashionItem == nil
ids := extractFashionIDs(items)
fashionItems, _ := fashionContract.ListFashionItems(ctx, ids)
byID := sliceutils.IndexBy(fashionItems, func(f *dto.FashionItemRes) uuid.UUID { return f.ID })
// stitch vào res trước khi trả
```

Nếu thiếu, UI thiếu ảnh/category/style → phải rà 7 caller fashion.

---

## 4. Doc mới

**File**: `docs/system-design/contract-purity-pattern.md` — định nghĩa Pure Contract Usecase, before/after, quy tắc caller enrich bằng `IndexBy`/`GroupBy`, link tới Constitution §II.

---

## 5. API Endpoint — không đổi

