# Feature Specification: Củng cố Ranh Giới Module — Endpoint Ownership, Auth Middleware & Swagger Per-Module

**Feature Branch**: `010-module-ownership-refactor`

**Created**: 2026-08-01 | **Last Updated**: 2026-08-01

**Status**: Draft

**Input**: User description: "Thảo luận, hiện dự án đang theo modular monolith nhưng db vẫn chưa chia schema, và mỗi module vẫn chưa tự định nghĩa endpoint của nó. Chọn hướng: đưa endpoint vào module (6 module tự đăng ký route qua presentation/router), đưa middleware vào shared/presentation (auth middleware thuộc module identity qua contract), và sinh swagger per-module. Chia làm 3 đợt: Đợt 0 tách middleware, Đợt 1 đưa endpoint vào module, Đợt 2 swagger per-module."

## User Scenarios & Testing

### Đợt 0 — Tách Middleware theo tầng (Priority: P1)

**Why this priority**: Middleware đang nằm trong `internal/api/middleware` — tầng API, trong khi module (wardrobe, brand, fashion...) cần dùng `AuthMiddleware` khi tự đăng ký route ở Đợt 1. Nếu không tách trước, module sẽ phụ thuộc ngược vào tầng API (internal/api). Đây là điều kiện tiên quyết cho Đợt 1.

#### User Story 0.1 — Middleware dùng chung về shared/presentation (Priority: P1)

Các middleware mang tính hạ tầng HTTP dùng chung (CORS, timeout, rate-limit, role authorize, swagger lock, global error handler) chuyển về `internal/shared/presentation/middleware/`. Middleware này KHÔNG được phụ thuộc vào bất kỳ business module nào — chỉ dùng các gói shared/pkg/config.

**Independent Test**: Biên dịch và chạy toàn bộ API; mọi endpoint hiện có vẫn hoạt động với đúng middleware như trước (CORS, timeout, rate-limit, role, swagger lock, error handler). Kiểm tra import: không có package nào trong shared middleware import `internal/modules/`.

**Acceptance Scenarios**:

1. **Given** project đang biên dịch với middleware ở vị trí cũ, **When** chuyển middleware dùng chung sang `internal/shared/presentation/middleware/`, **Then** project biên dịch thành công và tất cả endpoint giữ nguyên behavior (CORS header, timeout 504, rate-limit 429, role 403, swagger lock, error JSON)
2. **Given** shared middleware mới, **When** kiểm tra dependencies, **Then** không có import nào trỏ tới `internal/modules/`
3. **Given** folder `internal/api/middleware` đã trống, **When** kiểm tra repository, **Then** folder đó đã bị xóa

#### User Story 0.2 — Auth middleware thuộc module identity, expose qua contract (Priority: P1)

AuthMiddleware là trách nhiệm của module identity (dùng trực tiếp `identityerrors` và `ITokenBlacklistService`). Nó nằm tại `internal/modules/identity/presentation/middleware/`. Các module khác (và cả NewEngine) chỉ phụ thuộc interface `IAuthMiddleware` trong `identity/presentation/contract/`, không import implementation.

**Independent Test**: Module khác (ví dụ wardrobe router) sử dụng AuthMiddleware qua interface `IAuthMiddleware` — không import trực tiếp `identity/presentation/middleware`. Toàn bộ endpoint private vẫn require JWT hợp lệ như hiện tại.

**Acceptance Scenarios**:

1. **Given** AuthMiddleware chuyển vào module identity, **When** kiểm tra ai đang dùng nó, **Then** các consumer (router module, NewEngine) chỉ phụ thuộc `identity/presentation/contract.IAuthMiddleware`, không import file `auth.go` trực tiếp
2. **Given** request có JWT hợp lệ, **When** gọi endpoint private bất kỳ, **Then** được chấp nhận (200) — hành vi xác thực giữ nguyên 100%
3. **Given** request không có JWT, **When** gọi endpoint private, **Then** trả 401 với message tiếng Việt như trước
4. **Given** user bị blacklist, **When** gọi endpoint private với JWT hợp lệ, **Then** trả 401 với message "Tài khoản đã bị khoá hoặc vô hiệu hoá. Vui lòng liên hệ CSKH."

