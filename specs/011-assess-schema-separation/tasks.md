# Tasks: Xử lý tech debt cross-module & tách schema theo module

**Input**: Design documents từ `/specs/011-assess-schema-separation/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/contract-design.md, quickstart.md

**Tests**: Được yêu cầu cho một số task (ghi rõ trong từng task).

**Organization**: Tasks nhóm theo **Sprint → Phase → Task** (phân cấp: Sprint lớn nhất → Phase → Task → Sub-task nhỏ nhất). Mỗi task thuộc 1 user story (US1-US4) hoặc phase setup/polish.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Chạy song song (khác file, không phụ thuộc)
- **[Story]**: US1 (đánh giá), US2 (wardrobe↔fashion), US3 (dashboard), US4 (tách schema)
- Sub-task chi tiết được mô tả trong từng task theo dấu gạch con.

---

# SPRINT 1 — Đánh giá + Xử lý Tech Debt

**Mục tiêu**: (1) Đồng bộ tài liệu + đánh giá khả thi, (2) 0 cross-module DB access còn lại, (3) `go vet` + `make test` pass.

---

## PHASE 1.1 — Đánh giá & Nghiên cứu (US1) [P1]

**Goal**: Hiểu đúng phạm vi vi phạm + bản đồ schema + đánh giá khả thi trước khi code.

**Independent Test**: Đối chiếu 2 tài liệu tech debt + §VII với mã nguồn — khớp 100% (không còn `brand_conversations` sai, đủ bảng thiếu `user_subscriptions`/`loyalty_accounts`/`loyalty_tiers`/`brand_benefits`, đủ vi phạm import constants). Tài liệu đánh giá trả lời đủ 3 câu hỏi (phân loại, FK rào cản, kết luận khả thi).

- [ ] T001 [P] [US1] Bổ sung vào `docs/temp/technical-debt/cross-module-db-access.md`: vi phạm import domain constants (`fashion_item_repo_hybrid.go:7` import `wardrobestatus`), ghi chú `IWardrobeContract` đã có sẵn `GetStaleProcessingItems`/`MarkProcessingFailed`/`MarkProcessingNeedsReview`/`CompleteProcessingSuccess`, đánh dấu trạng thái "đang xử lý trong Sprint 1" cho các mục wardrobe↔fashion.
- [ ] T002 [P] [US1] Cập nhật `docs/temp/technical-debt/dashboard-cross-module-queries.md`: gỡ bảng `brand_conversations` (không còn được query), bổ sung `user_subscriptions` (dashboard_admin_repo.go:45), `loyalty_accounts` + `loyalty_tiers` (dashboard_brand_repo.go:58-60), `brand_benefits` (dashboard_brand_repo.go:94), bổ sung vi phạm import constants (`loyaltypointlotstatus`, `aiusageeventstatus`, `depositstatus`, `deposittransactiontype`).
- [ ] T003 [P] [US1] Cập nhật `docs/system-design/architecture.md` §VII: thay mục "Core Tables" (bảng ma `loyalty_points`/`campaigns`/`digital_samples`/`sample_votes`) bằng bảng mapping thực tế (42 bảng theo module, đối chiếu `database/scripts/closy_db-schema.sql`); thêm ghi chú FK xuyên module hiện hữu + kế hoạch xử lý (giữ FK tới `users`, cắt FK còn lại).
- [ ] T004 [US1] Tạo mới `docs/temp/technical-debt/schema-separation-assessment.md` gồm 3 phần: (1) phân loại & ưu tiên vi phạm (wardrobe→fashion HIGH, fashion→wardrobe HIGH, dashboard LOW), (2) danh sách FK xuyên module — **9 FK phi-users cần CẮT** (đi qua contract) + **~30 FK trỏ tới `users` được GIỮ xuyên schema** theo Q1/Q2, (3) kết luận khả thi + chi phí + rủi ro + thứ tự thực hiện; ghi nhận phương án vector search đã chốt (`GetUserFashionItemIDs` + query `fashion_items WHERE id IN (...) ORDER BY embedding`) và đính chính nhận định HNSW.

**Checkpoint 1.1**: 2 tech debt docs + §VII khớp 100% code; tài liệu đánh giá trả lời 3 câu hỏi.

---

## PHASE 1.2 — Refactor Cross-Module DB Access Wardrobe ↔ Fashion (US2) [P1]

**Goal**: Wardrobe không còn JOIN/UPDATE `fashion_items`; fashion không còn query `wardrobe_items`. Mọi access qua contract.

**Independent Test**: `go vet ./internal/modules/wardrobe/...` + scan imports pass — không còn file nào JOIN/UPDATE bảng module khác; worker claim/processing hoạt động đúng (unit test + chạy worker).

### Implementation

- [ ] T005 [P] [US2] Thêm method vào `IFashionContract` (`internal/modules/fashion/presentation/contract/fashion_contract.go`) — chỉ các thao tác ĐỒNG BỘ: `GetStaleProcessingInfo(fashionItemIDs []uuid.UUID) ([]dto.FashionProcessingInfoDTO, error)` và `ReassignCategoryForCatalogItems(fromCategoryID, toCategoryID uuid.UUID, wardrobeItemIDs []uuid.UUID) error`. KHÔNG thêm claim/processing state vào contract đồng bộ (chuyển outbox, xem T013/T014). Không trùng lặp `GetStaleProcessingItems` (đã thuộc `IWardrobeContract`).
- [ ] T006 [US2] Implement các method `IFashionContract` trên fashion module: use case + repository method query/update `fashion_items` (`GetStaleProcessingInfo`, `ReassignCategoryForCatalogItems`) trong `internal/modules/fashion/application/usecase/` và `internal/modules/fashion/infrastructure/persistence/`.
- [ ] T007 [US2] Cập nhật Wire DI wiring fashion contract impl trong `internal/di/` (fashion provider expose đủ method mới).
- [ ] T008 [P] [US2] Thêm method `GetUserFashionItemIDs(ctx, userID) ([]uuid.UUID, error)` vào `IWardrobeContract` (`internal/modules/wardrobe/presentation/contract/wardrobe_contract.go`) — implement chỉ `SELECT fashion_item_id FROM wardrobe_items WHERE user_id = ? AND status = ? AND is_deleted = false`; không trả entity.
- [ ] T009 [US2] Refactor `internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`: thay UPDATE `fashion_items` trực tiếp L468 (`ReassignCategoryForCatalogItems`) bằng gọi `IFashionContract.ReassignCategoryForCatalogItems`; các UPDATE claim/processing state (L234, L266, L376) chuyển sang outbox event (T013); thay JOIN đọc `fashion_items`/`categories` (L67-68, L200-201, L253-254, L331-332, L409-410, L424-425, L433, L460-461) bằng lấy dữ liệu qua contract + stitch bằng `sliceutils.IndexBy`/`GroupBy`.
- [ ] T010 [US2] Đảm bảo wardrobe không còn import entity/repo fashion; inject `IFashionContract` vào wardrobe provider (DI) trong `internal/di/` và `internal/modules/wardrobe/presentation/contract/wardrobe_contract_impl.go`.
- [ ] T011 [US2] Refactor `GetHybridCandidates` trong `internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid.go`: bỏ `Table("wardrobe_items")` + JOIN; thay bằng query `fashion_items WHERE id IN (?)` + `ORDER BY embedding <=> ?` + giữ filter seasonality/category/excluded trên fashion module; giữ RRF merge + fallback logic.
- [ ] T012 [US2] Cập nhật fashion provider (DI) nhận `IWardrobeContract` cho `GetHybridCandidates` trong `internal/di/` và `internal/modules/fashion/`.
- [ ] T013 [US2] Chuyển critical path worker sang eventual consistency: wardrobe ghi `wardrobe_items` + publish outbox event `wardrobe.processing.state` (payload: item_id, fashion_item_id, action, processing_version, occurred_at) trong cùng transaction (transactional outbox). Action gồm: `claim_manual_retry`, `claim_stale_retry`, `mark_failed`, `mark_needs_review`, `complete_success` tại `internal/modules/wardrobe/presentation/worker/` và `internal/shared/infrastructure/messaging/outbox.go`.
- [ ] T014 [US2] Tạo fashion consumer/worker nhận event `wardrobe.processing.state`, cập nhật `fashion_items` (claim retry / processing state); KHÔNG qua contract đồng bộ; xử lý race condition claim bằng row lock `FOR UPDATE` + version check + idempotent consumer tại `internal/modules/fashion/presentation/worker/`.
- [ ] T015 [P] [US2] Unit test flow worker → outbox event → fashion consumer update `fashion_items`; verify không mất event (outbox retry) và idempotent (không double-increment `processing_version`); đo latency claim retry ≤ +100ms so với baseline trước refactor (SC-007) trong `internal/modules/fashion/` test files.
- [ ] T016 [P] [US2] Scan hardcode giá trị domain constant trong wardrobe/fashion (thay bằng symbol import theo exception Hiến pháp II) — các file trong `internal/modules/wardrobe/`, `internal/modules/fashion/`, `internal/modules/dashboard/`.

**Checkpoint 1.2**: 0 JOIN/UPDATE cross-module trong `wardrobe_repo.go`; `GetHybridCandidates` không chạm `wardrobe_items`; chạy `make vet` + `make test` pass.

---

## PHASE 1.3 — Refactor Dashboard Cross-Module Queries (US3) [P1]

**Goal**: Dashboard chỉ truy cập qua contract reader; 0 raw query bảng module khác.

**Independent Test**: Chạy toàn bộ API dashboard (user/brand/admin) — response giống hệt trước refactor (snapshot test); scan xác nhận dashboard không còn raw query bảng module khác.

### Implementation

- [ ] T017 [P] [US3] Thêm method vào `IBrandDashboardReader` (`internal/modules/brand/presentation/contract/dashboard_reader.go`): `GetCustomerAcquisition`, `GetLoyaltyTierDistribution`, `GetPointExpiryForecast`, `GetBenefitRedemptionAnalytics`, `GetCatalogStats`, `GetCustomerSpendSegmentation`, `GetBrandNewCustomerCount`, `GetBrandCustomerClaimRate`, `GetBrandRedemptionCount`, `GetBrandItemCount`, `GetBrandItemCountByType` — implement qua brand use case + repo (chuyển query từ `dashboard_brand_repo.go` sang brand module, trả DTO).
- [ ] T018 [P] [US3] Thêm method vào `ISubscriptionDashboardReader` (`internal/modules/subscription/presentation/contract/dashboard_reader.go`): `GetSubscriptionDistribution`, `GetRevenueBreakdown`, `GetRenewalStats` — implement qua subscription use case + repo.
- [ ] T019 [P] [US3] Thêm method vào `IAICostDashboardReader` (`internal/modules/aiexecutor/presentation/contract/dashboard_reader.go`): `GetAIUsageByOperation`, `GetTotalAIRequestCount`, `GetAIErrorBreakdown` — implement qua aiexecutor use case + repo.
- [ ] T020 [P] [US3] Thêm method `GetWardrobeCategoryDistribution(ctx, userID)` vào `IWardrobeDashboardReader` (`internal/modules/wardrobe/presentation/contract/dashboard_reader.go`) — implement qua wardrobe use case + repo (trả wardrobe items + fashion_item_id của user).
- [ ] T021 [P] [US3] Tạo mới `IFashionDashboardReader` trong `internal/modules/fashion/presentation/contract/dashboard_reader.go` (fashion module sở hữu `fashion_items`/`categories`) với method: `GetCategoriesByIDs(ctx, categoryIDs)`, `GetCategoryIDsByFashionItemIDs(ctx, fashionItemIDs)` + impl + DI.
- [ ] T022 [US3] Refactor `internal/modules/dashboard/infrastructure/persistence/dashboard_brand_repo.go`, `dashboard_admin_repo.go`, `dashboard_user_repo.go`: inject các contract reader (DI), thay từng raw query bằng gọi reader method + map sang DTO hiện tại; xoá query bảng module khác, giữ query bảng dashboard snapshot của chính module.
- [ ] T023 [US3] Snapshot test dashboard: chạy toàn bộ API dashboard (user/brand/admin) so sánh response trước/sau refactor giống hệt; unit test từng reader method (empty data → zero value, đúng format) trong `internal/modules/dashboard/`.

**Checkpoint 1.3**: Dashboard hoạt động đúng; SC-004 đạt (0 cross-module DB access).

---

# SPRINT 2 — Tách Database Schema Vật Lý

**Mục tiêu**: 42 bảng + 1 sequence sang schema module; FK xuyên module được xử lý theo Q1/Q2; migration thuần `SET SCHEMA` có Down đầy đủ; toàn bộ test suite pass.

---

## PHASE 2.1 — Migration tách schema (US4) [P2]

**Goal**: Mỗi module có schema PostgreSQL riêng; migration round-trip không mất dữ liệu.

**Independent Test**: `make migration-up` + `make migration-down` trên DB test — round-trip không mất dữ liệu; sau khi up, mọi bảng ở đúng schema module; app khởi động với search_path và chạy test suite pass.

### Implementation

- [ ] T024 [US4] Thiết kế schema mapping mục tiêu trong `docs/temp/technical-debt/schema-separation-mapping.md`: liệt kê 42 bảng → schema đích theo module (identity: `users`/`refresh_tokens`/`user_style_profiles`; wardrobe: `wardrobe_items`/`outfits`/`outfit_items`; fashion: `fashion_items`/`categories`/`conversational_contexts`/`messages`; brand: 15 bảng; subscription: 10 bảng + sequence `deposit_transactions_order_code_seq`; aiexecutor: 6 bảng; dashboard: 3 bảng; shared: `outbox_records`); liệt kê 9 FK xuyên module cần cắt + ~30 FK tới `users` giữ xuyên schema.
- [ ] T025 [US4] Viết Goose migration `database/migrations/20260801000000_separate_module_schemas.sql`: **Up** = `CREATE SCHEMA` từng module + `ALTER TABLE ... SET SCHEMA` (42 bảng) + `ALTER SEQUENCE deposit_transactions_order_code_seq SET SCHEMA subscription` + `ALTER INDEX ... SET SCHEMA` (index unique/partial) + `DROP CONSTRAINT` 9 FK xuyên module + `ADD CONSTRAINT` giữ FK tới `users` (schema-qualified `identity.users`); **Down** = `SET SCHEMA public` ngược lại + tái lập 9 FK xuyên module (check dữ liệu không orphan trước `ADD CONSTRAINT`) + drop schema module.
- [ ] T026 [US4] Chạy `make migration-up` + `make migration-down` trên DB test; verify round-trip không mất dữ liệu (so sánh count bảng trước/sau).
- [ ] T027 [US4] Cập nhật DSN thêm `search_path` = `identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared,public` trong `internal/shared/infrastructure/db/db.go` + `internal/config/`; xác nhận GORM `NamingStrategy` không bị ảnh hưởng; chạy toàn bộ test suite trên schema tách.

**Checkpoint 2.1**: Migration round-trip an toàn; app hoạt động với search_path multi-schema.

---

## PHASE 2.2 — Xử lý FK xuyên module còn lại + verify (US4) [P2]

**Goal**: Integrity sau khi cắt FK được enforce qua contract; toàn bộ luồng nghiệp vụ hoạt động đúng.

**Independent Test**: `make check` pass; các luồng nghiệp vụ chính (worker, AI recommendation, dashboard, payment, loyalty) hoạt động đúng; HNSW index vẫn dùng.

### Implementation

- [ ] T028 [US4] Verify integrity sau khi cắt FK: (a) với mỗi FK bị cắt (`wardrobe_items→fashion_items`, `brand_items→fashion_items`, `digital_sample_responses→outfits`, `subscription_plans→ai_cost_policies`, `ai_policy_grants→subscription_plans`, ...) xác nhận mọi luồng ghi liên quan đều qua contract; (b) rà soát xoá mềm/delete cascade bị mất do cắt FK — bổ sung logic xoá qua contract nếu cần; (c) verify các FK GIỮ (xuyên schema trỏ tới `users`, ~30 FK) vẫn enforce: xoá `users` → cascade delete ở các bảng liên quan, chèn bản ghi vi phạm `user_id` không tồn tại → bị chặn bởi FK (scan `internal/modules/`).
- [ ] T029 [US4] Chạy `make check` (fmt + tidy + generate + vet + test + build) pass.
- [ ] T030 [US4] Chạy các luồng nghiệp vụ chính trên schema tách: worker processing, AI recommendation (vector search), dashboard (user/brand/admin), payment (PayOS webhook → deposit → wallet → subscription), loyalty (earn/redeem/expire).
- [ ] T031 [US4] Verify HNSW index vẫn dùng trong vector search: `EXPLAIN (ANALYZE)` query `SELECT id FROM fashion_items WHERE id IN (...) ORDER BY embedding <=> '...' LIMIT 20` — phải thấy `Index Scan using fitems_embedding_cosine_idx`.
- [ ] T032 [US4] Cập nhật `database/scripts/closy_db-schema.sql` (dump schema mới schema-qualified) + `docs/system-design/architecture.md` §VII theo schema tách.

**Checkpoint 2.2**: SC-005, SC-006 đạt; round-trip migration không mất dữ liệu.

---

## PHASE 2.3 — Polish & Cross-Cutting Concerns

**Purpose**: Hoàn thiện và xác nhận tổng thể.

- [ ] T033 [P] Chạy quickstart.md validation: toàn bộ kịch bản Sprint 1 & Sprint 2 trong `specs/011-assess-schema-separation/quickstart.md` pass (kiểm tra HNSW index, outbox event, migration round-trip, schema mapping, search_path).
- [ ] T034 [P] Rà soát toàn bộ: không còn hardcode domain constant, không còn import module khác trong use case/repo, tài liệu `docs/` không còn thông tin lỗi thời về schema.

**Checkpoint 2.3**: Toàn bộ checklist feature đạt; sẵn sàng integration & PR.

---

## Dependencies & Execution Order

### Phase Dependencies

```
SPRINT 1
├── PHASE 1.1 (đánh giá — US1): T001-T003 song song → T004
├── PHASE 1.2 (wardrobe↔fashion — US2): phụ thuộc T004
│     ├── T005 → T006 → T007
│     ├── T008 → T009 → T010 (phụ thuộc T005-T007 cho contract impl)
│     ├── T011 → T012 (phụ thuộc T008)
│     ├── T013 → T014 → T015 (phụ thuộc T005-T007)
│     └── T016 (song song)
└── PHASE 1.3 (dashboard — US3): độc lập với PHASE 1.2, song song
      ├── T017, T018, T019, T020, T021 (song song nhau)
      ├── T022 (phụ thuộc T017-T021)
      └── T023

