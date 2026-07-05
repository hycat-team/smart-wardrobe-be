# PHASE 6: Tạo DTO đầy đủ cho Wardrobe & Brand phục vụ Fashion Ranking
**Thời điểm thực hiện:** Sau khi Phase 5 hoàn thành và hệ thống ổn định với kiến trúc mới.

## Mục tiêu

Tạo các DTO mới trong `wardrobe/application/dto/` và `brand/application/dto/` chứa đầy đủ field cần thiết cho các thuật toán ranking của module `fashion`, nhằm thay thế hoàn toàn việc import trực tiếp `domain/entities` của module khác (vi phạm Rule 1.1).

## Lý do

Module `fashion` hiện đang import trực tiếp:
- `wardrobe/domain/entities` — dùng `WardrobeItem` trong các struct `CandidateForPrompt`, `CandidateForRanking`, `RankedCandidate` và các hàm scoring/ranking
- `brand/domain/entities` — dùng `BrandItem` trong các struct trên và mapper

Các entity này được dùng với số lượng field hạn chế, có thể thay thế bằng DTO.

---

## Phân tích field sử dụng

### WardrobeItem — fields được dùng trong fashion ranking

| Field | Nguồn gốc | Dùng trong |
|-------|-----------|------------|
| `ID` | `AuditableEntity.BaseEntity` | `types.go`, `scoring.go`, `ranking.go`, test files |
| `FashionItemID *uuid.UUID` | Trực tiếp | `ranking.go:202`, `scoring.go:264`, `mapper/recommendation.go:233` |
| `LastUsedAt *time.Time` | Trực tiếp | `scoring.go:229` |

### BrandItem — fields được dùng trong fashion ranking

| Field | Nguồn gốc | Dùng trong |
|-------|-----------|------------|
| `ID` | `AuditableEntity.BaseEntity` | `mapper/recommendation.go:255` |
| `BrandID uuid.UUID` | Trực tiếp | `mapper/recommendation.go:256` |
| `Brand.Name` | Qua `*Brand` relation | `mapper/recommendation.go:251-252` |
| `Name string` | Trực tiếp | `mapper/recommendation.go:259` |
| `ItemType` | Trực tiếp | `mapper/recommendation.go:258` |
| `Price *float64` | Trực tiếp | `mapper/recommendation.go:260` |

---

## Kế hoạch thực hiện

### Bước 6.1: Tạo DTO cho Wardrobe

Tạo file `internal/modules/wardrobe/application/dto/ranking_dto.go`:

```go
package dto

import (
    "time"
    "github.com/google/uuid"
)

// WardrobeItemRankingDTO contains fields needed by fashion ranking algorithms.
// Derived from wardrobe/domain/entities.WardrobeItem.
type WardrobeItemRankingDTO struct {
    ID           uuid.UUID
    FashionItemID *uuid.UUID
    LastUsedAt   *time.Time
}
```

### Bước 6.2: Tạo DTO cho Brand

Tạo file `internal/modules/brand/application/dto/ranking_dto.go`:

```go
package dto

import (
    "github.com/google/uuid"
)

// BrandItemRankingDTO contains fields needed by fashion ranking algorithms.
// Derived from brand/domain/entities.BrandItem.
type BrandItemRankingDTO struct {
    ID        uuid.UUID
    BrandID   uuid.UUID
    BrandName string  // flattened from Brand.Name
    Name      string
    ItemType  string
    Price     *float64
}
```

### Bước 6.3: Cập nhật `types` package trong fashion

Sửa file `internal/modules/fashion/application/usecase/ai/recommendation/types/types.go`:

1. **Thay thế các import:**
   - `wardrobe_entities "smart-wardrobe-be/internal/modules/wardrobe/domain/entities"` → `wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"`
   - `brand_entities "smart-wardrobe-be/internal/modules/brand/domain/entities"` → `brandDto "smart-wardrobe-be/internal/modules/brand/application/dto"`

2. **Cập nhật các struct:**
   ```go
   type CandidateForPrompt struct {
       Item        *wardrobeDto.WardrobeItemRankingDTO
       Tags        []string
       ItemContext outfititemcontext.OutfitItemContext
       BrandItem   *brandDto.BrandItemRankingDTO
   }

   type CandidateForRanking struct {
       Item            *wardrobeDto.WardrobeItemRankingDTO
       Source          string
       VectorScore     float64
       LexicalScore    float64
       RetrievalScore  float64
       RetrievalRank   int
       RetrievalSource string
       ItemContext     outfititemcontext.OutfitItemContext
       BrandItem       *brandDto.BrandItemRankingDTO
   }

   type RankedCandidate struct {
       Item          *wardrobeDto.WardrobeItemRankingDTO
       Score         float64
       Tags          []string
       Source        string
       RetrievalRank int
       ItemContext   outfititemcontext.OutfitItemContext
       BrandItem     *brandDto.BrandItemRankingDTO
   }
   ```

### Bước 6.4: Cập nhật các file sử dụng `types` struct

