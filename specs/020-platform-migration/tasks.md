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

## Path Conventions (mô hình C)

- Service modules: `services/account/`, `services/commerce/`, `services/ai/` (mỗi module một `go.mod`, `cmd/<service>/main.go`, bootstrap/di riêng)
- Module paths: `smart-wardrobe-be/services/<service>`, `smart-wardrobe-be/shared`, `smart-wardrobe-be/contracts`
- Shared module: `shared/` (config, infrastructure, presentation, domain, application, pkg)
- Contracts: `contracts/<owner>/v1/` (proto + generated + client)
- Ingress: `deployments/nginx/` (server block `api.closy` giữ nguyên + `api-v2` route tới Cloud Run) + `deployments/token-helper/` (sinh/cache ID token)
- Docs: `docs/operations/`, `docs/system-design/`, `docs/decisions/`
- Spec artifacts: `specs/020-platform-migration/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Khởi tạo workspace, tooling, khung service, thư mục docs, chốt tag legacy.

- [ ] T001 Khởi tạo `go.work` + `go.mod` cho 5 module (`services/account`, `services/commerce`, `services/ai`, `shared`, `contracts`); tạo `services/<service>/database/migrations/` (Account sẽ giữ 24 migration lịch sử); `go work sync` pass
- [ ] T002 [P] Add gRPC/protobuf dependencies (`google.golang.org/grpc`, `google.golang.org/protobuf`) và `go mod tidy` từng module dùng gRPC
- [ ] T003 [P] Add buf tooling tại root: `buf.yaml` + `buf.gen.yaml` (output `contracts/`, plugin `protoc-gen-go`, `protoc-gen-go-grpc`)
- [ ] T004 [P] Tạo skeleton mỗi service trong `services/`: `services/account/cmd/account/main.go`, `services/commerce/cmd/commerce/main.go`, `services/ai/cmd/ai/main.go` + `services/<service>/internal/bootstrap` + `services/<service>/internal/di` (minimal boot, chưa routes) + `services/<service>/Dockerfile` **riêng từng service**
- [ ] T005 [P] Tạo skeleton `deployments/nginx/` (server block `api.closy` giữ nguyên + `api-v2` route tới 3 Cloud Run) và `deployments/token-helper/` (helper sinh/cache Google ID token)
- [ ] T006 Tag `monolith-legacy` tại commit baseline = **HEAD của `main` ngay trước commit 020 đầu tiên** (commit monolith-green cuối cùng); ghi chú frozen vào tag message
- [ ] T007 **Viết lại Makefile cho multi-module (FR-038)**: thay `WIRE_PATH`/`SWAG_MAIN_FILE`/`GO_BUILD_PACKAGE` gắn monolith bằng target per-service + workspace (`wire svc=`, `swagger svc=`, `build svc=`, `build-all`, `test-workspace`, `migrate-up svc=`, `migrate-create svc= name=`, `migrate-status svc=`, `buf-gen`, `buf-lint`, `nginx-reload`, `token-helper-build`); cập nhật `tidy`/`vet`/`test` theo `go.work`; di chuyển `hygiene-scan` tool vào một module; sửa `test-integration` path; `docker-build svc=`
- [ ] T008 [P] Tạo skeleton `docs/operations/` (runbook 2 chiều) và cập nhật `docs/system-design/microservices/` (bỏ nội dung monolith)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Mô hình cấu hình dùng chung cho mọi story. **PHẢI xong trước US1–US7.**

- [ ] T009 Add config fields trong `shared/config/config.go`: `Elasticsearch.Enabled`, `Database.MigrateOnStart` (per-service, bật ở managed), `Database.ListenerHost/ListenerPort/ListenerSslMode`, `Database.DrillSchema`, `Redis.TLSEnabled`, `RabbitMQ.TLSEnabled/ManagementTLSEnabled`, `Outbox.ListenerEnabled`, `Jwt.Algorithm/PrivateKey/PublicKey`, `Ingress.AccountBaseURL/CommerceBaseURL/AIBaseURL/TokenHelperURL/TokenAudience/AuthHeader`, `Server.SSE.AIChunkDeadlineSeconds/HeartbeatSeconds`
- [ ] T010 Boolean env override + validation trong `shared/config/loader.go` và `shared/config/validation.go` (JWT key theo algorithm; drill schema chỉ dùng khi chạy drill; `migrate_on_start` bật ở managed cho từng service; ingress URL/audience bắt buộc ở managed)
- [ ] T011 Parameterize `configs/config.yaml` và `configs/config.production.yaml`: thay host/endpoint literal bằng `${DB_HOST}`, `${REDIS_HOST}`, `${RABBITMQ_HOST}`, `${ELASTICSEARCH_ADDRESSES}`; đặt `server.sse.ai_chat_chunk_deadline_seconds: 30` và `server.sse.heartbeat_seconds: 15` **giống nhau ở cả hai môi trường**
- [ ] T012 Update `.env.example` và `.env.production.example` (ES enabled, TLS toggles, migrate_on_start, listener_*, drill_schema, JWT keys, ingress URLs/audience/token helper)

**Checkpoint**: 5 profile (local / vps-legacy / vps / managed / managed-drill) load được, workspace build xanh.

---

## Phase 3: User Story 1 - Tắt search engine mặc định (Priority: P1) 🎯 MVP

**Goal**: ES tắt qua cấu hình, search fallback DB, không sinh DLQ.
**Independent Test**: quickstart VS1.

### Tests for User Story 1

- [ ] T013 [P] [US1] Unit test ES disabled client: `IndexDocument`/`DeleteDocument` no-op, `Search` trả lỗi, không start health goroutine
- [ ] T014 [P] [US1] Unit test search fallback khi ES disabled (wardrobe item read usecase)

### Implementation for User Story 1

- [ ] T015 [US1] Guard `Enabled` trong search client (bỏ health goroutine, no-op index/delete, search lỗi)
- [ ] T016 [US1] Guard initial sync trong search_sync usecase (skip khi disabled)
- [ ] T017 [US1] Guard `RunInitialSyncWithRetry` start trong search_sync worker
- [ ] T018 [US1] Phơi trạng thái `elasticsearch` trong health endpoint
- [ ] T019 [US1] Regenerate Wire; xác nhận 3 service build

**Checkpoint**: US1 hoạt động độc lập; VS1 pass.

---

## Phase 4: User Story 2 - Chạy ổn định trên pooler (Priority: P1)

**Goal**: App boot dù `LISTEN` không khả dụng; outbox poll fallback; migration có kiểm soát.
**Independent Test**: quickstart VS2.

### Tests for User Story 2

- [ ] T020 [P] [US2] Unit test outbox listener no-op/non-fatal + poll fallback
- [ ] T021 [P] [US2] Unit test `migrate_on_start=false` bỏ qua migration

### Implementation for User Story 2

- [ ] T022 [US2] Refactor outbox listener: `NewOutboxListener` không fatal; trả no-op listener khi disabled/lỗi; dùng listener DSN riêng (direct 5432)
- [ ] T023 [US2] Gate `RunMigrations` theo `database.migrate_on_start` trong bootstrap
- [ ] T024 [US2] Regenerate Wire; xác nhận boot với listener disabled
- [ ] T025 [US2] Ghi lệnh migrate session-DSN vào runbook bảng biến môi trường

**Checkpoint**: US2 hoạt động độc lập; VS2 pass.

---

## Phase 5: User Story 5 - Tách 3 service module C (Priority: P1)

**Goal**: 3 service module build/deploy độc lập; ranh giới do compiler enforce; JWT bất đối xứng + ID token; origin-auth; swagger per-service đủ 9 section.
**Independent Test**: quickstart VS3 + VS4.

> Chỉ 4 domain contract vượt ranh giới service lên protobuf (Identity, Subscription, Admin, AI).
> Brand/Fashion/Wardrobe/Notification giữ in-process (xem contracts §5).

### Contracts (protobuf)

- [ ] T026 [P] [US5] `contracts/identity/v1/identity.proto` (`IdentityService` + `IdentityDashboardService`)
- [ ] T027 [P] [US5] `contracts/subscription/v1/subscription.proto` (`SubscriptionService` + `SubscriptionDashboardService`)
- [ ] T028 [P] [US5] `contracts/admin/v1/admin.proto` (`AdminService`)
- [ ] T029 [P] [US5] `contracts/ai/v1/ai.proto` (`AIOrchestratorService` + `AIDashboardService`)
- [ ] T030 [US5] Generate stubs (`make buf-gen`) vào `contracts/<owner>/v1/`; commit generated code; wire vào `generate-check`

### Module split + dual adapter

- [ ] T031 [US5] Chuyển module identity/subscription/admin/notification vào `services/account/internal/modules/` (giữ nguyên logic, đổi import path); đồng thời chuyển **24 migration lịch sử** từ `database/migrations/` sang `services/account/database/migrations/` (kèm `embed.go`)
- [ ] T032 [US5] Chuyển module wardrobe/fashion/brand/dashboard vào `services/commerce/internal/modules/`
- [ ] T033 [US5] Chuyển module aiexecutor vào `services/ai/internal/modules/`; tách `internal/shared/**` thành module `shared/` (đặt HTTP/SSE/middleware tại `shared/presentation`; bootstrap chuyển về `services/<service>/internal/bootstrap`)
- [ ] T034 [US5] Xóa `cmd/server` khỏi `main`; verify workspace không còn reference monolith
- [ ] T035 [P] [US5] gRPC client cho `IUserBriefContract` + `IUserDashboardReader` — `contracts/identity/v1/client/`
- [ ] T036 [P] [US5] gRPC client cho `IUserSubscriptionContract` + `IUserQuotaContract` + `ISubscriptionDashboardReader` — `contracts/subscription/v1/client/`
- [ ] T037 [P] [US5] gRPC client cho `IAuditLogContract` — `contracts/admin/v1/client/`
- [ ] T038 [P] [US5] gRPC client cho `IAIOrchestrator` + `IAICostDashboardReader` — `contracts/ai/v1/client/`
- [ ] T039 [US5] gRPC server (phụ thuộc **contract interface**, không inject usecase) — `services/<service>/internal/modules/<owner>/presentation/grpc/` của identity/subscription/admin/aiexecutor
- [ ] T040 [US5] Wire composition `services/account/` (host gRPC Identity/Subscription/Admin; không client ngoài)
- [ ] T041 [US5] Wire composition `services/commerce/` (client Identity/Subscription/Admin/AI + 3 read)
- [ ] T042 [US5] Wire composition `services/ai/` (host AI; client Identity/Subscription/Admin)
- [ ] T043 [US5] Verify compile-time: import thử implementation xuyên service và xác nhận build fail

### Ingress auth (Cloud Run IAM — thay origin-auth cũ)

- [ ] T044 [US5] **Bỏ** origin-auth middleware `X-Gateway-Secret`; auth do Cloud Run IAM ở ingress. Bảo đảm service KHÔNG yêu cầu header gateway secret và `Authorization` (Closy JWT) được giữ nguyên (FR-033)
- [ ] T045 [P] [US5] Unit test (không cần Cloud Run): service KHÔNG yêu cầu `X-Gateway-Secret`; `Authorization` (Closy JWT) đi qua nguyên vẹn. Smoke test end-to-end IAM (thiếu token bị từ chối) nằm ở **T073 (Phase 6)**

### Auth liên service (2 lớp)

- [ ] T046 [US5] JWT bất đối xứng: ký ES256 (identity) — jwtutils
- [ ] T047 [US5] JWT verify middleware mọi service + hỗ trợ legacy HS256 giai đoạn chuyển tiếp + tra cứu thu hồi phiên qua Redis blacklist dùng chung (FR-023)
- [ ] T048 [P] [US5] Test ký/verify ES256 + từ chối HS256 khi tắt legacy + test session revocation bị chặn qua Redis blacklist (FR-023)
- [ ] T049 [US5] Lớp hạ tầng: gRPC client gắn Cloud Run ID token (`run.invoker`) khi gọi service khác (FR-034)

### Swagger per-service (9 section)

- [ ] T050 [US5] `swag init` cho service Account (4 module) — output `services/account/api/swagger/`
- [ ] T051 [US5] `swag init` cho service Commerce (4 module) — output `services/commerce/api/swagger/`
- [ ] T052 [US5] `swag init` cho service AI (1 module) — output `services/ai/api/swagger/`
- [ ] T053 [US5] Makefile targets swagger per-service + CI assert hợp 3 service đủ 9 section (fail nếu thiếu)
- [ ] T054 [US5] Xóa pipeline swagger monolith cũ; cập nhật swagger_access_code theo từng service

### Tests hợp đồng

- [ ] T055 [P] [US5] Contract behavior test dùng chung in-process + client adapter (8 contract)
- [ ] T056 [P] [US5] Idempotency test quota/audit/daily-quota (chống trừ/ghi trùng khi retry)

### Documentation

- [ ] T057 [P] [US5] `docs/system-design/microservices/00-tong-quan-3-service.md` (mô hình C, bỏ monolith)
- [ ] T058 [P] [US5] `docs/system-design/microservices/01-ranh-gioi-va-contract.md` (bảng proto vs in-process)
- [ ] T059 [P] [US5] `docs/system-design/microservices/02-grpc-va-event.md` (gồm ID token + JWT hai lớp)
- [ ] T060 [P] [US5] `docs/system-design/microservices/03-adapter-va-composition.md`
- [ ] T061 [P] [US5] `docs/system-design/microservices/04-auth-asymmetric-jwt.md` (gồm ID token + JWT hai lớp)
- [ ] T062 [P] [US5] `docs/system-design/microservices/05-repo-va-ci.md`
- [ ] T063 [P] [US5] `docs/system-design/microservices/06-lo-trinh-va-checklist.md`

**Checkpoint**: 3 binary build; boundary compile-time verified; VS3/VS4 pass.

---

## Phase 6: User Story 3 - Ingress + Compute + Song song (Priority: P1)

**Goal**: VPS Nginx là ingress (api.closy giữ nguyên; api-v2 → 3 Cloud Run), DNS Vercel, TLS cho api-v2, Cloud Run IAM + token helper, Cloud Run trial có cap, per-service migration on-start, data wiring, SSE ổn định, runbook song song (không cutover).
**Independent Test**: quickstart VS6.

### Code (TLS + managed wiring)

- [ ] T064 [P] [US3] TLS cho Redis (`redis.tls_enabled`)
- [ ] T065 [P] [US3] TLS cho RabbitMQ (`amqps` + `TLSClientConfig`)
- [ ] T066 [US3] HTTPS cho management/gauge API broker
- [ ] T067 [US3] Test TLS toggle on/off (assert cấu hình client)

### Ingress (VPS Nginx + Vercel DNS + Cloud Run IAM)

- [ ] T068 [US3] Thêm bản ghi DNS `api-v2.closy.hycat.online` trên **Vercel** trỏ về VPS hiện tại (A hoặc CNAME); **không** NS-delegate, **không** Cloudflare (FR-029)
- [ ] T069 [US3] Thêm server block Nginx cho `api-v2`: route longest-prefix tới 3 Cloud Run `run.app` (xử lý chồng lấn `/me/*`, `/admin/*`); **giữ nguyên** block `api.closy` → `http://backend:8080`. Route map single-source; nguồn suy ra từ `internal/api/routes/*` (FR-028)
- [ ] T070 [US3] Mở rộng cert Let's Encrypt gồm `api-v2.closy.hycat.online` (`certbot --expand`/`--cert-name`) + xác nhận renew tự động (FR-040)
- [ ] T071 [US3] **Token helper** trên VPS: sinh/cache/refresh Google ID token (audience = `run.app` URL) cho service account least-privilege (`roles/run.invoker` trên đúng 3 service); Nginx `auth_request` + `auth_request_set` inject `X-Serverless-Authorization` (FR-033/FR-039)
- [ ] T072 [US3] SSE qua Nginx: `proxy_buffering off` cho path SSE, timeout đọc đủ lớn, heartbeat 15s, `ai_chat_chunk_deadline_seconds` = 30s; test stream AI dài thật (FR-036, SC-014)
- [ ] T073 [US3] Cloud Run IAM: tắt `allUsers`, grant `run.invoker`, verify audience = `run.app` URL; smoke test thiếu token bị từ chối (FR-033, SC-013); `proxy_ssl_server_name on` + resolve động, không pin IP (FR-037)

### Compute (Cloud Run trial + cap)

- [ ] T074 [US3] Deploy 6 units (3 HTTP scale-to-zero + 3 worker min=1) region `asia-southeast1`, request-based billing
- [ ] T075 [US3] Billing cap + alert + max-instances cap

### Data (đúng giới hạn gói funded)

- [ ] T076 [US3] Supabase: **1 database dùng chung** + credential riêng; pooler app / direct listener; verify `LISTEN` (FR-025)
- [ ] T077 [US3] **Per-service migration on-start (FR-035)**: mỗi service chạy Goose provider trên `services/<service>/database/migrations` khi khởi động, dùng **bảng version riêng** (`goose.WithTableName`: Account giữ `goose_db_version`, Commerce/AI dùng bảng hậu tố) + **`goose.WithAllowOutofOrder(true)`** + **advisory lock dùng chung** (session locker, cùng lock id); verify không chạy lại 24 migration lịch sử khi Account khởi động
- [ ] T078 [US3] CloudAMQP Tiger: vhost riêng; verify quorum queue (fallback classic durable + ADR nếu không hỗ trợ); share trong 100 connections, prefetch 5
- [ ] T079 [US3] Redis Cloud Essentials: TLS, share trong 256 connections; verify throughput/bandwidth cho tải demo
- [ ] T080 [US3] Drill runbook dùng `database.drill_schema` rồi xoá, không đụng schema chính (FR-027)
- [ ] T081 [US3] Dump dữ liệu VPS → Supabase cho `api-v2` + verify checksum; ghi rõ đây là **song song** (monolith vẫn là nguồn ghi chính), re-dump khi cần đồng bộ (US3 AC2)
- [ ] T082 [US3] Gate song song: xác nhận `api-v2` chạy ổn qua VPS Nginx; `api.closy` **0 thay đổi routing** (SC-015); ghi mốc

### Runbook song song + CI/CD

- [ ] T083 [P] [US3] `docs/operations/ha-tang/00-tong-quan.md`
- [ ] T084 [P] [US3] `docs/operations/ha-tang/01-supabase-postgres.md`
- [ ] T085 [P] [US3] `docs/operations/ha-tang/02-redis-cloud.md`
- [ ] T086 [P] [US3] `docs/operations/ha-tang/03-cloudamqp-rabbitmq.md`
- [ ] T087 [P] [US3] `docs/operations/ha-tang/04-compute-cloudrun.md`
- [ ] T088 [P] [US3] `docs/operations/ha-tang/05-vps-nginx-ingress-cloudrun.md` (server block api-v2, giữ nguyên api.closy)
- [ ] T089 [P] [US3] `docs/operations/ha-tang/06-elasticsearch-tat-mac-dinh.md`
- [ ] T090 [P] [US3] `docs/operations/ha-tang/07-bang-bien-moi-truong.md`
- [ ] T091 [P] [US3] `docs/operations/ha-tang/08-kien-truc-song-song-tong-quan.md`
- [ ] T092 [P] [US3] `docs/operations/ha-tang/09-chuyen-du-lieu-vps-supabase.md` (dump/restore + backfill + checksum)
- [ ] T093 [P] [US3] `docs/operations/ha-tang/10-dns-vercel-ci-cd.md` (bản ghi `api-v2` trên Vercel + workflow platform)
- [ ] T094 [US3] `.github/workflows/deploy-platform.yml` (build 3 image từ 3 Dockerfile riêng → Cloud Run HTTP + worker); giữ workflow VPS cho monolith song song
- [ ] T095 [US3] `docs/operations/nginx-ingress-cloudrun-auth.md` (route map single-source, token helper, rotate SA key, smoke test IAM)
- [ ] T096 [US3] `docs/operations/manual-cutover-api-closy-to-cloudrun.md`: runbook **cutover thủ công** `api.closy` → Cloud Run (điều kiện tiên quyết, đổi Nginx/DNS, đồng bộ dữ liệu, kiểm thử, **rollback về monolith**); ghi rõ timing do owner quyết định (FR-041)

**Checkpoint**: Runbook song song thực thi được; VS6 pass (api-v2 qua Nginx, api.closy không đổi).

---

## Phase 7: User Story 4 + 6 + 7 - VPS 3-service + Repatriation (gồm teardown) + funding clock (Priority: P2)

**Goal**: Đường về VPS là 3 service compose + Nginx gateway (không monolith); repatriation gồm teardown trial; funding clock có chủ.
**Independent Test**: quickstart VS5 + VS8.

- [ ] T097 [US4] `deployments/docker-compose.prod.yml` mới: 3 service + postgres+pgvector/redis/rabbitmq + nginx gateway + certbot
- [ ] T098 [US4] Nginx route map (repatriation) multi-host: **cả `api.closy` và `api-v2` → 3 service compose** (dùng chung route map, không còn monolith mặc định); monolith compose chỉ là **rollback tạm thời** nếu owner chọn; smoke test route
- [ ] T099 [US4] Nginx ingress auth khi self-host (không còn Cloud Run IAM) + rate-limit/body-limit/timeout/security-header/logging-request-ID (ghi rõ trong runbook repatriation)
- [ ] T100 [US4] ACME challenge local cho cả 2 host + certbot renew
- [ ] T101 [US4] `docs/operations/ha-tang/11-ve-lai-vps.md`: migrate ngược (Supabase→Postgres, drain AMQP, Redis snapshot/warmup), DNS về VPS, tắt funded trước renew
- [ ] T102 [US4] `docs/operations/repatriate-to-vps.md`: runbook repatriation đầy đủ 2 chiều, **bao gồm bước teardown trial** (viết song song từ bây giờ)
- [ ] T103 [US4] Env template profile `vps` (sslmode=disable, TLS off, migrate_on_start=true)
- [ ] T104 [US4] Kiểm thử local: 3 service profile vps hoạt động, không sửa code
- [ ] T105 [US6] `docs/operations/gcp-trial-teardown.md`: thứ tự xóa (Cloud Run → artifacts → alerts), verify không còn resource tính phí — **là một bước trong repatriation**, không tách rời
- [ ] T106 [US6] Drill teardown trên môi trường Cloud + đối chiếu billing trial về 0 (SC-010)
- [ ] T107 [US7] Funding clock T-30/T-14/T-7/T-1 vào plan + checklist ký xác nhận (backup verify, restore drill, DNS rehearsal)
- [ ] T108 [US7] Drill repatriation đầy đủ trên môi trường Cloud trước T-30; ghi kết quả + thời gian (mục tiêu ≤ 4h)

**Checkpoint**: Quay về VPS được; VS5 pass; teardown nằm trong repatriation; funding clock có chủ.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T109 [P] Update `README.md` (multi-module workspace, 3 service, funded platforms, ES optional, repatriation)
- [ ] T110 [P] Update `docs/development/local-setup.md` (compose 3 service local, managed environment)
- [ ] T111 [P] Update `docs/system-design/architecture.md` (mô hình C + ES optional + ingress Nginx + Cloud Run)
- [ ] T112 [P] Verify ADR links từ plan/spec: `docs/decisions/021-funded-platforms-and-repatriation.md` + `docs/decisions/023-vps-nginx-ingress-cloud-run-iam.md` (và ADR 022 email)
- [ ] T113 CI gate: thêm job build 3 binary service trên mỗi PR (`make build` workspace) + parity swagger 9 section (FR-020)
- [ ] T114 Chạy gate đầy đủ `make wire && make lint && make test-race`; kiểm tra portability (không vendor lock-in ngoài pgvector/routing) (FR-007)
- [ ] T115 Verify quickstart VS1–VS9 end-to-end; ghi kết quả vào `specs/020-platform-migration/quickstart.md` (phụ lục kết quả)
- [ ] T116 [P] Cập nhật `docs/project/funding-and-platform-lifecycle.md`: Supabase **1 database dùng chung** (không tách schema staging/prod) + **teardown là một bước trong repatriation**; đồng bộ với spec/ADR 021

---

## Phase 9: Cutover thủ công (owner-triggered) — FR-041

**Purpose**: Runbook + drill cho việc đưa `api.closy` sang Cloud Run; **thực thi khi owner quyết định** `api-v2` đủ ổn định.

- [ ] T117 Drill cutover + rollback trên môi trường Cloud (đổi Nginx `api.closy` → Cloud Run, xác minh, rồi rollback về monolith); đo thời gian; ghi lại bước lệch
- [ ] T118 (owner-triggered) Thực thi cutover `api.closy` → Cloud Run theo runbook khi owner quyết định; verify đầy đủ; giữ đường rollback về monolith; chỉ đổi Nginx/DNS + đồng bộ dữ liệu (không sửa mã nghiệp vụ)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: không phụ thuộc (T006 tag legacy làm sớm để frozen).
- **Foundational (Phase 2)**: phụ thuộc Setup; BLOCKS mọi user story.
- **US1 (Phase 3, P1)** và **US2 (Phase 4, P1)**: sau Foundational; độc lập nhau.
- **US5 (Phase 5, P1)**: sau Foundational; proto T026–T029 song song; split module trước adapters.
- **US3 (Phase 6, P1)**: sau Foundational + US5 (cần 3 service image để deploy). Riêng T068 (DNS + Nginx ingress) chạy được ngay sau Phase 2. **Không có cutover** trong phase này.
- **US4/US6/US7 (Phase 7, P2)**: sau US3 (cần route map + image 3 service); runbook viết song song từ Phase 6.
- **Polish (Phase 8)**: sau các story mong muốn.
- **Cutover thủ công (Phase 9)**: sau Phase 7/8; T117 (drill) bắt buộc trước T118; T118 **owner-triggered** (chỉ chạy khi owner quyết định `api-v2` ổn định).

### Within Each User Story

- Tests trước implementation (TDD cho hành vi cốt lõi).
- Proto trước stubs trước adapters trước composition.
- Route map Nginx trước khi bật `api-v2`; smoke test đủ 3 service.
- Backup/verify trước mọi thao tác dữ liệu production.
- Migration per-service (T077) phải xong trước khi deploy service lần đầu (Account phải chạy trước AI/Commerce vì giữ 24 migration lịch sử).
- Cutover `api.closy` (T118) chỉ sau khi T117 drill pass và owner xác nhận `api-v2` ổn định; luôn giữ đường rollback về monolith.

### Parallel Opportunities

- T002/T003/T004/T005/T008 song song.
- Proto T026–T029 song song; client adapters T035–T038 song song sau T030.
- Swagger T050–T052 song song; docs T057–T063 song song.
- Runbook T083–T093 song song hoàn toàn.
- Docs US5 + runbook US3 có thể song song sau Foundational.
- T068 (DNS + Nginx ingress) song song với US5.

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
5. Phase 9 (T117/T118): runbook + drill cutover; **thực thi cutover `api.closy` khi owner quyết định** `api-v2` ổn định.
6. Funding clock → repatriation. Polish.

### Notes

- `monolith-legacy` là tag frozen; KHÔNG build monolith trong CI, KHÔNG nhận code mới.
- Mọi thay đổi route `api-v2` phải pass smoke test tới đủ 3 service trước khi merge; `api.closy` MUST NOT đổi routing.
- Backup + verify checksum trước mọi thao tác dữ liệu production.
- Không merge task nào vi phạm portability 2 chiều (§VIII).
- Toàn bộ config dùng chung `configs/`; không tạo nhánh config theo môi trường cho SSE/heartbeat.
