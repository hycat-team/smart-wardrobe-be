# PHASE 6: Thay thế Entity bằng DTO trong Fashion Ranking
**Thời điểm thực hiện:** Sau khi Phase 5 hoàn thành và hệ thống ổn định với kiến trúc mới.

## Mục tiêu

Thay thế hoàn toàn việc import trực tiếp `wardrobe/domain/entities` và `brand/domain/entities` trong module `fashion` bằng DTO từ `wardrobe/application/dto/` và `brand/application/dto/`.

**Chiến lược:** Mở rộng DTO hiện có (`WardrobeItemRes`, `BrandItemRes`) thay vì tạo DTO mới.

---

## Phân tích field usage (đã verify với code thực tế)

### WardrobeItem — tất cả fields được dùng trong fashion ranking

| Field | Kiểu | Dùng trong | Ghi chú |
|-------|------|------------|---------|
| `ID` | `uuid.UUID` | Mọi file | Map key, so sánh, sort |
| `FashionItemID` | `*uuid.UUID` | Mọi file | Lookup `fashionItemMap` để get `FashionItemRes` |
| `LastUsedAt` | `*time.Time` | `scoring.go:228-241` | Tính điểm recency; `nil` = new-item-bonus |
| `Status` | `WardrobeItemStatus` | `candidate_helpers.go:72` | Filter `== InWardrobe` |
| `IsDeleted` | `bool` | `candidate_helpers.go:72` | Filter `!IsDeleted` |
| `UserID`, `Price`, `ItemType` | — | **Không dùng** | |

### BrandItem — tất cả fields được dùng trong fashion ranking

| Field | Kiểu | Dùng trong | Ghi chú |
|-------|------|------------|---------|
| `ID` | `uuid.UUID` | `mapper/recommendation.go:254` | Map vào response DTO |
| `BrandID` | `uuid.UUID` | `mapper/recommendation.go:255` | Map vào response DTO |
| `Brand.Name` | `string` (qua relation) | `mapper/recommendation.go:251` | **Cần flatten** thành `BrandName` |
| `Name` | `string` | `mapper/recommendation.go:258` | Map vào response DTO |
| `ItemType` | `BrandItemType` | `mapper/recommendation.go:257` | Cast `string(...)` |
| `Price` | `*float64` | `mapper/recommendation.go:259` | Map vào response DTO |
| `FashionItemID`, `ProductCode`, `Description`, `Status` | — | **Không dùng** | |

---

## Kế hoạch thực hiện

### Bước 6.1: Mở rộng `WardrobeItemRes` trong `wardrobe/application/dto/wardrobe.go`

Thêm 2 field vào struct `WardrobeItemRes`:

```go
type WardrobeItemRes struct {
    // ... các field hiện tại (ID, UserID, Category, Price, Status, IsLocked,
    //     ItemContext, BrandItem, FashionItem, CreatedAt) ...

    LastUsedAt *time.Time   `json:"lastUsedAt,omitempty"`  // THÊM
    IsDeleted  bool         `json:"isDeleted"`              // THÊM
}
```

**Lưu ý:** `Status` đã có trong `WardrobeItemRes` với type `wardrobestatus.WardrobeItemStatus`. Việc fashion module import `wardrobestatus` để so sánh `item.Status == wardrobestatus.InWardrobe` là **chấp nhận được** (constants cross-module import cho phép).

**Lưu ý về `FashionItemID`:** Entity dùng `FashionItemID *uuid.UUID` (nullable), nhưng `WardrobeItemRes` có `FashionItem *FashionItemRes` với field `FashionItem.ID`. Pattern access mới:
- Entity cũ: `item.FashionItemID != nil { id = *item.FashionItemID }`
- DTO mới: `item.FashionItem != nil { id = item.FashionItem.ID }`

### Bước 6.2: Mở rộng `BrandItemRes` trong `brand/application/dto/brand_item.go`

Thêm 1 field vào struct `BrandItemRes`:

```go
type BrandItemRes struct {
    // ... các field hiện tại (ID, BrandID, FashionItemID, ProductCode, Name,
    //     Description, Price, ItemType, Status, FashionItem, CreatedAt, UpdatedAt) ...

    BrandName string `json:"brandName,omitempty"`  // THÊM — flatten từ Brand.Name
}
```

**Kiểm tra type của `ItemType`:** Nếu là `branditemtype.BrandItemType` (const), giữ nguyên; nếu là `string`, giữ nguyên. Fashion code dùng `string(cand.BrandItem.ItemType)` khi mapping.

