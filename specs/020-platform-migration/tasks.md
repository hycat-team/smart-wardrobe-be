---
description: "Task list cho feature 020-platform-migration (mô hình C multi-module + funded platforms + repatriation)"
---

# Tasks: Chuyển đổi hạ tầng sang nền tảng funded và tách 3 service (multi-module workspace)

**Input**: Design documents from `specs/020-platform-migration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md
**Constitution**: v3.0.0 (single service topology, compiler-enforced boundaries, portability)
**ADR**: `docs/decisions/021-funded-platforms-and-repatriation.md`

**Tests**: Có bao gồm test cho các hành vi cốt lõi (cờ cấu hình, fallback, adapter, parity) vì
tiêu chí chấp nhận yêu cầu kiểm chứng được và người dùng yêu cầu "mọi thứ chắc chắn".

**Organization**: Task nhóm theo user story để mỗi story hoàn thành và kiểm thử độc lập.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Có thể chạy song song (khác file, không phụ thuộc task chưa xong)
- **[Story]**: US1..US7 tương ứng spec.md
- Đường dẫn file cụ thể trong mỗi task
- **ID hậu tố** (`T004a`, `T007a`...): task bổ sung sau khi chốt danh sách gốc, **không renumber** để giữ ổn định tham chiếu (chạy ngay sau task cùng tiền tố)

## Path Conventions (mô hình C)

- Service modules: `services/account/`, `services/commerce/`, `services/ai/` (mỗi module một `go.mod`, `cmd/<service>/main.go`, bootstrap/di riêng)
- Module paths: `smart-wardrobe-be/services/<service>`, `smart-wardrobe-be/shared`, `smart-wardrobe-be/contracts`
- Shared module: `shared/` (config, infrastructure, presentation, domain, application, pkg)
- Contracts: `contracts/<owner>/v1/` (proto + generated + client)
- Ingress: `deployments/nginx/` (server block `api.closy` giữ nguyên + `api-v2` route tới Cloud Run) + `deployments/token-helper/` (sinh/cache ID token)
- Deployment: `deployments/docker-compose.yml` (local), `deployments/docker-compose.prod.yml` (VPS + profile `rollback`), `deployments/monolith-legacy/Dockerfile` (FR-043), `.dockerignore` ở root
- Docs: `docs/operations/`, `docs/system-design/`, `docs/decisions/`
- Spec artifacts: `specs/020-platform-migration/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Khởi tạo workspace, tooling, khung service, thư mục docs, chốt tag legacy.

- [x] T001 Khởi tạo `go.work` + `go.mod` cho 5 module (`services/account`, `services/commerce`, `services/ai`, `shared`, `contracts`); tạo `services/<service>/database/migrations/` (Account sẽ giữ toàn bộ migration lịch sử); `go work sync` pass
- [x] T002 [P] Add gRPC/protobuf dependencies (`google.golang.org/grpc`, `google.golang.org/protobuf`) và `go mod tidy` từng module dùng gRPC
- [x] T003 [P] Add buf tooling tại root: `buf.yaml` + `buf.gen.yaml` (output `contracts/`, plugin `protoc-gen-go`, `protoc-gen-go-grpc`)
- [x] T004 [P] Tạo skeleton mỗi service trong `services/`: `services/account/cmd/account/main.go`, `services/commerce/cmd/commerce/main.go`, `services/ai/cmd/ai/main.go` + `services/<service>/internal/bootstrap` + `services/<service>/internal/di` (minimal boot, chưa routes) + `services/<service>/Dockerfile` **riêng từng service (multi-stage: golang builder → distroless static nonroot; entrypoint `cmd/<svc>`; env `PORT` mặc định 8080; healthcheck `/api/v1/health`; binary nhúng tzdata `import _ "time/tzdata"`)**
- [x] T004a [P] Tạo `.dockerignore` ở **repo root** (loại `.git`, `docs`, `specs`, `*_test.go`, `testdata`, `deployments`, `*.md`); ghi rõ **build context = repo root** (`docker build -f services/<svc>/Dockerfile .`) vì cần `go.work` + `shared/` + `contracts/` (FR-042)
- [x] T004b [P] Verify **build độc lập từng image**: `docker build` cho từng service chỉ với chính Dockerfile đó (không build/generate service khác; `contracts/` generated đã commit) → 3 image chạy được standalone (SC-017)
- [x] T005 [P] Tạo skeleton `deployments/nginx/` (server block `api.closy` giữ nguyên + `api-v2` route tới 3 Cloud Run) và `deployments/token-helper/` (helper sinh/cache Google ID token)
- [x] T006 Tag `monolith-legacy` tại commit baseline = **HEAD của `main` ngay trước commit 020 đầu tiên** (commit monolith-green cuối cùng); ghi chú frozen vào tag message
- [x] T006a Tạo `deployments/monolith-legacy/Dockerfile` build **image rollback monolith** từ working tree của tag `monolith-legacy` (context = repo root tại commit đó, multi-stage, cùng quy ước `PORT`/healthcheck); chỉ phục vụ nhánh rollback, MUST NOT nhận code mới (FR-043)
- [x] T007 **Viết lại Makefile cho multi-module (FR-038)**: thay `WIRE_PATH`/`SWAG_MAIN_FILE`/`GO_BUILD_PACKAGE` gắn monolith bằng target per-service + workspace (`wire svc=`, `swagger svc=`, `build svc=`, `build-all`, `test-workspace`, `migrate-up svc=`, `migrate-create svc= name=`, `migrate-status svc=`, `buf-gen`, `buf-lint`, `nginx-reload`, `token-helper-build`, `image-build svc=`, `image-push svc=`, `deploy svc=`, `image-build-monolith-rollback`); cập nhật `tidy`/`vet`/`test` theo `go.work`; di chuyển `hygiene-scan` tool vào một module; sửa `test-integration` path; `docker-build svc=`
- [x] T007a [P] Thiết lập **Artifact Registry** (`asia-southeast1-docker.pkg.dev/<project>/smart-wardrobe`) cho image Cloud Run + quy ước tag `<svc>:<git-sha>` (immutable) và `:latest` chỉ trong CI; **giữ GHCR cho image VPS** (rollback + repatriation compose, dùng token sẵn có); ghi vào runbook `deploy-platform.yml`/docs (FR-042, FR-051)
- [x] T008 [P] Tạo skeleton `docs/operations/` (runbook 2 chiều) và cập nhật `docs/system-design/microservices/` (bỏ nội dung monolith)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Mô hình cấu hình dùng chung cho mọi story. **PHẢI xong trước US1–US7.**

- [x] T009 Add config fields trong `shared/config/config.go`: `App.Mode` (`APP_MODE`: http|worker|all, mặc định `all`), `Elasticsearch.Enabled`, `Database.MigrateOnStart` (per-service, bật ở managed), `Database.ListenerHost/ListenerPort/ListenerSslMode`, `Database.DrillSchema`, `Database.QueryExecMode` (mặc định `simple_protocol` khi dùng pooler), `Redis.TLSEnabled`, `RabbitMQ.TLSEnabled/ManagementTLSEnabled`, `Outbox.ListenerEnabled`, `Jwt.Algorithm/PrivateKey/PublicKey`, `Ingress.AccountBaseURL/CommerceBaseURL/AIBaseURL/TokenHelperURL/TokenAudience/AuthHeader`, `Server.SSE.AIChunkDeadlineSeconds/HeartbeatSeconds`
- [x] T010 Boolean env override + validation trong `shared/config/loader.go` và `shared/config/validation.go` (JWT key theo algorithm; drill schema chỉ dùng khi chạy drill; `migrate_on_start` bật ở managed cho từng service; ingress URL/audience bắt buộc ở managed; **`PORT` → `server.port` ưu tiên cao nhất**, fallback `SERVER_PORT`/YAML — FR-049)
- [x] T011 Parameterize `configs/config.yaml` và `configs/config.production.yaml`: thay host/endpoint literal bằng `${DB_HOST}`, `${REDIS_HOST}`, `${RABBITMQ_HOST}`, `${ELASTICSEARCH_ADDRESSES}`; đặt `server.sse.ai_chat_chunk_deadline_seconds: 30` và `server.sse.heartbeat_seconds: 15` **giống nhau ở cả hai môi trường**
- [x] T012 Update `.env.example` và `.env.production.example` (ES enabled, TLS toggles, migrate_on_start, listener_*, drill_schema, JWT keys, ingress URLs/audience/token helper)

**Checkpoint**: 5 profile (local / vps-legacy / vps / managed / managed-drill) load được, workspace build xanh.

---

## Phase 3: User Story 1 - Tắt search engine mặc định (Priority: P1) 🎯 MVP

**Goal**: ES tắt qua cấu hình, search fallback DB, không sinh DLQ.
**Independent Test**: quickstart VS1.

### Tests for User Story 1

- [x] T013 [P] [US1] Unit test ES disabled client: `IndexDocument`/`DeleteDocument` no-op, `Search` trả lỗi, không start health goroutine
- [x] T014 [P] [US1] Unit test search fallback khi ES disabled (wardrobe item read usecase)

### Implementation for User Story 1