Sửa các file sau để sử dụng DTO mới thay vì entity:

1. **`fashion/application/usecase/ai/recommendation/ranking/scoring.go`**
   - Đổi tham số `item *wardrobe_entities.WardrobeItem` → `item *wardrobeDto.WardrobeItemRankingDTO`
   - `item.LastUsedAt` → không đổi (field giống nhau)
   - `item.FashionItemID` → không đổi (field giống nhau)
   - Xóa import `wardrobe_entities`

2. **`fashion/application/usecase/ai/recommendation/ranking/pool_expansion.go`**
   - Cập nhật signature và body tương tự
   - Xóa import `wardrobe_entities`

3. **`fashion/application/usecase/ai/recommendation/ranking/fallback_hsl.go`**
   - Cập nhật signature các hàm `isNeutral`, `colorsMatch`, `FallbackSearchDocument`
   - Đổi `*wardrobe_entities.WardrobeItem` → `*wardrobeDto.WardrobeItemRankingDTO`
   - Xóa import `wardrobe_entities`

4. **`fashion/application/usecase/ai/recommendation/ranking/ranking.go`**
   - Cập nhật các hàm xử lý `candidate.Item`
   - `candidate.Item.ID` → không đổi
   - `candidate.Item.FashionItemID` → không đổi

5. **`fashion/application/usecase/ai/recommendation/candidate_helpers.go`**
   - Cập nhật các hàm tạo `CandidateForRanking`, `CandidateForPrompt`
   - Mapping từ entity → DTO tại nơi tạo: dùng `mapper` để chuyển đổi
   - Xóa import `wardrobe_entities`, `brand_entities`
   - **Lưu ý:** `BrandItem.Brand` relation cần được resolve qua contract trước khi map vào DTO

6. **`fashion/application/mapper/recommendation.go`**
   - Cập nhật hàm `enrichWardrobeItemRes` để dùng `*brandDto.BrandItemRankingDTO`
   - `cand.BrandItem.Brand.Name` → `cand.BrandItem.BrandName` (đã flatten trong DTO)
   - Xóa import `wardrobe_entities`, `brand_entities`

### Bước 6.5: Cập nhật các handler/usecase tạo dữ liệu ranking

Tại `candidate_helpers.go` và các nơi khác tạo `CandidateForRanking` / `CandidateForPrompt`:
- Thay vì gán trực tiếp entity `Item: wardrobeItem`, cần map entity → DTO:
  ```go
  Item: &wardrobeDto.WardrobeItemRankingDTO{
      ID:           wardrobeItem.ID,
      FashionItemID: wardrobeItem.FashionItemID,
      LastUsedAt:   wardrobeItem.LastUsedAt,
  }
  ```
- Tương tự cho `BrandItemRankingDTO`:
  ```go
  BrandItem: &brandDto.BrandItemRankingDTO{
      ID:        brandItem.ID,
      BrandID:   brandItem.BrandID,
      BrandName: brandItem.Brand.Name, // cần kiểm tra nil
      Name:      brandItem.Name,
      ItemType:  string(brandItem.ItemType),
      Price:     brandItem.Price,
  }
  ```

### Bước 6.6: Cập nhật test files

Sửa các file test trong `fashion/application/usecase/ai/recommendation/`:

1. **`scoring_test.go`**, **`fallback_hsl_test.go`**, **`synthesis_test.go`**
   - Tạo `WardrobeItemRankingDTO` thay vì `wardrobe_entities.WardrobeItem` trong test data
   - Xóa import `wardrobe_entities`

2. **`ranking/*_test.go`**
   - Cập nhật tương tự

### Bước 6.7: Xóa các import entity còn sót

Sau khi hoàn thành các bước trên, kiểm tra toàn bộ module `fashion` không còn import:
- `smart-wardrobe-be/internal/modules/wardrobe/domain/entities`
- `smart-wardrobe-be/internal/modules/brand/domain/entities`

Nếu còn import nào, refactor tiếp.

### Bước 6.8: Xác minh

1. Chạy `go vet ./internal/modules/fashion/...` — không còn lỗi import chéo
2. Chạy `make test` — toàn bộ test pass
3. Xác nhận không còn vi phạm Rule 1.1 trong `fashion` module

---

## Rủi ro & Lưu ý

1. **Performance:** DTO mapping tạo thêm allocations nhỏ. Các ranking function được gọi nhiều lần, cần kiểm tra benchmark nếu nghi ngờ.
2. **BrandItem.Brand relation:** Trong `mapper/recommendation.go:251`, code truy cập `cand.BrandItem.Brand.Name`. Đây là loaded relation. Với DTO approach, cần resolve Brand name trước khi tạo DTO (tức là brand name phải được load từ DB hoặc cache trước).
3. **Test data:** Các test hiện tại tạo `WardrobeItem` chủ yếu với `ID` và `FashionItemID`. DTO cũng chỉ cần các field này, nên việc chuyển đổi đơn giản.
