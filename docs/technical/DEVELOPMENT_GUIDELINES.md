# 🛠️ SmartWardrobe Backend - Development & Technical Guidelines

This document serves as the mandatory **Development & Technical Guidelines (Development Handbook)** that all developers and AI agents must follow when building new modules, features, or components for the SmartWardrobe Backend.

The **Identity Module** is defined as the baseline / gold standard for module directory structure, data flow, error handling, Swagger API documentation annotations, and Dependency Injection patterns.

---

## 📁 1. Standard Module Directory Structure (Identity as Baseline)

Each module must be encapsulated inside `internal/modules/<module_name>` and partitioned into isolated layers adhering strictly to Clean Architecture principles:

```text
📁 internal/modules/identity
│
├── 📁 domain                      # 1. Core Domain Layer (Nghiệp vụ cốt lõi)
│    └── 📁 repositories           # Repository interfaces defining data persistence contracts (e.g., user_repo.go)
│
├── 📁 application                 # 2. Application Layer (Tầng Ứng dụng)
│    ├── 📁 contract               # Cross-module communication interfaces & implementations (Loose Coupling)
│    ├── 📁 dto                    # Data Transfer Objects (Request/Response structs)
│    ├── 📁 mapper                 # Data transformers (e.g., Entity <-> DTO mapper mappings)
│    ├── 📁 usecase                # Core application usecase flows (AuthUseCase, UserUseCase)
│    └── 📄 provider.go            # Application layer specific Wire Provider Set
│
├── 📁 infrastructure              # 3. Infrastructure Layer (Tầng Hạ tầng)
│    ├── 📁 persistence            # Concrete GORM repository implementations for Postgres DB
│    ├── 📁 caching                # Redis caching services (e.g., OTP storage, token blacklisting)
│    ├── 📁 security               # Security services (e.g., Bcrypt password hashing, token verification)
│    ├── 📁 communication          # External communication integrations (Gmail SMTP, SMS)
│    └── 📄 provider.go            # Infrastructure layer specific Wire Provider Set
│
├── 📁 presentation                # 4. Presentation Layer (Tầng Giao tiếp)
│    ├── 📁 handler                # Request handlers and routes payload bindings (AuthHandler, MeHandler)
│    └── 📄 provider.go            # Presentation layer specific Wire Provider Set
│
└── 📄 provider.go                 # 5. Outermost Module Provider (Gom nhóm Provider toàn module)
```

---

## 🏗️ 2. Dependency Injection Guidelines (Google Wire)

Do not manually instantiate dependencies or create cyclic provider setups. Each module must implement a structured, hierarchical provider pattern:

1. **Sub-Layers (`application`, `infrastructure`, `presentation`)**: Package their internal services into a layer-specific `ProviderSet` variable within their respective `provider.go` files.
2. **Outermost Module `provider.go`**: Group the sub-layer `ProviderSet`s into a single module-level set:

    ```go
    package identity

    import (
        "smart-wardrobe-be/internal/modules/identity/application"
        "smart-wardrobe-be/internal/modules/identity/infrastructure"
        "smart-wardrobe-be/internal/modules/identity/presentation"
        "github.com/google/wire"
    )

    var ProviderSet = wire.NewSet(
        presentation.ProviderSet,
        application.ProviderSet,
        infrastructure.ProviderSet,
    )
    ```

3. **Global DI registration**: Register the module-level `identity.ProviderSet` inside `internal/di/wire.go`.

---

## 📡 3. Handler Standards (Presentation Layer)

This is a **mandatory** standard when implementing API Handlers. It guarantees API response consistency, centralized error handling, and robust Swagger auto-generation.

### 🚫 Rule 1: Never Call `c.JSON()` Directly
To ensure all API endpoints return a unified response envelope, Handlers **are strictly prohibited** from calling Gin's `c.JSON(http.StatusOK, ...)` directly. Instead, call the standard presentation functions defined in the `shared_pres` package (`smart-wardrobe-be/internal/shared/presentation`):

- **Standard Success Response (`200 OK`)**:
    ```go
    shared_pres.Success(c, "Success message", data)
    ```
- **Created Success Response (`201 Created`)**:
    ```go
    shared_pres.Created(c, "Resource created successfully", data)
    ```

### 🚫 Rule 2: Handlers Must Return `error` & Use `WrapHandler`
Every Handler method must comply with the standard signature: `func (h *MyHandler) MyMethod(c *gin.Context) error`.

- **On Success**: Return `nil` at the end of the method after calling `shared_pres.Success`.
- **On Failure**: Return the error object directly (`return err`). Centralized middleware will capture and serialize the error response. Never write an error response or capture errors manually inside a Handler.
- **Route Registration**: Handlers must be wrapped using `shared_pres.WrapHandler`:
    ```go
    group.POST("/login", shared_pres.WrapHandler(h.authHandler.Login))
    ```

---

## 📝 4. Swagger Annotations Standards (swag)

Every Handler method must include declarative Swagger comments in Vietnamese (or English as requested by team setup) to populate Swagger UI accurately. Follow this template strictly:

```go
// Register register a new user account
// @Summary Đăng ký tài khoản               <- Short functional summary
// @Description Đăng ký tài khoản mới cho người dùng và gửi OTP xác thực qua email <- Detailed description
// @Tags Auth                               <- API Grouping tag (Auth, Me, etc.)
// @Accept json                              <- Request Content-Type
// @Produce json                             <- Response Content-Type
// @Param body body dto.RegisterReq true "Thông tin đăng ký"  <- Input payload schema and desc
// @Success 200 {object} shared_pres.APIResponse <- Success response envelope schema
// @Router /api/v1/auth/register [post]     <- HTTP method and path mapping
func (h *AuthHandler) Register(c *gin.Context) error { ... }
```

> [!IMPORTANT]
> For endpoints that return specific payload models, the `@Success` annotation must clearly specify the DTO mapping to let Swagger render precise response documentation:
> `// @Success 200 {object} shared_pres.APIResponse{data=dto.UserRes} "User profile information"`

---

## ⚠️ 5. Error Handling & Validation Standards

### A. Input Payload Validation
All incoming client JSON payloads must be validated at the Handler layer using Gin's binding tags (`c.ShouldBindJSON`). When validation fails, translate the native errors to user-friendly messages using the custom utility `validation.TranslateValidationError`:

```go
var input dto.RegisterReq
if err := c.ShouldBindJSON(&input); err != nil {
    // Translate validation rules into readable validation errors and return to WrapHandler
    return validation.TranslateValidationError(err)
}
```

### B. Business / Domain Exception Propagation
Deeper layers (`Usecase`, `Service`, `Repository`) must bubble up exceptions using the structured types provided by the `errorcode` package (`smart-wardrobe-be/internal/shared/application/constants/errorcode`):

- **Not Found Errors**:
    ```go
    return errorcode.NewNotFound("User not found with the provided ID.")
    ```
- **Conflict / Already Exists Errors**:
    ```go
    return errorcode.NewConflict("This email address is already registered.")
    ```
- **Unauthorized / Session Errors**:
    ```go
    return errorcode.NewUnauthorized("Session expired. Please log in again.")
    ```
- **Bad Request / Validation Errors**:
    ```go
    return errorcode.NewBadRequest("Incorrect OTP verification code.")
    ```
- *And other types available in the `errorcode` package...*

The underlying `WrapHandler` mechanism working together with the `GlobalErrorHandler` middleware will automatically catch these exceptions, map them to their corresponding HTTP status codes, and serialize them into a unified JSON format for the client. Handlers must never perform manual HTTP status parsing!
