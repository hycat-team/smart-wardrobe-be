# Báo Cáo Vi Phạm Kiến Trúc (Architectural Violation Report) - Cập Nhật Quy Tắc Contract

Báo cáo này được cập nhật theo tinh chỉnh quy tắc mới nhất từ phía bạn:
1. **Quy tắc về Vị trí và Thiết kế Contract:**
   - Contract bản chất là sự tương tác với bên ngoài (cho phép scale lên gRPC sau này), do đó nó phải là package nằm trong tầng presentation: **`presentation/contract/`** thay vì là một layer riêng ở root của module.
   - Tầng `presentation/contract/` tự định nghĩa interface và tự implement interface đó.
   - Struct triển khai của Contract **phải inject các UseCase** (tương tự như cách HTTP Handler đang làm), **tuyệt đối không được inject hoặc gọi trực tiếp tầng Repository/Database**.
2. **Quy tắc về vị trí DTO:** Tất cả các DTO cần được gom tập trung về thư mục `application/dto` thay vì để rải rác ở `application/` hoặc `contract/`.
3. **Quy tắc về Inject Contract nội bộ:** Tuyệt đối không được inject contract interface của module đó vào constructor hoặc struct nội bộ của chính module đó để sử dụng.

---

## 1. Vi Phạm Quy Tắc: Vị trí của Contract đặt sai tầng (Đang nằm ở root của Module)
> **Quy tắc:** Thư mục `contract/` không được nằm ở root của module. Nó phải được di chuyển vào bên trong tầng presentation: `presentation/contract/`.

