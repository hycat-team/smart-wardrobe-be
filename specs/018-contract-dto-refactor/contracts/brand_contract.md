# Contract Specification: IBrandContract

**Feature**: `018-contract-dto-refactor`
**Interface Path**: `internal/modules/brand/presentation/contract/brand_contract.go`

---

## 1. Interface Definition

```go
package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type IBrandContract interface {
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemContractDTO, []*fashion_dto.FashionItemRes, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemContractDTO, *fashion_dto.FashionItemRes, error)
	ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	GetBrandItemByID(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemContractDTO, error)
	GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemContractDTO, error)
	GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemContractDTO, error)
}
```

---

## 2. Breaking Changes & Migration Details

| Phương thức | Kiểu cũ | Kiểu mới | Ghi chú di chuyển |
|:---|:---|:---|:---|
| `GetBrandItemsByFashionItemIDs` | `[]*dto.BrandItemBriefRes` | `[]*dto.BrandItemContractDTO` | Không còn chứa `FashionItem` lồng rỗng; Caller đã có sẵn `fashionItemID` |
| `GetBrandItemByID` | `*dto.BrandItemRes` | `*dto.BrandItemContractDTO` | Chỉ chứa thông tin sản phẩm nhãn hàng và `FashionItemID` |
| `GetBrandItemsByIDs` | `[]*dto.BrandItemRes` | `[]*dto.BrandItemContractDTO` | Dùng cho batch lookups giữa các module |
| `ListEligibleBrandItemsForStyling` | `([]*dto.BrandItemRes, ...)` | `([]*dto.BrandItemContractDTO, ...)` | Phục vụ styling/recommendation workflows |
| `CheckBrandItemEligibility` | `(bool, *dto.BrandItemRes, ...)` | `(bool, *dto.BrandItemContractDTO, ...)` | Kiểm tra quyền truy cập mẫu thử số |

---

## 3. Caller Stitching Pattern Example

```go
// Ví dụ tại wardrobe/application/usecase/wardrobe/item/item_uc_read.go
brandItems, err := uc.brandContract.GetBrandItemsByFashionItemIDs(ctx, fashionIDs)
if err != nil {
    return nil, err
}

brandMap := sliceutils.IndexBy(brandItems, func(b *dto.BrandItemContractDTO) uuid.UUID {
    return b.FashionItemID
})

// Khi map sang WardrobeItemRes cho HTTP API response:
if brand, ok := brandMap[*item.FashionItemID]; ok {
    res.BrandItem = &dto.BrandItemBriefRes{
        ID:          brand.ID,
        BrandID:     brand.BrandID,
        BrandName:   brand.BrandName,
        ProductCode: brand.ProductCode,
        ItemType:    string(brand.ItemType),
        Name:        brand.Name,
        Price:       brand.Price,
        Status:      string(brand.Status),
    }
}
```
