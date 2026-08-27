# Feature Specification: Refactor Contract Purity & Xử Lý Detour Call + Deadcode

**Feature Branch**: `015-contract-purity-refactor`

**Created**: 2026-08-27

**Status**: Draft

**Input**: User description: "tuân thủ @AGENTS.md và @docs/development/agent-workflow.md, lên specs để thực hiện và xử lí toàn bộ vấn đề tìm được trong cả 2 folder bạn vừa tạo trong docs/temp"

**Tài liệu nguồn**:
- `docs/temp/wardrobe-fashion-cycle-refactor/` — phân tích cycle 2-way `wardrobe ↔ fashion` và kế hoạch event-driven (file 01-04 + README)
- `docs/temp/detour-call-audit/` — báo cáo audit 3 vấn đề: Detour Call, Pure Contract Violation, Contract Deadcode (file 01-05)

---

## Tóm tắt bối cảnh

Audit toàn bộ 8 module của dự án phát hiện **3 nhóm vấn đề** về kiến trúc Contract (đã rà lại method-level để tránh overengineering):

| Vấn đề | Số lượng (method-level) | Root cause | Severity |
|---|---|---|---|
| **Detour Call** (chain 2-hop qua lại) | 7 chuỗi | Contract usecase enrich ngầm qua fashionContract | HIGH + MEDIUM |
| **Pure Contract Violation** (leaky method trong contract usecase) | 2 contract (wardrobe **4/8 alive leaky** — 7/15 nếu tính 3 deadcode trùng đã tách; brand 5/9 method **dùng chung handler+contract** nên phải tách `BrandItemContractUseCase` riêng) — `Category`/`Subscription` đã pure ở method-level, không tính | Enrich trong contract usecase | HIGH |
| **Contract Deadcode** (method không có caller) | 12 method | Interface Segregation vi phạm | LOW |
| **Dead Wiring** (inject nhưng không dùng) | 2 (wardrobe `aiService` + subscription `userBriefContract`) | Lỗi refactor / inject không dùng | LOW |

**Mục tiêu feature**: Thuần hóa contract usecase — **sửa trực tiếp nếu đã tách** (`WardrobeContractUseCase` đã tách), **tạo mới nếu đang dùng chung với handler** (`BrandItemUseCase` dùng chung handler+contract → tạo `BrandItemContractUseCase` riêng), xóa deadcode, và ép caller phải tự enrich khi cần — tránh làm handler mất enrich.

**Phân biệt 2 loại usecase**:
- **Usecase thường / handler usecase** (`WardrobeItemUseCase`, `CategoryUseCase`, `BrandItemUseCase` thường, `SubscriptionUseCase`) — phục vụ handler/worker, được phép enrich.
- **Contract usecase** (`WardrobeContractUseCase` ở `wardrobe/application/usecase/wardrobe/contractuc/` đã tách; `BrandItemContractUseCase` sẽ tạo mới ở `brand/application/usecase/item/contractuc/` vì brand đang dùng chung) — chuyên phục vụ contract, phải thuần (chỉ `*Repo` + mapper, không `fashionContract`/`userSubContract`/`userBriefContract`).

**Rủi ro khi thuần hóa**: Caller hiện đang dùng nested field đã enrich (ví dụ `item.FashionItem`) sẽ thành `nil` nếu không gọi thêm contract. Phải kiểm tra và ép caller gọi thêm `fashionContract` tương ứng (ví dụ `ListFashionItems`) + stitch bằng `sliceutils.IndexBy`/`GroupBy`.

---

## Clarifications

### Session 2026-08-27

