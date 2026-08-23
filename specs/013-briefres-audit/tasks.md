# Tasks: Audit Response Structs & BriefRes Pattern

**Input**: Design documents từ /specs/013-briefres-audit/

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Không yêu cầu TDD trong spec — KHÔNG tạo test tasks riêng; dùng make check + make test làm cổng verify sau mỗi module (FR-015).

**Organization**: Tasks được nhóm theo User Story. US1-US3 (audit) đã hoàn thành ở giai đoạn plan. US4 (implement) là pha chính, chia theo thứ tự ưu tiên module.

## Format: [ID] [P?] [Story] Description

- **[P]**: Chạy song song (khác file, không dependency)
- **[Story]**: US1/US2/US3/US4
- Include exact file paths

## Path Conventions

Single project (backend): internal/modules/<name>/... tại repository root.

---

## Phase 1: Setup (Đã hoàn thành ở giai đoạn plan) ✅

**Purpose**: Các artifact audit đã được tạo ở giai đoạn /speckit-plan

- [x] T001 [US1] Audit module brand → reports/brand.md
- [x] T002 [US1] Audit module identity → reports/identity.md
- [x] T003 [US1] Audit module wardrobe → reports/wardrobe.md
- [x] T004 [US1] Audit module fashion → reports/fashion.md
- [x] T005 [US1] Audit module subscription → reports/subscription.md
- [x] T006 [US1] Audit module dashboard → reports/dashboard.md
- [x] T007 [US1] Audit module aiexecutor → reports/aiexecutor.md
- [x] T008 [US1] Audit module admin → reports/admin.md
- [x] T009 [US1] Audit module notification → reports/notification.md
- [x] T010 [US1] Tổng hợp findings → research.md + data-model.md + contracts/affected-api-contracts.md

---

## Phase 2: Foundational (Blocking Prerequisites) — Resolve Open Questions + UserBriefRes

**Purpose**: Giải quyết các open questions chặn implement (research.md §5) + tạo struct BriefRes nền tảng dùng chung nhiều module (UserBriefRes).

**⚠️ CRITICAL**: Phải hoàn tất trước khi implement bất kỳ user story nào

- [x] T011 Xác nhận open question #1 (contract identity): default = thêm method `GetBriefByIDs` trả `*UserBriefRes`, giữ `GetByID` trả `*UserRes` (brand admin cần full). Ghi quyết định vào research.md §5
- [x] T012 [P] Xác nhận open question #2 (SSE completed): default = đổi nguồn phát sang brief struct, bỏ `any`. Ghi vào research.md §5
- [x] T013 [P] Xác nhận open question #3 (WardrobeItemRes contract): default = không co nhỏ, thêm `WardrobeItemBriefRes` cho HTTP. Ghi vào research.md §5
- [x] T014 [P] Xác nhận open question #4 (batch-upload): default = thêm `TaskID` vào `WardrobeItemBriefRes`. Ghi vào research.md §5
- [x] T015 [P] Xác nhận open question #5-#7: FashionItemRes canonical giữ nguyên; `OutfitBriefRes` giữ `createdAt`; `/me/avatar` trả `UserBriefRes`. Ghi vào research.md §5
- [x] T016 Tạo struct UserBriefRes tại internal/modules/identity/application/dto/profile_res.go (fields: id, username, email, firstName, lastName, gender, avatarUrl)
- [x] T017 Tạo mapper MapUserBrief tại internal/modules/identity/application/mapper/user_mapper.go (không fetch subscription/body profile)
- [x] T018 [P] Cập nhật contract identity theo quyết định T011 (default: thêm method `GetBriefByIDs` trả `*UserBriefRes`, giữ `GetByID` trả `*UserRes`) tại internal/modules/identity/presentation/contract/

**Checkpoint**: Foundation sẵn sàng — user story implement có thể bắt đầu theo ưu tiên module

---

## Phase 3: User Story 4 - Implement Brand module refactor (Priority: P1) 🎯 MVP

**Goal**: Implement các BriefRes và trim field cho module brand (nhiều vấn đề nhất: bảo mật + payload)

**Independent Test**: go build ./internal/modules/brand/... + make test pass; các API user/public trả struct mới, admin/portal không đổi (verify qua git diff)

### Implementation for User Story 4 (Brand)