- [x] T015 [US1] Guard `Enabled` trong search client (bỏ health goroutine, no-op index/delete, search lỗi)
- [x] T016 [US1] Guard initial sync trong search_sync usecase (skip khi disabled)
- [x] T017 [US1] Guard `RunInitialSyncWithRetry` start trong search_sync worker
- [x] T018 [US1] Phơi trạng thái `elasticsearch` trong health endpoint
- [x] T019 [US1] Regenerate Wire; xác nhận 3 service build

**Checkpoint**: US1 hoạt động độc lập; VS1 pass.

---

## Phase 4: User Story 2 - Chạy ổn định trên pooler (Priority: P1)

**Goal**: App boot dù `LISTEN` không khả dụng; outbox poll fallback; migration có kiểm soát.
**Independent Test**: quickstart VS2.

### Tests for User Story 2

- [x] T020 [P] [US2] Unit test outbox listener no-op/non-fatal + poll fallback
- [x] T021 [P] [US2] Unit test `migrate_on_start=false` bỏ qua migration

### Implementation for User Story 2

- [x] T022 [US2] Refactor outbox listener: `NewOutboxListener` không fatal; trả no-op listener khi disabled/lỗi; dùng listener DSN riêng (direct 5432)
- [x] T023 [US2] Gate `RunMigrations` theo `database.migrate_on_start` trong bootstrap
- [x] T024 [US2] Regenerate Wire; xác nhận boot với listener disabled
- [x] T025 [US2] Ghi lệnh migrate session-DSN vào runbook bảng biến môi trường
- [x] T025a [US2] **Tương thích Supavisor transaction pooler**: bật `default_query_exec_mode=simple_protocol` cho runtime (6543); bỏ `search_path` khỏi DSN client và set **server-side** (`ALTER ROLE`/`ALTER DATABASE` trên Supabase); test thực: query resolve đúng các schema module khai báo trong `database.search_path`, 0 lỗi prepared statement; giữ direct 5432 cho migrate/`LISTEN`/advisory lock (FR-046, SC-020)

**Checkpoint**: US2 hoạt động độc lập; VS2 pass.

---

## Phase 5: User Story 5 - Tách 3 service module C (Priority: P1)

**Goal**: 3 service module build/deploy độc lập; ranh giới do compiler enforce; JWT bất đối xứng + ID token; ingress auth bằng **Cloud Run IAM** (đã bỏ origin-auth `X-Gateway-Secret`); swagger per-service đủ 9 section.
**Independent Test**: quickstart VS3 + VS4.

> Chỉ 4 domain contract vượt ranh giới service lên protobuf (Identity, Subscription, Admin, AI).
> Brand/Fashion/Wardrobe/Notification giữ in-process (xem contracts §5).

### Contracts (protobuf)

- [x] T026 [P] [US5] `contracts/identity/v1/identity.proto` (`IdentityService` + `IdentityDashboardService`)
- [x] T027 [P] [US5] `contracts/subscription/v1/subscription.proto` (`SubscriptionService` + `SubscriptionDashboardService`)
- [x] T028 [P] [US5] `contracts/admin/v1/admin.proto` (`AdminService`)
- [x] T029 [P] [US5] `contracts/ai/v1/ai.proto` (`AIOrchestratorService` + `AIDashboardService`)
- [x] T030 [US5] Generate stubs (`make buf-gen`) vào `contracts/<owner>/v1/`; commit generated code; wire vào `generate-check`

### Module split + dual adapter

- [x] T031 [US5] Chuyển module identity/subscription/admin/notification vào `services/account/internal/modules/` (giữ nguyên logic, đổi import path); đồng thời chuyển **toàn bộ migration lịch sử hiện có** từ `database/migrations/` sang `services/account/database/migrations/` (kèm `embed.go`)
- [x] T032 [US5] Chuyển module wardrobe/fashion/brand/dashboard vào `services/commerce/internal/modules/`
- [x] T033 [US5] Chuyển module aiexecutor vào `services/ai/internal/modules/`; tách `internal/shared/**` thành module `shared/` (đặt HTTP/SSE/middleware tại `shared/presentation`; bootstrap chuyển về `services/<service>/internal/bootstrap`)
- [x] T034 [US5] Xóa `cmd/server` khỏi `main`; verify workspace không còn reference monolith — **đã xoá `cmd/`, `internal/`, `api/` (swagger monolith), root `Dockerfile`, root `go.mod`/`go.sum`, `database/migrations` rỗng; `go.work` bỏ module root; 3 Dockerfile service bỏ `COPY go.mod go.sum`; Makefile dọn hết `LEGACY_*` + target legacy; `token-helper` tách module riêng; errorutils bỏ path monolith**. 5 module + token-helper build/vet/test xanh
- [x] T035 [P] [US5] gRPC client cho `IUserBriefContract` + `IUserDashboardReader` — `contracts/identity/v1/client/` (+ **owner transport mapper** `services/account/contract/identity/transport/` giữ enum domain kín; helper map lỗi `contracts/internal/grpcx/`)
- [x] T036 [P] [US5] gRPC client cho `IUserSubscriptionContract` + `IUserQuotaContract` + `ISubscriptionDashboardReader` — `contracts/subscription/v1/client/` (+ owner transport mapper `services/account/contract/subscription/transport/`; `decimal` ⇄ string giữ precision, `has_expires_at` cho con trỏ `*time.Time`)
- [x] T037 [P] [US5] gRPC client cho `IAuditLogContract` — `contracts/admin/v1/client/` (+ owner transport mapper `services/account/contract/admin/transport/`; payload `map[string]any` ⇄ `google.protobuf.Struct`)
- [x] T038 [P] [US5] gRPC client cho `IAIOrchestrator` + `IAICostDashboardReader` — `contracts/ai/v1/client/` (+ owner transport mapper `services/ai/contract/aiexecutor/transport/`; stream `GenerateChatTextStream` ⇄ channel pair; `ResponseSchema any` ⇄ JSON string; **không** dùng enum domain)
- [x] T039 [US5] gRPC server (phụ thuộc **contract interface**, không inject usecase) — `services/<service>/internal/modules/<owner>/presentation/grpc/` của identity/subscription/admin/aiexecutor (+ DTO⇄proto ở `transport/`; compile-time `var _` assert; stream server⇄channel bridge cho AI); đồng thời cung cấp **in-process impl** (`presentation/contract/`) cho 4 domain vượt biên để giữ dual-adapter (FR-019)
- [x] T040 [US5] Wire composition `services/account/` (host gRPC Identity/Subscription/Admin; không client ngoài) — `services/account/internal/di/{app.go,wire.go,wire_gen.go}`: `AccountApp` nhúng `AccountBundle` + 5 gRPC server từ contract port; regenerate bằng `wire`
- [x] T041 [US5] Wire composition `services/commerce/` (client Identity/Subscription/Admin/AI + 3 read) — `services/commerce/internal/di/{app.go,connections.go,wire.go,wire_gen.go}`: `CommerceContracts` (8 contract qua gRPC) + dial Account/AI từ `Ingress.*BaseURL` (https→TLS, http→plaintext); `go mod tidy` thêm `grpc v1.83.2`
- [x] T042 [US5] Wire composition `services/ai/` (host AI; client Identity/Subscription/Admin) — `services/ai/internal/di/{app.go,connections.go,wire.go,wire_gen.go}`: `AIContracts` (UserSubscription/AuditLog qua gRPC tới Account; Identity-auth verify cục bộ nên không cần UserBrief client); dial dùng chung `contracts/grpcx.DialService`; `go mod tidy` thêm `grpc v1.83.2`
- [x] T121 [US5] Gỡ import implementation xuyên service — **đã xoá hẳn `services/{account,commerce,ai}/providers/`** (transitional bundle chỉ phục vụ monolith); Account composition root tự khai báo `AccountApp` với router/worker/contract của chính mình; commerce/ai `internal/di` chỉ dùng gRPC client (T035–T038). Verify: 0 import `services/*/providers`; 5 module build xanh; kiểm tra biên dịch phủ định giữ hiệu lực (§I/§II)
- [x] T034a [US5] **Tách bootstrap theo `APP_MODE`** (http|worker|all): cả 3 service có runtime thật (`bootstrap/{app,router,worker}.go` + `grpc.go` cho Account/AI + `di` wire thật + `cmd` thật); worker (consumer/outbox/cron) chỉ chạy ở `worker`/`all`, `http` chỉ HTTP+gRPC; Account/AI 1 port h2c mux HTTP+gRPC, Commerce HTTP thuần (không host gRPC); migration/validator/graceful shutdown; health `/api/v1/health` mọi mode (FR-044, SC-018)
- [x] T123 [US5] **Bỏ hardcode cổng 8080 trong code**: thay literal `port = "8080"` trong `services/<svc>/cmd/<svc>/main.go` bằng giá trị đọc từ config (`PORT` → config → `SERVER_PORT`/YAML) per FR-049/SC-023; default local chỉ đặt ở `configs/config.yaml` (`server.port`) kèm ghi chú; assert **0 literal `"8080"` trong `services/**/*.go`** (grep) — **sau T040–T042/T034a** (bootstrap thật thay skeleton của T004) — **verified: grep 0 hit, 3 service build xanh**
- [x] T034b [US5] **Worker singleton + advisory lock**: cấu hình deploy worker `min-instances=1` **+ `max-instances=1`** (thực thi ở **T074/T075**); bọc advisory lock Postgres cho job không idempotent — `shared/infrastructure/db/job_locker.go` (`TryLock` session-pin + IDs `8_200_101/102` + direct DSN, không pooler); `SubscriptionRenewalWorker` + `LoyaltyPointExpiryWorker` skip khi lock bị giữ (FR-045, SC-019)
- [x] T043 [US5] Verify compile-time: import thử implementation xuyên service và xác nhận build fail (đã verify cho `domain/entities`/`domain/repositories`; case `providers` được T121 verify lại)