---

### Đợt 1 — Module tự định nghĩa endpoint (Priority: P1)

**Why this priority**: Đây là mục tiêu chính của feature. Hiện tất cả route tập trung trong `internal/api/routes/<nhóm>/` (10 folder: auth, me, admin, subscription, wardrobe, outfit, category, brand, fashion, dashboard), trong đó router `admin`, `wardrobe`, `brand` import handler của NHIỀU module khác — vi phạm ranh giới module. Chuyển route về từng module giúp mỗi module tự quản surface area của nó và nhất quán với cơ chế archive module.

#### User Story 1.1 — Mỗi module tự đăng ký endpoint qua presentation/router (Priority: P1)

Mỗi module business có `presentation/router/` tự đăng ký endpoint của mình, chỉ tham chiếu handler module đó:
- **identity**: `/auth/*`, `/me/*`, `/admin/users*`
- **subscription**: `/subscriptions/*`
- **wardrobe**: `/wardrobe-items/*`, `/me/wardrobe-items/*`, `/system-catalog/*`, `/outfits/*`, `/me/outfits/*`, `/categories`, `/admin/wardrobe-items/*`, `/admin/categories/*`
- **dashboard**: `/dashboard/analytics`, `/me/dashboard/*`, `/brand/dashboard/*`, `/admin/dashboard/*`, drilldown
- **brand**: `/brands/*`, `/brand-items/*`, `/brand-benefits/*`, `/me/brand-loyalties/*`, `/brand-portal/*`, `/admin/brands/*`
- **fashion**: `/ai/*`

Các route cross-module đang trộn lẫn (router `admin` đang trộn identity+dashboard+wardrobe; router `brand` chứa `/brand/dashboard` của dashboard; router `wardrobe` chứa `/me/dashboard` của dashboard) phải được phân về đúng module sở hữu handler.

**Independent Test**: `internal/api/routes/` chỉ còn `router.go` (NewEngine) và `provider.go` (AppRouter). 10 folder route cũ bị xóa. Kiểm tra import: không router module nào import handler module khác.

**Acceptance Scenarios**:

1. **Given** toàn bộ endpoint hiện tại (130 paths trong swagger), **When** hoàn tất refactor, **Then** 100% endpoint giữ nguyên path, method, middleware (auth/role) và response — không có breaking change
2. **Given** router `admin` cũ, **When** chia tách, **Then** `/admin/users*` thuộc identity, `/admin/dashboard*` thuộc dashboard, `/admin/wardrobe-items*` + `/admin/categories*` thuộc wardrobe
3. **Given** router `brand` cũ, **When** tách, **Then** nhóm `/brand/dashboard/*` chuyển sang router dashboard module, phần còn lại giữ nguyên tại brand
4. **Given** router `wardrobe` cũ, **When** tách, **Then** nhóm `/me/dashboard/*` chuyển sang router dashboard module
5. **Given** module router mới, **When** kiểm tra imports, **Then** mỗi router chỉ import handler/contract của chính module nó (ngoại trừ `identity_contract.IAuthMiddleware` dùng chung)
6. **Given** `internal/api/routes/<nhóm>`, **When** refactor hoàn tất, **Then** 10 folder (auth, me, admin, subscription, wardrobe, outfit, category, brand, fashion, dashboard) bị xóa

#### User Story 1.2 — AppRouter chỉ còn ghép engine (Priority: P1)

`internal/api/routes/provider.go` (AppRouter) chỉ chứa 6 router module và được Wire ghép từ ProviderSet của từng module. `router.go` (NewEngine) gọi `Init(api)` cho 6 router module theo thứ tự.

**Independent Test**: `make generate` chạy thành công (Wire sinh đúng graph); `make build` thành công.

