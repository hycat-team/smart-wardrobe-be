# ARCHITECTURE & TECHNICAL DEVELOPMENT RULES

> Updated at **T111** for the 3-service topology (020). The modular-monolith
> baseline (`cmd/server`, root `internal/`) is retired; source of truth is
> `.specify/memory/constitution.md` v3.0.0 + `specs/020-platform-migration/`.

## I. SYSTEM ARCHITECTURE PARADIGM

The backend is implemented as **3 independently deployable Go services** on a
multi-module workspace (`go.work`) with **strict clean layer boundaries**
(detailed in Constitution §IV).

- **Three deployables:** Account (`identity,subscription,admin,notification`),
  Commerce Core (`wardrobe,fashion,brand,dashboard`), AI Engine (`aiexecutor`) —
  each with its own Go module, container, health check, config and scaling.
  Cross-service calls go through `contracts/` (gRPC) or DTO contracts, never
  domain entities. DAG: Commerce→Account, Commerce→AI, AI→Account.
- **Shared infrastructure:** Postgres (+pgvector), Redis, RabbitMQ, Cloudinary,
  external AI providers (and optional Elasticsearch) are initialized once per
  service binary and reused across modules through DI (Wire).
- **Entity ownership per module:** each module owns its persistent models under
  `services/<service>/internal/modules/<name>/domain/entities/`. Cross-module
  navigation properties (struct pointers to entities in other modules) are
  **strictly prohibited**.
- **No DDD aggregate/event model:** the codebase does not model aggregate roots, value objects, or domain events in a DDD sense. Cross-module integration is handled through contracts plus shared infrastructure.
- **Module boundaries are real, but uneven:** every business domain lives in
  `services/<service>/internal/modules/<name>`. The exact provider wiring shape
  (layered vs flattened) may differ per module, but the folder convention is consistent.
- **Service boundaries are compiler-enforced:** a service module cannot import
  implementation packages of another service's modules (see
  `docs/system-design/microservices/01-ranh-gioi-va-contract.md`).
- **Archived modules:** a module removed from runtime keeps its source tree but is excluded from DI wiring, route registration, and Swagger generation. Git history is the authoritative record for what was removed and why.

### Layer dependency graph

Within each module, dependencies follow a strict inward-directed layering:

```
presentation → application → domain ← infrastructure
```

- **presentation** (handlers, workers, contracts) depends on **application** (via use case interfaces). Never calls infrastructure or domain directly.
- **application** (use cases, mappers, DTOs) depends on **domain** (via repository interfaces). Never depends on infrastructure.
- **domain** (entities, repository interfaces, constants) has **ZERO dependencies** on any other layer.
- **infrastructure** (GORM repos, caching, messaging, AI clients) depends on **domain** (implements repository interfaces). Never called directly by presentation.

For full details and strict rules, see Constitution §IV.

---

## II. DIRECTORY LAYOUT CONVENTIONS

The top-level structure is stable. Service code lives in `services/<service>/`;
shared code in `shared/`; contracts in `contracts/`.

```text
root/smart-wardrobe-be
|
|-- go.work                       # services/account, services/commerce, services/ai, shared, contracts (+ token-helper)
|-- services/<service>/           # one Go module per service (account|commerce|ai)
|   |-- cmd/<service>/main.go     # Entry point (APP_MODE=http|worker|all)
|   |-- internal/
|   |   |-- bootstrap/            # App lifecycle, routes engine, gRPC hosting, workers
|   |   |-- di/                   # Composition root (Wire injector + wire_gen.go)
|   |   `-- modules/<module>/     # See §IV for the canonical module layout
|   |-- api/swagger/              # Generated per-service Swagger docs
|   |-- database/migrations/      # Goose migrations owned by this service
|   `-- Dockerfile                # Independent image (binary + own migrations)
|
|-- shared/                       # Cross-service library (own Go module)
|   |-- config/                   # Go configuration loader (env-driven)
|   |-- application/              # DTOs, events, constants, use-case interfaces
|   |-- domain/                   # Shared base entities, constants
|   |-- infrastructure/           # DB, caching, media, AI, messaging, search
|   |-- presentation/             # Shared HTTP helpers, SSE, middleware (auth, rate-limit, CORS, timeout, role, swagger lock, error handler)
|   |-- pkg/                      # Logger, utils (incl. jwtutils, errorutils)
|   `-- tools/                    # hygiene-scan, etc.
|
|-- contracts/<owner>/v1/         # *.proto, *.pb.go, client/ (+ grpcx helpers)
|
|-- configs/                      # YAML configuration files (parameterised, no hardcoded hosts)
|
|-- database/init/                # DB initialization scripts (not migrations)
|
|-- deployments/                  # docker-compose.yml (local), docker-compose.prod.yml (VPS + rollback), nginx/, token-helper/, monolith-legacy/, rabbitmq/
|
|-- docs
|   |-- project/                 # Vision, scope, glossary, changelog, funding lifecycle
|   |-- system-design/           # Architecture, database, auth, AI pipeline, microservices/
|   |-- api/                     # API specs per domain
|   |-- development/             # Setup, conventions, testing, branch strategy
|   |-- decisions/               # Architectural Decision Records (ADRs)
|   `-- archive/                 # Historical docs (only when explicitly referenced)
|
|-- Makefile
```