### Ingress auth (Cloud Run IAM — thay origin-auth cũ)

- [x] T044 [US5] **Verify đã bỏ origin-auth**: xác nhận service KHÔNG yêu cầu header `X-Gateway-Secret` (hiện code không còn reference `X-Gateway-Secret`; nếu phát hiện thì gỡ) và `Authorization` (Closy JWT) được giữ nguyên; auth do Cloud Run IAM ở ingress (FR-033) — **verified: 0 reference `X-Gateway-Secret` trong code/config (chỉ còn trong docs lịch sử)**
- [x] T045 [P] [US5] Unit test (không cần Cloud Run): service KHÔNG yêu cầu `X-Gateway-Secret`; `Authorization` (Closy JWT) đi qua nguyên vẹn — `services/account/internal/modules/identity/presentation/middleware/auth_ingress_test.go` (3 test: không cần gateway secret / gateway-only bị chặn / token nguyên vẹn). Smoke test end-to-end IAM (thiếu token bị từ chối) nằm ở **T073 (Phase 6)**

### Auth liên service (2 lớp)

- [x] T046 [US5] JWT bất đối xứng: ký ES256 (identity) — jwtutils (`shared/pkg/utils/jwtutils/es256.go`: `LoadECDSAPrivateKey`/`LoadECDSAPublicKey` từ PEM + `GenerateTokenES256`; HS256 giữ nguyên; smoke sign/verify ES256 pass)
- [x] T047 [US5] JWT verify middleware mọi service + hỗ trợ legacy HS256 giai đoạn chuyển tiếp + tra cứu thu hồi phiên qua Redis blacklist dùng chung (FR-023) — `jwtutils/verify.go` (`ValidateTokenFlexible`: peek alg → ES256/HS256-legacy; fail-closed khi thiếu public key); `AuthMiddleware` dùng flexible verifier, blacklist lookup giữ nguyên; test HS256 hiện có vẫn pass
- [x] T048 [P] [US5] Test ký/verify ES256 + từ chối HS256 khi tắt legacy + test session revocation bị chặn qua Redis blacklist (FR-023) — `shared/pkg/utils/jwtutils/es256_test.go` (5 test: sign/verify, cross-alg reject, legacy on/off, expiry/type, PEM load) + `auth_revocation_test.go` (token/user revoked bị chặn)
- [x] T049 [US5] Lớp hạ tầng: gRPC client gắn Cloud Run ID token (`run.invoker`) khi gọi service khác (FR-034) — `contracts/grpcx/{auth.go,dial.go}`: `TokenProvider` + provider **metadata server (stdlib, 0 dependency mới)** chỉ bật khi `IsManagedEnv` (local/VPS = no-op); cache theo `audience` (refresh theo claim `exp`); unary+stream interceptor gắn `authorization: Bearer <id token>`, fail-fast khi mint lỗi; `audience` = `Ingress.AccountBaseURL/AIBaseURL`. **Key 2 lớp chốt**: `authorization` = ID token (IAM), **`x-closy-jwt`** = Closy JWT app (`ContextWithAppJWT`; T039/T073a dùng cùng hằng số). **(Quyết định chốt A2 — metadata stdlib, không SDK — ngày 2026-09-17.)** 6 test pass
- [x] T049a [US5] Chuyển **global rate-limit shield** trong `shared/presentation/middleware/ratelimit.go` sang backend **Redis dùng chung** (`IRateLimiter` — `redis_rate_limiter.go`) để chính xác qua nhiều instance; thêm rule **IP+user/token cho login/OTP** (`LoginRateLimitMiddleware`, gắn vào `authApi` group) + rule chung API; test 2 instance mô phỏng (FR-054, SC-028) — `ratelimit_redis_test.go` (shared backend, 429 nhất quán)

### Swagger per-service (9 section)

- [x] T050 [US5] `swag init` cho service Account (4 module) — output `services/account/api/swagger/` (chạy từ service dir; root invocation không resolve cross-module)
- [x] T051 [US5] `swag init` cho service Commerce (4 module) — output `services/commerce/api/swagger/`
- [x] T052 [US5] `swag init` cho service AI (1 module) — output `services/ai/api/swagger/`
- [x] T053 [US5] Makefile targets swagger per-service (chỉ Makefile); **job CI parity 9 section thuộc T113** để tránh sửa `.github/workflows/ci.yml` hai lần — **đã sửa `swagger`/`swagger-all` chạy từ service dir** (root invocation không resolve cross-module types)
- [x] T054 [US5] Xóa pipeline swagger monolith cũ (**đã xoá `api/swagger` + target `swagger-legacy` trong T034**); cập nhật `swagger_access_code` theo từng service — `SwaggerLockMiddleware` đã gắn mỗi engine, `SWAGGER_ACCESS_CODE` env-driven trong `.env.*`; image chứa `api/swagger` (`COPY` trong 3 Dockerfile); `make run` cd vào service dir để `./api/swagger` resolve

### Tests hợp đồng

> Ngoại lệ TDD có chủ đích cho US5: contract behavior test cần **cả hai adapter** (in-process + gRPC) đã tồn tại nên T055/T056 chạy **sau T040–T042**; các unit test khác vẫn theo TDD.

- [x] T055 [P] [US5] Contract behavior test dùng chung in-process + client adapter (8 contract) — round-trip proto⇄DTO cho 4 transport (identity/subscription/admin/ai: enum, decimal, map, Struct, JSON schema, timestamps) + interop client↔server thật qua localhost (identity); kèm kiểm tra **Pure Contract Usecase** (Constitution §II): contract usecase chỉ dùng repo/mapper của module mình, không gọi contract module khác để enrich
- [x] T056 [P] [US5] Idempotency test quota/audit/daily-quota (chống trừ/ghi trùng khi retry) — `quota_test.go::TestGetAndResetDailyQuotaIsIdempotentUnderRetry` (retry sau timeout không reset lần 2, đúng 1 update). Ghi nhận: `Update*Quota` là increment (không idempotent tự thân) nên caller (gRPC client) MUST NOT retry — client hiện fail-fast không retry; advisory lock (T034b) chống chạy trùng khi overlap/instance restart; audit append-only qua outbox

### Documentation

- [x] T057 [P] [US5] `docs/system-design/microservices/00-tong-quan-3-service.md` (mô hình C, bỏ monolith)
- [x] T058 [P] [US5] `docs/system-design/microservices/01-ranh-gioi-va-contract.md` (bảng proto vs in-process)
- [x] T059 [P] [US5] `docs/system-design/microservices/02-grpc-va-event.md` (gồm ID token + JWT hai lớp)
- [x] T060 [P] [US5] `docs/system-design/microservices/03-adapter-va-composition.md`
- [x] T061 [P] [US5] `docs/system-design/microservices/04-auth-asymmetric-jwt.md` (gồm ID token + JWT hai lớp)
- [x] T062 [P] [US5] `docs/system-design/microservices/05-repo-va-ci.md`
- [x] T063 [P] [US5] `docs/system-design/microservices/06-lo-trinh-va-checklist.md`

**Checkpoint**: 3 binary build; boundary compile-time verified; VS3/VS4 pass (swagger parity 9 section verify thủ công ở Phase 5; job CI parity nằm ở T113).

---

## Phase 6: User Story 3 - Ingress + Compute + Song song (Priority: P1)

**Goal**: VPS Nginx là ingress (api.closy giữ nguyên; api-v2 → 3 Cloud Run), DNS Vercel, TLS cho api-v2, Cloud Run IAM + token helper, Cloud Run trial có cap, per-service migration on-start, data wiring, SSE ổn định, runbook song song (không cutover).
**Independent Test**: quickstart VS6.

### Code (TLS + managed wiring)

- [x] T064 [P] [US3] TLS cho Redis (`redis.tls_enabled`) — `redisOptions()`: `TLSConfig{MinVersion: TLS1.2}` khi bật, plaintext khi tắt
- [x] T065 [P] [US3] TLS cho RabbitMQ (`amqps` + `TLSClientConfig`) — `amqpURL()`/`amqpTLSConfig()`: `amqps` + TLS1.2 khi bật
- [x] T066 [US3] HTTPS cho management/gauge API broker — `FetchQueueDepths(..., useTLS)` + `mgmtURL()`; gauge probe truyền `ManagementTLSEnabled`
- [x] T067 [US3] Test TLS toggle on/off (assert cấu hình client) — `tls_test.go` (Redis/AMQP/mgmt, on/off, 5 test pass, không cần broker thật)