- Q: Triển khai theo thứ tự nào cho 4 user story (P1/P1/P2/P3)? → A: Tuần tự theo priority (P1 → P1 → P2 → P3) — giảm rủi ro, dễ rollback, phù hợp workflow Spec-kit.
- Q: Có cần cập nhật Constitution thêm nguyên tắc "Pure Contract" không? → A: Có — thêm nguyên tắc Pure Contract vào §II (Strict Module Boundaries) để ngăn chặn leaky contract tái diễn.
- Q: Có cần viết ADR cho quyết định tách pure contract không? → A: Có, nhưng viết doc dài trong `docs/system-design/` (thay vì ADR ngắn) — Constitution phải ref tới vị trí file đó.
- Q: Nếu benchmark tăng >10% DB call thì xử lý thế nào? → A: Optimize (batch query/cache) trước khi merge — đảm bảo NFR-001 được tuân thủ, tránh tích lũy technical debt.
- Q: Có cần feature flag để rollback nhanh không? → A: Không cần — refactor nội bộ, API contract không đổi, dùng git revert khi cần.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Loại bỏ Detour Call trong workflow fashion (Priority: P1)

Là **developer/maintainer** của SmartWardrobe Backend, tôi muốn mọi cuộc gọi từ fashion use case sang wardrobe/brand contract **chỉ trigger đúng 1 lần** (không bị "đi vòng" qua module khác rồi quay lại), để:
- Debug dễ dàng hơn (trace stack rõ ràng).
- Performance tốt hơn (giảm N+1 query pattern ẩn).
- Coupling giảm (fashion không bị phụ thuộc ngầm vào enrichment logic của wardrobe).

**Why this priority**: Đây là root cause ảnh hưởng nhiều flow business quan trọng: cleanup failed items, stale processing recovery, chat AI, outfit recommendation. Fix cái này sẽ giải quyết được 7/7 chuỗi detour call.

**Independent Test**: 
- Trigger `FashionWorkerUseCase.CleanupFailedItems` → assert chỉ có 1 call tới `wardrobeContract.GetFailedItemsForCleanup` (không có call ngược `fashionContract`).
- Trigger `WardrobeChatUseCase.buildChatGenerationInput` → assert chỉ fetch wardrobe items 1 lần, sau đó fashion caller tự gọi `fashionContract.ListFashionItems` riêng.

**Acceptance Scenarios**:
1. **Given** fashion use case gọi `wardrobeContract.GetWardrobeItemsByUserID`, **When** request được xử lý, **Then** wardrobe contract chỉ trả về wardrobe items (không enrich fashion data).
2. **Given** fashion use case cần enrich fashion data, **When** nhận kết quả từ wardrobe, **Then** fashion caller tự gọi `fashionContract.ListFashionItems` với IDs thu được.
3. **Given** 7 chuỗi detour call đã được phát hiện, **When** refactor hoàn tất, **Then** audit script verify được 0 chuỗi còn lại.

---

### User Story 2 - Thuần hóa Contract Usecase (Priority: P1)

Là **architect/lead developer**, tôi muốn mọi contract usecase trở nên thuần (chỉ gọi repo nội bộ) — **sửa trực tiếp nếu đã tách, tạo mới nếu đang dùng chung với handler** — để:
- Caller biết trước contract không enrich ngầm.
- Trách nhiệm rõ ràng: contract chỉ là "data fetcher", caller lo enrich.
- Handler giữ nguyên enrich (không bị ảnh hưởng), contract tách riêng để thuần.

**Why this priority**: Pure Contract Violation là root cause của Detour Call. Thuần hóa contract usecase = tự động giải quyết Detour Call.

**Independent Test**: 
- Đọc `WardrobeContractUseCase` (wardrobe đã tách riêng `contractuc`) → assert 4 alive leaky đã bỏ `fashionContract`/`userSubContract`, chỉ còn `wardrobeRepo` + mapper (3 deadcode sẽ xóa).
- Đọc `BrandItemContractUseCase` mới (`brand/application/usecase/item/contractuc/contract_uc.go`) → assert 5 method pure chỉ `brandRepo`/`brandItemRepo`/`customerRepo`/`benefitUC`, không `fashionContract`/`userBriefContract`, chỉ trả `FashionItemID` thô; đồng thời verify `BrandItemUseCase` cũ (handler) vẫn enrich bình thường.
- `CategoryUseCase`/`SubscriptionUseCase` đã pure ở method-level (verify, không cần sửa).

