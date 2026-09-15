# Implementation Plan: Chuyển đổi hạ tầng sang nền tảng funded và tách 3 service (multi-module workspace)

**Branch**: `020-platform-migration` | **Date**: 2026-09-15 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/020-platform-migration/spec.md`
**ADR**: `docs/decisions/021-funded-platforms-and-repatriation.md`
**Constitution**: v3.0.0 (single service topology, multi-module workspace, portability)

## Summary

Di chuyển SmartWardrobe Backend từ VPS self-hosted sang bộ nền tảng được tài trợ trong 3 tháng
(Cloudflare Pro, Supabase Pro, CloudAMQP Tough Tiger, Redis Cloud Essentials; compute Cloud Run
trial + teardown), đồng thời tách mã nguồn thành 3 service triển khai độc lập trên multi-module
Go workspace (`go.work`) — **bỏ monolith khỏi `main`**, giữ tag `monolith-legacy` frozen.

Edge là Cloudflare Pro gắn vào **subdomain zone `api-v2.closy.hycat.online`** (NS-delegate từ
Vercel, không move zone gốc) + Worker edge-router; cutover `api.closy...` bằng SaaS custom
hostname. Hết 3 tháng: **repatriation bắt buộc toàn bộ về VPS** (3 service compose + Nginx
gateway + dữ liệu migrate ngược).

Cách tiếp cận kỹ thuật:
1. Bổ sung cờ cấu hình để cùng image chạy mọi môi trường (ES optional, TLS cache/broker,
   migrate-on-start, outbox listener tách DSN, gateway secret).
2. Tách thành 3 Go module service (trong `services/`) + module `shared` + module `contracts` trên
   `go.work`; ranh giới service do compiler
   enforce; hợp đồng liên service qua gRPC (4 proto domain), nội service in-process.
3. Route table edge định nghĩa một lần (single-source), sinh Worker routes (Cloud) và Nginx
   includes (VPS), CI check parity — để repatriation không mất routing.
4. Swagger generate theo từng service, hợp 3 service giữ đủ 9 section, CI parity check.
5. Bộ runbook 2 chiều viết song song từ bây giờ: lên Cloud, cutover, teardown trial,
   repatriation về VPS + funding clock T-30/14/7/1.

## Technical Context

**Language/Version**: Go 1.25.7, multi-module workspace (`go.work`)

**Primary Dependencies**: Gin (HTTP), Google Wire (DI), GORM + pgx (Postgres), go-redis/v9
(cache), amqp091-go (RabbitMQ), Goose v3 (migration), zap (logging),
`google.golang.org/grpc` + `google.golang.org/protobuf` + `buf` CLI (inter-service gRPC).

**Storage**:
- PostgreSQL (Supabase Pro, extension `pgvector`), 8 schema theo module, **1 project + 1 database
  dùng chung** cho môi trường Cloud (không tách schema staging/prod; drill dùng schema tạm).
  Runtime qua Supavisor pooler (transaction mode, 6543); migration (qua **migration job riêng**) +
  `LISTEN` qua direct connection (5432). `sslmode=require`.
- Redis Cloud Essentials 250MB — kết nối TLS (`rediss`), share trong 256 connections.
- RabbitMQ CloudAMQP Tough Tiger — `amqps` TLS, share trong 100 connections; topology
  (quorum queue + DLX + delivery-limit, **verify quorum trên shared plan**, fallback classic
  durable) do code tự khai báo.
- Elasticsearch: **optional, mặc định tắt** (`ELASTICSEARCH_ENABLED=false`), fallback database.

**Edge**:
- Cloudflare Pro subdomain zone `api-v2...` (NS-delegate) + Worker router (~30 longest-prefix
  rule, xử lý chồng lấn `/me/*`, `/admin/*`); WAF Managed + OWASP Core; 2 rate-limit rule;
  Worker tiêm `X-Gateway-Secret`.
- SaaS custom hostname cho `api.closy...` lúc cutover; Redirect Rule `api-v2` → `api.closy` sau đó.
- SSE/AI streaming qua Worker (passthrough): heartbeat 15s, `ai_chat_chunk_deadline_seconds` = 30s
  (thống nhất mọi môi trường), origin trả `X-Accel-Buffering: no`, fallback `run.app` khi vượt
  Workers Free; test stream thật trước cutover.

**Testing**: `make test`, `make test-race`; test adapter gRPC/in-process, cờ cấu hình,
fallback search, parity swagger (9 section), parity route Worker↔Nginx.

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
- Chỉ dùng 4 hạng mục funded + trial compute; không LB/Armor/DB-managed ngoài danh sách.
- Không vendor lock-in ngoài pgvector và routing edge (portability 2 chiều).
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
│   │   └── Dockerfile             # image riêng, KHÔNG dùng chung
│   ├── commerce/                  # MODULE service Commerce Core
│   │   ├── go.mod
│   │   ├── cmd/commerce/main.go
│   │   ├── internal/
│   │   │   ├── bootstrap/         # app lifecycle riêng của Commerce
│   │   │   ├── di/
│   │   │   └── modules/{wardrobe,fashion,brand,dashboard}/
│   │   ├── api/swagger/           # 4 module
│   │   └── Dockerfile             # image riêng
│   └── ai/                        # MODULE service AI Engine
│       ├── go.mod
│       ├── cmd/ai/main.go
│       ├── internal/
│       │   ├── bootstrap/         # app lifecycle riêng của AI
│       │   ├── di/
│       │   └── modules/aiexecutor/
│       ├── api/swagger/           # 1 module
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
├── edge/worker/                   # Cloudflare Worker router + routes.yaml (single-source)
├── configs/                       # YAML parameter hoá (dùng chung mọi môi trường)
├── deployments/
│   ├── docker-compose.yml         # Local dev (3 service + infra local)
│   └── docker-compose.prod.yml    # VPS: 3 service + infra + nginx gateway (repatriation)
└── docs/
    ├── operations/                # Runbook 2 chiều (tiếng Việt)
    └── system-design/microservices/
```

**Structure Decisions (chốt 2026-09-15)**:
0. **"Mô hình C"** = multi-module Go workspace (`go.work`): mỗi service một Go module riêng, service
   gom trong `services/`, `cmd/server` bỏ khỏi `main` (giữ tag `monolith-legacy` frozen), hợp đồng
   liên service ở module `contracts/`. Khác với single-module multi-binary (1 `go.mod`, nhiều `cmd/`).
1. **Gom service vào `services/`** để tách bạch deployable (`services/*`) với thư viện dùng chung
   (`shared/`), hợp đồng (`contracts/`) và edge (`edge/`). Module path khớp thư mục
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

**Adapter layout**:
- **Inbound** — gRPC **server** ở `services/<service>/internal/modules/<owner>/presentation/grpc/`,
  phụ thuộc **contract interface** (không inject usecase trực tiếp).
- **Outbound** — gRPC **client** ở `contracts/<owner>/v1/client/`, implements contract của owner.
- Swagger: mỗi service `swag init` trên các module nó chứa; docs index ở edge.

**Map phase plan ↔ tasks**: Phase A ≈ tasks Phase 1–2; Phase B ≈ Phase 5; Phase C ≈ Phase 6;
Phase D ≈ Phase 6; Phase E ≈ Phase 5/8; Phase F ≈ Phase 7.

## Phase Plan (tóm tắt — chi tiết ở tasks.md)

- **Phase A — Config portability (P1)**: ES flag + no-op + skip sync; TLS Redis/RabbitMQ;
  `migrate_on_start`; outbox listener tách DSN + non-fatal/poll fallback; gateway secret;
  parameter hoá configs; hạ `ai_chat_chunk_deadline_seconds` 180s → 30s + heartbeat 15s
  (thống nhất mọi môi trường); entry-point test cho workspace.
- **Phase B — Tách 3 service module C (P1, blocked bởi A)**: `go.work` + module layout;
  `contracts/` + buf; adapter gRPC; composition `cmd/<service>`; JWT bất đối xứng; origin-auth
  middleware (verify `X-Gateway-Secret`); CI build 3 binary (compiler-boundary, không
  manifest-check); tag `monolith-legacy`.
- **Phase C — Edge + Compute (P1)**: NS-delegate + zone Pro + Worker router (single-source
  route table); WAF/rate-limit; Cloud Run HTTP+worker + billing cap; **migration job riêng**;
  SaaS custom hostname (chuẩn bị, chưa bật); SSE qua Worker passthrough (heartbeat 15s,
  chunk deadline 30s).
- **Phase D — Data + Cutover (P1, mốc chấm bài)**: Supabase **1 database dùng chung** + migration
  job; CloudAMQP/Redis wiring + verify giới hạn gói; runbook cutover; drill bằng schema tạm;
  dump VPS → Supabase tại merge/cutover; cutover `api.closy` trước ngày chấm.
- **Phase E — Docs & Runbook 2 chiều (P2, song song)**: `docs/operations/*` (lên Cloud,
  cutover, teardown, repatriation), ADR 021 (xong), microservices guides, README/local-setup.
- **Phase F — Repatriation gồm teardown trial (P2, theo funding clock)**: teardown trial + bill
  verify (một bước trong repatriation); compose VPS 3-service + Nginx gateway (từ route
  single-source); migrate ngược; DNS về VPS; tắt funded items trước renew; drill VS5/VS8.

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