To discover the current service inventory:
```bash
ls services/
```

---

## III. SHARED DATA MODEL & INFRASTRUCTURE

### Shared base types

Only foundational types remain in `shared/domain/entities/`:

- `base.go` — `BaseEntity`, `AuditableEntity`, `SoftDeleteEntity` (embedded by all module entities)
- `jsonb.go` — `JSONDocument`, `BodyProfile`, `PreferredColors`
- `vector.go` — `Vector` (pgvector wrapper)

Each business module owns its specific entities under `services/<service>/internal/modules/<name>/domain/entities/`. Cross-module entity references are forbidden — modules that need data from another module must call its contract (see §VI).

### Shared infrastructure

The `shared/` package owns reusable integrations for:

- `infrastructure/db`: Postgres connection and unit of work
- `infrastructure/caching`: Redis connection
- `infrastructure/media`: Cloudinary media service
- `infrastructure/ai`: provider-agnostic AI service with primary/fallback config
- `infrastructure/messaging`: RabbitMQ client and event publishing
- `infrastructure/search`: Elasticsearch client
- `presentation`: shared HTTP response and SSE helpers

Modules must reuse these services instead of re-instantiating them locally.

---

## IV. MODULE ORGANIZATION RULES

### Canonical module layout

Each business domain lives under `services/<service>/internal/modules/<name>` and keeps clear separation between layers:

```text
services/<service>/internal/modules/<name>
|
|-- application
|   |-- dto                     # Request/response structs
|   |-- errors                  # Domain-specific error definitions
|   |-- interface
|   |   `-- usecase             # Use case interfaces (consumed by handlers, contract, workers)
|   |-- mapper                  # Entity/DTO ↔ response mappers
|   `-- usecase                 # Use case implementations
|
|-- domain
|   |-- constants               # Module-specific enum constants
|   |-- entities                # Persistent GORM models (owned by this module)
|   `-- repositories            # Repository interfaces
|-- infrastructure
|   |-- persistence             # GORM repository implementations
|   |-- caching                 # (optional) module-specific caching
|   |-- messaging               # (optional) module-specific messaging
|   |-- payment                 # (optional) payment adapter
|   |-- search                  # (optional) search adapter
|   `-- security                # (optional) security adapter
|-- presentation
|   |-- contract                # Cross-module service interfaces & implementations (DTOs only)
|   |-- handler                 # HTTP handlers (thin, delegate to use cases)
|   |-- middleware              # (optional) module-owned Gin middleware (e.g., identity AuthMiddleware)
|   |-- router                  # (optional) module self-registers its endpoint group via Init(api)
|   `-- worker                  # Background workers (optional)
`-- provider.go                 # Wire ProviderSet for the module
```

Not every sub-folder is required; a module includes only what it needs. However, `application/usecase`, `domain/repositories`, `infrastructure/persistence`, and `presentation/handler` are expected in every module.

### Wiring shape variance

Modules may adopt one of two provider wiring styles:

- **Layered** — separate `wire.NewSet(...)` per layer (`application`, `infrastructure`, `presentation`). `identity` is the reference module for this style.
- **Flattened** — a single `provider.go` aggregates repositories, use cases, handlers, and workers directly. `wardrobe`, `fashion`, and `brand` follow this style.

The `presentation/provider.go` file is optional; some modules wire handlers and workers directly from the module-level `provider.go`.

### Cross-module contracts

Modules expose their public service interfaces via a package under `presentation/contract/` (e.g., `internal/modules/identity/presentation/contract`). Contracts sit inside `presentation/contract/`, not at the module root or under `application/`. A module that needs services from another module depends on its contract package, not on its internal use cases directly.

**Rules:**
- Contracts expose **only DTOs** (defined in `application/dto/`), never domain entities.
- Package `presentation/contract/` must define the contract interface and implement it.
- The contract struct implementation must delegate execution to application use cases via usecase interfaces.
- Workers call use cases through interfaces defined in `application/interface/usecase/`.
- Repositories must not `.Preload()` or `.Joins()` entities from other modules.

**Strictly prohibited — never import:**
- Another module's `domain/entities/` (direct struct pointers or cross-module DB joins)
- Another module's `domain/repositories/` (repository interfaces)
- Another module's `infrastructure/persistence/` (repository implementations)

**Allowed cross-module imports (with rationale):**

| Import | Allowed | Rationale |
|--------|---------|-----------|
| `presentation/contract/` of other modules | ✅ | Application layer needs to call cross-module services. Contracts are the official interface boundary. |
| `application/dto` of other modules | ✅ | DTOs are shared data shapes. Direct import avoids duplication and keeps type safety across module boundaries. |
| `domain/constants/` of other modules | ✅ | Pure enum/const values without behavior. Duplicating them would cause maintenance overhead. Always reference the constant symbol — never hardcode the value. |

**Rationale:** These patterns are accepted because the codebase is a modular monolith where modules share the same deployment and business domain. Strict isolation via contracts-only would create excessive ceremony without practical benefit for intra-domain communication. The prohibition on importing entities, repositories, and persistence remains absolute to preserve data ownership and prevent tight coupling at the data layer.

### Non-standard directories

The canonical layout covers the most common folders. The following extensions exist in the codebase and are valid:

- `application/interface/` may contain sub-packages beyond `usecase/` (e.g., `caching/`, `event/`, `payment/`, `security/`) for external service interfaces.
- `application/vo/` — value objects (e.g., `identity` module).
- `application/validator/` — startup validation (e.g., `subscription` module).
- `infrastructure/constants/` — infrastructure-level constants (e.g., `wardrobe` module).

### Service-only modules

Some modules do not follow the full canonical layout because they serve a different purpose. For example, `notification` is a service-only module — it has no DB entities, no HTTP handlers, and uses `infrastructure/communication/` (SMTP sender) instead of `infrastructure/persistence/`. This is a valid design pattern for modules that only provide outbound integrations.

### Presentation provider variance

The `presentation/provider.go` file is optional. Only `identity` uses the layered provider style (separate `wire.NewSet(...)` per layer). All other modules wire handlers and workers directly from the module-level `provider.go` (flattened style).

### Workers are first-class presentation adapters

Background workers live in `presentation/worker/`, are resolved through DI, and registered in `bootstrap.AppWorkers`. They are started and stopped alongside the HTTP server. Workers are used for scheduled tasks (renewals, expiry, reconciliation) and async AI processing (batch upload, analyze, cleanup, recovery).

Current workers are discoverable in `bootstrap.AppWorkers` at `internal/bootstrap/app.go`.

---

## V. ROUTING & SURFACE AREAS

The Gin engine is assembled in `internal/api/routes/router.go`. Route groups are registered under `/api/v1/`:

```
/api/v1/health
/api/v1/auth/...
/api/v1/me/...
/api/v1/admin/...
```

Each business module **self-registers its own endpoint group** via `presentation/router/router.go` (an `Init(api *gin.RouterGroup)` method). `internal/api/routes/provider.go` (`AppRouter`) only composes the six module routers through Wire, and `internal/api/routes/router.go` (`NewEngine`) invokes `Init(api)` on each module router in order. To see the current module routers:

```bash
ls services/*/internal/modules/*/presentation/router/
```

