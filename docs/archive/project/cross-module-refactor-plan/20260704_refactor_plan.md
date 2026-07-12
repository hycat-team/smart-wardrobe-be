# KẾ HOẠCH REFACTOR KIẾN TRÚC HỆ THỐNG (SMART WARDROBE BACKEND)
**Ngày tạo plan:** 2026-07-04 (Cập nhật: 2026-07-05)

Tài liệu này vạch ra kế hoạch refactor chi tiết chia nhỏ thành từng bước độc lập để khắc phục toàn bộ các vi phạm kiến trúc được tìm thấy trong [Báo Cáo Phân Tích Vi Phạm Kiến Trúc](architecture_analysis_report.md).

**Nguyên tắc thiết kế cốt lõi:**
- **Contract** là lớp trung gian giữa các module với nhau, tương tự Handler: nó inject UseCase interface, gọi UseCase và trả kết quả. Contract **không implement UseCase**.
- **UseCase** là nơi chứa nghiệp vụ, implement UseCase interface. UseCase **không biết và không implement Contract**.
- **DTO** được định nghĩa tập trung tại `application/dto/` và được Contract import để sử dụng trong khai báo hàm.

Kế hoạch được chia thành **6 Phase** với các bước cực kỳ nhỏ nhằm đảm bảo tính cô đọng, dễ thực hiện và dễ kiểm soát rủi ro.

> **Xem thêm:** [Phase 6 — Tạo DTO cho Wardrobe & Brand](20260704_refactor_plan_phase6_dto_migration.md)

---

## PHASE 1: Gom nhóm và tập trung DTO về `application/dto`
*Mục tiêu: Đảm bảo toàn bộ DTO được định nghĩa ở `application/dto` thay vì để rải rác ở `contract` hoặc file khác. `contract` sẽ **import DTO từ `application/dto`** để dùng trong khai báo hàm. Xóa các DTO dư thừa.*

