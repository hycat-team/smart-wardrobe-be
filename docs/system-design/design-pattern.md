# Design Patterns & Conventions

This document describes the core design patterns, conventions, and utility packages used throughout the project.

---

## Utility Packages (`pkg/utils/`)

### `contextutils` — Request-scoped context values

Extracts authenticated user information from the Gin context set by the auth middleware.

| Function | Returns | When to use |
|---|---|---|
| `GetUserId(c)` | `(uuid.UUID, error)` | Endpoints that **require** authentication |
| `GetUserIdOptional(c)` | `uuid.UUID` | Endpoints where auth is **optional** (returns `uuid.Nil` if not authenticated) |
| `GetRoleSlug(c)` | `(roleslug.RoleSlug, error)` | Endpoints that need the user's role |
| `GetEmail(c)` | `(string, error)` | When the user's email is needed |
| `GetAccessToken(c)` | `(string, error)` | When the raw JWT access token is needed |

**Usage example** (required auth):
```go
userID, err := contextutils.GetUserId(c)
if err != nil {
    return err
}
```

**Usage example** (optional auth):
```go
userID := contextutils.GetUserIdOptional(c)
// userID may be uuid.Nil — check it explicitly if needed
```

### `validation` — Request binding with Vietnamese error messages

| Function | Purpose |
|---|---|
| `BindJSON(c, &obj)` | Binds request body, normalizes Unicode NFC, returns structured `apperror` |
| `BindQuery(c, &obj)` | Binds query params, returns structured `apperror` |

Custom validators registered at startup via `validator` tags:
- `username` — 4–20 alphanumeric, `.`, `_` without consecutive or leading/trailing separators
- `password_complexity` — ≥8 chars with upper, lower, digit, and special character
- `neqfield` — field must differ from another field
- `nfcmax` — max rune count after NFC normalization

**Guideline**: Always use `validation.BindJSON` / `validation.BindQuery` in handlers rather than raw `c.ShouldBindJSON` to get consistent Vietnamese error messages.

### `httputils` — HTTP client helpers

- `DownloadImage(cli, ctx, url)` — downloads image bytes with a 15 MB limit, returns `([]byte, mimeType, error)`.

### `errorutils` — Error mapping and stack trace filtering

| Function | Purpose |
|---|---|
| `ToAppError(err)` | Converts any `error` to `*apperror.Error` with a stack trace |
| `MapErrorToProblem(err)` | Returns `(httpStatus, title, message)` for the HTTP response |
| `FilterStackTraceArray(rawStack)` | Filters Go runtime stack to keep only project-relevant frames |
| `PrimaryStackFrame(rawStack)` | Returns the first meaningful project frame |

### Other utility packages

| Package | Contents |
|---|---|
| `jwtutils` | JWT token validation and parsing |
| `stringutils` | String manipulation helpers |
| `sliceutils` | Slice operation helpers |
| `timeutils` | Time formatting and calculation helpers |
| `streamutils` | Streaming / chunking helpers |
| `colorutils` | Color conversion utilities |

---

## Error Pattern

All business and system errors are centralized through `internal/shared/application/constants/apperror/`. The pattern provides a structured error type, factory constructors, sentinel errors, and domain-level extensions.

### Core error type (`error.go`)

```go
type Error struct {
    Status     int                   `json:"status"`     // HTTP status code
    Title      string                `json:"title"`       // Short human-readable title (Vietnamese)
    Message    string                `json:"message"`     // Detail message for the client
    Errors     []ValidationErrorItem `json:"errors"`      // Field-level validation errors
    StackTrace any                   `json:"stackTrace"`  // Debug stack trace (omitted in production)
    Cause      error                 `json:"-"`           // Wrapped original error
}
```

Key methods:
- `Error()` — implements the `error` interface; returns `Message`, fallback to `Cause.Error()`, fallback to HTTP status text
- `Unwrap()` — returns the wrapped `Cause` for `errors.As`/`errors.Is` compatibility
- `Stack()` — returns the raw stack captured at creation
- `Is(target)` — compares by `Status` + `Message` so sentinel errors can be matched with `errors.Is`
- `WithStackTrace(stackTrace)` — returns a shallow copy with `StackTrace` set (for debug responses)

### Factory constructors (`dynamic_errors.go`)

Each maps an HTTP status to a consistent Vietnamese title:

