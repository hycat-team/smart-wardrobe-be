# Implementation Plan: Chuyển đổi hạ tầng sang nền tảng funded và tách 3 service (multi-module workspace)

**Branch**: `020-platform-migration` | **Date**: 2026-09-15 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/020-platform-migration/spec.md`
**ADR**: `docs/decisions/021-funded-platforms-and-repatriation.md`
**Constitution**: v3.0.0 (single service topology, multi-module workspace, portability)

## Summary

Di chuyển SmartWardrobe Backend sang kiến trúc 3 service độc lập trên **Cloud Run** (multi-module Go
workspace, `go.work`; bỏ monolith khỏi `main`, giữ tag `monolith-legacy` frozen), phục vụ qua **VPS
Nginx hiện hữu làm ingress**, chạy **song song** với monolith đang phục vụ `api.closy...`.

`api-v2.closy.hycat.online` → Vercel DNS → **VPS Nginx** → 3 Cloud Run (`run.app`), xác thực bằng
**Cloud Run IAM** (Google ID token gửi ở `X-Serverless-Authorization`; token do token helper trên VPS
sinh/cache). `api.closy...` → VPS Nginx → `http://backend:8080` (monolith) — **giữ nguyên**; cutover
`api.closy...` sang Cloud Run là **thao tác thủ công có runbook**, timing do owner quyết định dựa trên
độ ổn định của `api-v2`. Cloudflare Pro là hạng mục funded nhưng **không nằm trong đường API**.
Data: Supabase Pro (1 database dùng chung) + CloudAMQP + Redis Cloud; compute Cloud Run trial +
billing cap + teardown. Hết 3 tháng: repatriation về VPS (3 service compose + Nginx + migrate ngược).

Cách tiếp cận kỹ thuật:
1. Bổ sung cờ cấu hình để cùng image chạy mọi môi trường (ES optional, TLS cache/broker,
   migrate-on-start, outbox listener tách DSN).
2. Tách thành 3 Go module service (trong `services/`) + module `shared` + module `contracts` trên
   `go.work`; ranh giới service do compiler enforce; hợp đồng liên service qua gRPC (4 proto domain),
   nội service in-process.
3. **Ingress = Nginx hiện hữu**: thêm server block `api-v2` route tới 3 Cloud Run; `api.closy` giữ
   nguyên route monolith. Route map single-source trong Nginx + smoke test.
4. **Ingress auth**: token helper trên VPS sinh/cache ID token (audience = `run.app` URL); Nginx
   inject `X-Serverless-Authorization`; không bật `allUsers`.
5. Swagger generate theo từng service, hợp 3 service giữ đủ 9 section, CI parity check.
6. Bộ runbook song song từ bây giờ: lên Cloud (api-v2), teardown trial, repatriation về VPS +
   funding clock T-30/14/7/1.

## Technical Context

**Language/Version**: Go 1.25.7, multi-module workspace (`go.work`)

**Primary Dependencies**: Gin (HTTP), Google Wire (DI), GORM + pgx (Postgres), go-redis/v9
(cache), amqp091-go (RabbitMQ), Goose v3 (migration), zap (logging),
`google.golang.org/grpc` + `google.golang.org/protobuf` + `buf` CLI (inter-service gRPC).

**Storage**:
- PostgreSQL (Supabase Pro, extension `pgvector`), 8 schema theo module, **1 project + 1 database
  dùng chung** cho môi trường Cloud (không tách schema staging/prod; drill dùng schema tạm).
  Runtime qua Supavisor pooler (transaction mode, 6543); **migration do chính service chạy khi khởi
  động** + `LISTEN` qua direct connection (5432), có **advisory lock dùng chung**. `sslmode=require`.
  - **Migration ownership**: 24 migration lịch sử thuộc `services/account/database/migrations/`;
    migration mới đặt trong `services/<service>/database/migrations/` của service phát sinh.
  - **Bảng version riêng mỗi service**: Account giữ `goose_db_version` (đã có 24 lịch sử);
    Commerce/AI dùng `goose_db_version_commerce` / `goose_db_version_ai` → tránh lệch ordering.
  - Bật `goose.WithAllowOutofOrder(true)` (chịu migration merge muộn trong cùng service); advisory
    lock dùng chung (cùng lock id) để serialize.
