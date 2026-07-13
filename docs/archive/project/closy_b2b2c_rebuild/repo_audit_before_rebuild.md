# Repo Audit Before Rebuild

Phase: `phase_00_alignment`

## Current module structure

Runtime modules currently under `internal/modules`:

- `identity`
- `subscription`
- `wardrobe`
- `community`

Shared entities are centralized in `internal/shared/domain/entities`.
Shared infrastructure exists under `internal/shared/infrastructure` with DB, Redis caching, AI, messaging, media, and Elasticsearch integrations.

Phase mapping note:

- Keep the existing modular monolith structure.
- `brand` does not exist yet.
- `community` still exists and must be archived/removed through the planned phases, not by ad hoc deletion in Phase 00.

## Current migration tool

The project uses Goose SQL migrations.

- Migration directory: `migrations`
- Embedded migration entrypoint: `migrations/migrations.go`
- Runtime migration runner: `internal/shared/infrastructure/db/migration.go`
- Startup execution: `internal/bootstrap/app.go`
- Makefile targets: `make migration-status`, `make migration-up`, `make migration-down`, `make migration-create name=...`

Runtime migrations are embedded with:

```go
//go:embed *.sql
var EmbedFS embed.FS
```

`init-db` contains baseline SQL but must not be modified directly.

## Current users schema

Source of truth inspected:

- `init-db/01-schema.sql`
- `internal/shared/domain/entities/identity_entities.go`

Current `users` fields:

```text
id UUID PK default gen_random_uuid()
username VARCHAR(255) NOT NULL
email VARCHAR(255) UNIQUE NOT NULL
password_hash VARCHAR(255) NOT NULL
first_name VARCHAR(255) NULL
last_name VARCHAR(255) NULL
date_of_birth DATE NULL
address VARCHAR(255) NULL
gender INT NULL
role_slug VARCHAR(50) NOT NULL
body_profile JSONB NULL
status SMALLINT NOT NULL DEFAULT 0
is_deleted BOOLEAN NOT NULL DEFAULT FALSE
avatar_url VARCHAR(500) NULL
avatar_public_id VARCHAR(255) NULL
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Important findings:

- `email` is currently `NOT NULL`.
- `password_hash` is currently `NOT NULL`.
- There is no `phone_e164`.
- There is no `registration_source`.
- `status` is `SMALLINT`, mapped by `userstatus.UserStatus`.
- Existing register/login flow is email/username + password with email OTP confirmation.

## Current wardrobe_items schema

Source of truth inspected:

- `init-db/01-schema.sql`
- `init-db/02-indexes.sql`
- `internal/shared/domain/entities/wardrobe_entities.go`

Current `wardrobe_items` fields:

```text
id UUID PK default gen_random_uuid()
user_id UUID NOT NULL FK users(id)
category_id UUID NULL FK categories(id)
image_url VARCHAR(500) NOT NULL
image_public_id VARCHAR(255) NOT NULL
color VARCHAR(50) NULL
color_hex VARCHAR(7) NULL
color_hue DOUBLE PRECISION NULL
color_saturation DOUBLE PRECISION NULL
color_lightness DOUBLE PRECISION NULL
style VARCHAR(100) NULL
material VARCHAR(100) NULL
pattern VARCHAR(100) NULL
fit VARCHAR(50) NULL
seasonality VARCHAR(100) NULL
description TEXT NULL
price DECIMAL(12,2) NULL
status SMALLINT NOT NULL DEFAULT 0
item_type SMALLINT NOT NULL DEFAULT 0
embedding VECTOR(768) NULL
last_used_at TIMESTAMPTZ NULL
processing_retry_count INT NOT NULL DEFAULT 0
processing_version INT NOT NULL DEFAULT 0
processing_started_at TIMESTAMPTZ NULL
last_processing_attempt_at TIMESTAMPTZ NULL
processing_error_reason TEXT NULL
review_reason VARCHAR(100) NULL
is_deleted BOOLEAN NOT NULL DEFAULT FALSE
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Important findings:

- Metadata currently lives directly on `wardrobe_items`.
- `embedding` dimension is `VECTOR(768)`.
- HNSW vector index exists on `wardrobe_items.embedding`: `witems_embedding_cosine_idx`.
- Lexical GIN index exists on a `to_tsvector` expression over `wardrobe_items` metadata: `idx_wardrobe_items_lexical_search`.
- No `fashion_items` table exists yet.

## Current outfit_items schema

Source of truth inspected:

- `init-db/01-schema.sql`
- `internal/shared/domain/entities/wardrobe_entities.go`

Current `outfit_items` fields:

```text
outfit_id UUID NOT NULL FK outfits(id)
item_id UUID NOT NULL FK wardrobe_items(id)
position_x DOUBLE PRECISION NOT NULL DEFAULT 0.0
position_y DOUBLE PRECISION NOT NULL DEFAULT 0.0
scale DOUBLE PRECISION NOT NULL DEFAULT 1.0
layer_order SMALLINT NOT NULL DEFAULT 1
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
PRIMARY KEY (outfit_id, item_id)
```

Important findings:

- Primary key is composite: `(outfit_id, item_id)`.
- `item_id` currently points to `wardrobe_items(id)`.
- There is no `fashion_item_id`.
- There is no `item_context`.

## Current auth and OTP flow summary

Source files inspected:

