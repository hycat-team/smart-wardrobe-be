# Tasks: Refactor Contract Purity & Xử Lý Detour Call + Deadcode

**Input**: Design documents from `specs/015-contract-purity-refactor/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/internal-contracts.md, quickstart.md

**Branch**: `015-contract-purity-refactor` | **Spec**: [spec.md](spec.md)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Khởi tạo audit script và doc nền để verify trước/sau

- [ ] T001 Tạo audit script `scripts/audit-contract-purity.ps1` (logic: đếm detour chain 7, pure violation method-level 2 contract, deadcode 12, dead wiring 1 bằng Select-String/rg) — file `scripts/audit-contract-purity.ps1`
- [ ] T002 [P] Tạo `docs/system-design/contract-purity-pattern.md` bản nháp (định nghĩa Pure Contract Usecase, phân biệt usecase thường vs contract usecase, before/after, quy tắc caller enrich bằng IndexBy/GroupBy, nil risk) — file `docs/system-design/contract-purity-pattern.md`
- [ ] T003 Chạy baseline audit (ghi nhận 7 detour / 2 contract leaky method-level / 12 deadcode / 1 dead wiring) vào `specs/015-contract-purity-refactor/checklists/baseline.md` — file `specs/015-contract-purity-refactor/checklists/baseline.md`

**Checkpoint**: Có script audit và doc nháp; baseline đã ghi nhận.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Chuẩn bị để thuần hóa, không block handler

**⚠️ CRITICAL**: Phải hoàn tất trước khi bất kỳ User Story nào bắt đầu

- [ ] T004 Xác nhận `internal/di/wire_gen.go` hiện build pass (`make wire && make build`) — file `internal/di/wire_gen.go`
- [ ] T005 [P] Đọc `docs/system-design/design-pattern.md` và ghi note tuân thủ — file `specs/015-contract-purity-refactor/research.md`
- [ ] T005a [P] Verify method-level pure cho `CategoryContract.GetCategory` và `Subscription` 8 delegate (đã pure) bằng `rg` — expected: `CategoryContract.GetCategory 1/1 pure` + `Subscription 8/8 pure` → ghi "Category/Subscription no task needed (intentional) — verified pure method-level" vào `specs/015-contract-purity-refactor/checklists/affected-files.md` — file `specs/015-contract-purity-refactor/checklists/affected-files.md`
- [ ] T006 Tạo backup danh sách file sẽ sửa — file `specs/015-contract-purity-refactor/checklists/affected-files.md`

**Checkpoint**: Foundation ready — tuần tự P1→P1→P2→P3 theo clarification.

---

## Phase 3: User Story 2 - Thuần hóa Contract Usecase (Priority: P1) 🎯 MVP — Root Cause

**Goal**: Wardrobe contract uc (4/8 alive leaky) sửa trực tiếp cho thuần + Brand contract uc mới tách riêng cho thuần (5 method) — handler `BrandItemUseCase` giữ nguyên enrich

**Independent Test**: Đọc `WardrobeContractUseCase` → chỉ `Repo` + mapper; đọc `BrandItemContractUseCase` mới → chỉ `brandRepo`/`brandItemRepo`/`customerRepo`/`benefitUC` không `fashionContract`/`userBriefContract`; đồng thời verify `BrandItemUseCase` handler vẫn enrich; audit `pure: 0`; `make wire && make build` pass

### Implementation cho US2

- [ ] T007 [US2] Sửa `WardrobeContractUseCase` trong `internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go` — bỏ field `userSubContract`/`fashionContract` khỏi struct `WardrobeContractUseCase:24-28` và constructor `NewWardrobeContractUseCase:30-39` — file `internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go`
- [ ] T007a [US2] Thêm field `FashionItemID *uuid.UUID \`json:"fashionItemId,omitempty"\`` (camelCase — Hiến pháp Code Style) vào `WardrobeItemRes` trong `internal/modules/wardrobe/application/dto/wardrobe.go:54-67` và cập nhật `mapper.MapToWardrobeItemRes` trong `internal/modules/wardrobe/application/mapper/wardrobe.go:39-55` để map `FashionItemID` từ `item.FashionItemID` kể cả khi `FashionItem == nil` — file `internal/modules/wardrobe/application/dto/wardrobe.go` và `internal/modules/wardrobe/application/mapper/wardrobe.go`
- [ ] T008 [US2] Thuần hóa 4 method alive leaky của `WardrobeContractUseCase` — bỏ enrich `fashionContract`: `GetWardrobeItemsByUserID:149-179` (bỏ ListFashionItems 164), `GetActiveWardrobeItemsByUserID:183-213` (bỏ 198), `GetFailedItemsForCleanup:247-277` (bỏ 262), `GetStaleProcessingItems:283-334` (bỏ GetStaleProcessingInfo 299 + ListFashionItems 309, chỉ filter wardrobe) — trả `FashionItem == nil` + `FashionItemID` — file `internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go`
- [ ] T009 [US2] Cập nhật `internal/modules/wardrobe/provider.go` bỏ inject `fashionContract`/`userSubContract` cho `WardrobeContractUseCase` — file `internal/modules/wardrobe/provider.go`
- [ ] T010 [US2] Tạo mới `BrandItemContractUseCase` pure ở `internal/modules/brand/application/usecase/item/contractuc/contract_uc.go` (interface `IBrandItemContractUseCase` ở `brand/application/interface/usecase/brand_item_contract_uc.go`) — KHÔNG sửa `BrandItemUseCase` hiện tại (`usecase.go` + `helper.go` giữ nguyên cho handler). Implement 5 method pure không `fashionContract`/`userBriefContract`: `ListEligibleBrandItemsForStyling`, `CheckBrandItemEligibility`, `GetBrandItemByIDDirect`, `GetBrandItemsByFashionItemIDs`, `GetBrandItemsByIDs` (mapper thuần riêng, không gọi `fashionContract`) — trả `FashionItem == nil` + `FashionItemID` — file `internal/modules/brand/application/usecase/item/contractuc/contract_uc.go` + `mapper.go`
- [ ] T010a [US2] Cập nhật `BrandContract` (`internal/modules/brand/presentation/contract/brand_contract_impl.go`) chuyển inject từ `IBrandItemUseCase` sang `IBrandItemContractUseCase` cho 5 method; wire: thêm ProviderSet `contractuc.NewBrandItemContractUseCase` vào `brand/application.ProviderSet` và `brand.ProviderSet` — file `internal/modules/brand/presentation/contract/brand_contract_impl.go`, `internal/modules/brand/application/provider.go`, `internal/modules/brand/provider.go`
- [ ] T011 [US2] Thêm unit test verify pure cho `WardrobeContractUseCase` 4 method (`TestWardrobeContractUseCase_Pure_NoCrossModuleCall`) và `BrandItemContractUseCase` 5 method mới (`TestBrandItemContractUseCase_Pure_NoCrossModuleCall` — mock assert `fashionContract`/`userBriefContract` times(0)), đồng thời verify `BrandItemUseCase` handler vẫn enrich — file `internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc_test.go` và `internal/modules/brand/application/usecase/item/contractuc/contract_uc_test.go`
- [ ] T011a [US2] Chạy `make wire` và `make build` verify sau US2 — file `internal/di/wire_gen.go`

