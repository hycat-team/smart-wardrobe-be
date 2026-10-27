---
description: "Task list for feature 033 - personalized styling, catalog suggestions, style learning, and recency rotation"
---

# Tasks: Cá nhân hoá phối đồ và gợi ý bổ sung

**Input**: Design documents from `specs/033-personalized-styling/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`,
`contracts/personalization-api.md`, `quickstart.md`

**Tests**: Có. Plan yêu cầu unit test cho thuật toán (blend, cooldown, learning), DB integration test cho
migration/clone/outbox, contract test cho proto/cache và compatibility test cho client cũ. Test task nằm
trong từng user story, sau task implementation tương ứng.

**Organization**: Task theo user story. Hạ tầng dùng chung (plan flag, schema, repo, event, helper) nằm ở
Foundational để US1–US4 có thể chạy song song sau đó.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: chạy song song được (file khác nhau, không phụ thuộc task chưa xong)
- **[Story]**: US1…US7 (xem `spec.md`)
- Mỗi task ghi rõ file(s) cần chạm.

## Path Conventions

Đường dẫn tương đối từ repo backend `smart-wardrobe-be/`. Hai repo client là sibling:
`../smart-wardrobe-fe` (Next.js) và `../smart-wardrobe-mobile` (Flutter).

---

## Phase 1: Setup

**Purpose**: Chuẩn bị môi trường, kill switch và bảo trì kho mã trước khi đổi schema.

- [x] T001 Xác nhận toolchain và nhánh: `git status`, `make buf-lint`, `make migration-check`,
  `make wire svc=account` và `make wire svc=commerce` (wire ghi file; kiểm tra drift bằng `git diff`),
  đọc lại `Makefile` cho `migration-check`, `test-db-integration`, `check`; tạo scratch DB theo
  `quickstart.md` nếu cần.
- [x] T002 Thêm `.specify/feature.json` vào `.gitignore` và chạy `git rm --cached .specify/feature.json`
  (giữ file trên disk); Orchestrator thực hiện thao tác index này dưới git policy của
  `@speckit-orchestrate`. Xác nhận `git check-ignore -v .specify/feature.json` và
  `git ls-files --error-unmatch .specify/feature.json` báo không tracked.
- [x] T003 [P] Thêm cấu hình `Personalization` và các field RAG mới vào `shared/config/config.go`; thêm
  block `personalization:` (enabled, max_styles, style_blend_weight, style_blend_weight_with_request,
  learned_style_weight, learned_embedding_weight, embedding_timeout_seconds, learning_decay,
  gap_score_threshold, suggestion_max_per_role, suggestion_max_total) và block `rag:` bổ sung
  (recently_worn_penalty_days=7, recently_recommended_penalty_days=3, stale_candidate_slots=4,
  stale_eligible_days=30, cooldown_score_tolerance=0.15) vào `configs/config.yaml` và
  `configs/config.production.yaml`. Đối
  chiếu đúng block trong `quickstart.md` §1.

**Checkpoint**: Toolchain xanh, `.gitignore` đúng, config load được.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Quyền lợi gói, schema, entity/repo, event và helper dùng chung. Tất cả user story đều đọc/ghi
trên nền này.

**⚠️ CRITICAL**: Không bắt đầu user story nào trước khi phase này xong.

### Account - quyền lợi gói

- [x] T004 [P] Tạo migration `services/account/database/migrations/<ts>_add_plan_personalization.sql`:
  thêm cột `subscription.subscription_plans.personalization_enabled boolean NOT NULL DEFAULT false`;
  bật `true` cho plan `is_active = true AND price > 0`; Down drop cột. Dùng `StatementBegin/End`.
- [x] T005 [P] Thêm field `PersonalizationEnabled` vào entity
  `services/account/internal/modules/subscription/domain/entities/plan.go`, DTO
  `services/account/dto/subscription/billing_dto.go` (`SubscriptionPlanDTO`, `SubscriptionPlanBriefRes`,
  `UserSubscriptionOverviewRes`, `CreateSubscriptionPlanReq`, `UpdateSubscriptionPlanReq`), mapper
  `services/account/internal/modules/subscription/application/mapper/plan.go`, và usecase
  `services/account/internal/modules/subscription/application/usecase/plan/plan_uc.go`.
- [x] T006 [P] Thêm `PersonalizationEnabled` vào `services/account/dto/subscription/contract_dto.go`
  (`UserSubscriptionContractDTO`), `services/account/dto/subscription/quota.go` (nếu cần cho read model),
  `services/account/internal/modules/subscription/application/mapper/subscription.go`
  (`BuildUserSubscriptionOverviewRes`, `MapUserSubscriptionOverviewToContractDTO`,
  `ContractDTOToCacheEntry`, `CacheEntryToContractDTO`) và
  `services/account/internal/modules/subscription/application/interface/caching/overview_cache.go`
  (`SubscriptionOverviewCacheEntry`, `json:"personalizationEnabled,omitempty"`).
- [x] T007 [P] Thêm key `personalizationEnabled` vào `BuildPlanBenefitSnapshot` trong
  `services/account/internal/modules/subscription/application/usecase/subscription/internal_subscription_apply_workflow.go`.
- [x] T008 Thêm field 11 `bool personalization_enabled` vào `contracts/subscription/v1/subscription.proto`;
  chạy `make buf-gen`; cập nhật
  `services/account/contract/subscription/transport/to_proto.go`,
  `services/account/contract/subscription/transport/mapper.go`,
  `services/account/contract/subscription/transport/mapper_test.go`; cập nhật client
  `contracts/subscription/v1/client/` nếu có mapping overview.

