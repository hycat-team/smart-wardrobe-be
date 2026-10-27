# Implementation Plan: Cá nhân hoá phối đồ và gợi ý bổ sung

**Branch**: `033-personalized-styling` | **Date**: 2026-10-27 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/033-personalized-styling/spec.md`

## Summary

Feature bổ sung bốn năng lực liên quan nhưng độc lập triển khai được:

1. **Hồ sơ gu cho gói trả phí**: CRUD hồ sơ gu (phong cách yêu thích, phong cách tránh, bật/tắt) cùng
   biểu diễn ngữ nghĩa của gu; áp dụng như ưu tiên mềm trong pipeline phối đồ.
2. **Gợi ý bổ sung ngoài tủ**: sau khi tạo outfit, phát hiện vai trò thiếu/chưa phù hợp, tìm món trong
   tủ đồ hệ thống, trả về mảng `suggestions` riêng; khi lưu outfit, tự tạo bản sao thuộc người dùng
   trong cùng transaction.
3. **Học gu từ hành vi**: mỗi lần lưu outfit phát một sự kiện qua outbox; worker cập nhật gu suy ra
   (`learned_styles`, `learned_embedding`) với cơ chế suy giảm; gu người dùng khai không bao giờ bị ghi
   đè. Phản hồi like/dislike dùng chung một usecase học.
4. **Xoay vòng đồ lâu không dùng**: sửa tín hiệu recency (thêm `last_recommended_at`), mở rộng cửa sổ
   phạt, sửa cách tính "món mới", dành chỗ trong tập ứng viên cho đồ lâu không dùng, và ưu tiên theo
   vai trò khi món đầu tiên đang trong vùng làm mát.

Cách tiếp cận: giữ toàn bộ cá nhân hoá là **ưu tiên mềm**; không thêm hard filter vào retrieval để
không phá cấu trúc outfit. Gu được lưu ở Commerce (`fashion` schema) vì chỉ pipeline phối đồ dùng và
đã có sẵn embedding provider + pgvector. Quyền lợi gói đi qua `subscription_plans.personalization_enabled`
→ benefit snapshot → contract DTO → Redis cache → gRPC proto → Commerce. Việc học gu đi qua outbox +
RabbitMQ để không làm chậm/không làm hỏng thao tác lưu outfit.

## Technical Context

**Language/Version**: Go 1.25+ (multi-module Go workspace, `go.work`); Gin, GORM, Wire, Goose.

**Primary Dependencies**: PostgreSQL + pgvector (đã dùng cho `fashion.fashion_items.embedding`), Redis
(overview cache, recommendation snapshot, catalog ID cache), RabbitMQ + transactional outbox, gRPC +
protobuf (`contracts/subscription/v1`), Swag. Embedding đi qua `IAIOrchestrator` (Gemini/OpenAI) — operation
`embedding` đang là always-free trong AI cost policy.

**Storage**:
- Account: thêm cột `subscription.subscription_plans.personalization_enabled` (migration Account).
- Commerce: bảng mới `fashion.user_style_profiles`, `fashion.style_signals`; cột mới
  `wardrobe.wardrobe_items.last_recommended_at` (migration Commerce).
- Redis: `personalization:rec:{recommendationId}` (snapshot phục vụ feedback, TTL 7 ngày);
  `personalization:catalog:v1` (danh sách fashion item ID của tủ hệ thống, TTL 5 phút).
- Không thêm bảng snapshot recommendation vào DB; không thêm index HNSW cho vector gu.

**Testing**: `make lint`; `go test -race -count=3` scoped cho các package đã sửa; DB-backed integration
tests qua `make test-db-integration` cho migration + outbox + repo mới; contract tests cho proto field
mới và cache entry; `make migration-check`; `make check` trước khi push.

**Target Platform**: Linux containers (VPS Compose và managed platform); phát triển local trên Windows
qua Makefile.

**Project Type**: Backend đa service (Account, Commerce Core, AI Engine).

**Performance Goals**: gợi ý phối đồ p95 < 5 giây (SC-010). Các thao tác mới không thêm LLM call:
style profile chỉ đọc 1 row; catalog suggestions/completions chỉ chạy retrieval + template reason;
learning chạy bất đồng bộ. Style embedding chỉ sinh khi lưu hồ sơ gu (không sinh lúc gợi ý).

**Constraints**:
- Prompt budget hiện tại `20 + 3 < 24`; không thêm candidate vào prompt của outfit. Stale slots nằm
  trong ngân sách pool 20, không cộng thêm.
- Không thêm hạ tầng mới; tái sử dụng PostgreSQL/Redis/RabbitMQ/pgvector.
- API thay đổi theo hướng additive; client cũ vẫn đọc được.
- Outbox là at-least-once → learning phải idempotent.
- Migration phải chạy online: thêm cột nullable/default và bảng mới, không rewrite bảng lớn.
- Không hard-filter theo gu; không phá cấu trúc outfit hiện có.

**Scale/Scope**: 200k user; 1 row hồ sơ gu/user; khoảng 1 signal cho mỗi lần lưu outfit; tủ hệ thống
có thể vài nghìn món; catalog ID cache giữ danh sách trong Redis thay vì quét lại mỗi request.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Verdict |
|---|---|---|
| I. Independently deployable services | Không thêm service; Account và Commerce vẫn deploy độc lập | PASS |
| II. Module boundaries & DTO contracts | Hồ sơ gu ở Commerce; cross-module qua `IWardrobeContract`, `IUserSubscriptionContract`; event payload đặt trong `application/dto`; không import entity/repo chéo module | PASS |
| III. Async event-driven processing | Learning gu dùng outbox + RabbitMQ, ghi outbox trong cùng transaction với outfit | PASS |
| IV. Clean architecture layering | Domain repo interface → infrastructure GORM → application usecase → handler mỏng | PASS |
| V. Transaction & error handling | Clone + outfit + outbox trong `IUnitOfWork`; lỗi qua `apperror` | PASS |
| VI. AI pipeline, cost control & fallback | Embedding fail-soft; không thêm paid LLM call; HSL fallback áp dụng cùng ưu tiên mềm; response schema không đổi | PASS |
| VII. Design patterns | Mapper trong `application/mapper`; `Get...` naming; helper `_helper.go`; DTO camelCase | PASS |
| VIII. Infrastructure portability | Chỉ dùng PostgreSQL + pgvector + Redis + RabbitMQ đã có; không vendor lock-in | PASS |
| Governance: Goose migrations | Migration Account cho plan; Commerce cho fashion/wardrobe; `make migration-check` phải xanh | PASS |
| Governance: frontend guide | Có thay đổi endpoint/DTO → bắt buộc `specs/033-personalized-styling/frontend-guide.md` | PASS (planned artifact) |
| Governance: ADR | Quyết định vị trí hồ sơ gu + learning + rotation cần ADR mới | PASS (planned artifact) |

Không có vi phạm hiến pháp. `Complexity Tracking` để trống.

**Post-design re-check (Phase 1)**: vẫn PASS. Catalog retrieval không join chéo module: Wardrobe contract
trả danh sách `fashion_item_id` của tủ hệ thống, Fashion chạy hybrid retrieval trên tập ID đó. Event
payload là DTO application, không phải entity. Learning idempotent bằng `signal_id`; outbox at-least-once
không tạo cập nhật kép.

## Project Structure

### Documentation (this feature)

```text
specs/033-personalized-styling/
├── plan.md                              # File này
├── research.md                          # Phase 0 output
├── data-model.md                        # Phase 1 output
├── quickstart.md                        # Phase 1 output
├── contracts/
│   └── personalization-api.md           # Phase 1 output (HTTP + gRPC + event)
├── frontend-guide.md                    # Bắt buộc theo constitution (viết ở phase implementation)
├── checklists/requirements.md
├── deferred.md                          # Luôn tạo ở Phase 10; ghi "không có phát hiện" nếu rỗng
└── tasks.md                             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
services/account/
├── database/migrations/
│   └── <ts>_add_plan_personalization.sql          # add column + enable paid plans + down
├── dto/subscription/
│   ├── billing_dto.go                             # plan DTO + create/update req
│   └── contract_dto.go                            # UserSubscriptionContractDTO
├── contract/subscription/
│   ├── subscription_contract.go
│   ├── subscription_contract_impl.go
│   └── transport/{to_proto.go,mapper.go,mapper_test.go}
└── internal/modules/subscription/
    ├── domain/entities/plan.go                    # PersonalizationEnabled
    ├── application/mapper/{plan.go,subscription.go}
    ├── application/interface/caching/overview_cache.go
    ├── application/usecase/plan/plan_uc.go
    ├── application/usecase/subscription/internal_subscription_apply_workflow.go
    └── application/usecase/subscription/subscription_uc.go (no change expected)