**Checkpoint**: US2 xong → audit `pure: 0` method-level; build pass. Có thể commit PR riêng. **Lưu ý nil**: caller sẽ nhận `FashionItem == nil` — phải làm US1 ngay sau.

---

## Phase 4: User Story 1 - Ép caller tự enrich, tránh nil (Priority: P1)

**Goal**: 7 chuỗi detour về 0, caller tự gọi `fashionContract` + `IndexBy`/`GroupBy` để bù enrich

**Independent Test**: Trigger `FashionWorkerUseCase.CleanupFailedItems` và `WardrobeChatUseCase.buildChatGenerationInput` → assert chỉ 1 call tới wardrobeContract pure + 1 call tới fashionContract riêng; audit `detour: 0`; không còn `FashionItem == nil` do thiếu enrich

**Phụ thuộc**: Cần US2 hoàn tất (contract đã thuần, FashionItem nil nếu không enrich)

### Implementation cho US1

- [ ] T012 [US1] Sửa `FashionWorkerUseCase.CleanupFailedItems` trong `internal/modules/fashion/application/usecase/worker/worker_uc.go:143` — sau `wardrobeContract.GetFailedItemsForCleanup` pure (`FashionItem == nil`, có `FashionItemID` mới) **chỉ gọi `fashionContract.GetFashionItem(ctx, FashionItemID)`** để lấy `ImagePublicID` cho `DeleteImage` (KHÔNG dùng `fashionItemRepo` trực tiếp — tuân Hiến pháp §II chỉ gọi qua `presentation/contract`); sửa `if item.FashionItem != nil` thành `if item.FashionItemID != nil` — file `internal/modules/fashion/application/usecase/worker/worker_uc.go`
- [ ] T013 [US1] Sửa `FashionWorkerUseCase.RecoverStaleProcessingItems` trong `internal/modules/fashion/application/usecase/worker/worker_uc.go:208` — sau `GetStaleProcessingItems` pure, bỏ check `FashionItem == nil` (luôn true), thay bằng: extract `FashionItemID` từ `WardrobeItemRes`, gọi `fashionContract.GetFashionItem` để check `ProcessingRetryCount`/`Stale` (KHÔNG gọi repo trực tiếp) — file `internal/modules/fashion/application/usecase/worker/worker_uc.go`
- [ ] T014 [US1] Sửa `WardrobeChatUseCase.buildChatGenerationInput` trong `internal/modules/fashion/application/usecase/ai/chat/workflows.go:249` — sau `wardrobeContract.GetWardrobeItemsByUserID` pure tự gọi `fashionContract.ListFashionItems` với IDs và `IndexBy` để bù `FashionItem.Category`/`Style` — file `internal/modules/fashion/application/usecase/ai/chat/workflows.go`
- [ ] T015 [P] [US1] Sửa `WardrobeChatUseCase.buildChatGenerationInput` trong `internal/modules/fashion/application/usecase/ai/chat/workflows.go:259` — sau `brandContract.ListEligibleBrandItemsForStyling` pure (FashionItem nil) tự gọi `fashionContract.ListFashionItems` — file `internal/modules/fashion/application/usecase/ai/chat/workflows.go`
- [ ] T016 [P] [US1] Sửa `OutfitRecommendationUseCase.validateAndGetActiveItems` trong `internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go:59` — sau `wardrobeContract.GetActiveWardrobeItemsByUserID` pure tự enrich — file `internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go`
- [ ] T017 [P] [US1] Sửa `OutfitRecommendationUseCase.filterCandidates` trong `internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go:108` — sau `brandContract.ListEligibleBrandItemsForStyling` pure tự enrich — file `internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go`
- [ ] T018 [US1] Chạy `make lint` và `make test` verify sau US1, audit `detour: 0` — file `internal/modules/fashion/application/usecase/worker/worker_uc.go` (và các file trên)