Shared HTTP middleware (CORS, timeout, rate-limit, role authorize, swagger lock, global error handler) lives in `shared/presentation/middleware/`; module-owned middleware (e.g., identity `AuthMiddleware`) lives in `services/<service>/internal/modules/<name>/presentation/middleware/` and is consumed through `presentation/contract/` interfaces.

Swagger is served from `/swagger` and static docs assets from `/api-docs`. Per-module Swagger specs are generated under `api/swagger/modules/<name>/swagger.json` and selectable via the Swagger UI dropdown.

---

## VI. IMPLEMENTATION GUARDRAILS

- Keep handlers thin and delegate execution to use cases.
- Keep repositories inside module infrastructure or shared infrastructure packages.
- Prefer contracts for cross-module interaction instead of importing another module's internal use case implementation directly.
- **Cross-module data stitching** must use `sliceutils.IndexBy` / `sliceutils.GroupBy` for O(1) lookup instead of nested loops.
- A module's entity files belong in `services/<service>/internal/modules/<name>/domain/entities/`, not in shared.
- Reuse shared services for AI, media, messaging, caching, DB, and search instead of re-instantiating them inside modules.
- Use `shared_pres.Success()`, `shared_pres.Created()`, and `shared_pres.WrapHandler()` for consistent HTTP responses.
- Use `apperror` types for error propagation; let the global error middleware map them to HTTP responses.
- Handler signature: `func (h *Handler) Method(c *gin.Context) error`.
- A module's entity files belong in `services/<service>/internal/modules/<name>/domain/entities/`.
- A module that has been removed from the runtime (archived) should remain as a tree entry but be excluded from DI and route registration.

---

## VII. DATABASE SCHEMA MAPPING

The system uses a PostgreSQL relational database (with pgvector) as the Single Source of Truth, integrating with GORM for entity management.

> **Trạng thái hiện tại** (Sprint 2, migration `20260801000000_separate_module_schemas`): 45 bảng + 1 sequence đã tách sang schema theo module (`identity`, `wardrobe`, `fashion`, `brand`, `subscription`, `aiexecutor`, `dashboard`, `shared`). DSN dùng `search_path = public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared` (`public` đứng đầu để `goose_db_version` luôn resolve). FK trỏ tới `users` được GIỮ xuyên schema; 9 FK xuyên module phi-users được CẮT và enforce qua contract ở app layer.

### Tables by Module (45 bảng + 1 sequence)

**identity** (3): `users`, `refresh_tokens`, `user_style_profiles`

**wardrobe** (3): `wardrobe_items`, `outfits`, `outfit_items`

**fashion** (4): `fashion_items`, `categories`, `conversational_contexts`, `messages`

**brand** (15): `brands`, `brand_members`, `brand_customers`, `brand_customer_claims`, `loyalty_programs`, `loyalty_tiers`, `loyalty_accounts`, `loyalty_point_transactions`, `loyalty_point_lots`, `brand_benefits`, `benefit_redemptions`, `brand_conversations`, `brand_conversation_messages`, `brand_items`, `digital_sample_responses`

**subscription** (10 + 1 sequence): `subscription_plans`, `user_subscriptions`, `user_daily_quotas`, `user_wallets`, `deposit_transactions`, `wallet_statements`, `provider_payment_events`, `provider_webhook_inbox`, `user_subscription_events`, `subscription_renewal_attempts` (+ sequence `deposit_transactions_order_code_seq`, OWNED BY `deposit_transactions.order_code`)

**aiexecutor** (6): `ai_cost_policies`, `ai_cost_policy_operations`, `ai_policy_grants`, `ai_usage_period_ledgers`, `ai_usage_events`, `ai_model_configs`

**dashboard** (3): `dashboard_admin_daily_snapshots`, `dashboard_brand_daily_snapshots`, `dashboard_user_wardrobe_summaries`

**shared** (1): `outbox_records`

### Access Matrix — module nào đọc/ghi schema của module nào (qua contract)