### Commerce - schema và entity

- [x] T009 [P] Tạo migration `services/commerce/database/migrations/<ts>_add_personalization.sql`:
  `fashion.user_style_profiles` (PK user_id, FK `identity.users` cascade, jsonb arrays, vector 768,
  hash/model, learned_*), `fashion.style_signals` (PK id, source, outfit_id, recommendation_id,
  fashion_item_ids, style_snapshot, index + partial unique), `wardrobe.wardrobe_items.last_recommended_at`;
  checks và Down drop đúng thứ tự. DDL theo `data-model.md`.
- [x] T010 [P] Thêm entity `services/commerce/internal/modules/fashion/domain/entities/user_style_profile.go`
  và `.../style_signal.go`; đăng ký `TableName()` trong
  `services/commerce/internal/modules/fashion/domain/entities/table_name.go`.
- [x] T011 [P] Thêm `IUserStyleProfileRepository` và `IStyleSignalRepository` vào
  `services/commerce/internal/modules/fashion/domain/repositories/interfaces.go`:
  `GetByUserID`, `Upsert`, `Delete`, `UpdateLearnedWithLock`, `InsertIfAbsent`, `CountByUserID`.
- [x] T012 [P] Implement `services/commerce/internal/modules/fashion/infrastructure/persistence/user_style_profile_repo.go`
  (GORM; `SELECT ... FOR UPDATE` cho `UpdateLearnedWithLock`; upsert idempotent).
- [x] T013 [P] Implement `services/commerce/internal/modules/fashion/infrastructure/persistence/style_signal_repo.go`
  (`INSERT ... ON CONFLICT (id) DO NOTHING`; partial unique cho feedback).

### Wardrobe - tín hiệu và contract

- [x] T014 [P] Thêm `LastRecommendedAt *time.Time` vào entity
  `services/commerce/internal/modules/wardrobe/domain/entities/wardrobe_item.go`, DTO
  `services/commerce/internal/modules/wardrobe/application/dto/contract_dto.go` và mapper
  `services/commerce/internal/modules/wardrobe/application/mapper/wardrobe.go`.
- [x] T015 [P] Thêm method vào `IWardrobeItemRepository`
  (`services/commerce/internal/modules/wardrobe/domain/repositories/interfaces.go`) và implement trong
  `services/commerce/internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`:
  `TouchLastRecommendedAt(ctx, ids, at)`, `GetStaleActiveItems(ctx, userID, limit)` (status InWardrobe,
  item_type UserItem, order `last_used_at ASC NULLS FIRST, last_recommended_at ASC NULLS FIRST`),
  `GetActiveSystemCatalogItemsByIDs(ctx, ids)` (chỉ SystemCatalogItem + InWardrobe + chưa xoá).
- [x] T016 [P] Thêm `GetSystemCatalogFashionItemIDs(ctx, limit)`, `GetOwnedFashionItemIDs(ctx, userID)`
  (mọi row chưa xoá, bất kỳ status), `GetStaleActiveWardrobeItemsByUserID(ctx, userID, limit)` (trả DTO)
  và `TouchLastRecommendedAt(ctx, wardrobeItemIDs, at)` vào
  `services/commerce/internal/modules/wardrobe/presentation/contract/wardrobe_contract.go` + impl
  `.../wardrobe_contract_impl.go`; khai interface usecase trong
  `.../application/interface/usecase/contract_uc.go` và implement trong
  `.../application/usecase/wardrobe/contractuc/contract_uc.go` (chỉ đọc/ghi `wardrobeRepo`, không gọi
  contract khác).

### Event hạ tầng

- [x] T017 [P] Thêm `TopicOutfitSaved = "wardrobe.outfit.saved"` vào
  `shared/application/constants/eventconstants/eventconstants.go`, `QueueStyleLearning` vào
  `.../eventconstants/queues.go`, `RoutingKeyOutfitSaved` vào
  `shared/infrastructure/messaging/rabbitmq_constants.go`; thêm binding quorum + DLQ cho
  `style_learning_queue` trong `shared/infrastructure/messaging/topology.go`.
- [x] T018 [P] Tạo `OutfitSavedEventPayload` trong
  `services/commerce/internal/modules/wardrobe/application/dto/outfit_event.go` (`signalId`, `userId`,
  `outfitId`, `fashionItemIds`, `source`, `occurredAt`).

### Helper dùng chung

- [x] T019 [P] Tạo `services/commerce/internal/modules/fashion/application/usecase/personalization/shared/style_lexicon.go`:
  map canonical style → câu mô tả tiếng Việt để enrich embedding; `NormalizeStyleLabels` dùng taxonomy
  hiện có, giữ nguyên giá trị lạ.
- [x] T020 [P] Tạo `.../personalization/shared/embedding_helper.go`: `BlendVectors(query, style, weight)`,
  `MeanVectors`, `HashStyleProfile`; xử lý dimension 768 và vector rỗng. Đặt `NormalizeVector` duy nhất
  ở `shared/pkg/utils/sliceutils/sliceutils.go`; helper chỉ gọi lại, không định nghĩa bản thứ hai.

### DI và build

