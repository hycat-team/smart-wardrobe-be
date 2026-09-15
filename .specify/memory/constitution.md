<!--
Sync Impact Report:
- Version change: 2.1.0 → 3.0.0 (MAJOR — backwards-incompatible governance change)
  Reasoning: This amendment REMOVES previously valid configurations. The monolith topology
  (`cmd/server`) is dropped from `main`: it no longer builds in CI, is no longer a supported
  runtime, and survives only as the frozen tag `monolith-legacy` for emergency reference. The
  codebase moves from a single Go module to a multi-module Go workspace (`go.work`), which
  changes the module/import governance in §I/§II. The supported VPS runtime becomes the
  3-service Compose topology. Hence MAJOR.
- Modified principles:
  - §I Independently Deployable Services (multi-module workspace): single topology
    (Account, Commerce Core, AI Engine); `cmd/server` removed; service manifests replaced
    by compiler-enforced boundaries via `go.work`.
  - §II Cross-Service Communication: dual adapter kept (in-process within a service, gRPC
    across services); all monolith references removed; contract code under `contracts/`.
  - §VIII Infrastructure Portability: funding schedule / repatriation timeline kept in
    `docs/project/funding-and-platform-lifecycle.md` (project lifecycle), NOT in this
    constitution; only code-level portability rules remain here.
  - Technology Stack: Go workspace (`go.work`) multi-module; per-service Swagger with
    union-section preservation.
  - Governance: production topology = services on every environment; monolith is not a
    runtime; dual-topology permanence removed.
- Added sections: None.
- Removed sections: Dual-topology allowance, monolith build gate, strangler coexistence.
- Templates requiring updates:
  - .specify/templates/plan-template.md (✅ verified/no updates needed)
  - .specify/templates/spec-template.md (✅ verified/no updates needed)
  - .specify/templates/tasks-template.md (✅ verified/no updates needed)
