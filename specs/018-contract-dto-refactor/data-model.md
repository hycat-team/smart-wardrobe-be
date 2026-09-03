# Data Model Specification: Contract DTOs

**Feature**: `018-contract-dto-refactor`
**Date**: 2026-09-03
**Status**: Draft

---

## 1. Wardrobe Module Contract DTOs

**File**: `internal/modules/wardrobe/application/dto/contract_dto.go`
**Package**: `dto`

```go
package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

// WardrobeItemContractDTO represents the pure data container of a wardrobe item
// for inter-module communication via IWardrobeContract.
// It contains ONLY properties owned by the Wardrobe domain plus foreign key references.
// Cross-module entity pointers (e.g. FashionItem, BrandItem) MUST NEVER be placed here.
type WardrobeItemContractDTO struct {
	ID            uuid.UUID                         `json:"id"`
	UserID        uuid.UUID                         `json:"userId"`
	Price         *float64                          `json:"price,omitempty"`
	Status        wardrobestatus.WardrobeItemStatus `json:"status"`
	IsLocked      bool                              `json:"isLocked"`
	ItemContext   string                            `json:"itemContext,omitempty"`
	FashionItemID *uuid.UUID                        `json:"fashionItemId,omitempty"`
	BrandItemID   *uuid.UUID                        `json:"brandItemId,omitempty"`
	CreatedAt     time.Time                         `json:"createdAt"`
	LastUsedAt    *time.Time                        `json:"lastUsedAt,omitempty"`
	IsDeleted     bool                              `json:"isDeleted"`
	TaskID        *uuid.UUID                        `json:"taskId,omitempty"`
}
```

### Các ràng buộc:
- `FashionItemID`: Con trỏ `*uuid.UUID`. Bắt buộc kiểm tra nil-guard khi dereference.
- Không chứa `FashionItem *FashionItemRes` hoặc `BrandItem *BrandItemBriefRes`.

---

## 2. Brand Module Contract DTOs

**File**: `internal/modules/brand/application/dto/contract_dto.go`
**Package**: `dto`

```go
package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemtype"

	"github.com/google/uuid"
)

// BrandItemContractDTO represents the pure data container of a brand item
// for inter-module communication via IBrandContract.
// It contains ONLY properties owned by the Brand domain plus foreign key references.
// Cross-module entity pointers (e.g. FashionItem) MUST NEVER be placed here.
type BrandItemContractDTO struct {
	ID            uuid.UUID                       `json:"id"`
	BrandID       uuid.UUID                       `json:"brandId"`
	BrandName     string                          `json:"brandName,omitempty"`
	FashionItemID uuid.UUID                       `json:"fashionItemId"`
	ProductCode   *string                         `json:"productCode,omitempty"`
	Name          string                          `json:"name"`
	Description   *string                         `json:"description,omitempty"`
	Price         *float64                        `json:"price,omitempty"`
	ItemType      branditemtype.BrandItemType     `json:"itemType"`
	Status        branditemstatus.BrandItemStatus `json:"status"`
	CreatedAt     time.Time                       `json:"createdAt"`
	UpdatedAt     time.Time                       `json:"updatedAt"`
	TaskID        *uuid.UUID                      `json:"taskId,omitempty"`
}
```

### Các ràng buộc:
- `FashionItemID`: Giá trị `uuid.UUID` đại diện cho liên kết sang thực thể Fashion.
- Không chứa `FashionItem *fashion_dto.FashionItemRes` hay `*fashion_dto.FashionItemBriefRes`.

---

## 3. Other Modules Contract DTOs (Phase 2)

### 3.1 Identity Module
**File**: `internal/modules/identity/application/dto/contract_dto.go`

```go
package dto

import "github.com/google/uuid"

// UserBriefContractDTO represents the minimal user profile for cross-module identity queries.
type UserBriefContractDTO struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	FullName  string    `json:"fullName"`
	AvatarUrl *string   `json:"avatarUrl,omitempty"`
	Role      string    `json:"role"`
}
```

### 3.2 Subscription Module
**File**: `internal/modules/subscription/application/dto/contract_dto.go`

Chứa `UserSubscriptionOverviewContractDTO` và `UserQuotaContractDTO` (di chuyển từ `subscription_dto.go`).

### 3.3 Fashion Module
**File**: `internal/modules/fashion/application/dto/contract_dto.go`

Chứa `FashionItemContractDTO`, `FashionItemBriefContractDTO`, `FashionProcessingInfoContractDTO`.

### 3.4 Notification & Admin Module
- `notification/application/dto/contract_dto.go`: `SendEmailInput`
- `admin/application/dto/contract_dto.go`: `CreateAuditLogReq`

---

## 4. Mapper Functions Cần Bổ Sung

1. **Module Wardrobe** (`internal/modules/wardrobe/application/mapper/wardrobe.go`):
   ```go
   func MapToWardrobeItemContractDTO(item *entities.WardrobeItem) *dto.WardrobeItemContractDTO
   func MapToWardrobeItemContractDTOList(items []*entities.WardrobeItem) []*dto.WardrobeItemContractDTO
   ```
2. **Module Brand** (`internal/modules/brand/application/mapper/brand_item_mapper.go`):
   ```go
   func MapToBrandItemContractDTO(item *entities.BrandItem, brandName string) *dto.BrandItemContractDTO
   func MapToBrandItemContractDTOList(items []*entities.BrandItem, brandNameMap map[uuid.UUID]string) []*dto.BrandItemContractDTO
   ```