### Ingress (VPS Nginx + Vercel DNS + Cloud Run IAM)

- [ ] T070 [US3] Mở rộng cert Let's Encrypt gồm `api-v2.closy.hycat.online` (`certbot --expand`/`--cert-name`) + xác nhận renew tự động (FR-040) — **trước T068** (spec edge case: cert chưa gồm host mới ⇒ TLS handshake fail)
- [x] T069 [US3] Thêm server block Nginx cho `api-v2`: route longest-prefix tới 3 Cloud Run `run.app` (xử lý chồng lấn `/me/*`, `/admin/*`); **giữ nguyên** block `api.closy` → `http://backend:8080`. Route map single-source; nguồn suy ra từ route đã tách theo service ở T040–T042 (FR-028)
- [ ] T068 [US3] Thêm bản ghi DNS `api-v2.closy.hycat.online` trên **Vercel** trỏ về VPS hiện tại (A hoặc CNAME); **không** NS-delegate, **không** Cloudflare (FR-029) — **sau T069/T070** để không mở traffic khi cert/route chưa sẵn sàng
- [x] T071 [US3] **Token helper** trên VPS: sinh/cache/refresh Google ID token (audience = `run.app` URL) cho service account least-privilege (`roles/run.invoker` trên đúng 3 service); Nginx `auth_request` + `auth_request_set` inject `X-Serverless-Authorization` (FR-033/FR-039) — **implemented: `mintGoogleIDToken` dùng `google.golang.org/api/idtoken` (`WithCredentialsJSON`) + cache/refresh-ahead 5m; bind `:PORT` mọi interface để Nginx container gọi qua service name; service `token-helper` trong prod compose (profile `parallel`, context `deployments/token-helper`); sửa Dockerfile build standalone (go.sum đủ, builder `golang:1.26`); **wiring audience per-service**: generator phát `auth_request /_token_helper_<svc>` + template định nghĩa 3 `location = /_token_helper_<svc>` truyền `?audience=${..._RUN_URL}` (trước đó helper nhận thiếu audience → 400); unit test cache/refresh/handler/healthz/listenAddr**. Verify: `go test -race ./...` xanh; image build OK; smoke container `/healthz`→200, `/token` (key giả)→502 (không còn 503 "not implemented"), thiếu audience→400; `make nginx-check` 154/154 per FR-033/FR-039 (missing)
- [ ] T072 [US3] SSE qua Nginx: `proxy_buffering off` cho path SSE, timeout đọc đủ lớn, heartbeat 15s, `ai_chat_chunk_deadline_seconds` = 30s; test stream AI dài thật (FR-036, SC-014)
- [ ] T073 [US3] Cloud Run IAM: tắt `allUsers`, grant `run.invoker`, verify audience = `run.app` URL; smoke test thiếu token bị từ chối (FR-033, SC-013); `proxy_ssl_server_name on` + resolve động, không pin IP (FR-037)

### Quan trắc (FR-057 — thực thi trước deploy)

- [x] T119 [US3] Chuyển logger 3 service sang log JSON có cấu trúc ra stdout trên Cloud Run (giữ console màu cho local/dev qua `APP_MODE`/`ENV`), để Cloud Logging thu thập được per FR-057 (missing) — **verified: `logger.New()` đã JSON hoá mọi env trừ dev** (`logger_json_test.go`: managed→JSON hợp lệ, dev→console); không cần đổi code
- [ ] T120 [US3] Thêm metrics + alert Cloud Monitoring tối thiểu cho health/latency/error-rate của 3 service (cả HTTP và worker) + runbook vận hành `docs/operations/ha-tang/14-monitoring.md`, per FR-057/SC-031 (missing) — **trước T074** và **trước T082/T117** (gate đo được FR-041 cần metrics tồn tại trước)

### Compute (Cloud Run trial + cap)

- [ ] T075 [US3] Billing cap + alert + **cap autoscale theo ngân sách (FR-047)**: HTTP `max-instances=2`, worker `max-instances=1` (theo yêu cầu singleton FR-045 ở T034b), `concurrency=40`; **đặt cap/alert trước T074**; phần verify tổng client conn ≤ giới hạn gói (Supabase pooler đo thực −20% headroom, AMQP ≤100, Redis ≤256) bằng test tải nhẹ chạy **sau T074** (SC-021)
- [ ] T074 [US3] Deploy 6 units (3 HTTP scale-to-zero + 3 worker `min=1`/`max=1`) region `asia-southeast1`, request-based billing; set `APP_MODE=http`/`worker` tường minh cho từng deployment (FR-044/FR-045) — **sau T075 (cap), T119/T120 (quan trắc) và T034a/T034b (APP_MODE)**
- [ ] T073a [US3] **gRPC qua Cloud Run HTTP/2**: verify 3 HTTP service (đã deploy ở T074) chạy với `--use-http2` — nếu thiếu thì bổ sung vào cấu hình deploy (T094/T074); gRPC client dial `run.app` + TLS + ID token metadata + JWT metadata riêng + keepalive; verify `grpc.health.v1` + call thực Commerce→Account/AI, AI→Account (FR-048, SC-022) — **sau T074** và sau T049 (ID token client)
- [ ] T073b [US3] **Verify media upload** (Cloudinary), health `GET /api/v1/health` và timezone `Asia/Ho_Chi_Minh` trên image distroless nonroot từ Cloud Run (FR-049/FR-053, SC-024, SC-027) — **sau T074**

### Data (đúng giới hạn gói funded)

- [ ] T076 [US3] Supabase: **1 database dùng chung** + credential riêng; pooler app / direct listener; verify `LISTEN` (FR-025)
- [x] T077 [US3] **Per-service migration on-start (FR-035)**: mỗi service chạy Goose provider trên `services/<service>/database/migrations` khi khởi động; config `database.migration_table` (`DATABASE_MIGRATION_TABLE`: account=`goose_db_version` mặc định, commerce/ai=bảng hậu tố) / `database.migration_allow_out_of_order` (`DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER`, managed=true) + env trong `.env.example`; `goose.WithTableName` + `goose.WithAllowOutofOrder` + **advisory lock dùng chung** (session locker, cùng lock id); verify không chạy lại các migration lịch sử đã apply khi Account khởi động; **connection migrate tách sang direct — xem T141 (Option A)**
- [ ] T078 [US3] CloudAMQP Tiger: vhost riêng; verify quorum queue (fallback classic durable + ADR nếu không hỗ trợ); share trong 100 connections, prefetch 5
- [ ] T079 [US3] Redis Cloud Essentials: TLS, share trong 256 connections; verify throughput/bandwidth cho tải demo
- [ ] T080 [US3] Drill runbook dùng `database.drill_schema` rồi xoá, không đụng schema chính (FR-027)
- [ ] T081 [US3] Dump dữ liệu VPS → Supabase cho `api-v2` + verify checksum; ghi rõ đây là **song song** (monolith vẫn là nguồn ghi chính), re-dump khi cần đồng bộ (US3 AC2)
- [ ] T082 [US3] Gate song song: xác nhận `api-v2` chạy ổn qua VPS Nginx; `api.closy` **0 thay đổi routing** (SC-015); xác nhận **0 request `api-v2` đi qua Cloudflare** (SC-016); ghi mốc

### Runbook song song + CI/CD

- [x] T083 [P] [US3] `docs/operations/ha-tang/01-tong-quan.md`
- [x] T084 [P] [US3] `docs/operations/ha-tang/02-supabase-postgres.md`
- [x] T085 [P] [US3] `docs/operations/ha-tang/03-redis-cloud.md`
- [x] T086 [P] [US3] `docs/operations/ha-tang/04-cloudamqp-rabbitmq.md`
- [x] T087 [P] [US3] `docs/operations/ha-tang/05-compute-cloudrun.md`
- [x] T088 [P] [US3] `docs/operations/ha-tang/06-vps-nginx-ingress-cloudrun.md` (server block api-v2, giữ nguyên api.closy)
- [x] T089 [P] [US3] `docs/operations/ha-tang/07-elasticsearch-tat-mac-dinh.md`
- [x] T090 [P] [US3] `docs/operations/ha-tang/08-bang-bien-moi-truong.md`
- [x] T091 [P] [US3] `docs/operations/ha-tang/09-kien-truc-song-song-tong-quan.md`
- [x] T092 [P] [US3] `docs/operations/ha-tang/10-chuyen-du-lieu-vps-supabase.md` (dump/restore + backfill + checksum)
- [x] T093 [P] [US3] `docs/operations/ha-tang/11-dns-vercel-ci-cd.md` (bản ghi `api-v2` trên Vercel + workflow platform)
- [x] T094 [US3] `.github/workflows/deploy-platform.yml` — **CI matrix theo từng service** (account/commerce/ai độc lập: build image từ Dockerfile riêng → push Artifact Registry theo `<svc>:<sha>` → `gcloud run deploy` HTTP + worker của service đó, **tiêm env secrets từ GitHub Actions secrets**, không in giá trị ra log); 1 service fail không chặn 2 còn lại (FR-042, FR-052) — HTTP scale-to-zero + worker singleton, IAM closed, WIF auth; monolith đã xoá nên không còn workflow VPS song song (rollback ở `deploy-production.yml`)
- [ ] T094a [US3] Verify **deploy độc lập** end-to-end: deploy riêng lẻ từng service (không rebuild service khác), pin version theo SHA từng service, chạy smoke test `api-v2` tới đúng service vừa deploy (SC-017)
- [x] T095 [US3] `docs/operations/nginx-ingress-cloudrun-auth.md` (route map single-source, token helper, rotate SA key, smoke test IAM, ghi residual risk không có WAF/rate-limit ở Nginx — FR-054)
- [x] T096 [US3] `docs/operations/manual-cutover-api-closy-to-cloudrun.md`: runbook **cutover thủ công** `api.closy` → Cloud Run (điều kiện tiên quyết, đổi Nginx/DNS, đồng bộ dữ liệu, kiểm thử, **rollback 2 mức: Cloud Run revision-level + Nginx về monolith**); ghi rõ timing do owner quyết định (FR-041); MUST ghi **ngưỡng gate đo được cố định** (smoke 100% pass, error-rate ≤ 1%, p95 ≤ 500ms, SSE không rớt, ổn định **7 ngày liên tiếp** — per FR-041/SC-015, runbook không nới ngưỡng)

