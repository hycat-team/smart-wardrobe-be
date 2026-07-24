# User Dashboard API Contracts

**Base path**: `/api/v1/me/dashboard`
**Auth**: JWT (scope tự động theo user)

---

## 12. GET /wardrobe/category-distribution

**Response 200**:
```json
{
  "success": true,
  "data": {
    "totalItems": 42,
    "categories": [
      { "categoryId": "uuid", "categoryName": "Áo Sơ Mi", "itemCount": 12, "percentage": 28.57 },
      { "categoryId": "uuid", "categoryName": "Quần Jean", "itemCount": 10, "percentage": 23.81 },
      { "categoryId": "uuid", "categoryName": "Giày", "itemCount": 6, "percentage": 14.29 }
    ]
  }
}
```

**DTOs**:
```go
type CategoryDistributionDTO struct {
    CategoryID   uuid.UUID `json:"categoryId"`
    CategoryName string    `json:"categoryName"`
    ItemCount    int       `json:"itemCount"`
    Percentage   float64   `json:"percentage"`
}

type WardrobeCategoryDistributionRes struct {
    TotalItems int                       `json:"totalItems"`
    Categories []CategoryDistributionDTO `json:"categories"`
}
```
