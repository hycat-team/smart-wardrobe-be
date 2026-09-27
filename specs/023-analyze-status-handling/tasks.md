---
description: "Task list for feature 023-analyze-status-handling"
---

# Tasks: XÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½ kÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ch bÃƒÂ¡Ã‚ÂºÃ‚Â£n theo trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i phÃƒÆ’Ã‚Â¢n tÃƒÆ’Ã‚Â­ch trang phÃƒÂ¡Ã‚Â»Ã‚Â¥c AI

**Input**: Design documents from `/specs/023-analyze-status-handling/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: CÃƒÆ’Ã‚Â³ bao gÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“m task test cho cÃƒÆ’Ã‚Â¡c Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â¡n vÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ logic quan trÃƒÂ¡Ã‚Â»Ã‚Âng (theo quy Ãƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc dÃƒÂ¡Ã‚Â»Ã‚Â± ÃƒÆ’Ã‚Â¡n: `make test-race` lÃƒÆ’Ã‚Â  gate bÃƒÂ¡Ã‚ÂºÃ‚Â¯t buÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c).

**Organization**: Tasks grouped by user story Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ cÃƒÆ’Ã‚Â³ thÃƒÂ¡Ã‚Â»Ã†â€™ triÃƒÂ¡Ã‚Â»Ã†â€™n khai/kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: CÃƒÆ’Ã‚Â³ thÃƒÂ¡Ã‚Â»Ã†â€™ chÃƒÂ¡Ã‚ÂºÃ‚Â¡y song song (file khÃƒÆ’Ã‚Â¡c nhau, khÃƒÆ’Ã‚Â´ng phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c task chÃƒâ€ Ã‚Â°a xong)
- **[Story]**: US1ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“US5 tÃƒâ€ Ã‚Â°Ãƒâ€ Ã‚Â¡ng ÃƒÂ¡Ã‚Â»Ã‚Â©ng spec.md
- Bao gÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“m Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Âng dÃƒÂ¡Ã‚ÂºÃ‚Â«n file chÃƒÆ’Ã‚Â­nh xÃƒÆ’Ã‚Â¡c

## Path Conventions

- Workspace multi-module: `shared/`, `services/commerce/`, `services/account/`, `services/ai/`
- Feature docs: `specs/023-analyze-status-handling/`
- LÃƒâ€ Ã‚Â°u ÃƒÆ’Ã‚Â½: module `wardrobe`, `fashion`, `brand` Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Âu thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c service `commerce` ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ mÃƒÂ¡Ã‚Â»Ã‚Âi regenerate Swagger dÃƒÆ’Ã‚Â¹ng `make swagger svc=commerce` (khÃƒÆ’Ã‚Â´ng cÃƒÆ’Ã‚Â³ `svc=brand`/`svc=wardrobe`)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: ChuÃƒÂ¡Ã‚ÂºÃ‚Â©n bÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ migration, config, env cho toÃƒÆ’Ã‚Â n feature.

- [X] T001 [P] TÃƒÂ¡Ã‚ÂºÃ‚Â¡o Goose migration `services/commerce/database/migrations/<YYYYMMDDHHMMSS>_add_brand_item_failed_status.sql` (Up: drop + add `chk_brand_items_status` gÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“m `'failed'`, schema-qualify `brand.brand_items`; Down: revert vÃƒÂ¡Ã‚Â»Ã‚Â `draft|active|archived`)
- [X] T002 [P] ThÃƒÆ’Ã‚Âªm config `ReplayBufferSize` (mapstructure `replay_buffer_size`, default 100) vÃƒÆ’Ã‚Â  `ReplayBufferTTLSeconds` (mapstructure `replay_buffer_ttl_seconds`, default 300) vÃƒÆ’Ã‚Â o struct `SSE` trong `shared/config/config.go`; thÃƒÆ’Ã‚Âªm block `server.sse.replay_buffer_size` + `server.sse.replay_buffer_ttl_seconds` vÃƒÆ’Ã‚Â o `configs/config.yaml` vÃƒÆ’Ã‚Â  `configs/config.production.yaml`; override env `SSE_REPLAY_BUFFER_SIZE` + `SSE_REPLAY_BUFFER_TTL_SECONDS` trong `shared/config/loader.go`; validation `> 0` cho cÃƒÂ¡Ã‚ÂºÃ‚Â£ hai trong `shared/config/validation.go`
- [X] T003 [P] CÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t `.env.example`, `.env.production.example`, `.env.aws.example` vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi `SSE_REPLAY_BUFFER_SIZE` vÃƒÆ’Ã‚Â  `SSE_REPLAY_BUFFER_TTL_SECONDS` (comment tiÃƒÂ¡Ã‚ÂºÃ‚Â¿ng Anh)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Transport realtime (Redis Pub/Sub) + buffer phÃƒÆ’Ã‚Â¡t lÃƒÂ¡Ã‚ÂºÃ‚Â¡i + chuyÃƒÂ¡Ã‚Â»Ã†â€™n cÃƒÆ’Ã‚Â¡c publisher hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n cÃƒÆ’Ã‚Â³. ChÃƒÂ¡Ã‚ÂºÃ‚Â·n US2 vÃƒÆ’Ã‚Â  US4.

**ÃƒÂ¢Ã…Â¡Ã‚Â ÃƒÂ¯Ã‚Â¸Ã‚Â CRITICAL**: HoÃƒÆ’Ã‚Â n thÃƒÆ’Ã‚Â nh trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc khi bÃƒÂ¡Ã‚ÂºÃ‚Â¯t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â§u US2/US4 (US1 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p, chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° cÃƒÂ¡Ã‚ÂºÃ‚Â§n migration T001 nÃƒÂ¡Ã‚ÂºÃ‚Â¿u chÃƒÂ¡Ã‚ÂºÃ‚Â¡m brand).

- [X] T004 [P] ThÃƒÆ’Ã‚Âªm interface `IRealtimePublisher` (`Publish(ctx, taskId, data)`) vÃƒÆ’Ã‚Â  `IRealtimeRelay` (`Start(ctx)`/`Stop()`), kÃƒÆ’Ã‚Â¨m helper channel `sse:task:{taskId}`, trong `shared/application/sse/realtime.go`
- [X] T005 [P] Implement `RedisRealtimePublisher` (`*redis.Client`, `PUBLISH sse:task:{taskId}`) trong `shared/infrastructure/messaging/redis_realtime_publisher.go`
- [X] T006 [P] Implement `SSERelay` (`PSubscribe sse:task:*` ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ `ISSEManager.Publish`) trong `shared/presentation/sse_relay.go` (log lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i, khÃƒÆ’Ã‚Â´ng chÃƒÂ¡Ã‚ÂºÃ‚Â·n nghiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡p vÃƒÂ¡Ã‚Â»Ã‚Â¥)
- [X] T007 MÃƒÂ¡Ã‚Â»Ã…Â¸ rÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng `SSEManager` trong `shared/application/sse/sse_manager.go`: ring buffer N event/`taskId` (N tÃƒÂ¡Ã‚Â»Ã‚Â« config `replay_buffer_size`), `Subscribe` phÃƒÆ’Ã‚Â¡t lÃƒÂ¡Ã‚ÂºÃ‚Â¡i buffer FIFO cho subscriber mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi; loÃƒÂ¡Ã‚ÂºÃ‚Â¡i event khi vÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£t N hoÃƒÂ¡Ã‚ÂºÃ‚Â·c hÃƒÂ¡Ã‚ÂºÃ‚Â¿t TTL (`replay_buffer_ttl_seconds`) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â KHÃƒÆ’Ã¢â‚¬ÂNG xÃƒÆ’Ã‚Â³a buffer khi subscriber cuÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi cÃƒÆ’Ã‚Â¹ng `Unsubscribe` (Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ reconnect muÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢n sau khi phiÃƒÆ’Ã‚Âªn kÃƒÂ¡Ã‚ÂºÃ‚Â¿t thÃƒÆ’Ã‚Âºc vÃƒÂ¡Ã‚ÂºÃ‚Â«n phÃƒÆ’Ã‚Â¡t lÃƒÂ¡Ã‚ÂºÃ‚Â¡i Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c)
- [X] T008 Ãƒâ€žÃ‚ÂÃƒâ€žÃ†â€™ng kÃƒÆ’Ã‚Â½ provider + bind mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi (`RedisRealtimePublisher` ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ `IRealtimePublisher`, `SSERelay`) trong `services/commerce/internal/di/wire.go`, rÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“i chÃƒÂ¡Ã‚ÂºÃ‚Â¡y `make wire`
- [X] T009 KhÃƒÂ¡Ã‚Â»Ã…Â¸i Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng/dÃƒÂ¡Ã‚Â»Ã‚Â«ng `SSERelay` trong `services/commerce/internal/bootstrap/app.go` cho mode `http|all` (KHÃƒÆ’Ã¢â‚¬ÂNG Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°a vÃƒÆ’Ã‚Â o `bootstrap.Workers`)
- [X] T010 Thay cÃƒÆ’Ã‚Â¡c lÃƒÂ¡Ã‚Â»Ã‚Âi gÃƒÂ¡Ã‚Â»Ã‚Âi SSE publish hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n cÃƒÆ’Ã‚Â³ sang `IRealtimePublisher`: `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/fashion_event_uc.go` (2 chÃƒÂ¡Ã‚Â»Ã¢â‚¬â€) vÃƒÆ’Ã‚Â  `services/commerce/internal/modules/brand/application/usecase/item/fashion_event_uc.go` (2 chÃƒÂ¡Ã‚Â»Ã¢â‚¬â€)

**Checkpoint**: Transport realtime sÃƒÂ¡Ã‚ÂºÃ‚Âµn sÃƒÆ’Ã‚Â ng cho cÃƒÂ¡Ã‚ÂºÃ‚Â£ 2 topology.

---

## Phase 3: User Story 1 - RÃƒÆ’Ã‚Â  soÃƒÆ’Ã‚Â¡t & xÃƒÆ’Ã‚Â¡c nhÃƒÂ¡Ã‚ÂºÃ‚Â­n mÃƒÆ’Ã‚Â³n Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“ AI khÃƒÆ’Ã‚Â´ng chÃƒÂ¡Ã‚ÂºÃ‚Â¯c chÃƒÂ¡Ã‚ÂºÃ‚Â¯n (Priority: P1) ÃƒÂ°Ã…Â¸Ã…Â½Ã‚Â¯ MVP

**Goal**: NgÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Âi dÃƒÆ’Ã‚Â¹ng xÃƒÆ’Ã‚Â¡c nhÃƒÂ¡Ã‚ÂºÃ‚Â­n mÃƒÆ’Ã‚Â³n `needs_review` (cÃƒÆ’Ã‚Â³ danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡) ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ `inWardrobe`.

**Independent Test**: TÃƒÂ¡Ã‚ÂºÃ‚Â¡o mÃƒÆ’Ã‚Â³n `needsReview` cÃƒÆ’Ã‚Â³ category hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡ ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ gÃƒÂ¡Ã‚Â»Ã‚Âi `confirm-review` ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ status `inWardrobe` vÃƒÆ’Ã‚Â  tham gia gÃƒÂ¡Ã‚Â»Ã‚Â£i ÃƒÆ’Ã‚Â½ phÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“.

- [X] T011 [US1] ThÃƒÆ’Ã‚Âªm method `ConfirmWardrobeReview(ctx, userID, itemID, now)` vÃƒÆ’Ã‚Â o `services/commerce/internal/modules/wardrobe/domain/repositories/interfaces.go` + impl atomic (WHERE status = NeedsReview) trong `services/commerce/internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`
- [X] T012 [P] [US1] ThÃƒÆ’Ã‚Âªm `IsWearableCategory(ctx, categoryID uuid.UUID) (bool, error)` vÃƒÆ’Ã‚Â o `services/commerce/internal/modules/fashion/presentation/contract/fashion_contract.go` + impl trong `fashion_contract_impl.go` (dÃƒÆ’Ã‚Â¹ng ÃƒÆ’Ã‚Â¡nh xÃƒÂ¡Ã‚ÂºÃ‚Â¡ categoryÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢wearing role, `other`/khÃƒÆ’Ã‚Â´ng cÃƒÆ’Ã‚Â³ = false)
- [X] T013 [US1] Implement `ConfirmWardrobeItemReview` trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go`: kiÃƒÂ¡Ã‚Â»Ã†â€™m tra ownership + status `NeedsReview`; lÃƒÂ¡Ã‚ÂºÃ‚Â¥y category qua `IFashionContract.GetFashionItem` tÃƒÂ¡Ã‚Â»Ã‚Â« `item.FashionItemID` (nÃƒÂ¡Ã‚ÂºÃ‚Â¿u `FashionItemID == nil` hoÃƒÂ¡Ã‚ÂºÃ‚Â·c fashion item/category nil ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ trÃƒÂ¡Ã‚ÂºÃ‚Â£ `ErrConfirmReviewCategoryRequired`); gÃƒÂ¡Ã‚Â»Ã‚Âi `IsWearableCategory`; chuyÃƒÂ¡Ã‚Â»Ã†â€™n trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i nguyÃƒÆ’Ã‚Âªn tÃƒÂ¡Ã‚Â»Ã‚Â­ vÃƒÆ’Ã‚Â  publish `wardrobe.created` trong `uow.Execute`
- [X] T014 [P] [US1] ThÃƒÆ’Ã‚Âªm lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i `ErrConfirmReviewForbidden`, `ErrConfirmReviewCategoryRequired` trong `services/commerce/internal/modules/wardrobe/application/errors/errors.go`
- [X] T015 [US1] ThÃƒÆ’Ã‚Âªm method vÃƒÆ’Ã‚Â o interface usecase (`.../application/interface/usecase/wardrobe_uc.go`), handler `ConfirmWardrobeItemReview` trong `.../wardrobe/presentation/handler/wardrobe_item_handler.go`, route `POST /wardrobe-items/:id/confirm-review` trong `.../wardrobe/presentation/router/router.go`
- [X] T016 [P] [US1] Unit test UC confirm-review (ownership sai, status sai, thiÃƒÂ¡Ã‚ÂºÃ‚Â¿u category, thÃƒÆ’Ã‚Â nh cÃƒÆ’Ã‚Â´ng) trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/*_test.go`; Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“ng thÃƒÂ¡Ã‚Â»Ã‚Âi xÃƒÆ’Ã‚Â¡c minh FR-003/SC-001 (mÃƒÆ’Ã‚Â³n sau xÃƒÆ’Ã‚Â¡c nhÃƒÂ¡Ã‚ÂºÃ‚Â­n xuÃƒÂ¡Ã‚ÂºÃ‚Â¥t hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n trong gÃƒÂ¡Ã‚Â»Ã‚Â£i ÃƒÆ’Ã‚Â½ phÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“) qua quickstart G1
- [X] T017 [US1] Regenerate Swagger cho endpoint mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi: `make swagger svc=commerce` (wardrobe/brand Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Âu thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c service commerce)

**Checkpoint**: US1 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p hoÃƒÂ¡Ã‚ÂºÃ‚Â¡t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng vÃƒÆ’Ã‚Â  kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c.

---

## Phase 4: User Story 2 - Theo dÃƒÆ’Ã‚Âµi tiÃƒÂ¡Ã‚ÂºÃ‚Â¿n trÃƒÆ’Ã‚Â¬nh realtime Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â§y Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Â§, khÃƒÆ’Ã‚Â´ng mÃƒÂ¡Ã‚ÂºÃ‚Â¥t sÃƒÂ¡Ã‚Â»Ã‚Â± kiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n (Priority: P1)

**Goal**: PhÃƒÆ’Ã‚Â¡t SSE `processing` cho tÃƒÂ¡Ã‚Â»Ã‚Â«ng mÃƒÆ’Ã‚Â³n + phÃƒÆ’Ã‚Â¡t lÃƒÂ¡Ã‚ÂºÃ‚Â¡i khi reconnect.

**Independent Test**: Batch upload ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ SSE nhÃƒÂ¡Ã‚ÂºÃ‚Â­n `processing` tÃƒÂ¡Ã‚Â»Ã‚Â«ng mÃƒÆ’Ã‚Â³n, rÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“i `completed/needs_review/failed`; reconnect giÃƒÂ¡Ã‚Â»Ã‚Â¯a chÃƒÂ¡Ã‚Â»Ã‚Â«ng nhÃƒÂ¡Ã‚ÂºÃ‚Â­n lÃƒÂ¡Ã‚ÂºÃ‚Â¡i event.

- [X] T018 [P] [US2] ThÃƒÆ’Ã‚Âªm `Processing AnalyzeTaskStatus = "processing"` vÃƒÆ’Ã‚Â o `shared/application/constants/analyzetaskstatus/analyzetaskstatus.go`
- [X] T019 [P] [US2] ThÃƒÆ’Ã‚Âªm topic `TopicFashionProcessingWardrobe`/`TopicFashionProcessingBrand` vÃƒÆ’Ã‚Â o `shared/application/constants/eventconstants/eventconstants.go` vÃƒÆ’Ã‚Â  routing key alias trong `shared/infrastructure/messaging/rabbitmq_constants.go`
- [X] T020 [US2] Bind thÃƒÆ’Ã‚Âªm routing key processing vÃƒÆ’Ã‚Â o 2 binding analyzed hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n cÃƒÆ’Ã‚Â³ trong `shared/infrastructure/messaging/topology.go` (khÃƒÆ’Ã‚Â´ng tÃƒÂ¡Ã‚ÂºÃ‚Â¡o queue/DLQ mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi)
- [X] T021 [US2] Publish event `fashion.event.processing.*` ÃƒÂ¡Ã‚Â»Ã…Â¸ Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â§u `ProcessBackgroundBatchUploadJob` trong `services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go` (trong `uow.Execute`, switch `job.ItemType`)
- [X] T022 [US2] ThÃƒÆ’Ã‚Âªm `HandleFashionProcessing` + dispatch nhÃƒÆ’Ã‚Â¡nh `Processing` trong `HandleFashionEvent` tÃƒÂ¡Ã‚ÂºÃ‚Â¡i `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/fashion_event_uc.go` + interface `.../interface/usecase/wardrobe_uc.go`; Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â£m bÃƒÂ¡Ã‚ÂºÃ‚Â£o idempotent (khÃƒÆ’Ã‚Â´ng Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢i DB nÃƒÂ¡Ã‚ÂºÃ‚Â¿u item khÃƒÆ’Ã‚Â´ng cÃƒÆ’Ã‚Â²n `Processing`; vÃƒÂ¡Ã‚ÂºÃ‚Â«n phÃƒÆ’Ã‚Â¡t SSE nÃƒÂ¡Ã‚ÂºÃ‚Â¿u kÃƒÆ’Ã‚Âªnh mÃƒÂ¡Ã‚Â»Ã…Â¸) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â FR-013
- [X] T023 [US2] ThÃƒÆ’Ã‚Âªm `HandleFashionProcessing` + dispatch nhÃƒÆ’Ã‚Â¡nh `Processing` cho brand tÃƒÂ¡Ã‚ÂºÃ‚Â¡i `services/commerce/internal/modules/brand/application/usecase/item/fashion_event_uc.go` + interface `.../interface/usecase/brand_item_uc.go`; Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â£m bÃƒÂ¡Ã‚ÂºÃ‚Â£o idempotent (khÃƒÆ’Ã‚Â´ng Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢i DB nÃƒÂ¡Ã‚ÂºÃ‚Â¿u item khÃƒÆ’Ã‚Â´ng cÃƒÆ’Ã‚Â²n trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i Ãƒâ€žÃ¢â‚¬Ëœang xÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½; vÃƒÂ¡Ã‚ÂºÃ‚Â«n phÃƒÆ’Ã‚Â¡t SSE nÃƒÂ¡Ã‚ÂºÃ‚Â¿u kÃƒÆ’Ã‚Âªnh mÃƒÂ¡Ã‚Â»Ã…Â¸) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â FR-013
- [X] T024 [P] [US2] Unit test: SSE manager buffer/replay (gÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“m loÃƒÂ¡Ã‚ÂºÃ‚Â¡i theo size + TTL, khÃƒÆ’Ã‚Â´ng xÃƒÆ’Ã‚Â³a khi hÃƒÂ¡Ã‚ÂºÃ‚Â¿t subscriber) trong `shared/application/sse/*_test.go`; dispatch `processing` ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ payload SSE Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Âºng (wardrobe + brand) trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/*_test.go` + `services/commerce/internal/modules/brand/application/usecase/item/*_test.go`; idempotency khi nhÃƒÂ¡Ã‚ÂºÃ‚Â­n event `processing` trÃƒÆ’Ã‚Â¹ng (khÃƒÆ’Ã‚Â´ng Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢i DB dÃƒâ€ Ã‚Â° thÃƒÂ¡Ã‚Â»Ã‚Â«a) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â FR-013; vÃƒÆ’Ã‚Â  xÃƒÆ’Ã‚Â¡c minh payload sÃƒÂ¡Ã‚Â»Ã‚Â± kiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n `completed`/`failed`/`needs_review` Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â§y Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Â§ (Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹nh danh mÃƒÆ’Ã‚Â³n, chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° sÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœ tiÃƒÂ¡Ã‚ÂºÃ‚Â¿n trÃƒÆ’Ã‚Â¬nh, dÃƒÂ¡Ã‚Â»Ã‚Â¯ liÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡u tÃƒÆ’Ã‚Â³m tÃƒÂ¡Ã‚ÂºÃ‚Â¯t, thÃƒÆ’Ã‚Â´ng bÃƒÆ’Ã‚Â¡o lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â FR-005
- [X] T025 [US2] ThÃƒÆ’Ã‚Âªm annotation Swagger mÃƒÆ’Ã‚Â´ tÃƒÂ¡Ã‚ÂºÃ‚Â£ trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i `processing` trong handler wardrobe (`services/commerce/internal/modules/wardrobe/presentation/handler/*.go`) + brand (`services/commerce/internal/modules/brand/presentation/handler/*.go`) vÃƒÆ’Ã‚Â  regenerate `make swagger svc=commerce`; KHÃƒÆ’Ã¢â‚¬ÂNG chÃƒÂ¡Ã‚ÂºÃ‚Â¡y `swagger-combine` (dÃƒÆ’Ã‚Â nh cho T059)

**Checkpoint**: US2 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p hoÃƒÂ¡Ã‚ÂºÃ‚Â¡t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng vÃƒÆ’Ã‚Â  kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c.

---

## Phase 5: User Story 3 - ThÃƒÂ¡Ã‚Â»Ã‚Â­ phÃƒÆ’Ã‚Â¢n tÃƒÆ’Ã‚Â­ch lÃƒÂ¡Ã‚ÂºÃ‚Â¡i an toÃƒÆ’Ã‚Â n & khÃƒÆ’Ã‚Â´ng mÃƒÂ¡Ã‚ÂºÃ‚Â¥t chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â°nh sÃƒÂ¡Ã‚Â»Ã‚Â­a thÃƒÂ¡Ã‚Â»Ã‚Â§ cÃƒÆ’Ã‚Â´ng (Priority: P2)

**Goal**: Retry chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° khi trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡, trÃƒÂ¡Ã‚ÂºÃ‚Â£ `taskId`, bÃƒÂ¡Ã‚ÂºÃ‚Â£o toÃƒÆ’Ã‚Â n danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n cÃƒÆ’Ã‚Â³.

**Independent Test**: Retry khi Ãƒâ€žÃ¢â‚¬Ëœang xÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½/Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â£ dÃƒÆ’Ã‚Â¹ng Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ tÃƒÂ¡Ã‚Â»Ã‚Â« chÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi; sÃƒÂ¡Ã‚Â»Ã‚Â­a danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c rÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“i retry ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c giÃƒÂ¡Ã‚Â»Ã‚Â¯ nguyÃƒÆ’Ã‚Âªn; response cÃƒÆ’Ã‚Â³ `taskId`.

- [X] T026 [US3] ThÃƒÆ’Ã‚Âªm `PreserveCategory bool` vÃƒÆ’Ã‚Â o `FashionAnalyzeJobDTO` trong `services/commerce/internal/modules/fashion/application/dto/fashion.go`
- [X] T027 [US3] ThÃƒÆ’Ã‚Âªm field `TaskID *string` (`json:"taskId,omitempty"`) vÃƒÆ’Ã‚Â o `WardrobeItemRes` (`services/commerce/internal/modules/wardrobe/application/dto/item.go`) vÃƒÆ’Ã‚Â  populate tÃƒÂ¡Ã‚ÂºÃ‚Â¡i `services/commerce/internal/modules/wardrobe/application/mapper/wardrobe.go` (mapper DTO phÃƒÂ¡Ã‚ÂºÃ‚Â£i nÃƒÂ¡Ã‚ÂºÃ‚Â±m trong `application/mapper`) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â tÃƒÂ¡Ã‚ÂºÃ‚Â¡o field TRÃƒâ€ Ã‚Â¯ÃƒÂ¡Ã‚Â»Ã…Â¡C Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ UC dÃƒÆ’Ã‚Â¹ng Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c
- [X] T028 [US3] BÃƒÂ¡Ã‚Â»Ã‚Â ghi Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â¨ `category_id` khi `PreserveCategory` && Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â£ cÃƒÆ’Ã‚Â³ category trong `buildCompletedItemUpdate` (`services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go`)
- [X] T029 [US3] CÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t `RetryWardrobeAnalysis` trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go` (phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c T026, T027): giÃƒÂ¡Ã‚Â»Ã‚Â¯ guard trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i + claim nguyÃƒÆ’Ã‚Âªn tÃƒÂ¡Ã‚Â»Ã‚Â­, sinh `taskId` mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi, set `PreserveCategory` khi fashion item Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â£ cÃƒÆ’Ã‚Â³ category, trÃƒÂ¡Ã‚ÂºÃ‚Â£ `taskId` trong response
- [X] T030 [P] [US3] Unit test retry: guard trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i (Ãƒâ€žÃ¢â‚¬Ëœang xÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½/khÃƒÆ’Ã‚Â´ng hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡), preserve category, trÃƒÂ¡Ã‚ÂºÃ‚Â£ taskId trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/*_test.go`
- [X] T031 [US3] Regenerate Swagger cho `retry-analysis`: `make swagger svc=commerce`

**Checkpoint**: US3 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p hoÃƒÂ¡Ã‚ÂºÃ‚Â¡t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng vÃƒÆ’Ã‚Â  kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c.

---

## Phase 6: User Story 4 - NhÃƒÆ’Ã‚Â¢n viÃƒÆ’Ã‚Âªn nhÃƒÆ’Ã‚Â£n hÃƒÆ’Ã‚Â ng xÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½ sÃƒÂ¡Ã‚ÂºÃ‚Â£n phÃƒÂ¡Ã‚ÂºÃ‚Â©m lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i/cÃƒÂ¡Ã‚ÂºÃ‚Â§n rÃƒÆ’Ã‚Â  soÃƒÆ’Ã‚Â¡t (Priority: P2)

**Goal**: Brand phÃƒÆ’Ã‚Â¢n biÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡t `failed` vs `draft`, hiÃƒÂ¡Ã‚Â»Ã†â€™n thÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ lÃƒÆ’Ã‚Â½ do, chÃƒÂ¡Ã‚ÂºÃ‚Â·n Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢i khÃƒÂ¡Ã‚Â»Ã‚Âi `failed`, cÃƒÆ’Ã‚Â³ retry-analysis.

**Independent Test**: Brand item lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ `status=failed` + reason; Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢i trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i thÃƒÂ¡Ã‚Â»Ã‚Â§ cÃƒÆ’Ã‚Â´ng bÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ tÃƒÂ¡Ã‚Â»Ã‚Â« chÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi; retry hoÃƒÂ¡Ã‚ÂºÃ‚Â¡t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng vÃƒÆ’Ã‚Â  trÃƒÂ¡Ã‚ÂºÃ‚Â£ `taskId`.

- [X] T032 [P] [US4] ThÃƒÆ’Ã‚Âªm `Failed BrandItemStatus = "failed"` vÃƒÆ’Ã‚Â o `services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus/brand_item_status.go`
- [X] T033 [US4] Map `HandleFashionFailed` ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ `Failed` (thay vÃƒÆ’Ã‚Â¬ `Draft`) trong `services/commerce/internal/modules/brand/application/usecase/item/fashion_event_uc.go`
- [X] T034 [P] [US4] MÃƒÂ¡Ã‚Â»Ã…Â¸ rÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng `FashionItemContractDTO` (`services/commerce/internal/modules/fashion/application/dto/contract_dto.go`) thÃƒÆ’Ã‚Âªm `ProcessingErrorReason`, `ReviewReason`; Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â·t hÃƒÆ’Ã‚Â m map entityÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢DTO trong `services/commerce/internal/modules/fashion/application/mapper/` (tuÃƒÆ’Ã‚Â¢n thÃƒÂ¡Ã‚Â»Ã‚Â§ Code Style: mapper trong `application/mapper`) vÃƒÆ’Ã‚Â  gÃƒÂ¡Ã‚Â»Ã‚Âi tÃƒÂ¡Ã‚Â»Ã‚Â« `fashion_item_uc.go` (refactor `mapToFashionItemRes` nÃƒÂ¡Ã‚ÂºÃ‚Â¿u cÃƒÂ¡Ã‚ÂºÃ‚Â§n)
- [X] T035 [P] [US4] ThÃƒÆ’Ã‚Âªm `ProcessingErrorReason`, `ReviewReason` vÃƒÆ’Ã‚Â o `BrandItemRes`/`BrandItemBriefRes` (`services/commerce/internal/modules/brand/application/dto/brand_item.go`) + map tÃƒÂ¡Ã‚ÂºÃ‚Â¡i `services/commerce/internal/modules/brand/application/mapper/brand_item_mapper.go`
- [X] T036 [US4] Guard `UpdateBrandItemStatus` (tÃƒÂ¡Ã‚Â»Ã‚Â« chÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â·t thÃƒÂ¡Ã‚Â»Ã‚Â§ cÃƒÆ’Ã‚Â´ng `failed`; tÃƒÂ¡Ã‚Â»Ã‚Â« chÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi mÃƒÂ¡Ã‚Â»Ã‚Âi thay Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢i khi item Ãƒâ€žÃ¢â‚¬Ëœang `failed`) + lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i `ErrBrandItemStatusLocked` trong `services/commerce/internal/modules/brand/application/errors/errors.go`; sÃƒÂ¡Ã‚Â»Ã‚Â­a tÃƒÂ¡Ã‚ÂºÃ‚Â¡i `services/commerce/internal/modules/brand/application/usecase/item/crud_uc.go` (LÃƒâ€ Ã‚Â¯U ÃƒÆ’Ã‚Â: cÃƒÆ’Ã‚Â¹ng file vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi T037 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ chÃƒÂ¡Ã‚ÂºÃ‚Â¡y tuÃƒÂ¡Ã‚ÂºÃ‚Â§n tÃƒÂ¡Ã‚Â»Ã‚Â±, khÃƒÆ’Ã‚Â´ng song song)
- [X] T037 [US4] Validate status + guard `failed` trong `UpdateBrandItem` (`services/commerce/internal/modules/brand/application/usecase/item/crud_uc.go`) (cÃƒÆ’Ã‚Â¹ng file vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi T036 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ chÃƒÂ¡Ã‚ÂºÃ‚Â¡y sau T036)
- [X] T038 [US4] Clear `review_reason`/`processing_error_reason` khi thÃƒÆ’Ã‚Â nh cÃƒÆ’Ã‚Â´ng trong `completeProcessedItem` (`services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go`)
- [X] T039 [US4] Implement `RetryBrandItemAnalysis(ctx, staffUserID, brandID, itemID) (*dto.BrandItemRes, error)` trong `services/commerce/internal/modules/brand/application/usecase/item/crud_uc.go` (hoÃƒÂ¡Ã‚ÂºÃ‚Â·c file `retry_uc.go` mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi cÃƒÆ’Ã‚Â¹ng thÃƒâ€ Ã‚Â° mÃƒÂ¡Ã‚Â»Ã‚Â¥c) + khai bÃƒÆ’Ã‚Â¡o interface usecase; thÃƒÆ’Ã‚Âªm method repo `ClaimManualAnalysisRetry(ctx, itemID, now) (*entities.BrandItem, bool, error)` vÃƒÆ’Ã‚Â o `brand/domain/repositories/brand_item_interfaces.go` + impl trong `brand/infrastructure/persistence/brand_item_repo.go`; thÃƒÆ’Ã‚Âªm error `ErrBrandRetryForbidden`, `ErrBrandRetryInProgress` trong `brand/application/errors/errors.go`; Ãƒâ€žÃ¢â‚¬ËœiÃƒÂ¡Ã‚Â»Ã‚Âu kiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n: `failed`, hoÃƒÂ¡Ã‚ÂºÃ‚Â·c `draft` + `reviewReason != nil`; sinh `taskId` mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi, gÃƒÂ¡Ã‚Â»Ã‚Âi `fashionContract.ResetProcessingState`, publish job analyze trong `uow.Execute`; trÃƒÂ¡Ã‚ÂºÃ‚Â£ `BrandItemRes` kÃƒÆ’Ã‚Â¨m `taskId`
- [X] T040 [US4] ThÃƒÆ’Ã‚Âªm handler + route `POST /brand-portal/brands/:brandId/items/:itemId/retry-analysis` trong `services/commerce/internal/modules/brand/presentation/handler/brand_portal_item_handler.go` + `.../presentation/router/router.go`
- [X] T041 [P] [US4] Unit test: brand failed mapping, guard status, Ãƒâ€žÃ¢â‚¬ËœiÃƒÂ¡Ã‚Â»Ã‚Âu kiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n retry (`failed` / `draft`+reason / draft thÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Âng bÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ tÃƒÂ¡Ã‚Â»Ã‚Â« chÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi) trong `services/commerce/internal/modules/brand/application/usecase/item/*_test.go`
- [X] T042 [US4] ThÃƒÆ’Ã‚Âªm annotation Swagger cho endpoint retry brand (trong `brand_portal_item_handler.go`) vÃƒÆ’Ã‚Â  regenerate `make swagger svc=commerce` (brand portal thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c service commerce; `swagger-combine` Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ ÃƒÂ¡Ã‚Â»Ã…Â¸ T059)

**Checkpoint**: US4 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p hoÃƒÂ¡Ã‚ÂºÃ‚Â¡t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng vÃƒÆ’Ã‚Â  kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c.

---

## Phase 7: User Story 5 - LÃƒÂ¡Ã‚Â»Ã‚Âc theo danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c phÃƒÂ¡Ã‚ÂºÃ‚Â£i tÃƒÆ’Ã‚Â´n trÃƒÂ¡Ã‚Â»Ã‚Âng kÃƒÂ¡Ã‚ÂºÃ‚Â¿t quÃƒÂ¡Ã‚ÂºÃ‚Â£ rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng (Priority: P2)

**Goal**: PhÃƒÆ’Ã‚Â¢n biÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡t "khÃƒÆ’Ã‚Â´ng lÃƒÂ¡Ã‚Â»Ã‚Âc" vs "lÃƒÂ¡Ã‚Â»Ã‚Âc nhÃƒâ€ Ã‚Â°ng rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng"; danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c/tÃƒÂ¡Ã‚Â»Ã‚Â« khÃƒÆ’Ã‚Â³a khÃƒÆ’Ã‚Â´ng khÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºp ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ 0 kÃƒÂ¡Ã‚ÂºÃ‚Â¿t quÃƒÂ¡Ã‚ÂºÃ‚Â£; khÃƒÆ’Ã‚Â´ng lÃƒÂ¡Ã‚Â»Ã‚Âc ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ toÃƒÆ’Ã‚Â n bÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ mÃƒÆ’Ã‚Â³n (kÃƒÂ¡Ã‚Â»Ã†â€™ cÃƒÂ¡Ã‚ÂºÃ‚Â£ `fashion_item_id` NULL).

**Independent Test**: (G8 trong quickstart) danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ `total=0`; khÃƒÆ’Ã‚Â´ng truyÃƒÂ¡Ã‚Â»Ã‚Ân danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ tÃƒÂ¡Ã‚ÂºÃ‚Â¥t cÃƒÂ¡Ã‚ÂºÃ‚Â£ mÃƒÆ’Ã‚Â³n; tÃƒÂ¡Ã‚Â»Ã‚Â« khÃƒÆ’Ã‚Â³a khÃƒÆ’Ã‚Â´ng khÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºp ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ 0.

- [X] T043 [US5] ThÃƒÆ’Ã‚Âªm tham sÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœ `filterByFashionItems bool` vÃƒÆ’Ã‚Â o interface `WardrobeItemRepository` (`services/commerce/internal/modules/wardrobe/domain/repositories/interfaces.go`) cho `CountByUserIDAndFilters`, `GetByUserIDAndFiltersPaginated`, `CountItems`, `GetItemsPaginated`, `GetItems` (giÃƒÂ¡Ã‚Â»Ã‚Â¯ nguyÃƒÆ’Ã‚Âªn chÃƒÂ¡Ã‚Â»Ã‚Â¯ kÃƒÆ’Ã‚Â½ `GetByUserID`) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â¢y lÃƒÆ’Ã‚Â  thay Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢i phÃƒÆ’Ã‚Â¡ vÃƒÂ¡Ã‚Â»Ã‚Â¡ build nÃƒÆ’Ã‚Âªn MUST cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“ng thÃƒÂ¡Ã‚Â»Ã‚Âi impl + mÃƒÂ¡Ã‚Â»Ã‚Âi caller + mock (T044, T045, T046) trong cÃƒÆ’Ã‚Â¹ng mÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢t bÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc commit
- [X] T044 [US5] CÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t impl trong `services/commerce/internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`: `buildUserWardrobeFilterQuery` + `buildWardrobeItemSearchQuery` chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° ÃƒÆ’Ã‚Â¡p `fashion_item_id IN ?` khi `filterByFashionItems == true` (kÃƒÂ¡Ã‚Â»Ã†â€™ cÃƒÂ¡Ã‚ÂºÃ‚Â£ slice rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ `IN ()`); khi `false` bÃƒÂ¡Ã‚Â»Ã‚Â hoÃƒÆ’Ã‚Â n toÃƒÆ’Ã‚Â n bÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ lÃƒÂ¡Ã‚Â»Ã‚Âc (trÃƒÂ¡Ã‚ÂºÃ‚Â£ cÃƒÂ¡Ã‚ÂºÃ‚Â£ bÃƒÂ¡Ã‚ÂºÃ‚Â£n ghi `fashion_item_id` NULL). Ãƒâ€žÃ‚ÂÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“ng thÃƒÂ¡Ã‚Â»Ã‚Âi cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t mÃƒÂ¡Ã‚Â»Ã‚Âi caller bÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ ÃƒÂ¡Ã‚ÂºÃ‚Â£nh hÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã…Â¸ng bÃƒÂ¡Ã‚Â»Ã…Â¸i chÃƒÂ¡Ã‚Â»Ã‚Â¯ kÃƒÆ’Ã‚Â½ mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi: `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go` (`CountItemsByFashionIDs` ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ truyÃƒÂ¡Ã‚Â»Ã‚Ân `filterByFashionItems=true`; fashion rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ 0), `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/search_sync/search_sync_uc.go` (`GetItems(ctx, nil, ...)` ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ truyÃƒÂ¡Ã‚Â»Ã‚Ân `false`), vÃƒÆ’Ã‚Â  mock test `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_search_fallback_test.go` (cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t override `CountItems` + `GetItemsPaginated`)
- [X] T045 [US5] SÃƒÂ¡Ã‚Â»Ã‚Â­a `GetWardrobeItems` trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_read.go`: `filterByFashionItems = categorySlug != nil`; short-circuit khi cÃƒÂ¡Ã‚Â»Ã‚Â bÃƒÂ¡Ã‚ÂºÃ‚Â­t mÃƒÆ’Ã‚Â  `filterFashionIDs` rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng (trÃƒÂ¡Ã‚ÂºÃ‚Â£ `PaginationResult` rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng, KHÃƒÆ’Ã¢â‚¬ÂNG gÃƒÂ¡Ã‚Â»Ã‚Âi repo); truyÃƒÂ¡Ã‚Â»Ã‚Ân cÃƒÂ¡Ã‚Â»Ã‚Â xuÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœng `CountByUserIDAndFilters` + `GetByUserIDAndFiltersPaginated`
- [X] T046 [US5] SÃƒÂ¡Ã‚Â»Ã‚Â­a fallback DB trong `GetSystemCatalogWardrobeItems` (`.../wardrobe/item/item_uc_read.go`) vÃƒÆ’Ã‚Â  `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/catalog/catalog_uc.go`: `filterByFashionItems = (queryStr != nil || categorySlug != nil)`; short-circuit khi rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng; truyÃƒÂ¡Ã‚Â»Ã‚Ân cÃƒÂ¡Ã‚Â»Ã‚Â xuÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœng `CountItems`/`GetItemsPaginated`
- [X] T047 [P] [US5] Unit test: danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c/tÃƒÂ¡Ã‚Â»Ã‚Â« khÃƒÆ’Ã‚Â³a rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ 0 (khÃƒÆ’Ã‚Â´ng gÃƒÂ¡Ã‚Â»Ã‚Âi repo do short-circuit); khÃƒÆ’Ã‚Â´ng lÃƒÂ¡Ã‚Â»Ã‚Âc ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ toÃƒÆ’Ã‚Â n bÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢; xÃƒÆ’Ã‚Â¡c minh cÃƒÂ¡Ã‚Â»Ã‚Â `filterByFashionItems` truyÃƒÂ¡Ã‚Â»Ã‚Ân Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Âºng xuÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœng repo mock (usecase, `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/{item,catalog}/*_test.go`) vÃƒÆ’Ã‚Â  repo ÃƒÆ’Ã‚Â¡p `IN`/bÃƒÂ¡Ã‚Â»Ã‚Â filter Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Âºng (`services/commerce/internal/modules/wardrobe/infrastructure/persistence/*_test.go`) (SC-011, SC-012, FR-019..FR-024)
- [X] T048 [US5] Ãƒâ€žÃ‚ÂÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi chiÃƒÂ¡Ã‚ÂºÃ‚Â¿u & cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t tÃƒÆ’Ã‚Â i liÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡u hÃƒÆ’Ã‚Â nh vi lÃƒÂ¡Ã‚Â»Ã‚Âc: `docs/api/wardrobe/wardrobe-api.md`; xÃƒÆ’Ã‚Â¡c nhÃƒÂ¡Ã‚ÂºÃ‚Â­n `specs/023-analyze-status-handling/frontend-guide.md` + `contracts/api-contracts.md` Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â£ nÃƒÆ’Ã‚Âªu rÃƒÆ’Ã‚Âµ "trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ sau" cÃƒÂ¡Ã‚Â»Ã‚Â§a API lÃƒÂ¡Ã‚Â»Ã‚Âc danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c (FR-018)

**Checkpoint**: US5 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p hoÃƒÂ¡Ã‚ÂºÃ‚Â¡t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng vÃƒÆ’Ã‚Â  kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c.

---

## Phase 8: Cross-Cutting - Realtime Best-effort & Observability (FR-025, FR-026)

**Goal**: LÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i phÃƒÆ’Ã‚Â¡t/relay sÃƒÂ¡Ã‚Â»Ã‚Â± kiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n KHÃƒÆ’Ã¢â‚¬ÂNG chÃƒÂ¡Ã‚ÂºÃ‚Â·n nghiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡p vÃƒÂ¡Ã‚Â»Ã‚Â¥ (DB vÃƒÂ¡Ã‚ÂºÃ‚Â«n lÃƒÆ’Ã‚Â  nguÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“n sÃƒÂ¡Ã‚Â»Ã‚Â± thÃƒÂ¡Ã‚ÂºÃ‚Â­t); cÃƒÆ’Ã‚Â³ log + metric Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ vÃƒÂ¡Ã‚ÂºÃ‚Â­n hÃƒÆ’Ã‚Â nh phÃƒÆ’Ã‚Â¡t hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n mÃƒÂ¡Ã‚ÂºÃ‚Â¥t sÃƒÂ¡Ã‚Â»Ã‚Â± kiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n.

**Independent Test**: NgÃƒÂ¡Ã‚ÂºÃ‚Â¯t Redis ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ upload/retry/confirm vÃƒÂ¡Ã‚ÂºÃ‚Â«n thÃƒÆ’Ã‚Â nh cÃƒÆ’Ã‚Â´ng; log `warn` kÃƒÆ’Ã‚Â¨m `taskId`; counters trong health `/api/v1/commerce/health` tÃƒâ€žÃ†â€™ng.

> PhÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c Phase 2 (publisher/relay). RiÃƒÆ’Ã‚Âªng T049 (tÃƒÂ¡Ã‚ÂºÃ‚Â¡o file metrics) Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â cÃƒÆ’Ã‚Â³ thÃƒÂ¡Ã‚Â»Ã†â€™ lÃƒÆ’Ã‚Â m ngay; T050ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T052 phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c Phase 2. LÃƒâ€ Ã‚Â°u ÃƒÆ’Ã‚Â½: T050 chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â°nh `redis_realtime_publisher.go`/`sse_relay.go` (tÃƒÂ¡Ã‚ÂºÃ‚Â¡o ÃƒÂ¡Ã‚Â»Ã…Â¸ T005/T006) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â nÃƒÆ’Ã‚Âªn gÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢p nhÃƒÆ’Ã‚Â¡nh best-effort vÃƒÆ’Ã‚Â o T005/T006 ngay khi viÃƒÂ¡Ã‚ÂºÃ‚Â¿t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ trÃƒÆ’Ã‚Â¡nh sÃƒÂ¡Ã‚Â»Ã‚Â­a lÃƒÂ¡Ã‚ÂºÃ‚Â¡i.

- [X] T049 [P] ThÃƒÆ’Ã‚Âªm counters/gauges in-memory + `GetRealtimeMetrics()` trong `shared/application/sse/realtime_metrics.go` theo pattern `shared/infrastructure/messaging/metrics.go` (`realtime_publish_total`, `realtime_publish_failed`, `realtime_relay_total`, `realtime_relay_failed`, gauge `realtime_subscribers`)
- [X] T050 XÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½ best-effort: `shared/infrastructure/messaging/redis_realtime_publisher.go` + `shared/presentation/sse_relay.go` ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â nhÃƒÆ’Ã‚Â¡nh lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i Redis chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° log `warn` kÃƒÆ’Ã‚Â¨m `taskId`, tÃƒâ€žÃ†â€™ng counter `*_failed`, vÃƒÆ’Ã‚Â  KHÃƒÆ’Ã¢â‚¬ÂNG propagate lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i lÃƒÆ’Ã‚Âªn usecase/worker
- [X] T051 CÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t gauge subscriber trong `shared/application/sse/sse_manager.go` tÃƒÂ¡Ã‚ÂºÃ‚Â¡i `Subscribe`/`Unsubscribe` (atomic +1/ÃƒÂ¢Ã‹â€ Ã¢â‚¬â„¢1)
- [X] T052 ThÃƒÆ’Ã‚Âªm khÃƒÆ’Ã‚Â³a `"realtime": sse.GetRealtimeMetrics()` vÃƒÆ’Ã‚Â o payload health `services/commerce/internal/bootstrap/router.go` (cÃƒÂ¡Ã‚ÂºÃ‚Â¡nh `"elasticsearch"`)
- [X] T053 [P] Unit test: publisher/relay trÃƒÂ¡Ã‚ÂºÃ‚Â£ nil + tÃƒâ€žÃ†â€™ng `*_failed` khi Redis lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i trong `shared/infrastructure/messaging/*_test.go` + `shared/presentation/*_test.go`; `GetRealtimeMetrics` phÃƒÂ¡Ã‚ÂºÃ‚Â£n ÃƒÆ’Ã‚Â¡nh subscribe/unsubscribe (`shared/application/sse/*_test.go`) (SC-013, SC-014)

**Checkpoint**: Realtime best-effort + observability hoÃƒÂ¡Ã‚ÂºÃ‚Â¡t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng, khÃƒÆ’Ã‚Â´ng chÃƒÂ¡Ã‚ÂºÃ‚Â·n nghiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡p vÃƒÂ¡Ã‚Â»Ã‚Â¥.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: TÃƒÆ’Ã‚Â i liÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡u, tÃƒÆ’Ã‚Â i liÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡u frontend, kiÃƒÂ¡Ã‚Â»Ã†â€™m chÃƒÂ¡Ã‚Â»Ã‚Â©ng toÃƒÆ’Ã‚Â n cÃƒÂ¡Ã‚Â»Ã‚Â¥c.

- [X] T054 [P] CÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t API docs: `docs/api/wardrobe/constants.md` (thÃƒÆ’Ã‚Âªm `processing`), `docs/api/wardrobe/wardrobe-api.md` (SSE `processing`, replay, confirm-review, retry `taskId`, lÃƒÂ¡Ã‚Â»Ã‚Âc danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng), `docs/api/brand/brand-portal-api.md` + `docs/api/brand/constants.md` (brand `failed` + reason + retry), `docs/api/fashion/constants.md`
- [X] T055 [P] CÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t `docs/system-design/event-driven-design.md` (topic `fashion.event.processing`, sÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœ queue/binding; best-effort realtime qua Redis Pub/Sub + metrics) vÃƒÆ’Ã‚Â  `docs/system-design/core-feature-sequences.md` (luÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“ng realtime + replay)
- [X] T056 CÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t `specs/023-analyze-status-handling/frontend-guide.md` bÃƒÂ¡Ã‚ÂºÃ‚Â±ng nÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢i dung thÃƒÂ¡Ã‚ÂºÃ‚Â­t tÃƒÂ¡Ã‚Â»Ã‚Â« code/contracts: hoÃƒÆ’Ã‚Â n thiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n mÃƒÂ¡Ã‚Â»Ã‚Â¥c 3 (bÃƒÂ¡Ã‚ÂºÃ‚Â£ng API "trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ sau" + vÃƒÆ’Ã‚Â­ dÃƒÂ¡Ã‚Â»Ã‚Â¥ request/response), mÃƒÂ¡Ã‚Â»Ã‚Â¥c 4 (payload SSE mÃƒÂ¡Ã‚ÂºÃ‚Â«u cho tÃƒÂ¡Ã‚Â»Ã‚Â«ng trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i + reconnect/replay/taskId + refetch dÃƒÂ¡Ã‚Â»Ã‚Â± phÃƒÆ’Ã‚Â²ng), mÃƒÂ¡Ã‚Â»Ã‚Â¥c 5 (mÃƒÆ’Ã‚Â£ lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i & thÃƒÆ’Ã‚Â´ng bÃƒÆ’Ã‚Â¡o thÃƒÂ¡Ã‚ÂºÃ‚Â­t), mÃƒÂ¡Ã‚Â»Ã‚Â¥c 6 (checklist), gÃƒÂ¡Ã‚Â»Ã‚Â¡ bÃƒÂ¡Ã‚Â»Ã‚Â mÃƒÂ¡Ã‚Â»Ã‚Âi placeholder "sÃƒÂ¡Ã‚ÂºÃ‚Â½ chÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœt khi triÃƒÂ¡Ã‚Â»Ã†â€™n khai"
- [X] T057 KiÃƒÂ¡Ã‚Â»Ã†â€™m tra & cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t lÃƒÂ¡Ã‚ÂºÃ‚Â§n cuÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi `specs/023-analyze-status-handling/frontend-guide.md` (BÃƒÂ¡Ã‚ÂºÃ‚Â®T BUÃƒÂ¡Ã‚Â»Ã‹Å“C ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â sau khi mÃƒÂ¡Ã‚Â»Ã‚Âi task triÃƒÂ¡Ã‚Â»Ã†â€™n khai xong): Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Âc thuÃƒÂ¡Ã‚ÂºÃ‚Â§n tÃƒÂ¡Ã‚Â»Ã‚Â« gÃƒÆ’Ã‚Â³c nhÃƒÆ’Ã‚Â¬n FE, Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi chiÃƒÂ¡Ã‚ÂºÃ‚Â¿u **tÃƒÂ¡Ã‚Â»Ã‚Â«ng** FR/SC, tÃƒÂ¡Ã‚Â»Ã‚Â«ng user story (US1ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“US5), tÃƒÂ¡Ã‚Â»Ã‚Â«ng contract (`contracts/api-contracts.md`, `event-contracts.md`, `realtime-transport.contract.md`) vÃƒÆ’Ã‚Â  tÃƒÂ¡Ã‚Â»Ã‚Â«ng kÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ch bÃƒÂ¡Ã‚ÂºÃ‚Â£n `quickstart.md`; bÃƒÂ¡Ã‚ÂºÃ‚Â£o Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â£m khÃƒÆ’Ã‚Â´ng bÃƒÂ¡Ã‚Â»Ã‚Â sÃƒÆ’Ã‚Â³t status/endpoint/field/lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i/event/metric nÃƒÆ’Ã‚Â o; diÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¦n Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â¡t rÃƒÆ’Ã‚Âµ rÃƒÆ’Ã‚Â ng, khÃƒÆ’Ã‚Â´ng mÃƒâ€ Ã‚Â¡ hÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“; xÃƒÆ’Ã‚Â¡c nhÃƒÂ¡Ã‚ÂºÃ‚Â­n FE triÃƒÂ¡Ã‚Â»Ã†â€™n khai Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c **chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi tÃƒÆ’Ã‚Â i liÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡u nÃƒÆ’Ã‚Â y, khÃƒÆ’Ã‚Â´ng cÃƒÂ¡Ã‚ÂºÃ‚Â§n Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Âc code backend hay Swagger**; bÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢ sung sÃƒÂ¡Ã‚Â»Ã‚Â­a cÃƒÆ’Ã‚Â¡c thiÃƒÂ¡Ã‚ÂºÃ‚Â¿u sÃƒÆ’Ã‚Â³t phÃƒÆ’Ã‚Â¡t hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n vÃƒÆ’Ã‚Â  ghi changelog
- [X] T058 ChÃƒÂ¡Ã‚ÂºÃ‚Â¡y `make wire && make lint && make test-race` vÃƒÆ’Ã‚Â  kiÃƒÂ¡Ã‚Â»Ã†â€™m chÃƒÂ¡Ã‚Â»Ã‚Â©ng theo `specs/023-analyze-status-handling/quickstart.md` (8 kÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ch bÃƒÂ¡Ã‚ÂºÃ‚Â£n: G1ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“G6, G8, G9 ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â G7 gÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢p trong G2; gÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“m G8 lÃƒÂ¡Ã‚Â»Ã‚Âc danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng vÃƒÆ’Ã‚Â  G9 realtime best-effort), trong Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â³ cÃƒÆ’Ã‚Â³ kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ realtime ÃƒÂ¡Ã‚Â»Ã…Â¸ cÃƒÂ¡Ã‚ÂºÃ‚Â£ 2 topology (`APP_MODE=all` vÃƒÆ’Ã‚Â  tÃƒÆ’Ã‚Â¡ch `http`/`worker`) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â FR-016/SC-009
- [X] T059 [P] RÃƒÆ’Ã‚Â  soÃƒÆ’Ã‚Â¡t khÃƒÆ’Ã‚Â´ng cÃƒÆ’Ã‚Â²n TODO/placeholder; xÃƒÆ’Ã‚Â¡c nhÃƒÂ¡Ã‚ÂºÃ‚Â­n migration chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° cÃƒÆ’Ã‚Â²n `add_brand_item_failed_status`; chÃƒÂ¡Ã‚ÂºÃ‚Â¡y `make swagger-combine` Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t bundle combined (nÃƒÂ¡Ã‚ÂºÃ‚Â¿u baseline yÃƒÆ’Ã‚Âªu cÃƒÂ¡Ã‚ÂºÃ‚Â§u)
- [X] T060 [P] XÃƒÆ’Ã‚Â¡c minh tÃƒâ€ Ã‚Â°Ãƒâ€ Ã‚Â¡ng thÃƒÆ’Ã‚Â­ch ngÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c (FR-015/SC-008): thÃƒÆ’Ã‚Âªm test Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â£m bÃƒÂ¡Ã‚ÂºÃ‚Â£o client cÃƒâ€¦Ã‚Â© khÃƒÆ’Ã‚Â´ng vÃƒÂ¡Ã‚Â»Ã‚Â¡ khi nhÃƒÂ¡Ã‚ÂºÃ‚Â­n trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i `processing` mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi ÃƒÂ¡Ã‚Â»Ã…Â¸ SSE (`shared/application/sse/*_test.go`) vÃƒÆ’Ã‚Â  khi `taskId` lÃƒÆ’Ã‚Â  optional trong `WardrobeItemRes` (`services/commerce/internal/modules/wardrobe/application/mapper/*_test.go`); xÃƒÆ’Ã‚Â¡c nhÃƒÂ¡Ã‚ÂºÃ‚Â­n `failed` cÃƒÂ¡Ã‚Â»Ã‚Â§a brand lÃƒÆ’Ã‚Â  giÃƒÆ’Ã‚Â¡ trÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi additive (khÃƒÆ’Ã‚Â´ng phÃƒÆ’Ã‚Â¡ payload cÃƒâ€¦Ã‚Â©)
- [X] T061 [P] TÃƒÂ¡Ã‚ÂºÃ‚Â¡o `specs/023-analyze-status-handling/playbook.md`: playbook vÃƒÂ¡Ã‚ÂºÃ‚Â­n hÃƒÆ’Ã‚Â nh mÃƒÆ’Ã‚Â´ tÃƒÂ¡Ã‚ÂºÃ‚Â£ kÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ch bÃƒÂ¡Ã‚ÂºÃ‚Â£n xÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½ cho tÃƒÂ¡Ã‚Â»Ã‚Â«ng trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i phÃƒÆ’Ã‚Â¢n tÃƒÆ’Ã‚Â­ch (Ãƒâ€žÃ¢â‚¬Ëœang xÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½, hoÃƒÆ’Ã‚Â n tÃƒÂ¡Ã‚ÂºÃ‚Â¥t, thÃƒÂ¡Ã‚ÂºÃ‚Â¥t bÃƒÂ¡Ã‚ÂºÃ‚Â¡i, cÃƒÂ¡Ã‚ÂºÃ‚Â§n rÃƒÆ’Ã‚Â  soÃƒÆ’Ã‚Â¡t) cho cÃƒÂ¡Ã‚ÂºÃ‚Â£ B2C vÃƒÆ’Ã‚Â  B2B, kÃƒÆ’Ã‚Â¨m hÃƒÆ’Ã‚Â nh vi khi realtime best-effort (Redis giÃƒÆ’Ã‚Â¡n Ãƒâ€žÃ¢â‚¬ËœoÃƒÂ¡Ã‚ÂºÃ‚Â¡n) vÃƒÆ’Ã‚Â  cÃƒÆ’Ã‚Â¡ch dÃƒÆ’Ã‚Â¹ng metric/log Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ phÃƒÆ’Ã‚Â¡t hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n mÃƒÂ¡Ã‚ÂºÃ‚Â¥t sÃƒÂ¡Ã‚Â»Ã‚Â± kiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â FR-014/SC-007

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: KhÃƒÆ’Ã‚Â´ng phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â bÃƒÂ¡Ã‚ÂºÃ‚Â¯t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â§u ngay.
- **Foundational (Phase 2)**: PhÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c Setup; chÃƒÂ¡Ã‚ÂºÃ‚Â·n US2, US4 vÃƒÆ’Ã‚Â  Phase 8 (G9). US1/US5 chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° cÃƒÂ¡Ã‚ÂºÃ‚Â§n hÃƒÂ¡Ã‚ÂºÃ‚Â¡ tÃƒÂ¡Ã‚ÂºÃ‚Â§ng hiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n cÃƒÆ’Ã‚Â³ ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â khÃƒÆ’Ã‚Â´ng cÃƒÂ¡Ã‚ÂºÃ‚Â§n transport.
- **User Stories (Phase 3ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“7)**:
  - US1: sau T001, Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p.
  - US2: sau Phase 2.
  - US3: **Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p vÃƒÂ¡Ã‚Â»Ã‚Â logic** ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â T029 trÃƒÂ¡Ã‚ÂºÃ‚Â£ `taskId` khÃƒÆ’Ã‚Â´ng phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c US2. SSE transport (Phase 2) chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° cÃƒÂ¡Ã‚ÂºÃ‚Â§n Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ client *tiÃƒÆ’Ã‚Âªu thÃƒÂ¡Ã‚Â»Ã‚Â¥* sÃƒÂ¡Ã‚Â»Ã‚Â± kiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n realtime; US3 vÃƒÂ¡Ã‚ÂºÃ‚Â«n kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p qua response `taskId` + trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i item.
  - US4: sau Phase 2 (SSE transport) + T001; retry dÃƒÆ’Ã‚Â¹ng chung hÃƒÂ¡Ã‚ÂºÃ‚Â¡ tÃƒÂ¡Ã‚ÂºÃ‚Â§ng vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi US3 (T029) nhÃƒâ€ Ã‚Â°ng Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p nghiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡p vÃƒÂ¡Ã‚Â»Ã‚Â¥ brand.
  - US5: Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p vÃƒÂ¡Ã‚Â»Ã‚Â logic (khÃƒÆ’Ã‚Â´ng cÃƒÂ¡Ã‚ÂºÃ‚Â§n Phase 2/migration) nhÃƒâ€ Ã‚Â°ng **dÃƒÆ’Ã‚Â¹ng chung `domain/repositories/interfaces.go` + `infrastructure/persistence/wardrobe_repo.go` vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi US1 (T011)** ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ phÃƒÂ¡Ã‚ÂºÃ‚Â£i chÃƒÂ¡Ã‚ÂºÃ‚Â¡y SAU US1 hoÃƒÂ¡Ã‚ÂºÃ‚Â·c khÃƒÆ’Ã‚Â´ng song song vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi US1 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ trÃƒÆ’Ã‚Â¡nh xung Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢t file.
- **Cross-Cutting (Phase 8 ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â G9)**: sau Phase 2 (cÃƒÂ¡Ã‚ÂºÃ‚Â§n publisher/relay tÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“n tÃƒÂ¡Ã‚ÂºÃ‚Â¡i); Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi US1/US3/US4/US5.
- **Polish (Phase 9)**: sau tÃƒÂ¡Ã‚ÂºÃ‚Â¥t cÃƒÂ¡Ã‚ÂºÃ‚Â£ story + Phase 8. **T057 phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c toÃƒÆ’Ã‚Â n bÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ task triÃƒÂ¡Ã‚Â»Ã†â€™n khai (T001ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T053) vÃƒÆ’Ã‚Â  T054ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T056.** T060 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p, cÃƒÆ’Ã‚Â³ thÃƒÂ¡Ã‚Â»Ã†â€™ chÃƒÂ¡Ã‚ÂºÃ‚Â¡y song song sau khi US1ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“US5 xong.

### User Story Dependencies

- **US1 (P1)**: Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p (chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° cÃƒÂ¡Ã‚ÂºÃ‚Â§n migration nÃƒÂ¡Ã‚ÂºÃ‚Â¿u Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Â¥ng brand ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â khÃƒÆ’Ã‚Â´ng).
- **US2 (P1)**: Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p sau Foundational.
- **US3 (P2)**: Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p vÃƒÂ¡Ã‚Â»Ã‚Â logic (khÃƒÆ’Ã‚Â´ng phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c US2). SSE transport chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° cÃƒÂ¡Ã‚ÂºÃ‚Â§n cho trÃƒÂ¡Ã‚ÂºÃ‚Â£i nghiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡m realtime phÃƒÆ’Ã‚Â­a client.
- **US4 (P2)**: Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p sau Foundational + migration T001.
- **US5 (P2)**: Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p vÃƒÂ¡Ã‚Â»Ã‚Â logic (khÃƒÆ’Ã‚Â´ng cÃƒÂ¡Ã‚ÂºÃ‚Â§n Phase 2/migration); nhÃƒâ€ Ã‚Â°ng cÃƒÆ’Ã‚Â¹ng file repo/interface vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi **US1** ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ trÃƒÆ’Ã‚Â¡nh chÃƒÂ¡Ã‚ÂºÃ‚Â¡y song song vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi US1.
- **Phase 8 (G9)**: phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c Phase 2; khÃƒÆ’Ã‚Â´ng phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c story nÃƒÆ’Ã‚Â o khÃƒÆ’Ã‚Â¡c.

### Within Each User Story

- Models/constants ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ services ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ handlers/router ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ tests ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ swagger.
- Story hoÃƒÆ’Ã‚Â n tÃƒÂ¡Ã‚ÂºÃ‚Â¥t vÃƒÆ’Ã‚Â  kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc khi sang story kÃƒÂ¡Ã‚ÂºÃ‚Â¿.

### Parallel Opportunities

- Setup: T001, T002, T003 song song.
- Foundational: T004, T005, T006 song song; T007 phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c T002; T008 sau T004ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T007; T009 sau T008; T010 sau T008.
- US1: T012, T014, T016 song song vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi T011/T013/T015.
- US2: T018, T019 song song.
- US4: T032, T034, T035 song song.
- US5: T043 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ [T044 ÃƒÂ¢Ã‹â€ Ã‚Â¥ T045] ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T046 (T045 vÃƒÆ’Ã‚Â  T046 cÃƒÆ’Ã‚Â¹ng sÃƒÂ¡Ã‚Â»Ã‚Â­a `item_uc_read.go` ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ chÃƒÂ¡Ã‚ÂºÃ‚Â¡y tuÃƒÂ¡Ã‚ÂºÃ‚Â§n tÃƒÂ¡Ã‚Â»Ã‚Â±); T047, T048 song song sau T044ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T046.
- Phase 8 (G9): T049, T053 song song; T050 sau T049; T051, T052 sau T049/T050.
- Polish: T054, T055, T059, T060, T061 song song.

---

## Parallel Example: User Story 1

```bash
# Song song (khÃƒÆ’Ã‚Â¡c file, khÃƒÆ’Ã‚Â´ng phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c):
Task: "T012 [P] [US1] ThÃƒÆ’Ã‚Âªm IsWearableCategory vÃƒÆ’Ã‚Â o IFashionContract + impl"
Task: "T014 [P] [US1] ThÃƒÆ’Ã‚Âªm ErrConfirmReviewForbidden/ErrConfirmReviewCategoryRequired"
Task: "T016 [P] [US1] Unit test confirm-review"

# TuÃƒÂ¡Ã‚ÂºÃ‚Â§n tÃƒÂ¡Ã‚Â»Ã‚Â± sau Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â³:
Task: "T011 [US1] Repo ConfirmWardrobeReview"
Task: "T013 [US1] UC ConfirmWardrobeItemReview (depends T011, T012)"
Task: "T015 [US1] Handler + route (depends T013, T014)"
```

---

## Parallel Example: User Story 5

```bash
# Song song sau khi T043ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T046 xong (khÃƒÆ’Ã‚Â¡c file, khÃƒÆ’Ã‚Â´ng phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c):
Task: "T047 [P] [US5] Unit test lÃƒÂ¡Ã‚Â»Ã‚Âc rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng (ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢0) / khÃƒÆ’Ã‚Â´ng lÃƒÂ¡Ã‚Â»Ã‚Âc (ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢toÃƒÆ’Ã‚Â n bÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢)"
Task: "T048 [US5] CÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t docs hÃƒÆ’Ã‚Â nh vi lÃƒÂ¡Ã‚Â»Ã‚Âc (wardrobe-api + xÃƒÆ’Ã‚Â¡c nhÃƒÂ¡Ã‚ÂºÃ‚Â­n frontend-guide)"

# TuÃƒÂ¡Ã‚ÂºÃ‚Â§n tÃƒÂ¡Ã‚Â»Ã‚Â± / song song:
Task: "T043 [US5] Interface + repo signature (filterByFashionItems)"
Task: "T044 [US5] wardrobe_repo.go impl + cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t caller contractuc/search_sync + mock"   # ÃƒÂ¢Ã‹â€ Ã‚Â¥ T045 (khÃƒÆ’Ã‚Â¡c file)
Task: "T045 [US5] GetWardrobeItems short-circuit + truyÃƒÂ¡Ã‚Â»Ã‚Ân cÃƒÂ¡Ã‚Â»Ã‚Â (item_uc_read.go)"
Task: "T046 [US5] Catalog fallback short-circuit + truyÃƒÂ¡Ã‚Â»Ã‚Ân cÃƒÂ¡Ã‚Â»Ã‚Â (item_uc_read.go + catalog_uc.go)"  # sau T045 (cÃƒÆ’Ã‚Â¹ng file)
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1: Setup (T001ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T003).
2. Phase 3: US1 (T011ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T017) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â cÃƒÆ’Ã‚Â³ thÃƒÂ¡Ã‚Â»Ã†â€™ lÃƒÆ’Ã‚Â m ngay sau T001.
3. **STOP & VALIDATE**: chÃƒÂ¡Ã‚ÂºÃ‚Â¡y kÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ch bÃƒÂ¡Ã‚ÂºÃ‚Â£n G1 trong quickstart.
4. Demo nÃƒÂ¡Ã‚ÂºÃ‚Â¿u sÃƒÂ¡Ã‚ÂºÃ‚Âµn sÃƒÆ’Ã‚Â ng.

### Incremental Delivery

1. Setup + Foundational ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ transport sÃƒÂ¡Ã‚ÂºÃ‚Âµn sÃƒÆ’Ã‚Â ng.
2. US1 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ validate ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ MVP.
3. US2 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ validate ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ realtime Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â§y Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Â§.
4. US3 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ validate ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ retry an toÃƒÆ’Ã‚Â n.
5. US4 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ validate ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ parity brand.
6. US5 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ validate ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ lÃƒÂ¡Ã‚Â»Ã‚Âc danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Âºng (rÃƒÂ¡Ã‚Â»Ã¢â‚¬â€ng ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ 0 kÃƒÂ¡Ã‚ÂºÃ‚Â¿t quÃƒÂ¡Ã‚ÂºÃ‚Â£).
7. Phase 8 (G9) ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ validate ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ realtime best-effort + observability.
8. Polish ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t docs, **hoÃƒÆ’Ã‚Â n thiÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡n frontend-guide (T056ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T057)**, kiÃƒÂ¡Ã‚Â»Ã†â€™m chÃƒÂ¡Ã‚Â»Ã‚Â©ng (T058).

### Parallel Team Strategy

- Dev A: US1 (Ãƒâ€ Ã‚Â°u tiÃƒÆ’Ã‚Âªn hoÃƒÆ’Ã‚Â n tÃƒÂ¡Ã‚ÂºÃ‚Â¥t sÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºm ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â US5 dÃƒÆ’Ã‚Â¹ng chung `interfaces.go`/`wardrobe_repo.go`).
- Sau Foundational: US2, US3, US4 **KHÃƒÆ’Ã¢â‚¬ÂNG chÃƒÂ¡Ã‚ÂºÃ‚Â¡y song song tuyÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡t Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi** vÃƒÆ’Ã‚Â¬ cÃƒÆ’Ã‚Â¹ng sÃƒÂ¡Ã‚Â»Ã‚Â­a `fashion/application/usecase/worker/analyze_uc.go` (T021/T028/T038) vÃƒÆ’Ã‚Â  `brand/application/usecase/item/fashion_event_uc.go` (T023/T033). Ãƒâ€žÃ‚ÂiÃƒÂ¡Ã‚Â»Ã‚Âu phÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi: US2 trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc (processing + handler), rÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“i US3/US4; hoÃƒÂ¡Ã‚ÂºÃ‚Â·c phÃƒÆ’Ã‚Â¢n dev theo file.
- US5 chÃƒÂ¡Ã‚ÂºÃ‚Â¡y sau US1 (dÃƒÆ’Ã‚Â¹ng chung file repo/interface).
- Phase 8 (G9) sau Phase 2; T049 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p cÃƒÆ’Ã‚Â³ thÃƒÂ¡Ã‚Â»Ã†â€™ giao cho dev Ãƒâ€žÃ¢â‚¬Ëœang rÃƒÂ¡Ã‚ÂºÃ‚Â£nh.

---

## Notes

- `[P]` = khÃƒÆ’Ã‚Â¡c file, khÃƒÆ’Ã‚Â´ng phÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c task chÃƒâ€ Ã‚Â°a xong.
- `[Story]` map tÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi user story Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ truy vÃƒÂ¡Ã‚ÂºÃ‚Â¿t.
- MÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i story Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p, cÃƒÆ’Ã‚Â³ checkpoint kiÃƒÂ¡Ã‚Â»Ã†â€™m thÃƒÂ¡Ã‚Â»Ã‚Â­ riÃƒÆ’Ã‚Âªng.
- **T057 lÃƒÆ’Ã‚Â  cÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¢ng bÃƒÂ¡Ã‚ÂºÃ‚Â¯t buÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c cuÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi cÃƒÆ’Ã‚Â¹ng** cho `frontend-guide.md`: khÃƒÆ’Ã‚Â´ng Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c coi feature hoÃƒÆ’Ã‚Â n tÃƒÂ¡Ã‚ÂºÃ‚Â¥t nÃƒÂ¡Ã‚ÂºÃ‚Â¿u tÃƒÆ’Ã‚Â i liÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡u FE cÃƒÆ’Ã‚Â²n thiÃƒÂ¡Ã‚ÂºÃ‚Â¿u/mÃƒâ€ Ã‚Â¡ hÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“.
- Commit theo tÃƒÂ¡Ã‚Â»Ã‚Â«ng task/nhÃƒÆ’Ã‚Â³m logic; tham chiÃƒÂ¡Ã‚ÂºÃ‚Â¿u `docs/development/git-workflow.md`.
- TrÃƒÆ’Ã‚Â¡nh task mÃƒâ€ Ã‚Â¡ hÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“; khÃƒÆ’Ã‚Â´ng sÃƒÂ¡Ã‚Â»Ã‚Â­a cÃƒÆ’Ã‚Â¹ng file ÃƒÂ¡Ã‚Â»Ã…Â¸ nhiÃƒÂ¡Ã‚Â»Ã‚Âu task song song.
- Regenerate Swagger (`make swagger svc=commerce`) lÃƒÂ¡Ã‚ÂºÃ‚Â·p theo tÃƒÂ¡Ã‚Â»Ã‚Â«ng story lÃƒÆ’Ã‚Â  **chÃƒÂ¡Ã‚Â»Ã‚Â§ Ãƒâ€žÃ¢â‚¬ËœÃƒÆ’Ã‚Â­ch** (incremental delivery); T059 chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° chÃƒÂ¡Ã‚ÂºÃ‚Â¡y `make swagger-combine` Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã†â€™ cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t bundle combined.

---

## Phase 10: Convergence

- [X] T062 CRITICAL Add atomic claim for brand retry: method `ClaimManualAnalysisRetry(ctx, brandItemID, now) (*entities.BrandItem, bool, error)` in `services/commerce/internal/modules/brand/domain/repositories/brand_item_interfaces.go` + impl (WHERE status = Failed OR (status = Draft AND has review reason), atomic status/TaskID transition) in `services/commerce/internal/modules/brand/infrastructure/persistence/brand_item_repo.go`; use it in `services/commerce/internal/modules/brand/application/usecase/item/retry_uc.go` so concurrent retries cannot create duplicate sessions per FR-011 (partial)
- [X] T063 Reject brand retry while the item is already processing and use the existing `ErrBrandRetryInProgress` in `services/commerce/internal/modules/brand/application/usecase/item/retry_uc.go` per FR-011 (partial)
- [X] T064 Move the fashion entityÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢DTO reason mapping out of `services/commerce/internal/modules/fashion/application/usecase/fashion_item/fashion_item_uc.go` into a mapper under `services/commerce/internal/modules/fashion/application/mapper/` per plan T034 / Constitution Code Style (contradicts)
- [X] T065 Increment `realtime_relay_failed` on the actual relay error path in `shared/presentation/sse_relay.go` (not only when Redis client is nil) per FR-026 (partial)
- [X] T066 Document `reviewReason`/`processingErrorReason` in `docs/api/fashion/constants.md` per T054 (partial)
- [X] T067 Add unit tests asserting the SSE payload completeness for `completed`/`failed`/`needs_review` (item id, progress index, summary data, error message) per FR-005 (partial)

---

## Phase 11: Convergence

- [X] T068 Strengthen brand retry anti-duplicate: reject a retry while a previously claimed retry is still in progress (e.g. clear the fashion `reviewReason` on claim via a contract method, or track a processing marker) so a rapid repeat cannot pass the `draft + reviewReason` gate again and create a duplicate session in `services/commerce/internal/modules/brand/application/usecase/item/retry_uc.go` per FR-011 / plan R7 (partial)
- [X] T069 Return `ErrBrandItemInvalidStatus` (instead of the reused `ErrBenefitInvalidStatus`) for an invalid status in `UpdateBrandItemStatus` in `services/commerce/internal/modules/brand/application/usecase/item/crud_uc.go` per plan T036 (partial)

---

## Phase 12: Convergence

- [X] T070 [P] Create shared reason-code constants package `services/commerce/internal/modules/fashion/domain/constants/analysisreason/analysisreason.go` (`uncertain_category`, `multiple_items_detected`, `full_body_outfit_detected`, `analysis_temporary_error`, `auto_retry_exceeded`) and replace hardcoded Vietnamese messages in `fashion/application/usecase/worker/worker_uc.go` (remove 2 consts), `fashion/application/usecase/worker/analyze_uc.go`, `fashion/application/usecase/worker/cleanup_uc.go`, and `wardrobe/application/usecase/wardrobe/item/item_uc_write.go` per FR-030 (missing)
- [X] T071 [P] Change `FashionItemContractDTO` json tags to camelCase (`categoryId, imageUrl, colorHex, colorHue, colorSaturation, colorLightness, processingErrorReason, reviewReason, createdAt, updatedAt`; keep `imagePublicID` as `json:"-"`) and `CategoryContractDTO.sortOrder` in `services/commerce/internal/modules/fashion/application/dto/contract_dto.go` per FR-032 (missing)
- [X] T072 Show reason only nested in `fashionItem`: remove `ProcessingErrorReason`/`ReviewReason` from `BrandItemRes` + `BrandItemBriefRes` and remove `reasonOf` usage in `services/commerce/internal/modules/brand/application/mapper/brand_item_mapper.go`; extend `FashionItemBriefRes` (`services/commerce/internal/modules/fashion/application/dto/fashion.go`) with `reviewReason`/`processingErrorReason`; add them to `FashionItemRes` + map in `services/commerce/internal/modules/wardrobe/application/mapper/wardrobe.go`; copy reasons in `fashion/presentation/contract/fashion_contract_impl.go:ListFashionItemsBrief` and in worker brief builders per FR-031 (missing)
- [X] T073 Make SSE `error` a code: set `Error: analysisreason.ErrorAnalysisTemporary` in `services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go:handleJobFailure` and keep pass-through in wardrobe/brand consumers; update `data-model.md` Ãƒâ€šÃ‚Â§5.2 + `contracts/event-contracts.md` per FR-030 (partial)
- [X] T074 AI contract P3: add `CategoryConfident bool json:"category_confident"` (flexible UnmarshalJSON, safe defaults) to `FashionMetadataResult` in `services/commerce/internal/modules/fashion/application/dto/fashion.go`; update prompt rules in `services/commerce/internal/modules/fashion/application/usecase/worker/prompts.go`; route in `ProcessBackgroundBatchUploadJob` (`analyze_uc.go`): `!IsSingleItem` ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ `markImageInvalid(reason)` ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ `failed`, `IsSingleItem && !CategoryConfident` ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ `markNeedsReview(uncertain_category)`, else completed per FR-027/FR-028/FR-029 (missing). Safe defaults: `category_confident` thiÃƒÂ¡Ã‚ÂºÃ‚Â¿u/null ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ false; mÃƒÆ’Ã‚Â¢u thuÃƒÂ¡Ã‚ÂºÃ‚Â«n `is_single_item=false` + `review_reason=uncertain_category` ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ `failed` + `multiple_items_detected`
- [X] T075 Gate `confirm-review` to `uncertain_category` only in `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go:ConfirmWardrobeItemReview` (reject other reasons) per FR-027/FR-033 (missing)
- [X] T076 Gate `retry-analysis` by reason in `wardrobe/.../item/item_uc_write.go:RetryWardrobeAnalysis` and `brand/.../usecase/item/retry_uc.go`: reject when `failed` with `multiple_items_detected`/`full_body_outfit_detected` (new error code, e.g. `ErrRetryInvalidImage`); allow `analysis_temporary_error`/`auto_retry_exceeded`/`uncertain_category` per FR-033 (missing). Khi thÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÂ¡Ã‚ÂºÃ‚Â¡i thÃƒÂ¡Ã‚ÂºÃ‚Â¥t bÃƒÂ¡Ã‚ÂºÃ‚Â¡i ÃƒÂ¡Ã‚Â»Ã…Â¸ bÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc publish job, ghi `processingErrorReason=analysis_temporary_error` cho fashion item cÃƒÆ’Ã‚Â¹ng lÃƒÆ’Ã‚Âºc set `failed` per FR-036
- [X] T077 Brand exhausted (Option 1): in `services/commerce/internal/modules/fashion/application/usecase/worker/cleanup_uc.go` scan stale via `fashionItemRepo.GetStaleProcessingItems`, map owner through wardrobe + brand contracts; add `GetWardrobeItemsByFashionItemIDs` (wardrobe repo + contract) and inject `brandContract` into `FashionWorkerUseCase` (+ wire); change `FashionProcessingExhaustedEventPayload` to `{itemId, itemType, fashionItemId, reason, userId?}`; refactor `publishProcessingRetry` per owner per FR-034/FR-035 (missing)
- [X] T078 Brand exhausted consumers: add `QueueBrandProcessingExhausted` binding (+ DLQ) in `shared/infrastructure/messaging/topology.go`; add brand `processing_exhausted_worker.go` + `HandleFashionProcessingExhausted` (set brand item `failed`) + interface; update wardrobe + ES exhausted consumers to the new payload per FR-034 (missing)
- [X] T079 Update non-frontend docs for G10: `specs/023-analyze-status-handling/contracts/api-contracts.md` (reason nested camelCase, mÃƒÆ’Ã‚Â£ lÃƒÆ’Ã‚Â½ do, gate) and `data-model.md`; regenerate `make swagger svc=commerce` + `make swagger-combine` per FR-017/FR-018 (missing)
- [X] T080 Add/adjust unit tests: AI routing 3 nhÃƒÆ’Ã‚Â¡nh, `CategoryConfident` unmarshal, confirm/retry reason gates (wardrobe+brand), reason mapping nested (wardrobe `FashionItemRes`/brief + brand no root reason), `ListFashionItemsBrief` copies reason, SSE `error` code, cleanup owner mapping + brand exhaustedÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢failed; run `make wire && make lint && make test-race` per FR-024/FR-027..FR-036 (missing)
- [X] T081 Update `specs/023-analyze-status-handling/frontend-guide.md` **Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â§y Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Â§** cho G10: (0) TL;DR ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â ÃƒÂ¡Ã‚ÂºÃ‚Â£nh khÃƒÆ’Ã‚Â´ng hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡ ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ `failed`; (1.2/1.3) `needs_review` chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° cÃƒÆ’Ã‚Â²n `uncertain_category`, `failed` gÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“m ÃƒÂ¡Ã‚ÂºÃ‚Â£nh khÃƒÆ’Ã‚Â´ng hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡; (2.1/2.2) ma trÃƒÂ¡Ã‚ÂºÃ‚Â­n caseÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢status cÃƒÂ¡Ã‚ÂºÃ‚Â­p nhÃƒÂ¡Ã‚ÂºÃ‚Â­t; (2.3) gate theo reason; (2.4) bÃƒÂ¡Ã‚ÂºÃ‚Â£ng nhÃƒÆ’Ã‚Â£n UI + hÃƒÆ’Ã‚Â nh Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ng (confirm chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° `uncertain_category`; retry chÃƒÂ¡Ã‚ÂºÃ‚Â·n ÃƒÂ¡Ã‚ÂºÃ‚Â£nh khÃƒÆ’Ã‚Â´ng hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡); **bÃƒÂ¡Ã‚ÂºÃ‚Â£ng mÃƒÆ’Ã‚Â£ lÃƒÆ’Ã‚Â½ do** (review + error + SSE `error`); (3.2) reason **nested trong `fashionItem`** + camelCase beforeÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢after + vÃƒÆ’Ã‚Â­ dÃƒÂ¡Ã‚Â»Ã‚Â¥ JSON brand/wardrobe (bÃƒÂ¡Ã‚Â»Ã‚Â root reason); (4.2) SSE `error` lÃƒÆ’Ã‚Â  mÃƒÆ’Ã‚Â£; (5) mÃƒÆ’Ã‚Â£ lÃƒÂ¡Ã‚Â»Ã¢â‚¬â€i thao tÃƒÆ’Ã‚Â¡c (`ErrRetryInvalidImage`); (6) checklist; (7) changelog. Ãƒâ€žÃ‚ÂÃƒÂ¡Ã‚ÂºÃ‚Â£m bÃƒÂ¡Ã‚ÂºÃ‚Â£o FE triÃƒÂ¡Ã‚Â»Ã†â€™n khai chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi tÃƒÆ’Ã‚Â i liÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡u nÃƒÆ’Ã‚Â y per FR-017/FR-018/SC-010 (missing)

- [X] T082 Update `specs/023-analyze-status-handling/playbook.md` for G10: bÃƒÂ¡Ã‚ÂºÃ‚Â£ng mÃƒÆ’Ã‚Â£ lÃƒÆ’Ã‚Â½ do (review/error/SSE `error`), luÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“ng ÃƒÂ¡Ã‚ÂºÃ‚Â£nh khÃƒÆ’Ã‚Â´ng hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡ (failed, khÃƒÆ’Ã‚Â´ng confirm/retry, hÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºng tÃƒÂ¡Ã‚ÂºÃ‚Â£i ÃƒÂ¡Ã‚ÂºÃ‚Â£nh khÃƒÆ’Ã‚Â¡c), `needs_review` chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° `uncertain_category`, brand exhaustedÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢failed; Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â£m bÃƒÂ¡Ã‚ÂºÃ‚Â£o bao phÃƒÂ¡Ã‚Â»Ã‚Â§ 100% trÃƒÂ¡Ã‚ÂºÃ‚Â¡ng thÃƒÆ’Ã‚Â¡i/luÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“ng per FR-014/SC-007 (missing)

### Dependencies & Execution Order (Phase 12)

- **ThÃƒÂ¡Ã‚Â»Ã‚Â© tÃƒÂ¡Ã‚Â»Ã‚Â± khuyÃƒÂ¡Ã‚ÂºÃ‚Â¿n nghÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹**: T070 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T071 (ÃƒÂ¢Ã‹â€ Ã‚Â¥ T070) ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T074 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T073 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T072 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T075 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T076 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T077 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T078 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T079 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T081 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T082 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T080.
- **File dÃƒÆ’Ã‚Â¹ng chung (phÃƒÂ¡Ã‚ÂºÃ‚Â£i chÃƒÂ¡Ã‚ÂºÃ‚Â¡y tuÃƒÂ¡Ã‚ÂºÃ‚Â§n tÃƒÂ¡Ã‚Â»Ã‚Â±, khÃƒÆ’Ã‚Â´ng [P])**:
  - `fashion/application/usecase/worker/analyze_uc.go`: T070, T073, T074.
  - `fashion/application/dto/fashion.go`: T074, T072.
  - `wardrobe/application/usecase/wardrobe/item/item_uc_write.go`: T070, T075, T076.
  - `fashion/application/usecase/worker/cleanup_uc.go`: T070, T077.
- **PhÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c logic**: T077 (producer payload/owner mapping) trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc T078 (topology + consumer brand); T075/T076 sau T074 (reason mÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi cÃƒÆ’Ã‚Â³ nghÃƒâ€žÃ‚Â©a); T079/T081 sau toÃƒÆ’Ã‚Â n bÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ code; T080 cuÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi cÃƒÆ’Ã‚Â¹ng.
- **[P]**: chÃƒÂ¡Ã‚Â»Ã¢â‚¬Â° T071 (`contract_dto.go`) thÃƒÂ¡Ã‚Â»Ã‚Â±c sÃƒÂ¡Ã‚Â»Ã‚Â± Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p, cÃƒÆ’Ã‚Â³ thÃƒÂ¡Ã‚Â»Ã†â€™ chÃƒÂ¡Ã‚ÂºÃ‚Â¡y song song T070.

---

## Phase 13: Convergence

- [X] T083 Add a terminal marker for stale-processing recovery: clear `processing_started_at` (and `last_processing_attempt_at`) on terminal transitions ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â `CompleteProcessingSuccess`, `MarkProcessingFailed`, `MarkProcessingNeedsReview`, `SetProcessingErrorReason` in `services/commerce/internal/modules/fashion/infrastructure/persistence/fashion_item_repo.go` ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â so `GetStaleProcessingItems` only matches genuinely in-flight items and cannot re-queue completed/failed/needs_review items per FR-035 (partial)
- [X] T084 Update `specs/023-analyze-status-handling/data-model.md` Ãƒâ€šÃ‚Â§5.2 SSE payload: `error` is a reason code (not "khi failed") per FR-030 / T073 (partial)

---

## Phase 14: Convergence

- [X] T085 Mark analysis as started so stale-processing recovery covers every in-flight item: set `processing_started_at` (and clear the terminal reasons) when the analyze worker begins a job ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â add a repo method (e.g. `MarkProcessingStarted(ctx, itemID, now)`) in `services/commerce/internal/modules/fashion/infrastructure/persistence/fashion_item_repo.go` and call it at the start of `ProcessBackgroundBatchUploadJob` in `services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go` (and/or set `ProcessingStartedAt` in `CreateFashionItem`), so `GetStaleProcessingItems` catches initial-upload and post-reset items stuck in processing per FR-035 (partial)

---

## Phase 15: Convergence

- [X] T086 Fix stale-processing query so recovery can retry repeatedly: in `services/commerce/internal/modules/fashion/infrastructure/persistence/fashion_item_repo.go:GetStaleProcessingItems`, drop the `last_processing_attempt_at IS NULL` condition (it excludes items right after the first republish because `IncrementProcessingRetryCount` sets that column) and instead match `processing_started_at IS NOT NULL AND processing_started_at < ?`; otherwise a stuck item is recovered only once and never reaches the exhausted threshold per FR-035 (partial)

---

## Phase 16: G11 - BÃƒÂ¡Ã‚Â»Ã‚Â confirm-review, gÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢p vÃƒÆ’Ã‚Â o retry kÃƒÆ’Ã‚Â¨m danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c (US1/FR-037..041)

**Goal**: `needs_review` (`uncertain_category`) xÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½ bÃƒÂ¡Ã‚ÂºÃ‚Â±ng `retry-analysis` kÃƒÆ’Ã‚Â¨m `categoryId`; chÃƒÂ¡Ã‚ÂºÃ‚Â¿ Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢ danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c cÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœ Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹nh tÃƒÂ¡Ã‚ÂºÃ‚Â¡o metadata Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚ÂºÃ‚Â§y Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Â§ vÃƒÆ’Ã‚Â  khÃƒÆ’Ã‚Â´ng tÃƒÆ’Ã‚Â¡i sinh `needs_review`; prompt fail-fast; brand nhÃƒÂ¡Ã‚ÂºÃ‚Â­n `categoryId` qua retry.

**Independent Test**: `needs_review` khÃƒÆ’Ã‚Â´ng truyÃƒÂ¡Ã‚Â»Ã‚Ân danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ tÃƒÂ¡Ã‚Â»Ã‚Â« chÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi; truyÃƒÂ¡Ã‚Â»Ã‚Ân danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c hÃƒÂ¡Ã‚Â»Ã‚Â£p lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¡ ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ `Processing` ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ `inWardrobe` vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi metadata + giÃƒÂ¡Ã‚Â»Ã‚Â¯ danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c; endpoint confirm cÃƒâ€¦Ã‚Â© ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ 404.

> **Superseded (F1 fix)**: cÃƒÆ’Ã‚Â¡c task US1 Phase 3 ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â **T011, T013, T014, T015, T016** ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â vÃƒÆ’Ã‚Â  **T075** (Phase 12) triÃƒÂ¡Ã‚Â»Ã†â€™n khai `confirm-review`; nay bÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ thay thÃƒÂ¡Ã‚ÂºÃ‚Â¿ bÃƒÂ¡Ã‚Â»Ã…Â¸i Phase 16 (G11): endpoint/handler/UC/repo/errors cÃƒÂ¡Ã‚Â»Ã‚Â§a confirm Ãƒâ€žÃ¢â‚¬ËœÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã‚Â£c gÃƒÂ¡Ã‚Â»Ã‚Â¡ (T087ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Å“T089). GiÃƒÂ¡Ã‚Â»Ã‚Â¯ nguyÃƒÆ’Ã‚Âªn lÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹ch sÃƒÂ¡Ã‚Â»Ã‚Â­ (append-only), khÃƒÆ’Ã‚Â´ng chÃƒÂ¡Ã‚ÂºÃ‚Â¡y lÃƒÂ¡Ã‚ÂºÃ‚Â¡i cÃƒÆ’Ã‚Â¡c task nÃƒÆ’Ã‚Â y.

- [X] T087 [US1] Remove the confirm-review endpoint: delete route `POST /:id/confirm-review` in `services/commerce/internal/modules/wardrobe/presentation/router/router.go`, handler `ConfirmWardrobeItemReview` and const `msgWardrobeConfirmReviewSuccess` in `services/commerce/internal/modules/wardrobe/presentation/handler/wardrobe_item_handler.go` (missing)
- [X] T088 [US1] Remove UC `ConfirmWardrobeItemReview` in `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go` and its interface method in `services/commerce/internal/modules/wardrobe/application/interface/usecase/wardrobe_uc.go` (missing)
- [X] T089 [US1] Remove repo `ConfirmWardrobeReview` (interface `services/commerce/internal/modules/wardrobe/domain/repositories/interfaces.go` + impl `services/commerce/internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`), remove errors `ErrConfirmReviewForbidden`/`ErrConfirmReviewCategoryRequired` in `services/commerce/internal/modules/wardrobe/application/errors/errors.go`, and delete `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_confirm_review_test.go` + trim related mock methods in `.../wardrobe/item/*_test.go` (missing)
- [X] T090 [P] [US1] Add optional `CategoryID *uuid.UUID` retry request (e.g. `RetryWardrobeAnalysisReq`) in `services/commerce/internal/modules/wardrobe/application/dto/item.go` and bind an optional body in `services/commerce/internal/modules/wardrobe/presentation/handler/wardrobe_item_handler.go:RetryWardrobeAnalysis` (missing)
- [X] T091 [P] [US4] Add optional `categoryId` to the brand retry request in `services/commerce/internal/modules/brand/application/dto/brand_item.go` and bind the body in `services/commerce/internal/modules/brand/presentation/handler/brand_portal_item_handler.go:RetryBrandItemAnalysis` (missing)
- [X] T092 [US1] Update `RetryWardrobeAnalysis` in `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go`: for `NeedsReview` require a valid category (`IsWearableCategory`; tÃƒÂ¡Ã‚Â»Ã‚Â« request hoÃƒÂ¡Ã‚ÂºÃ‚Â·c danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c cÃƒÆ’Ã‚Â³ sÃƒÂ¡Ã‚ÂºÃ‚Âµn), persist via `fashionContract.UpdateFashionItemCategory`, publish the analyze job with `FixedCategory=true` + `PreserveCategory=true`; add error `ErrRetryCategoryRequired` in `services/commerce/internal/modules/wardrobe/application/errors/errors.go` per FR-037 (missing)
- [X] T093 [US4] Update `RetryBrandItemAnalysis` in `services/commerce/internal/modules/brand/application/usecase/item/retry_uc.go`: accept `categoryId`, validate + persist via `fashionContract.UpdateFashionItemCategory`, publish with `FixedCategory=true` + `PreserveCategory=true` per FR-041 (missing)
- [X] T094 [US1] Add `FixedCategory bool` to `FashionAnalyzeJobDTO` (`services/commerce/internal/modules/fashion/application/dto/fashion.go`); branch in `ProcessBackgroundBatchUploadJob` (`services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go`) for the fixed-category path (metadata-only, skip `is_single_item`/`category_confident` gating, keep category) and make `buildCompletedItemUpdate` use the known category name for the embedding text per FR-038 (missing)
- [X] T095 [P] Prompt changes in `services/commerce/internal/modules/fashion/application/usecase/worker/prompts.go`: add `getVisionMetadataPrompt(category)` (metadata-only) and make `getVisionSystemPrompt` fail-fast in order (no item / invalid image / uncertain category / full) per FR-040 (missing)
- [X] T096 [US1] Update `specs/023-analyze-status-handling/frontend-guide.md`: remove confirm; document `retry-analysis` request `categoryId` + needs_review flow (pick category ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ retry ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ Processing ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ SSE) + "trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ sau" + checklist per FR-039/FR-017 (missing)
- [X] T097 [P] Update `specs/023-analyze-status-handling/playbook.md`: needs_review xÃƒÂ¡Ã‚Â»Ã‚Â­ lÃƒÆ’Ã‚Â½ bÃƒÂ¡Ã‚ÂºÃ‚Â±ng retry kÃƒÆ’Ã‚Â¨m danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c; gÃƒÂ¡Ã‚Â»Ã‚Â¡ luÃƒÂ¡Ã‚Â»Ã¢â‚¬Å“ng confirm per FR-014/SC-007 (missing)
- [X] T098 Regenerate Swagger (confirm removed; retry body `categoryId`): `make swagger svc=commerce` + `make swagger-combine` (missing)
- [X] T099 Add/adjust unit tests: retry `needs_review` thiÃƒÂ¡Ã‚ÂºÃ‚Â¿u danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ reject; cÃƒÆ’Ã‚Â³ danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c ÃƒÂ¢Ã¢â‚¬Â¡Ã¢â‚¬â„¢ `Processing` + `FixedCategory` job + giÃƒÂ¡Ã‚Â»Ã‚Â¯ danh mÃƒÂ¡Ã‚Â»Ã‚Â¥c; nhÃƒÆ’Ã‚Â¡nh fixed-category khÃƒÆ’Ã‚Â´ng vÃƒÆ’Ã‚Â o `needs_review`; brand retry vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi `categoryId`; prompt fail-fast; dÃƒÂ¡Ã‚Â»Ã‚Ân test confirm; run `make wire && make lint && make test-race` (missing)

### Dependencies & Execution Order (Phase 16)

- **ThÃƒÂ¡Ã‚Â»Ã‚Â© tÃƒÂ¡Ã‚Â»Ã‚Â± khuyÃƒÂ¡Ã‚ÂºÃ‚Â¿n nghÃƒÂ¡Ã‚Â»Ã¢â‚¬Â¹**: T087 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T088 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T089 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ [T090 ÃƒÂ¢Ã‹â€ Ã‚Â¥ T091] ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T092 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T093 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T094 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T095 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T096 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T097 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T098 ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬â„¢ T099.
- **File dÃƒÆ’Ã‚Â¹ng chung (tuÃƒÂ¡Ã‚ÂºÃ‚Â§n tÃƒÂ¡Ã‚Â»Ã‚Â±)**: `wardrobe/.../item/item_uc_write.go` (T088, T092); `wardrobe/.../wardrobe_item_handler.go` (T087, T090); `wardrobe/application/errors/errors.go` (T089, T092); `fashion/.../worker/analyze_uc.go` (T094).
- **PhÃƒÂ¡Ã‚Â»Ã‚Â¥ thuÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c logic**: T090 (DTO) trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc T092; T091 (brand DTO) trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc T093; T094/T095 (worker/prompt) trÃƒâ€ Ã‚Â°ÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºc T099; T098/T096/T097 sau code; T099 cuÃƒÂ¡Ã‚Â»Ã¢â‚¬Ëœi cÃƒÆ’Ã‚Â¹ng.
- **[P]**: T090, T091, T095 Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã¢â€žÂ¢c lÃƒÂ¡Ã‚ÂºÃ‚Â­p file, cÃƒÆ’Ã‚Â³ thÃƒÂ¡Ã‚Â»Ã†â€™ chÃƒÂ¡Ã‚ÂºÃ‚Â¡y song song vÃƒÂ¡Ã‚Â»Ã¢â‚¬Âºi cÃƒÆ’Ã‚Â¡c task khÃƒÆ’Ã‚Â¡c khÃƒÆ’Ã‚Â´ng Ãƒâ€žÃ¢â‚¬ËœÃƒÂ¡Ã‚Â»Ã‚Â¥ng file.

## Phase 17: Convergence (G11 doc reconciliation)

**Goal**: Ã„ÂÃ¡Â»â€œng bÃ¡Â»â„¢ tÃƒÂ i liÃ¡Â»â€¡u API/design cÃƒÂ²n sÃƒÂ³t mÃƒÂ´ tÃ¡ÂºÂ£ `confirm-review` (Ã„â€˜ÃƒÂ£ gÃ¡Â»Â¡) vÃƒÂ  hÃƒÂ nh vi `retry-analysis` kÃƒÂ¨m `categoryId`/chÃ¡ÂºÂ¿ Ã„â€˜Ã¡Â»â„¢ danh mÃ¡Â»Â¥c cÃ¡Â»â€˜ Ã„â€˜Ã¡Â»â€¹nh, Ã„â€˜Ã¡Â»Æ’ khÃ¡Â»â€ºp FR-030/FR-037/FR-039/FR-041.

- [X] T100 [P] Update `docs/api/wardrobe/wardrobe-api.md`: xÃƒÂ³a dÃƒÂ²ng mÃƒÂ´ tÃ¡ÂºÂ£ endpoint `confirm-review` (Ã„â€˜ÃƒÂ£ gÃ¡Â»Â¡) vÃƒÂ  bÃ¡Â»â€¢ sung body `categoryId` bÃ¡ÂºÂ¯t buÃ¡Â»â„¢c cho `needs_review` trong luÃ¡Â»â€œng `retry-analysis`; dÃƒÂ¹ng "trÃ†Â°Ã¡Â»â€ºc Ã¢â€ â€™ sau" per FR-039/FR-037/FR-018 (contradicts)
- [X] T101 [P] Update `docs/api/brand/brand-portal-api.md`: `retry-analysis` nhÃ¡ÂºÂ­n body tÃƒÂ¹y chÃ¡Â»Ân `categoryId` (bÃ¡ÂºÂ¯t buÃ¡Â»â„¢c khi `draft` chÃ¡Â»Â duyÃ¡Â»â€¡t) + chÃ¡ÂºÂ¿ Ã„â€˜Ã¡Â»â„¢ danh mÃ¡Â»Â¥c cÃ¡Â»â€˜ Ã„â€˜Ã¡Â»â€¹nh; lÃƒÂ m rÃƒÂµ trÃ†Â°Ã¡Â»Âng SSE `error` lÃƒÂ  **mÃƒÂ£ lÃƒÂ½ do** (khÃƒÂ´ng phÃ¡ÂºÂ£i cÃƒÂ¢u thÃƒÂ´ng bÃƒÂ¡o) per FR-041/FR-030/FR-018 (partial)
- [X] T102 Reconcile `specs/023-analyze-status-handling/spec.md`: FR-027 bÃ¡Â»Â cÃ¡Â»Â¥m "tÃ¡Â»Â« chÃ¡Â»â€˜i confirm"; FR-028 thay "cho phÃƒÂ©p confirm" bÃ¡ÂºÂ±ng "xÃ¡Â»Â­ lÃƒÂ½ bÃ¡ÂºÂ±ng `retry-analysis` kÃƒÂ¨m danh mÃ¡Â»Â¥c hÃ¡Â»Â£p lÃ¡Â»â€¡" (Ã„â€˜Ã¡Â»â€œng bÃ¡Â»â„¢ FR-037/FR-039) per FR-039 (contradicts)
- [X] T103 Reconcile `specs/023-analyze-status-handling/data-model.md`: cÃ¡ÂºÂ­p nhÃ¡ÂºÂ­t cÃƒÂ¡c mÃ¡Â»Â¥c cÃƒÂ²n mÃƒÂ´ tÃ¡ÂºÂ£ `confirm-review` (state machine Ã‚Â§2, constraint Ã‚Â§5, reason gate, gate line ~Ã‚Â§9) theo Ã‚Â§10 G11; giÃ¡Â»Â¯ append-only, khÃƒÂ´ng Ã„â€˜Ã¡Â»â€¢i schema per FR-039 (partial)

### Dependencies & Execution Order (Phase 17)

- **ThÃ¡Â»Â© tÃ¡Â»Â± khuyÃ¡ÂºÂ¿n nghÃ¡Â»â€¹**: [T100 Ã¢Ë†Â¥ T101] Ã¢Ë†Â¥ [T102 Ã¢Ë†Â¥ T103] Ã¢â‚¬â€ cÃƒÂ¡c task Ã„â€˜Ã¡Â»â„¢c lÃ¡ÂºÂ­p file, cÃƒÂ³ thÃ¡Â»Æ’ chÃ¡ÂºÂ¡y song song.
- **KhÃƒÂ´ng cÃ¡ÂºÂ§n chÃ¡ÂºÂ¡y lÃ¡ÂºÂ¡i code**: chÃ¡Â»â€° cÃ¡ÂºÂ­p nhÃ¡ÂºÂ­t tÃƒÂ i liÃ¡Â»â€¡u; gate `make lint && make test-race` giÃ¡Â»Â¯ nguyÃƒÂªn xanh.

## Phase 18: Convergence (contract/plan reconciliation)

**Goal**: Äá»“ng bá»™ cÃ¡c contract/plan cÃ²n sÃ³t hÃ nh vi G10/G11 (nguá»“n `wardrobe.event.created`, trÆ°á»ng `fixedCategory`, `error` lÃ  mÃ£; bá» `confirm-review` khá»i plan map vÃ  báº£ng mÃ£ lÃ½ do).

- [X] T104 [P] Reconcile `specs/023-analyze-status-handling/contracts/event-contracts.md`: Â§2 Ä‘á»•i nguá»“n `wardrobe.event.created` tá»« "confirm-review publish" sang "phÃ¢n tÃ­ch hoÃ n táº¥t (fashion.event.analyzed)"; Â§3 thÃªm field `fixedCategory` (bool, cháº¿ Ä‘á»™ danh má»¥c cá»‘ Ä‘á»‹nh); Â§4 sá»­a `error` lÃ  **mÃ£ lÃ½ do** per FR-030/FR-038/FR-039 (partial)
- [X] T105 [P] Reconcile `specs/023-analyze-status-handling/plan.md`: cáº­p nháº­t file-structure touch-points â€” bá» `+ ConfirmReview/claim` (`repositories/interfaces.go`), `+ confirm/retry errors` (`errors.go`), `+ ConfirmWardrobeItemReview` (`wardrobe_uc.go`), `+ confirm-review route` (`presentation/{}`); pháº£n Ã¡nh G11 per FR-039 (partial)
- [X] T106 [P] Reconcile `specs/023-analyze-status-handling/contracts/api-contracts.md`: bá» cá»™t **Confirm** trong báº£ng catalog mÃ£ lÃ½ do Â§4.1 (khÃ´ng cÃ²n `confirm-review`) per FR-039 (partial)

### Dependencies & Execution Order (Phase 18)

- **Thá»© tá»± khuyáº¿n nghá»‹**: [T104 âˆ¥ T105 âˆ¥ T106] â€” Ä‘á»™c láº­p file.
- **KhÃ´ng cáº§n cháº¡y láº¡i code**: chá»‰ cáº­p nháº­t tÃ i liá»‡u/contract; gate giá»¯ nguyÃªn xanh.