**Acceptance Scenarios**:

1. **Given** DI đã cập nhật, **When** chạy `make generate`, **Then** Wire sinh `wire_gen.go` không lỗi
2. **Given** app khởi động, **When** gọi `/api/v1/health` và một endpoint mỗi module, **Then** tất cả phản hồi bình thường
3. **Given** app khởi động, **When** kiểm tra route registry, **Then** không có route nào bị panic do conflict path

---

### Đợt 2 — Swagger per-module (Priority: P2)

**Why this priority**: Giá trị đem lại là trải nghiệm tra cứu API tốt hơn (docs phản ánh ranh giới module), nhưng không block deployment. Không ảnh hưởng tới runtime — là cải tiến tài liệu kỹ thuật.

#### User Story 2.1 — Gán tag swagger theo module (Priority: P2)

Realign `@Tags` trên handler để nhóm endpoint theo module (thay vì tag bề mặt như `Admin`, `Me` đang rải rác ở nhiều module). Giữ các tag nhóm hiện có trong từng module (ví dụ module brand vẫn có `Brand`, `Brand Portal`, `Brand Chat`...), nhưng đảm bảo **mỗi tag thuộc đúng một module** — không còn tag chung dùng ở nhiều module. Mapping tag→module được dùng để split.

**Independent Test**: Quét annotation trong `internal/modules/` — mỗi handler chỉ có tag thuộc module nó, không còn tag chung (`Admin`, `Me`) xuất hiện ở 3 module khác nhau.

**Acceptance Scenarios**:

1. **Given** handler trong module wardrobe, **When** kiểm tra annotation, **Then** mọi tag của handler thuộc module wardrobe (không còn tag `Admin` dùng chung cho 3 module)
2. **Given** spec full `swagger.json`, **When** nhóm paths theo mapping tag→module, **Then** mỗi path chỉ gom vào đúng một module; tổng paths của 6 spec module bằng đúng số paths master
3. **Given** một tag nhóm đang thuộc module A, **When** kiểm tra, **Then** tag đó không xuất hiện ở handler module B

#### User Story 2.2 — Sinh swagger.json cho từng module + dropdown (Priority: P2)

Giữ 1 lần sinh master spec; script tách (split) `paths` theo mapping tag→module để tạo `api/swagger/modules/<tên>.json` cho từng module. UI Swagger có dropdown cạnh version selector để chọn xem spec full hoặc spec từng module.

**Independent Test**: Chạy `make swagger` — thư mục `api/swagger/modules/` có file JSON cho 6 module; mở `/swagger` thấy dropdown chọn module; chọn module bất kỳ chỉ hiển thị endpoint module đó.

**Acceptance Scenarios**:

1. **Given** chạy `make swagger`, **When** kiểm tra output, **Then** `api/swagger/modules/` có 6 file `swagger.json` (identity, subscription, wardrobe, dashboard, brand, fashion)
2. **Given** mở Swagger UI, **When** chọn module từ dropdown, **Then** chỉ hiển thị endpoint của module đó
3. **Given** mở Swagger UI, **When** chọn spec full (mặc định), **Then** hiển thị đầy đủ 130 paths như trước
4. **Given** chạy `make generate-check` (CI), **When** có thay đổi annotation, **Then** file swagger master và file per-module đều được coi là thay đổi (diff được kiểm tra)

---

### Edge Cases