- [x] T021 Đăng ký repo/usecase/provider mới trong `services/commerce/internal/modules/fashion/provider.go`,
  `services/commerce/internal/modules/wardrobe/provider.go` (nếu cần) và
  `services/commerce/internal/bootstrap/worker.go`; chạy `make wire svc=commerce` và `go build ./...` cho commerce.
- [x] T022 [P] DB integration test nền cho migration + repo trong `services/commerce/internal/modules/fashion/infrastructure/persistence/personalization_schema_test.go` và
  `services/commerce/internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo_personalization_test.go`:
  dùng disposable DB dựng đúng `database/init/*.sql` + `make migrate-up` (không `AutoMigrate`/tự tạo schema),
  DSN có `search_path` theo `docs/development/local-setup.md`; test unique partial feedback, FK cascade,
  `GetStaleActiveItems` order, clone catalog filter; đặt tên `TestIntegration*`.
- [x] T023 [P] Contract test cho proto/cache: `services/account/contract/subscription/transport/mapper_test.go`
  (field 11 round-trip, client cũ thiếu field → false) và test cache entry cũ trong
  `services/account/internal/modules/subscription/application/mapper/subscription_test.go`.
- [x] T024 [P] Test hạ tầng event: `shared/infrastructure/messaging/topology_test.go` xác nhận queue
  `style_learning_queue` là quorum + DLQ; test payload marshal/unmarshal trong
  `services/commerce/internal/modules/wardrobe/application/dto/outfit_event_test.go`.

**Checkpoint**: `make wire svc=account` + `make wire svc=commerce` xanh; migration up/down/up xanh; build commerce/account xanh; repo/event test
nền xanh.

---

## Phase 3: User Story 1 - Người dùng trả phí khai báo gu và nhận gợi ý đúng gu (Priority: P1) 🎯 MVP

**Goal**: CRUD hồ sơ gu + entitlement; gợi ý phối đồ áp dụng gu như ưu tiên mềm (vector + scoring +
prompt + fallback), request style thắng gu.

**Independent Test**: PUT gu 2 style + 1 excluded, gọi recommendation không có style và có style mâu thuẫn;
kiểm tra thứ hạng, explanation, và user free bị 403.

### Implementation

- [x] T025 [P] [US1] Tạo DTO `services/commerce/internal/modules/fashion/application/dto/style_profile.go`:
  `UpdateStyleProfileReq` (pointer fields), `StyleProfileRes`, `StyleSuggestionItem`,
  `StyleSuggestionRes`; validation tags cho độ dài từng giá trị.
- [x] T026 [P] [US1] Tạo mapper `services/commerce/internal/modules/fashion/application/mapper/style_profile.go`
  map entity → response (không trả embedding/hash).
- [x] T027 [US1] Implement `.../profile/profile_uc.go`: `GetProfile`, `UpdateProfile` (thứ tự: trim →
  bỏ rỗng → dedupe → kiểm độ dài → từ chối khi vượt `max_styles` → embed fail-soft, hash skip),
  `DeleteProfile`; đọc entitlement qua
  `IUserSubscriptionContract`; nếu thiếu entitlement thì PUT/DELETE trả `ErrPersonalizationNotAvailable`
  (403), GET vẫn đọc; nếu `personalization.enabled = false` thì PUT/DELETE trả
  `ErrPersonalizationDisabled` (403), GET vẫn đọc. Tạo các constructor lỗi
  `ErrPersonalizationNotAvailable`, `ErrPersonalizationDisabled`, `ErrStyleLimitExceeded`,
  `ErrStyleValueInvalid` trong `.../fashion/application/errors/` (dùng `apperror`).
- [x] T028 [US1] Tạo handler `services/commerce/internal/modules/fashion/presentation/handler/style_profile_handler.go`
  (GET/PUT/DELETE `/me/style-profile`) + swagger annotation; đăng ký route trong
  `services/commerce/internal/modules/fashion/presentation/router/router.go` dưới `privateApi` với
  `/me/style-profile`.
- [x] T029 [US1] Implement `services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/personalization_helper.go`:
  load profile theo `subOverview.PersonalizationEnabled`; nếu `profile.enabled = false` hoặc không có
  quyền lợi thì trả `PersonalizationContext` rỗng; nếu bật, merge intent (request style thắng), tính
  `styleVector` từ explicit + learned với trọng số config.
- [x] T030 [US1] Sửa `.../recommendation/candidate_helpers.go`:
  `buildRecommendationQueryVector` blend style vector + normalize; fallback text-only khi không có vector.
- [x] T031 [US1] Sửa `.../recommendation/ranking/scoring.go`:
  thêm `personal-style-match`, `learned-style-match`, `avoid-style`; không hard filter.
- [x] T032 [US1] Sửa `.../recommendation/synthesis/prompt.go` (context `personal_style`, `learned_style`,
  `avoid_styles`) và `.../synthesis/generation.go` (system prompt rule ưu tiên gu).
- [x] T033 [US1] Sửa `.../recommendation/ranking/fallback_hsl.go` để áp dụng cùng style scoring.
- [x] T034 [US1] Thêm `RecommendationID uuid.UUID` vào `RecommendedOutfitRes` trong
  `.../application/dto/recommendation.go`; sinh `recommendationId` trong
  `.../recommendation/workflow.go`; nối `PersonalizationContext` vào workflow/`workflow_helpers.go`;
  đảm bảo `filterCandidates` nhận context, không đổi chữ ký handler.
