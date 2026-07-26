# Tasks: Technical Debt & Business Logic Remediation — Sprint 2

**Input**: Design documents from `specs/009-technical-debt-remediation/`

**Scope**: Sprint 2 — Cross-Module Compliance & N+1 Fixes (Priority: P1→P2)

| US | Description | Priority |
|----|-------------|----------|
| US1 | Kiểm tra và chuẩn hóa việc dùng domain constants | P1 |
| US2 | Cross-module use case calls qua presentation/contract | P1 |
| US3 | Fix N+1 queries & missing indexes | P2 |

---

## Phase 1: Setup (Database Migrations)

- [x] T001 Create Goose migration adding indexes: `CREATE INDEX idx_wardrobe_items_user_status ON wardrobe_items(user_id, is_deleted, status)`, `CREATE INDEX idx_wardrobe_items_fashion_item ON wardrobe_items(fashion_item_id)`, `CREATE INDEX idx_wardrobe_items_type_status ON wardrobe_items(item_type, is_deleted)` in `database/migrations/`

---

## Phase 2: User Story 1 — Hardcoded constants scan & fix (Priority: P1)

**Goal**: Domain constants cross-module là allowed, nhưng module nào đang **hardcode giá trị** (string literal, số) thay vì import constant → sửa để tránh typo.

**Independent Test**: Scan codebase — không còn file nào hardcode giá trị domain constant từ module khác.

- [x] T002 [US1] Scan `internal/modules/fashion/` — tìm các file dùng `"active"`, `"inactive"`, `"draft"`, `"pending"`, số `0`, `1`, `2` trong context của `wardrobestatus`, `outfititemcontext`, `wardrobestatus` — thay bằng import `wardrobe/domain/constants/wardrobestatus`, `outfititemcontext`, etc.
- [x] T003 [US1] Scan `internal/modules/dashboard/` — tìm hardcode value của `depositstatus`, `aiusageeventstatus`, `loyaltypointlotstatus` — thay bằng import từ module sở hữu
- [x] T004 [US1] Scan `internal/modules/subscription/` — tìm hardcode value của `benefitresolution`, `loyaltypointlotstatus` — thay bằng import `brand/domain/constants/benefitresolution`, `loyaltypointlotstatus`
- [x] T005 [US1] Scan `internal/modules/brand/` — tìm hardcode value của `fashionitemtype` — thay bằng import `fashion/domain/constants/fashionitemtype`
- [x] T006 [P] [US1] Scan `internal/modules/wardrobe/` — tìm hardcode value của `fashionitemtype` — thay bằng import `fashion/domain/constants/fashionitemtype`
- [x] T007 [US1] Scan `internal/modules/fashion/` — tìm hardcode value của `loyaltypointlotstatus` — thay bằng import `brand/domain/constants/loyaltypointlotstatus`

**Checkpoint**: US1 hoàn chỉnh — không còn hardcode domain constant values trong toàn bộ codebase

---

## Phase 3: User Story 2 — Cross-module use case → contract (Priority: P1)

**Goal**: Fashion module import wardrobe use case trực tiếp → wardrobe module expose function qua `presentation/contract/`, fashion gọi qua contract interface.

**Independent Test**: Fashion module không còn import `wardrobe/application/usecase/`. Mọi cross-module data access qua contract.

- [x] T008 [P] [US2] Create wardrobe contract interface: `internal/modules/wardrobe/presentation/contract/wardrobe_contract.go` — define `IWardrobeContract` với method `GetSharedWardrobeData(ctx, ...)` và các method khác mà fashion module cần (dựa trên normalization.go và helper.go usage)
- [x] T009 [US2] Implement wardrobe contract: `internal/modules/wardrobe/presentation/contract/wardrobe_contract_impl.go` — delegate implementation to wardrobe application use case via usecase interface
- [x] T010 [US2] Refactor `internal/modules/fashion/application/usecase/ai/recommendation/parser/normalization.go` — remove import `wardrobe/application/usecase/wardrobe/shared`, inject `IWardrobeContract` và gọi qua contract
- [x] T011 [US2] Refactor `internal/modules/fashion/application/usecase/ai/chat/helper.go` — remove import `wardrobe/application/usecase/wardrobe/shared`, inject `IWardrobeContract` và gọi qua contract
- [x] T012 [US2] Update fashion DI provider: wire `IWardrobeContract` into fashion module's provider.go (fashion module nhận contract interface, không import wardrobe module)
- [x] T013 [US2] Verify: run `go vet ./internal/modules/fashion/...` — confirm no import of `wardrobe/application/usecase/`