- Redis Cloud Essentials 250MB — kết nối TLS (`rediss`), share trong 256 connections.
- RabbitMQ CloudAMQP Tough Tiger — `amqps` TLS, share trong 100 connections; topology
  (quorum queue + DLX + delivery-limit, **verify quorum trên shared plan**, fallback classic
  durable) do code tự khai báo.
- Elasticsearch: **optional, mặc định tắt** (`ELASTICSEARCH_ENABLED=false`), fallback database.

**Ingress (VPS Nginx hiện hữu)**:
- `api.closy...` → `http://backend:8080` (monolith) — **giữ nguyên**, không đổi routing.
- `api-v2.closy...` → `https://<service>.run.app` (3 Cloud Run), route longest-prefix (xử lý chồng
  lấn `/me/*`, `/admin/*`).
- **DNS**: bản ghi trên **Vercel** trỏ về VPS (A/CNAME). Không NS-delegate, không Cloudflare DNS.
- **Auth**: Cloud Run IAM (`roles/run.invoker`, không `allUsers`); Nginx inject ID token qua
  `X-Serverless-Authorization` (token helper trên VPS, audience = `run.app` URL); `Authorization`
  (Closy JWT) giữ nguyên.
- **TLS**: Nginx terminate cho cả 2 hostname; cert Let's Encrypt mở rộng gồm `api-v2`.
- **Không** dùng Cloudflare / Load Balancer / API Gateway trong đường API.
- **SSE**: Nginx tắt buffering; heartbeat 15s; `ai_chat_chunk_deadline_seconds` = 30s.

**Testing**: `make test`, `make test-race`; test adapter gRPC/in-process, cờ cấu hình,
fallback search, parity swagger (9 section), smoke test route `api-v2` tới 3 service qua Nginx, và
smoke test request thiếu ID token bị Cloud Run từ chối.

**Target Platform**:
- Managed (3 tháng): Cloud Run `asia-southeast1` (HTTP scale-to-zero; worker min-instances≥1,
  CPU always) + trial billing cap + teardown.
- Self-hosted: Docker Compose trên VPS — **3 service** (song song legacy monolith lúc đầu,
  repatriation sau 3 tháng) + Nginx gateway + certbot.
- Local: Docker Compose.

**Project Type**: web-service, multi-module workspace (3 service modules + shared).

**Performance Goals**: giữ nguyên hành vi API; search fallback giữ timeout hiện có; cutover ≤ 2h,
repatriation ≤ 4h (gồm migrate ngược) trong maintenance window.

**Constraints**:
- Chỉ dùng 4 hạng mục funded + trial compute; **không** LB / API Gateway chỉ để làm ingress.
- Không vendor lock-in ngoài pgvector và routing Nginx (portability 2 chiều).
- Mọi tham số hạ tầng qua biến môi trường; không hardcode host/endpoint.
- App khởi động dù `LISTEN` không khả dụng; ES tắt không sinh lỗi hàng loạt.
- CI build 3 service binary; tag `monolith-legacy` frozen, không nhận code mới.

**Scale/Scope**: 9 module nghiệp vụ + shared; 23 worker nền; 3 service
(Account, Commerce Core, AI Engine); 1 project Supabase **1 database dùng chung**; funding clock 3 tháng.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status |
|-----------|------|--------|
| §I Services, multi-module | 3 service modules + `shared` + `contracts` trên `go.work`; monolith xóa khỏi main, tag legacy frozen | ✅ PASS |
| §II Module Boundaries | Nội service in-process qua contract; liên service gRPC; compiler chặn import xuyên service | ✅ PASS |
| §III Event-Driven + Outbox | Giữ transactional outbox; tách DSN listener, poll fallback | ✅ PASS |
| §IV Clean Architecture | Không đổi layering; adapter gRPC ở presentation/infrastructure; client ở `contracts/<owner>/v1/client` | ✅ PASS |
| §V Transaction & Error | Giữ `apperror` + `IUnitOfWork`; Goose + session lock | ✅ PASS |
| §VI AI Pipeline | `aiexecutor` giữ logic; composition sang service AI Engine | ✅ PASS |
| §VII Design Patterns | Tuân thủ docs/system-design; bổ sung docs mới | ✅ PASS |
| §VIII Portability | Env-driven, TLS toggle, không Supabase Auth/RLS/Storage; cùng image chạy mọi môi trường | ✅ PASS |
| Governance | Atomic steps; `make wire && make lint && make test-race`; constitution 3.0.0 | ✅ PASS |