SPRINT 2 (chỉ khi Sprint 1 xong — SC-004 đạt)
├── PHASE 2.1: T024 → T025 → T026 → T027
└── PHASE 2.2: T028 → T029 → T030 → T031 → T032
```

### User Story Dependencies

- **US1 (P1)**: Không phụ thuộc — bắt đầu ngay.
- **US2 (P1)**: Phụ thuộc US1 (T004 — tài liệu đánh giá chốt phạm vi).
- **US3 (P1)**: Độc lập với US2 — có thể song song.
- **US4 (P2)**: Chỉ bắt đầu khi Sprint 1 xong (SC-004: 0 cross-module DB access).

### Within Each User Story

- Contract interface trước, implement sau.
- Fashion expose contract trước → wardrobe gọi.
- Dashboard reader contract trước → dashboard refactor.

### Parallel Opportunities

| Tasks | Lý do |
|-------|--------|
| T001, T002, T003 | Sửa 3 tài liệu khác nhau, độc lập |
| T005-T016 (US2) vs T017-T023 (US3) | 2 module khác nhau, không phụ thuộc |
| T017, T018, T019, T020, T021 | 5 contract reader khác nhau |
| T016 | Scan constants độc lập |
| T033, T034 | Polish, độc lập |

### Parallel Example: Phase 1.2 (US2)

```bash
# Launch contract interfaces together:
Task: "T005 IFashionContract processing/claim methods"
Task: "T008 GetUserFashionItemIDs trong IWardrobeContract"