**Acceptance Scenarios**:
1. **Given** `WardrobeContractUseCase` đã tách, có 4/8 alive leaky, **When** sửa trực tiếp cho thuần, **Then** mỗi method chỉ gọi `wardrobeRepo` + mapper, không gọi `fashionContract`/`userSubContract`.
2. **Given** `BrandItemUseCase` đang dùng chung handler+contract, **When** tạo `BrandItemContractUseCase` riêng cho thuần, **Then** contract caller nhận `BrandItemRes` với `FashionItem == nil` (còn `FashionItemID`), trong khi handler caller (`GetBrandItemForStaff`…) vẫn nhận `FashionItem` đầy đủ.
3. **Given** contract đã thuần, **When** caller cần enrich, **Then** caller phải tự gọi `fashionContract.ListFashionItems`/`GetFashionItem` và stitch bằng `sliceutils.IndexBy`/`GroupBy` — verify không còn `nil` do thiếu enrich.
4. **Given** refactor thuần hóa, **When** chạy test suite, **Then** tất cả test cũ vẫn pass — API surface giữ nguyên, chỉ thay đổi bên trong.

---

### User Story 3 - Dọn dẹp Contract Deadcode (Priority: P2)

Là **developer**, tôi muốn mỗi method trong contract interface phải có ít nhất 1 caller thực sự, để:
- Giảm surface area, dễ review và maintain.
- Tránh nhầm lẫn (developer mới không gọi method không có caller vì nghĩ nó quan trọng).
- Tăng độ tin cậy của contract (mỗi method đều "live").

**Why this priority**: Quick win, ít rủi ro, làm trước để giảm noise trước khi refactor lớn. Nhưng không critical như P1.

**Independent Test**:
- Chạy audit script (grep) trên 12 method deadcode đã phát hiện → assert 0 caller ngoài `_impl.go`/`_test.go`.
- Sau khi xóa, `make build` + `make test` vẫn pass.

**Acceptance Scenarios**:
1. **Given** 12 method deadcode đã được liệt kê, **When** xóa khai báo + impl, **Then** không có test nào bị break.
2. **Given** wire_gen.go đã được regenerate, **When** `make build`, **Then** không có lỗi missing provider.
3. **Given** contract đã sạch deadcode, **When** audit lại, **Then** mọi method còn lại đều có caller.

---

### User Story 4 - Sửa Dead Wiring (Priority: P3)

Là **developer**, tôi muốn mỗi dependency inject vào struct phải thực sự được sử dụng, để:
- Tránh khởi tạo object thừa (tốn memory).
- Tăng tính minh bạch (rõ ràng struct cần gì).
- Tránh confuse khi đọc code (inject thấy có nhưng không dùng → tưởng đã dùng).

**Why this priority**: Low impact, chỉ 1 case. Làm cuối cùng.

**Independent Test**:
- Đọc struct `WardrobeItemUseCase` → assert field `aiService` đã được xóa (vì không dùng).
- `make build` vẫn pass sau khi xóa.

**Acceptance Scenarios**:
1. **Given** `WardrobeItemUseCase` có field `aiService` không dùng, **When** xóa field + param constructor, **Then** struct và constructor gọn hơn.
2. **Given** wire_gen.go đã được regenerate, **When** `make build`, **Then** không lỗi.

---

## Edge Cases