contracts/subscription/v1/
├── subscription.proto                             # UserSubscriptionOverview field 11
└── *.pb.go                                        # make buf-gen

services/commerce/
├── database/migrations/
│   └── <ts>_add_personalization.sql               # fashion tables + wardrobe column + indexes + down
└── internal/
    ├── bootstrap/worker.go                        # StyleLearning worker
    ├── di/wire.go                                 # (provider set already includes fashion)
    ├── modules/fashion/
    │   ├── domain/entities/
    │   │   ├── user_style_profile.go
    │   │   ├── style_signal.go
    │   │   └── table_name.go
    │   ├── domain/repositories/interfaces.go       # IUserStyleProfileRepository, IStyleSignalRepository
    │   ├── infrastructure/persistence/
    │   │   ├── user_style_profile_repo.go
    │   │   ├── style_signal_repo.go
    │   │   └── style_signal_repo_test.go
    │   ├── infrastructure/caching/                  # Redis impl for personalization ports
    │   ├── application/dto/
    │   │   ├── style_profile.go                    # request/response + suggestion DTOs
    │   │   └── recommendation.go                   # recommendationId, suggestions, personal style
    │   ├── application/mapper/
    │   │   ├── style_profile.go
    │   │   └── recommendation.go
    │   ├── application/errors/                      # personalization error constructors
    │   ├── application/interface/caching/           # IPersonalizationCatalogCache, IRecommendationSnapshotCache
    │   ├── application/interface/usecase/
    │   │   ├── style_profile_uc.go
    │   │   ├── style_learning_uc.go
    │   │   ├── outfit_completion_uc.go
    │   │   └── outfit_feedback_uc.go
    │   ├── application/usecase/personalization/
    │   │   ├── shared/style_lexicon.go             # enrich text + normalize labels
    │   │   ├── shared/embedding_helper.go          # build/merge/normalize vectors
    │   │   ├── profile/profile_uc.go               # CRUD + cold-start
    │   │   ├── learning/learning_uc.go             # worker handler + feedback signal apply
    │   │   ├── feedback/feedback_uc.go
    │   │   └── completion/completion_uc.go
    │   ├── application/usecase/ai/recommendation/
    │   │   ├── usecase.go                          # deps + style repo + cache
    │   │   ├── workflow.go                         # load profile, suggestions, snapshot, observe
    │   │   ├── workflow_helpers.go                 # filterCandidates + suggestions
    │   │   ├── personalization_helper.go           # intent merge + blend weights
    │   │   ├── candidate_helpers.go                # style vector blend
    │   │   ├── ranking/scoring.go                  # personal/learned/avoid + recency fix
    │   │   ├── ranking/ranking.go                  # cooldown adjustment + stats
    │   │   ├── ranking/pool_expansion.go            # stale candidate append helper
    │   │   ├── ranking/fallback_hsl.go              # personalization-aware fallback
    │   │   ├── synthesis/prompt.go                 # personal_style context
    │   │   ├── synthesis/generation.go             # system prompt rule
    │   │   └── types/structs.go                    # stale source, cooldown, stats
    │   ├── presentation/handler/
    │   │   ├── style_profile_handler.go
    │   │   └── fashion_ai_handler.go               # recommendation + feedback + completions
    │   ├── presentation/router/router.go            # new routes
    │   ├── presentation/worker/style_learning_worker.go
    │   └── provider.go
    └── modules/wardrobe/
        ├── domain/entities/wardrobe_item.go         # LastRecommendedAt
        ├── domain/repositories/interfaces.go        # TouchLastRecommendedAt, GetStaleActiveItems,
        │                                            # GetActiveSystemCatalogItemsByIDs
        ├── infrastructure/persistence/wardrobe_repo.go
        ├── application/dto/
        │   ├── contract_dto.go                      # LastRecommendedAt
        │   ├── outfit.go                            # ItemContext optional
        │   └── outfit_event.go                      # OutfitSavedEventPayload
        ├── application/mapper/wardrobe.go
        ├── application/usecase/outfit/outfit_uc.go  # UoW + clone + event + touch
        ├── application/usecase/wardrobe/catalog/catalog_uc.go  # fix catalog-init validation
        ├── application/errors/errors.go
        ├── application/interface/usecase/contract_uc.go
        ├── application/usecase/wardrobe/contractuc/contract_uc.go
        ├── presentation/contract/
        │   ├── wardrobe_contract.go                 # GetSystemCatalogFashionItemIDs, GetOwnedFashionItemIDs,
        │   │                                        # GetStaleActiveWardrobeItemsByUserID, TouchLastRecommendedAt
        │   └── wardrobe_contract_impl.go
        └── presentation/handler/outfit_handler.go