- [x] T035 [US1] Thêm test: `style_profile_uc_test.go` (normalize/cap/dedupe/fail-soft; embed lỗi không
  ghi `embedding_hash`; PUT lại thì thử lại; PUT/DELETE thiếu entitlement → 403; kill switch → 403),
  `personalization_helper_test.go` (request thắng, weight, request rỗng, `enabled=false` → context rỗng,
  mất quyền lợi → context rỗng), `candidate_helpers_test.go` (blend/normalize),
  `scoring_test.go` (cộng/trừ điểm), `synthesis_test.go` (context key).
- [x] T036 [US1] Chạy `make wire svc=commerce`, `go test -race -count=3` cho
  `./services/commerce/internal/modules/fashion/...` và `./services/account/internal/modules/subscription/...`;
  chạy kịch bản US1 trong `quickstart.md` và ghi kết quả.

**Checkpoint**: US1 chạy độc lập: PUT gu → recommendation thay đổi; free user bị 403; AI lỗi vẫn giữ gu ở
mức text.

---

## Phase 4: User Story 2 - Gợi ý món còn thiếu từ tủ đồ hệ thống và tự đưa vào tủ khi lưu (Priority: P1)

**Goal**: Suggestions tách riêng cho role thiếu/chưa phù hợp, lấy từ catalog, dedupe món đã sở hữu; lưu
outfit clone catalog trong cùng transaction; fix catalog-init.

**Independent Test**: Tủ thiếu footwear → suggestions có footwear; lưu outfit có món catalog → có bản sao;
vượt hạn mức → 400 và 0 bản sao; catalog-init cross-user bị từ chối.

### Implementation

- [x] T037 [P] [US2] Thêm DTO `OutfitSuggestionRes` và field `Suggestions []OutfitSuggestionRes`
  (`omitempty`) vào `services/commerce/internal/modules/fashion/application/dto/recommendation.go`;
  thêm mapper trong `.../application/mapper/recommendation.go`.
- [x] T038 [P] [US2] Tạo hai port `IPersonalizationCatalogCache` và `IRecommendationSnapshotCache` trong
  `services/commerce/internal/modules/fashion/application/interface/caching/`, impl Redis trong
  `services/commerce/internal/modules/fashion/infrastructure/caching/`; key `personalization:catalog:v1`
  (TTL 5 phút) và `personalization:rec:{recommendationId}` (TTL 7 ngày), fail-soft; đăng ký trong
  `services/commerce/internal/modules/fashion/provider.go`.
- [x] T039 [US2] Implement retrieval catalog trong `services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go`:
  lấy ID qua `wardrobeContract.GetSystemCatalogFashionItemIDs`, cache, gọi
  `fashionItemRepo.GetHybridCandidates`, loại `fashion_item_id` user đã sở hữu (lấy qua
  `wardrobeContract.GetOwnedFashionItemIDs`, mọi status chưa xoá).
- [x] T040 [US2] Implement gap detection + suggestion builder trong `services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go`:
  role thiếu hoặc score dưới `gap_score_threshold`; tối đa 1 món/role, 3 món/response; reason template
  tiếng Việt; dùng cùng semantic query/style vector/hard filters.
- [x] T041 [US2] Nối suggestions vào `.../recommendation/workflow.go` sau khi có outfit cuối (AI hoặc HSL)
  và trước khi dựng response; đảm bảo không trừ quota.
- [x] T042 [US2] Thêm `ItemContext *string` vào `SaveOutfitItemReq` trong
  `services/commerce/internal/modules/wardrobe/application/dto/outfit.go` với
  `binding:"omitempty,oneof=user_wardrobe brand_item system_catalog"`.
- [x] T043 [US2] Sửa `services/commerce/internal/modules/wardrobe/application/usecase/outfit/outfit_uc.go`:
  resolve theo `itemContext` hoặc auto (wardrobe → brand → catalog); validate catalog qua
  `GetActiveSystemCatalogItemsByIDs`; nếu user đã sở hữu `fashion_item_id` (row chưa xoá, bất kỳ status)
  thì tái sử dụng row, không clone; nếu chưa, clone row `wardrobe_items`; kiểm tra `MaxWardrobeItems`;
  dedupe input; lỗi mới.
- [x] T044 [US2] Bọc `CreateWithItems` + clone + `TouchLastUsedAt` + outbox publish trong `uow.Execute`
  trong `outfit_uc.go`; đảm bảo `GetDB(ctx)` dùng transaction.
- [x] T045 [US2] Áp dụng cùng logic cho `UpdateOutfit` trong `outfit_uc.go`; chỉ touch `last_used_at` cho
  item mới thêm; diff item cũ/mới từ `outfitRepo.GetByID` (preload items).
- [x] T046 [US2] Fix `InitClosetFromCatalog` trong
  `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/catalog/catalog_uc.go`:
  chỉ nhận `item_type = SystemCatalogItem`, `is_deleted = false`, `status = InWardrobe`.
- [x] T047 [US2] Thêm lỗi mới vào `services/commerce/internal/modules/wardrobe/application/errors/errors.go`:
  `ErrCatalogItemUnavailable`, `ErrWardrobeLimitExceededForOutfit`, `ErrDuplicateOutfitItem`.
- [x] T048 [US2] Cập nhật swagger annotation + response trong `services/commerce/internal/modules/wardrobe/presentation/handler/outfit_handler.go` và
  `services/commerce/internal/modules/fashion/presentation/handler/fashion_ai_handler.go`; chạy
  `make swagger svc=commerce`.