**Checkpoint**: US1 xong → audit `detour: 0`; không còn nil do thiếu enrich; test pass.

---

## Phase 5: User Story 3 - Dọn dẹp Contract Deadcode (Priority: P2)

**Goal**: 12 method deadcode bị xóa (trong đó 3 trùng leaky)

**Independent Test**: Grep 12 method → 0 caller ngoài impl/test; `make build` + `make test` pass

### Implementation cho US3

- [ ] T019 [US3] Xóa 7 method deadcode khỏi `IWardrobeContract` trong `internal/modules/wardrobe/presentation/contract/wardrobe_contract.go` (`CopyItemToUser`, `VerifyItemsForPost`, `GetItemsByIDs`, `GetUserFashionItemIDs`, `MarkProcessingFailed`, `MarkProcessingNeedsReview`, `CompleteProcessingSuccess`) — file `internal/modules/wardrobe/presentation/contract/wardrobe_contract.go`
- [ ] T020 [US3] Xóa impl tương ứng 7 method trong `internal/modules/wardrobe/presentation/contract/wardrobe_contract_impl.go` và trong `WardrobeContractUseCase` (`contract_uc.go`) nếu không còn dùng nội bộ — file `internal/modules/wardrobe/presentation/contract/wardrobe_contract_impl.go` và `internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go`
- [ ] T021 [P] [US3] Xóa 2 method deadcode khỏi `IBrandContract` trong `internal/modules/brand/presentation/contract/brand_contract.go` (`CheckBrandFeatureAccess`, `UpdateBrandItemStatus`) và impl — file `internal/modules/brand/presentation/contract/brand_contract.go` và `internal/modules/brand/presentation/contract/brand_contract_impl.go`
- [ ] T022 [P] [US3] Xóa 3 method deadcode khỏi subscription: `IUserSubscriptionContract.GetUserSubscription` và `ISubscriptionPlanContract.GetDefaultSubscriptionPlanID`, `IsPremiumPlan` (nếu rỗng xóa cả contract) + impl — file `internal/modules/subscription/presentation/contract/subscription_contract.go` và `subscription_contract_impl.go`
- [ ] T023 [US3] Cập nhật test fake (`fakeEmptyWardrobeContract` v.v.) xóa method deadcode — file `internal/modules/fashion/application/usecase/ai/chat/workflows_test.go` và các `_test.go` liên quan
- [ ] T024 [US3] Chạy `make wire` và `make build` verify sau US3 — file `internal/di/wire_gen.go`