- Docs requiring manual follow-up (outside this skill's scope):
  - docs/system-design/architecture.md, README.md, docs/development/local-setup.md
  - docs/operations/* (cutover + repatriation runbooks: VPS runs the 3-service topology)
  - deployments/docker-compose.prod.yml (3 service containers + infra)
  - docs/project/funding-and-platform-lifecycle.md (funding clock + repatriation schedule)
- Follow-up TODOs: tag `monolith-legacy` at the last monolith-green commit before the split.
-->

# SmartWardrobe Backend Constitution

## Core Principles

### I. Independently Deployable Services (Multi-Module Workspace)
The backend MUST be structured as independently deployable services built from a
multi-module Go workspace (`go.work`):

- **Service topology — the ONLY supported topology**: three services — Account
  (`identity` + `subscription` + `admin` + `notification`), Commerce Core
  (`wardrobe` + `fashion` + `brand` + `dashboard`), and AI Engine (`aiexecutor`).
  Each service MUST be independently deployable: its own Go module, container,
  health check, configuration, and scaling; it MUST NOT require redeploying
  another service to ship. This topology runs on every environment — managed
  platform (Cloud Run) and self-hosted VPS (Docker Compose).
- **Monolith retired**: `cmd/server` is REMOVED from `main`. The last monolith-green
  commit MUST be preserved as the frozen tag `monolith-legacy` for emergency
  reference only. It is NOT built in CI, NOT a runtime, and MUST NOT receive new
  business logic. The VPS runtime is the 3-service Compose topology, not the monolith.
- **Module boundaries enforced by the compiler**: a service module (`services/<service>`) MUST NOT
  import implementation packages (`domain/entities`, `domain/repositories`,
  `infrastructure/**`, `application/usecase/**`) of modules owned by another
  service. The `go.work` + per-module `go.mod` layout makes cross-service
  implementation imports a build failure.

Shared infrastructure (Postgres, Redis, RabbitMQ, Cloudinary, external AI providers, and the
optional Elasticsearch) MUST be initialized once per binary/service and reused across modules via
dependency injection (DI) through Wire. Each module owns its persistent models under
`internal/modules/<name>/domain/entities/`. Direct struct pointers or cross-module database
joins/preloads on entities of other modules are strictly prohibited.

**Environment switching**: switching environments (managed ⇄ self-hosted) MUST require
only configuration and DNS changes — never code changes (see §VIII).

### II. Strict Module Boundaries & DTO Contracts
Cross-module communication MUST go through the public service interfaces defined in
`presentation/contract/` using DTOs, never domain entities. Cross-module data stitching
must use `sliceutils.IndexBy` or `sliceutils.GroupBy` for O(1) lookups instead of
nested loops.

**Allowed cross-module imports:**

| Import Target | Allowed | Rationale |
|---------------|---------|-----------|
| `presentation/contract/` | ✅ | Application layer needs to call cross-module services. Contracts are the official interface boundary. |
| `application/dto` | ✅ | DTOs are shared data shapes. Direct import avoids duplication and keeps type safety across module boundaries. |
| `domain/constants/` | ✅ | Pure enum/const values without behavior. Duplicating them would cause maintenance overhead. Always reference the constant symbol — never hardcode the value. |

**Prohibited cross-module imports (strictly enforced):**

| Import Target | Prohibited | Rationale |
|---------------|------------|-----------|
| `domain/entities/` | ❌ | Preserves data ownership. Prevents tight coupling at the data layer and cross-module DB joins. |
| `domain/repositories/` | ❌ | Repository interfaces belong to the owning module only. |
| `infrastructure/persistence/` | ❌ | Implementation details must not leak across module boundaries. |

**Rationale:** The services share the same module boundaries and business domain, so strict
isolation via contracts-only still creates excessive ceremony without practical benefit for
in-domain communication. The prohibition on importing entities, repositories, and persistence
remains absolute to preserve data ownership and prevent tight coupling at the data layer.

**Pure Contract Usecase Principle (added in 1.8.0):**
Contract usecases (those serving `presentation/contract/`) MUST be pure — they may only access repositories and mappers of their own module (e.g., `wardrobeRepo`, `brandRepo`/`brandItemRepo`/`customerRepo` + `benefitUC` intra-module). They MUST NOT call `presentation/contract/` of other modules to enrich data (no `fashionContract`/`userSubContract`/`userBriefContract` inside contract usecases). If the contract usecase is shared with handlers (e.g., `BrandItemUseCase` used by both `BrandContract` and `BrandPortalHandler`), a separate contract usecase MUST be extracted (e.g., `BrandItemContractUseCase` at `brand/application/usecase/item/contractuc/`). Callers requiring enriched data MUST self-enrich after receiving the pure result: collect `FashionItemID` (exposed as `fashionItemId` camelCase in DTOs) and call the appropriate `fashionItemRepo` (if caller is inside the `fashion` module, per §IV) or `fashionContract` (if caller is outside) followed by `sliceutils.IndexBy`/`GroupBy` stitching. Wardrobe pure results MUST expose `FashionItemID` even when `FashionItem == nil`, with nil-guard for cases where `FashionItemID == nil` (e.g., failed upload before fashion record creation). See `docs/system-design/contract-purity-pattern.md` for before/after and stitching guide.

**Cross-Service Communication & Dual Adapter:**
- Within a single service, cross-module communication MUST use
  `presentation/contract/` interfaces in-process.
- Across service boundaries, synchronous queries MUST use gRPC (protobuf), and asynchronous
  side effects MUST use RabbitMQ with the transactional outbox pattern (see §III).
- Every cross-module contract interface that spans a service boundary MUST provide two
  adapters: an in-process implementation (service-local) and a gRPC client
  implementation (split). The composition root (Wire) selects the adapter; business
  logic MUST NOT change with the environment.
- CI MUST build every service binary (`services/<service>/cmd/<service>`). Cross-service contract
  code lives under `contracts/` (protobuf sources, generated stubs, and per-owner clients).

### III. Asynchronous Event-Driven Processing
RabbitMQ must be used to process resource-intensive or slow tasks (e.g., AI analysis, background
syncing) asynchronously to prevent HTTP Gateway timeouts. Background workers (scheduled or event
consumers) are first-class presentation adapters located under `presentation/worker/` and must
coordinate gracefully with the app lifecycle. All cross-module write side effects must prefer
event publishing over direct function calls.

**Outbox pattern**: All event publishing MUST use the transactional outbox pattern — the event
record is written to the `outbox_records` table within the same database transaction as the
business operation. A dedicated outbox worker asynchronously publishes events to RabbitMQ and
marks them as published. This guarantees at-least-once delivery and prevents event loss during
broker outages. The `topic` field in the outbox record determines the RMQ routing key, enabling
a single outbox table and worker to serve all modules.

### IV. Clean Architecture & Strict Layer Dependency
Within each module, dependencies must follow a strict inward-directed layering:

```
presentation → application → domain ← infrastructure
```

- **presentation** (handlers, workers, contracts): depends on **application** (via use case
  interfaces). Never calls infrastructure or domain directly.
- **application** (use cases, mappers, DTOs): depends on **domain** (via repository interfaces).
  Orchestrates business logic. For cross-module communication, application MAY import
  `presentation/contract/` of other modules. Never depends on infrastructure.
- **domain** (entities, repository interfaces, constants): **innermost layer with ZERO
  dependencies** on any other layer. No imports to application, infrastructure, or presentation.
- **infrastructure** (GORM repositories, caching, messaging, external API clients): depends on
  **domain** (implements repository interfaces). Never called directly by presentation.

**Strict rules**:
- NO layer may skip or bypass another layer (e.g., presentation MUST NOT call
  infrastructure/repositories directly).
- NO layer may depend on a layer at the same level or outward.
- Handlers must remain thin: delegate business logic to use cases, return errors via
  `func (h *Handler) Method(c *gin.Context) error` wrapped by `shared_pres.WrapHandler`.

### V. Standardized Transaction & Error Handling
Business and system errors must be centralized using `apperror` constructor factories mapping
HTTP status codes to Vietnamese titles/messages. Global error middleware maps returned errors to
structured JSON, capturing stack traces only in non-production. Transactional operations across
multiple repository calls must use the Unit of Work (`IUnitOfWork.Execute`) pattern, propagating
transactions implicitly via `context.Context` rather than manual transaction tracking.

### VI. AI Pipeline, Cost Control & Local Fallbacks
AI processing must implement a reservation-first policy engine using row-level database locks (`FOR UPDATE`)
on usage ledgers. Paid LLM calls must be protected against currency leakages, automatically falling
back to local processing (e.g., local HSL matching or Gemma models) if quotas are exceeded or network
failures occur. AI responses must enforce schema constraints (Gemini Response Schema) and undergo strict
balanced bracket scanning, placeholder checking, and anti-SQL/tsquery injection validation.

### VII. Design Patterns & Conventions Compliance
Before writing or modifying any code in this project, agents MUST read and comply with
`docs/system-design/design-pattern.md`. This document is the authoritative source for
project-specific design patterns, utility package usage, repository conventions, error
handling recipes, pagination patterns, handler structure, and all other coding conventions
not covered by this constitution. It may evolve independently — always read the latest
version. In case of conflict between this constitution and `design-pattern.md`, this
constitution takes precedence for architectural boundaries and principles; `design-pattern.md`
governs implementation-level conventions.

### VIII. Infrastructure Portability & No Vendor Lock-in
The system MUST run on both self-hosted infrastructure (Docker Compose / VPS) and managed
infrastructure (Supabase, CloudAMQP, Redis Cloud, Cloud Run, Cloudflare).

- All infrastructure connection parameters (host, port, `sslmode`, TLS, migrate-on-start,
  outbox listener endpoint, and similar) MUST be configurable via environment variables.
  Config files MUST NOT hardcode environment-specific hosts or endpoints.
- The codebase MUST NOT depend on vendor-exclusive features (e.g., Supabase Auth/RLS/Storage,
  Railway volumes). Only plain PostgreSQL, Redis, and RabbitMQ capabilities may be used, so that
  a reverse cutover to the VPS remains possible.
- Switching environments MUST require only configuration and DNS changes — never code changes.

**Rationale**: The same service images MUST run identically on self-hosted and managed
infrastructure; any environment difference MUST be expressed as configuration, never as code
forks. (Funding schedules, expiry dates, and repatriation timelines are project-lifecycle
concerns and live in `docs/project/funding-and-platform-lifecycle.md`, not here.)

## Technology Stack & Database Constraints
- **Language & Runtime**: Golang compiler (v1.25.+) with Google Wire for dependency injection.
  The repository is a **multi-module Go workspace** (`go.work`): one module per service plus
  shared modules. Cross-service implementation imports MUST fail at compile time.
- **Web & API Framework**: Gin web framework with automated Swagger generation via Swag
  (per-service generation; the union of services MUST preserve all documented sections).
- **Relational & Cache Storage**: PostgreSQL (with pgvector wrapper) for main datastore, Redis for caching.
- **Search & Messaging**: Elasticsearch client for optional search indexing; RabbitMQ for event queuing.
- **Search Optionality**: Elasticsearch is NOT a mandatory infrastructure dependency. It MUST be
  disableable by default (`ELASTICSEARCH_ENABLED=false`) and MUST automatically fall back to
  database-backed search when disabled or unavailable.
- **Goose Database Migrations**: All database schema changes MUST be implemented via Goose SQL
  migration files (`make migration-create` and `make migration-up`). Directly editing initial
  database scripts in `/init-db` is strictly prohibited.
- **Database Schema**: Core tables, schema-per-module separation, FK rules, and access matrix
  are documented in `docs/system-design/architecture.md §VII`. **Note (giai đoạn 020)**: khi
  `architecture.md` còn là baseline cũ, nguồn chuẩn cho schema/config là
  `specs/020-platform-migration/data-model.md`; `architecture.md` sẽ được cập nhật ở task T111.

## Code Style, Naming & Language Standards
- **Language Boundaries**: Swagger text, request validation errors, and all client-facing messages
  MUST be written in Vietnamese. Internal code comments, log messages, and database/code identifiers
  (variables, struct fields, functions, tables) MUST be in English.
- **Naming Conventions**: Repository queries and data lookup methods MUST prefer the `Get...` prefix
  (e.g., `GetByID`) over `Find...` when retrieving entities.
- **JSON Naming for API DTOs**: All DTO response fields serialized to JSON for frontend communication
  MUST use camelCase (e.g., `wardrobeItemId`, `subscriptionRevenueVnd`). Internal DTOs used
  exclusively for event messaging (RabbitMQ), search indexing (Elasticsearch), or AI provider
  response parsing MAY use snake_case for compatibility with their respective systems.
- **File Organization**: Mapper functions converting entities to DTOs must live inside the
  `application/mapper` folder (never in use case files). Single-use helpers must end with the
  `_helper.go` suffix.

## Access Control (RBAC) & Data Privacy Policies
- **Role-Based Access Control (RBAC)**:
  - **System RoleSlug (`users.role_slug`)**: Chỉ có 2 vai trò hệ thống — `admin` (quản trị viên hệ
    thống) và `user` (người dùng thông thường). Đây là vai trò hệ thống duy nhất; cấm thêm RoleSlug
    mới ngoài 2 giá trị này.
  - **Hạng theo gói subscription (Subscription tiers)**: *Guest*, *Standard User*, *Premium User*
    là HẠNG THEO GÓI CƯỚC (PlanKind: `free` / `premium`), KHÔNG phải vai trò hệ thống:
    - *Guest* (chưa đăng nhập): chỉ được xem gói cước hoặc trang public.
    - *Standard User* (gói `free`): tủ đồ giới hạn và quota AI hàng ngày.
    - *Premium User* (gói `premium`): tủ đồ không giới hạn và quota AI phối đồ mở rộng.
  - **B2B Roles** (membership trong brand — không phải RoleSlug): *Brand Owner* (full quyền cấu hình
    Brand Portal và service billing), *Brand Staff* (nhân viên vận hành của nhãn hàng).
  - **System Admin**: toàn quyền review trạng thái brand, duyệt đăng ký, và troubleshoot.
- **Data Privacy & Anonymization**:
  - All brand portal reports must use aggregated and anonymized data. Sharing personally identifiable
    information (PII) is strictly prohibited.
  - B2C users retain the right to opt-in or opt-out of virtual digital wardrobe and survey data
    sharing at any time through application settings.

## Governance
- The Project Constitution is the authoritative document superseding other developer practices.
  It contains **code, system, and architecture rules only**; project-lifecycle matters (funding,
  schedules, staffing) live in `docs/project/`.
- Any architectural changes, new principles, or redefinitions of existing boundaries require
  major/minor/patch version updates.
- **Production topology**: every environment (self-hosted VPS and managed platform)
  MUST run the service topology (Account, Commerce Core, AI Engine). There is no
  monolith runtime. The frozen `monolith-legacy` tag exists for emergency reference
  only and MUST NOT be extended.
- **Migration legitimacy**: the monolith → independently deployable services migration is a
  valid architectural evolution, provided each step is atomic and keeps the build green
  (`make wire && make lint && make test-race`).
- **Independent deployment check**: each service MUST be shippable without redeploying another
  service. Every production deploy path (VPS Compose, managed platform) MUST build and run the
  services independently.
- **Environment portability check**: every pull request that touches infrastructure code MUST
  verify that switching environments requires only configuration and DNS changes, not code changes.
- Validation checks: Every pull request must verify compliance with layer boundaries (no cross-layer
  skipping), standard error handling, and JSON naming conventions for frontend-facing DTOs.
- Archived modules must retain their folder tree structure but be completely removed from Wire DI
  configuration and route registration.
- Refer to `docs/development/local-setup.md` for running and executing development code using
  Makefile commands (e.g., `make dev`).

**Version**: 3.0.0 | **Ratified**: 2026-07-13 | **Last Amended**: 2026-09-15
