# Tasks: Củng cố Ranh Giới Module — Endpoint Ownership, Auth Middleware & Swagger Per-Module

**Input**: Design documents từ `/specs/010-module-ownership-refactor/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, contracts/ (auth_middleware, swagger-module-mapping)

**Tests**: Không có test tasks riêng — các acceptance scenario được kiểm chứng qua `make check` / smoke test thủ công (xem quickstart.md).

**Organization**: Tasks được nhóm theo user story để thực hiện & kiểm thử độc lập từng story.

## Mapping User Story (từ spec.md)

| Story label | Spec story | Đợt |
|---|---|---|
| [US0-1] | US0.1 — Middleware dùng chung về shared/presentation (P1) | Đợt 0 |
| [US0-2] | US0.2 — Auth middleware thuộc module identity qua contract (P1) | Đợt 0 |
| [US1-1] | US1.1 — Mỗi module tự đăng ký endpoint qua presentation/router (P1) | Đợt 1 |
| [US1-2] | US1.2 — AppRouter chỉ còn ghép engine (P1) | Đợt 1 |
| [US2-1] | US2.1 — Gán tag swagger theo module (P2) | Đợt 2 |
| [US2-2] | US2.2 — Sinh swagger.json per-module + dropdown (P2) | Đợt 2 |

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Chạy song song (khác file, không phụ thuộc)
- **[Story]**: User story thuộc về
- Đường dẫn file chính xác trong mô tả

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Không có project initialization mới — đây là refactor code thuần, môi trường (Go, Wire, Swag, Make) đã sẵn sàng. Cài tools nếu máy chưa có.

- [x] T001 Cài tools phát triển nếu thiếu: `make install-tools` (Wire + Swag) và `make install-goose`

**Checkpoint**: Môi trường sẵn sàng.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Trong lúc refactor, một route-registry test (FR-017) đã được tạo với golden baseline chụp từ code hiện tại để bảo vệ 150 routes khi di chuyển qua module. Sau khi feature hoàn tất, test + golden + Makefile targets liên quan bị **gỡ bỏ** (xem spec Clarifications và research R1) — route được khai báo tĩnh, swagger.json + `generate-check` là route inventory đủ.

> Ghi chú lịch sử: T002–T004 (tạo routes_test.go + golden baseline) đã được thực thi trong lúc refactor để xác nhận giữ nguyên 100% route, sau đó các file bị xóa.

---

## Phase 3: User Story 0-1 — Middleware dùng chung về shared/presentation (Priority: P1) 🎯 MVP

**Goal**: 6 middleware hạ tầng (cors, timeout, ratelimit, role, swagger_lock, error_handler) chuyển về `internal/shared/presentation/middleware/`, không phụ thuộc business module; xóa `internal/api/middleware`.

**Independent Test**: `make build` + `make test` pass; `internal/api/middleware` đã xóa; không package nào trong `internal/shared/presentation/middleware` import `internal/modules/`.

### Implementation for User Story 0-1

- [x] T005 [P] [US0-1] Tạo `internal/shared/presentation/middleware/cors.go` (di chuyển nguyên văn từ `internal/api/middleware/cors.go`)
- [x] T006 [P] [US0-1] Tạo `internal/shared/presentation/middleware/timeout.go` (di chuyển nguyên văn từ `internal/api/middleware/timeout.go`)
- [x] T007 [P] [US0-1] Tạo `internal/shared/presentation/middleware/ratelimit.go` (di chuyển nguyên văn từ `internal/api/middleware/ratelimit.go`)
- [x] T008 [P] [US0-1] Tạo `internal/shared/presentation/middleware/role.go` (di chuyển nguyên văn từ `internal/api/middleware/role.go`)
- [x] T009 [P] [US0-1] Tạo `internal/shared/presentation/middleware/swagger_lock.go` (di chuyển nguyên văn từ `internal/api/middleware/swagger_lock.go`)
- [x] T010 [P] [US0-1] Tạo `internal/shared/presentation/middleware/error_handler.go` (di chuyển nguyên văn từ `internal/api/middleware/error_handler.go`)
- [x] T011 [P] [US0-1] Tạo `internal/shared/presentation/middleware/error_handler_test.go` (di chuyển nguyên văn từ `internal/api/middleware/error_handler_test.go`)
- [x] T012 [US0-1] Cập nhật import middleware trong `internal/api/routes/router.go` sang `internal/shared/presentation/middleware` (GlobalErrorHandler, CORSMiddleware, GlobalTimeoutMiddleware, RateLimitMiddleware)
- [x] T013 [US0-1] Cập nhật import middleware trong `internal/di/wire.go` (`middleware.NewRateLimitMiddleware`) sang `internal/shared/presentation/middleware`
- [x] T014 [US0-1] Cập nhật chuỗi noise-candidate trong `internal/pkg/utils/errorutils/errorutils.go` (lines 22-27): `internal/api/middleware.CORSMiddleware` → `internal/shared/presentation/middleware.CORSMiddleware`; `internal/api/routes.NewEngine.(*RateLimitMiddleware).Handle` → `internal/shared/presentation/middleware.(*RateLimitMiddleware).Handle`
- [x] T015 [US0-1] Cập nhật stack frame path trong `internal/pkg/utils/errorutils/errorutils_test.go` (lines 60-65, 129-132) sang path middleware mới
- [x] T016 [US0-1] Xóa các file shared đã di chuyển khỏi `internal/api/middleware` (cors.go, timeout.go, ratelimit.go, role.go, swagger_lock.go, error_handler.go, error_handler_test.go) — GIỮ LẠI `auth.go` (chưa di chuyển, sẽ xử lý ở US0-2)
- [x] T017 [US0-1] Chạy `make build` + `go test ./...` — xác nhận toàn bộ test pass (baseline không đổi)

**Checkpoint**: US0-1 xong — middleware shared hoạt động, API giữ nguyên behavior.

---

## Phase 4: User Story 0-2 — Auth middleware thuộc module identity (Priority: P1)

**Goal**: `auth.go` về `internal/modules/identity/presentation/middleware/`, expose interface `IAuthMiddleware` qua `identity/presentation/contract/`; consumer chỉ phụ thuộc interface.

**Independent Test**: `make build` + `make test` pass; router module dùng `IAuthMiddleware` không import `identity/presentation/middleware`; endpoint private vẫn 401 không JWT / 200 có JWT.

### Implementation for User Story 0-2

- [x] T018 [P] [US0-2] Tạo `internal/modules/identity/presentation/middleware/auth.go` (di chuyển nguyên văn từ `internal/api/middleware/auth.go` — giữ nguyên logic `authenticate()`, vẫn dùng `identityerrors` và `ITokenBlacklistService`)
- [x] T019 [P] [US0-2] Tạo `internal/modules/identity/presentation/contract/auth_middleware.go` — interface `IAuthMiddleware` với `Handle() gin.HandlerFunc` và `OptionalHandle() gin.HandlerFunc` (chỉ import `gin`)
- [x] T020 [US0-2] Cập nhật `internal/modules/identity/presentation/provider.go`: thêm `middleware.NewAuthMiddleware` + `wire.Bind(new(contract.IAuthMiddleware), new(*middleware.AuthMiddleware))`
- [x] T021 [US0-2] Cập nhật `internal/di/wire.go`: thay `middleware.NewAuthMiddleware` (path cũ) bằng provider identity; đảm bảo `NewRateLimitMiddleware` trỏ `shared/presentation/middleware`
- [x] T022 [US0-2] Xóa `internal/api/middleware/auth.go` (đã di chuyển sang identity ở T018) rồi xóa folder `internal/api/middleware` (đã trống) — CHỈ làm sau khi T020+T021 hoàn tất và build còn pass
- [x] T023 [US0-2] Chạy `make generate` (Wire regenerate `internal/di/wire_gen.go`) + `make build`

**Checkpoint**: US0-2 xong — AuthMiddleware thuộc identity, wire binding hoạt động.

---

## Phase 5: User Story 1-1 — Mỗi module tự đăng ký endpoint qua presentation/router (Priority: P1)

**Goal**: 6 module có `presentation/router/router.go` tự đăng ký endpoint, chỉ tham chiếu handler module mình; phân tách route cross-module; xóa 10 folder route cũ.

**Independent Test**: `internal/api/routes/<nhóm>` 10 folder đã xóa; không router module nào import handler module khác (chỉ `identity_contract.IAuthMiddleware`).

### Implementation for User Story 1-1

- [x] T024 [P] [US1-1] Tạo `internal/modules/identity/presentation/router/router.go`: nhận `identity_contract.IAuthMiddleware` + handler identity; đăng ký `/auth/*`, `/me/*`, `/admin/users*` (di chuyển từ `internal/api/routes/auth/router.go`, `me/router.go`, phần admin/users của `admin/router.go`)
- [x] T025 [P] [US1-1] Tạo `internal/modules/subscription/presentation/router/router.go`: nhận `IAuthMiddleware` + `*config.Config`; đăng ký `/subscriptions/*` (di chuyển từ `internal/api/routes/subscription/router.go`)
- [x] T026 [P] [US1-1] Tạo `internal/modules/wardrobe/presentation/router/router.go`: nhận `IAuthMiddleware` + handler wardrobe; đăng ký `/wardrobe-items/*`, `/me/wardrobe-items/*`, `/system-catalog/wardrobe-items`, `/outfits/*`, `/me/outfits/*`, `/categories`, `/admin/wardrobe-items/*`, `/admin/categories/*` (di chuyển từ `internal/api/routes/wardrobe/router.go` TRỪ nhóm `/me/dashboard`, `outfit/router.go`, `category/router.go`, phần admin của `admin/router.go`)
- [x] T027 [P] [US1-1] Tạo `internal/modules/dashboard/presentation/router/router.go`: nhận `IAuthMiddleware`; đăng ký `/dashboard/analytics`, `/me/dashboard/*`, `/brand/dashboard/*`, `/admin/dashboard/*` + drilldown (di chuyển từ `dashboard/router.go`, nhóm `/me/dashboard` của `wardrobe/router.go`, nhóm `/brand/dashboard` của `brand/router.go`, nhóm admin dashboard của `admin/router.go`)
- [x] T028 [P] [US1-1] Tạo `internal/modules/brand/presentation/router/router.go`: nhận `IAuthMiddleware`; đăng ký `/brands/*`, `/brand-items/*`, `/brand-benefits/*`, `/me/brand-loyalties/*`, `/brand-portal/*`, `/admin/brands/*` (di chuyển từ `brand/router.go` TRỪ nhóm `/brand/dashboard`)
- [x] T029 [P] [US1-1] Tạo `internal/modules/fashion/presentation/router/router.go`: nhận `IAuthMiddleware`; đăng ký `/ai/*` (di chuyển từ `fashion/router.go`)
- [x] T030 [P] [US1-1] Thêm `router.NewRouter` vào ProviderSet của từng module (`internal/modules/{identity,subscription,wardrobe,dashboard,brand,fashion}/provider.go` hoặc `identity/presentation/provider.go`)
- [x] T031 [US1-1] Giữ nguyên middleware từng nhóm route khi di chuyển (vd `/brand/dashboard` và `/admin/*` giữ đúng `RolesAuthorize`; `/brand-items/:itemId` giữ `OptionalHandle`) — verify qua diff từng route
- [x] T032 [US1-1] Xóa 10 folder `internal/api/routes/{auth,me,admin,subscription,wardrobe,outfit,category,brand,fashion,dashboard}`
- [x] T033 [US1-1] Chạy `make build` + `make test` — xác nhận 100% path/method giữ nguyên (smoke test một endpoint mỗi module); sau đó scan import: `rg "internal/modules/(?!identity/)(?:wardrobe|brand|fashion|subscription|dashboard)/presentation/handler" internal/modules/*/presentation/router --pcre2` xác nhận không router module nào import handler module khác (FR-004/SC-002); `go list ./...` thành công xác nhận không import cycle (SC-006)

**Checkpoint**: US1-1 xong — mỗi module tự quản endpoint.

---

## Phase 6: User Story 1-2 — AppRouter chỉ còn ghép engine (Priority: P1)

**Goal**: `internal/api/routes/provider.go` (AppRouter) chỉ chứa 6 router module; `router.go` (NewEngine) gọi `Init(api)` cho 6 router theo thứ tự; Wire ghép từ ProviderSet module.

**Independent Test**: `make generate` + `make build` + `make test` pass; app khởi động không panic route conflict.

### Implementation for User Story 1-2

- [x] T034 [US1-2] Viết lại `internal/api/routes/provider.go`: `AppRouter` chứa 6 field (IdentityRouter, SubscriptionRouter, WardrobeRouter, DashboardRouter, BrandRouter, FashionRouter); `RouterSet` chỉ gồm `wire.NewSet(NewEngine, wire.Struct(new(AppRouter), "*"))`
- [x] T035 [US1-2] Viết lại `internal/api/routes/router.go` (NewEngine): gọi `Init(api)` cho 6 router module theo thứ tự R7 trong `research.md` (identity → subscription → wardrobe → dashboard → brand → fashion)
- [x] T036 [US1-2] Cập nhật `internal/di/wire.go`: bỏ `routes.RouterSet` khai báo các `NewRouter` cũ; giữ `NewEngine` + `wire.Struct(new(routes.AppRouter), "*")`; router do module ProviderSet provide
- [x] T037 [US1-2] Chạy `make generate` (Wire) + `make build`
- [x] T038 [US1-2] Chạy `make test` — không panic khi dựng engine

**Checkpoint**: US1-2 xong — `internal/api/routes/` chỉ còn `router.go` + `provider.go`.

---

## Phase 7: User Story 2-1 — Gán tag swagger theo module (Priority: P2)

**Goal**: Realign `@Tags` để mỗi tag thuộc đúng 1 module; tách tag `Admin` theo module (research R3, contracts/swagger-module-mapping.md).

**Independent Test**: Quét annotation `internal/modules/` — không còn tag `Admin` dùng chung; mapping khớp `contracts/swagger-module-mapping.json`.

### Implementation for User Story 2-1

- [x] T039 [P] [US2-1] Đổi `@Tags Admin` → `@Tags Admin Identity` trong `internal/modules/identity/presentation/handler/admin_handler.go`
- [x] T040 [P] [US2-1] Đổi `@Tags Admin` → `@Tags Admin Wardrobe` trong handler admin của wardrobe (`internal/modules/wardrobe/presentation/handler/wardrobe_item_handler.go`, `category_handler.go`)
- [x] T041 [P] [US2-1] Đổi `@Tags Admin` → `@Tags Admin Brand` trong handler admin của brand (`internal/modules/brand/presentation/handler/`)
- [x] T042 [US2-1] Quét toàn bộ `internal/modules/*/presentation/handler/` — xác nhận không còn tag nào xuất hiện ở 2+ module khác nhau
- [x] T043 [US2-1] Chạy `make swagger` — regenerate master spec, xác nhận `swagger.json` phản ánh tag mới

**Checkpoint**: US2-1 xong — tag swagger map 1:1 với module.

---

## Phase 8: User Story 2-2 — Sinh swagger.json per-module + dropdown (Priority: P2)

**Goal**: Script Go tách master spec thành 6 file `api/swagger/modules/<tên>/swagger.json`; Swagger UI có dropdown chọn module.

**Independent Test**: `make swagger` sinh 6 file; mở `/swagger` thấy dropdown; tổng paths 6 spec == paths master.

### Implementation for User Story 2-2

- [x] T044 [US2-2] Tạo `internal/tools/swagger-split/main.go`: đọc master `api/swagger/swagger.json` + mapping `api/swagger/swagger-module-mapping.json` (bản runtime, copy nội dung từ `specs/010-module-ownership-refactor/contracts/swagger-module-mapping.json`); lọc `paths` theo operation tags (mọi tag operation thuộc 1 module mới tính); giữ `info`, `securityDefinitions`; ghi `api/swagger/modules/<name>/swagger.json`; fail nếu operation không map được vào module nào
- [x] T045 [US2-2] Tạo mapping runtime `api/swagger/swagger-module-mapping.json` (copy từ `specs/010-module-ownership-refactor/contracts/swagger-module-mapping.json`) rồi cập nhật `Makefile` target `swagger`: sau `swag init`, chạy `go run ./internal/tools/swagger-split -master api/swagger/swagger.json -mapping api/swagger/swagger-module-mapping.json -out api/swagger/modules`
- [x] T046 [US2-2] Cập nhật `api/swagger/index.html`: thêm vào mảng `urls` trong `SwaggerUIBundle` các entry `/api-docs/modules/<name>/swagger.json` cho 6 module (giữ `v1 - Tất cả` mặc định)
- [x] T047 [US2-2] Chạy `make swagger` — xác nhận `api/swagger/modules/` có 6 file; tổng paths 6 spec == paths master
- [x] T048 [US2-2] Chạy `make generate-check` — xác nhận CI diff bao phủ cả `api/swagger/modules/` (thư mục con của `SWAG_OUTPUT_DIR`)

**Checkpoint**: US2-2 xong — docs swagger phản ánh ranh giới module.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Cập nhật tài liệu kiến trúc, gate chất lượng cuối cùng.

- [x] T049 [P] Cập nhật `docs/system-design/architecture.md`: §II — bỏ mô tả `internal/api/middleware` (dòng "internal/api/middleware # Gin middleware"); §IV — thêm `presentation/router` và `presentation/middleware` vào canonical module layout; §V — mô tả module tự đăng ký endpoint qua `presentation/router`, `internal/api/routes` chỉ ghép engine; §III — thêm mô tả middleware dùng chung thuộc `shared/presentation/middleware`
- [x] T050 [P] Scan `docs/development/agent-workflow.md` và các file trong `docs/development/`; nếu có nhắc `internal/api/routes/<nhóm>` hoặc `internal/api/middleware` thì cập nhật path theo cấu trúc mới (presentation/router, shared/presentation/middleware)
- [x] T051 [P] Cập nhật `docs/project/changelog.md`: thêm ghi chú đổi tag swagger (`Admin` → `Admin Identity` / `Admin Wardrobe` / `Admin Brand`) — thay đổi docs, không phải API contract
- [x] T052 Chạy `make check` cuối cùng (fmt-check, tidy-check, generate-check, vet, test, build) — toàn bộ pass
- [x] T053 Chạy smoke test theo `quickstart.md`: health check, 1 endpoint mỗi module, dropdown swagger

**Checkpoint**: Feature hoàn tất, sẵn sàng merge.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Không phụ thuộc — chạy ngay
- **User Stories (Phase 3+)**: Đều phụ thuộc Setup
  - US0-1 + US0-2 (Đợt 0) phải xong TRƯỚC US1-1/US1-2 (Đợt 1) vì router module cần `IAuthMiddleware` (spec Assumptions: thứ tự đợt bắt buộc)
  - US1-1 + US1-2 (Đợt 1) phải xong trước US2-1/US2-2 (Đợt 2) — swagger tag/script độc lập về code nhưng nằm sau để giảm churn
  - US2-1 có thể song song US2-2 về mặt kỹ thuật, nhưng US2-2 phụ thuộc kết quả mapping của US2-1
- **Polish (Phase 9)**: Phụ thuộc toàn bộ user story mong muốn hoàn tất

### User Story Dependencies

- **US0-1 (P1)**: Sau Setup — độc lập
- **US0-2 (P1)**: Sau Setup — độc lập với US0-1 (khác file) nhưng cùng Đợt 0
- **US1-1 (P1)**: Phụ thuộc US0-2 (cần `IAuthMiddleware`)
- **US1-2 (P1)**: Phụ thuộc US1-1 (AppRouter chứa 6 router module)
- **US2-1 (P2)**: Phụ thuộc US1-1/US1-2 (giảm churn), độc lập về kỹ thuật
- **US2-2 (P2)**: Phụ thuộc US2-1 (mapping tag) + US1-1/US1-2

### Within Each User Story

- Di chuyển file nguyên văn → đổi import → build → test
- Story xong trước khi chuyển priority tiếp theo

### Parallel Opportunities

- **US0-1**: T005–T011 (7 file middleware) song song hoàn toàn [P]
- **US0-2**: T018, T019 song song; T020–T023 tuần tự
- **US1-1**: T024–T030 (6 router + 6 provider) song song [P]; T031–T033 tuần tự
- **US2-1**: T039–T041 song song [P]; T042–T043 tuần tự
- **Polish**: T049–T051 song song [P]

---

## Parallel Example: User Story 1-1 (di chuyển route)

```bash
# Launch 6 module routers song song (khác file):
Task: "Tạo identity presentation/router/router.go"
Task: "Tạo subscription presentation/router/router.go"
Task: "Tạo wardrobe presentation/router/router.go"
Task: "Tạo dashboard presentation/router/router.go"
Task: "Tạo brand presentation/router/router.go"
Task: "Tạo fashion presentation/router/router.go"

# Sau khi 6 router + provider xong, tuần tự:
Task: "Xóa 10 folder route cũ"
Task: "make build + make test"
```

---

## Implementation Strategy

### MVP First (User Story 0-1 Only — Đợt 0 phần shared middleware)

1. Phase 1: Setup
2. Phase 3: US0-1 — shared middleware
3. **STOP và VALIDATE**: `make build` + `make test`; `internal/api/middleware` còn lại chỉ `auth.go` (chờ US0-2)
4. Deploy/demo nếu cần

### Incremental Delivery

1. Đợt 0 (US0-1 + US0-2): Tách middleware — sau mỗi story test độc lập → demo
2. Đợt 1 (US1-1 + US1-2): Module tự đăng ký endpoint → smoke test + `make test` là cổng chặn breaking change → demo
3. Đợt 2 (US2-1 + US2-2): Swagger per-module → demo dropdown
4. Mỗi đợt giữ nguyên API, không phá story trước

### Parallel Team Strategy

1. Setup cùng nhau
2. Sau Setup: Dev A → US0-1, Dev B → US0-2 (song song Đợt 0)
3. Hết Đợt 0: Dev A→US1-1, Dev B→US1-2 (song song Đợt 1)
4. Hết Đợt 1: Dev A→US2-1, Dev B→US2-2 (song song Đợt 2)

---

## Notes

- [P] tasks = khác file, không phụ thuộc
- [Story] label map task tới user story cho traceability
- Mỗi user story độc lập hoàn tất và kiểm thử được
- Commit sau mỗi task hoặc nhóm logic
- Tránh: task mơ hồ, conflict cùng file, cross-story dependency phá vỡ independence