- **Caller bị nil do thiếu enrich**: Sau khi thuần hóa, `item.FashionItem` sẽ `nil` nếu caller không gọi thêm `fashionContract`. Phải rà 7 caller của `WardrobeContract`/`BrandContract` và ép gọi thêm contract tương ứng (ví dụ `ListFashionItems` + `IndexBy`). Nếu sót, UI sẽ thiếu ảnh/category/style.
- **Performance regression**: Nếu trước đây 1 call enrich được batch, sau khi tách caller phải gọi 2 call riêng → có thể tăng latency. Cần benchmark.
- **Test fake phải update**: Các `fakeEmptyWardrobeContract` (chỉ embed interface) có thể không cần update nhiều, nhưng các fake dùng method enrich sẽ phải viết lại để mock thêm `fashionContract`.
- **Order of migration**: Thuần hóa contract usecase trước, sau đó update caller trong cùng commit/PR. Nếu làm ngược lại, caller sẽ nhận `nil`.
- **Concurrent calls**: Khi tách contract, có thể caller gọi 2 contract riêng biệt → cần đảm bảo không phụ thuộc transaction chung (caller stitch ngoài transaction).
- **Không overengineering**: `CategoryUseCase`/`SubscriptionUseCase` đã pure ở method-level → không tạo `*PureUseCase` mới, không sửa.

---

## Requirements *(mandatory)*

### Functional Requirements

#### Nhóm 1: Thuần hóa Contract Usecase (Root cause - P1) — sửa trực tiếp nếu đã tách, tạo mới nếu đang dùng chung với handler

- **FR-001**: System MUST sửa trực tiếp `WardrobeContractUseCase` (`wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go:24-28`) cho thuần — wardrobe **đã tách** riêng `contractuc` (handler dùng `WardrobeItemUseCase`, contract dùng `WardrobeContractUseCase`): bỏ field `fashionContract` + `userSubContract`, chỉ giữ `wardrobeRepo`; 4 method alive leaky (`GetWardrobeItemsByUserID:164`, `GetActiveWardrobeItemsByUserID:198`, `GetFailedItemsForCleanup:262`, `GetStaleProcessingItems:299+309`) phải bỏ enrich `fashionContract`/`userSubContract`, chỉ trả `WardrobeItemRes` với `FashionItem == nil` + thêm field `FashionItemID` (mới) để caller còn ID để tự enrich (3 method deadcode `CopyItemToUser:69`, `VerifyItemsForPost:92`, `GetItemsByIDs:230` trùng deadcode sẽ bị xóa ở FR-007 nên không sửa).
- **FR-002**: `CategoryUseCase`/`SubscriptionUseCase` đã pure ở method-level (verify: `CategoryContract.GetCategory` chỉ `categoryRepo`, `UserSubscriptionContract` 3 method chỉ repo) → **KHÔNG sửa**, không tạo `*PureUseCase` mới — tránh overengineering.
- **FR-003**: System MUST tạo **BrandItemContractUseCase riêng** (`brand/application/usecase/item/contractuc/contract_uc.go`) cho thuần — vì `BrandItemUseCase` hiện **dùng chung** cho cả handler và contract (`brand/presentation/contract/brand_contract_impl.go:16-17` inject `IBrandItemUseCase` và `brand/presentation/handler/brand_portal_handler.go:17` cũng inject `IBrandItemUseCase`). Sửa trực tiếp `BrandItemUseCase` sẽ làm handler (`GetBrandItemForStaff`, `GetBrandItemForUser`, `ListBrandProductsForCustomer`…) mất `FashionItem` enrich. Vì vậy:
  - Giữ nguyên `BrandItemUseCase` (`brand/application/usecase/item/usecase.go` + `helper.go`) cho handler — **KHÔNG đụng**, vẫn được phép enrich qua `fashionContract`/`userBriefContract`.
  - Tạo mới `BrandItemContractUseCase` (interface `IBrandItemContractUseCase`) chỉ phụ thuộc `brandRepo`/`brandItemRepo`/`customerRepo`/`benefitUC` (KHÔNG `fashionContract`/`userBriefContract`), implement 5 method phiên bản pure:
    - `ListEligibleBrandItemsForStyling` — bỏ `userBriefContract.GetByID:87` (caller fashion đã có userID từ auth; nếu caller là worker không có auth thì `customerRepo`+`benefitUC` intra là đủ) và bỏ `fashionContract.ListFashionItems:157` (trả `BrandItemRes` với `FashionItem == nil`, `FashionItemID` vẫn có).
    - `CheckBrandItemEligibility` — bỏ `userBriefContract.GetByID:185` + `fashionContract.GetFashionItem:227`.
    - `GetBrandItemByIDDirect:570-579` (mapper pure `mapToRes` không gọi `fashionContract.GetFashionItem:21`), `GetBrandItemsByFashionItemIDs:581-597` (`mapSliceToBriefRes:33` bỏ `ListFashionItemsBrief`), `GetBrandItemsByIDs:599-608` (`mapSliceToRes:54` bỏ `ListFashionItems`).
  - `BrandContract` (`brand/presentation/contract/brand_contract_impl.go`) chuyển từ inject `IBrandItemUseCase` sang `IBrandItemContractUseCase` cho 5 method trên; các method pure còn lại (`ListActiveBrandItemFashionIDs:231`, `SetFashionItemID:546`, `UpdateBrandItemStatusDirect:558`) giữ trong contract uc thuần (không cần enrich).
  - Wire: thêm `brand/application/usecase/item/contractuc` ProviderSet, `brand/application.ProviderSet` include `contractuc.NewBrandItemContractUseCase`, `brand.ProviderSet` bind `IBrandItemContractUseCase`.
