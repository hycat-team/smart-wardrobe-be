# Contract Purity Pattern (Bản nháp)

## 1. Định nghĩa & Nguyên tắc
Trong cấu trúc Modular Monolith của dự án SmartWardrobe, tầng **Presentation Contract** đóng vai trò là "cổng giao tiếp duy nhất" chéo module (Cross-Module Communication).

Để ngăn chặn các vấn đề về **Detour Call** (gọi vòng chéo module làm tăng độ phức tạp) và đảm bảo tính đóng gói độc lập, chúng ta áp dụng nguyên tắc **Pure Contract**:
- **Pure Contract**: Một Contract Implementation (và Use Case tương ứng phục vụ nó) chỉ được thao tác trực tiếp với dữ liệu (Repository, Domain) của chính module đó.
- **Không tự ý Enrich ngầm**: Không được gọi Contract của module khác từ bên trong Contract Use Case để tự làm đầy thông tin (ví dụ: `WardrobeContract` không được gọi `FashionContract` để điền thông tin ảnh/dòng thời trang).
- **Trách nhiệm của Caller (Caller-Enriched)**: Bên nào gọi Contract và có nhu cầu hiển thị thông tin liên kết thì bên đó (Caller) có trách nhiệm gọi tiếp các Contract hoặc Repository tương ứng để làm đầy và tự thực hiện ghép dữ liệu (Stitching) bằng hàm tiện ích `sliceutils.IndexBy` hoặc `sliceutils.GroupBy` với độ phức tạp $O(1)$.

## 2. So sánh Before/After

### Luồng cũ (Leaky/Detour):
```
FashionUseCases ➔ WardrobeContract ➔ WardrobeContractUseCase ➔ (Gọi chéo) FashionContract ➔ Trả về kèm dữ liệu làm sẵn
```

### Luồng mới (Pure):
```
1. FashionUseCases ➔ WardrobeContract ➔ Trả về WardrobeItemRes thô (chỉ có FashionItemID, FashionItem = nil)
2. FashionUseCases ➔ Gọi Repo/Contract nội bộ để tự enrich dữ liệu dựa trên FashionItemID
```

## 3. Cách stitch dữ liệu (Stitching)
Sử dụng `sliceutils.IndexBy` / `GroupBy` của package `sliceutils` (Hiến pháp §II):

**Caller nội bộ fashion (dùng repo trực tiếp, tuân §IV):**
```go
// Lấy danh sách thô (pure, FashionItem == nil nhưng có FashionItemID)
items, err := uc.wardrobeContract.GetWardrobeItemsByUserID(ctx, userID)

// Gom IDs cần làm giàu (kèm nil guard)
fashionIDs := make([]uuid.UUID, 0, len(items))
for _, item := range items {
    if item.FashionItemID != nil {
        fashionIDs = append(fashionIDs, *item.FashionItemID)
    }
}
fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)

// Gọi repo nội bộ để lấy dữ liệu làm giàu (không gọi fashionContract của chính mình)
fashionItems, err := uc.fashionItemRepo.GetByIDs(ctx, fashionIDs)

// Stitching O(1)
fashionMap := sliceutils.IndexBy(fashionItems, func(f *entities.FashionItem) uuid.UUID {
    return f.ID
})
for _, item := range items {
    if item.FashionItemID != nil {
        if fi, ok := fashionMap[*item.FashionItemID]; ok {
            // Map vào DTO trước khi trả về
        }
    }
}
```

**Caller ngoài module (ví dụ brand → fashion):** dùng `fashionContract.ListFashionItems` tương tự, rồi `IndexBy`.

## 4. Trường hợp dùng chung handler + contract (Brand)

`BrandItemUseCase` từng được dùng chung bởi `BrandContract` và `BrandPortalHandler`. Nếu thuần hóa trực tiếp sẽ làm handler mất `FashionItem` (ví dụ `GetBrandItemForStaff`).

**Giải pháp:** giữ `BrandItemUseCase` cho handler (vẫn enrich), tạo `BrandItemContractUseCase` riêng ở `brand/application/usecase/item/contractuc/` chỉ phụ thuộc `brandRepo`/`brandItemRepo`/`customerRepo`/`benefitUC` (không `fashionContract`/`userBriefContract`). `BrandContract` chuyển sang inject `IBrandItemContractUseCase`.

Wardrobe đã tách sẵn `WardrobeContractUseCase` ở `wardrobe/application/usecase/wardrobe/contractuc/` nên chỉ cần sửa trực tiếp 4 alive leaky + thêm `FashionItemID`.

## 5. Nil guard và FashionItemID

`WardrobeItemRes` sau pure luôn có `FashionItemID *uuid.UUID \`json:"fashionItemId,omitempty"\`` (camelCase) kể cả khi `FashionItem == nil` — mapper `MapToWardrobeItemRes` gán `FashionItemID: item.FashionItemID`.

- **Worker `CleanupFailedItems` / `RecoverStaleProcessingItems`:** nếu `item.FashionItemID == nil` (upload lỗi trước khi tạo fashion record) → bỏ qua `fashionItemRepo.GetByID` và bỏ qua xóa ảnh Cloudinary, chỉ xóa tủ đồ.
- Nếu `!= nil` → `fashionItemRepo.GetByID(ctx, *FashionItemID)` để lấy `ImagePublicID` / `ProcessingRetryCount`.

`BrandItemRes` đã có `FashionItemID` sẵn, caller tương tự dùng `FashionItemID`.

## 6. Audit guide (method-level) và Wire

- **Audit:** dùng `rg`/`Select-String` đếm detour 7 chuỗi, pure violation 2 contract (wardrobe 4/8 alive, brand 5/9), deadcode 12, dead wiring 2 (`wardrobe aiService` + `subscription userBriefContract`). Script `scripts/audit-contract-purity.ps1` trả về `0/0/0/0` sau refactor.
- **Wire:** Wardrobe bỏ `fashionContract`/`userSubContract`; Brand thêm `item/contractuc` ProviderSet + bind `IBrandItemContractUseCase`; xóa dead wiring provider; `make wire && make build` phải pass.
- **Performance:** benchmark dự đoán bằng đếm `fashionItemRepo.GetByIDs` trước/sau cho 4 flow `GetWardrobeItemsByUserID`/`GetActive...`/`GetFailed...`/`GetStale...`, gộp batch nếu tăng >10%.