### Bước 6.3: Cập nhật `types` package trong fashion

Sửa file `internal/modules/fashion/application/usecase/ai/recommendation/types/types.go`:

1. **Thay thế import:**
   - `wardrobe_entities "smart-wardrobe-be/internal/modules/wardrobe/domain/entities"` → `wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"`
   - `brand_entities "smart-wardrobe-be/internal/modules/brand/domain/entities"` → `brandDto "smart-wardrobe-be/internal/modules/brand/application/dto"`

2. **Cập nhật 3 struct:**

```go
type CandidateForPrompt struct {
    Item        *wardrobeDto.WardrobeItemRes       // thay vì *wardrobe_entities.WardrobeItem
    Tags        []string
    ItemContext outfititemcontext.OutfitItemContext
    BrandItem   *brandDto.BrandItemRes             // thay vì *brand_entities.BrandItem
}

type CandidateForRanking struct {
    Item            *wardrobeDto.WardrobeItemRes   // thay vì *wardrobe_entities.WardrobeItem
    Source          string
    VectorScore     float64
    LexicalScore    float64
    RetrievalScore  float64
    RetrievalRank   int
    RetrievalSource string
    ItemContext     outfititemcontext.OutfitItemContext
    BrandItem       *brandDto.BrandItemRes         // thay vì *brand_entities.BrandItem
}

type RankedCandidate struct {
    Item          *wardrobeDto.WardrobeItemRes     // thay vì *wardrobe_entities.WardrobeItem
    Score         float64
    Tags          []string
    Source        string
    RetrievalRank int
    ItemContext   outfititemcontext.OutfitItemContext
    BrandItem     *brandDto.BrandItemRes           // thay vì *brand_entities.BrandItem
}
```

### Bước 6.4: Cập nhật các file xử lý ranking

**Pattern chung cho tất cả:** `item.FashionItemID` → `item.FashionItem` (và `*item.FashionItemID` → `item.FashionItem.ID`)

#### 6.4.1 `ranking/scoring.go`
- Đổi tham số `item *wardrobe_entities.WardrobeItem` → `item *wardrobeDto.WardrobeItemRes`
- `item.LastUsedAt` — giữ nguyên
- `item.FashionItemID` — đổi thành `item.FashionItem` trong `fashionForItemFromMap()`
- Xóa import `wardrobe_entities`
- Hàm `fashionForItemFromMap`: `item.FashionItemID != nil` → `item.FashionItem != nil`; `*item.FashionItemID` → `item.FashionItem.ID`

#### 6.4.2 `ranking/pool_expansion.go`
- Đổi tất cả `*wardrobe_entities.WardrobeItem` → `*wardrobeDto.WardrobeItemRes`
- `item.FashionItemID != nil` → `item.FashionItem != nil`
- `*item.FashionItemID` → `item.FashionItem.ID`
- `item.ID` — giữ nguyên
- Xóa import `wardrobe_entities`

#### 6.4.3 `ranking/fallback_hsl.go`
- Đổi tất cả `*wardrobe_entities.WardrobeItem` → `*wardrobeDto.WardrobeItemRes`
- `item.FashionItemID != nil` → `item.FashionItem != nil`
- `*item.FashionItemID` → `item.FashionItem.ID`
- `item.ID` — giữ nguyên
- Xóa import `wardrobe_entities`

#### 6.4.4 `ranking/ranking.go`
- `candidate.Item.FashionItemID` → `candidate.Item.FashionItem`
- `*candidate.Item.FashionItemID` → `candidate.Item.FashionItem.ID`
- `candidate.Item.ID` — giữ nguyên
- Xóa import `wardrobe_entities` (nếu có)

#### 6.4.5 `candidate_helpers.go`
- `validateAndGetActiveItems`:
  - Đổi `[]*wardrobe_entities.WardrobeItem` → `[]*wardrobeDto.WardrobeItemRes`
  - `item.Status == wardrobestatus.InWardrobe` — giữ nguyên (`WardrobeItemRes.Status` là cùng type)
  - `!item.IsDeleted` — giữ nguyên (đã thêm vào DTO)
  - Import `wardrobestatus` — được phép