**Checkpoint**: Runbook song song thực thi được; VS6 pass (api-v2 qua Nginx, api.closy không đổi).

---

## Phase 7: User Story 4 + 6 + 7 - VPS 3-service + Repatriation (gồm teardown) + funding clock (Priority: P2)

**Goal**: Đường về VPS là 3 service compose + Nginx gateway (không monolith); repatriation gồm teardown trial; funding clock có chủ.
**Independent Test**: quickstart VS5 + VS8.

- [x] T097 [US4] `deployments/docker-compose.prod.yml` mới: 3 service + postgres+pgvector/redis/rabbitmq + nginx gateway + certbot; thêm **profile `rollback`** dùng image `monolith-legacy` (biến `MONOLITH_IMAGE`) để quay lại monolith tạm thời (FR-043) — viết lại: `account/commerce/ai` (image GHCR `${*_IMAGE}`, healthcheck `/api/v1/health`), nginx depends_on 3 service, bỏ ES (mặc định tắt)
- [ ] T097a [US4] Build + push **image rollback `monolith-legacy`** vào **GHCR** (dùng token/CI sẵn có) từ tag frozen (`make image-build-monolith-rollback`) **trước khi hết cửa sổ funded**; verify profile `rollback` khởi động và phục vụ `api.closy` (SC-017, FR-051)
- [x] T097b [US4] `.github/workflows/deploy-vps-services.yml` mới: `workflow_dispatch` + approval environment; **matrix 3 service** build từ `services/<svc>/Dockerfile` → push **GHCR** → `docker compose -f deployments/docker-compose.prod.yml up -d <svc>`; **pin image theo digest**; dùng cho repatriation + vận hành VPS sau đó; smoke test `api.closy` (FR-056, SC-030)
- [x] T097c [US4] Chuyển `.github/workflows/deploy-production.yml` thành `workflow_dispatch` build **từ tag `monolith-legacy`** (checkout ref=tag) → GHCR `monolith-legacy:<sha>` → deploy qua compose **profile `rollback`** (`MONOLITH_IMAGE`, pin digest); chỉ rollback khẩn cấp; MUST NOT build monolith từ `main` (FR-056, SC-030) — **chạy trước T034** — dùng `git archive monolith-legacy` + `deployments/monolith-legacy/Dockerfile`, xác nhận chuỗi `rollback`, GHCR pin digest, dừng 3 service khi rollback
- [x] T098 [US4] Nginx route map (repatriation) multi-host: **cả `api.closy` và `api-v2` → 3 service compose** (dùng chung route map, không còn monolith mặc định); `api-v2` **redirect 301 → `api.closy`** trong grace period rồi **xoá bản ghi DNS** (FR-055); monolith compose chỉ là **rollback tạm thời** nếu owner chọn; smoke test route
- [x] T099 [US4] Nginx ingress auth khi self-host (không còn Cloud Run IAM) + **body-limit/timeout/security-header/logging-request-ID** (rate-limit đã nằm ở app — FR-054; ghi rõ residual risk "không WAF đầy đủ" trong runbook repatriation)
- [ ] T100 [US4] ACME challenge local cho cả 2 host + certbot renew
- [x] T101 [US4] `docs/operations/ha-tang/12-ve-lai-vps.md`: migrate ngược (Supabase→Postgres, drain AMQP, Redis snapshot/warmup), DNS về VPS, tắt funded trước renew
- [x] T102 [US4] `docs/operations/repatriate-to-vps.md`: runbook repatriation đầy đủ 2 chiều, **bao gồm bước teardown trial** (viết song song từ bây giờ)
- [x] T122 [US4] `docs/operations/ha-tang/13-ngung-vps-va-thu-hoi-secret.md`: quy trình **thu hồi credential cũ** (JWT/SA key/DB/AMQP/Redis, rotate + revoke) và **ngừng VPS cũ** sau khi xác nhận cutover/repatriation thành công (FR-016) — **sau T098/T101/T102**
- [x] T103 [US4] Env template profile `vps` (sslmode=disable, TLS off, migrate_on_start=true)
- [x] T104 [US4] Kiểm thử local: 3 service profile vps hoạt động, không sửa code — **verified T104**: 3 image build xanh (sửa Dockerfile: golang:1.26, shared/go.sum, token-helper go.mod, cross-service source, COPY configs); compose valid; up 6 container **healthy** (migrations chạy 24 rows + 8 schema, validator pass, `/api/v1/health` 200); phát hiện & sửa 3 lỗi thật (boot order validator→migration, distroless wget healthcheck → binary probe, MONOLITH_IMAGE placeholder); dọn sạch test artifacts
- [x] T105 [US6] `docs/operations/gcp-trial-teardown.md`: thứ tự xóa (Cloud Run → artifacts → alerts), verify không còn resource tính phí — **là một bước trong repatriation**, không tách rời
- [ ] T106 [US6] Drill teardown trên môi trường Cloud + đối chiếu billing trial về 0 (SC-010)
- [x] T107 [US7] Funding clock T-30/T-14/T-7/T-1 vào plan + checklist ký xác nhận (backup verify, restore drill, DNS rehearsal) — `docs/project/funding-clock-checklist.md` (4 checkpoint + người ký + bằng chứng)
- [ ] T108 [US7] Drill repatriation đầy đủ trên môi trường Cloud trước T-30; ghi kết quả + thời gian (mục tiêu ≤ 4h)

**Checkpoint**: Quay về VPS được; VS5 pass; teardown nằm trong repatriation; funding clock có chủ.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [x] T109 [P] Update `README.md` (multi-module workspace, 3 service, funded platforms, ES optional, repatriation)
- [x] T110 [P] Update `docs/development/local-setup.md` (compose 3 service local, managed environment) — module paths → `services/<svc>/internal/modules`, migration → `migrate-* svc=` per-service
- [x] T111 [P] Update `docs/system-design/architecture.md` (mô hình C + ES optional + ingress Nginx + Cloud Run) — banner hết stale, §I paradigm 3 service, §II layout `services/`, sweep path `internal/` cũ
- [x] T112 [P] Verify ADR links từ plan/spec: `docs/decisions/021-funded-platforms-and-repatriation.md` + `docs/decisions/023-vps-nginx-ingress-cloud-run-iam.md` (và ADR 022 email) — **đã thêm 023/022 vào plan.md ADR line** (spec đã có)
- [x] T113 **Viết lại `.github/workflows/ci.yml` cho workspace**: cache `**/go.sum` + `go.work.sum`; job build **matrix 3 service** (`make build svc=`); `buf lint` + proto drift (`buf-gen` + git diff); **parity swagger** (python: 8 module HTTP + notification worker-only exempted, fail nếu thiếu, min 50 paths); job `image-build` matrix (no push); giữ workspace jobs; **không build monolith** (FR-020, FR-042)
- [ ] T114 Chạy gate đầy đủ `make wire && make lint && make test-race`; kiểm tra portability (không vendor lock-in ngoài pgvector/routing) (FR-007); xác nhận build/push/deploy độc lập 3 service + image rollback monolith (SC-017)
- [x] T116 [P] Cập nhật `docs/project/funding-and-platform-lifecycle.md`: Supabase **1 database dùng chung** (không tách schema staging/prod) + **teardown là một bước trong repatriation**; đồng bộ với spec/ADR 021 — **verified: doc đã ghi cả hai điểm, không cần sửa**

---

## Phase 9: Cutover thủ công (owner-triggered) — FR-041

**Purpose**: Runbook + drill cho việc đưa `api.closy` sang Cloud Run; **thực thi khi owner quyết định** `api-v2` đủ ổn định.