- [x] T049 [US2] Thêm test: `workflow_helpers_test.go` (gap/suggestion limit/dedupe; suggestion thoả
  hard filter + gu → SC-014; user free vẫn có suggestions), `catalog_cache_test.go` (miss/hit),
  `outfit_uc_clone_test.go` (clone + limit + rollback + không tạo job analyze; reuse row nếu đã sở hữu),
  `catalog_uc_test.go` (cross-user bị từ chối). Chạy `go test -race -count=3` scoped.
- [x] T050 [US2] Chạy kịch bản US2 trong `quickstart.md` (bao gồm kiểm tra SQL không có clone mồ côi) và
  ghi kết quả.

**Checkpoint**: US2 chạy độc lập: suggestions đúng role, clone atomic, limit reject, catalog-init an toàn.

---

## Phase 5: User Story 3 - Xoay vòng để đồ lâu không dùng có cơ hội quay lại (Priority: P2)

**Goal**: Sửa tín hiệu recency, thêm stale slots, cooldown theo role, không touch khi update metadata.

**Independent Test**: Món vừa lưu bị giảm ưu tiên; món lâu không dùng xuất hiện; đổi tên outfit không đổi
`last_used_at`; mọi role trong cooldown vẫn ra outfit.

### Implementation

- [x] T051 [P] [US3] Sửa `.../ranking/scoring.go`: nới `recently_worn_penalty_days` 3 → 7, penalty tối đa
  0.45; thêm penalty 0.12 cho `last_recommended_at` gần đây; "món mới" theo `created_at`; bỏ điều kiện
  `styleMatched || occasion` cho long-unworn bonus.
- [x] T052 [P] [US3] Thêm `CandidateSourceStale`, `RankedCandidate.Cooldown`, các counter
  `StaleSlotsAdded`, `CooldownSubstitutions`, `RecentlyUsedRepeatRatio`,
  `RecentlyRecommendedRepeatRatio`, `PersonalStyleMatchRatio`, `ReturnedItemIDs`, `StaleItemIDs`,
  `StaleEligibleItemIDs`, `CooldownItemIDs` vào `.../recommendation/types/structs.go`.
- [x] T053 [US3] Lấy stale candidates qua `wardrobeContract.GetStaleActiveWardrobeItemsByUserID` với
  `limit = MaxWardrobeItems` (rộng, không phải 4), rồi lọc trong Fashion theo `last_used_at` cũ hơn
  `stale_eligible_days` (30) hoặc null, hard filter và role; sort `last_used_at ASC NULLS FIRST,
  last_recommended_at ASC NULLS FIRST`; lấy `stale_candidate_slots`; tính `hybridLimit = minimumPool -
  brandCandidates - staleSlots`; dedupe theo `fashion_item_id`; gắn source `stale`.
  `stale_eligible_item_ids` = toàn bộ tập đủ điều kiện sau lọc (không cap 4); bảo đảm
  `StaleItemIDs ⊆ StaleEligibleItemIDs`.
- [x] T054 [US3] Thêm `ApplyRecencyCooldown` trong `.../ranking/ranking.go`: ưu tiên món ngoài vùng làm
  mát trong `cooldown_score_tolerance`; nếu toàn role trong vùng làm mát thì giữ nguyên; đếm
  `CooldownSubstitutions`.
- [x] T055 [US3] Dedupe theo `fashion_item_id` và lấy `max(last_used_at)` trước scoring trong
  `workflow_helpers.go`.
- [x] T056 [US3] Gọi `wardrobeContract.TouchLastRecommendedAt` best-effort sau khi dựng response trong
  `workflow.go` (chỉ primary + alternatives là wardrobe item); lỗi log và lần sau thử lại. Thêm
  structured log với đầy đủ counter + `returned_item_ids`/`stale_item_ids`/`stale_eligible_item_ids`/
  `cooldown_item_ids`.
- [x] T057 [US3] Thêm prompt rule ưu tiên món không có tag `recency-cooldown` trong
  `.../synthesis/prompt.go`; thêm tag trong `ranking.go`.
- [x] T058 [US3] Thêm test: `scoring_test.go` (window/new-item/bonus), `ranking_test.go` (cooldown +
  all-cooldown fallback), `workflow_helpers_test.go` (stale slots, dedupe clone, `StaleItemIDs ⊆
  StaleEligibleItemIDs`). Test order của repo stale nằm ở T022.
- [x] T059 [US3] Chạy kịch bản US3 trong `quickstart.md`; kiểm tra log `stale_slots_added`,
  `cooldown_substitutions`, `recently_used_repeat_ratio`, `recently_recommended_repeat_ratio`,
  `personal_style_match_ratio`, `returned_item_ids`, `stale_item_ids`, `stale_eligible_item_ids`,
  `cooldown_item_ids`.

**Checkpoint**: US3 chạy độc lập; món lâu không dùng vào được pool; outfit vẫn hợp lệ.

---

## Phase 6: User Story 4 - Hệ thống tự học gu từ những bộ đồ người dùng đã lưu (Priority: P2)

**Goal**: Outbox event → worker → signal → learned_styles/learned_embedding, idempotent, không ghi đè gu
khai.

**Independent Test**: Lưu outfit → learned tăng trong 60 giây; gu khai không đổi; retry không cộng hai lần;
update chỉ đổi tên không tạo signal.

### Implementation