**Checkpoint**: US3 xong → audit `deadcode: 0`; build pass.

---

## Phase 6: User Story 4 - Sửa Dead Wiring (Priority: P3)

**Goal**: 1 dead wiring bị xóa

**Independent Test**: Struct `WardrobeItemUseCase` không còn `aiService`; `make build` pass

### Implementation cho US4

- [ ] T025 [US4] Xóa field `aiService aiexecutor_contract.IAIOrchestrator` khỏi struct `WardrobeItemUseCase:41` và param constructor `:57` trong `internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_read.go` — file `internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_read.go`
- [ ] T026 [US4] Xóa provider tương ứng khỏi `internal/modules/wardrobe/provider.go` — file `internal/modules/wardrobe/provider.go`
- [ ] T027 [US4] Chạy `make wire` và `make build` verify sau US4 — file `internal/di/wire_gen.go`

**Checkpoint**: US4 xong → dead wiring 0; build pass.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Hoàn thiện doc, constitution, verify tổng thể

- [ ] T028 Hoàn thiện `docs/system-design/contract-purity-pattern.md` (thêm ví dụ before/after thuần hóa trực tiếp, nil risk, audit guide method-level) — file `docs/system-design/contract-purity-pattern.md`
- [ ] T029 Cập nhật `.specify/memory/constitution.md` §II thêm "Pure Contract Usecase Principle" và link tới `docs/system-design/contract-purity-pattern.md`, bump version `1.7.0 → 1.8.0 (MINOR)` + Sync Impact Report theo Governance — file `.specify/memory/constitution.md`
- [ ] T030 [P] Chạy audit script tổng hợp verify 0/0/0/0 cho 4 metric — file `scripts/audit-contract-purity.ps1`
- [ ] T031 [P] Chạy `make lint` toàn repo — file `Makefile`
- [ ] T032 [P] Chạy `make test` toàn repo — file `internal/...`
- [ ] T033 Benchmark NFR-001 (dự đoán, không chạy thật): đếm số lần gọi `fashionContract` trước/sau cho flow `GetWardrobeItemsByUserID`/`GetActiveWardrobeItemsByUserID` (chat/recommendation) và `GetFailedItemsForCleanup`/`GetStaleProcessingItems` — nếu dự đoán tăng >10% thì gộp batch `ListFashionItems` — file `internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go`
- [ ] T034 Chạy `quickstart.md` validation 4 case manual (upload, retry, cleanup, outfit) — file `specs/015-contract-purity-refactor/quickstart.md`
- [ ] T035 Cleanup `docs/temp/` và cập nhật `specs/015-contract-purity-refactor/checklists/requirements.md` — file `specs/015-contract-purity-refactor/checklists/requirements.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — có thể bắt đầu ngay
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS tất cả user stories
- **User Stories (Phase 3-6)**: Depend on Foundational
  - **US2 (Pure — Phase 3) phải hoàn tất trước US1 (Caller — Phase 4)** vì US1 cần contract thuần (FashionItem nil nếu không enrich)
  - US3 Deadcode có thể song song với US2 nhưng khuyến nghị sau US2 để tránh conflict file contract (có 3 method trùng leaky)
  - US4 Dead wiring độc lập, có thể làm bất kỳ lúc nào sau Foundational
- **Polish (Phase 7)**: Depends on tất cả US hoàn tất

### User Story Dependencies

- **US2 (P1 — Pure)**: Can start after Foundational — BLOCKS US1
- **US1 (P1 — Caller enrich)**: Depends on US2 — sau US2 xong mới bắt đầu để xử lý nil
- **US3 (P2 — Deadcode)**: Can start after Foundational — nhưng nên sau US2 (3 method trùng)
- **US4 (P3 — Dead wiring)**: Can start after Foundational — độc lập

### Parallel Opportunities

- Phase 1: T001 ∥ T002
- Phase 2: T004 ∥ T005
- Phase 3: Chỉ 1 module wardrobe + 1 module brand tuần tự (không parallel vì ít file)
- Phase 4: T015, T016, T017 có thể parallel (khác file)
- Phase 5: T021 ∥ T022
- Phase 7: T030, T031, T032 có thể parallel

---

## Parallel Example: Phase 4 (Caller Enrich)

```bash
# Có thể chạy song song 3 caller enrich:
Task T015: workflows.go:259 (brand enrich)
Task T016: workflow_helpers.go:59
Task T017: workflow_helpers.go:108
```

---

## Implementation Strategy

### MVP First (US2 Pure Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: US2 — thuần hóa 7+5 method trực tiếp
4. **STOP and VALIDATE**: audit `pure: 0`, `make wire && make build` pass (nhưng caller sẽ nil nếu chưa làm US1 — chỉ validate unit)

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. US2 Pure → `pure: 0` (nhưng caller nil — chưa demo được)
3. US1 Caller enrich → `detour: 0` + hết nil → Demo
4. US3 Deadcode → `deadcode: 0` → Demo
5. US4 Dead wiring → Demo
6. Polish → Constitution + doc + benchmark

### Suggested MVP Scope

**MVP = Phase 1 + Phase 2 + Phase 3 + Phase 4 (US2 + US1)**
- US2 thuần hóa mà không có US1 sẽ gây nil → phải gộp US2+US1 thành MVP thực sự
- Giải quyết root cause + hết detour + hết nil

---

## Notes

- [P] tasks = khác file, không phụ thuộc
- [Story] label maps task tới user story cụ thể
- Mỗi phase nên commit riêng (tuần tự P1→P1→P2→P3, mỗi story 1 commit/PR)
- Verify `make test` và `make lint` sau mỗi phase
- Wardrobe đã tách: không tạo `*PureUseCase` mới — sửa trực tiếp `WardrobeContractUseCase`; Brand đang dùng chung handler+contract nên **phải tạo** `BrandItemContractUseCase` mới ở `item/contractuc/` để tách — không sửa `BrandItemUseCase` handler để tránh mất enrich
- `Category`/`Subscription` đã pure ở method-level → không tạo task tách
- File `docs/system-design/contract-purity-pattern.md` sẽ tạo ở Phase 1 nháp, hoàn thiện ở Phase 7
- Constitution update ở Phase 7 T029 (sau verify, bump 1.7.0→1.8.0 MINOR + Governance Sync Impact Report); doc viết trước ở T002