- **FR-004**: System MUST KHÔNG tách `SubscriptionUseCase` workflow — đã pure ở method-level (verify 8/8 delegate pure), không cần `IUserSubscriptionPureUseCase` mới.

#### Nhóm 2: Ép caller tự Enrich (P1) — xử lý nil

- **FR-005**: System MUST rà caller sau thuần hóa, ép gọi thêm `fashionItemRepo` của chính module `fashion` (hoặc các repositories nội bộ tương ứng — tuân Hiến pháp §IV, Use Case không được gọi Contract của chính module mình) để tránh `nil` (vì `FashionItem == nil` nếu không enrich, nhưng `FashionItemID` vẫn có):
  - `FashionWorkerUseCase.CleanupFailedItems` (worker_uc.go:143:155) — sau thuần hóa `FashionItem == nil`, không lấy được `ImagePublicID` từ `item.FashionItem`; phải đổi sang dùng `item.FashionItemID` (mới thêm vào `WardrobeItemRes` với json `fashionItemId`):
    - **Nil Guard**: Nếu `item.FashionItemID == nil` (trường hợp upload lỗi trước khi kịp tạo fashion item), bỏ qua bước gọi `fashionItemRepo.GetByID` và bỏ qua xóa ảnh Cloudinary, trực tiếp thực hiện xóa item tủ đồ.
    - Nếu `item.FashionItemID != nil`, **chỉ gọi `fashionItemRepo.GetByID(ctx, *FashionItemID)`** để lấy `ImagePublicID` trước khi `DeleteImage`/`DeleteWardrobeItem`.
  - `FashionWorkerUseCase.RecoverStaleProcessingItems` (worker_uc.go:208:217) — check `item.FashionItem == nil` sẽ luôn true sau thuần hóa → phải sửa logic: **bỏ check `FashionItem == nil`**, extract `FashionItemID`, gọi `fashionItemRepo.GetByID` (kèm nil guard cho `FashionItemID`) trước khi check `ProcessingRetryCount`.
  - `WardrobeChatUseCase.buildChatGenerationInput` (chat/workflows.go:249:266 `item.FashionItem.Category`/`Style`) → sau pure `FashionItem == nil` phải tự gọi `fashionItemRepo.GetByIDs` với IDs từ `FashionItemID` + `sliceutils.IndexBy` để bù.
  - `WardrobeChatUseCase.buildChatGenerationInput` (chat/workflows.go:259) — qua `brandContract.ListEligibleBrandItemsForStyling` pure (`BrandItemRes.FashionItem == nil` nhưng `FashionItemID` vẫn có) → caller (thuộc module fashion) tự gọi `fashionItemRepo.GetByIDs` để lấy thông tin ảnh hiển thị brand item.
  - `OutfitRecommendationUseCase.validateAndGetActiveItems` (recommendation/workflow_helpers.go:59) — tương tự wardrobe, tự enrich qua `fashionItemRepo.GetByIDs`.
  - `OutfitRecommendationUseCase.filterCandidates` (recommendation/workflow_helpers.go:108) — qua brand, tự enrich qua `fashionItemRepo.GetByIDs`.
  - Tất cả caller phải stitch bằng `sliceutils.IndexBy`/`GroupBy` (Constitution §II) và dùng `FashionItemID` mới (wardrobe) hoặc `FashionItemID` sẵn có (brand).