| Constructor | HTTP Status | Title |
|---|---|---|
| `NewBadRequest(detail)` | 400 | "Thao tác không thành công" |
| `NewUnauthorized(detail)` | 401 | "Lỗi xác thực" |
| `NewForbidden(detail)` | 403 | "Không có quyền truy cập" |
| `NewNotFound(detail)` | 404 | "Không tìm thấy dữ liệu" |
| `NewConflict(detail)` | 409 | "Dữ liệu bị trùng lặp" |
| `NewTooManyRequest(detail)` | 429 | "Quá nhiều yêu cầu" |
| `NewInternalError(detail)` | 500 | "Lỗi hệ thống" |
| `NewServiceUnavailable(detail)` | 503 | "Dịch vụ tạm thời gián đoạn" |

Usage:
```go
return nil, apperror.NewNotFound("Không tìm thấy brand.")
return nil, apperror.NewForbidden("Bạn không có quyền quản trị brand này.")
```

### Sentinel errors (`static_errors.go`)

Common reusable errors are defined as named functions for consistency:

```go
apperror.ErrUnauthorized()      // 401 — "Vui lòng đăng nhập"
apperror.ErrForbidden()         // 403 — "Không có quyền truy cập"
apperror.ErrInvalidId()         // 400 — "Id không hợp lệ"
apperror.ErrBusiness()          // 400 — "Thao tác không thành công"
apperror.ErrConflictDuplicate() // 409 — "Dữ liệu bị trùng lặp"
apperror.ErrInternalServer()    // 500 — "Lỗi không mong muốn. Vui lòng thử lại"
```

### `Wrap` and `From` — Wrapping non-app errors (`error.go`)

```go
// Wrap converts a plain error into *apperror.Error (500) with a stack trace.
// If the error is already *apperror.Error, returns as-is.
apperror.Wrap(err, "fallback message")

// From extracts *apperror.Error from any error, or wraps it as 500.
apperror.From(err)
```

`From` also handles common edge cases:
- UUID parse errors → `NewBadRequest`
- Empty error strings → `NewInternalError`

### Domain-level errors (`internal/<module>/application/errors/`)

Each module defines its own error constructors in a separate `application/errors` package. These should compose the shared factory constructors:

```go
// Example from internal/modules/brand/application/errors/errors.go
package branderrors

func ErrBrandNotFound() *apperror.Error {
    return apperror.NewNotFound("Không tìm thấy brand.")
}

func ErrBrandPortalForbidden() *apperror.Error {
    return apperror.NewForbidden("Bạn không có quyền quản trị brand này.")
}

func ErrBrandNotActive() *apperror.Error {
    return apperror.NewForbidden("Brand chưa active hoặc đã bị khóa.")
}

func ErrInvalidBrandStatus(status any) *apperror.Error {
    return apperror.NewBadRequest(fmt.Sprintf("Trạng thái brand không hợp lệ: %v.", status))
}
```

### Global error handler (middleware)

Errors returned from handlers flow through the global error handler registered in the Gin engine. The middleware:
1. Receives the error via `c.Error(err)`
2. Converts it with `apperror.From(err)` to ensure type consistency
3. Returns a structured JSON response including stack trace in non-production environments

### Validation errors

When `validation.BindJSON` or `validation.BindQuery` fails, they return `*apperror.Error` with `Errors` populated as field-level items:

```json
{
  "status": 400,
  "title": "Thao tác không thành công",
  "message": "Vui lòng nhập email",
  "errors": [
    { "field": "email", "message": "Vui lòng nhập email" },
    { "field": "password", "message": "mật khẩu không được vượt quá 255 kí tự" }
  ]
}
```

### Best practices

1. **Return domain errors from use cases**, not from handlers or repositories:
   ```go
   // ❌ Repository returning app error
   func (r *XxxRepo) GetByID(...) (*Entity, *apperror.Error)

   // ✅ Repository returns plain error
   func (r *XxxRepo) GetByID(...) (*Entity, error)
   // ✅ Use case translates to domain error
   if item == nil {
       return nil, branderrors.ErrBrandNotFound()
   }
   ```

2. **Use `apperror.From` in the global error handler** to normalize any error type — don't call `From` in every use case.
3. **Sentinel errors for common cases**: Use `apperror.ErrForbidden()`, `apperror.ErrUnauthorized()` directly.
4. **Domain errors for module-specific cases**: Define in `application/errors/` with meaningful Vietnamese messages.
5. **Never expose raw database errors** — always wrap or replace with a user-friendly message.
6. **Validation errors are automatic** — `validation.BindJSON` / `validation.BindQuery` handle translation; no manual mapping needed.

---

## Pagination Pattern

The project uses a reusable generic pagination system built on generics (Go 1.18+).

### Core types (in `internal/shared/application/dto/pagination.go`)