**Kết luận gate**: Không có vi phạm cần biện luận → bảng Complexity Tracking để trống. (Vòng đời kinh phí/funding clock xem `docs/project/funding-and-platform-lifecycle.md`, không thuộc hiến pháp.)

## Project Structure

### Documentation (this feature)

```text
specs/020-platform-migration/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output (VS1–VS8)
├── contracts/           # Phase 1 output (inter-service contracts + origin-auth)
└── tasks.md             # Phase 2 output
```

### Source Code (repository root — mô hình C, gom trong `services/`)

```text
smart-wardrobe-be/
├── go.work                        # Workspace: services/{account,commerce,ai}, shared, contracts
├── services/                      # Nhóm deployable (phân biệt rõ với thư viện dùng chung)
│   ├── account/                   # MODULE service Account (module path smart-wardrobe-be/services/account)
│   │   ├── go.mod
│   │   ├── cmd/account/main.go    # entry point (Wire composition)
│   │   ├── internal/
│   │   │   ├── bootstrap/         # app lifecycle riêng của Account
│   │   │   ├── di/                # composition root riêng của Account
│   │   │   └── modules/{identity,subscription,admin,notification}/
│   │   ├── api/swagger/           # Swagger per-service (4 module)
│   │   ├── database/migrations/   # 24 migration lịch sử (Account giữ) + migration mới của Account
│   │   └── Dockerfile             # image riêng, KHÔNG dùng chung
│   ├── commerce/                  # MODULE service Commerce Core
│   │   ├── go.mod
│   │   ├── cmd/commerce/main.go
│   │   ├── internal/
│   │   │   ├── bootstrap/         # app lifecycle riêng của Commerce
│   │   │   ├── di/
│   │   │   └── modules/{wardrobe,fashion,brand,dashboard}/
│   │   ├── api/swagger/           # 4 module
│   │   ├── database/migrations/   # migration mới do Commerce sở hữu
│   │   └── Dockerfile             # image riêng
│   └── ai/                        # MODULE service AI Engine
│       ├── go.mod
│       ├── cmd/ai/main.go
│       ├── internal/
│       │   ├── bootstrap/         # app lifecycle riêng của AI
│       │   ├── di/
│       │   └── modules/aiexecutor/
│       ├── api/swagger/           # 1 module
│       ├── database/migrations/   # migration mới do AI sở hữu
│       └── Dockerfile             # image riêng
├── shared/                        # MODULE dùng chung (module path smart-wardrobe-be/shared)
│   ├── go.mod
│   ├── config/                    # loader, validation, config.go
│   ├── infrastructure/{db,caching,messaging,search,media,ai}/
│   ├── presentation/              # HTTP helper, SSE, middleware chung, wrap, error
│   ├── domain/                    # base entity, jsonb, vector, constants
│   ├── application/               # DTO/event/media dùng chung
│   └── pkg/                       # logger, utils, sliceutils
├── contracts/                     # MODULE generated contracts (module path smart-wardrobe-be/contracts)
│   ├── go.mod
│   └── <owner>/v1/                #   *.proto, *.pb.go, client/
├── configs/                       # YAML parameter hoá (dùng chung mọi môi trường)
├── database/init/                 # SQL khởi tạo hạ tầng local (KHÔNG phải migration)
├── deployments/
│   ├── docker-compose.yml         # Local dev (3 service + infra local)
│   ├── docker-compose.prod.yml    # VPS: 3 service + infra (repatriation)
│   ├── nginx/                     # Ingress config: server block api.closy (giữ nguyên) + api-v2 (route tới Cloud Run) + token-helper
│   ├── token-helper/              # helper sinh/cache Google ID token cho Cloud Run (chạy trên VPS)
│   └── rabbitmq/
├── buf.yaml / buf.gen.yaml        # buf tooling (sinh proto → contracts/)
├── Makefile                       # viết lại: target per-service + target workspace
└── docs/
    ├── operations/                # Runbook 2 chiều (tiếng Việt)
    └── system-design/microservices/
```