- [ ] T060 [P] [US4] Implement `.../personalization/learning/learning_uc.go`:
  `ApplySignal(ctx, signal)` insert signal idempotent + `UpdateLearnedWithLock`; decay 0.9, top-K 10;
  `learned_embedding` từ `fashionItemRepo.GetByIDs`; weight theo source; nếu
  `personalization.enabled = false` thì ack và bỏ qua.
- [ ] T061 [P] [US4] Tạo `services/commerce/internal/modules/fashion/presentation/worker/style_learning_worker.go`
  consume `style_learning_queue`, gọi `IStyleLearningUseCase`; dùng `workerlog`.
- [ ] T062 [US4] Đăng ký worker trong `services/commerce/internal/bootstrap/worker.go` (field + `All()`),
  `services/commerce/internal/modules/fashion/provider.go`; chạy `make wire svc=commerce`.
- [ ] T063 [US4] Phát event trong `SaveOutfit` (`outfit_uc.go`): sinh `signalId`, payload, publish
  `TopicOutfitSaved` trong `uow.Execute` cùng transaction.
- [ ] T064 [US4] Phát event trong `UpdateOutfit` chỉ khi tập item thay đổi; payload chỉ item mới thêm.
- [ ] T065 [US4] Map feedback `saved` thành source `4 feedback_saved` weight 0.25 (dùng chung usecase) —
  phối hợp với US6.
- [ ] T066 [US4] Thêm test: `learning_uc_test.go` (decay/topK/idempotent/no-embedding),
  worker integration test (`outbox → queue → signal → profile`), `outfit_uc_event_test.go` (retry không
  cộng hai lần, update metadata không signal).
- [ ] T067 [US4] Chạy kịch bản US4 trong `quickstart.md`; kiểm tra `learned_styles`/`learned_at` bằng SQL.

**Checkpoint**: US4 chạy độc lập; learning bất đồng bộ, idempotent, không đụng gu khai.

---

## Phase 7: User Story 5 - Người dùng mới được gợi ý gu từ chính tủ đồ (Priority: P2)

**Goal**: API cold-start trả top style + itemCount/ratio/sampleSize, không tự lưu.

**Independent Test**: Tủ có phân bố style rõ → suggestions đúng; không xác nhận → profile không đổi.

### Implementation

- [ ] T068 [P] [US5] Thêm `SuggestStyles` vào
  `.../personalization/profile/profile_uc.go`: kiểm tra entitlement (403 nếu thiếu), lấy active items
  qua `IWardrobeContract`, lấy style qua `fashionItemRepo.GetByIDs`, normalize nhãn, đếm top style,
  tính `ratio`.
- [ ] T069 [US5] Thêm handler `GetStyleSuggestions` trong
  `.../fashion/presentation/handler/style_profile_handler.go` và route
  `GET /me/style-profile/suggestions` trong `router.go`; swagger annotation.
- [ ] T070 [US5] Thêm test `profile_uc_test.go` (sampleSize=0, normalize, ratio, không tự lưu,
  thiếu entitlement → 403) và chạy kịch bản US5 trong `quickstart.md`.

**Checkpoint**: US5 chạy độc lập (phụ thuộc route/CRUD của US1 nhưng không phụ thuộc recommendation).

---

## Phase 8: User Story 6 - Người dùng phản hồi để tinh chỉnh gu (Priority: P3)

**Goal**: Feedback like/dislike/saved cho recommendation, xác thực chủ sở hữu, chống trùng.

**Independent Test**: Gửi like → learned tăng; gửi lại → không signal thứ hai; user khác → 403; snapshot
hết hạn → 404.

### Implementation

- [ ] T071 [US6] Ghi snapshot qua port `IRecommendationSnapshotCache` (impl Redis ở T038) với key
  `personalization:rec:{recommendationId}`, TTL 7 ngày, trong `.../recommendation/workflow.go` (ID đã
  được sinh ở T034); snapshot gồm userId, fashionItemIds, styles, createdAt.
- [ ] T072 [US6] Thêm DTO `SubmitRecommendationFeedbackReq` + `.../personalization/feedback/feedback_uc.go`:
  đọc snapshot qua `IRecommendationSnapshotCache`, xác thực user, signal ID deterministic, gọi
  `IStyleLearningUseCase.ApplySignal`.
- [ ] T073 [US6] Thêm handler + route `POST /ai/outfit-recommendations/feedback` trong `services/commerce/internal/modules/fashion/presentation/handler/fashion_ai_handler.go`
  và `services/commerce/internal/modules/fashion/presentation/router/router.go`; swagger.
- [ ] T074 [US6] Thêm lỗi `ErrRecommendationSnapshotNotFound`/`ErrFeedbackForbidden` vào
  `.../fashion/application/errors/`; test snapshot/dedupe/ownership.
- [ ] T075 [US6] Chạy kịch bản US6 trong `quickstart.md` (bao gồm xoá key Redis để test 404).

**Checkpoint**: US6 chạy độc lập sau US4 (dùng chung learning usecase).

---

## Phase 9: User Story 7 - Hoàn thiện một bộ đồ đã có (Priority: P3)

**Goal**: Endpoint completions cho tập item đã sở hữu, chỉ trả suggestions, không trừ quota.

**Independent Test**: 2 món thiếu footwear → suggestions footwear; món người khác → 403; đủ cấu trúc →
rỗng; quota không đổi.

### Implementation