```go
type PaginationQuery struct {
    Page  int `form:"page"`
    Limit int `form:"limit"`
}

func (q PaginationQuery) Normalize() PaginationQuery {
    // defaults: Page=1, Limit=20
}

func (q PaginationQuery) Offset() int {
    return (normalized.Page - 1) * normalized.Limit
}

type PaginationMetadata struct {
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    TotalItems int64 `json:"totalItems"`
    TotalPages int   `json:"totalPages"`
}

type PaginationResult[T any] struct {
    Items    []T                `json:"items"`
    Metadata PaginationMetadata `json:"metadata"`
}

func BuildPaginationMetadata(query PaginationQuery, totalItems int64) PaginationMetadata
```

### Repository-side helper (in `internal/shared/infrastructure/repositories/pagination.go`)

```go
func ApplyPagination(db *gorm.DB, query PaginationQuery) *gorm.DB
```

### Step-by-step recipe

**1. Define a Filter struct** in the domain repository package:

```go
type XxxFilter struct {
    BrandID  uuid.UUID
    Status   *branditemstatus.BrandItemStatus
    Page     int
    Limit    int
}
```

**2. Define a ListResult struct** in the same package:

```go
type XxxListResult struct {
    Items      []*entities.Xxx
    TotalCount int64
}
```

**3. Add the paginated method** to the repository interface:

```go
GetByXxxPaginated(ctx, filter XxxFilter) (*XxxListResult, error)
```

**4. Implement in the persistence layer** — always do `Count` before `Find`:

```go
func (r *XxxRepo) GetByXxxPaginated(ctx context.Context, filter XxxFilter) (*repositories.XxxListResult, error) {
    db := r.GetDB(ctx).Where("brand_id = ?", filter.BrandID)

    var totalCount int64
    if err := db.Model(&entities.Xxx{}).Count(&totalCount).Error; err != nil {
        return nil, err
    }

    paginationQuery := shared_dto.PaginationQuery{Page: filter.Page, Limit: filter.Limit}
    db = shared_persist.ApplyPagination(db, paginationQuery)

    var items []*entities.Xxx
    if err := db.Order("created_at DESC").Find(&items).Error; err != nil {
        return nil, err
    }

    return &repositories.XxxListResult{Items: items, TotalCount: totalCount}, nil
}
```

**5. Add a Query DTO** in the `dto` package:

```go
type GetXxxQueryReq struct {
    shared_dto.PaginationQuery
}

type XxxListRes = shared_dto.PaginationResult[*XxxRes]
```

**6. Bind in the handler**:

```go
var query dto.GetXxxQueryReq
if err := validation.BindQuery(c, &query); err != nil {
    return err
}
```

**7. Call from the use case**:

```go
result, err := uc.repo.GetByXxxPaginated(ctx, filter)
// ...
return &dto.XxxListRes{
    Items:    mapper.MapXxx(result.Items),
    Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
}, nil
```

---

## Handler Pattern

### Error-returning handler

Every handler function returns `error` instead of calling `c.JSON` directly. This keeps error handling centralized.

```go
// handler definition
func (h *XxxHandler) DoSomething(c *gin.Context) error {
    // ... business logic ...
    shared_pres.Success(c, "Thành công", res)
    return nil
}
```

### Wrapping with Gin

The `shared_pres.WrapHandler` adapter converts the error-returning function to a standard `gin.HandlerFunc`:

```go
router.GET("/path", shared_pres.WrapHandler(h.DoSomething))
```

If the handler returns an error, `WrapHandler` calls `c.Error(err)` and aborts, which is then caught by the global error middleware.

### Response helpers (in `internal/shared/presentation/response.go`)

```go
shared_pres.Success(c, message, data)    // 200 OK
shared_pres.Created(c, message, data)    // 201 Created
```

### Auth middleware

| Middleware | Behavior |
|---|---|
| `r.authMiddleware.Handle()` | Requires valid JWT; aborts with 401 if missing/invalid |
| `r.authMiddleware.OptionalHandle()` | Parses JWT if present but does **not** fail if missing; sets `userId` in context when available |

Role authorization is applied as a separate middleware:

```go
portal.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User, roleslug.Admin))
```

### Message constants

Success/error message strings must be declared as constants at the top of the handler file, not hard-coded inline:

```go
const (
    msgBrandItemCreateSuccess = "Tạo sản phẩm brand thành công"
    msgBrandItemGetSuccess    = "Lấy thông tin sản phẩm brand thành công"
)
```

---

## Repository & Generic Repository Pattern

### Generic interface (in `internal/shared/domain/repositories/generic.go`)

```go
type IGenericRepository[T any, ID any] interface {
    GetByID(ctx context.Context, id ID) (*T, error)
    GetAll(ctx context.Context) ([]*T, error)
    Create(ctx context.Context, entity *T) error
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, id ID) error
}
```

### Generic implementation (in `internal/shared/infrastructure/repositories/generic_impl.go`)

