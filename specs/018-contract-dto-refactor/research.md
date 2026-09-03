# Technical Research: Contract DTO Isolation & Purity Refactor

**Feature**: `018-contract-dto-refactor`
**Date**: 2026-09-03
**Status**: Completed

---

## 1. Vấn Đề Nghiên Cứu & Bối Cảnh Kỹ Thuật

### 1.1 Hiện trạng xung đột DTO (Over-sharing DTO)
Hiện tại, `WardrobeItemRes` / `WardrobeItemBriefRes` trong `wardrobe` và `BrandItemRes` / `BrandItemBriefRes` trong `brand` đang bị gánh vác hai trách nhiệm đối lập:
- Phục vụ API Client: Yêu cầu giàu dữ liệu (Enriched), có trường `FashionItem` để Frontend hiển thị hình ảnh, danh mục, màu sắc.
- Phục vụ Contract: Yêu cầu dữ liệu thô (Pure), các trường lồng `FashionItem` luôn luôn là `nil` (do Contract Use Case tuân thủ Pure Contract Principle của Hiến pháp §II).

### 1.2 Quyết định kiến trúc đã làm rõ (Clarification Decisions)
1. **Vị trí và Naming**:
   - Tạo file `internal/modules/<module>/application/dto/contract_dto.go`.
   - Nằm cùng package `dto` trong thư mục `application/dto/` để tuân thủ quy tắc import của Hiến pháp §II (`application/dto` được phép import chéo module).
   - Tên struct quy chuẩn sử dụng hậu tố `ContractDTO` (ví dụ `WardrobeItemContractDTO`, `BrandItemContractDTO`).
2. **Chiến lược phân kỳ (Phasing)**:
   - **Phase 1 (Priority P1)**: Refactor dứt điểm 2 module vi phạm rò rỉ quan hệ thực thể là `wardrobe` và `brand`, đồng thời cập nhật toàn bộ callers (`fashion`, `dashboard`, `workers`) và mappers.
   - **Phase 2 (Priority P2)**: Chuẩn hóa di chuyển các DTO phục vụ contract vào `contract_dto.go` cho các module còn lại (`identity`, `subscription`, `fashion`, `admin`, `notification`).
3. **Cơ chế chuyển đổi (Cutover)**:
   - Áp dụng **Strict Cutover** trực tiếp trên method signature của các interface trong `presentation/contract/`. Trình biên dịch Go (`go build`) sẽ đảm bảo bắt trọn vẹn 100% tất cả callers cần cập nhật.

---

## 2. Ma Trận Audit Tác Động (Impact & Blast Radius Matrix)

### 2.1 Interface `IWardrobeContract` và các Callers
- **Interface**: `internal/modules/wardrobe/presentation/contract/wardrobe_contract.go`
- **Chữ ký thay đổi**:
  - `GetWardrobeItemsByUserID(...) ([]*dto.WardrobeItemContractDTO, error)`
  - `GetActiveWardrobeItemsByUserID(...) ([]*dto.WardrobeItemContractDTO, error)`
  - `GetFailedItemsForCleanup(...) ([]*dto.WardrobeItemContractDTO, error)`
  - `GetStaleProcessingItems(...) ([]*dto.WardrobeItemContractDTO, error)`
- **Các Callers bị tác động**:
  1. `fashion/application/usecase/ai/recommendation/usecase.go` & `workflow_helpers.go`
  2. `fashion/application/usecase/ai/chat/usecase.go` & `workflows.go`
  3. `fashion/application/usecase/worker/cleanup_uc.go` & `worker_uc.go`
  4. `fashion/application/usecase/category/category_uc.go`
  5. `dashboard/application/usecase/user_dashboard_uc_impl.go`
  6. `dashboard/application/usecase/brand_dashboard_uc_impl.go`
  7. Các mock interface và unit test tương ứng (`worker_uc_test.go`, `workflows_test.go`, `user_dashboard_uc_test.go`, `brand_dashboard_uc_test.go`).

