# Tasks: Brand RBAC Tập Trung & Bổ Sung Nghiệp Vụ Quản Trị Admin

**Input**: Design documents từ `/specs/012-brand-rbac-admin/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/, quickstart.md

**Tests**: Spec không yêu cầu TDD; các task test chỉ gồm cập nhật test hiện có để `make test` pass + test bảo mật trọng điểm (IDOR, middleware, cache).

**Organization**: Tasks nhóm theo user story để implement/test độc lập từng story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Chạy song song được (khác file, không phụ thuộc task chưa xong)
- **[Story]**: User story thuộc (US1–US10)
- Kèm file path cụ thể.

## Path Conventions

- **Backend (modular monolith)**: `internal/modules/<module>/...`, `internal/shared/...`, `internal/pkg/...`
- **DB**: `database/migrations/`
- **Docs**: `docs/`
- **Sinh code**: `make wire`, `make swagger`, `make check`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Khởi tạo hạ tầng DB và chuẩn chung cho feature.

- [X] T001 Tạo **1 migration duy nhất** `database/migrations/20260817090000_add_session_tracking_and_admin_audit_logs.sql` gộp 2 thay đổi DB (đầy đủ `-- +goose Up` / `-- +goose Down`): (a) thêm cột `device_name varchar(255)`, `device_ip varchar(45)`, `user_agent varchar(512)`, `last_active_at timestamptz` vào `identity.refresh_tokens`; (b) tạo bảng `shared.admin_audit_logs` (id uuid PK, actor_user_id uuid NOT NULL FK→identity.users, action varchar(100), target_type varchar(50), target_id uuid NULL, payload jsonb default '{}', request_ip varchar(45) NULL, created_at timestamptz default now()) + 3 index (actor_user_id), (target_type,target_id), (created_at)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Nền tảng Brand RBAC dùng chung cho US1 và US2 — context keys, cache membership, sửa `GetUserBrandRole`, middleware brand. KHÔNG được bắt đầu US1/US2 khi phase này chưa xong.

- [X] T002 [P] Thêm hằng số `CtxBrandId = "brandId"`, `CtxBrandRole = "brandRole"` vào `internal/pkg/utils/contextutils/key.go`
- [X] T003 [P] Thêm getter `GetBrandID(c) (uuid.UUID, error)` và `GetBrandRole(c) (brandmemberrole.BrandMemberRole, error)` vào `internal/pkg/utils/contextutils/user.go`
- [X] T004 [P] Tạo interface `IBrandMembershipCache` trong `internal/modules/brand/application/interface/caching/brand_membership_cache.go` (Get/Set/InvalidateByUserID/InvalidateByBrandID; Get trả nil khi miss/Redis lỗi)
- [X] T005 Tạo impl `RedisBrandMembershipCache` trong `internal/modules/brand/infrastructure/caching/redis_brand_membership_cache.go` (key `brand:members:{userID}`, JSON map brandID→role, TTL từ config `BrandMembershipCacheTTL` default 5 phút, nil-safe khi redisClient nil) + thêm field config TTL trong `internal/config/config.go` + mapping env trong `configs/config.yaml`
- [X] T006 Sửa `GetUserBrandRole` trong `internal/modules/brand/application/usecase/brand/usecase.go` → read-through cache (miss thì `memberRepo.GetActiveByUserID` + filter brand active → set cache); trả `branderrors.ErrBrandPortalForbidden` khi không phải member active / brand không active (KHÔNG còn trả `("staff", nil)`); thêm field `membershipCache` vào struct `BrandUseCase` và tham số vào `NewBrandUseCase`
- [X] T007 Thêm vào `IBrandUseCase` (`internal/modules/brand/application/interface/usecase/brand_uc.go`) 2 method invalidation: `InvalidateUserMembershipCache(ctx, userID) error`, `InvalidateBrandMembersCache(ctx, brandID) error` + impl trong `usecase.go`
- [X] T008 Tạo `BrandRoleMiddleware` trong `internal/modules/brand/presentation/middleware/brand_role.go` — `PathAuthorize(brandIDParam, allowedRoles...)`, `QueryAuthorize(brandIDParam, allowedRoles...)`, functional option `WithAllowAdmin()`; flow: GetUserId → resolve brandID (path/query) → admin+allowAdmin ⇒ set context; ngược lại `GetUserBrandRole` → role không thuộc allowedRoles ⇒ `apperror.NewForbidden("Bạn không có quyền truy cập nhãn hàng này.")` → set `CtxBrandId`/`CtxBrandRole`
- [X] T009 Tạo contract `IBrandRoleMiddleware` trong `internal/modules/brand/presentation/contract/brand_role_middleware.go` (2 method trả `gin.HandlerFunc`, option `WithAllowAdmin`)
- [X] T010 Cập nhật `internal/modules/brand/provider.go`: thêm `caching.NewRedisBrandMembershipCache`, `middleware.NewBrandRoleMiddleware`, `wire.Bind(contract.IBrandRoleMiddleware, *middleware.BrandRoleMiddleware)`

**Checkpoint**: `make wire` + `make build` pass; middleware/cache/GetUserBrandRole sẵn sàng cho US1 và US2.

---

## Phase 3: User Story 1 - Khóa lỗ hổng IDOR Brand Dashboard (Priority: P1) 🎯 MVP

**Goal**: Chỉ member active (owner/staff) hoặc admin hệ thống mới xem được dữ liệu `/brand/dashboard/*`; chặn truy vấn dynamic query target=brand cho người không phải member.

**Independent Test**: User thường (không phải member brand A) gọi `GET /brand/dashboard/kpi-cards?brandId=A` → 403; staff/owner A → 200; admin → 200 (bypass); `POST /dashboard/analytics/query` target=brand với user thường → 403.

- [X] T011 [US1] Sửa `internal/modules/dashboard/presentation/router/router.go` + `internal/modules/dashboard/provider.go`: inject `contract.IBrandRoleMiddleware` vào dashboard router; gắn `QueryAuthorize("brandId", Owner, Staff, WithAllowAdmin())` lên group `/brand/dashboard` và `/brand/dashboard/drilldown`
- [X] T012 [US1] Sửa check target=brand trong `internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go` (dòng ~58-68): dựa vào error từ `GetUserBrandRole` (err != nil ⇒ forbidden), bỏ logic phụ thuộc `role==""`; đảm bảo user không member ⇒ `apperror.NewForbidden`
- [X] T013 [US1] Sửa `GetPointsLiability` trong `internal/modules/dashboard/application/usecase/brand_dashboard_uc_impl.go`: đọc `brandRole` từ context (`contextutils.GetBrandRole`) thay vì gọi `brandReader.GetUserBrandRole` lại; giữ `mapper.RedactPointsLiabilityForRole`
- [X] T014 [US1] Bổ sung/update test trong `internal/modules/dashboard/application/usecase/dynamic_query_uc_test.go` và `brand_dashboard_uc_test.go`: ca không member ⇒ 403 (dynamic query), ca member ⇒ 200; cập nhật mock `GetUserBrandRole` để trả error khi không member

**Checkpoint**: Chạy quickstart mục A — IDOR bị chặn, member/admin truy cập đúng.

---

## Phase 4: User Story 2 - Brand RBAC về middleware + cache membership (Priority: P1)

**Goal**: Toàn bộ route portal check quyền qua middleware (2 nhóm member/owner); gỡ check khỏi usecase; invalidation cache khi membership/brand đổi.

**Independent Test**: Staff gọi `PUT /loyalty/program` → 403; owner → 200; không member → 403; Redis có key `brand:members:{userId}`; gỡ/đổi role member → quyền phản ánh ngay.

- [X] T015 [P] [US2] Thêm gọi `InvalidateUserMembershipCache` cho từng user bị ảnh hưởng trong `AddBrandMembers` (`internal/modules/brand/application/usecase/brand/usecase.go`)
- [X] T016 [P] [US2] Thêm gọi `InvalidateBrandMembersCache` trong luồng đổi status brand (`UpdateBrandStatus` — path admin) tại `internal/modules/brand/application/usecase/brand/usecase.go`
- [X] T017 [US2] Tổ chức lại `internal/modules/brand/presentation/router/router.go`: group `/brand-portal` áp dụng `PathAuthorize("brandId", Owner, Staff)` cho toàn bộ route có `:brandId`; nhóm owner-only `PathAuthorize("brandId", Owner)` cho `PUT /loyalty/program`, `POST /loyalty/tiers`, `PUT /loyalty/tiers/:tierId`; 3 route không có `:brandId` (`/me/brands`, `POST /brands`, `profile-images/upload-signature`) không gắn middleware
- [X] T018 [US2] Gỡ lệnh `RequireBrandRoleShared`/`RequireBrandRole` khỏi các usecase HTTP: `internal/modules/brand/application/usecase/loyalty/usecase.go`, `benefit/usecase.go`, `item/usecase.go`, `claim/usecase.go`, `chat/usecase.go`, `brand/usecase.go`
- [X] T019 [US2] Rà soát worker nội bộ (`internal/modules/brand/presentation/worker/loyalty_point_expiry_worker.go` và các worker khác): xác nhận không đi qua các method đã gỡ check; giữ helper `RequireBrandRoleShared` (helper.go) cho luồng nội bộ nếu cần
- [X] T020 [US2] Cập nhật test brand module bị ảnh hưởng (usecase test trỏ tới hành vi forbidden cũ) — chuyển kỳ vọng check về tầng middleware

**Checkpoint**: Chạy quickstart mục B — portal RBAC qua middleware, cache hoạt động, invalidation đúng.

---

## Phase 5: User Story 3 - Ma trận quyền owner/staff được tài liệu hóa (Priority: P2)

**Goal**: Ma trận quyền chính thức là nguồn duy nhất cho middleware và docs.

**Independent Test**: Đối chiếu ma trận (`docs/system-design/brand-permission-matrix.md`) với nhóm middleware ở US2 — không lệch.

- [X] T021 [US3] Tạo `docs/system-design/brand-permission-matrix.md` (copy chuẩn hóa từ `specs/012-brand-rbac-admin/contracts/brand-permission-matrix.md`): 2 vai trò owner/staff, nhóm member vs owner-only, ghi rõ admin bypass brand dashboard + portal yêu cầu membership
- [X] T022 [P] [US3] Thêm reference về ma trận trong `docs/api/constants/brand.md` (mục BrandMemberRole) và `docs/api/brand-api.md` (mục quyền portal, sửa danh sách role cũ bị lặp "staff")
- [X] T023 [P] [US3] Thêm reference về ma trận trong `docs/api/loyalty-api.md` (các endpoint owner-only: program/tiers) — thống nhất không mâu thuẫn

**Checkpoint**: Đọc ma trận xác định được nhóm quyền từng route; middleware khớp ma trận.

---

## Phase 6: User Story 4 - Admin xử lý giao dịch thanh toán bất thường (Priority: P2)

**Goal**: Endpoint admin để list/detail giao dịch + 4 thao tác resolve (retry/resolve-paid/cancelled/creation-failed); bắt buộc bằng chứng khi resolve-paid.

**Independent Test**: Admin list giao dịch; resolve-paid có bằng chứng → success + ví/đăng ký cập nhật; thiếu bằng chứng → 4xx; user thường → 403.

- [X] T024 [P] [US4] Thêm `ListAdmin(ctx, filter, page, limit)` (filter status/fromDate/toDate) vào interface `IDepositTransactionRepository` (`internal/modules/subscription/domain/repositories/deposit_transaction_repo.go`) + impl trong `internal/modules/subscription/infrastructure/persistence/`
- [X] T025 [US4] Tạo `AdminBillingHandler` trong `internal/modules/subscription/presentation/handler/admin_billing_handler.go`: GET list, GET detail (`GetByID`), POST `retry-investigation`, `resolve-paid` (bắt buộc `reference`+`paymentLinkId`, thiếu ⇒ `apperror.NewBadRequest`), `resolve-cancelled`, `resolve-creation-failed`; map `ManualPaymentResolutionInput{OrderCode, ExpectedStatus(từ DB), ActorAdminID(userID từ context), Reason, Evidence}`; gọi `IPaymentResolutionUseCase`; thêm annotation swagger `@Tags Admin Subscription`
- [X] T026 [US4] Thêm group admin routes trong `internal/modules/subscription/presentation/router/router.go`: `/admin/subscriptions/transactions*` với `auth.Handle()` + `RolesAuthorize(roleslug.Admin)`
- [X] T027 [US4] Cập nhật `internal/modules/subscription/provider.go` (module root): thêm `handler.NewAdminBillingHandler` vào ProviderSet + `make wire`

**Checkpoint**: Chạy quickstart mục C (Payment resolution).

---

## Phase 7: User Story 5 - Admin quản lý subscription & gói cước (Priority: P2)

**Goal**: Endpoint admin CRUD plan + quản lý đăng ký user (list/cancel về free/grant thủ công + đồng bộ grant AI).

**Independent Test**: Admin tạo/tắt plan → public list phản ánh; cancel user → về free + grant AI thu hồi; grant user → nhận plan + grant AI.

- [X] T028 [P] [US5] Mở rộng `ISubscriptionPlanUseCase` + impl (`internal/modules/subscription/application/usecase/plan/`): `CreatePlan`, `UpdatePlan`, `SetPlanActive` (chống trùng slug qua `GetBySlug`; inactive ⇒ không xuất hiện public)
- [X] T029 [P] [US5] Thêm `ListAdmin(ctx, filter)` vào `IUserSubscriptionRepository` (`internal/modules/subscription/domain/repositories/user_subscription_repo.go`) + impl (filter planCode/status + phân trang)
- [X] T030 [US5] Mở rộng `ISubscriptionUseCase` + impl (`internal/modules/subscription/application/usecase/subscription/`): `AdminList`, `AdminCancel` (downgrade về free qua state machine `state_machine.go`), `AdminGrant` (kích hoạt luồng `PlanChanged` hiện có để đồng bộ grant AI)
- [X] T031 [US5] Tạo `AdminSubscriptionHandler` trong `internal/modules/subscription/presentation/handler/admin_subscription_handler.go`: `GET/POST /admin/subscriptions/plans`, `PUT/PATCH /admin/subscriptions/plans/:id`, `GET /admin/subscriptions`, `POST /admin/subscriptions/:userId/cancel`, `POST /admin/subscriptions/:userId/grant`; swagger `@Tags Admin Subscription`
- [X] T032 [US5] Thêm group routes trong `internal/modules/subscription/presentation/router/router.go` (`/admin/subscriptions*`, `/admin/subscriptions/plans*`) + thêm `handler.NewAdminSubscriptionHandler` vào `internal/modules/subscription/provider.go` + `make wire`

**Checkpoint**: Chạy quickstart mục C (Plan & subscription).

---

## Phase 8: User Story 6 - Admin quản lý tài khoản người dùng (Priority: P2)

**Goal**: Endpoint admin user: detail, tạo user, đổi role (chỉ admin seed phong admin), soft-deactivate, quản lý phiên (xem/revoke) với thông tin thiết bị.

**Independent Test**: Admin xem chi tiết; tạo user → login được; đổi role user→admin bằng admin thường → 403, bằng admin seed → 200; deactivate → không login; sessions có device/IP; revoke → token hết hiệu lực.

- [X] T033 [P] [US6] Cập nhật `Login` và `RefreshToken` trong `internal/modules/identity/application/usecase/session_uc.go`: nhận + lưu `device_name`/`device_ip`/`user_agent` lúc tạo phiên, cập nhật `last_active_at` lúc refresh (handler `auth_handler.go` truyền từ header `User-Agent`/`X-Forwarded-For`/RemoteAddr)
- [X] T034 [P] [US6] Thêm method đọc phiên vào repo refresh token (`internal/modules/identity/infrastructure/persistence/refresh_token_repo_impl.go`): `ListByUserID` (kèm device fields); mở rộng entity `RefreshToken` trong `internal/modules/identity/domain/entities/user.go`
- [X] T035 [US6] Mở rộng `IUserUseCase` (`internal/modules/identity/application/interface/usecase/user_uc.go`) + impl (`user_uc.go`): `GetUserDetail`, `CreateUser` (bcrypt hash + username/email unique, role mặc định user), `UpdateUserRole` (chỉ admin seed được đổi thành admin; chặn self/admin-khác), `ListUserSessions`, `RevokeUserSessions`; tái sử dụng `UpdateUserStatus` cho deactivate
- [X] T036 [US6] Cập nhật `AdminHandler` (`internal/modules/identity/presentation/handler/admin_handler.go`): thêm handlers `GET /admin/users/:id`, `POST /admin/users`, `PATCH /admin/users/:id/role`, `DELETE /admin/users/:id`, `GET /admin/users/:id/sessions`, `POST /admin/users/:id/sessions/revoke`; swagger `@Tags Admin User`
- [X] T037 [US6] Thêm routes trong `internal/modules/identity/presentation/router/router.go` group `/admin/users` + wire handler

**Checkpoint**: Chạy quickstart mục C (User admin).

---

## Phase 9: User Story 7 - Admin quản lý thành viên nhãn hàng (Priority: P3)

**Goal**: Endpoint admin quản lý brand members (list/add staff/đổi role/status/gỡ) + transfer ownership (chọn hạ staff hoặc gỡ; chặn khi còn 1 owner).

**Independent Test**: Admin thêm staff → vào portal; transfer với `oldOwnerHandling=staff` → chủ cũ thành staff; `remove` → chủ cũ bị gỡ; 1 owner → 4xx.

- [X] T038 [P] [US7] Tạo `IBrandAdminUseCase` + impl trong `internal/modules/brand/application/usecase/admin/brand_admin_uc.go`: `ListMembers`, `AddStaff`, `UpdateMember` (role/status), `RemoveMember` (guard single-owner), `TransferOwnership` (`newOwnerUserId`, `oldOwnerHandling: staff|remove`; guard single-owner; invalidate cache 2 user liên quan)
- [X] T039 [US7] Tạo `AdminBrandHandler` trong `internal/modules/brand/presentation/handler/admin_brand_handler.go`: `GET/POST /admin/brands/:brandId/members`, `PATCH/DELETE /admin/brands/:brandId/members/:memberId`, `POST /admin/brands/:brandId/transfer-ownership`; swagger `@Tags Admin Brand`
- [X] T040 [US7] Thêm group admin routes trong `internal/modules/brand/presentation/router/router.go` (`/admin/brands/:brandId/members*`, `/admin/brands/:brandId/transfer-ownership`) với `RolesAuthorize(roleslug.Admin)` + wire handler

**Checkpoint**: Chạy quickstart mục C (Brand member admin).

---

## Phase 10: User Story 8 - Admin quản trị AI executor (Priority: P3)

**Goal**: Dựng tầng HTTP mới cho aiexecutor: endpoint admin xem/sửa pricing, xem usage theo user, xem/tạo grant.

**Independent Test**: Admin xem pricing; sửa giá → chi phí mới theo giá mới; xem usage user; tạo grant → hạn mức user phản ánh.

- [X] T041 [P] [US8] Thêm `ListAdmin(ctx)` vào `IAIModelConfigRepository` (`internal/modules/aiexecutor/domain/repositories/pricing_repo.go`) + impl; thêm `GetUsageByUser(ctx, userID, from, to)` và `ListGrantsByUser(ctx, userID)` vào `IAICostRepository` (`ai_cost_repo.go`) + impl (`internal/modules/aiexecutor/infrastructure/persistence/`)
- [X] T042 [US8] Tạo `IAIAdminUseCase` + impl trong `internal/modules/aiexecutor/application/usecase/admin/ai_admin_uc.go`: `ListPricing`, `UpdatePricing` (invalidate redis pricing cache / để TTL), `GetUserUsage`, `ListGrants`, `CreateGrant` (dùng `CreateGrant` + `ExpireGrantsByUserID` nếu cần)
- [X] T043 [US8] Tạo `AdminHandler` (`internal/modules/aiexecutor/presentation/handler/admin_handler.go`) + `Router` (`internal/modules/aiexecutor/presentation/router/router.go`): group `/admin/ai/*` (`GET /admin/ai/pricing`, `PUT /admin/ai/pricing/:id`, `GET /admin/ai/usage`, `GET /admin/ai/grants`, `POST /admin/ai/grants`) với `auth.Handle()` + `RolesAuthorize(roleslug.Admin)`; swagger `@Tags Admin AI`
- [X] T044 [US8] Cập nhật `internal/modules/aiexecutor/provider.go` (thêm handler/router/usecase) + `internal/api/routes/provider.go` (thêm `AIExecutorRouter` vào AppRouter) + `internal/api/routes/router.go` (gọi `Init`) + `internal/di/wire.go` + `make wire`

**Checkpoint**: Chạy quickstart mục C (AI executor admin).

---

## Phase 11: User Story 9 - Audit log cho hành động quản trị (Priority: P3)

**Goal**: Module `admin` mới ghi/xem audit cho mọi hành động admin — **bất đồng bộ qua outbox + RabbitMQ**; giữ 12 tháng + cleanup worker.

**Independent Test**: Hành động admin (vd đổi status user) → sau vài nhịp consumer, có bản ghi audit; `GET /admin/audit-logs` trả đúng; user thường → 403; bản ghi > 12 tháng bị dọn.

- [X] T045 [P] [US9] Tạo module `admin`: entity `internal/modules/admin/domain/entities/admin_audit_log.go`, repo interface `internal/modules/admin/domain/repositories/admin_audit_log_repo.go` (`Create`, `List(filter)`, `DeleteOlderThan`), impl gorm `internal/modules/admin/infrastructure/persistence/admin_audit_log_repo.go` (bảng `shared.admin_audit_logs`)
- [X] T046 [P] [US9] Tạo `IAuditLogUseCase` + impl `internal/modules/admin/application/usecase/audit_log_uc.go` (`Record`, `List`, `DeleteOlderThan`); interface `internal/modules/admin/application/interface/usecase/audit_log_uc.go`
- [X] T047 [P] [US9] Thêm hằng số messaging: `eventconstants.TopicAuditLog = "admin.audit.log"` trong `internal/shared/application/constants/eventconstants/eventconstants.go`; `QueueAuditLog = "audit_log_queue"` + `RoutingKeyAuditLog` trong `internal/shared/infrastructure/messaging/rabbitmq_constants.go`; bind queue trong `internal/shared/infrastructure/messaging/rabbitmq_connection.go`; tạo contract `IAuditLogContract` + impl trong `internal/modules/admin/presentation/contract/audit_log_contract.go` — `Record` = `event.IEventPublisher.Publish(ctx, TopicAuditLog, RecordAuditInput)` (ghi outbox, KHÔNG ghi DB trực tiếp)
- [X] T048 [US9] Tạo `AdminAuditHandler` + `Router` trong `internal/modules/admin/presentation/`: `GET /admin/audit-logs` (filter actorUserId/action/targetType/targetId/fromDate/toDate/page/limit) với `auth.Handle()` + `RolesAuthorize(roleslug.Admin)`; swagger `@Tags Admin Audit`
- [X] T049 [US9] Tạo consumer worker `internal/modules/admin/presentation/worker/audit_log_consumer.go` (`event.IMessageConsumer.RegisterConsumer(QueueAuditLog, handler)` → unmarshal payload → `IAuditLogUseCase.Record` → ghi bảng, ack/nack theo pattern `email_worker.go`) + cleanup worker `audit_cleanup_worker.go` (xóa `created_at < now()-12 months` định kỳ hàng ngày); đăng ký cả 2 vào bootstrap worker
- [X] T050 [US9] Tạo `internal/modules/admin/provider.go` (entity/repo/usecase/contract/handler/router/consumer/cleanup); đăng ký `admin.ProviderSet` vào `internal/di/wire.go` + `AdminRouter` vào `internal/api/routes/provider.go` + `internal/api/routes/router.go` + `make wire`
- [X] T051 [US9] Inject `IAuditLogContract` vào các admin handler đã xây (identity `admin_handler.go`, subscription `admin_billing_handler.go` + `admin_subscription_handler.go`, brand `admin_brand_handler.go`, aiexecutor `admin_handler.go`) + thêm lệnh `Record` với action/target/payload theo `contracts/admin-audit-log.md` (không chứa secret/token/password)


**Checkpoint**: Chạy quickstart mục C (Audit log).

---

## Phase 12: User Story 10 - Đồng bộ tài liệu docs-code và ghi nhận out-scope (Priority: P3)

**Goal**: Sửa mâu thuẫn docs theo chuẩn code; ghi 9 mục out-scope vào docs.

**Independent Test**: Đọc docs không còn mâu thuẫn với code (transaction_type, plan code, khái niệm role, ma trận); tài liệu out-of-scope đủ 9 mục.

- [X] T052 [P] [US10] Sửa `docs/api/dashboard-api-v2.md`: `DIRECT_PURCHASE` → `direct_purchase` (dòng 36, 31) và plan codes `FREE/PREMIUM_MONTHLY/PREMIUM_YEARLY` → khớp `free`/`premium-monthly` (dòng 13, 20)
- [X] T053 [P] [US10] Sửa `docs/system-design/architecture.md` (dòng ~287-291): đổi khái niệm "B2C Roles" → "hạng theo gói subscription (PlanKind free/premium)"; làm rõ `roleslug` chỉ có admin/user
- [X] T054 [P] [US10] Sửa `docs/api/brand-api.md` dòng 480: sửa danh sách role cũ bị gõ lặp "staff, staff, staff"; reference ma trận quyền `docs/system-design/brand-permission-matrix.md`
- [X] T055 [P] [US10] Tạo `docs/decisions/out-of-scope-2026-08.md`: liệt kê 9 mục (RBAC toàn hệ thống giữ `users.role_slug`, campaign, support ticket, virtual trial, notification đa kênh + quản lý, swap/re-roll AI, AI OOTD generator, body-shape validation, garment decay) kèm lý do
- [X] T056 [P] [US10] Cập nhật `docs/product/roadmap.md` + `docs/README.md`: trỏ tới ADR out-of-scope và ma trận quyền


**Checkpoint**: Đối chiếu docs với code theo quickstart mục D — không còn mâu thuẫn.

---

## Phase 13: Polish & Cross-Cutting Concerns

**Purpose**: Đảm bảo chất lượng tổng thể, sinh code, bảo mật và kiểm chứng end-to-end.

- [X] T057 Chạy `make wire` + `make swagger` → commit `wire_gen.go` + `api/swagger/`; chạy `make check` (fmt, tidy, generate-check, vet, test, build) và sửa mọi lỗi phát sinh
- [X] T058 [P] Bổ sung unit test `BrandRoleMiddleware` trong `internal/modules/brand/presentation/middleware/brand_role_test.go` (path/query resolve, admin bypass, role không thuộc allowedRoles ⇒ 403, context set đúng)
- [X] T059 [P] Bổ sung unit test cache membership trong `internal/modules/brand/infrastructure/caching/redis_brand_membership_cache_test.go` (get/set/invalidate/fallback nil client)
- [X] T060 [P] Bổ sung test admin user role-restriction trong `internal/modules/identity/application/usecase/user_uc_test.go` (chặn đổi role thành admin khi không phải seed; chặn self)
- [X] T061 [P] Bổ sung test `PaymentResolutionUseCase` cho ca thiếu evidence và trạng thái không khớp (`internal/modules/subscription/application/usecase/resolution/payment_resolution_uc_test.go` nếu chưa có)
- [X] T062 Chạy quickstart.md mục A–D end-to-end (curl + kiểm tra Redis key `brand:members:*`, audit rows, docs) và ghi kết quả
- [X] T063 Rà soát bảo mật cuối: không cross-module FK ngoài `admin_audit_logs.actor_user_id→identity.users`; payload audit không chứa secret; context key không hardcode; worker nền không bỏ sót check sau khi gỡ khỏi usecase
- [X] T064 [P] Bổ sung đo metric cache hit/miss cho `RedisBrandMembershipCache` (counter trong `internal/modules/brand/infrastructure/caching/redis_brand_membership_cache.go`, log/đọc được) để kiểm chứng SC-003 (cache hit ≥90% trên portal/dashboard) — và bổ sung ghi chú cách đo vào `quickstart.md`


---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Không phụ thuộc — bắt đầu ngay
- **Foundational (Phase 2)**: Phụ thuộc Setup — **BLOCK tất cả user story**
- **User Stories (Phase 3+)**:
  - US1, US2 (P1): phụ thuộc Foundational; US2 phụ thuộc US1 về contract middleware đã wire (US1 áp dụng cho dashboard, US2 cho portal) — có thể song song sau Foundational nếu đủ nhân sự
  - US3 (P2): phụ thuộc US2 (đối chiếu ma trận với nhóm router)
  - US4, US5, US6 (P2): phụ thuộc Foundational; độc lập nhau
  - US7, US8 (P3): phụ thuộc Foundational; độc lập nhau
  - US9 (P3): phụ thuộc Foundational; retrofit ghi audit vào US4–US8 (task T051) — các story US4–US8 hoạt động đầy đủ trước khi có audit
  - US10 (P3): phụ thuộc US3 (ma trận) + độc lập phần còn lại
- **Polish (Phase 13)**: Phụ thuộc tất cả user story mong muốn

### User Story Dependencies

- **US1 (P1)**: Sau Foundational; không phụ thuộc story khác
- **US2 (P1)**: Sau Foundational; tái sử dụng middleware từ Foundational
- **US3 (P2)**: Sau US2
- **US4 (P2)**: Sau Foundational; không phụ thuộc US1/US2
- **US5 (P2)**: Sau Foundational; độc lập
- **US6 (P2)**: Sau Foundational + Setup (migration session); độc lập
- **US7 (P3)**: Sau Foundational; độc lập
- **US8 (P3)**: Sau Foundational; độc lập
- **US9 (P3)**: Sau Foundational; tích hợp audit vào US4–US8 (T051)
- **US10 (P3)**: Sau US3

### Within Each User Story

- Infra/repo trước usecase; usecase trước handler; handler trước route; route trước swagger/wire; test đi kèm trước khi chốt checkpoint.
- Chạy `make generate` sau mỗi phase có thêm provider/handler mới.

### Parallel Opportunities

- Phase 2: T002/T003/T004 song song; T005/T006/T007 nối tiếp sau T004; T008/T009 sau T006; T010 sau T008+T009
- Sau Foundational: US4, US5, US6 có thể chạy song song (khác module: subscription/identity); US7, US8 song song
- US4+US5 cùng module subscription → chạy tuần tự hoặc cẩn thận file khác nhau (admin_billing_handler.go vs admin_subscription_handler.go — khác file, song song được)
- T052–T056 (US10 docs) song song
- T058–T061 (unit test) song song

---

## Parallel Example: Foundational (Phase 2)

```bash
# Song song (khác file):
Task: "Thêm CtxBrandId/CtxBrandRole vào internal/pkg/utils/contextutils/key.go"
Task: "Thêm GetBrandID/GetBrandRole vào internal/pkg/utils/contextutils/user.go"
Task: "Tạo IBrandMembershipCache interface trong internal/modules/brand/application/interface/caching/brand_membership_cache.go"

# Nối tiếp sau IBrandMembershipCache:
Task: "Tạo RedisBrandMembershipCache trong internal/modules/brand/infrastructure/caching/redis_brand_membership_cache.go"
Task: "Sửa GetUserBrandRole trong internal/modules/brand/application/usecase/brand/usecase.go"
Task: "Tạo BrandRoleMiddleware trong internal/modules/brand/presentation/middleware/brand_role.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 — IDOR + US2 — RBAC, cả hai P1)

1. Hoàn tất Phase 1: Setup (1 migration gộp)
2. Hoàn tất Phase 2: Foundational (context, cache, GetUserBrandRole, middleware, wire)
3. Hoàn tất Phase 3 (US1) → Phase 4 (US2)
4. **STOP & VALIDATE**: quickstart mục A + B; `make check` pass
5. Deploy/demo nếu sẵn sàng — đã đóng lỗ hổng bảo mật + RBAC tập trung

### Incremental Delivery

1. Setup + Foundational → nền RBAC sẵn sàng
2. US1 + US2 → **MVP bảo mật** (đóng IDOR, RBAC qua middleware)
3. US3 (ma trận + docs quyền) → tài liệu chuẩn
4. US4 → US5 → US6 (admin P2, theo thứ tự) → demo từng nhóm
5. US7 → US8 → US9 (admin P3 + audit) → demo từng nhóm
6. US10 (docs sync + out-of-scope) → hoàn tất feature
7. Polish (generate, test, quickstart, security review)

### Parallel Team Strategy

1. Team hoàn thành Setup + Foundational
2. Sau Foundational:
   - Dev A: US1 + US2 (RBAC/IDOR — ưu tiên cao nhất)
   - Dev B: US4/US5 (subscription admin)
   - Dev C: US6 (identity user admin)
3. Sau US1/US2: Dev A tiếp US3 → US10; Dev B/C tiếp US7/US8/US9 (độc lập)
4. Hợp nhất tại Polish (generate, test, quickstart)

---

## Notes

- [P] task = khác file, không phụ thuộc
- Mỗi user story độc lập implement/test được
- Chạy `make wire` sau mỗi phase thêm provider/handler; `make swagger` sau khi thêm annotation
- Commit sau mỗi task hoặc nhóm logic
- Tránh: task mơ hồ, xung đột cùng file, cross-story dependency phá vỡ tính độc lập

---

## Phase 14: Convergence

**Purpose**: Sửa các gap còn lại từ convergence check — tuân thủ §VII design-pattern.md.

- [X] T065 [P] Tạo `internal/modules/admin/application/errors/errors.go` — domain error constructors cho admin module dùng `apperror` factories (vd: `ErrAuditLogNotFound`, `ErrInvalidFilter`) thay vì raw errors; cập nhật usecase/handler để dùng error constructors mới per §VII (partial)
- [X] T066 [P] Tạo `internal/modules/admin/application/mapper/audit_log.go` — extract mapper entity→DTO từ `audit_log_uc.go:56-68` vào dedicated mapper package; refactor usecase gọi mapper thay vì inline for-loop per §VII (contradicts)
- [X] T067 [P] Kiểm tra `internal/modules/identity/application/dto/admin_user_dto.go` — xác nhận `AdminUserListRes` đã chứa `PaginationMetadata`; nếu chưa thì wrap bằng `shared_dto.PaginationResult[*AdminUserDTO]` trong handler per §VII pagination pattern (partial)