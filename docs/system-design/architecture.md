# ARCHITECTURE & TECHNICAL DEVELOPMENT RULES

## I. SYSTEM ARCHITECTURE PARADIGM

The backend is implemented as a **pragmatic modular monolith** in Go with **clean layer boundaries where practical**, but not every module is shaped identically.

- **Single deployable application:** all modules run inside one Gin-based HTTP service started from `cmd/server/main.go`.
- **Shared infrastructure:** Postgres, Redis, RabbitMQ, Cloudinary, external AI providers, and Elasticsearch are initialized once and reused across modules through DI.
- **Database-first relational model:** shared GORM entities live under `internal/shared/domain/entities` and represent the canonical relational model used across the system.
- **No DDD aggregate/event model:** the codebase does not model aggregate roots, value objects, or domain events in a DDD sense. Cross-module integration is handled through contracts plus shared infrastructure.
- **Module boundaries are real, but uneven:** `identity` still follows the most explicit multi-layer provider structure; `subscription`, `wardrobe`, and `community` are still modularized, but some provider wiring is flattened at module root.

---

## II. CURRENT PROJECT DIRECTORY TREE

The following tree reflects the current repository shape and should be treated as the source of truth for architecture discussions:

```text
root/smart-wardrobe-be
|
|-- cmd
|   `-- server
|       `-- main.go
|
|-- config
|   |-- config.go
|   |-- config.handler.go
|   `-- config.helper.go
|
|-- docs
|   |-- business
|   |-- technical
|   |-- temp
|   |-- docs.go
|   |-- index.html
|   |-- swagger.json
|   `-- swagger.yaml
|
|-- internal
|   |-- api
|   |   |-- middleware
|   |   `-- routes
|   |-- bootstrap
|   |   `-- app.go
|   |-- di
|   |   |-- wire.go
|   |   `-- wire_gen.go
|   |-- modules
|   |   |-- identity
|   |   |   |-- application
|   |   |   |-- contract
|   |   |   |-- domain
|   |   |   |-- infrastructure
|   |   |   |-- presentation
|   |   |   `-- provider.go
|   |   |-- subscription
|   |   |   |-- application
|   |   |   |-- contract
|   |   |   |-- domain
|   |   |   |-- infrastructure
|   |   |   |-- presentation
|   |   |   `-- provider.go
|   |   |-- wardrobe
|   |   |   |-- application
|   |   |   |-- contract
|   |   |   |-- domain
|   |   |   |-- infrastructure
|   |   |   |-- presentation
|   |   |   `-- provider.go
|   |   `-- community
|   |       |-- application
|   |       |-- domain
|   |       |-- infrastructure
|   |       |-- presentation
|   |       `-- provider.go
|   `-- shared
|       |-- application
|       |   |-- ai
|       |   |-- constants
|       |   |-- dto
|       |   |-- event
|       |   `-- media
|       |-- domain
|       |   |-- constants
|       |   |-- entities
|       |   |-- money
|       |   `-- repositories
|       |-- infrastructure
|       |   |-- ai
|       |   |-- caching
|       |   |-- db
|       |   |-- media
|       |   |-- messaging
|       |   |-- repositories
|       |   `-- search
|       |-- presentation
|       `-- provider.go
|
`-- pkg
    |-- logger
    `-- utils
```

---

## III. SHARED DATA MODEL & INFRASTRUCTURE

### Shared entities

All persistent models are centralized under `internal/shared/domain/entities`. The current entity set spans identity, subscription, wardrobe, outfit, chat, and community features, including:

- Identity: `User`, `UserStyleProfile`, `RefreshToken`
- Subscription/Billing: `SubscriptionPlan`, `UserSubscription`, `UserDailyQuota`, `UserWallet`, `DepositTransaction`, `WalletStatement`
- Wardrobe/AI: `Category`, `WardrobeItem`, `ConversationalContext`, `Message`, `Outfit`, `OutfitItem`
- Community: `Post`, `PostScoreSnapshot`, `PostItem`, `PostMedia`, `Comment`, `Like`, `TransferRequest`

This is more than the older 16-table assumption and must be documented as a shared model pool rather than a small fixed schema.

### Shared infrastructure

The `internal/shared` package is not limited to DB primitives. It currently owns reusable integrations for:

- `infrastructure/db`: Postgres connection and unit of work
- `infrastructure/caching`: Redis connection
- `infrastructure/media`: Cloudinary media service
- `infrastructure/ai`: provider-agnostic AI service with primary/fallback config
- `infrastructure/messaging`: RabbitMQ client and event publishing
- `infrastructure/search`: Elasticsearch client
- `presentation`: shared HTTP response and SSE helpers

---

## IV. MODULE ORGANIZATION RULES

### Current layering reality

Modules are expected to preserve application boundaries, but the exact provider layout differs by module:

- `identity` is the closest reference module for explicit sub-layer provider sets.
- `subscription` keeps layered folders, but presentation providers are wired directly from handlers/workers at module root instead of a dedicated `presentation/provider.go`.
- `wardrobe` and `community` are fully modularized by folder, but their `provider.go` files aggregate repositories, use cases, handlers, and workers directly.

Because of this, architecture docs must describe **layer intent plus current wiring shape**, not an idealized tree that only partially exists.

### Cross-module contracts

Cross-module communication currently uses module-level `contract` packages such as:

- `internal/modules/identity/contract`
- `internal/modules/subscription/contract`
- `internal/modules/wardrobe/contract`

These contracts do not currently live under `application/contract/`. Any new documentation must match that repo convention unless the codebase is explicitly refactored.

### Workers are first-class presentation adapters

Background workers are part of the real runtime architecture and should be documented alongside HTTP handlers:

- `subscription`: scheduled renewal worker
- `community`: post hotness worker
- `wardrobe`: batch upload worker, search sync worker, failed-items cleanup worker

Workers are resolved through DI and attached to `bootstrap.AppWorkers`.

---

## V. ROUTING & SURFACE AREAS

The API surface is broader than the older architecture note implied. Current route groups include:

- `auth`
- `me`
- `subscriptions`
- `wardrobe-items`
- `ai`
- `outfits`
- `categories`
- `posts`
- `transfers`
- `admin`

Swagger is served from `/swagger` and static docs assets are served from `/api-docs`.

---

## VI. IMPLEMENTATION GUARDRAILS

- Keep handlers thin and delegate execution to use cases.
- Keep repositories inside module infrastructure or shared infrastructure packages.
- Prefer contracts for cross-module interaction instead of importing another module's internal use case implementation directly.
- Reuse shared services for AI, media, messaging, caching, DB, and search instead of re-instantiating them inside modules.
- Update architecture docs whenever a module, integration, or runtime worker is added. Do not leave the repo tree frozen to an older snapshot.