- [ ] T117 Drill cutover + rollback trên môi trường Cloud (đổi Nginx `api.closy` → Cloud Run, xác minh, rồi rollback về monolith); đo thời gian; ghi lại bước lệch
- [ ] T115 Verify quickstart VS1–VS10 end-to-end; ghi kết quả vào `specs/020-platform-migration/quickstart.md` (phụ lục kết quả) — **sau T117** (VS9 là cutover drill)
- [ ] T118 (owner-triggered) Thực thi cutover `api.closy` → Cloud Run theo runbook khi owner quyết định; verify đầy đủ; giữ đường rollback về monolith; chỉ đổi Nginx/DNS + đồng bộ dữ liệu (không sửa mã nghiệp vụ)

---

## Dependencies & Execution Order

### Critical Path & Cross-Phase Ordering (authoritative)

> Thứ tự dưới đây **thắng** thứ tự số phase khi mâu thuẫn. Task giữ nguyên ID; chỉ đổi vị trí thực thi.

1. **Mở đường rollback**: `T097` (compose profile `rollback`) → `T097c` (workflow deploy monolith từ tag `monolith-legacy`) → `T034` (xoá `cmd/server`). T097 phải xong trước T097c vì T097c dùng profile do T097 định nghĩa; T097c phải xong trước T034 để giữ đường deploy/rollback monolith (FR-056).
2. **Ranh giới service**: `T035–T038` (gRPC client, [P]) → `T039` (gRPC server) → `T040–T042` (Wire composition 3 service). `T121` (gỡ import implementation xuyên service ở `commerce/providers` + `ai/providers`) **chỉ sau T040–T042 và gắn với T034** (monolith `internal/di` đang tiêu thụ các package này; gỡ sớm phá build xanh).
3. **Runtime topology**: `T040–T042` → `T034a` (tách bootstrap theo `APP_MODE`) → `T123` (bỏ hardcode `8080`) → `T034b` (worker singleton + advisory lock).
4. **Quan trắc**: `T119` (log JSON) + `T120` (metrics/alert) phải xong **trước `T074`** (deploy) và **trước `T082`/`T117`** (gate đo được FR-041).
5. **Deploy managed**: `T075` (đặt billing cap + cap autoscale) → `T074` (deploy 6 units) → `T073a`/`T073b` (verify gRPC HTTP/2, media, health). `T074` cần `T034a/T034b` xong.
6. **Ingress**: `T070` (mở rộng cert) → `T069` (server block Nginx) → `T068` (bản ghi DNS Vercel) — tránh mở traffic khi cert/route chưa sẵn sàng.
7. **Cutover**: `T117` (drill) → `T115` (verify quickstart VS1–VS10, gồm VS9) → `T118` (owner-triggered).

### Phase Dependencies

- **Setup (Phase 1)**: không phụ thuộc (T006 tag legacy làm sớm để frozen; T006a/T004/T004a/T004b/T007a chuẩn bị container + registry).
- **Foundational (Phase 2)**: phụ thuộc Setup; BLOCKS mọi user story.
- **US1 (Phase 3, P1)** và **US2 (Phase 4, P1)**: sau Foundational; độc lập nhau.
- **US5 (Phase 5, P1)**: sau Foundational; proto T026–T029 song song; split module trước adapters; T121 **sau T040–T042 và gắn với T034** (giữ build monolith xanh; xem Critical Path); T034a/T034b sau T040–T042; T049a (rate-limit Redis-backed) sau T049.
- **US3 (Phase 6, P1)**: sau Foundational + US5 (cần 3 service image để deploy). T119/T120 (quan trắc) chạy trước T074; thứ tự T070→T069→T068; T075 trước T074; T073a/T073b sau T074. **Không có cutover** trong phase này.
- **US4/US6/US7 (Phase 7, P2)**: sau US3 (cần route map + image 3 service); runbook viết song song từ Phase 6; T097 + T097a + T097c phải xong **trước khi teardown AR** và **trước T034**; T097b (workflow deploy 3-service VPS) là đường deploy chính khi repatriation; T122 (ngừng VPS + thu hồi credential, FR-016) sau T098/T101/T102.
- **Polish (Phase 8)**: sau các story mong muốn (T115 đã chuyển sang Phase 9, chạy sau T117).
- **Cutover thủ công (Phase 9)**: sau Phase 7/8; T117 (drill) bắt buộc trước T115 và T118; T118 **owner-triggered** (chỉ chạy khi owner quyết định `api-v2` ổn định).
- **Convergence (Phase 10)**: không còn task tồn đọng; T119/T120 đã về **Phase 6 (mục Quan trắc)**, T121 và T123 về **Phase 5**, T122 về **Phase 7** — xem "Critical Path & Cross-Phase Ordering".

### Within Each User Story

- Tests trước implementation (TDD cho hành vi cốt lõi); **ngoại lệ US5**: T055/T056 chạy sau T040–T042 vì contract behavior test cần cả hai adapter đã tồn tại.
- Proto trước stubs trước adapters trước composition.
- Route map Nginx trước khi bật `api-v2`; smoke test đủ 3 service.
- Backup/verify trước mọi thao tác dữ liệu production.
- Migration per-service (T077) phải xong trước khi deploy service lần đầu (Account phải chạy trước AI/Commerce vì giữ toàn bộ migration lịch sử).
- Cutover `api.closy` (T118) chỉ sau khi T117 drill pass và owner xác nhận `api-v2` ổn định; luôn giữ đường rollback về monolith.

### Parallel Opportunities

- T002/T003/T004/T004a/T005/T008 song song; T004b sau T004; T007a sau T007.
- Build/push/deploy **độc lập từng service** (T094/T094a) — 3 service không chặn nhau.
- Proto T026–T029 song song; client adapters T035–T038 song song sau T030.
- Swagger T050–T052 song song; docs T057–T063 song song.
- Runbook T083–T093 song song hoàn toàn.
- Docs US5 + runbook US3 có thể song song sau Foundational.
- T069/T070 (Nginx server block + cert) song song với US5; T068 (DNS) sau T069/T070.
- T119/T120 (quan trắc) song song với T064–T067 (TLS) và chạy trước T074.
- T123 (bỏ hardcode 8080) song song sau T034a; T122 (ngừng VPS + thu hồi credential) song song với T105–T108.

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Phase 1 + Phase 2.
2. US1 (ES off) + US2 (pooler resilience).
3. Validate VS1/VS2 → hệ thống chạy được trên managed data layer với ES tắt. **Đây là MVP vận hành.**

### Incremental Delivery

1. MVP (US1+US2) → managed data layer.
2. US5 (3 service module C) → image 3 service.
3. US3 (ingress Nginx + Cloud Run + migration on-start) → `api-v2` chạy song song; `api.closy` không đổi.
4. US4/US6/US7 (VPS 3-service + repatriation gồm teardown) → đường về sẵn sàng.
5. Phase 9 (T117 → T115 → T118): runbook + drill cutover; **thực thi cutover `api.closy` khi owner quyết định** `api-v2` ổn định.
6. Funding clock → repatriation. Polish.

### Notes

- `monolith-legacy` là tag frozen; KHÔNG build monolith trong CI per-commit, KHÔNG nhận code mới. Chỉ build **image rollback MỘT LẦN** (T097a) trước khi hết cửa sổ funded để giữ nhánh rollback (FR-043).
- **Thứ tự tới hạn**: `T097` → `T097c` (workflow deploy monolith từ tag) MUST xong **trước `T034`** (xoá `cmd/server`) — nếu không sẽ mất đường deploy/rollback monolith (FR-056). Xem "Critical Path & Cross-Phase Ordering".
- Ba service build/push/deploy **độc lập** (image riêng, tag theo SHA, CI matrix): không rebuild service khác, không deploy all-or-nothing (FR-042, SC-017).
- Mọi thay đổi route `api-v2` phải pass smoke test tới đủ 3 service trước khi merge; `api.closy` MUST NOT đổi routing.
- Backup + verify checksum trước mọi thao tác dữ liệu production.
- Không merge task nào vi phạm portability 2 chiều (§VIII).
- Toàn bộ config dùng chung `configs/`; không tạo nhánh config theo môi trường cho SSE/heartbeat.
- **Cross-service transport mapping (A′)**: enum/const gắn entity MUST ở `domain/constants/` (giữ chiều phụ thuộc §IV). Owner phơi mapper proto⇄DTO tại `services/<svc>/contract/<owner>/transport/` (ngoài `internal/`) để cả gRPC server nội bộ và `contracts/<owner>/v1/client` dùng chung; client MUST NOT tham chiếu enum domain. Áp dụng cho T036–T038.

---

## Phase 10: Convergence

**Purpose**: Truy vết kết quả `/speckit-converge` 2026-09-17. **Không còn task tồn đọng trong phase này** — các task phát hiện đã được đặt vào đúng phase thực thi để file đọc theo thứ tự thi công:

- `T119` (log JSON) + `T120` (metrics/alert Cloud Monitoring) → **Phase 6 — mục "Quan trắc"**, trước `T074`.
- `T121` (gỡ import implementation xuyên service) → **Phase 5, sau `T042`**, chỉ thực thi khi `T040–T042` xong **và `T034`** (xoá monolith) đã thực thi — nếu `T034` chờ `T097→T097c` thì `T121` chờ theo.
- `T122` (runbook ngừng VPS + thu hồi credential, FR-016) → **Phase 7**, sau `T102`.
- `T123` (bỏ hardcode cổng `8080`, SC-023) → **Phase 5**, sau `T034a`.