`GenericRepository[T, ID]` provides:
- Preload support: relations specified at construction are always eager-loaded
- Transaction awareness: `GetDB(ctx)` checks for a transaction in context (set by UoW)
- `GetQueryWithPreload(ctx)` for custom queries that need the same preloads

### Domain-specific repository

Each domain defines its own interface embedding `IGenericRepository`:

```go
type IBrandItemRepository interface {
    shared_repos.IGenericRepository[entities.BrandItem, uuid.UUID]
    GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandItem, error)
    GetByBrandIDPaginated(ctx context.Context, filter BrandItemFilter) (*BrandItemListResult, error)
}
```

### Domain-specific implementation

The concrete repository embeds `GenericRepository` and adds custom methods:

```go
type BrandItemRepository struct {
    shared_persist.GenericRepository[entities.BrandItem, uuid.UUID]
}

func NewBrandItemRepository(db *gorm.DB) repositories.IBrandItemRepository {
    return &BrandItemRepository{
        GenericRepository: *shared_persist.NewGenericRepository[entities.BrandItem, uuid.UUID](db, []string{"Brand", "FashionItem"}),
    }
}
```

### Best practices

- **Preload at construction**: Specify relations in `NewGenericRepository` so they're always loaded when using inherited methods (`GetByID`, `GetAll`)
- **Custom queries use `GetQueryWithPreload`**: When writing custom query methods, always use `r.GetQueryWithPreload(ctx)` to get the same preload behavior
- **Transaction-safe `GetDB`**: The inherited `GetDB(ctx)` automatically finds and uses an active transaction from context when inside a `UnitOfWork.Execute` block
- **Domain interfaces in `domain/repositories/`**: Interfaces belong in the domain layer, implementations in `infrastructure/persistence/`

---

## Unit of Work Pattern

### Interface (in `internal/shared/domain/repositories/uow.go`)

```go
type IUnitOfWork interface {
    Execute(ctx context.Context, fn func(ctx context.Context) error) error
}
```

### Implementation (in `internal/shared/infrastructure/db/uow.go`)

`GormUnitOfWork.Execute` wraps `fn` in a database transaction:

```go
func (u *GormUnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
    return u.db.Transaction(func(tx *gorm.DB) error {
        txCtx := InjectTx(ctx, tx)
        return fn(txCtx)
    })
}
```

The transaction is injected into context via `InjectTx`. Repository methods automatically detect it through `GetDB(ctx)` → `db.GetTx(ctx)`, so no explicit transaction passing is needed.

### Usage in use cases

```go
if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
    if err := uc.repoA.Create(txCtx, entityA); err != nil {
        return err
    }
    if err := uc.repoB.Update(txCtx, entityB); err != nil {
        return err
    }
    return nil
}); err != nil {
    return nil, err
}
```

All operations inside `fn` share the same database transaction. If any returns an error, the entire transaction is rolled back.

---

## Application, Use Case, and Handler Structure

- Mapper functions converting between entity/domain models and DTOs/responses must be placed in the `application/mapper` folder of the respective module. Do not place mappers in the use case file.
- Within `application/mapper`, it is allowed to split into multiple mapper files by business group to avoid a single file becoming too long.
- Helpers serving only one use case/use case file must be extracted into a separate file in the same package, and the filename must have the `_helper.go` suffix.
- Do not create a helper if the function only wraps a simple expression and does not clarify the business meaning. Prefer inlining or reusing utilities in `pkg/utils`.
- Return messages in the presentation handler must be declared as variables/constants at the top of the handler file following the existing pattern, rather than hard-coding directly in each response.

### Layer responsibilities

```
┌─────────────────────────────────────────────┐
│ presentation/handler/                        │
│   - HTTP request/response (BindJSON, BindQuery)  │
│   - Calls use case, returns result via shared_pres │
│   - Message constants at top of file          │
├─────────────────────────────────────────────┤
│ application/usecase/                         │
│   - Business logic / orchestration           │
│   - Calls repository interfaces              │
│   - Rolls back via UnitOfWork on failure     │
├─────────────────────────────────────────────┤
│ application/mapper/                          │
│   - Entity ↔ DTO conversion                 │
│   - One file per business group              │
├─────────────────────────────────────────────┤
│ application/dto/                             │
│   - Request/response structs                 │
│   - Shared pagination DTOs                   │
├─────────────────────────────────────────────┤
│ application/interface/usecase/               │
│   - Use case interface definitions           │
├─────────────────────────────────────────────┤
│ domain/repositories/                         │
│   - Repository interfaces (Filter, ListResult) │
├─────────────────────────────────────────────┤
│ infrastructure/persistence/                  │
│   - Repository implementations (GORM)        │
└─────────────────────────────────────────────┘
```
