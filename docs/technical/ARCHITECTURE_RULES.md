# ARCHITECTURE & TECHNICAL DEVELOPMENT RULES

## I. SYSTEM ARCHITECTURE PARADIGM

The system is built strictly upon a **Pragmatic Modular Monolith** framework aligned with **Clean Architecture** boundaries where each module is isolated into self-contained layer packages within a unified Go application.

* **Database-First Approach:** All data schemas and relational boundaries are managed directly at the database tier. The application layer consumes entities generated from the database state.
* **No Domain-Driven Design (DDD):** The codebase strictly avoids DDD abstractions. Do not implement Aggregate Roots, Value Objects, or Domain Events at the entity level.
* **Centralized Data Context:** The entire system utilizes exactly **one unified Postgres database connection and GORM model pool** located in the Shared infrastructure package (`internal/shared/infrastructure/db`) to retain absolute foreign key relational integrity across modules.

---

## II. COMPLIANT PROJECT DIRECTORY TREE

The code workspace separates layer responsibilities into physical Go package boundaries. Implement and position all packages, code components, and dependency mappings precisely according to the following structural schema:

```text
📁 root/smart-wardrobe-be
│
├── 📁 cmd
│    └── 📁 server
│         └── 📄 main.go            # Entry point of the web server
│
├── 📁 config
│    ├── 📄 config.go              # Structured application config definition
│    └── 📄 config.handler.go      # Dynamic config loader (Environment mapping)
│
├── 📁 docs
│    ├── 📁 technical              # Technical specifications & guidelines
│    ├── 📁 business               # Business features & algorithms overview
│    └── 📄 index.html             # High-end Swagger UI static bundle
│
├── 📁 internal
│    ├── 📁 api
│    │    ├── 📁 middleware         # Dynamic global middlewares (Timeout, RateLimit, Auth)
│    │    └── 📁 routes             # Endpoint routing setups grouped by resource folders
│    │
│    ├── 📁 bootstrap
│    │    └── 📄 app.go             # Application bootstrapper
│    │
│    ├── 📁 di
│    │    ├── 📄 wire.go            # Wire build configurations (DI target signatures)
│    │    └── 📄 wire_gen.go        # Auto-generated Dependency Injection graph
│    │
│    ├── 📁 modules
│    │    ├── 📁 identity           # Identity & Authentication modular boundary
│    │    │    ├── 📁 domain        # Core repositories contract interfaces
│    │    │    ├── 📁 application   # Handlers data flows, Usecase execution
│    │    │    ├── 📁 infrastructure # GORM Postgres DB implementations, caching, etc.
│    │    │    ├── 📁 presentation  # Request handlers (controllers / API endpoints)
│    │    │    └── 📄 provider.go   # Module-level Google Wire provider aggregator
│    │    │
│    │    └── 📁 subscription       # Subscription modular boundary (loose coupling via contract)
│    │
│    └── 📁 shared
│         ├── 📁 application        # Shared app structures, global apperror definitions
│         ├── 📁 domain
│         │    ├── 📁 constants     # General enums and status identifiers
│         │    └── 📁 entities      # Unified GORM models (all 16 relational database tables mapped here)
│         │
│         ├── 📁 infrastructure
│         │    ├── 📁 db            # Centralized database connections (GORM DB instance pool)
│         │    └── 📁 repositories  # Generic Repository implementations (base CRUD using Go generics)
│         │
│         └── 📁 presentation       # Shared API Response envelope (`Success`, `Created`, `WrapHandler`)
│
└── 📁 pkg
     └── 📁 logger                  # Log Interface & implementations
```

---

## III. CORE IMPLEMENTATION DESIGN PATTERNS

### 1. Centralized Database Context & Unified GORM Models

The core engine enforces exactly one persistent access gateway. Individual module domain entities must never establish localized database context structures. All database structures are centralized inside `internal/shared/domain/entities`.

- **Entity Model Composition (Located in Shared Domain):**

```go
package entities

import (
	"time"
	"github.com/google/uuid"
)

// BaseEntity acts as the primary key and audit log blueprint
type BaseEntity struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

// AuditableEntity extends base with update tracking
type AuditableEntity struct {
	BaseEntity
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

// User represents the centralized model for user records
type User struct {
	AuditableEntity
	Username           string            `gorm:"type:varchar(255);not null"`
	Email              string            `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash       string            `gorm:"type:varchar(255);not null"`
	SubscriptionPlanID uuid.UUID         `gorm:"type:uuid;not null"`
	SubscriptionPlan   *SubscriptionPlan `gorm:"foreignKey:SubscriptionPlanID;constraint:OnDelete:RESTRICT"`
	IsDeleted          bool              `gorm:"type:boolean;not null;default:false"`
}
```

---

### 2. Extensible Generic Repository Pattern (Go Generics)

Standard infrastructure repositories inherit CRUD operations from the shared generic repository layer while extending custom query specifications for complex database query routines within their specific contract interfaces.

- **Shared Generic Repository Contract & Implementation:**

```go
package repositories

import (
	"context"
	"gorm.io/gorm"
)

type GenericRepository[T any, K any] struct {
	DB *gorm.DB
}

func NewGenericRepository[T any, K any](db *gorm.DB) *GenericRepository[T, K] {
	return &GenericRepository[T, K]{DB: db}
}
```

- **Domain Repository Interface (Located in Module Domain):**

```go
package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

type IUserRepository interface {
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	IsEmailExists(ctx context.Context, email string) (bool, error)
}
```

- **Infrastructure Implementation using Composition (Located in Module Infrastructure):**

```go
package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	shared_persist.GenericRepository[entities.User, uuid.UUID]
}

func NewUserRepository(db *gorm.DB) repositories.IUserRepository {
	return &UserRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.User, uuid.UUID](db),
	}
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	err := r.GenericRepository.DB.WithContext(ctx).Where("email = ? AND is_deleted = ?", email, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
```

---

## IV. STRICT AGENT CODE-GENERATION LAWS

* **Comment Decoration Prohibition:** Absolutely DO NOT inject ordered list sequence numbering arrays (e.g., `1.`, `2.`, `01.`, `Step 1:`) inside codebase comment expressions. Use pure textual characters, functional headers, or horizontal line layouts (e.g., `// ===`, `// ---`) to detail method blocks.
* **Anemic Handlers & Presentation Boundaries:** Presentational Handlers must remain highly anemic, delegating execution immediately by calling the Usecases inside the `Application` layer. All Feature domain logic, repository operations, validation structures, and transactional loops must live strictly inside the module's `application/usecase` folder.
* **Module Communication Guardrails:** Direct reference links across internal functional spaces of separate business modules are entirely blocked. All communication routines must path through exposed interface pipelines declared within target module `contract` packages. Direct domain entity sharing is strictly prohibited.