### Bước 1.1: Refactor DTO của Module `fashion`
- **Chi tiết:**
  1. Di chuyển các struct `FashionItemRes`, `CategoryRes`, `FashionAnalyzeJobDTO`, `CreateCategoryInput`, `UpdateCategoryInput` từ [service.go](file:///internal/modules/fashion/contract/service.go) sang file mới `internal/modules/fashion/application/dto/contract_dtos.go`.
  2. Cập nhật import trong [service.go](file:///internal/modules/fashion/contract/service.go) để tham chiếu đến package `fashion/application/dto` (ví dụ: `dto.FashionItemRes`).
  3. Cập nhật import ở các module khác đang gọi đến các DTO này (ví dụ `brand`, `wardrobe`) — lưu ý: đường dẫn import mới sẽ là `smart-wardrobe-be/internal/modules/fashion/application/dto`.

### Bước 1.2: Refactor DTO của Module `subscription`
- **Chi tiết:**
  1. Di chuyển `UserSubscriptionDTO` và `UserSubscriptionOverviewDTO` từ [dto.go](file:///internal/modules/subscription/contract/dto.go) sang file mới `internal/modules/subscription/application/dto/quota_dtos.go`.
  2. Xóa bỏ hoàn toàn file [dto.go](file:///internal/modules/subscription/contract/dto.go).
  3. Cập nhật import trong [service.go](file:///internal/modules/subscription/contract/service.go) và các usecase/controller đang tham chiếu.

### Bước 1.3: Refactor DTO/Options của Module `aiexecutor`
- **Chi tiết:**
  1. Di chuyển struct `OrchestratorOptions` từ [orchestrator.go](file:///internal/modules/aiexecutor/contract/orchestrator.go#L9) sang file mới `internal/modules/aiexecutor/application/dto/options.go`.
  2. Cập nhật import trong `aiexecutor/contract/orchestrator.go` và các nơi sử dụng.

### Bước 1.4: Dọn dẹp Code dư thừa (Dead Code) tại Module `identity`
- **Chi tiết:**
  1. Xác minh file [dto.go](file:///internal/modules/identity/contract/dto.go) không được import ở đâu khác.
  2. Xóa bỏ hoàn toàn file [dto.go](file:///internal/modules/identity/contract/dto.go).

---

## PHASE 2: Loại bỏ việc Inject Contract nội bộ (Vi phạm Rule 2.3)
*Mục tiêu: Đảm bảo không layer nội bộ nào của module inject và gọi contract interface của chính nó.*

### Bước 2.1: Loại bỏ `fashionContract` trong `WardrobeChatUseCase`
- **Chi tiết:**
  1. Sửa file [usecase.go](file:///internal/modules/fashion/application/usecase/ai/chat/usecase.go#L26) để loại bỏ dependency `fashionContract fashion_contract.IFashionContract`. Thay vào đó, inject `fashionItemRepo repositories.IFashionItemRepository`.
  2. Cập nhật constructor `NewWardrobeChatUseCase`.
  3. Trong [workflows.go](file:///internal/modules/fashion/application/usecase/ai/chat/workflows.go#L270), thay thế `uc.fashionContract.GetFashionItem` bằng truy vấn trực tiếp qua `uc.fashionItemRepo.GetByID`.

### Bước 2.2: Loại bỏ `fashionContract` trong `OutfitRecommendationUseCase`
- **Chi tiết:**
  1. Sửa file [usecase.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/usecase.go#L26) loại bỏ dependency `fashionContract`. Thay vào đó, inject `fashionItemRepo repositories.IFashionItemRepository`.
  2. Cập nhật constructor `NewOutfitRecommendationUseCase`.
  3. Trong [candidate_helpers.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/candidate_helpers.go#L205), thay thế `uc.fashionContract.ListFashionItems` bằng truy vấn qua `uc.fashionItemRepo.GetByIDs`.

### Bước 2.3: Loại bỏ `quotaContract` trong `SubscriptionUseCase`
- **Chi tiết:**
  1. Trong [subscription_uc.go](file:///internal/modules/subscription/application/usecase/subscription/subscription_uc.go#L37), đổi trường `quotaContract contract.IUserQuotaContract` thành `quotaUseCase uc_interfaces.IUserQuotaUseCase`.
  2. Cập nhật constructor `NewSubscriptionUseCase` để nhận và gán `quotaUseCase`.
  3. Cập nhật dòng code `uc.quotaContract.GetAndResetDailyQuota` thành `uc.quotaUseCase.GetAndResetDailyQuota`. **Lưu ý về kiểu trả về:**
     - Hiện tại method `GetDailyQuota` trong `ISubscriptionUseCase` trả về `*contract.UserSubscriptionDTO`.
     - Sau Phase 1.2, `UserSubscriptionDTO` được định nghĩa tại `application/dto/`, contract sẽ import và re-export type này.
     - Tuy nhiên đúng theo kiến trúc, usecase interface phải import trực tiếp từ `application/dto`. Việc chuyển import này sẽ được thực hiện chính xác trong **Phase 3** khi tái cấu trúc usecase interface và loại bỏ embedding.

---

## PHASE 3: Di chuyển và Tách biệt Contract về Presentation Layer (`presentation/contract/`)
*Mục tiêu: Đưa toàn bộ thư mục `contract/` từ root của module vào bên trong `presentation/contract/`. Triển khai struct Contract bằng cách inject Usecase.*

### Bước 3.1: Refactor Module `aiexecutor`
- **Chi tiết:**
  1. Tạo thư mục mới `internal/modules/aiexecutor/presentation/contract/`.
  2. Di chuyển interface `IAIOrchestrator` từ `aiexecutor/contract/orchestrator.go` sang `aiexecutor/presentation/contract/orchestrator.go`.
  3. Xóa thư mục cũ `aiexecutor/contract/`.
  4. **Tạo interface** `IAIOrchestratorUseCase` trong file mới `aiexecutor/application/interface/usecase/orchestrator_uc.go` với các method sau (lấy nguyên mẫu từ `contract.IAIOrchestrator`):
     - `AnalyzeImage(ctx, imageUrl, prompt) (string, error)`
     - `GenerateEmbeddings(ctx, chunks) ([][]float32, error)`
     - `GenerateChatText(ctx, systemPrompt, userPrompt, options) (string, error)`
     - `GenerateRecommendationText(ctx, systemPrompt, userPrompt, options) (string, error)`
     - `GenerateChatTextStream(ctx, systemPrompt, userPrompt, options) (<-chan string, <-chan error)`
     - **Lưu ý:** `OrchestratorOptions` đã được chuyển về `application/dto/options.go` ở Phase 1.3, nên signature dùng `dto.OrchestratorOptions`.
  5. Thiết lập struct `OrchestratorUseCase` ở tầng usecase implement `IAIOrchestratorUseCase` thay vì `contract.IAIOrchestrator`.
     - Sửa constructor `NewOrchestratorUseCase` để trả về `uc_interfaces.IAIOrchestratorUseCase`.
  6. Tạo file mới `aiexecutor/presentation/contract/orchestrator_contract.go` chứa struct `AIOrchestratorContract` thực hiện implement `contract.IAIOrchestrator` bằng cách inject và gọi `IAIOrchestratorUseCase`.

### Bước 3.2: Refactor Module `identity`
- **Chi tiết:**
  1. Tạo thư mục mới `internal/modules/identity/presentation/contract/`.
  2. Di chuyển `identity/contract/service.go` sang `identity/presentation/contract/user_contract.go` (định nghĩa `IUserContract`).
  3. Xóa thư mục cũ `identity/contract/`.
  4. **Loại bỏ việc nhúng** `contract.IUserContract` trong `IUserUseCase` tại `identity/application/interface/usecase/user_uc.go`.
  5. **Thêm tường minh** 4 method sau vào `IUserUseCase` (sử dụng kiểu từ `application/dto`):
     - `GetByID(ctx, userID) (*dto.UserRes, error)`
     - `GetByIDs(ctx, userIDs) ([]*dto.UserRes, error)`
     - `GetByUsername(ctx, username) (*dto.UserRes, error)`
     - `GetStyleProfile(ctx, userID) (*dto.UserStyleProfileRes, error)`
     - *(Các method này đã được `UserUseCase` struct implement sẵn, chỉ cần thêm vào interface)*
  6. Tạo struct `UserContract` (trong `identity/presentation/contract/user_contract_impl.go`) implement `IUserContract` bằng cách inject và gọi `IUserUseCase`. Mỗi method của `UserContract` đơn giản delegate sang method tương ứng của `IUserUseCase`.

### Bước 3.3: Refactor Module `subscription`
- **Chi tiết:**
  1. Tạo thư mục mới `internal/modules/subscription/presentation/contract/`.
  2. Di chuyển `subscription/contract/service.go` sang `subscription/presentation/contract/subscription_contract.go`.
  3. Xóa thư mục cũ `subscription/contract/`.
  4. **Loại bỏ việc nhúng** các contract interface khỏi các usecase interface tương ứng:
     - **`ISubscriptionUseCase`** (`subscription_usecase.go`): bỏ dòng `contract.IUserSubscriptionContract`
     - **`IUserQuotaUseCase`** (`quota_usecase.go`): bỏ dòng `contract.IUserQuotaContract`
     - **`ISubscriptionPlanUseCase`** (`plan_usecase.go`): bỏ dòng `contract.ISubscriptionPlanContract`
  5. **Thêm tường minh** các method từ contract vào từng usecase interface:
     - **`ISubscriptionUseCase`** thêm 3 method (dùng `dto.UserSubscriptionDTO`, `dto.UserSubscriptionOverviewDTO` từ `application/dto`):
       - `GetUserSubscription(ctx, userID) (*dto.UserSubscriptionDTO, error)`
       - `GetUserSubscriptionOverview(ctx, userID) (*dto.UserSubscriptionOverviewDTO, error)`
       - `GetUserSubscriptionOverviews(ctx, userIDs) (map[uuid.UUID]*dto.UserSubscriptionOverviewDTO, error)`
       - *(Đồng thời sửa method `GetDailyQuota` đang trả `*contract.UserSubscriptionDTO` thành `*dto.UserSubscriptionDTO`)*
     - **`IUserQuotaUseCase`** thêm 3 method (interface hiện tại chỉ có embedding, sau khi bỏ sẽ rỗng):
       - `GetAndResetDailyQuota(ctx, userID) (*dto.UserSubscriptionDTO, error)`
       - `UpdateOutfitQuota(ctx, userID, count) error`
       - `UpdateAiChatQuota(ctx, userID, count) error`
     - **`ISubscriptionPlanUseCase`** thêm 2 method:
       - `GetDefaultSubscriptionPlanID(ctx) (uuid.UUID, error)`
       - `IsPremiumPlan(ctx, planID) (bool, error)`
  6. Tạo file `subscription/presentation/contract/subscription_contract_impl.go` chứa các struct `UserSubscriptionContract`, `SubscriptionPlanContract` và `UserQuotaContract` thực hiện triển khai các interface contract tương ứng, inject các usecase nghiệp vụ để ủy quyền xử lý.

### Bước 3.4: Refactor Module `wardrobe`
- **Chi tiết:**
  1. Tạo thư mục mới `internal/modules/wardrobe/presentation/contract/`.
  2. Di chuyển `wardrobe/contract/service.go` sang `wardrobe/presentation/contract/wardrobe_contract.go`.
  3. Xóa thư mục cũ `wardrobe/contract/`.
  4. **Loại bỏ việc nhúng** `contract.IWardrobeContract` trong `IWardrobeContractUseCase` tại `wardrobe/application/interface/usecase/contract_uc.go`.
  5. **Thêm tường minh** các method từ `contract.IWardrobeContract` vào `IWardrobeContractUseCase` (interface hiện tại chỉ có embedding, sau khi bỏ sẽ rỗng):
     - `CopyItemToUser(ctx, sourceItemID, targetUserID) (*dto.WardrobeItemRes, error)`
     - `UpdateItemStatus(ctx, itemID, status) error`
     - `VerifyItemsForPost(ctx, userID, itemIDs) error`
     - `SetFashionItemID(ctx, itemID, fashionItemID) error`
     - `GetItemsByIDs(ctx, itemIDs) ([]*dto.WardrobeItemRes, error)`
  6. Tạo file `wardrobe/presentation/contract/wardrobe_contract_impl.go` chứa struct `WardrobeContract` triển khai `IWardrobeContract`, inject `IWardrobeUseCase` hoặc `IWardrobeContractUseCase` để thực thi.
     - **Chú ý:** Cần xác định usecase interface phù hợp để inject: có thể tái sử dụng `IWardrobeContractUseCase` (sau khi đã thêm method tường minh) hoặc tạo interface usecase riêng.

### Bước 3.5: Refactor Module `fashion`
- **Chi tiết:**
  1. Tạo thư mục mới `internal/modules/fashion/presentation/contract/`.
  2. Di chuyển `fashion/contract/service.go` sang `fashion/presentation/contract/fashion_contract.go`.
  3. Xóa thư mục cũ `fashion/contract/`.
  4. Chuyển struct `FashionContract` hiện tại từ `fashion/contract/service_impl.go` sang `fashion/presentation/contract/fashion_contract_impl.go` **(nhưng chưa sửa nội dung)**.
  5. **Tạo UseCase interface mới** để `FashionContract` có thể inject:
     - **`IFashionItemUseCase`** trong file mới `fashion/application/interface/usecase/fashion_item_uc.go` với 2 method:
       - `GetFashionItem(ctx, id) (*dto.FashionItemRes, error)`
       - `ListFashionItems(ctx, ids) ([]*dto.FashionItemRes, error)`
     - **`ICategoryUseCase`** trong file mới `fashion/application/interface/usecase/category_uc.go` với 8 method:
       - `GetCategory(ctx, id) (*dto.CategoryRes, error)`
       - `ListCategories(ctx, ids) ([]*dto.CategoryRes, error)`
       - `GetAllCategories(ctx) ([]*dto.CategoryRes, error)`
       - `GetCategoryBySlug(ctx, slug) (*dto.CategoryRes, error)`
       - `GetCategoryByName(ctx, name) (*dto.CategoryRes, error)`
       - `CreateCategory(ctx, input) (*dto.CategoryRes, error)`
       - `UpdateCategory(ctx, id, input) (*dto.CategoryRes, error)`
       - `DeleteCategory(ctx, id) error`
  6. **Tạo UseCase struct implementation**:
     - Tạo file `fashion/application/usecase/fashion_item/fashion_item_uc.go` chứa struct `FashionItemUseCase` implement `IFashionItemUseCase`, inject `IFashionItemRepository`.
     - Tạo file `fashion/application/usecase/category/category_uc.go` chứa struct `CategoryUseCase` implement `ICategoryUseCase`, inject `ICategoryRepository`.
     - *(Các logic mapping hiện tại từ `FashionContract` sẽ được chuyển vào các usecase này)*
  7. **Sửa struct `FashionContract`** để inject `IFashionItemUseCase` và `ICategoryUseCase` thay vì repository trực tiếp. Mỗi method của `FashionContract` sẽ delegate sang usecase tương ứng:
     - `GetFashionItem` → `fashionItemUC.GetFashionItem`
     - `ListFashionItems` → `fashionItemUC.ListFashionItems`
     - `GetCategory` → `categoryUC.GetCategory`
     - `ListCategories` → `categoryUC.ListCategories`
     - `GetAllCategories` → `categoryUC.GetAllCategories`
     - `GetCategoryBySlug` → `categoryUC.GetCategoryBySlug`
     - `GetCategoryByName` → `categoryUC.GetCategoryByName`
     - `CreateCategory` → `categoryUC.CreateCategory`
     - `UpdateCategory` → `categoryUC.UpdateCategory`
     - `DeleteCategory` → `categoryUC.DeleteCategory`
     - Giữ lại `eventPublisher event.IEventPublisher` vì đây là shared infrastructure, không phải repository.

### Bước 3.6: Refactor Module `brand`
- **Chi tiết:**
  1. Tạo thư mục mới `internal/modules/brand/presentation/contract/`.
  2. Di chuyển `brand/contract/service.go` sang `brand/presentation/contract/brand_contract.go`.
  3. Chuyển struct `BrandContract` từ `brand/contract/service_impl.go` sang `brand/presentation/contract/brand_contract_impl.go`.
  4. Xóa thư mục cũ `brand/contract/`.

---

## PHASE 4: Giải quyết vi phạm import chéo lớp Private (Rule 1.1)
*Mục tiêu: Đảm bảo cross-module chỉ tương tác qua `contract` và `dto` từ `application/dto`, tuyệt đối không truy cập trực tiếp entities, repositories, constants, errors của module khác.*

### Bước 4.1: Sửa vi phạm trong `aiexecutor/infrastructure/persistence/ai_cost_repo_impl.go`
- **Chi tiết:**
  1. Usecase ở tầng Application của `aiexecutor` sẽ gọi sang `subscription/contract` để lấy thông tin Subscription DTO trước khi gọi repo.
  2. Chuyển thông tin plan của user (như `SubscriptionPlanID`) thành tham số truyền vào hàm `ResolvePolicy` của repo.
  3. Sửa lại repo impl để loại bỏ việc truy vấn bảng `UserSubscription` và xóa import `"smart-wardrobe-be/internal/modules/subscription/domain/entities"`.

### Bước 4.2: Thay thế việc gọi trực tiếp domain entities của module khác trong `fashion`
- **Chi tiết:**
  1. Tại các file mappers, helpers và workflows của `fashion`, thay thế các biến thực thể của `wardrobe/entities` bằng DTO tương ứng (nhận từ `wardrobe/contract`).
  2. Thay thế `brand/entities` bằng `dto.BrandItemRes` (nhận từ `brand/contract`).
  3. Xóa tất cả các import liên quan đến thực thể bên ngoài module.
  4. **Lưu ý đối với các file phức tạp (ranking algorithms):** Các file trong `fashion/application/usecase/ai/recommendation/ranking/*.go` sử dụng nhiều field từ `wardrobe/domain/entities` và `brand/domain/entities` trong thuật toán. Việc thay thế các entity này bằng DTO đòi hỏi phải có DTO đầy đủ field tại `wardrobe/application/dto/` và `brand/application/dto/`. Xem chi tiết tại **[Phase 6 — Tạo DTO cho Wardrobe & Brand](20260704_refactor_plan_phase6_dto_migration.md)**.

### Bước 4.3: Refactor các repository chéo module trong `fashion`
- **Chi tiết:**
  1. Sửa file [interfaces.go](file:///internal/modules/fashion/domain/repositories/interfaces.go) để loại bỏ việc import repositories của `brand` và `wardrobe`.
  2. Thay thế các logic cần thực thi chéo bằng cách lấy dữ liệu qua contract ở tầng Usecase và truyền tham số dạng Slice/Map xuống tầng dưới.
  3. Việc này phụ thuộc vào DTO từ `wardrobe/application/dto/` và `brand/application/dto/` đã được định nghĩa đầy đủ — xem **[Phase 6](20260704_refactor_plan_phase6_dto_migration.md)**.

### Bước 4.4: Sửa vi phạm import Constants, Errors của module khác
- **Chi tiết:**
  1. *Đối với lỗi (Errors):* Module `fashion` sẽ tự định nghĩa các lỗi ứng dụng (như lỗi vượt quá quota) ngay tại package errors nội bộ của nó (ví dụ: `fashion/application/errors/`) để sử dụng nội bộ, không import trực tiếp `subscription/application/errors` hay `wardrobe/application/errors`.
  2. *Đối với hằng số (Constants):* Di chuyển các hằng số dùng chung toàn hệ thống (ví dụ: `roleslug` từ `identity`, `benefitresolution` từ `brand`) vào thư mục tập trung `internal/shared/domain/constants/` để tất cả các module cùng import.
  3. Cập nhật lại các import hướng về `shared`.

---

## PHASE 5: Tái tạo Wire DI & Xác minh tính đúng đắn

### Bước 5.0: Baseline kiểm thử trước khi refactor
- **Chi tiết:**
  1. Chạy `go vet` và `make test` để ghi nhận trạng thái hiện tại:
     ```bash
     go vet ./...
     make test
     ```
  2. Lưu kết quả test hiện tại làm baseline. Nếu test fail trước khi refactor, cần fix trước.

### Bước 5.1: Cập nhật Provider Set & Chạy Wire
- **Chi tiết:**
  1. Cập nhật các file `provider.go` ở mỗi module để đảm bảo các contract implementation mới ở tầng presentation được đăng ký chính xác vào DI container:
     - **identity:** `wire.Bind(new(contract.IUserContract), new(presentation_contract.UserContract))` (thay vì trỏ đến usecase)
     - **subscription:** `wire.Bind(...)` cho cả 3 contract interface trỏ đến các struct mới trong `presentation/contract/`
     - **wardrobe:** `contractuc.NewWardrobeContractUseCase` → `presentation_contract.NewWardrobeContract`
     - **brand:** `contract.NewBrandContract` → `presentation_contract.NewBrandContract`
     - **fashion:** `contract.NewFashionContract` → `presentation_contract.NewFashionContract`
     - **aiexecutor:** thêm `presentation_contract.NewAIOrchestratorContract` vào provider set
  2. Chạy lệnh tái tạo code DI:
     ```bash
     make wire
     ```

### Bước 5.2: Chạy kiểm thử tự động
- **Chi tiết:**
  1. Chạy toàn bộ các test case có sẵn trong dự án để đảm bảo không phát sinh lỗi liên kết hoặc cú pháp:
     ```bash
     make test
     ```

---

## PHASE 6 (Bổ sung): Tạo DTO đầy đủ cho Wardrobe & Brand

> Phase này được tách thành file riêng: [20260704_refactor_plan_phase6_dto_migration.md](20260704_refactor_plan_phase6_dto_migration.md)

Phase 6 giải quyết việc tạo DTO mới trong `wardrobe/application/dto/` và `brand/application/dto/` chứa đầy đủ field cần thiết cho các thuật toán ranking của module `fashion`, nhằm thay thế hoàn toàn việc import trực tiếp `domain/entities` của module khác (vi phạm Rule 1.1).

**Thời điểm thực hiện:** Sau khi Phase 5 hoàn thành và toàn bộ hệ thống hoạt động ổn định với kiến trúc mới.