**Checkpoint**: US2 hoàn chỉnh — không còn cross-module use case import, tất cả qua contract

---

## Phase 4: User Story 3 — Fix N+1 queries & missing indexes (Priority: P2)

**Goal**: Sửa tất cả N+1 query patterns. Tuân thủ module boundaries — không cross-module join/preload. Entity module A không preload entity module B.

**Independent Test**: Chạy từng query với 1000+ records, verify số database calls giảm, không cross-module access.

### Implementation

- [x] T014 [US3] Refactor `ai_cost_repo_impl.go` `ExpireUnknown()` — replace `for _, id := range ids { r.finishWithoutCost(ctx, id, ...) }` with single batch `UPDATE ai_usage_events SET status = ?, ... WHERE request_id IN ?` and batch `UPDATE ai_usage_period_ledgers SET ... WHERE id IN (SELECT ledger_id FROM ai_usage_events WHERE request_id IN ?)` in `internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl.go`
- [x] T015 [US3] Refactor `brand_member_repo.go` `BulkUpdate()` — wrap all `Save()` calls in single transaction: `return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error { for _, m := range members { tx.Save(m) } })` in `internal/modules/brand/infrastructure/persistence/brand_member_repo.go`
- [x] T016 [US3] Refactor `outfit_repo.go` `GetByUserID()` and `GetByUserIDPaginated()` — replace `r.GetDB(ctx)` with `r.GetQueryWithPreload(ctx)` to preload Items correctly in `internal/modules/wardrobe/infrastructure/persistence/outfit_repo.go`. Ensure preload stays within wardrobe module (OutfitItem là entity của wardrobe, không cross-module)
- [x] T017 [US3] Refactor `fashion_item_repo_hybrid_helpers.go` `recommendationSearchDocumentSubquerySQL()` — replace 8 correlated subqueries to `fashion_items` with single `LEFT JOIN fashion_items ON fashion_items.id = wardrobe_items.fashion_item_id` in `internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid_helpers.go`
- [x] T018 [US3] Refactor `fashion_item_repo_hybrid.go` `GetHybridCandidates()` — replace repeated `(SELECT embedding FROM fashion_items WHERE id = wardrobe_items.fashion_item_id)` with CTE or derived table in `internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid.go`
- [x] T019 [US3] Refactor `wardrobe_repo.go` — replace `IN (SELECT ...)` subquery patterns with `LEFT JOIN` where possible (vi dụ: `fashion_item_id IN (SELECT id FROM fashion_items WHERE ...)` → `LEFT JOIN fashion_items ON ...`) in `internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`. Đảm bảo không cross-module join (fashion_items là cùng module fashion)
- [x] T020 [US3] Run Goose migration up for new indexes (T001) — verify indexes created

**Checkpoint**: US3 hoàn chỉnh — N+1 patterns eliminated, new indexes active

---

## Dependencies & Execution Order

### Phase Dependencies

```
T001 ───── T002 ───── T003 ───── T004 ───── T005 ───── T006 ───── T007 (US1)
                  │
                  T008 ───── T009 ───── T010 ───── T011 ───── T012 ───── T013 (US2)
                  │
                  T014 ───── T015 ───── T016 ───── T017 ───── T018 ───── T019 ───── T020 (US3)
```

### Parallel Opportunities

| Tasks | Reason |
|-------|--------|
| T002, T003, T004, T005, T006, T007 | Scan các module khác nhau, không phụ thuộc |
| T008, T009 (contract) độc lập với | US1 và US3 — có thể chạy song song |
| T014, T015, T016, T017, T018, T019 | Sửa các file khác nhau, không phụ thuộc nhau |

### Implementation Strategy

1. **US1 first (P1)**: Scan + fix hardcoded constants — nhanh, ít rủi ro
2. **US2 (P1)**: Contract refactor — cần cẩn thận vì ảnh hưởng DI wiring
3. **US3 (P2)**: N+1 fixes — có thể chạy song song với US2
