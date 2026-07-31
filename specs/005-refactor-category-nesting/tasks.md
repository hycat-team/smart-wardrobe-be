# Tasks - Refactor Category Nesting

## Phase 1: Setup

- [x] T001 Kiểm tra trạng thái hiện tại của nhánh Git và codebase đảm bảo sạch trước khi refactor.

## Phase 2: Foundational

- [x] T002 Định nghĩa struct `CategoryBriefRes` trong DTOs của module fashion [contract_dtos.go](../../internal/modules/fashion/application/dto/contract_dtos.go).
- [x] T003 Cấu trúc lại `FashionItemRes` và `FashionItemBriefRes` trong [contract_dtos.go](../../internal/modules/fashion/application/dto/contract_dtos.go) để sử dụng con trỏ `Category *CategoryBriefRes`.
- [x] T004 Cập nhật mapping dữ liệu sang cấu trúc mới trong [fashion_contract_impl.go](../../internal/modules/fashion/presentation/contract/fashion_contract_impl.go) và [fashion_item_uc.go](../../internal/modules/fashion/application/usecase/fashion_item/fashion_item_uc.go).

## Phase 3: User Story 1 - Wardrobe Item Nesting

- [x] T005 [P] [US1] Sửa đổi struct `FashionItemRes` nội bộ và loại bỏ `CategoryRes` ở root level của các DTO trong [wardrobe.go DTO](../../internal/modules/wardrobe/application/dto/wardrobe.go).
- [x] T006 [P] [US1] Cập nhật các DTO liên quan trong [outfit.go DTO](../../internal/modules/wardrobe/application/dto/outfit.go).
- [x] T007 [US1] Cập nhật logic ánh xạ của mapper trong [wardrobe.go mapper](../../internal/modules/wardrobe/application/mapper/wardrobe.go).
- [x] T008 [US1] Cập nhật logic mapper của outfit trong [outfit.go mapper](../../internal/modules/wardrobe/application/mapper/outfit.go).
- [x] T009 [US1] Đồng bộ hóa mapping của Elasticsearch và mapper tìm kiếm trong [search_mapper.go](../../internal/modules/wardrobe/application/mapper/search_mapper.go) và [wardrobe_helper.go](../../internal/modules/wardrobe/infrastructure/search/wardrobe_helper.go).

## Phase 4: User Story 2 - Brand Item Nesting & AI Sync

- [x] T010 [P] [US2] Cập nhật struct `FashionItemRes` nội bộ của module brand trong [brand_item.go DTO](../../internal/modules/brand/application/dto/brand_item.go).
- [x] T011 [US2] Cập nhật logic mapper tương ứng trong [brand_item_mapper.go](../../internal/modules/brand/application/mapper/brand_item_mapper.go).
- [x] T012 [US2] Sửa đổi truy cập từ trường phẳng sang nested category trong toàn bộ AI Recommendation Engine (`workflow_helpers.go`, `synthesis/prompt.go`, `ranking/scoring.go`, `ranking/ranking.go`, và các test suites liên quan).
- [x] T013 [US2] Cập nhật AI Background Worker trong [worker_uc.go](../../internal/modules/fashion/application/usecase/worker/worker_uc.go).

## Phase 5: Polish & Validation

- [x] T014 Chạy lệnh `make wire` để tái tạo mã nguồn DI tự động.
- [x] T015 Biên dịch dự án bằng `make build` và xác thực kiểm thử thông qua `make test`.
