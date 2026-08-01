# Implementation Plan: Xử lý tech debt cross-module & tách schema theo module

**Branch**: `011-assess-schema-separation` | **Date**: 2026-08-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/011-assess-schema-separation/spec.md`

## Summary

Feature triển khai theo **2 Sprint lớn** (phân cấp: Sprint → Phase → Task → Sub-task):

- **Sprint 1 — Đánh giá + Xử lý tech debt**: (a) phase đánh giá/nghiên cứu làm tiền đề, (b) refactor cross-module DB access wardrobe↔fashion qua contract (kèm chuyển claim/processing state sang eventual consistency qua outbox, giữ đọc stale info + reassign category đồng bộ qua contract), (c) refactor dashboard queries qua contract reader.
- **Sprint 2 — Tách database schema vật lý**: tách 42 bảng + 1 sequence từ `public` sang schema riêng theo module (identity, wardrobe, fashion, brand, subscription, aiexecutor, dashboard, shared), cắt ~9 FK xuyên module (giữ ~30 FK tới `users`), migration thuần `SET SCHEMA` có Down đầy đủ, cập nhật DSN/search_path/GORM.

Các quyết định đã chốt ở clarify: giữ FK xuyên schema tới `users` (Q1), cắt FK xuyên module còn lại (Q2), không phân quyền DB-level (Q3), migration thuần `SET SCHEMA` viết Down (Q4), eventual consistency qua outbox cho critical path worker (Q5).

## Technical Context

**Language/Version**: Go 1.25+ (Gin, GORM, Google Wire, Goose)

**Primary Dependencies**: gorm.io/gorm, gorm.io/driver/postgres, github.com/pressly/goose/v3, github.com/google/wire, RabbitMQ (amqp), outbox pattern

**Storage**: PostgreSQL 16 + pgvector (HNSW index trên `fashion_items.embedding`, `user_style_profiles.taste_embedding`)

**Testing**: Go test (`make test`), `go vet` (`make vet`), `make check` (fmt + tidy + generate + vet + test + build)

**Target Platform**: Linux server (Docker), single deployable Gin service

**Project Type**: Modular monolith web-service (single DB schema hiện tại `public`)

**Performance Goals**: Vector search phải tiếp tục dùng HNSW index (không regression); worker claim retry không bị tăng latency thêm quá 100ms

**Constraints**: 1 connection chung tới DB (không phân quyền DB-level); không sửa `database/init`; mọi thay đổi DB qua Goose migration mới; không cross-module import/join/preload entity; eventual consistency chấp nhận độ trễ nhỏ

**Scale/Scope**: 7 module sở hữu bảng + shared; 42 bảng + 1 sequence cần `SET SCHEMA`

## Constitution Check

*GATE: Must pass trước Phase 0 research. Re-check sau Phase 1 design.*

| # | Nguyên tắc Hiến pháp | Trạng thái | Đánh giá cho feature này |
|---|----------------------|------------|---------------------------|
| I | Pragmatic Modular Monolith — cấm cross-module joins/preloads | ⚠️ Vi phạm hiện hữu | Feature này SỬA vi phạm: Sprint 1 refactor hết cross-module JOIN/UPDATE qua contract. Đúng tinh thần Hiến pháp |
| II | Strict Module Boundaries & DTO Contracts | ✅ Tuân thủ | Mọi cross-module access qua `presentation/contract/` bằng DTO; stitch bằng sliceutils |
| III | Event-Driven + Outbox | ✅ Tuân thủ | Q5 chốt: critical path chuyển sang eventual consistency, publish event qua transactional outbox |
| IV | Clean Architecture & Strict Layer | ✅ Tuân thủ | Refactor giữ nguyên cấu trúc presentation → application → domain ← infrastructure |
| V | Transaction & Error Handling (UoW) | ✅ Tuân thủ | Event outbox ghi trong cùng transaction với business op |
| VI | AI Pipeline & Cost Control | ✅ Tuân thủ | Không thay đổi logic AI; giữ HNSW index |

**Gate**: Không có violation không hợp lý. Các vi phạm hiện hữu (I) chính là đối tượng được xử lý trong feature.

## Project Structure

### Documentation (this feature)

```text
specs/011-assess-schema-separation/
├── spec.md               # Feature specification
├── plan.md               # This file
├── research.md           # Phase 0 output (Sprint 1 - Phase 1.1)
├── data-model.md         # Phase 1 output - schema mapping theo module
├── quickstart.md         # Phase 1 output - validation guide
├── contracts/            # Phase 1 output - contract design
├── tasks.md              # /speckit-tasks output (NOT created by plan)
└── checklists/
```

### Source Code (repository root)

```text
internal/
├── modules/
│   ├── wardrobe/
│   │   ├── presentation/contract/    # + GetUserFashionItemIDs, + reader methods
│   │   └── infrastructure/persistence/wardrobe_repo.go   # bỏ JOIN/UPDATE fashion
│   ├── fashion/
│   │   ├── presentation/contract/    # + processing/claim contract, + IFashionDashboardReader
│   │   └── infrastructure/persistence/fashion_item_repo_hybrid.go  # query id IN + embedding
│   ├── dashboard/
│   │   └── infrastructure/persistence/   # 3 file repo → gọi contract reader
│   ├── brand/        # + contract reader methods (loyalty/customer/catalog stats)
│   ├── subscription/ # + contract reader methods (subscription/renewal/payment)
│   ├── aiexecutor/   # + contract reader methods (ai usage)
│   └── identity/     # (sprint 2) schema riêng
├── shared/
│   ├── infrastructure/db/db.go        # Sprint 2: search_path/schema config
│   ├── infrastructure/messaging/outbox.go  # reuse cho event processing
│   └── ...
internal/di/          # Wire DI cập nhật contract wiring
database/migrations/  # + migration tách schema (Sprint 2)
database/scripts/closy_db-schema.sql  # schema dump đối chiếu
docs/                 # + cập nhật architecture.md §VII, 2 tech debt docs, tài liệu đánh giá
```

**Structure Decision**: Giữ nguyên cấu trúc modular monolith hiện có; chỉ thêm method contract, refactor repo, thêm migration. Không tạo project/package mới ngoài contract files trong module hiện hữu.

## Complexity Tracking

> Feature triển khai 2 sprint lớn nhưng không vi phạm Hiến pháp → không cần justify complexity. Các task được phân cấp Sprint → Phase → Task → Sub-task ở dưới.

---

# KẾ HOẠCH TRIỂN KHAI CHI TIẾT

> Phân cấp: **Sprint (lớn nhất) → Phase → Task → Sub-task (nhỏ nhất)**

---

## SPRINT 1 — Đánh giá + Xử lý Tech Debt

**Mục tiêu**: (1) Đồng bộ tài liệu + đánh giá khả thi, (2) 0 cross-module DB access còn lại, (3) `go vet` + `make test` pass.

---

### PHASE 1.1 — Đánh giá & Nghiên cứu (US1) [P1]

**Goal**: Hiểu đúng phạm vi vi phạm + bản đồ schema + đánh giá khả thi trước khi code.

#### Task 1.1.1 — Cập nhật tài liệu `cross-module-db-access.md`

**File**: `docs/temp/technical-debt/cross-module-db-access.md`

- Sub-task 1.1.1.1: Bổ sung vi phạm import domain constants cross-module vào bảng vi phạm (kiểm chứng `fashion_item_repo_hybrid.go:7` import `wardrobestatus`).
- Sub-task 1.1.1.2: Ghi chú `IWardrobeContract` đã có sẵn `GetStaleProcessingItems`, `MarkProcessingFailed`, `MarkProcessingNeedsReview`, `CompleteProcessingSuccess` (minh chứng hướng fix khả thi).
- Sub-task 1.1.1.3: Đánh dấu trạng thái "đang xử lý trong Sprint 1" cho các mục thuộc wardrobe↔fashion.

#### Task 1.1.2 — Cập nhật tài liệu `dashboard-cross-module-queries.md`

**File**: `docs/temp/technical-debt/dashboard-cross-module-queries.md`

- Sub-task 1.1.2.1: Gỡ bảng `brand_conversations` (không còn được query trong `dashboard_brand_repo.go`).
- Sub-task 1.1.2.2: Bổ sung các bảng thiếu: `user_subscriptions` (admin repo L45), `loyalty_accounts` + `loyalty_tiers` (brand repo L58-60), `brand_benefits` (brand repo L94).
- Sub-task 1.1.2.3: Bổ sung mục vi phạm import domain constants (`loyaltypointlotstatus`, `aiusageeventstatus`, `depositstatus`, `deposittransactiontype`).

#### Task 1.1.3 — Cập nhật `architecture.md` §VII

**File**: `docs/system-design/architecture.md`

- Sub-task 1.1.3.1: Thay mục "Core Tables" (bảng ma: `loyalty_points`, `campaigns`, `digital_samples`, `sample_votes`) bằng bảng mapping thực tế (42 bảng) theo module sở hữu (đối chiếu `database/scripts/closy_db-schema.sql`).
- Sub-task 1.1.3.2: Thêm ghi chú về FK xuyên module hiện hữu (rào cản tách schema) và kế hoạch xử lý (giữ FK tới `users`, cắt FK còn lại).

#### Task 1.1.4 — Tạo tài liệu đánh giá khả thi tách schema

**File mới**: `docs/temp/technical-debt/schema-separation-assessment.md`

- Sub-task 1.1.4.1: Phần 1 — Phân loại & ưu tiên vi phạm (wardrobe→fashion HIGH, fashion→wardrobe HIGH, dashboard LOW).
- Sub-task 1.1.4.2: Phần 2 — Danh sách FK xuyên module: **9 FK phi-users cần CẮT** (đi qua contract) + **~30 FK trỏ tới `users` được GIỮ xuyên schema** theo Q1/Q2.
- Sub-task 1.1.4.3: Phần 3 — Kết luận khả thi: chi phí, rủi ro (migration consolidated, users hub, vector search), thứ tự thực hiện, giới hạn chấp nhận được.
- Sub-task 1.1.4.4: Ghi nhận phương án vector search đã chốt (`GetUserFashionItemIDs` + query `fashion_items WHERE id IN (...) ORDER BY embedding`) và đính chính nhận định HNSW.

**Checkpoint 1.1**: 2 tech debt docs + §VII khớp 100% code; tài liệu đánh giá trả lời 3 câu hỏi.

---

### PHASE 1.2 — Refactor Cross-Module DB Access Wardrobe ↔ Fashion (US2) [P1]

**Goal**: Wardrobe không còn JOIN/UPDATE `fashion_items`; fashion không còn query `wardrobe_items`. Mọi access qua contract.

#### Task 1.2.1 — Fashion expose contract đồng bộ (stale info + reassign category)

**File**: `internal/modules/fashion/presentation/contract/fashion_contract.go` (+ impl, + use case, + repo)

- Sub-task 1.2.1.1: Thêm vào `IFashionContract` (chỉ các thao tác ĐỒNG BỘ): `GetStaleProcessingInfo(fashionItemIDs []uuid.UUID) ([]dto.FashionProcessingInfoDTO, error)` và `ReassignCategoryForCatalogItems(fromCategoryID, toCategoryID uuid.UUID, wardrobeItemIDs []uuid.UUID) error`.
- Sub-task 1.2.1.2: Implement qua use case fashion + repository method trên `fashion_items`.
- Sub-task 1.2.1.3: Cập nhật Wire DI wiring fashion contract impl.

> **Phân định (I3)**: claim retry + processing state (mark failed / needs review / complete success) KHÔNG thêm vào `IFashionContract` đồng bộ — chuyển sang outbox event (Task 1.2.4). `GetStaleProcessingItems` vốn thuộc `IWardrobeContract` (operate trên `wardrobe_items`) được giữ nguyên, KHÔNG trùng lặp sang `IFashionContract`.

#### Task 1.2.2 — Wardrobe gọi qua `IFashionContract` thay vì raw JOIN/UPDATE

**File**: `internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go` (+ wardrobe application/usecase, + wardrobe contract impl)

- Sub-task 1.2.2.1: Thay UPDATE `fashion_items` trực tiếp (L468 `ReassignCategoryForCatalogItems`) bằng gọi `IFashionContract.ReassignCategoryForCatalogItems`; các UPDATE claim/processing state (L234, L266, L376) chuyển sang outbox event (Task 1.2.4).
- Sub-task 1.2.2.2: Thay các JOIN đọc `fashion_items`/`categories` (L67-68, L200-201, L253-254, L331-332, L409-410, L424-425, L433, L460-461) bằng lấy fashion data qua contract + stitch bằng `sliceutils.IndexBy`/`GroupBy`.
- Sub-task 1.2.2.3: Đảm bảo wardrobe không còn import entity/repo fashion; inject `IFashionContract` vào wardrobe provider (DI).
- Sub-task 1.2.2.4: `go vet ./internal/modules/wardrobe/...` pass; scan xác nhận không còn `fashion_items` raw access.

#### Task 1.2.3 — Refactor `GetHybridCandidates` theo phương án `GetUserFashionItemIDs`

**File**: `internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid.go`, `internal/modules/wardrobe/presentation/contract/wardrobe_contract.go`

- Sub-task 1.2.3.1: Thêm method `GetUserFashionItemIDs(ctx, userID)` vào `IWardrobeContract` (implement: chỉ `SELECT fashion_item_id FROM wardrobe_items WHERE user_id = ? AND status = ? AND is_deleted = false`).
- Sub-task 1.2.3.2: Refactor `GetHybridCandidates`: bỏ `Table("wardrobe_items")` + JOIN; thay bằng query `fashion_items WHERE id IN (?)` + `ORDER BY embedding <=> ?` + giữ filter seasonality/category/excluded trên fashion module.
- Sub-task 1.2.3.3: Giữ nguyên RRF merge + fallback logic; verify HNSW index vẫn được dùng (EXPLAIN).
- Sub-task 1.2.3.4: Cập nhật fashion provider (DI) nhận `IWardrobeContract`.

#### Task 1.2.4 — Chuyển critical path worker sang eventual consistency (outbox)

**File**: wardrobe worker (`internal/modules/wardrobe/presentation/worker/`), outbox infra (`internal/shared/infrastructure/messaging/outbox.go`), fashion consumer

- Sub-task 1.2.4.1: Wardrobe ghi `wardrobe_items` + publish outbox event `wardrobe.processing.state` (payload chứa itemID, fashionItemID, action, processingVersion, occurred_at) trong cùng transaction (transactional outbox, Hiến pháp III). Action gồm: `claim_manual_retry`, `claim_stale_retry`, `mark_failed`, `mark_needs_review`, `complete_success`.
- Sub-task 1.2.4.2: Fashion module tạo consumer/worker nhận event, cập nhật `fashion_items` (claim retry / processing state) — KHÔNG qua contract đồng bộ.
- Sub-task 1.2.4.3: Xử lý race condition claim (2 worker): row lock `FOR UPDATE` trên bảng sở hữu + version check; idempotent consumer.
- Sub-task 1.2.4.4: Unit test cho flow worker → event → fashion update; verify không mất event (outbox retry); đo latency claim retry ≤ +100ms so với trước (SC-007).

#### Task 1.2.5 — Chuẩn hóa import domain constants cross-module

**File**: scan toàn bộ wardrobe/fashion/dashboard

- Sub-task 1.2.5.1: Scan hardcode giá trị domain constant (thay bằng symbol import, theo exception Hiến pháp II).
- Sub-task 1.2.5.2: `go vet` + scan xác nhận không còn hardcode.

**Checkpoint 1.2**: 0 JOIN/UPDATE cross-module trong wardrobe_repo; `GetHybridCandidates` không chạm `wardrobe_items`; claim/processing state qua outbox, đọc stale info + reassign category qua contract; `make test` pass.

---

### PHASE 1.3 — Refactor Dashboard Cross-Module Queries (US3) [P1]

**Goal**: Dashboard chỉ truy cập qua contract reader; 0 raw query bảng module khác.

#### Task 1.3.1 — Bổ sung `IBrandDashboardReader`

**File**: `internal/modules/brand/presentation/contract/dashboard_reader.go` (+ impl, + use case, + repo brand)

- Sub-task 1.3.1.1: Thêm method: `GetCustomerAcquisition`, `GetLoyaltyTierDistribution`, `GetPointExpiryForecast`, `GetBenefitRedemptionAnalytics`, `GetCatalogStats`, `GetCustomerSpendSegmentation`, `GetBrandNewCustomerCount`, `GetBrandCustomerClaimRate`, `GetBrandRedemptionCount`, `GetBrandItemCount`, `GetBrandItemCountByType`.
- Sub-task 1.3.1.2: Implement qua brand use case (chuyển query từ `dashboard_brand_repo.go` sang brand module, trả DTO).

#### Task 1.3.2 — Bổ sung `ISubscriptionDashboardReader`

**File**: `internal/modules/subscription/presentation/contract/dashboard_reader.go` (+ impl, + use case, + repo)

- Sub-task 1.3.2.1: Thêm method: `GetSubscriptionDistribution`, `GetRevenueBreakdown`, `GetRenewalStats`.

#### Task 1.3.3 — Bổ sung `IAICostDashboardReader`

**File**: `internal/modules/aiexecutor/presentation/contract/dashboard_reader.go` (+ impl, + use case, + repo)

- Sub-task 1.3.3.1: Thêm method: `GetAIUsageByOperation`, `GetTotalAIRequestCount`, `GetAIErrorBreakdown`.

#### Task 1.3.4 — Bổ sung `IWardrobeDashboardReader` + tạo `IFashionDashboardReader`

**File**: `internal/modules/wardrobe/presentation/contract/dashboard_reader.go`, `internal/modules/fashion/presentation/contract/dashboard_reader.go` (mới)

- Sub-task 1.3.4.1: Thêm `GetWardrobeCategoryDistribution` vào `IWardrobeDashboardReader`.
- Sub-task 1.3.4.2: Tạo `IFashionDashboardReader` (fashion module sở hữu `fashion_items`/`categories`).

#### Task 1.3.5 — Refactor 3 dashboard repos qua contract reader

**File**: `internal/modules/dashboard/infrastructure/persistence/dashboard_brand_repo.go`, `dashboard_admin_repo.go`, `dashboard_user_repo.go`

- Sub-task 1.3.5.1: Dashboard inject các contract reader (DI); thay từng raw query bằng gọi reader method + map sang DTO hiện tại.
- Sub-task 1.3.5.2: Xoá các query bảng module khác; giữ lại query bảng dashboard snapshot của chính module.
- Sub-task 1.3.5.3: `go vet` + scan xác nhận dashboard không còn raw access bảng brand/subscription/aiexecutor/wardrobe/fashion.

#### Task 1.3.6 — Snapshot test dashboard

**File**: test files trong dashboard module

- Sub-task 1.3.6.1: Chạy toàn bộ API dashboard (user/brand/admin), so sánh response trước/sau refactor giống hệt.
- Sub-task 1.3.6.2: Unit test cho từng reader method (empty data → zero value, đúng format).

**Checkpoint 1.3**: Dashboard hoạt động đúng; SC-004 đạt (0 cross-module DB access).

---

## SPRINT 2 — Tách Database Schema Vật Lý

**Mục tiêu**: 42 bảng + 1 sequence sang schema module; FK xuyên module được xử lý theo Q1/Q2; migration thuần `SET SCHEMA` có Down đầy đủ; toàn bộ test suite pass.

### PHASE 2.1 — Migration tách schema

#### Task 2.1.1 — Thiết kế schema mapping mục tiêu

**File mới**: `docs/temp/technical-debt/schema-separation-mapping.md` (hoặc ghi trong data-model.md)

- Sub-task 2.1.1.1: Liệt kê 42 bảng → schema đích theo module: identity (`users`, `refresh_tokens`, `user_style_profiles`), wardrobe (`wardrobe_items`, `outfits`, `outfit_items`), fashion (`fashion_items`, `categories`, `conversational_contexts`, `messages`), brand (15 bảng `brands*`/`loyalty*`/`benefit*`), subscription (10 bảng + sequence `deposit_transactions_order_code_seq`), aiexecutor (6 bảng `ai_*`), dashboard (3 bảng `dashboard_*`), shared (`outbox_records`).
- Sub-task 2.1.1.2: Liệt kê 9 FK xuyên module cần cắt + ~30 FK tới `users` giữ xuyên schema.

#### Task 2.1.2 — Viết Goose migration tách schema (Up + Down)

**File mới**: `database/migrations/20260801000000_separate_module_schemas.sql`

- Sub-task 2.1.2.1: **Up**: `CREATE SCHEMA` cho từng module; `ALTER TABLE ... SET SCHEMA` (42 bảng); `ALTER SEQUENCE deposit_transactions_order_code_seq SET SCHEMA subscription`; `ALTER INDEX ... SET SCHEMA` (index unique/partial); `DROP CONSTRAINT` 9 FK xuyên module; `ADD CONSTRAINT` giữ FK tới `users` (schema-qualified `identity.users`).
- Sub-task 2.1.2.2: **Down**: `SET SCHEMA public` ngược lại; tái lập 9 FK xuyên module (kiểm tra dữ liệu không orphan trước `ADD CONSTRAINT`); drop các schema module.
- Sub-task 2.1.2.3: Chạy `make migration-up` + `make migration-down` trên DB test, verify round-trip không mất dữ liệu.

#### Task 2.1.3 — Cập nhật search_path / GORM schema config

**File**: `internal/shared/infrastructure/db/db.go`, `internal/config` (+ config yaml)

- Sub-task 2.1.3.1: Cấu hình DSN thêm `search_path` bao gồm tất cả schema module (`identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared,public`).
- Sub-task 2.1.3.2: Xác nhận GORM `NamingStrategy` không bị ảnh hưởng (bảng đã schema-qualified qua search_path); các raw SQL hiện có hoạt động đúng nhờ search_path.
- Sub-task 2.1.3.3: Chạy toàn bộ test suite trên schema tách.

### PHASE 2.2 — Xử lý FK xuyên module còn lại + verify

#### Task 2.2.1 — Verify integrity qua contract sau khi cắt FK

**File**: scan toàn bộ module ghi chéo

- Sub-task 2.2.1.1: Với mỗi FK bị cắt (`wardrobe_items→fashion_items`, `brand_items→fashion_items`, `digital_sample_responses→outfits`, `subscription_plans→ai_cost_policies`, `ai_policy_grants→subscription_plans`, ...), xác nhận mọi luồng ghi liên quan đều qua contract (đã hoàn thành Sprint 1).
- Sub-task 2.2.1.2: Rà soát xoá mềm/delete cascade bị mất do cắt FK — bổ sung logic xoá qua contract nếu cần.
- Sub-task 2.2.1.3: Verify các FK GIỮ (xuyên schema trỏ tới `users`, ~30 FK) vẫn enforce sau migration: vd xoá `users` → cascade delete ở các bảng liên quan; chèn bản ghi vi phạm `user_id` không tồn tại → bị chặn bởi FK.

#### Task 2.2.2 — Kiểm thử toàn diện

- Sub-task 2.2.2.1: Chạy `make check` (fmt + tidy + generate + vet + test + build) pass.
- Sub-task 2.2.2.2: Chạy các luồng nghiệp vụ chính: worker processing, AI recommendation (vector search), dashboard, payment (PayOS webhook), loyalty.
- Sub-task 2.2.2.3: Verify HNSW index vẫn dùng trong vector search (EXPLAIN).
- Sub-task 2.2.2.4: Cập nhật `database/scripts/closy_db-schema.sql` dump + `architecture.md` §VII theo schema mới (schema-qualified).

**Checkpoint 2.2**: SC-005, SC-006 đạt; round-trip migration không mất dữ liệu.

### PHASE 2.3 — Polish & Cross-Cutting Concerns

- Sub-task 2.3.1: Chạy quickstart.md validation (Sprint 1 & 2) — kiểm tra HNSW index, outbox event, migration round-trip, schema mapping, search_path.
- Sub-task 2.3.2: Rà soát toàn bộ: không còn hardcode domain constant, không còn import module khác trong use case/repo, tài liệu `docs/` không còn thông tin lỗi thời về schema.

**Checkpoint 2.3**: Toàn bộ checklist feature đạt; sẵn sàng integration & PR.

---

## Dependencies & Execution Order

```
SPRINT 1
├── PHASE 1.1 (đánh giá) ── tiền đề cho mọi thứ
│     ├── Task 1.1.1 ── Task 1.1.2 ── Task 1.1.3 (có thể song song)
│     └── Task 1.1.4 (phụ thuộc 1.1.1-1.1.3)
├── PHASE 1.2 (wardrobe↔fashion) ── phụ thuộc Task 1.1.4
│     ├── Task 1.2.1 ── Task 1.2.2 (kế tiếp)
│     ├── Task 1.2.3 (độc lập, song song với 1.2.1/1.2.2)
│     ├── Task 1.2.4 (phụ thuộc 1.2.1/1.2.2)
│     └── Task 1.2.5 (song song)
└── PHASE 1.3 (dashboard) ── độc lập với PHASE 1.2, có thể song song
      ├── Task 1.3.1 ── Task 1.3.2 ── Task 1.3.3 ── Task 1.3.4 (song song nhau)
      ├── Task 1.3.5 (phụ thuộc 1.3.1-1.3.4)
      └── Task 1.3.6

SPRINT 2 (chỉ khi Sprint 1 xong — SC-004 đạt)
├── PHASE 2.1
│     ├── Task 2.1.1 ── Task 2.1.2 ── Task 2.1.3
└── PHASE 2.2
      ├── Task 2.2.1
      └── Task 2.2.2
      └── PHASE 2.3 (Polish) — sau PHASE 2.2
```

### Execution Strategy

1. **Sprint 1 trước**: PHASE 1.1 làm trước để chốt phạm vi chính xác. PHASE 1.2 và 1.3 có thể chạy song song (tách module khác nhau).
2. **Sprint 2 chỉ bắt đầu** khi SC-004 đạt (0 cross-module DB access) — vì tách schema sẽ làm raw cross-module query fail ngay.
3. Chạy `make test` sau mỗi Phase; `make check` sau mỗi Sprint.

## Artifacts được sinh trong plan này

- `specs/011-assess-schema-separation/research.md` — Phase 0 output
- `specs/011-assess-schema-separation/data-model.md` — schema mapping theo module
- `specs/011-assess-schema-separation/contracts/` — contract design
- `specs/011-assess-schema-separation/quickstart.md` — validation guide