- `filterCandidates`:
  - Map entity → DTO khi tạo `CandidateForRanking`/`CandidateForPrompt`:
    ```go
    Item: &wardrobeDto.WardrobeItemRes{
        ID:          wardrobeItem.ID,
        Status:      wardrobeItem.Status,
        IsDeleted:   wardrobeItem.IsDeleted,
        LastUsedAt:  wardrobeItem.LastUsedAt,
        FashionItem: &fashion_dto.FashionItemRes{ID: *wardrobeItem.FashionItemID},
    }
    ```
  - `c.Item.FashionItemID` → `c.Item.FashionItem`
  - `*c.Item.FashionItemID` → `c.Item.FashionItem.ID`
  - Mock brand item:
    ```go
    BrandItem: &brandDto.BrandItemRes{
        ID:        brandItem.ID,
        BrandID:   brandItem.BrandID,
        BrandName: brandItem.Brand.Name, // nil check trước
        Name:      brandItem.Name,
        ItemType:  string(brandItem.ItemType), // cast nếu ItemType là const
        Price:     brandItem.Price,
    }
    ```
  - Xóa import `wardrobe_entities`, `brand_entities`

#### 6.4.6 `mapper/recommendation.go`
- `enrichWardrobeItemRes`:
  - `cand.BrandItem.Brand.Name` → `cand.BrandItem.BrandName`
  - `cand.BrandItem.Brand != nil` → không cần (đã flatten)
  - `string(cand.BrandItem.ItemType)` → `cand.BrandItem.ItemType` (nếu type là string)
- `resolvePrimaryCandidate`, `resolveAlternativeCandidates`, `matchesRole`:
  - `item.FashionItemID` → `item.FashionItem`
  - `*item.FashionItemID` → `item.FashionItem.ID`
- `MapLLMResponseToGroups`:
  - `*primary.FashionItemID` → `primary.FashionItem.ID`
- Xóa import `wardrobe_entities`, `brand_entities`

### Bước 6.5: Cập nhật test files

**Pattern chung:** Entity → DTO mapping trong test data:

```go
// Entity cũ:
&wardrobe_entities.WardrobeItem{
    AuditableEntity: wardrobe_entities.AuditableEntity{
        BaseEntity: wardrobe_entities.BaseEntity{ID: itemID},
    },
    FashionItemID: &fashionID,
}

// DTO mới:
&wardrobeDto.WardrobeItemRes{
    ID: itemID,
    FashionItem: &fashion_dto.FashionItemRes{ID: fashionID},
}
```

Các file cần sửa:
- `ranking/scoring_test.go` — thay entity bằng DTO, xóa import `wardrobe_entities`
- `ranking/fallback_hsl_test.go` — tương tự
- `synthesis/synthesis_test.go` — tương tự

### Bước 6.6: Xóa các import entity còn sót

Kiểm tra toàn bộ module `fashion` không còn import:
- `smart-wardrobe-be/internal/modules/wardrobe/domain/entities`
- `smart-wardrobe-be/internal/modules/brand/domain/entities`

```bash
go vet ./internal/modules/fashion/...
```

Nếu còn lỗi `undefined: wardrobe_entities` hoặc `undefined: brand_entities`, tìm và xử lý tiếp.

### Bước 6.7: Xác minh

1. `go vet ./internal/modules/fashion/...` — không lỗi
2. `go test ./internal/modules/fashion/...` — test pass
3. Kiểm tra không còn vi phạm Rule 1.1 (import entity/repository từ module khác)

---

## Lưu ý quan trọng

1. **Pattern thay thế `FashionItemID`:** Entity dùng `*uuid.UUID`, DTO có `FashionItem *FashionItemRes`. Cần đổi:
   - `if item.FashionItemID != nil { id := *item.FashionItemID }` → `if item.FashionItem != nil { id := item.FashionItem.ID }`

2. **`Status` type:** `WardrobeItemRes.Status` là `wardrobestatus.WardrobeItemStatus`. Import `wardrobestatus` trong fashion module là **chấp nhận được**.

3. **`BrandItemRes.ItemType`:** Kiểm tra type thực tế. Nếu là `string` thì dùng trực tiếp; nếu là `branditemtype.BrandItemType` thì cast `string(...)`.

4. **Test data:** Các test hiện tại tạo entity với `ID` và `FashionItemID`. Mapping DTO tương ứng: `FashionItemID: &id` → `FashionItem: &FashionItemRes{ID: id}`.

5. **Thứ tự thực hiện khuyến nghị:** Bước 6.1 (DTO) → 6.2 (DTO) → 6.3 (types) → 6.4.1-6.4.4 (ranking) → 6.4.5 (candidate_helpers) → 6.4.6 (mapper) → 6.5 (tests) → 6.6 (cleanup) → 6.7 (verify).