> 1 connection chung (Q3): mọi module **có thể** query mọi schema qua `search_path` — không có phân quyền DB-level. Boundary được enforce ở **app layer** (Hiến pháp II): chỉ truy cập dữ liệu module khác qua `presentation/contract/`, cấm import entity/repo cross-module (đã verify SC-004: 0 raw cross-module DB access). Bảng này liệt kê các **kênh contract hợp lệ** đang dùng.

| Module (caller) | Sở hữu schema | Truy cập schema của module khác | Kênh contract |
|---|---|---|---|
| **wardrobe** | `wardrobe` | `fashion` (đọc category/fashion item; write reassign category) | `IFashionContract` |
| **fashion** | `fashion` | `wardrobe` (đọc item, update status, delete) | `IWardrobeContract` |
|  |  | `subscription` (đọc quota/overview) | `IUserSubscriptionContract` |
|  |  | `brand` (đọc brand item hợp lệ cho styling) | `IBrandContract` |
|  |  | `identity` (đọc user) | `IUserContract` |
| **brand** | `brand` | `identity` (đọc user theo id/username/email) | `IUserContract` |
| **subscription** | `subscription` | `identity` (auth middleware — router, không truy cập dữ liệu) | `IAuthMiddleware` |
| **aiexecutor** | `aiexecutor` | `subscription` (đọc overview/quotas cho reservation policy) | `IUserSubscriptionContract` |
| **identity** | `identity` | `notification` (gửi email/SMS) | `INotificationContract` |
|  |  | `subscription` (đọc subscription overview) | `IUserSubscriptionContract` |
| **dashboard** | `dashboard` | `aiexecutor` (AI usage/cost stats) | `IAICostDashboardReader` |
|  |  | `brand` (customer/loyalty/catalog analytics) | `IBrandDashboardReader` |
|  |  | `fashion` (category/fashion item mapping) | `IFashionDashboardReader` |
|  |  | `identity` (user stats) | `IUserDashboardReader` |
|  |  | `subscription` (distribution/revenue/renewal) | `ISubscriptionDashboardReader` |
|  |  | `wardrobe` (summary/underutilized/fashion ids) | `IWardrobeDashboardReader` |

**Ghi chú access theo chiều dữ liệu**: dashboard chỉ **đọc** qua DashboardReader; fashion/wardrobe/brand có cả đọc + ghi hạn chế qua contract (write chỉ qua method contract được expose, ví dụ `IFashionContract.ReassignCategoryForCatalogItems`, `IWardrobeContract.UpdateItemStatus`); FK xuyên module đã cắt → mọi ghi chéo đều qua contract (không có raw UPDATE).

### Foreign Keys xuyên module

- **GIỮ (xuyên schema tới `users`, 30 FK)**: mọi module giữ FK trỏ `identity.users` (cascade/restrict/set-null giữ nguyên — verify sau migration: chèn `user_id` không tồn tại bị chặn, xoá `users` cascade đúng).
- **CẮT (phi-users, enforce qua contract, 9 FK)**: `ai_policy_grants.plan_id/source_event_id/source_deposit_transaction_id` (aiexecutor→subscription), `subscription_plans.ai_cost_policy_id` (subscription→aiexecutor), `wardrobe_items.fashion_item_id` (wardrobe→fashion), `outfit_items.fashion_item_id` (wardrobe→fashion), `brand_items.fashion_item_id` (brand→fashion), `digital_sample_responses.outfit_id` (brand→wardrobe), `dashboard_brand_daily_snapshots.brand_id` (dashboard→brand). Integrity được đảm bảo qua contract reader/writer ở app layer (không có luồng hard-delete `fashion_items`/`brands` cần cascade bị mất).

### Chỉ mục vector (pgvector HNSW) — giữ nguyên sau tách schema

- `fitems_embedding_cosine_idx` on `fashion.fashion_items.embedding` (HNSW `vector_cosine_ops`)
- `ustyles_embedding_cosine_idx` on `identity.user_style_profiles.taste_embedding` (HNSW `vector_cosine_ops`)

> Verified (Sprint 2): `EXPLAIN` vector search dùng `Index Scan using fitems_embedding_cosine_idx`; index def schema-qualified.

---

## VIII. AUTHENTICATION & ROLE AUTHORIZATION

The system delegates authorization according to the RBAC (Role-Based Access Control) model for system users, subscription tiers, and brand memberships.