- [ ] T076 [US7] Thêm DTO `CompleteOutfitReq`/`CompleteOutfitRes` trong
  `.../fashion/application/dto/style_profile.go` (hoặc file mới `outfit_completion.go`).
- [ ] T077 [US7] Implement `.../personalization/completion/completion_uc.go`: validate ownership qua
  `IWardrobeContract`, xác định role hiện có/thiếu, tái sử dụng suggestion builder của US2.
- [ ] T078 [US7] Thêm handler + route `POST /ai/outfit-completions` trong
  `fashion_ai_handler.go` / `router.go`; swagger; đảm bảo không gọi `GetAndResetDailyQuota`.
- [ ] T079 [US7] Thêm test ownership/completion + chạy kịch bản US7 trong `quickstart.md`, xác nhận
  `remainingQuota` không đổi.

**Checkpoint**: US7 chạy độc lập sau US2 (dùng chung suggestion builder).

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Tài liệu, ADR, frontend guide (web + mobile), gate cuối, deferred.

- [ ] T080 [P] Cập nhật `docs/system-design/ai-pipeline.md`: thêm gu cá nhân, stale slots/cooldown,
  suggestions, learning; ghi rõ các tín hiệu recency mới.
- [ ] T081 [P] Cập nhật `docs/system-design/architecture.md`: inventory bảng mới (fashion 2 bảng + cột
  wardrobe + cột plan), quan hệ, index; ghi rõ dung sai cache quyền lợi 60 giây (FR-004) và cách vận
  hành khi thu hồi quyền lợi.
- [ ] T082 [P] Cập nhật API docs: `docs/api/fashion/ai-styling-api.md` (recommendation response +
  feedback + completions + style profile), `docs/api/wardrobe/outfits-api.md` (`itemContext` + clone),
  `docs/api/wardrobe/wardrobe-api.md` (catalog-init chỉ nhận system catalog),
  `docs/api/subscription/subscription-api.md` + `docs/api/subscription/admin-subscription-api.md`
  (`personalizationEnabled`).
- [ ] T083 [P] Viết `docs/decisions/017-personalization-style-profile.md` (Context/Decision/Consequences,
  tham chiếu ADR 015) và thêm dòng vào `docs/decisions/README.md`.
- [ ] T084 [P] Viết `specs/033-personalized-styling/frontend-guide.md` cho **cả web và mobile**:
  - Đọc source web `../smart-wardrobe-fe`:
    `src/features/ai-stylist/types/index.ts`,
    `src/features/ai-stylist/components/AIStylistClient.tsx`,
    `src/features/outfits/types/index.ts`, `src/features/outfits/api/*`,
    `src/app/(guest)/auth/register/preferences/components/PreferencesClient.tsx`,
    `src/features/wardrobe/types/index.ts`,
    `src/app/(user)/wardrobe/explore/components/SystemCatalogClient.tsx`,
    `src/features/subscription/types/index.ts`, `src/lib/axios.ts`.
  - Đọc source mobile `../smart-wardrobe-mobile`:
    `lib/features/outfit_studio/models/outfit_models.dart`,
    `lib/features/outfit_studio/data/outfit_repository.dart`,
    `lib/features/outfit_studio/providers/ai_outfit_provider.dart`,
    `lib/features/outfit_studio/providers/outfit_studio_provider.dart`,
    `lib/features/auth/presentation/preferences_screen.dart`,
    `lib/features/auth/data/auth_repository.dart`,
    `lib/features/wardrobe/presentation/system_catalog_screen.dart`,
    `lib/features/wardrobe/data/wardrobe_repository.dart`,
    `lib/features/profile/data/payment_repository.dart`, `lib/core/network/api_client.dart`.
  - Guide phải liệt kê từng endpoint/event **mới/đổi/bị xoá** kèm method + path, và với mỗi **đổi** ghi
    **trước → sau**; nêu rõ web cần mở rộng `AIOutfitRecommendationRes`, `SaveOutfitItemReq`, wiring
    `PreferencesClient`; mobile cần thêm `RecommendedOutfitRes.recommendationId/suggestions`, thêm
    `OutfitRepository.updateOutfit` (hiện mobile chưa có PUT), wiring `preferences_screen.dart`, và
    thống nhất model Studio vs Stylist.
- [ ] T085 [P] Cập nhật `specs/033-personalized-styling/quickstart.md` với mục "Kết quả kiểm chứng" sau khi
  chạy hết các kịch bản (kịch bản nào đã chạy, kết quả, lệnh tái lập).
- [ ] T086 Chạy `make generate` (wire + swagger + nginx route map) và kiểm tra diff chỉ gồm thay đổi của
  feature.
- [ ] T087 Chạy gate đầy đủ: `make migration-check`, `make lint`,
  `go test -race -count=3` cho các package commerce/fashion, commerce/wardrobe, account/subscription,
  `make test-db-integration` (sau T093, phải chạy cả commerce), `make buf-lint`, `make buf-gen` +
  `git diff --exit-code contracts/`, cuối cùng `make check`.
- [ ] T088 Rà soát phát hiện ngoài phạm vi và ghi vào `specs/033-personalized-styling/deferred.md`
  (là gì, vì sao ngoài phạm vi, gì sẽ gỡ); nếu không, ghi rõ "không có phát hiện ngoài phạm vi". Ghi rõ
  inferred-vector prior là deferred enhancement.