Tham chiếu thứ tự đầy đủ: "Critical Path & Cross-Phase Ordering".

---

## Phase 11: Convergence

**Purpose**: Truy vết kết quả `/speckit-converge` 2026-09-18 — rà codebase hiện tại với spec/plan/tasks.
Task đánh dấu **CRITICAL** phải xử lý trước.

- [x] T124 **CRITICAL** Tạo lại tag `monolith-legacy` frozen tại commit monolith-green cuối cùng (ngay trước commit 020 đầu tiên) + push tag; verify `git rev-parse --verify refs/tags/monolith-legacy` thành công và `deploy-production.yml` checkout được (hiện tag KHÔNG tồn tại local/remote dù T006 đánh dấu xong, làm `deploy-production.yml:51` fail và chặn nhánh rollback) — **verified: annotated tag `monolith-legacy` đã tạo tại `2359c7e1`** (parent của commit 020 đầu tiên `20fa0f98`, ancestor của `main`). **Lưu ý: theo yêu cầu owner chỉ tạo tag LOCAL, CHƯA push** — CI `deploy-production.yml` vẫn cần `git push origin monolith-legacy` mới checkout được per T006/FR-018/FR-021/Constitution I (contradicts)
- [x] T125 Chạy gate đầy đủ `make wire && make lint && make test-race` toàn workspace; xác nhận không drift `wire_gen.go` và test-race xanh — **verified: `make wire-all` không drift; `make lint` pass (gofmt + vet); `make test-race` EXIT=0 6 module**. Phát hiện & sửa 2 lỗi thật: (a) test cũ `TestRunMigrationsIfEnabledPassesThrough` không khớp semantics nil-FS của T077 → truyền `fstest.MapFS{}` để chạm nhánh `db==nil`; (b) `TEST_CMD`/`RACE_TEST_CMD` pipe qua `findstr`/`grep` khiến exit code là của filter — vừa false-fail module không có test (`token-helper`), vừa **che giấu test fail thật** → bỏ pipe per T114 (missing)
- [x] T126 Dọn thư mục rỗng còn sót `internal/modules/wardrobe/application/usecase/category/` (dấu vết monolith, 0 file) để hoàn tất việc xoá `internal/` — **verified: đã xoá `internal/`** per T034 (partial)
- [x] T127 Viết lại `deployments/docker-compose.yml` (local) sang topology 3 service + Nginx gateway, khớp `plan.md` "Local dev (3 service + infra local)"; bỏ monolith `backend` (build root `Dockerfile` đã xoá ở T034), bỏ Elasticsearch (mặc định tắt), ngrok trỏ `http://nginx:80`; mỗi service đặt `DATABASE_MIGRATION_TABLE` riêng (`goose_db_version` / `_commerce` / `_ai`) + `DATABASE_MIGRATE_ON_START=true` + host `postgres`/`redis`/`rabbitmq` + `INGRESS_*`; thêm `deployments/nginx/local.conf` (HTTP route map, SSE buffering off) — **verified: `docker compose config` hợp lệ** per plan.md:186 (missing)
- [x] T128 **HIGH** Wire `DATABASE_MIGRATION_TABLE` per-service ở deploy config managed/VPS: `deployments/docker-compose.prod.yml` (account=`goose_db_version`, commerce=`goose_db_version_commerce`, ai=`goose_db_version_ai`) + `deploy-platform.yml` (thêm dòng echo tương ứng theo `matrix.service`) + `deploy-vps-services.yml` nếu cần; hiện cả hai đều để trống nên 3 service dùng chung `goose_db_version`, trái T077/FR-035 (bảng version riêng) — **verified: prod compose set per-service `environment:`; `deploy-platform.yml` inject `DATABASE_MIGRATION_TABLE` theo matrix; `deploy-vps-services.yml` không cần sửa vì compose override áp dụng**. Validate: `docker compose -f docker-compose.prod.yml config` EXIT=0; 4 workflow parse YAML OK per T077/FR-035 (missing)
- [x] T129 Sửa `make run svc=<account|commerce|ai>` (README §Local Setup) bị hỏng sau tách module: (a) Windows không nhận path `../../bin/<svc>.exe` → `RUN_BIN` per-OS dùng backslash; (b) CWD `services/<svc>` làm mất `configs/config.yaml` + `.env` → Makefile export `CONFIG_FILE` (repo config), `SERVER_PORT`, `REDIS_HOST`/`RABBITMQ_HOST`/`DB_LISTENER_HOST` (localhost); (c) `loadLocalDotEnv` thêm đường dẫn repo-root `.env` qua config dir parent; README cập nhật cách chạy nhiều service trên host (override `SERVER_PORT` + `INGRESS_*`) — **verified: `make run svc=account` qua được config/env/port, chỉ còn lỗi Redis unreachable do hạ tầng chưa bật; `go test -race shared/config` + `make lint` xanh** per README:73 (missing)
- [x] T130 Chống lệch route map nginx thủ công (FR-028): thêm tool `shared/tools/nginx-check` đối chiếu 3 `swagger.json` với 3 route map (`api-v2-routes/*.conf`, `templates/vps-3service.conf.template`, `local.conf`), kiểm tra mỗi path route đúng service theo longest-prefix; `make nginx-check` (+ vào `make check`) + step CI trong job swagger-parity; test đơn vị cho `classify`/`winner`/`parseRouteFile`; **vá gap phát hiện được: `/api/v1/categories` (commerce, public) thiếu ở cả 3 route map → đã thêm** — **verified: `make nginx-check` OK 154/154 paths; `go test ./tools/nginx-check` + `make lint` xanh** per FR-028 (missing)
- [x] T131 Sinh route map tự động từ swagger (zero-touch, FR-028/FR-036): tool `shared/tools/nginx-routes` tính prefix tối thiểu theo owner (longest-prefix, tôn trọng ranh giới dùng chung `/api/v1/me` → `location =`), minimize prefix bị bao, và **tự phát `location` `proxy_buffering off` cho mọi path `/stream` + `/sse`** (SSE không còn là exception thủ công; bỏ regex chết trong `api-v2.conf.template` và bỏ SSE override tay ở `local.conf`/`vps-3service`); ghi toàn bộ `api-v2-routes/*.conf` và vùng giữa marker `BEGIN/END GENERATED ROUTES` trong `vps-3service.conf.template` + `local.conf`; `make nginx-routes`; CI drift-check `make nginx-routes && git diff --exit-code -- deployments/nginx`; `.gitattributes` ép `deployments/nginx/** eol=lf`; test đơn vị cho compute/minimize/SSE/render/replaceRegion — **verified: `make nginx-routes` idempotent (29 locations, 3 SSE), `make nginx-check` 154/154, `go test ./tools/...` + `make lint` xanh, ci.yml parse OK** per FR-028/FR-036 (missing)
- [x] T132 Đưa sinh route map vào `make generate`: `generate: wire-all swagger-all nginx-routes` (đảm bảo thứ tự swagger trước); `generate-check` mở rộng diff sang `deployments/nginx`; **sửa bug `swagger-all`** (`cd ../commerce`/`cd ../ai` tương đối sai — mỗi dòng recipe là shell riêng nên phải `cd services/<svc>` từ repo root) khiến `make generate` fail; giữ thông báo 404 tiếng Việt có dấu trong `local.conf`/`vps-3service` — **verified: `make generate` chạy trọn wire→swagger→nginx-routes; wire/swagger `git diff` sạch; `make nginx-check` 154/154** per FR-038 (missing)
- [x] T133 Sửa Swagger UI 404 (FR-030): (a) `swag init` v1.16.6 chỉ sinh `docs.go/swagger.json/swagger.yaml`, KHÔNG có `index.html` → router `c.File("./api/swagger/index.html")` 404; thay bằng `shared/presentation/swagger.go` `SwaggerUIHandler()` (Swagger UI CDN trỏ `/api-docs/swagger.json`) dùng cho cả 3 service; (b) image distroless `nonroot` có `WORKDIR=/home/nonroot` khiến `./api/swagger` trỏ sai (files ở `/api/swagger`) → thêm `WORKDIR /` vào runtime stage 3 Dockerfile; (c) Nginx fallback JSON thiếu `charset=utf-8` gây mojibake tiếng Việt → `default_type "application/json; charset=utf-8"` ở `local.conf` + `vps-3service` — **verified end-to-end: `/swagger` → 200 text/html, `/api-docs/swagger.json` → 200 cho 3 service (8101/8102/8103); Nginx 404 trả `Content-Type: application/json; charset=utf-8`; `make test-race` + `make lint` xanh** per FR-030 (missing)
- [x] T134 Swagger kiểu monolith tại gateway (FR-030): tool `shared/tools/swagger-combine` gộp 3 spec thành bundle (`swagger.json` all + `services/<svc>` + `modules/<module>` split theo mapping cũ + `index.html` **đúng UI cũ** từ git `api/swagger/index.html`, thêm dropdown **Service** lọc Module); account phục vụ bundle (`/api-docs` + `/swagger`) và Nginx gateway route `/swagger` + `/api-docs` → account ở cả `local.conf`, `vps-3service`, `api-v2.conf.template`; mỗi service có `index.html` riêng (UI cũ) cho port trực tiếp; `make swagger-combine` nằm trong `make generate`; `.gitattributes` eol=lf; bỏ UI tự chế ở `shared/presentation/swagger.go` — **verified: `:8080/swagger` → 200 HTML (có `service-select`), `:8080/api-docs/swagger.json` (all) + `/services/commerce` + `/modules/wardrobe` → 200, 8101/8102/8103 → 200; `make generate` trọn vẹn; `make nginx-check` 154/154; `make lint` xanh** per FR-030 (missing)
- [x] T135 **CRITICAL** Sửa lỗi jsonb `22P02`/`42601` do `simple_protocol` (FR-046): `driver.Valuer` trả `[]byte` bị pgx encode thành `bytea` → cột `jsonb` từ chối. Sửa `Value()` trả **string** cho: `shared JSONDocument` + `bodyProfile` + `preferredColors` (`shared/domain/entities/jsonb.go`), `admin JSONDocument` (`admin/domain/entities/admin_audit_log.go`), và **outbox** `json.RawMessage` (`Payload`/`Headers`) → type mới `repositories.JSONField`; `Scan` chấp nhận cả `string`/`[]byte`. Audit toàn bộ 14 field `gorm:"type:jsonb"` — không còn kiểu nào bind sai; unit test cho từng Valuer + Scan — **verified: script thực tế với Postgres local (simple_protocol): `string`/`JSONDocument`/`BodyProfile`/`PreferredColors`/`JSONField` insert OK (trước đó `[]byte` → 42601, `JSONDocument` → 22P02); app end-to-end: `POST /auth/forgot-password` → 200 + `outbox_records` ghi jsonb thành công, worker outbox/email success; `make test-race` + `make lint` xanh** per FR-046 (contradicts)
- [x] T136 Khắc phục 404 do Nginx local giữ config cũ trong bộ nhớ: `local.conf` là bind-mount nên `make nginx-routes`/`make generate` cập nhật trên đĩa nhưng container chỉ đọc lúc start → route mới (`/api/v1/ai/outfit-recommendations`, `/api/v1/categories`…) 404 dù service có route. Thêm `make nginx-reload-local` (nginx -t + reload container `nginx-closy-dev`), `nginx-routes` in nhắc nhở, README ghi chú — **verified: sau reload POST `/api/v1/ai/outfit-recommendations` → 401 (đúng, có route), GET → 405; các API khác 200** per FR-028 (missing)
- [x] T137 **CRITICAL** Mode-hoá Nginx VPS/Cloud Run (FR-028/FR-056): tách `deployments/nginx/templates/{parallel,repatriation,rollback}/` (mỗi mode đúng 1 block cho `api.closy` + 1 cho `api-v2`, hết trùng `server_name`); prod compose mount `templates/${NGINX_MODE}` + mount thêm `api-v2-routes/`; `NGINX_ENVSUBST_FILTER` bổ sung `ACCOUNT_RUN_URL/COMMERCE_RUN_URL/AI_RUN_URL/TOKEN_HELPER_URL` (trước đó api-v2 render ra literal → hỏng); thêm service `token-helper` (profile `parallel`); `make nginx-reload NGINX_MODE=<mode>` render đúng subdir + xoá conf cũ; `.env.production.example` thêm `NGINX_MODE/COMPOSE_PROFILES/*_RUN_URL/TOKEN_HELPER_*`; cập nhật path trong `nginx-routes`/`nginx-check` — **verified: `docker compose -f docker-compose.prod.yml config` OK cho cả 3 mode (volume source đúng subdir); mỗi mode 1 file/domain; `make nginx-check` 154/154; `go test ./tools/...` + `make lint` xanh**. Lưu ý: token-helper mint thật vẫn là T071 (chưa xong) per FR-028 (contradicts)
- [x] T138 Dọn & chuẩn hoá config (FR-005/FR-046): (a) `query_exec_mode` chuyển thành **pooler-only** — bỏ `simple_protocol` khỏi `configs/*.yaml`, `db.go` thêm `queryExecMode()` ép `simple_protocol` khi port 6543, direct 5432 dùng prepared (tránh lớp lỗi `[]byte`→jsonb); (b) đưa default vào configs: `database.migration_allow_out_of_order: true`, `ingress.auth_header`, `jwt.algorithm: HS256`, `ssl_mode: "${DB_SSLMODE}"`, `listener_ssl_mode: "${DB_LISTENER_SSLMODE}"` (DB_SSLMODE trước đây là biến chết); loader default `disable` cho 2 ssl mode; (c) bỏ biến dư/chết khỏi `.env*.example` (`ELASTICSEARCH_ENABLED`, `OUTBOX_LISTENER_ENABLED`, `DB_LISTENER_PORT`, `DB_LISTENER_SSLMODE`(env), `REDIS_TLS_ENABLED`, `RABBITMQ_TLS_ENABLED`, `RABBITMQ_MANAGEMENT_TLS_ENABLED`, `DATABASE_MIGRATE_ON_START`, `DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER`, `DATABASE_QUERY_EXEC_MODE`, `JWT_ALGORITHM`, `INGRESS_AUTH_HEADER`, `SERVER_PORT` trùng); `.env.production.example` thêm `INGRESS_*` service-name cho gRPC VPS — **verified: `go test ./config/... ./infrastructure/db/...` + `make test-race` + `make lint` xanh; rebuild local stack boot OK, jsonb outbox write OK với prepared; docs `08-bang-bien-moi-truong.md` cập nhật** per FR-005/FR-046 (missing)
- [x] T139 Đồng bộ `.env`/`.env.production` theo example + hoàn thiện tài liệu vận hành env (FR-005): (a) tái tạo `.env`/`.env.production` đúng cấu trúc 2 file example, giữ nguyên secret, thêm key mới default, không BOM, key set khớp 100%; (b) viết lại `docs/operations/ha-tang/08-bang-bien-moi-truong.md` thành tham chiếu đầy đủ: 3 profile, bảng biến theo nhóm, **giải thích từng biến để trống** (local optional vs VPS must-fill), danh sách GitHub secrets managed, biến đã bỏ/không dùng; (c) phát hiện & sửa gap managed: `deploy-platform.yml` thiếu `INGRESS_TOKEN_AUDIENCE` + `DB_HOST/DB_PORT/DB_LISTENER_HOST/DB_LISTENER_SSLMODE` + `DB_SSLMODE=require`; `database.port`/`listener_port` chuyển env-driven (`DB_PORT`/`DB_LISTENER_PORT` int override, default 5432) — **verified: YAML parse OK; `make lint` + `make test-race` xanh; `docker compose config` (local + prod) OK; key sets khớp example** per FR-005 (missing)
- [x] T140 Runbook step-by-step host (đóng gap tài liệu vận hành): thêm `docs/operations/ha-tang/00-first-time-deploy-step-by-step.md` (thứ tự bắt buộc: push tag `monolith-legacy` → GCP Artifact Registry/WIF/SA/token-helper IAM/billing cap → Supabase pgvector+schemas+`ALTER ROLE search_path`+pooler/direct DSN → Redis/CloudAMQP → GitHub Environments + danh sách secrets/vars → `gh workflow run deploy-platform.yml` → VPS cert standalone + `NGINX_MODE=parallel`/`COMPOSE_PROFILES=parallel` + DNS Vercel + smoke → gate); bổ sung env-driven `REDIS_PORT`/`RABBITMQ_PORT` (loader) + workflow echo `${REDIS_PORT:-6379}`/`${RABBITMQ_PORT:-5671}` + bắt buộc `DB_LISTENER_HOST` (`:?`); con trỏ chéo ở 00/05/10 — **verified: YAML parse OK; `go test ./config/...` + `make lint` xanh**. Known limitations ghi rõ ở §11 runbook (migrate managed đang dùng connection runtime/pooler) per FR-013/FR-015 (missing)
- [x] T141 **CRITICAL** Option A — tách connection migration sang direct (FR-035/FR-046): thêm `shared/infrastructure/db/direct.go` (`DirectDSN` dựng DSN direct từ `ListenerHost/ListenerPort/ListenerSslMode`, fallback runtime fields; `OpenDirectDB` pool nhỏ); `RunMigrationsIfEnabled` bỏ tham số `*gorm.DB`, **tự mở direct connection** (Goose + session advisory lock) và đóng sau migrate; `OutboxListener` + `JobLocker` refactor dùng chung `DirectDSN`; bootstrap 3 service gọi chữ ký mới (`app.go`); runtime query thường **vẫn pooler 6543**. Audit 3 nơi session-scoped (Goose session locker, `LISTEN/NOTIFY`, job advisory lock) đều direct; test `TestDirectDSN` + stub `openMigrationDB`/no-tree — **verified: `go test ./infrastructure/db/...` + build 3 service xanh; `make lint` + `make test-race` (6 module) EXIT=0; spec FR-035 + plan Structure Decision 7 cập nhật** per FR-035/FR-046 (contradicts)