- `internal/api/routes/auth/router.go`
- `internal/modules/identity/application/dto/auth_req.go`
- `internal/modules/identity/application/usecase/register_uc.go`
- `internal/modules/identity/application/usecase/session_uc.go`
- `internal/modules/identity/application/usecase/password_recovery_uc.go`
- `internal/modules/identity/infrastructure/caching/redis_otp_service.go`
- `internal/shared/application/constants/otpconstants/otpconstants.go`

Current auth endpoints:

```text
POST /api/v1/auth/register
POST /api/v1/auth/register/confirm-otp
POST /api/v1/auth/register/resend-otp
POST /api/v1/auth/login
POST /api/v1/auth/refresh-token
POST /api/v1/auth/forgot-password
POST /api/v1/auth/forgot-password/confirm-otp
POST /api/v1/auth/forgot-password/resend-otp
POST /api/v1/auth/reset-password
POST /api/v1/auth/logout
```

Current behavior:

- Register requires `username`, `email`, `password`, `confirmPassword`, `firstName`, and `dateOfBirth`.
- Register creates a temporary user payload in Redis, sends OTP by email, and only creates `users` row after OTP confirmation.
- Login uses `loginName` plus `password`; `loginName` is username or email.
- Forgot password uses email OTP.
- Confirm forgot-password OTP returns a reset token.
- Reset password validates reset token and updates `password_hash`.

Current Redis OTP storage:

- Implementation: `internal/modules/identity/infrastructure/caching/redis_otp_service.go`
- Key builder: `internal/shared/application/constants/otpconstants/otpconstants.go`
- Key format: `otp:{purpose}:{keyType}:{email}`
- Stored keys per OTP: value, attempts, cooldown, data.

Phase impact:

- OTP is already Redis-backed; do not create an OTP table.
- Current OTP implementation is email-keyed, not phone-keyed.

## Current AI outfit route summary

Source files inspected:

- `internal/api/routes/wardrobe/router.go`
- `internal/modules/wardrobe/presentation/handler/wardrobe_ai_handler.go`
- `internal/modules/wardrobe/application/dto/recommendation.go`

Current route:

```text
POST /api/v1/ai/outfit-recommendations
```

Current route middleware:

- Requires auth.
- Requires `roleslug.User`.

Phase impact:

- Future styling/brand integration should extend this existing endpoint.
- Do not create a parallel AI outfit recommendation endpoint unless a phase explicitly requires it.

## Current search, vector, and Elastic summary

SQL/vector search files inspected:

- `internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo_hybrid.go`
- `internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo_hybrid_helpers.go`
- `init-db/02-indexes.sql`

Current SQL hybrid retrieval:

- Filters `wardrobe_items.user_id`, `wardrobe_items.status`, and `wardrobe_items.is_deleted`.
- Joins `categories` through `wardrobe_items.category_id`.
- Vector search orders by `wardrobe_items.embedding <=> ?`.
- Lexical search builds `to_tsvector` from:
  - `wardrobe_items.color`
  - `wardrobe_items.style`
  - `wardrobe_items.material`
  - `wardrobe_items.pattern`
  - `wardrobe_items.fit`
  - `wardrobe_items.seasonality`
  - `wardrobe_items.description`

Elasticsearch files inspected:

- `internal/shared/infrastructure/search/elasticsearch.go`
- `internal/modules/wardrobe/infrastructure/search/wardrobe_search.go`
- `internal/modules/wardrobe/infrastructure/search/wardrobe_index.go`
- `internal/modules/wardrobe/application/usecase/wardrobe/search_sync/search_sync_uc.go`
- `internal/modules/wardrobe/presentation/worker/search_sync_worker.go`

Current Elasticsearch behavior:

- Index name: `wardrobe_items`.
- Index document ID: `wardrobe_item.id`.
- Indexed document fields are denormalized from `WardrobeItem` metadata.
- Search targets `wardrobe_items` index.
- Search filters `item_type = 1` for system catalog.
- Search sync worker indexes only system catalog items (`item.ItemType == 1`).

AI context and embedding files inspected:

- `internal/modules/wardrobe/application/usecase/wardrobe/shared/fashion_helper.go`
- `internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go`
- `internal/modules/wardrobe/application/usecase/wardrobe/worker/worker_uc.go`
- `internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/candidate_helpers.go`
- `internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/ranking/pool_expansion.go`

Current behavior:

- Item rich text context is built from `WardrobeItem` metadata.
- Item embeddings are generated by shared AI service and stored on `wardrobe_items.embedding`.
- Recommendation query embedding is generated through shared AI service.
- Fallback/ranking code reads metadata from `WardrobeItem`.

Phase impact:

- Phase 03 must move SQL lexical search, vector search, AI context building, and Elasticsearch indexing/search to `fashion_items` while preserving ownership filtering through `wardrobe_items`.

## Risks / unknowns

- Phase 01 should preserve existing `users.email NOT NULL` and `users.password_hash NOT NULL`; MVP no longer moves to phone-first identity.
- Current auth is email/username + password with email OTP confirmation and should remain the MVP auth flow.
- Redis OTP is currently keyed by email; phone OTP requires extending the Redis OTP service/keying behavior without creating a DB OTP table.
- `community` module and community/resale tables are still present; archive must follow Phase 02, not Phase 00.
- `brand` module and brand tables do not exist yet.
- `fashion_items` does not exist yet, so Phase 03 requires migration/backfill plus compatibility reads.
- `outfit_items` currently has no surrogate ID and no `item_context`; Phase 04 must account for composite primary key behavior.
- Elasticsearch currently indexes only system catalog wardrobe items; Phase 03 must decide whether the index name remains `wardrobe_items` with `fashion_item_id` denormalized or moves to a new index with a compatibility plan.