- **Order của route registration**: Nếu NewEngine gọi `Init(api)` theo thứ tự không đúng giữa các module, có thể gây conflict wildcard path (ví dụ `/:id` trùng static segment). Phải xác minh không có panic route conflict khi khởi động.
- **Middleware áp dụng trong route di chuyển**: Khi chuyển nhóm route sang module khác (vd `/brand/dashboard` từ brand → dashboard), middleware `RolesAuthorize` của nhóm đó phải được giữ nguyên y hệt.
- **Contract `IAuthMiddleware` khi module identity bị archive**: Nếu identity bị archive khỏi runtime, `IAuthMiddleware` cũng biến mất → các module khác mất khả năng xác thực. (Chấp nhận được vì archive là quyết định thủ công toàn phần.)
- **Import cycle**: Sau khi tách, `shared/presentation/middleware` và `identity/presentation/contract` phải không tạo import cycle — Go sẽ fail build nếu có, nhưng cần verify rõ.
- **Swagger tag đổi tên**: Frontend/consumer đang dựa vào tag cũ trong docs sẽ thấy tag thay đổi — cần thông báo; đây chỉ là tài liệu không phải API contract.
- **Script split khi mapping tag sai**: Nếu một endpoint có tag không khớp mapping, endpoint đó phải xuất hiện trong spec full nhưng có thể lọt khỏi mọi spec module — cần kiểm tra tổng số paths không giảm.

## Requirements

### Functional Requirements

- **FR-001**: Hệ thống MUST đưa các middleware dùng chung (CORS, timeout, rate-limit, role, swagger lock, global error) vào `internal/shared/presentation/middleware/`, không phụ thuộc bất kỳ business module nào.
- **FR-002**: Hệ thống MUST đặt AuthMiddleware trong `internal/modules/identity/presentation/middleware/`.
- **FR-003**: Hệ thống MUST expose interface `IAuthMiddleware` (gồm `Handle()` và `OptionalHandle()`) qua `internal/modules/identity/presentation/contract/`.
- **FR-004**: Mọi consumer của AuthMiddleware (router các module, NewEngine) MUST chỉ phụ thuộc `IAuthMiddleware`, không import implementation.
- **FR-005**: Hệ thống MUST xóa hoàn toàn `internal/api/middleware`.
- **FR-006**: Mỗi module business (identity, subscription, wardrobe, dashboard, brand, fashion) MUST tự đăng ký endpoint qua `presentation/router/`, chỉ tham chiếu handler của chính module đó.
- **FR-007**: Toàn bộ endpoint hiện tại MUST giữ nguyên path, method, middleware (auth/role) và hành vi — không breaking change.
- **FR-008**: Router `admin` cũ MUST được phân tách: users→identity, dashboard→dashboard, wardrobe-items/categories→wardrobe.
- **FR-009**: Nhóm route cross-module (`/brand/dashboard/*`, `/me/dashboard/*`) MUST chuyển về router dashboard module.
- **FR-010**: `internal/api/routes/` MUST chỉ còn `router.go` (NewEngine) và `provider.go` (AppRouter) sau Đợt 1; 10 folder route cũ MUST bị xóa.
- **FR-011**: `AppRouter` MUST chỉ chứa 6 router module và được Wire ghép từ ProviderSet của từng module.
- **FR-012**: Hệ thống MUST realign `@Tags` swagger trên handler để mỗi tag chỉ thuộc đúng một module (giữ các tag nhóm hiện có trong module; không còn tag chung dùng ở nhiều module). Mapping tag→module dùng cho script split.
- **FR-013**: Hệ thống MUST sinh `api/swagger/modules/<tên>.json` cho 6 module từ master spec qua script split (giữ 1 lần `swag init`).
- **FR-014**: Swagger UI MUST có dropdown (cạnh version selector) cho phép chọn spec full hoặc spec từng module.
- **FR-015**: Cơ chế `make swagger` và `make generate-check` (CI diff) MUST bao phủ cả master spec và spec per-module.
- **FR-016**: Không tách database schema theo module trong feature này (chỉ code-level module ownership) — giữ nguyên DB dùng chung.

### Key Entities

- **Module business**: identity, subscription, wardrobe, dashboard, brand, fashion — mỗi module sở hữu `presentation/router/` (đăng ký endpoint) và `presentation/contract/` (expose service/auth interface).
- **`internal/api/routes`**: Tầng API chỉ còn vai trò ghép Gin engine (NewEngine) và AppRouter (composition root cho các router module).
- **`internal/shared/presentation/middleware`**: Nơi ở mới của middleware hạ tầng dùng chung, không phụ thuộc business module.
- **`identity/presentation/middleware` + `identity/presentation/contract`**: AuthMiddleware thuộc identity; contract `IAuthMiddleware` là cửa sổ cho module khác dùng.
- **`api/swagger/modules/*.json`**: Spec swagger tách theo module, sinh từ master spec qua script split.