**Structure Decisions (chốt 2026-09-15)**:
0. **"Mô hình C"** = multi-module Go workspace (`go.work`): mỗi service một Go module riêng, service
   gom trong `services/`, `cmd/server` bỏ khỏi `main` (giữ tag `monolith-legacy` frozen), hợp đồng
   liên service ở module `contracts/`. Khác với single-module multi-binary (1 `go.mod`, nhiều `cmd/`).
1. **Gom service vào `services/`** để tách bạch deployable (`services/*`) với thư viện dùng chung
   (`shared/`), hợp đồng (`contracts/`) và ingress (`deployments/nginx`). Module path khớp thư mục
   (`smart-wardrobe-be/services/account`).
2. **Dockerfile tách riêng theo service** (`services/account/Dockerfile`,
   `services/commerce/Dockerfile`, `services/ai/Dockerfile`) — không dùng chung 1 file/ARG để mỗi
   service build độc lập thật sự.
3. **HTTP/SSE/middleware dùng chung đặt tại `shared/presentation`** (thay vì
   `internal/shared/presentation` cũ).
4. **Bootstrap tách theo service** (`services/<service>/internal/bootstrap`) — không để app
   lifecycle dùng chung trong `shared/`, vì mỗi service có tập worker/route riêng.
5. **`go.work` gồm 5 module**: `services/account`, `services/commerce`, `services/ai`, `shared`,
   `contracts`.
6. **Logic nghiệp vụ không di chuyển nội dung**: tách chỉ đổi vị trí thành
   `services/<service>/internal/modules/*` và composition root theo service; ranh giới service do
   compiler enforce (service module không thấy implementation của module thuộc service khác).
7. **Migration ownership**: 24 migration lịch sử đặt trong **Account**; migration mới đặt trong
   service phát sinh; mỗi service **tự chạy migration của mình khi khởi động**; **mỗi service một
   bảng version riêng** (Account giữ `goose_db_version`, Commerce/AI có bảng hậu tố) + allow-out-of-order;
   advisory lock dùng chung (cùng lock id) để serialize.
8. **Makefile viết lại cho multi-module**: xem mục "Tooling & Makefile" bên dưới.
9. **Ingress = VPS Nginx hiện hữu** (không Cloudflare/LB/API Gateway trong đường API): thêm server
   block `api-v2` route tới 3 Cloud Run; `api.closy` giữ nguyên; token helper trên VPS inject
   `X-Serverless-Authorization`; DNS ở Vercel; cert Let's Encrypt mở rộng gồm `api-v2`.

**Tooling & Makefile (bắt buộc viết lại — FR-038)**:

Makefile hiện tại gắn cứng monolith, phải thay thế các biến/target sau:

| Hiện tại (monolith) | Vấn đề sau khi tách | Thay bằng |
|---|---|---|
| `WIRE_PATH := ./internal/di` | chỉ có 1 composition root | `wire` chạy per-service: `wire svc=account` → `<service>/internal/di` |
| `SWAG_MAIN_FILE := ./cmd/server/main.go` | không còn `cmd/server` | per-service swagger: `swagger svc=account` → `services/<svc>/api/swagger` |
| `GO_BUILD_PACKAGE := ./cmd/server` | không còn cmd/server | `build svc=<name>` + `build-all` (3 binary) |
| `WIRE_PATH`/`swagger`/`build`/`run`/`dev` | gắn path monolith | per-service + workspace |
| `MIGRATION_DIR := ./database/migrations` | 1 thư mục duy nhất | per-service: `migrate-up svc=account`, `migrate-create svc=x name=y` |
| `tidy` / `vet` / `test` / `fmt` | dùng `./...` (1 module) | chạy trong `go.work` hoặc per-service (`go work sync`, `go test ./services/...`) |
| `hygiene-scan` (`./internal/tools/hygiene-scan`) | tool ở root, không thuộc module | chuyển tool vào một module (đề xuất `shared/tools` hoặc module riêng) |
| `test-integration` path `./internal/...` | path monolith | cập nhật theo `services/<svc>/...` |
| `docker-build` (root Dockerfile) | bị thay bởi 3 Dockerfile | `docker-build svc=<name>` build `services/<svc>/Dockerfile` |