- [ ] T089 [P] Rà soát bảo mật và ghi kết quả vào `specs/033-personalized-styling/quickstart.md` (hoặc
  `specs/033-personalized-styling/deferred.md`): catalog-init cross-user, ownership feedback/completion,
  không log PII, clone limit race.
- [ ] T090 [P] Đo p95 recommendation trên scratch DB với catalog cache hit/miss; ghi số đo vào `docs/development/technical-notes.md`:
  kết luận SC-010 đạt/không đạt và hành động tiếp theo.
### Frontend Integration Guide *(required — hiến pháp §Governance)*

**Có đổi bề mặt client** → giữ nhánh này; đã giao cho T084 và T091.

- [ ] T091 Đồng bộ `specs/033-personalized-styling/frontend-guide.md` ở phase cuối, SAU khi mã nguồn ổn
  định: đọc lại guide và đối chiếu với source backend + hai repo client, có đếm số — mọi trường JSON của
  DTO, mọi giá trị enum, mọi route, mọi mã HTTP. Sửa mọi chỗ lệch; nếu phần nào chưa kiểm được, ghi rõ.

### Phát hiện ngoài phạm vi *(hiến pháp §Governance)*

- [ ] T092 Ghi mọi phát hiện ngoài phạm vi vào `specs/033-personalized-styling/deferred.md`, không để
  nguyên checkbox ở file này; nêu rõ task nào bị ảnh hưởng nếu có.
- [ ] T093 Mở rộng `Makefile` target `test-db-integration` để chạy
  `go test -race -p 1 -count=1 -timeout 20m -run Integration ./...` trong `services/commerce` (giữ account/ai); thêm ghi
  chú vào `docs/development/local-setup.md`. Đây là điều kiện để T022/T087 kiểm chứng SC-006.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: không phụ thuộc; bắt đầu ngay.
- **Foundational (Phase 2)**: phụ thuộc Setup; **chặn toàn bộ user story**.
- **US1 (Phase 3)**: sau Foundational; MVP.
- **US2 (Phase 4)**: sau Foundational; độc lập US1 (dùng helper chung ở Foundational).
- **US3 (Phase 5)**: sau Foundational; độc lập US1/US2.
- **US4 (Phase 6)**: sau Foundational; độc lập US1/US2/US3.
- **US5 (Phase 7)**: sau US1 (dùng route/CRUD style profile).
- **US6 (Phase 8)**: sau US4 (dùng learning usecase) và US1 (recommendation response).
- **US7 (Phase 9)**: sau US2 (dùng suggestion builder).
- **Polish (Phase 10)**: sau tất cả story mong muốn.

### User Story Dependencies

| Story | Phụ thuộc | Ghi chú |
|---|---|---|
| US1 | Foundational | MVP |
| US2 | Foundational | Có thể làm song song US1 |
| US3 | Foundational | Có thể làm song song US1/US2 |
| US4 | Foundational | Có thể làm song song US1/US2/US3 |
| US5 | US1 | Dùng chung route/handler |
| US6 | US1, US4 | Feedback + learning |
| US7 | US2 | Dùng chung suggestion builder |

### Within Each Story

- DTO/mapper trước usecase.
- Usecase trước handler/router.
- Clone/event phải nằm trong `IUnitOfWork`.
- Test sau implementation tương ứng; chạy scoped trước khi chuyển story.

---

## Parallel Example: US1

```text
T025 DTO style profile
T026 mapper style profile
T029 personalization helper
T030 blend vector
T031 scoring
T032 prompt
T033 HSL fallback
```

Các task trên chạm file khác nhau; T027/T028 phụ thuộc T025/T026; T034 phụ thuộc T029–T033.

## Parallel Example: US2

```text
T037 suggestion DTO/mapper
T042 SaveOutfitItemReq itemContext
T047 errors
T038 catalog cache
```

T039/T040 phụ thuộc T038 và retrieval hiện có; T043–T045 phụ thuộc T042; T046 độc lập.

---

## Implementation Strategy

### MVP First (User Story 1)

1. Hoàn tất Phase 1 + Phase 2.
2. Hoàn tất US1.
3. Chạy kịch bản US1 trong `quickstart.md`.
4. Demo: user premium khai gu và thấy recommendation thay đổi.

### P1 đầy đủ

US1 + US2: gu + gợi ý bổ sung + clone. Đây là bản phát hành P1 có giá trị đầy đủ.

### Incremental Delivery

1. Foundational.
2. US1 (MVP) → validate.
3. US2 (P1) → validate.
4. US3 (rotation) → validate.
5. US4 (learning) → validate.
6. US5 (cold start) → validate.
7. US6 (feedback) → validate.
8. US7 (completions) → validate.
9. Polish + frontend guide + gate.

### Parallel Team Strategy

- Sau Foundational: Dev A US1, Dev B US2, Dev C US3, Dev D US4.
- US5 sau US1; US6 sau US4; US7 sau US2.
- Frontend guide (T084) có thể bắt đầu sau khi contracts ổn định; T091 bắt buộc sau implementation.

---

## Notes

- `[P]` = file khác nhau, không phụ thuộc task chưa xong.
- Không hard-filter theo gu; không thêm candidate ngoài ngân sách prompt 20 + 3 < 24.
- Mọi thay đổi API phải additive; cache cũ thiếu field → `false`.
- Learning idempotent bằng `signal_id`; outbox at-least-once.
- Không log PII; chỉ internal user id + counters.
- Frontend guide phải bao phủ **cả web và mobile**, có before→after cho từng endpoint đổi.
