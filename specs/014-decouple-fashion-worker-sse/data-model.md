# Mô hình Dữ liệu & Sự kiện (Data & Event Model): 014-decouple-fashion-worker-sse

**Feature**: [spec.md](./spec.md) | **Date**: 2026-08-26

---

## 1. Cấu trúc Sự kiện Hàng đợi (Event Queue Payloads)

### 1.1 `FashionAnalyzedEventPayload`
Sự kiện được phát ra bởi `FashionWorkerUseCase` khi xử lý AI hoàn tất (hoặc thất bại / cần review).

```go
package dto

import (
    "smart-wardrobe-be/internal/modules/fashion/domain/constants/analyzetaskstatus"
    "smart-wardrobe-be/internal/modules/fashion/domain/constants/fashionitemtype"
    "github.com/google/uuid"
)

type FashionAnalyzedEventPayload struct {
    TaskID        string                              `json:"taskId"`
    ItemID        uuid.UUID                           `json:"itemId"`
    ItemType      fashionitemtype.FashionItemType     `json:"itemType"`       // "wardrobe" | "brand"
    FashionItemID *uuid.UUID                          `json:"fashionItemId,omitempty"`
    Status        analyzetaskstatus.AnalyzeTaskStatus `json:"status"`         // "completed" | "needs_review" | "failed"
    Total         int                                 `json:"total"`
    Index         int                                 `json:"index"`
    FashionBrief  *FashionItemBriefRes                `json:"fashionBrief,omitempty"`
    Error         string                              `json:"error,omitempty"`
}
```

* **Quy tắc chuyển đổi trạng thái (State Transition)**:
  * Khi AI nhận diện thành công $\rightarrow$ `Status = completed`.
  * Khi AI nhận diện với độ tin cậy thấp hoặc không phân định được danh mục $\rightarrow$ `Status = needs_review`.
  * Khi file ảnh lỗi, timeout hoặc sự cố không thể phục hồi $\rightarrow$ `Status = failed`.

---

## 2. Cấu trúc Dữ liệu SSE Client Payload (`SSEEventPayload`)

Payload phát qua kết nối SSE gửi về cho Frontend:

```go
type SSEEventPayload struct {
    ItemID uuid.UUID                           `json:"itemId"`
    Status analyzetaskstatus.AnalyzeTaskStatus `json:"status"` // "completed" | "failed" | "needs_review"
    Total  int                                 `json:"total"`
    Index  int                                 `json:"index"`
    Data   any                                 `json:"data,omitempty"`
    Error  string                              `json:"error,omitempty"`
}
```

### 2.1 Chi tiết trường `Data` của Wardrobe Module (`WardrobeItemBriefRes`)
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "price": 250000,
  "status": "in_wardrobe",
  "isLocked": false,
  "lastUsedAt": "2026-08-26T09:30:00Z",
  "taskId": "task-uuid-string",
  "fashionItem": {
    "id": "f1e2d3c4-b5a6-0987-fedc-ba0987654321",
    "imageUrl": "https://res.cloudinary.com/.../shirt.jpg",
    "color": "Xanh lam",
    "colorHex": "#0000FF",
    "style": "Casual",
    "category": {
      "id": "c1b2a3f4-0000-0000-0000-000000000001",
      "name": "Áo sơ mi",
      "slug": "ao-so-mi"
    }
  }
}
```

### 2.2 Chi tiết trường `Data` của Brand Module (`BrandItemBriefRes`)
```json
{
  "id": "b1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "Áo thun phong cách mùa hè",
  "price": 350000,
  "itemType": "standard",
  "status": "active",
  "fashionItem": {
    "id": "f1e2d3c4-b5a6-0987-fedc-ba0987654321",
    "imageUrl": "https://res.cloudinary.com/.../brand-tee.jpg",
    "colorHex": "#FFFFFF",
    "category": {
      "id": "c1b2a3f4-0000-0000-0000-000000000002",
      "name": "Áo thun",
      "slug": "ao-thun"
    }
  }
}
```
