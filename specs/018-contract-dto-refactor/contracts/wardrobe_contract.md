# Contract Specification: IWardrobeContract

**Feature**: `018-contract-dto-refactor`
**Interface Path**: `internal/modules/wardrobe/presentation/contract/wardrobe_contract.go`

---

## 1. Interface Definition

```go
package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

type IWardrobeContract interface {
	GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemContractDTO, error)
	GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemContractDTO, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemContractDTO, error)
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemContractDTO, error)
	CountItemsByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (int64, error)
}
```

---

## 2. Breaking Changes & Migration Details

| Phương thức | Kiểu cũ | Kiểu mới | Ghi chú di chuyển |
|:---|:---|:---|:---|
| `GetWardrobeItemsByUserID` | `[]*dto.WardrobeItemRes` | `[]*dto.WardrobeItemContractDTO` | Caller lấy `FashionItemID` để tự enrich qua `fashionItemRepo` hoặc `fashionContract` |
| `GetActiveWardrobeItemsByUserID` | `[]*dto.WardrobeItemRes` | `[]*dto.WardrobeItemContractDTO` | Phục vụ AI Chat / Recommendation workflows |
| `GetFailedItemsForCleanup` | `[]*dto.WardrobeItemRes` | `[]*dto.WardrobeItemContractDTO` | Worker kiểm tra `item.FashionItemID != nil` |
| `GetStaleProcessingItems` | `[]*dto.WardrobeItemRes` | `[]*dto.WardrobeItemContractDTO` | Worker kiểm tra `item.FashionItemID != nil` |
| `CountItemsByFashionIDs` | `(int64, error)` | `(int64, error)` | Không đổi |

---

## 3. Caller Stitching Pattern Example

```go
// Ví dụ tại fashion/application/usecase/ai/recommendation/usecase.go
items, err := uc.wardrobeContract.GetWardrobeItemsByUserID(ctx, userID)
if err != nil {
    return nil, err
}

fashionIDs := make([]uuid.UUID, 0, len(items))
for _, item := range items {
    if item.FashionItemID != nil {
        fashionIDs = append(fashionIDs, *item.FashionItemID)
    }
}
fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)

fashionEntities, err := uc.fashionItemRepo.GetByIDs(ctx, fashionIDs)
if err != nil {
    return nil, err
}

fashionMap := sliceutils.IndexBy(fashionEntities, func(f *entities.FashionItem) uuid.UUID {
    return f.ID
})
```