Dưới đây là các module đang đặt sai vị trí thư mục contract:
- **Module `aiexecutor`:** [aiexecutor/contract/](file:///internal/modules/aiexecutor/contract/) -> Cần chuyển sang `aiexecutor/presentation/contract/`
- **Module `brand`:** [brand/contract/](file:///internal/modules/brand/contract/) -> Cần chuyển sang `brand/presentation/contract/`
- **Module `fashion`:** [fashion/contract/](file:///internal/modules/fashion/contract/) -> Cần chuyển sang `fashion/presentation/contract/`
- **Module `identity`:** [identity/contract/](file:///internal/modules/identity/contract/) -> Cần chuyển sang `identity/presentation/contract/`
- **Module `subscription`:** [subscription/contract/](file:///internal/modules/subscription/contract/) -> Cần chuyển sang `subscription/presentation/contract/`
- **Module `wardrobe`:** [wardrobe/contract/](file:///internal/modules/wardrobe/contract/) -> Cần chuyển sang `wardrobe/presentation/contract/`

---

## 2. Vi Phạm Quy Tắc: Thiết kế Implementation của Contract (Gọi trực tiếp Repo hoặc Implement trực tiếp ở Usecase)
> **Quy tắc:** Struct triển khai Contract phải inject use case (không được inject repository trực tiếp). Tầng Usecase ở `application` không được trực tiếp implement contract.

Dưới đây là các vi phạm về mặt thiết kế lớp triển khai:

### 2.1. Contract gọi trực tiếp Repository
- **[service_impl.go](file:///internal/modules/fashion/contract/service_impl.go#L13)** (Module `fashion`):
  - Vi phạm: Struct `FashionContract` trực tiếp inject `repositories.IFashionItemRepository` và `repositories.ICategoryRepository` thay vì gọi thông qua Usecase.
  - *Giải pháp đúng:* Định nghĩa các Usecase nghiệp vụ cần thiết ở tầng `application/interface/usecase` (ví dụ `IFashionItemUseCase` hoặc bổ sung vào `IFashionUseCase`), sau đó inject các Usecase này vào `FashionContract` trong tầng `presentation/contract`.

### 2.2. Usecase ở tầng Application trực tiếp implement Contract
- **[user_uc.go](file:///internal/modules/identity/application/usecase/user_uc.go#L1)** (Module `identity`):
  - Vi phạm: Struct `UserUseCase` ở tầng usecase trực tiếp implement `contract.IUserContract`.
  - *Giải pháp đúng:* Struct implementation riêng của contract phải nằm ở `identity/presentation/contract/user_contract.go` và inject `IUserUseCase` để sử dụng.
- **[subscription_uc.go](file:///internal/modules/subscription/application/usecase/subscription/subscription_uc.go#L23)** và **[quota_uc.go](file:///internal/modules/subscription/application/usecase/quota/quota_uc.go#L18)** (Module `subscription`):
  - Vi phạm: Cả `SubscriptionUseCase` và `UserQuotaUseCase` đều trực tiếp implement các contract tương ứng.
  - *Giải pháp đúng:* Tách các implementation của contract ra các struct riêng đặt tại `subscription/presentation/contract/` và inject usecase tương ứng vào.
- **[contract_uc.go](file:///internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go#L22)** (Module `wardrobe`):
  - Vi phạm: Struct `WardrobeContractUseCase` dùng để implement contract nằm ở tầng application/usecase.
  - *Giải pháp đúng:* Di chuyển struct này về `wardrobe/presentation/contract/` và inject usecase nghiệp vụ tương ứng vào.
- **[service.go](file:///internal/modules/aiexecutor/application/usecase/orchestrator/service.go#L31)** (Module `aiexecutor`):
  - Vi phạm: Struct `OrchestratorUseCase` ở tầng usecase trực tiếp implement `contract.IAIOrchestrator`.
  - *Giải pháp đúng:* Tách struct implementation của contract ra đặt tại `aiexecutor/presentation/contract/` và inject usecase nghiệp vụ vào.

---

## 3. Vi Phạm Quy Tắc: Inject Contract của chính Module vào nội bộ
> **Quy tắc:** Tuyệt đối không được inject contract interface của module đó vào constructor hoặc struct nội bộ của chính module đó để sử dụng.

Dưới đây là các vị trí vi phạm thực tế:

### Module: `fashion`
- **[usecase.go (chat)](file:///internal/modules/fashion/application/usecase/ai/chat/usecase.go#L26)**:
  - Vi phạm: Inject `fashionContract fashion_contract.IFashionContract` vào struct `WardrobeChatUseCase` và constructor `NewWardrobeChatUseCase`.
  - Sử dụng tại **[workflows.go (chat)](file:///internal/modules/fashion/application/usecase/ai/chat/workflows.go#L270)**: Gọi `uc.fashionContract.GetFashionItem(ctx, ...)` để lấy dữ liệu.
  - *Giải pháp:* Nên gọi trực tiếp các repository nội bộ hoặc usecase nội bộ của module `fashion` thay vì đi vòng qua Contract interface của chính mình.
- **[usecase.go (recommendation)](file:///internal/modules/fashion/application/usecase/ai/recommendation/usecase.go#L26)**:
  - Vi phạm: Inject `fashionContract fashion_contract.IFashionContract` vào struct `OutfitRecommendationUseCase` và constructor `NewOutfitRecommendationUseCase`.
  - Sử dụng tại **[candidate_helpers.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/candidate_helpers.go#L205)**: Gọi `uc.fashionContract.ListFashionItems(ctx, ...)`.
  - *Giải pháp:* Thay thế bằng cách truy vấn qua repository nội bộ `repositories.IFashionItemRepository`.

### Module: `subscription`
- **[subscription_uc.go](file:///internal/modules/subscription/application/usecase/subscription/subscription_uc.go#L37)**:
  - Vi phạm: Inject `quotaContract contract.IUserQuotaContract` vào struct `SubscriptionUseCase` và constructor `NewSubscriptionUseCase`.
  - Sử dụng tại **[subscription_uc.go](file:///internal/modules/subscription/application/usecase/subscription/subscription_uc.go#L73)**: Gọi `uc.quotaContract.GetAndResetDailyQuota(ctx, userID)`.
  - *Giải pháp:* Nên gọi trực tiếp qua `UserQuotaUseCase` (hoặc interface của usecase) thay vì gọi qua contract.

---

## 4. Vi Phạm Quy Tắc: Import các layer private (Entities/Repos/Persistence) từ Module khác
> **Quy tắc:** Các entities trong mỗi module là private nội bộ. Một module tuyệt đối không được import `domain/entities`, `domain/repositories` hay `infrastructure/persistence` của module khác.

Dưới đây là các vi phạm nghiêm trọng khi import chéo các thực thể private:

### 4.1. Import chéo Domain Entities từ Module khác
- **[ai_cost_repo_impl.go](file:///internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl.go#L12)** (Module `aiexecutor`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/subscription/domain/entities"` và truy vấn trực tiếp model `UserSubscription` từ Database.
- **[recommendation.go](file:///internal/modules/fashion/application/mapper/recommendation.go#L14)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[helper.go](file:///internal/modules/fashion/application/usecase/ai/chat/helper.go#L12)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[workflows.go](file:///internal/modules/fashion/application/usecase/ai/chat/workflows.go#L18)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[workflows_test.go](file:///internal/modules/fashion/application/usecase/ai/chat/workflows_test.go#L13)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[candidate_helpers.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/candidate_helpers.go#L13)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/brand/domain/entities"`
- **[candidate_helpers.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/candidate_helpers.go#L24)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[fallback_hsl.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/ranking/fallback_hsl.go#L13)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[fallback_hsl_test.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/ranking/fallback_hsl_test.go#L10)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[pool_expansion.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/ranking/pool_expansion.go#L10)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[scoring.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/ranking/scoring.go#L14)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[scoring_test.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/ranking/scoring_test.go#L10)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[synthesis_test.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/synthesis/synthesis_test.go#L14)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`
- **[types.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/types/types.go#L7)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/brand/domain/entities"`
- **[types.go](file:///internal/modules/fashion/application/usecase/ai/recommendation/types/types.go#L11)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"`

### 4.2. Import chéo Domain Repositories từ Module khác
- **[interfaces.go](file:///internal/modules/fashion/domain/repositories/interfaces.go#L7)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/brand/domain/repositories"`
- **[interfaces.go](file:///internal/modules/fashion/domain/repositories/interfaces.go#L10)** (Module `fashion`):
  - Vi phạm: Import `"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"`

---

## 5. Báo Cáo các DTO để sai vị trí (Cần tập trung về `application/dto`)
> **Quy tắc:** Các DTO cần tập trung định nghĩa ở `application/dto` thay vì để rải rác ở `contract` hoặc các file khác.

Dưới đây là danh sách các DTO đang nằm sai vị trí cần refactor di chuyển:

### 5.1. DTO nằm ở package `contract`
Các DTO này cần được di chuyển về `application/dto` và để `contract` import chúng ngược lại:
1. **Module `fashion`**:
   - Các DTO: `FashionItemRes`, `CategoryRes`, `FashionAnalyzeJobDTO`, `CreateCategoryInput`, `UpdateCategoryInput`.
   - Đang nằm tại: **[service.go](file:///internal/modules/fashion/contract/service.go)**
   - *Đề xuất:* Chuyển toàn bộ các struct DTO trên sang file mới `fashion/application/dto/contract_dtos.go`.
2. **Module `subscription`**:
   - Các DTO: `UserSubscriptionDTO`, `UserSubscriptionOverviewDTO`.
   - Đang nằm tại: **[dto.go](file:///internal/modules/subscription/contract/dto.go)**
   - *Đề xuất:* Chuyển sang `subscription/application/dto/quota_dtos.go` và xóa bỏ file `subscription/contract/dto.go`.
3. **Module `aiexecutor`**:
   - Các DTO/Options: `OrchestratorOptions`.
   - Đang nằm tại: **[orchestrator.go](file:///internal/modules/aiexecutor/contract/orchestrator.go#L9)**
   - *Đề xuất:* Chuyển sang `aiexecutor/application/dto/options.go`.

### 5.2. DTO bị trùng lặp và không sử dụng (Dead Code)
- **Module `identity`**:
  - File **[dto.go](file:///internal/modules/identity/contract/dto.go)** định nghĩa các struct `UserRes` và `UserStyleProfileRes`. Tuy nhiên, interface thực tế trong `identity/contract/service.go` lại import và dùng DTO từ `identity/application/dto`.
  - *Đề xuất:* Xóa bỏ file **[dto.go](file:///internal/modules/identity/contract/dto.go)** này vì hoàn toàn không dùng đến.