### 2.2 Interface `IBrandContract` và các Callers
- **Interface**: `internal/modules/brand/presentation/contract/brand_contract.go`
- **Chữ ký thay đổi**:
  - `ListEligibleBrandItemsForStyling(...) ([]*dto.BrandItemContractDTO, []*fashion_dto.FashionItemRes, error)`
  - `GetBrandItemByID(...) (*dto.BrandItemContractDTO, error)`
  - `GetBrandItemsByFashionItemIDs(...) ([]*dto.BrandItemContractDTO, error)`
  - `GetBrandItemsByIDs(...) ([]*dto.BrandItemContractDTO, error)`
  - `CheckBrandItemEligibility(...) (bool, *dto.BrandItemContractDTO, *fashion_dto.FashionItemRes, error)`
- **Các Callers bị tác động**:
  1. `fashion/application/usecase/ai/recommendation/usecase.go` & `workflow_helpers.go`
  2. `fashion/application/usecase/ai/chat/usecase.go` & `workflows.go`
  3. `wardrobe/application/usecase/wardrobe/item/item_uc_read.go`
  4. `wardrobe/application/usecase/wardrobe/item/item_uc_write.go`
  5. `wardrobe/application/usecase/wardrobe/catalog/catalog_uc.go`
  6. `wardrobe/application/usecase/outfit/outfit_uc.go`
  7. `dashboard/application/usecase/*_dashboard_uc_impl.go` (brand, user, admin, drilldown, dynamic_query)
  8. Các unit test / mock tương ứng trong `dashboard` và `fashion`.

### 2.3 Các module còn lại (Phase 2)
- `identity`: Di chuyển `UserBriefRes` / `UserBriefContractDTO` vào `identity/application/dto/contract_dto.go`.
- `subscription`: Di chuyển `UserSubscriptionOverviewDTO`, `UserSubscriptionDTO` vào `subscription/application/dto/contract_dto.go`.
- `fashion`: Di chuyển `FashionItemContractDTO`, `FashionItemBriefContractDTO`, `FashionProcessingInfoDTO` vào `fashion/application/dto/contract_dto.go`.
- `notification`: Di chuyển `SendEmailInput` vào `notification/application/dto/contract_dto.go`.
- `admin`: Di chuyển `CreateAuditLogReq` vào `admin/application/dto/contract_dto.go`.

---

## 3. Best Practices & Kỹ Thuật Stitching Phía Caller

Tuân thủ nghiêm ngặt Hiến pháp §II:
1. **Caller trong nội bộ module Fashion**:
   - Khi nhận `WardrobeItemContractDTO` từ WardrobeContract:
     - Lọc danh sách `FashionItemID != nil`.
     - Gọi trực tiếp `fashionItemRepo.GetByIDs(ctx, ids)` (tuân thủ Clean Architecture §IV).
     - Sử dụng `sliceutils.IndexBy` để map thành `map[uuid.UUID]*entities.FashionItem` với độ phức tạp $O(1)$.
2. **Caller ngoài module Fashion (ví dụ Wardrobe gọi BrandContract)**:
   - Khi nhận `BrandItemContractDTO` từ BrandContract:
     - Dùng các trường thuộc tính của Brand đã có sẵn (`BrandID`, `BrandName`, `Price`, `Name`, `Status`).
     - Nếu cần thêm fashion, gọi `fashionContract.ListFashionItems(ctx, ids)` rồi `sliceutils.IndexBy`.
3. **Nil Guard cho `FashionItemID`**:
   - Trong `WardrobeItemContractDTO`, `FashionItemID` là `*uuid.UUID`. Các worker hoặc caller bắt buộc phải kiểm tra `!= nil` trước khi dereference để tránh panic.

---

## 4. Rủi Ro & Biện Pháp Kiểm Soát

| Rủi ro | Mức độ | Biện pháp kiểm soát |
|:---|:---|:---|
| Sót caller dẫn đến compile error | Cao | Strict cutover tận dụng `go build ./...` phát hiện 100% lỗi cú pháp ngay lập tức |
| Lỗi vỡ giao diện Web/Mobile (API regression) | Cao | Giữ nguyên các DTO trong API Handler (`WardrobeItemRes`, `BrandItemRes`). Chỉ thay đổi tầng Contract |
| Test suite bị fail do mock contract cũ | Trung bình | Cập nhật lại các file mock và unit test tương ứng trong `dashboard`, `fashion`, `wardrobe` |
| Wire DI bị ảnh hưởng | Thấp | Signature method thay đổi nhưng Provider set của Wire bind interface giữ nguyên; chạy `make wire` để tái kiểm tra |
