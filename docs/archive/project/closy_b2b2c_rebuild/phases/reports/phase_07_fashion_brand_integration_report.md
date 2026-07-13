# Phase 07 Report - Fashion Refactor & Brand Item Integration

Báo cáo kết quả triển khai Phase 07 (Tích hợp sản phẩm thương hiệu - Brand Items vào luồng phối đồ AI và tủ đồ cá nhân), hoàn thành tái cấu trúc và bổ sung các cơ chế phân quyền, lọc candidates, hiển thị ngữ cảnh đồ cá nhân vs đồ brand.

## Files Changed

- **Brand Module**:
  - `internal/modules/brand/domain/repositories/interfaces.go` (Thêm signature `GetByUserID` cho `IBrandCustomerRepository` và `GetByFashionItemID` cho `IBrandItemRepository`)
  - `internal/modules/brand/infrastructure/persistence/brand_repo.go` (Triển khai `GetByUserID` truy vấn membership của user tại mọi brand)
  - `internal/modules/brand/infrastructure/persistence/brand_item_repo.go` (Triển khai `GetByFashionItemID` và tự động preload `FashionItem.Category`)
  - `internal/modules/brand/contract/service.go` (Khai báo `CheckBrandItemEligibility` và `ListEligibleBrandItemsForStyling` trong `IBrandContract`)
  - `internal/modules/brand/application/interface/usecase/brand_core_uc.go` & `brand_core_uc.go` (Triển khai chi tiết logic nghiệp vụ kiểm tra quyền hội viên, trạng thái hoạt động, và quyền `SAMPLE_MIX_ACCESS` đối với các mẫu thử kỹ thuật số)
- **Wardrobe Module**:
  - `internal/shared/domain/entities/wardrobe_entities.go` (Bổ sung liên kết `BrandItem` vào cấu trúc `OutfitItem`)
  - `internal/modules/wardrobe/application/dto/wardrobe.go` (Định nghĩa `BrandItemBriefRes` và nhúng trường nhận diện vào `WardrobeItemRes`)
  - `internal/modules/wardrobe/application/dto/outfit.go` (Nhúng trường `brandItem` vào DTO `OutfitItemRes` phục vụ canvas đồ phối)
  - `internal/modules/wardrobe/application/mapper/outfit.go` (Triển khai mapping ánh xạ thông tin Brand Item trong `MapToOutfitRes`)
  - `internal/modules/wardrobe/infrastructure/persistence/outfit_repo.go` (Cấu hình `GetDetailByID` và viết helper `attachBrandItems` nạp đầy đủ thông tin sản phẩm của thương hiệu)
  - `internal/modules/wardrobe/application/usecase/outfit/outfit_uc.go` (Tiêm `brandContract` vào `OutfitUseCase`, cập nhật `SaveOutfit`/`UpdateOutfit` để kiểm tra độ hợp lệ của các sản phẩm thương hiệu tham gia vào bộ phối đồ)
- **Fashion Module**:
  - `internal/modules/wardrobe/application/dto/recommendation.go` (Bổ sung cờ `include_brand_items` vào `RecommendOutfitReq`)
  - `internal/modules/fashion/application/usecase/ai/recommendation/types/types.go` (Thêm trường nhận diện `ItemContext` và `BrandItem` vào cấu trúc candidate sử dụng kiểu `outfititemcontext.OutfitItemContext`)
  - `internal/modules/fashion/application/usecase/ai/recommendation/usecase.go` (Tiêm `brandContract` vào `OutfitRecommendationUseCase`)
  - `internal/modules/fashion/application/usecase/ai/recommendation/ranking/ranking.go` (Cập nhật `RankCandidates` để truyền dẫn đầy đủ thông tin ngữ cảnh và thương hiệu)
  - `internal/modules/fashion/application/usecase/ai/recommendation/candidate_helpers.go` (Cập nhật logic `filterCandidates` sử dụng hằng số `outfititemcontext.BrandItem` và `outfititemcontext.UserWardrobe` để ưu tiên tối đa 30% items của brand - tương đương tối đa 6 món đồ thương hiệu trong bể 20 candidates)
  - `internal/modules/fashion/application/usecase/ai/recommendation/synthesis/mapper.go` (Cấu hình mapper làm giàu `WardrobeItemRes` đầu ra sử dụng các hằng số context từ package `outfititemcontext`)
  - `internal/modules/fashion/application/usecase/ai/chat/usecase.go` & `workflows.go` (Tiêm `brandContract` vào chatbot chat session và message workflows)
  - `internal/modules/fashion/application/usecase/ai/chat/helpers.go` (Đưa danh sách sản phẩm thương hiệu hoạt động của user vào chatbot system prompt và hướng dẫn chatbot cách phản hồi bằng tiếng Việt)
- **Dependency Injection & Tests**:
  - `internal/modules/wardrobe/provider.go` (Tự động wire `IBrandContract` vào `OutfitUseCase`)
  - `internal/modules/brand/application/usecase/brand_core_uc_benefit_test.go` & `brand_core_uc_item_test.go` (Bổ sung các phương thức mock cho repository phục vụ chạy unit test thành công)

## Core Mechanisms Implemented

1. **Brand Item Eligibility Verification**:
   - Khi người dùng lưu hoặc cập nhật Outfit chứa đồ của Brand, hệ thống sẽ xác minh xem:
     - Thương hiệu (Brand) đó có trạng thái hoạt động (`ACTIVE`).
     - Khách hàng (User) đã đăng ký hội viên hoạt động (`ACTIVE`) tại Brand đó.
     - Nếu item là loại mẫu thử (`SAMPLE`), kiểm tra xem User có quyền lợi `"SAMPLE_MIX_ACCESS"` tại thương hiệu đó không. Nếu không, trả về lỗi cấm phối đồ.
2. **AI Candidates RAG Prioritization (Ratio maximum 30%)**:
   - Khi luồng gợi ý phối đồ AI kích hoạt cờ `include_brand_items`:
     - Hệ thống gọi `brandContract.ListEligibleBrandItemsForStyling` để lấy các sản phẩm/mẫu thử của brand mà user đủ quyền phối.
     - Chọn ngẫu nhiên hoặc lấy tối đa 30% của bể (tương đương 6 món đồ) nhét vào bể candidates xếp hạng cao để tăng tính đa dạng và tăng khả năng AI lựa chọn đề xuất.
     - 70% còn lại là đồ trong tủ cá nhân của user lấy qua hybrid search (RAG).
3. **Response Context Identification**:
   - Mọi item được trả về từ luồng AI và Canvas phối đồ đều đính kèm `itemContext` ("BRAND_ITEM" / "USER_WARDROBE") kèm theo thông tin chi tiết của thương hiệu (`brandItem`) nếu thuộc về brand, giúp Frontend hiển thị mác/nhãn phân biệt trực quan cho người dùng.

## Verification & Build Status

- **Dependency Injection**: Chạy lệnh `make wire` thành công.
- **Swagger Documentation**: Chạy lệnh `make swagger` thành công.
- **Automated Tests**: Toàn bộ unit tests của project pass (`go test ./...` thành công 100%).