- **FR-006**: System MUST đảm bảo functional behavior không đổi sau khi thuần hóa + enrich ở caller (cùng kết quả trả về cho user, không `nil` do thiếu enrich).

#### Nhóm 3: Dọn dẹp Deadcode (P2)

- **FR-007**: System MUST xóa 7 method deadcode của `IWardrobeContract`:
  - `CopyItemToUser`, `VerifyItemsForPost`, `GetItemsByIDs`, `GetUserFashionItemIDs`, `MarkProcessingFailed`, `MarkProcessingNeedsReview`, `CompleteProcessingSuccess`.
- **FR-008**: System MUST xóa 2 method deadcode của `IBrandContract`:
  - `CheckBrandFeatureAccess`, `UpdateBrandItemStatus`.
- **FR-009**: System MUST xóa 3 method deadcode của subscription contracts:
  - `IUserSubscriptionContract.GetUserSubscription`.
  - `ISubscriptionPlanContract.GetDefaultSubscriptionPlanID`, `IsPremiumPlan`.
- **FR-010**: System MUST regenerate `wire_gen.go` sau khi xóa các method contract.

#### Nhóm 4: Sửa Dead Wiring (P3) — 2 case

- **FR-011**: System MUST xóa 2 dead wiring:
  - `WardrobeItemUseCase.aiService aiexecutor_contract.IAIOrchestrator` (`wardrobe/application/usecase/wardrobe/item/item_uc_read.go:41,57`) — không dùng.
  - `SubscriptionUseCase.userBriefContract identity_contract.IUserBriefContract` (`subscription/application/usecase/subscription/subscription_uc.go:41,59,76`) — inject nhưng 0 call (dead wiring phát hiện khi rà shared usecase).
- **FR-012**: System MUST regenerate `wire_gen.go` sau khi xóa inject.

#### Nhóm 5: Audit & Verify

- **FR-013**: System MUST viết audit script (Go test hoặc shell script) để tự động phát hiện 3 vấn đề:
  - Detour call (chain 2-hop).
  - Pure contract violation.
  - Contract deadcode.
- **FR-014**: System MUST đạt 0/0/0 cho cả 3 metrics sau khi refactor hoàn tất (số detour, số violation, số deadcode).

### Non-Functional Requirements