- [x] T019 [P] [US4] Tạo struct BenefitBriefRes tại internal/modules/brand/application/dto/benefit.go (B1)
- [x] T020 [P] [US4] Tạo struct BrandItemBriefRes tại internal/modules/brand/application/dto/brand_item.go (B2, reconcile với wardrobe)
- [x] T021 [P] [US4] Tạo struct LoyaltyPointTransactionBriefRes tại internal/modules/brand/application/dto/loyalty.go (B6)
- [x] T022 [P] [US4] Tạo struct BrandCustomerBriefRes tại internal/modules/brand/application/dto/customer.go (B4)
- [x] T023 [P] [US4] Tạo struct BrandConversationUserRes tại internal/modules/brand/application/dto/chat.go (B5)
- [x] T024 [P] [US4] Tạo mappers tương ứng trong internal/modules/brand/application/mapper/brand.go (MapBenefitBrief, MapBrandItemBrief, MapLoyaltyPointTransactionBrief, MapBrandCustomerBrief, MapBrandConversationUser)
- [x] T025 [US4] Cập nhật BenefitRedemptionRes: bỏ benefitId, thêm benefit \*BenefitBriefRes tại internal/modules/brand/application/dto/benefit.go + internal/modules/brand/application/mapper/brand.go (reports/brand.md §Underdelivery #1)
- [x] T026 [US4] Cập nhật handler + mapper: GET /brands + GET /brands/:brandId trả BrandBriefRes (bỏ BrandRes public) tại internal/modules/brand/presentation/handler/brand_portal_handler.go + internal/modules/brand/application/mapper/brand.go
- [x] T027 [US4] Cập nhật handler + mapper: list brand items public/user dùng BrandItemBriefRes + FashionItemBriefRes (thay FashionItemRes) tại internal/modules/brand/presentation/handler/brand_portal_handler.go + internal/modules/brand/application/mapper/brand_item.go
- [x] T028 [US4] Cập nhật handler + mapper: POST /brand-benefits/:benefitId/redeem + GET /me/benefit-redemptions trả benefit \*BenefitBriefRes tại internal/modules/brand/presentation/handler/brand_loyalty_handler.go + internal/modules/brand/application/mapper/brand.go
- [x] T029 [US4] Cập nhật handler + mapper: POST /brands/:brandId/join-loyalty + POST /brands/claim trả BrandCustomerBriefRes (bỏ phoneE164 PII) tại internal/modules/brand/presentation/handler/brand_loyalty_handler.go + internal/modules/brand/application/mapper/brand.go
- [x] T030 [US4] Cập nhật handler + mapper: GET /me/brand-loyalties/:brandId/transactions trả LoyaltyPointTransactionBriefRes (bỏ idempotencyKey — bảo mật) tại internal/modules/brand/presentation/handler/brand_loyalty_handler.go + internal/modules/brand/application/mapper/loyalty.go
- [x] T031 [US4] Cập nhật handler + mapper: conversation user dùng BrandConversationUserRes + sender \*UserBriefRes (thêm vào BrandConversationMessageRes) tại internal/modules/brand/presentation/handler/brand_chat_handler.go + internal/modules/brand/application/mapper/brand.go
- [x] T032 [US4] Cập nhật handler + mapper: DigitalSampleResponseRes thêm user \*UserBriefRes cho staff feedback list tại internal/modules/brand/presentation/handler/brand_portal_handler.go + internal/modules/brand/application/mapper/brand.go
- [x] T033 [P] [US4] Cập nhật swagger annotations cho các handler brand thay đổi trong internal/modules/brand/presentation/handler/
- [x] T034 [US4] Chạy make check + make test — verify brand refactor không break admin/portal endpoints

**Checkpoint**: Brand module refactor hoàn tất — MVP (giảm payload lớn nhất + vá bảo mật idempotencyKey/phoneE164/audit fields)

---

## Phase 4: User Story 4 - Implement Wardrobe module refactor (Priority: P1)

**Goal**: Tạo WardrobeItemBriefRes + OutfitBriefRes, bỏ internal fields, populate brandItem

**Independent Test**: go build ./internal/modules/wardrobe/... + make test pass; list grid trả brief, detail giữ đầy đủ

### Implementation for User Story 4 (Wardrobe)

- [x] T035 [P] [US4] Tạo struct WardrobeItemBriefRes tại internal/modules/wardrobe/application/dto/wardrobe.go (B7)
- [x] T036 [P] [US4] Tạo struct OutfitBriefRes tại internal/modules/wardrobe/application/dto/outfit.go (B8)
- [x] T037 [US4] Tạo mapper MapToWardrobeItemBrief tại internal/modules/wardrobe/application/mapper/ (dùng FashionItemBriefRes + BrandItemBriefRes)
- [x] T038 [US4] Tạo mapper MapToOutfitBrief tại internal/modules/wardrobe/application/mapper/outfit.go
- [x] T039 [US4] Đổi list/grid endpoints sang WardrobeItemBriefRes: GET /me/wardrobe-items, clone, catalog-init, retry-analysis tại internal/modules/wardrobe/presentation/handler/wardrobe_item_handler.go + internal/modules/wardrobe/application/usecase/
- [x] T040 [US4] Batch-upload: dùng WardrobeItemBriefRes + giữ taskId (theo quyết định T014) tại internal/modules/wardrobe/presentation/handler/wardrobe_item_handler.go
- [x] T041 [US4] Detail GET /wardrobe-items/:id: bỏ userId/isDeleted/HSL khỏi JSON display tại internal/modules/wardrobe/application/mapper/ (không sửa struct canonical)
- [x] T042 [US4] Search public: đổi fashionItem sang FashionItemBriefRes tại internal/modules/wardrobe/application/dto/wardrobe.go (SearchWardrobeItemRes) + handler search
- [x] T043 [US4] GET /me/outfits dùng OutfitBriefRes; OutfitRes detail bỏ userId tại internal/modules/wardrobe/presentation/handler/wardrobe_outfit_handler.go
- [ ] T044 [US4] Populate brandItem trong read path wardrobe (qua brand contract) hoặc thêm brandName flatten tại internal/modules/wardrobe/application/mapper/ — theo quyết định research *(deferred: cần thêm GetBrandItemsByFashionItemIDs vào brand contract)*
- [x] T045 [P] [US4] Cập nhật swagger annotations cho wardrobe handlers
- [x] T046 [US4] Chạy make check + make test — verify wardrobe + không break AI pipeline (FashionItemRes giữ nguyên cho contract)

**Checkpoint**: Wardrobe module refactor hoàn tất

---

## Phase 5: User Story 4 - Implement Identity module refactor (Priority: P1)

**Goal**: Dùng UserBriefRes cho contract display + avatar PUT

**Independent Test**: go build ./internal/modules/identity/... + make test pass; brand member/customer nhận avatarUrl

### Implementation for User Story 4 (Identity)

- [ ] T047 [P] [US4] Cập nhật PUT /api/v1/me/avatar trả UserBriefRes thay UserRes tại internal/modules/identity/presentation/handler/me_handler.go (theo quyết định T015)
- [ ] T048 [US4] Cập nhật MapToUserRes không fetch subscription/body profile cho các lookup display (contract) — chỉ dùng MapUserBrief tại internal/modules/identity/application/mapper/user_mapper.go
- [ ] T049 [US4] Cập nhật các consumer brand (member/customer/chat) nhúng UserBriefRes thay vì flatten userFullName/userEmail tại internal/modules/brand/application/mapper/
- [ ] T050 [US4] Bỏ struct trùng lặp BrandCustomerUserRes tại internal/modules/brand/application/dto/customer.go nếu đã dùng chung UserBriefRes (theo quyết định T011)
- [ ] T051 [P] [US4] Cập nhật swagger annotations identity nếu cần
- [ ] T052 [US4] Chạy make check + make test — verify identity contract change không break brand/dashboard

**Checkpoint**: Identity module refactor hoàn tất

---

## Phase 6: User Story 4 - Implement Subscription module refactor (Priority: P2)

**Goal**: Trim fallback\*, tạo SubscriptionPlanBriefRes cho plans public

**Independent Test**: go build ./internal/modules/subscription/... + make test pass; daily-quota/overview không còn field fallback\*

### Implementation for User Story 4 (Subscription)

- [ ] T053 [P] [US4] Tạo struct SubscriptionPlanBriefRes tại internal/modules/subscription/application/dto/billing_dto.go (B11)
- [ ] T054 [US4] Tạo mapper MapPlanBrief tại internal/modules/subscription/application/mapper/
- [ ] T055 [US4] GET /api/v1/subscriptions/plans trả []SubscriptionPlanBriefRes (bỏ id, isActive) tại internal/modules/subscription/presentation/handler/billing_handler.go + internal/modules/subscription/application/usecase/; admin plans giữ SubscriptionPlanDTO
- [ ] T056 [US4] Trim fallbackPlanCode/TierRank/Kind khỏi UserSubscriptionOverviewDTO + UserSubscriptionDTO tại internal/modules/subscription/application/dto/quota_dtos.go + internal/modules/subscription/application/mapper/
- [ ] T057 [US4] Trim maxWardrobeItems/maxOutfits khỏi UserSubscriptionDTO (daily-quota context) tại internal/modules/subscription/application/dto/quota_dtos.go + internal/modules/subscription/application/mapper/
- [ ] T058 [US4] Chạy make check + make test — verify subscription

**Checkpoint**: Subscription module refactor hoàn tất

---

## Phase 7: User Story 4 - Implement Fashion module refactor (Priority: P2)

**Goal**: RecommendedFashionItemBriefRes + SSE completed brief data

**Independent Test**: go build ./internal/modules/fashion/... + make test pass; outfit recommendation payload giảm

### Implementation for User Story 4 (Fashion)

- [ ] T059 [P] [US4] Tạo struct RecommendedFashionItemBriefRes tại internal/modules/fashion/application/dto/recommendation.go (B9)
- [ ] T060 [US4] Cập nhật mapper MapRecommendedItemRes dùng RecommendedFashionItemBriefRes (bỏ material/pattern/fit) tại internal/modules/fashion/application/mapper/recommendation.go
- [ ] T061 [US4] SSE completed: đổi nguồn phát data sang brief item (theo quyết định T012) tại internal/modules/fashion/presentation/worker/ + phối hợp internal/modules/wardrobe/presentation/contract/
- [ ] T062 [P] [US4] (Ưu tiên thấp, conditional) Tạo ChatSessionBriefRes cho GET /api/v1/ai/chat/sessions tại internal/modules/fashion/application/dto/ + internal/modules/fashion/presentation/handler/fashion_ai_handler.go — CHỈ thực hiện nếu dev quyết định `contextSummary` không cần trong list (mặc định KHÔNG tạo, giữ ChatSessionRes)
- [ ] T063 [US4] Chạy make check + make test — verify fashion

**Checkpoint**: Fashion module refactor hoàn tất

---

## Phase 8: User Story 4 - Implement Dashboard module refactor (Priority: P2)

**Goal**: Bổ sung Item/User brief cho 2 struct underdelivery, bỏ BrandID/UpdatedAt KPI

**Independent Test**: go build ./internal/modules/dashboard/... + make test pass; digital-sample-lab analytics có item brief

### Implementation for User Story 4 (Dashboard)

- [ ] T064 [US4] DigitalSampleLabAnalyticsRes: thêm Item \*BrandItemBriefRes (hoặc SampleItemBriefRes có ImageURL) tại internal/modules/dashboard/application/dto/brand_dashboard_dto.go + usecase (join brand item)
- [ ] T065 [US4] DrillDownSampleFeedbackDTO: thêm User *UserBriefRes + Item *BrandItemBriefRes tại internal/modules/dashboard/application/dto/dynamic_query_dto.go + usecase (join identity + brand)
- [ ] T066 [US4] BrandKPICardsRes: bỏ BrandID + UpdatedAt tại internal/modules/dashboard/application/dto/brand_dashboard_dto.go + internal/modules/dashboard/application/usecase/
- [ ] T067 [US4] Chạy make check + make test — verify dashboard

**Checkpoint**: Dashboard module refactor hoàn tất

---

## Phase 9: User Story 4 - Implement aiexecutor module refactor (Priority: P3)

**Goal**: AICostPolicyBriefRes cho grant list

**Independent Test**: go build ./internal/modules/aiexecutor/... + make test pass; grant list hiển thị tên/mã policy

### Implementation for User Story 4 (aiexecutor)

- [ ] T068 [P] [US4] Tạo struct AICostPolicyBriefRes tại internal/modules/aiexecutor/application/dto/model_config.go (B12)
- [ ] T069 [US4] AIPolicyGrantDTO: thay AICostPolicyID bằng AICostPolicy \*AICostPolicyBriefRes (+ cân nhắc thêm Status) tại internal/modules/aiexecutor/application/dto/model_config.go + internal/modules/aiexecutor/application/usecase/admin/ (join AICostPolicy)
- [ ] T070 [US4] Chạy make check + make test — verify aiexecutor

**Checkpoint**: aiexecutor module refactor hoàn tất

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: frontend-note.md + verify toàn cục

- [ ] T071 Kiểm tra tổng thể: không có endpoint admin/portal nào bị thay đổi response (git diff + swagger review)
- [ ] T072 Tạo file **frontend-note.md** tại specs/013-briefres-audit/ — liệt kê toàn bộ API bị thay đổi, field cũ → field mới, hướng dẫn frontend sửa (tham chiếu contracts/affected-api-contracts.md + các report module)
- [ ] T073 Chạy make check + make test toàn bộ dự án — đảm bảo xanh
- [ ] T074 Cập nhật các open questions còn treo trong research.md §5 với quyết định cuối cùng
- [ ] T075 Rà soát dọn dead code phát hiện trong audit: UserQuotaRes tại internal/modules/identity/application/dto/profile_res.go, PendingTransferRes tại internal/modules/fashion/application/dto/ — nếu chưa thuộc scope nào
- [ ] T076 [P] Verify các BriefRes sẵn có (FR-009): BrandBriefRes, LoyaltyTierBriefRes, FashionItemBriefRes, BrandItemBriefRes (wardrobe), CategoryBriefRes — đối chiếu quyết định trong research.md §3, quyết định giữ nguyên hay điều chỉnh, ghi kết quả vào research.md
- [ ] T077 [P] (Ưu tiên thấp) Rút gọn BrandBriefRes: cân nhắc bỏ `description`/`backgroundUrl` tại internal/modules/brand/application/dto/brand.go — chỉ thực hiện nếu T076 xác nhận cần và không ảnh hưởng frontend

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Hoàn thành (audit đã xong ở plan)
- **Foundational (Phase 2)**: BLOCKS mọi implement — resolve open questions + UserBriefRes
- **User Story phases (Phase 3-9)**: Phụ thuộc Phase 2; chạy theo ưu tiên module hoặc song song
- **Polish (Phase 10)**: Phụ thuộc tất cả user story hoàn tất

### User Story Dependencies

- **US4 brand (P1)**: Bắt đầu sau Phase 2 — độc lập
- **US4 identity (P1)**: Phụ thuộc T016-T018 (UserBriefRes) — brand và dashboard phụ thuộc identity
- **US4 wardrobe (P1)**: Độc lập với brand; có thể song song
- **US4 subscription/fashion/dashboard (P2)**: Dashboard phụ thuộc identity (UserBriefRes)
- **US4 aiexecutor (P3)**: Độc lập

> **Lưu ý dependency UserBriefRes**: T016-T018 (tạo UserBriefRes) thuộc Phase 2 foundational — hoàn tất TRƯỚC Phase 3, nên T031/T032 (brand conversation/sample dùng `*UserBriefRes`) có thể chạy. T049 (identity consumer update brand mapper) và T065 (dashboard) đều phụ thuộc T016-T018 + T047. KHÔNG implement T049 trước khi T016-T018 xong.

### Within User Story 4

- DTO → Mapper → UseCase → Handler → Swagger → make check/test

### Parallel Opportunities

- Các DTO mới (T019-T023, T035-T036, T053, T059, T068) có thể tạo song song
- Brand/wardrobe/subscription/aiexecutor có thể implement song song (khác module)
- Identity phải xong trước dashboard + phần brand consumer nhúng UserBriefRes

---

## Parallel Example: User Story 4 (brand DTO creation)

`bash

# Launch tất cả DTO brand song song:

Task: "Tạo BenefitBriefRes tại brand/application/dto/benefit.go"
Task: "Tạo BrandItemBriefRes tại brand/application/dto/brand_item.go"
Task: "Tạo LoyaltyPointTransactionBriefRes tại brand/application/dto/loyalty.go"
Task: "Tạo BrandCustomerBriefRes tại brand/application/dto/customer.go"
Task: "Tạo BrandConversationUserRes tại brand/application/dto/chat.go"
`

---

## Implementation Strategy

### MVP First (Phase 1-3)

1. Phase 1 (audit) đã xong
2. Phase 2: Resolve open questions + UserBriefRes
3. Phase 3: Brand module (P1) — giảm payload lớn nhất + vá bảo mật
4. **STOP and VALIDATE**: make check + make test, verify admin/portal không đổi

### Incremental Delivery

1. Brand (MVP) → verify
2. Wardrobe + Identity → verify (Identity phải trước dashboard)
3. Subscription → verify
4. Fashion + Dashboard → verify
5. aiexecutor → verify
6. Polish: frontend-note.md + toàn cục

### Parallel Team Strategy

- Developer A: Brand module
- Developer B: Wardrobe module
- Developer C: Subscription module
- Developer D: Identity (UserBriefRes) — sau đó Dashboard phụ thuộc
- (Identity phải hoàn tất T016-T018 + T047-T052 trước khi dashboard T065)

---

## Notes

- [P] tasks = khác file, không dependency
- [US4] label gắn với User Story 4 (implement)
- Mọi thay đổi response phải cập nhật swagger (FR-015)
- Admin/portal endpoints KHÔNG được thay đổi (FR-014) — verify qua git diff
- Commit sau mỗi module hoàn tất (brand, wardrobe, identity, subscription, fashion, dashboard, aiexecutor)
- frontend-note.md là deliverable cuối cùng (FR-011)
- Tránh: task mơ hồ, conflict cùng file, cross-story dependency phá vỡ tính độc lập
