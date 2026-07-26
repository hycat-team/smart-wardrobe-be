<!--
Sync Impact Report:
- Version change: 1.3.0 -> 1.4.0
- Modified principles:
  - III. Asynchronous Event-Driven Processing (expanded with outbox pattern requirement)
- Added sections: None
- Removed sections: None
- Removed sections: None
- Templates requiring updates:
  - .specify/templates/plan-template.md (✅ verified/no updates needed)
  - .specify/templates/spec-template.md (✅ verified/no updates needed)
  - .specify/templates/tasks-template.md (✅ verified/no updates needed)
  - .specify/templates/constitution-template.md (✅ verified/no updates needed)
- Follow-up TODOs: None
-->

# SmartWardrobe Backend Constitution

## Core Principles

### I. Pragmatic Modular Monolith
All business modules run within a single deployable Gin-based HTTP service. Shared infrastructure
(Postgres, Redis, RabbitMQ, Cloudinary, external AI providers, Elasticsearch) must be initialized
once and reused across modules via dependency injection (DI) through Wire. Each module owns its
persistent models under `internal/modules/<name>/domain/entities/`. Direct struct pointers or
cross-module database joins/preloads on entities of other modules are strictly prohibited.

### II. Strict Module Boundaries & DTO Contracts
Cross-module communication MUST go through the public service interfaces defined in
`presentation/contract/` using DTOs, never domain entities. A module must never import another
module's internal use case implementations, entities, repositories, or persistence structures.
Cross-module data stitching must use `sliceutils.IndexBy` or `sliceutils.GroupBy` for O(1) lookups
instead of nested loops.

**Exception — Domain constants**: A module MAY import another module's `domain/constants/`
directly (e.g., `fashion` module importing `wardrobe/domain/constants/wardrobestatus`).
These are pure enum/const values without behavior, and duplicating them across modules
would cause maintenance overhead. However, the importing module MUST NOT **hardcode**
the constant's value (e.g., writing `"active"` instead of `wardrobestatus.Active`) —
always reference the constant symbol to avoid typo drift.

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
  Orchestrates business logic. Never depends on infrastructure.
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

## Technology Stack & Database Constraints
- **Language & Runtime**: Golang compiler (v1.25.+) with Google Wire for dependency injection.
- **Web & API Framework**: Gin web framework with automated Swagger generation via Swag.
- **Relational & Cache Storage**: PostgreSQL (with pgvector wrapper) for main datastore, Redis for caching.
- **Search & Messaging**: Elasticsearch client for search indexing; RabbitMQ for event queuing.
- **Goose Database Migrations**: All database schema changes MUST be implemented via Goose SQL
  migration files (`make migration-create` and `make migration-up`). Directly editing initial
  database scripts in `/init-db` is strictly prohibited.
- **Core Tables**: `users`, `wardrobe_items`, `outfits`, `brands`, `brand_members`, `loyalty_points`,
  `campaigns`, `digital_samples`, and `sample_votes`.

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
  - **B2C Roles**: *Guest* (can only view subscription plans or public pages), *Standard User*
    (limited wardrobe up to 50 items and daily AI quota), *Premium User* (unlimited wardrobe and
    expanded AI styling quota).
  - **B2B Roles**: *Brand Owner* (full access to configure Brand Portal and service billing),
    *Brand Staff* (employee of the brand).
  - **System Admin**: Full authority to review brand status, approve registrations, and troubleshoot.
- **Data Privacy & Anonymization**:
  - All brand portal reports must use aggregated and anonymized data. Sharing personally identifiable
    information (PII) is strictly prohibited.
  - B2C users retain the right to opt-in or opt-out of virtual digital wardrobe and survey data
    sharing at any time through application settings.

## Governance
- The Project Constitution is the authoritative document superseding other developer practices.
- Any architectural changes, new principles, or redefinitions of existing boundaries require
  major/minor/patch version updates.
- Validation checks: Every pull request must verify compliance with layer boundaries (no cross-layer
  skipping), standard error handling, and JSON naming conventions for frontend-facing DTOs.
- Archived modules must retain their folder tree structure but be completely removed from Wire DI
  configuration and route registration.
- Refer to `docs/development/local-setup.md` for running and executing development code using
  Makefile commands (e.g., `make dev`).

**Version**: 1.4.0 | **Ratified**: 2026-07-13 | **Last Amended**: 2026-07-26