shared/
├── application/constants/eventconstants/
│   ├── eventconstants.go                            # TopicOutfitSaved
│   └── queues.go                                    # QueueStyleLearning
├── infrastructure/messaging/
│   ├── rabbitmq_constants.go                        # RoutingKeyOutfitSaved
│   └── topology.go                                  # binding + DLQ
├── config/config.go                                 # Personalization + RAG fields
└── pkg/utils/sliceutils/sliceutils.go               # NormalizeVector (single home)

configs/config.yaml
configs/config.production.yaml
.gitignore                                           # .specify/feature.json

docs/
├── decisions/017-personalization-style-profile.md   # ADR mới
├── decisions/README.md                              # index
├── system-design/ai-pipeline.md
├── system-design/architecture.md
├── api/fashion/ai-styling-api.md
├── api/subscription/subscription-api.md
├── api/subscription/admin-subscription-api.md
├── api/wardrobe/outfits-api.md
└── api/wardrobe/wardrobe-api.md
```

**Structure Decision**: hồ sơ gu và tín hiệu học thuộc `fashion` (Commerce) vì chỉ pipeline phối đồ
dùng, đã có embedding provider + pgvector, và không cần thêm dependency AI vào Account. Quyền lợi gói
thuộc Account vì `subscription_plans` là Account-owned; Commerce chỉ đọc qua contract. Wardrobe sở hữu
`wardrobe_items` nên các tín hiệu `last_used_at`/`last_recommended_at` và clone nằm ở Wardrobe.

## Design Decisions

### D1 — Hồ sơ gu ở `fashion`, không ở `identity`

**Quyết định**: tạo `fashion.user_style_profiles` (1 row/user) với `styles`, `excluded_styles`,
`enabled`, `style_embedding vector(768)`, `embedding_hash`, `embedding_model`, `learned_styles`,
`learned_embedding`, `learned_at`.

**Lý do**: Account hiện không có dependency AI nào; đặt ở Account buộc phải thêm embedding provider vào
service identity hoặc sinh embedding bất đồng bộ qua service khác. Commerce đã có `IAIOrchestrator`,
config dimension 768, pgvector và toàn bộ consumer của gu. ADR 015 đã xoá bảng cũ vì feed bị gỡ; feature
này mở lại phạm vi **recommendation**, không phải feed, nên cần ADR mới thay vì tái sử dụng bảng cũ.

**Loại bỏ**: thêm cột vào `identity.users`/`body_profile` (semantic sai, Account phải gọi AI); tái tạo
`identity.user_style_profiles` (mở lại quyết định 015 mà không có lý do mới).

### D2 — Quyền lợi gói đi qua chuỗi snapshot → contract → cache → proto

**Quyết định**: thêm `personalization_enabled boolean NOT NULL DEFAULT false` vào `subscription_plans`;
migration bật `true` **một lần** cho các plan đang active có `price > 0` (backfill cho premium hiện có);
gói khuyến mãi giá 0 và gói tạo sau khi triển khai mặc định tắt, admin bật thủ công. Cờ runtime là
nguồn sự thật; không dùng `price > 0` để quyết định quyền lợi khi chạy. Field đi qua hai nhánh của cùng
một plan row: (a) `SubscriptionPlan` → `BuildPlanBenefitSnapshot` (snapshot quyền lợi lúc grant); (b)
`SubscriptionPlan` → `BuildUserSubscriptionOverviewRes` → `UserSubscriptionContractDTO` →
`SubscriptionOverviewCacheEntry` → proto `UserSubscriptionOverview` (field 11) → Commerce. Overview đọc
trực tiếp plan row, không đọc lại snapshot. Cache TTL 60 giây là dung sai đã biết và đã được ghi trong
spec FR-004/Assumptions.

**Lý do**: đây là chuỗi hiện có cho `MaxWardrobeItems`/quota; thêm một field theo đúng pattern sẽ không
tạo nguồn sự thật thứ hai. Cache overview TTL 60 giây chấp nhận trễ tối đa 60 giây khi admin bật/tắt
quyền lợi; không cần sweep cache.

**Loại bỏ**: dùng `price > 0` làm entitlement (không cross-service được, sai với gói khuyến mãi);
middleware gating (không tồn tại và không cần cho một field dữ liệu).

### D3 — Style embedding sinh lúc lưu hồ sơ, không sinh lúc gợi ý

**Quyết định**: PUT hồ sơ gu chuẩn hoá dữ liệu, ghép một câu mô tả gu (enrich known styles bằng lexicon
nội bộ), gọi `GenerateEmbeddings` với timeout ngắn, lưu `style_embedding` + `embedding_hash`. Nếu gọi
thất bại: vẫn lưu `styles`, để `style_embedding = NULL` **và `embedding_hash = NULL`** để lần PUT sau
thử lại; recommendation chuyển sang so khớp từ khoá. `embedding_hash` chỉ được ghi khi embed thành công;
hash không đổi thì bỏ qua gọi AI. Nếu người dùng không PUT lại, recommendation vẫn chạy ở mức text-level
cho tới lần cập nhật kế tiếp (đúng FR-011). Kill switch `personalization.enabled = false` được kiểm tra
ở profile usecase (chặn PUT/DELETE) và learning usecase (ack, không ghi); recommendation bỏ qua gu.

**Lý do**: request gợi ý không phải chờ thêm một provider call; operation embedding đang free nên chi phí
chỉ phát sinh khi người dùng đổi gu. Fail-soft giữ đúng FR-010/FR-011.

**Loại bỏ**: embed tại thời điểm gợi ý (thêm latency + failure mode vào hot path); chỉ dùng raw text
trong semantic query (mất trọng số gu, dễ bị ngữ cảnh request lấn át); lưu vector mà không có hash
(không biết khi nào cần embed lại).

### D4 — Gu là ưu tiên mềm; blend vector + scoring + prompt + fallback

**Quyết định**:
- `q = normalize(queryVector)`; nếu có style vector: `q = normalize((1-w)·q + w·normalize(styleVector))`,
  `w` mặc định 0.30 (config).
- Request `styleTarget` thắng gu lưu sẵn; khi mâu thuẫn, hạ `w` xuống 0.10.
- `profile.enabled = false` → bỏ qua toàn bộ gu (context rỗng), kể cả khi còn quyền lợi gói; đây là
  điều kiện bắt buộc cho US1 AS5/FR-012.
- Scoring thêm: `personal-style-match` (+0.25 nếu không có request style, +0.10 nếu có),
  `learned-style-match` (+0.10), `avoid-style` (−0.30). Style labels được normalize qua taxonomy hiện có.
- Prompt thêm `personal_style`, `learned_style`, `avoid_styles` vào `contextPayload`; system prompt thêm
  rule ưu tiên gu khi chất lượng tương đương.
- HSL fallback dùng cùng intent/scoring để không mất personalization khi AI lỗi.
- **Fix bắt buộc**: normalize vector sau `AdjustVectorLength` (hàm hiện chỉ truncate/pad).

**Lý do**: giữ nguyên precedent "category là soft filter" và bảo toàn cấu trúc outfit. Blend vector cho
phép gu ảnh hưởng retrieval mà không cần hard SQL filter.

**Loại bỏ**: hard filter SQL theo style (phá outfit, cần sửa fallback + structure guard); thêm branch
RRF thứ ba (tăng độ phức tạp, khó cân trọng số); chỉ dựa vào prompt (không thay đổi retrieval).

### D5 — Gợi ý bổ sung: gap-fill sau outfit, không nhét catalog vào prompt

**Quyết định**:
- Wardrobe contract thêm `GetSystemCatalogFashionItemIDs(ctx, limit)` (pure, chỉ đọc repo của Wardrobe).
- Fashion cache danh sách ID này qua port `IPersonalizationCatalogCache` (impl Redis ở
  `infrastructure/caching`, key `personalization:catalog:v1`, TTL 5 phút), rồi gọi
  `GetHybridCandidates` với tập ID đó. Loại các `fashion_item_id` người dùng đã sở hữu (định nghĩa:
  tồn tại row `wardrobe_items` chưa xoá, bất kỳ status; lấy qua `GetOwnedFashionItemIDs`).
- Gap detection: role không có món nào, hoặc role chỉ có ứng viên dưới ngưỡng `gap_score_threshold`.
- Retrieval catalog dùng **cùng** semantic query, style vector và hard filters như outfit (dịp, mùa,
  thời tiết, tông màu, gu cá nhân đang bật); reason template tiếng Việt nêu ngữ cảnh/gu.
- Trả `suggestions` tối đa 1 món/role, 3 món/response; không tính vào outfit, không trừ quota, không
  chạy LLM.
- Outfit completion endpoint tái sử dụng đúng builder này với tập item người dùng gửi lên.

**Lý do**: catalog trong prompt dễ khiến LLM ghép bộ toàn đồ catalog, tăng prompt budget và phá cam kết
"phối từ tủ của bạn". Contract + cache giữ đúng ranh giới module (không join `wardrobe_items` từ Fashion).

**Loại bỏ**: join trực tiếp `wardrobe_items` trong Fashion (vi phạm §II); đưa catalog vào prompt (rủi ro
chất lượng + budget); lưu bảng projection catalog riêng (đồng bộ phức tạp, không cần).

### D6 — Lưu outfit: clone catalog trong cùng transaction

**Quyết định**:
- `SaveOutfitItemReq` thêm `itemContext` optional (`user_wardrobe|brand_item|system_catalog`).
- Resolve: chỉ định rõ → đúng nguồn; không chỉ định → wardrobe → brand eligible → catalog.
- Catalog hợp lệ: `item_type = SystemCatalogItem`, `status = InWardrobe`, `is_deleted = false`,
  `fashion_item_id != nil`. Nếu người dùng **đã sở hữu** `fashion_item_id` (tồn tại row
  `wardrobe_items` chưa xoá, bất kỳ status), tái sử dụng row đó, không clone thêm. Nếu chưa sở hữu, tạo
  row `wardrobe_items` mới (`item_type = UserItem`, copy price); outfit item tham chiếu cùng
  `fashion_item_id` với `item_context = user_wardrobe`.
- Toàn bộ clone + `CreateWithItems` + `TouchLastUsedAt` + outbox publish nằm trong `IUnitOfWork`.
- Kiểm tra `MaxWardrobeItems` trước khi clone; vượt → từ chối toàn bộ, lỗi nêu rõ item.
- `UpdateOutfit` dùng cùng logic; chỉ touch `last_used_at` cho item mới thêm.
- Fix `InitClosetFromCatalog`: chỉ chấp nhận `item_type = SystemCatalogItem` và `is_deleted = false`.

**Lý do**: `outfit_items` lưu `fashion_item_id`, nên clone tái sử dụng đúng fashion item và ownership
được bảo đảm bằng row `wardrobe_items` của user. Transaction nguyên tử tránh bản sao mồ côi.

**Loại bỏ**: thêm `item_context = system_catalog` vào `outfit_items` (phải sửa toàn bộ read/delete path
và CHECK constraint); lưu outfit trước rồi clone sau (dữ liệu dở dang); tái sử dụng `InitClosetFromCatalog`
(đang thiếu validate, là lỗ hổng cross-user).

### D7 — Học gu: outbox + worker, idempotent, không ghi đè gu khai

**Quyết định**:
- `outfit.saved` event phát trong cùng transaction, payload gồm `signalId`, `userId`, `outfitId`,
  `fashionItemIds`, `source`, `occurredAt`.
- Worker `StyleLearningWorker` consume `style_learning_queue`, insert `fashion.style_signals` với PK
  `id` = payload `signalId` (ON CONFLICT DO NOTHING) rồi cập nhật hồ sơ trong cùng transaction.
- `learned_styles = topK(normalize(decay·learned_styles + counts(styles)))`, decay mặc định 0.9, K=10.
- `learned_embedding = normalize(decay·old + (1-decay)·normalize(mean(item_embeddings)))`.
- Chỉ item mới thêm khi `UpdateOutfit` mới tạo signal.
- Gu khai (`styles`) không bị sửa; learned có trọng số thấp hơn và không tạo hard filter.
- Feedback like/dislike dùng cùng `StyleLearningUseCase.ApplySignal`; signal ID deterministic cho
  feedback để chống trùng. `saved` feedback là tín hiệu nhẹ (0.25) để không cộng hai lần với sự kiện
  lưu outfit.

**Lý do**: đúng §III (write side effect chéo module qua event), không làm chậm lưu outfit, outbox bảo
đảm at-least-once và `signal_id` biến nó thành exactly-once về mặt hiệu lực.

**Loại bỏ**: cập nhật đồng bộ trong `SaveOutfit` (thêm latency, rollback coupling); database trigger
(không thấy được metadata/style); derive lazy từ signal mỗi lần gợi ý (đọc nhiều embedding, khó cache).

### D8 — Feedback: snapshot recommendation trong Redis

**Quyết định**: `recommendationId` (UUID) được sinh ở US1 cùng response; US6 ghi snapshot
`{userId, fashionItemIds, styles, createdAt}` qua port `IRecommendationSnapshotCache` (impl Redis ở
`infrastructure/caching`) TTL 7 ngày. Feedback endpoint chỉ nhận
`recommendationId + action`; đọc snapshot để xác thực chủ sở hữu và lấy item IDs. Signal ID deterministic
= hash(userId + recommendationId + action). Nếu snapshot hết hạn/không có → từ chối với lỗi rõ ràng.

**Lý do**: tránh bảng snapshot tăng trưởng theo mỗi lượt gợi ý; Redis đã có sẵn. Không tin item IDs do
client gửi lên.

**Loại bỏ**: lưu snapshot vào DB (tăng trưởng nhanh, không cần audit ở v1); token ký HMAC (thêm phức tạp
cho client); không xác thực (feedback giả có thể bẻ gu người khác).

### D9 — Rotation: hai tín hiệu, stale slots, cooldown theo vai trò

**Quyết định**:
- Thêm `wardrobe_items.last_recommended_at`; cập nhật best-effort sau khi trả kết quả cho primary +
  alternatives (chỉ wardrobe item). Lỗi ghi được log và lần gợi ý kế tiếp thử lại; đây là dung sai đã
  được ghi trong FR-054.
- `last_used_at` chỉ touch cho item mới thêm vào outfit; bỏ touch khi update metadata.
- Dedupe theo `fashion_item_id`, lấy `max(last_used_at)` trước scoring.
- Vùng làm mát: dùng trong 7 ngày hoặc được gợi ý trong 3 ngày. Giữ `rag.recently_worn_penalty_days`
  = 7 (nới từ 3), penalty tối đa 0.45; thêm `rag.recently_recommended_penalty_days` = 3 với penalty
  0.12; `rag.stale_candidate_slots` = 4; `rag.cooldown_score_tolerance` = 0.15.
- Sửa bonus: "món mới" theo `created_at`; bỏ điều kiện `styleMatched || occasion` cho long-unworn.
- Dành `stale_candidate_slots` (mặc định 4) trong pool 20 cho item
  `last_used_at NULLS FIRST, last_recommended_at NULLS FIRST` thoả hard filter; append với source `stale`.
  Ngân sách chính xác: `hybridLimit = minimumPool - brandCandidates - staleSlots`; stale item điền đúng
  số slot còn lại và dedupe theo `fashion_item_id`, không cộng thêm vào prompt budget.
- `RankCandidates` thêm bước `ApplyRecencyCooldown`: nếu role có món ngoài vùng làm mát trong ngưỡng
  `cooldown_score_tolerance` (mặc định 0.15), đưa món đó lên trước; nếu cả role đều trong vùng làm mát
  thì giữ nguyên.

**Lý do**: sửa đúng 10 nguyên nhân đã phân tích trong spec (cửa sổ ngắn, bonus bất đối xứng, chỉ rerank,
không nhớ đã gợi ý, touch sai, bản sao trùng, LLM không được dặn). Stale slots là cách duy nhất kéo đồ
cũ vào pool; cooldown theo role bảo đảm thay đổi quan sát được mà không phá cấu trúc.

**Loại bỏ**: chỉ tăng penalty (đồ cũ vẫn không vào pool); hard rotation (có thể tạo outfit kém); thêm
API "đã mặc" (ngoài phạm vi, đã chốt dùng save làm proxy).

### D10 — Cold-start: phân bố từ tủ đồ, không tự áp dụng vector suy ra

**Quyết định**: `GET /me/style-profile/suggestions` lấy active items qua Wardrobe contract, lấy style
qua `fashionItemRepo.GetByIDs`, normalize nhãn đã biết, trả top style kèm `itemCount`, `ratio`,
`sampleSize`. Không tự động lưu; người dùng xác nhận rồi mới PUT. Không dùng inferred embedding làm
prior trong v1 để giữ đúng FR-046 (gợi ý không tự áp dụng); ghi nhận là deferred enhancement.

**Lý do**: dữ liệu đã có sẵn, không cần AI; tránh thay đổi hành vi gợi ý trước khi người dùng đồng ý.

**Loại bỏ**: tự sinh vector trung bình từ tủ và áp dụng ngay (vi phạm tinh thần FR-046); dùng LLM sinh
style labels (tốn chi phí, không cần).

### D11 — Observability: structured log + RerankStats, không dashboard mới

**Quyết định**: mở rộng `RerankStats` với `StaleSlotsAdded`, `CooldownSubstitutions`,
`RecentlyUsedRepeatRatio`, `RecentlyRecommendedRepeatRatio`, `PersonalStyleMatchRatio`,
`ReturnedItemIDs`, `StaleItemIDs`, `StaleEligibleItemIDs`, `CooldownItemIDs`; log một dòng structured mỗi lần gợi ý với
`recommendationId`, `userId` (internal), `personalized`, `candidateCount`, các counter và danh sách ID
trên. Danh sách ID là mã món nội bộ, không phải PII, và đủ để tính SC-004 theo từng món. Không thêm
Prometheus/dashboard.

**Lý do**: đủ để tính SC-003/SC-004 bằng truy vấn log; chi phí thấp; không mở rộng scope vận hành.

**Loại bỏ**: dashboard admin mới (ngoài phạm vi); chỉ dựa vào log cũ (không có counter nên không đo được).

### D12 — Tương thích proto/cache và rollback

**Quyết định**: proto thêm field 11 `personalization_enabled` (additive). Cache entry thêm field với
`json:"personalizationEnabled,omitempty"`; cache cũ không có field → đọc là `false` (an toàn: tắt
personalization cho tới khi cache hết hạn). Response JSON thêm field optional; `recommendationId` và
`suggestions` additive. Migration Down drop đúng các cột/bảng mới; không xoá dữ liệu plan hiện có.

**Lý do**: client cũ vẫn chạy; rollback schema không mất dữ liệu ngoài phạm vi feature.

**Loại bỏ**: đổi tên/kiểu field cũ; backfill `last_recommended_at` (không có nguồn).

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Style embedding lệch do `AdjustVectorLength` không normalize | Chuẩn hoá thủ công trước/sau blend; unit test vector đơn vị; kiểm tra cosine với vector chuẩn |
| Gu ảnh hưởng quá mạnh làm mất đa dạng/occasion | Trọng số cấu hình; request style thắng; hard cap `w`; test scenario mâu thuẫn |
| Catalog suggestions gợi ý món người dùng đã sở hữu | Dedupe theo `fashion_item_id`; test với item đã clone |
| Clone + outfit không nguyên tử | `IUnitOfWork`; integration test rollback khi một item catalog bị gỡ giữa chừng |
| Vượt `MaxWardrobeItems` giữa lúc kiểm tra và insert | Kiểm tra trong transaction; nếu đua, transaction sau fail và rollback toàn bộ; lỗi rõ ràng |
| Outbox giao lặp gây cộng dồn learning | `style_signals.signal_id` PK + `ON CONFLICT DO NOTHING`; test giao 2 lần |
| Học gu thu hẹp gu (rich-get-richer) | Decay 0.9, top-K 10, learned weight thấp, explicit styles luôn thắng, excluded styles luôn trừ |
| Stale slots chiếm chỗ ứng viên tốt | Stale nằm trong pool 20, không cộng thêm; chỉ lấy item thoả hard filter + đúng role; đo SC-003/SC-004 |
| Cooldown làm mất món duy nhất của role | Nếu toàn role trong vùng làm mát, giữ nguyên (FR-060) |
| Proto/cache cũ không có field mới | Additive + `omitempty`; cache miss đọc false; compatibility test |
| Migration Account bật quyền cho sai gói | Chỉ bật plan `is_active` và `price > 0`; admin có thể tắt; test migration trên dữ liệu seed |
| Catalog ID cache cũ sau khi admin thêm/xoá món | TTL 5 phút; validate lại item khi lưu outfit nên món bị gỡ vẫn bị từ chối |
| Feedback snapshot hết hạn | TTL 7 ngày; trả lỗi rõ ràng; không ảnh hưởng learning từ outfit save |
| Log chứa PII | Chỉ log internal user id + aggregate counters; không log style text/email |

## Complexity Tracking

Không có vi phạm hiến pháp. Không cần justify ngoại lệ.