- **NFR-001**: Performance & Query Complexity — Tổng số DB call tăng không quá 10% so với trước refactor (do caller tự enrich bằng batch `ListFashionItems`). Thực hiện kiểm chứng bằng phân tích tĩnh (Static Query-Count & Call-Site Verification) thông qua đếm số lần gọi `fashionContract` trong code review thay vì load test động. Nếu phát hiện nguy cơ N+1 hoặc tăng quá 10%, bắt buộc gộp batch bằng `ListFashionItems` hoặc cache trong request lifecycle trước khi merge.
- **NFR-002**: Backward compatibility — API endpoint và DTO response KHÔNG đổi (WardrobeItemRes thêm `FashionItemID` là additive, không break). Caller bên ngoài (frontend, mobile) không cần thay đổi. Contract method signature KHÔNG đổi, chỉ thay đổi bên trong (FashionItem có thể `nil` nếu caller chưa enrich — caller phải xử lý).
- **NFR-003**: Test coverage — Mỗi contract usecase đã thuần hóa (`WardrobeContractUseCase` 4 method sửa trực tiếp, `BrandItemContractUseCase` 5 method tạo mới) phải có unit test verify không gọi cross-module contract (mock assert `fashionContract`/`userBriefContract` không được gọi). `BrandItemUseCase` handler giữ nguyên không cần test pure.
- **NFR-004**: Documentation — `docs/system-design/contract-purity-pattern.md` phải giải thích "pure contract usecase", before/after, quy tắc caller tự enrich bằng `sliceutils.IndexBy`/`GroupBy`, và cách dùng `FashionItemID` mới.

### Key Entities

- **Contract usecase** (`WardrobeContractUseCase` đã tách, `BrandItemContractUseCase` mới tách): Chuyên phục vụ contract, sau thuần hóa chỉ còn `*Repo` + mapper, không `fashionContract`/`userSubContract`/`userBriefContract`. Wardrobe sửa trực tiếp, Brand tạo mới ở `item/contractuc/`.
- **BrandItemUseCase (handler)**: Giữ nguyên cho handler (`GetBrandItemForStaff`, `ListBrandProductsForCustomer`…) — vẫn được phép enrich `FashionItem` qua `fashionContract`, không đụng để tránh break handler.
- **Caller (fashion use cases)**: Sau thuần hóa contract, phải tự gọi `fashionContract.ListFashionItems`/`GetFashionItem` + `sliceutils.IndexBy`/`GroupBy` để enrich, tránh `nil`.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Audit script verify được **0 chuỗi detour call** trong codebase (từ 7 hiện tại).
- **SC-002**: Audit script verify được **0 contract usecase leaky method** (từ 2 contract method-level hiện tại: wardrobe 7/15, brand 5/9; struct-level cũ là 4).
- **SC-003**: Audit script verify được **0 method contract deadcode** (từ 12 hiện tại).
- **SC-004**: Audit script verify được **0 dead wiring** (từ 2 hiện tại: wardrobe + subscription).
- **SC-005**: Tất cả test cũ (`make test`) vẫn pass sau khi refactor.
- **SC-006**: Tất cả lint check (`make lint`) pass sau khi refactor.
- **SC-007**: Wire build (`make wire && make build`) pass sau khi refactor.
- **SC-008**: Tổng số DB call tăng không quá 10% (xác nhận qua Static Query-Count Verification — đếm `fashionContract` trong code review — cho 4 flow bị ảnh hưởng: `GetWardrobeItemsByUserID`/`GetActiveWardrobeItemsByUserID` (chat/recommendation) và `GetFailedItemsForCleanup`/`GetStaleProcessingItems` (worker); nếu nguy cơ >10% phải gộp batch `ListFashionItems`).
- **SC-009**: Functional test 4 case manual (upload, retry analysis, cleanup failed, outfit recommendation) cho kết quả giống trước refactor.

---

## Assumptions