# Launch reader contracts together (US3):
Task: "T017 IBrandDashboardReader"
Task: "T018 ISubscriptionDashboardReader"
Task: "T019 IAICostDashboardReader"
Task: "T020 IWardrobeDashboardReader"
Task: "T021 IFashionDashboardReader"
```

---

## Implementation Strategy

### MVP First (Sprint 1 only)

1. Hoàn thành PHASE 1.1 (đánh giá) — chốt phạm vi chính xác.
2. Hoàn thành PHASE 1.2 (wardrobe↔fashion) — nhóm HIGH nhất.
3. Hoàn thành PHASE 1.3 (dashboard) — nhóm lớn nhất về số lượng.
4. **STOP và VALIDATE**: SC-004 (0 cross-module DB access) + `make check` pass.

### Incremental Delivery

1. Sprint 1 → deploy (tech debt xử lý xong, schema chưa đổi).
2. Sprint 2 → tách schema (chỉ khi Sprint 1 pass).

### Parallel Team Strategy

Với nhiều developer:
1. Team hoàn thành PHASE 1.1 cùng nhau.
2. Developer A: PHASE 1.2 (wardrobe↔fashion).
3. Developer B: PHASE 1.3 (dashboard).
4. Sprint 2: cả team tập trung vào migration + verify.

---

## Notes

- [P] tasks = khác file, không phụ thuộc.
- [Story] label maps task tới user story để traceability.
- Mỗi user story test độc lập được.
- Chạy `make vet` + `make test` sau mỗi Phase; `make check` sau mỗi Sprint.
- Không sửa `database/init`; mọi thay đổi DB qua Goose migration mới.
- Commit sau mỗi task hoặc nhóm task logic.
- Sprint 2 chỉ bắt đầu khi SC-004 đạt.
