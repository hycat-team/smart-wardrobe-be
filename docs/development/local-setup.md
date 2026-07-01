# SmartWardrobe Backend - Development & Technical Guidelines

This document describes the current development conventions that match the repository as it exists today.

The `identity` module remains the cleanest baseline for handler style, use case interfaces, repository separation, and DI layering. However, not every module mirrors its exact folder and provider layout.

---

## 1. Current module structure standard

Each business area lives under `internal/modules/<module_name>` and should keep clear separation between domain, application, infrastructure, and presentation concerns.

Typical folders present in the current codebase:

```text
internal/modules/<module_name>
|
|-- application
|   |-- dto
|   |-- errors
|   |-- interface
|   |   `-- usecase
|   |-- mapper
|   `-- usecase
|
|-- contract                # Current cross-module contract location
|-- domain
|   `-- repositories
|-- infrastructure
|   |-- persistence
|   |-- caching             # Present where needed
|   |-- communication       # Present where needed
|   |-- messaging           # Present where needed
|   |-- payment             # Present where needed
|   |-- search              # Present where needed
|   `-- security            # Present where needed
|-- presentation
|   |-- handler
|   `-- worker              # Present where needed
`-- provider.go
```

Important notes:

- `contract/` currently sits at module root, not under `application/contract/`.
- `presentation/provider.go` only exists in some modules. Other modules wire handlers/workers directly from module `provider.go`.
- `worker/` is a normal presentation adapter in this codebase and should be documented that way.

---

## 2. Dependency injection guidelines

Google Wire is the project-wide DI mechanism.

### Layered pattern still preferred

When a module already has sub-layer provider sets, preserve them:

- `application.ProviderSet`
- `infrastructure.ProviderSet`
- `presentation.ProviderSet`

This is the clearest pattern and `identity` is the best reference.

### Flattened pattern is currently valid

Some modules currently expose a single `provider.go` that aggregates:

- repositories
- external adapters
- use cases
- handlers
- workers

This is a valid style. Docs and code generation should not assume every module already has per-layer provider files.

### Global DI registration

`internal/di/wire.go` is the composition root. It wires:

- shared services
- all active modules (listed by imports in `wire.go`)
- middleware
- route groups
- runtime workers bundled in `bootstrap.AppWorkers`

Modules no longer in runtime (archived) are excluded from `wire.go` but may remain in the tree.

---

## 3. Handler standards

These rules are already reflected in the codebase and should remain mandatory for HTTP handlers.

### Do not write raw success envelopes manually

Prefer shared presentation helpers:

```go
shared_pres.Success(c, "Thông báo thành công", data)
shared_pres.Created(c, "Tạo mới thành công", data)
```

### Handlers return `error`

Standard handler signature:

```go
func (h *MyHandler) MyMethod(c *gin.Context) error
```

- Return `nil` after writing success output.
- Return the error directly on failure.
- Register handlers via `shared_pres.WrapHandler(...)`.

### Exception in current engine

There is still a direct `c.JSON(...)` call in the `/api/v1/health` route inside `internal/api/routes/router.go`. Treat this as an infrastructure exception for the health endpoint, not a new pattern for feature handlers.

---

## 4. Swagger annotation standards

Swagger comments in handlers should stay in Vietnamese and describe the real route paths currently exposed by routers.

Use this shape:

```go
// @Summary Đăng ký tài khoản
// @Description Đăng ký tài khoản mới cho người dùng và gửi OTP xác thực qua email
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.RegisterReq true "Thông tin đăng ký"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/register [post]
```

When response payloads are structured, continue using:

```go
// @Success 200 {object} shared_pres.APIResponse{data=dto.UserRes}
```

---

## 5. Validation and error handling

### Input validation

Request binding and validation should happen in handlers with translated messages:

```go
var input dto.RegisterReq
if err := c.ShouldBindJSON(&input); err != nil {
	return validation.TranslateValidationError(err)
}
```

### Error propagation

Use `apperror` types from `internal/shared/application/constants/apperror` in deeper layers and let centralized middleware map them to HTTP responses.

Handlers should not manually craft error JSON responses.

---

## 6. Routing conventions

The current route layout includes both public and authenticated resource groups. When adding new endpoints, match existing patterns:

- Use `/me/...` for resources owned by the authenticated user.
- Keep admin-only endpoints under `/api/v1/admin/...`.
- Keep AI endpoints under `/api/v1/ai/...`.
- Each business module registers its own route group under `/api/v1/<name>/...`.

To see the current route groups, check `internal/api/routes/` or `internal/api/routes/router.go`.

---

## 7. Search, messaging, and async workflow conventions

The repository includes asynchronous and search-oriented components that must be treated as standard architecture:

- RabbitMQ for event-driven background processing
- Elasticsearch for search indexing and querying
- Scheduled workers for renewal, expiry, reconciliation, AI processing, and cleanup

Current workers are discoverable in `bootstrap.AppWorkers` at `internal/bootstrap/app.go`.

Feature docs should mention these adapters whenever a flow depends on them.

---

## 8. Architectural guardrails

### Use case interface separation

Presentation code should depend on interfaces from `application/interface/usecase` where those interfaces exist.

### Module isolation

Use module contracts for cross-module collaboration. Current contract packages are discoverable under `internal/modules/<name>/contract/`.

### Shared entities are allowed by design

This codebase intentionally centralizes GORM entities in `internal/shared/domain/entities`. Module isolation here is enforced by repository and use case boundaries, not by duplicating entity definitions per module.

### Naming conventions

Prefer `Get...` over `Find...` for repository lookups when creating or refactoring code.

### Comment conventions

- Source-code comments should be plain text.
- Avoid numbered comment blocks.
- Avoid emojis in code comments.

### Language conventions

- Vietnamese: Swagger text and client-facing messages
- English: internal comments and logger messages