## Success Criteria

### Measurable Outcomes

- **SC-001**: 100% endpoint hiện tại (130 paths) giữ nguyên path, method, middleware và response sau Đợt 0 và Đợt 1 — không có breaking change với client.
- **SC-002**: 100% router module không import handler/implementation của module khác (chỉ import `identity_contract.IAuthMiddleware` dùng chung) sau Đợt 1.
- **SC-003**: `internal/api/middleware` và 10 folder `internal/api/routes/<nhóm>` bị xóa hoàn toàn, `internal/api/routes/` chỉ còn 2 file.
- **SC-004**: Dự án biên dịch sạch và vượt qua toàn bộ `make check` (fmt, tidy, generate-check, vet, test, build) sau mỗi đợt.
- **SC-005**: `api/swagger/modules/` có đủ 6 spec module; dropdown Swagger chọn được từng module và spec full vẫn đủ 130 paths.
- **SC-006**: Không có import cycle nào được giới thiệu (kiểm tra bằng build thành công và không có chuỗi import shared→module).

## Assumptions

- Không tách database schema theo module trong feature này — DB vẫn dùng chung một database/schema; chỉ refactor ở tầng code (endpoint ownership, middleware, swagger). Việc chia schema là over-engineering ở giai đoạn 1 binary đơn.
- Thứ tự thực hiện bắt buộc: Đợt 0 → Đợt 1 → Đợt 2. Đợt 0 là tiên quyết cho Đợt 1 (module cần dùng AuthMiddleware qua contract).
- Auth middleware được chọn theo "Pattern B" (identity-owned + contract), không phải shared DI inversion — để giữ nguyên `identityerrors` và `ITokenBlacklistService` không cần sửa.
- Việc đưa 6 file middleware phi-auth (cors/timeout/ratelimit/role/swagger_lock/error_handler) vào `shared/presentation/middleware` là phạm vi Đợt 0, chỉ riêng `auth.go` đi vào identity.
- Swagger realign tag theo Q1 (Option B): giữ các tag nhóm hiện có (ví dụ module brand có `Brand`, `Brand Portal`, `Brand Chat`...), chỉ đảm bảo mỗi tag thuộc đúng một module. Không gom về 1 tag/module để không mất phân nhóm docs; script split dùng mapping tag→module.
- `make generate` (wire + swagger) là cơ chế chuẩn để tái sinh code generated; `make check` là cổng chất lượng bắt buộc.
- Swagger UI hiện tại (`api/swagger/index.html`) dùng `SwaggerUIBundle({ urls: [...] })` — hỗ trợ sẵn dropdown multi-spec, không cần đổi thư viện.

## Clarifications

### Session 2026-08-01

- Q: Mức độ gom tag swagger theo module (Đợt 2) → A: Option B — giữ các tag nhóm hiện có trong từng module, chỉ đảm bảo mỗi tag thuộc đúng một module; dùng mapping tag→module cho script split (không gom về 1 tag/module).
- Q: Cơ chế kiểm chứng "không breaking change" sau refactor → A: Option A — thêm automated route-registry test (snapshot so sánh tập route `/api/v1/*` trước/sau refactor), chạy trong `make test`.
- Q: Có giữ route-registry test sau feature không? → A: Xóa hoàn toàn — giá trị chính (bảo vệ refactor) đã hoàn thành; route được khai báo tĩnh trong code nên rủi ro mất route thấp; swagger.json + `generate-check` là route inventory đủ. (Xóa `routes_test.go`, golden `testdata/routes_baseline.txt`, và các Makefile target liên quan.)