Target mới tối thiểu: `wire svc=`, `swagger svc=`, `build svc=`, `build-all`, `test-workspace`,
`migrate-up svc=`, `migrate-create svc= name=`, `migrate-status svc=`, `buf-gen`, `buf-lint`,
`nginx-reload` (render + `nginx -t` + reload), `token-helper-build`, `check` (CI suite theo workspace).

**Adapter layout**:
- **Inbound** — gRPC **server** ở `services/<service>/internal/modules/<owner>/presentation/grpc/`,
  phụ thuộc **contract interface** (không inject usecase trực tiếp).
- **Outbound** — gRPC **client** ở `contracts/<owner>/v1/client/`, implements contract của owner.
- Swagger: mỗi service `swag init` trên các module nó chứa; docs index phục vụ qua Nginx.

**Map phase plan ↔ tasks**: Phase A ≈ tasks Phase 1–2; Phase B ≈ Phase 5; Phase C ≈ Phase 6;
Phase D ≈ Phase 6; Phase E ≈ Phase 5/8; Phase F ≈ Phase 7; Phase G ≈ Phase 9.

## Phase Plan (tóm tắt — chi tiết ở tasks.md)

- **Phase A — Config portability (P1)**: ES flag + no-op + skip sync; TLS Redis/RabbitMQ;
  `migrate_on_start` (per-service, bật cho managed); outbox listener tách DSN + non-fatal/poll
  fallback; parameter hoá configs; hạ `ai_chat_chunk_deadline_seconds` 180s → 30s + heartbeat 15s
  (thống nhất mọi môi trường); **viết lại Makefile cho multi-module (FR-038)**; entry-point test cho
  workspace.
- **Phase B — Tách 3 service module C (P1, blocked bởi A)**: `go.work` + module layout;
  `contracts/` + buf; adapter gRPC; composition `cmd/<service>`; JWT bất đối xứng; CI build 3 binary
  (compiler-boundary, không manifest-check); tag `monolith-legacy`.
- **Phase C — Ingress + Compute (P1)**: thêm server block Nginx `api-v2` route tới 3 Cloud Run
  (giữ nguyên `api.closy`); DNS record trên Vercel; mở rộng cert Let's Encrypt; **token helper** sinh
  ID token; Cloud Run HTTP+worker + billing cap + IAM (`roles/run.invoker`, không `allUsers`);
  per-service migration on-start (advisory lock); SSE qua Nginx (buffering off, heartbeat 15s,
  deadline 30s).
- **Phase D — Data + Song song (P1)**: Supabase **1 database dùng chung** + migration ownership
  (Account giữ lịch sử); CloudAMQP/Redis wiring + verify giới hạn gói; runbook song song; drill bằng
  schema tạm; smoke test `api-v2` qua Nginx. Cutover `api.closy` **không** nằm trong phase này.
- **Phase E — Docs & Runbook 2 chiều (P2, song song)**: `docs/operations/*` (lên Cloud, song song,
  teardown, repatriation), ADR 021 + ADR 023, microservices guides, README/local-setup.
- **Phase F — Repatriation gồm teardown trial (P2, theo funding clock)**: teardown trial + bill
  verify (một bước trong repatriation); compose VPS 3-service + Nginx; migrate ngược; DNS về VPS;
  tắt funded items trước renew; drill VS5/VS8.
- **Phase G — Cutover thủ công (owner-triggered; docs deliverable, sau E/F)**: viết runbook cutover
  `api.closy` → Cloud Run + rollback (FR-041); drill trên môi trường Cloud; **thực thi chỉ khi owner
  quyết định** `api-v2` đủ ổn định. Chỉ đổi Nginx/DNS + đồng bộ dữ liệu, không sửa mã nghiệp vụ.

## Complexity Tracking

> Không có vi phạm Constitution Check — bảng để trống.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none) | — | — |

## Post-Design Constitution Re-check

Sau Phase 1, các quyết định thiết kế vẫn nằm trong ranh giới hiến pháp 3.0.0:
- Ranh giới service do compiler enforce, không import xuyên service → §I/§II PASS.
- Không truy vấn xuyên module; shared DB tách schema, khác DSN theo môi trường → §VIII PASS.
- Repatriation path (yêu cầu kỹ thuật) → §VIII PASS; lịch trình funding clock ở `docs/project/funding-and-platform-lifecycle.md`.
- Không ngoại lệ cần biện luận. **Gate: PASS.**