- **Migration strategy (Atomic MVP)**: US2 (Pure Contract) và US1 (Caller Enrich) tạo thành một khối nguyên tử (Atomic Unit). Dù phát triển tuần tự theo P1 (US2) → P1 (US1), hai story này bắt buộc phải tích hợp và kiểm thử đồng thời trong cùng một đợt merge/PR để loại trừ hoàn toàn rủi ro `nil pointer dereference` tại tầng caller.
- **Caller có thể tự gọi 2 contract riêng**: Caller (use case khác module) có khả năng gọi 2 contract trong cùng 1 method mà không phức tạp hóa logic.
- **Test pattern hiện tại (embed interface) đủ tốt**: Các fake contract hiện dùng embed interface có thể adapt dễ dàng khi tách pure use case.
- **Outbox pattern đã ổn định**: Không cần thay đổi event-driven infrastructure.
- **SLA performance cho phép tăng nhẹ DB call**: Người dùng chấp nhận tăng ~10% latency để đổi lấy code sạch hơn. Nếu phân tích tĩnh cho thấy vượt 10%, sẽ optimize (batch query/cache) trước khi merge.
- **Rollback dùng git revert**: Không cần feature flag vì refactor nội bộ, API contract không đổi. Git revert là đủ.
- **Documentation kèm theo**: Phải viết doc dài `docs/system-design/contract-purity-pattern.md` TRƯỚC khi code (Phase 1 T002 nháp, Phase 7 T028 hoàn thiện). Constitution sẽ ref tới file đó SAU khi refactor verify xong (Phase 7 T029, bump 1.7.0 → 1.8.0 MINOR theo Governance).

---

## Phạm vi áp dụng (Out of Scope)

- **Tách `dashboard` module**: Dashboard đã được audit ở phase trước, có 6 file use case inject contract (1-way), nhưng không phải root cause của vấn đề. Để phase sau nếu cần.
- **Tách thành microservice**: Chỉ là clean up contract, không thay đổi kiến trúc deployment.
- **Thay đổi API endpoint**: Giữ nguyên 100% API để tránh ảnh hưởng frontend.
- **Refactor event-driven cho các Write Contract Methods**: 4 direct write methods (`wardrobeContract.DeleteWardrobeItem`, `wardrobeContract.UpdateItemStatus`, `wardrobeContract.SetFashionItemID`, `brandContract.SetFashionItemID`) được xác định là pure database operations nội bộ của module đích, không gây detour call hay leak cross-module. Do đó, chúng được giữ lại an toàn dưới dạng Pure Contract trong feature này nhằm cô lập rủi ro refactor. Kế hoạch Event-Driven toàn diện (tại `docs/temp/wardrobe-fashion-cycle-refactor/04_final_plan.md`) được bảo lưu cho một feature chuyên biệt về Asynchronous Decoupling tiếp theo nếu có nhu cầu tách service vật lý.
- **Cập nhật Constitution ngay trong feature này**: Doc `docs/system-design/contract-purity-pattern.md` viết TRƯỚC (Phase 1 T002 nháp, Phase 7 T028 hoàn thiện), Constitution `1.7.0 → 1.8.0 (MINOR)` cập nhật SAU khi refactor verify xong (Phase 7 T029 thêm "Pure Contract Usecase Principle" vào §II + link tới doc, kèm Sync Impact Report) — tuân Governance.

---

## Tài liệu tham khảo

- `docs/temp/wardrobe-fashion-cycle-refactor/01_research_overview.md` — Tổng quan research
- `docs/temp/wardrobe-fashion-cycle-refactor/02_cycle_analysis.md` — Phân tích cycle 2-way
- `docs/temp/wardrobe-fashion-cycle-refactor/03_eventual_consistency_analysis.md` — Phân tích eventual consistency
- `docs/temp/wardrobe-fashion-cycle-refactor/04_final_plan.md` — Kế hoạch event-driven (chưa áp dụng)
- `docs/temp/detour-call-audit/01_overview.md` — Tổng quan audit
- `docs/temp/detour-call-audit/02_wardrobe_fashion.md` — Chi tiết wardrobe + fashion
- `docs/temp/detour-call-audit/03_brand_aiexecutor.md` — Chi tiết brand + aiexecutor
- `docs/temp/detour-call-audit/04_others.md` — Chi tiết identity, subscription, notification, admin
- `docs/temp/detour-call-audit/05_recommendations.md` — Đề xuất hướng giải quyết
- `.specify/memory/constitution.md` — Constitution (version 1.7.0)
- `docs/system-design/contract-purity-pattern.md` — **SẼ TẠO**: Doc giải thích Pure Contract pattern (P1 của feature này)