### 1. System Roles (`roleslug`)

- **User (`roleslug.User = "user"`)**: Standard authenticated consumer account.
- **Admin (`roleslug.Admin = "admin"`)**: System administrator with full authority across administration endpoints (`/admin/*`).

### 2. Subscription Tiers (Individual Consumer Quotas - `PlanKind`)

Consumer capabilities and limits are driven by active subscriptions (`PlanKind`):
- **Free Plan (`plankind.Free = "free"`)**: Standard digital wardrobe item limits and standard daily AI quotas.
- **Premium Plan (`plankind.Paid = "premium"`)**: Higher digital wardrobe limits and expanded daily AI outfit/chat styling quotas.

### 3. Business / Brand Roles (`brandmemberrole`)

Managed through the brand membership system (`brand_members` table):
- **Brand Owner (`brandmemberrole.Owner = "owner"`)**: Full access to configure Brand Portal, manage staff members, and brand settings.
- **Brand Staff (`brandmemberrole.Staff = "staff"`)**: Employee of the brand with delegated access to manage items, promotions, and customer chats.

---

## IX. SECURITY & DATA PRIVACY POLICY

Policy for protecting B2C user information when fashion brands extract data from Closy.

### 1. Data Anonymization

- All reports sent to the Brand Portal on the Insight Dashboard are aggregated data.
- Do not share personally identifiable information (Full name, Email, Phone number, Detailed address) of users with third parties or brands.
- Brands can only see overall statistics: e.g., "45% of female customers aged 18-24 love design dress A".

### 2. Opt-In / Opt-Out

- Users have the right to choose not to share their virtual digital wardrobe or anonymized survey information at any time through privacy settings on the app.

---

## X. EVENT-DRIVEN ARCHITECTURE

### Outbox pattern

All cross-module event publishing MUST use the **transactional outbox pattern**. Events are written to the `outbox_records` table within the same database transaction as the business operation. A dedicated `OutboxWorker` asynchronously publishes events to RabbitMQ and marks them as published. This guarantees at-least-once delivery and prevents event loss during broker outages.

- **Table:** `shared.outbox_records` (headers: `correlationId/eventVersion/publishedAt`)
- **Worker:** `shared/presentation/worker/outbox_worker.go` (poll `2s`, batch `200` parallel 10, `FOR UPDATE SKIP LOCKED` bọc transaction)
- **Routing:** The `topic` field in the outbox record determines the RabbitMQ routing key, enabling a single outbox table and worker to serve all modules.

### Event publishing preference

Cross-module write side effects **must prefer event publishing** over direct contract calls. Direct calls are acceptable only when the caller and callee share the same database transaction (Unit of Work) and immediate consistency is required.

### Cơ chế RabbitMQ hiện tại (016)

- **Exchange**: `smart_wardrobe_exchange` (topic) + DLX `smart_wardrobe_exchange.dlx`; wire qua `shared/infrastructure/messaging/topology.go` (registry).
- **Routing**: `module.event.action` với hậu tố `.wardrobe/.brand` cho fashion, wildcard `.#` thống nhất; point-to-point (1 key→1 queue) vs pub/sub (1 key→2 queue) ở mức cơ chế.
- **Queue**: quorum-ready single-node (1 node prod, `x-queue-type=quorum`, `x-delivery-limit=5`, `x-dead-letter-exchange`); downtime drain chấp nhận do demo chưa release.
- **Channel**: `publisherChannel` (Confirm) + consumer channel riêng `Qos(prefetch=20, global=false)`, `prefetch/concurrency` per-queue đọc từ registry `topology.go`; consumer đăng ký qua generic helper `event.RegisterTypedConsumer[T]` (application layer; unmarshal + `correlationId` ctx do helper xử lý, impl `RegisterTypedHandler` ở messaging).
- **Luồng**: `outbox → exchange → queue → consumer → Ack/Nack (Permanent→DLQ, Retryable→requeue)` với `correlationId` trace.

Chi tiết queue không liệt kê exhaustive; xem `topology.go` và `docs/system-design/event-driven-design.md` (đúng 6 cơ chế: exchange+DLS, routing, outbox, DLQ/retry, channel/prefetch, flow).