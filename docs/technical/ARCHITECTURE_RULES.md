# ARCHITECTURE & TECHNICAL DEVELOPMENT RULES

## I. SYSTEM ARCHITECTURE PARADIGM

The system is built strictly upon a **Pragmatic Modular Monolith** framework aligned with **Clean Architecture** boundaries where each layer within a module is isolated into its own dedicated C# project (`.csproj`).

- **Database-First Approach:** All data schemas and relational boundaries are managed directly at the database tier. The application layer consumes entities generated from the database state.
- **No Domain-Driven Design (DDD):** The codebase strictly avoids DDD abstractions. Do not implement Aggregate Roots, Value Objects, or Domain Events at the entity level.
- **Centralized Data Context:** The entire system utilizes exactly **one unified `DbContext**` located in the Shared infrastructure project to retain absolute foreign key relational integrity across modules.

---

## II. COMPLIANT PROJECT DIRECTORY TREE

The code workspace separates layer responsibilities into physical project boundaries (`.csproj`). Implement and position all target projects, code component groups, and namespace properties precisely according to the following structural schema:

SmartWardrobe is an example solution's name, please ask for the correct one

```text
📁 Solution: SmartWardrobe
│
├── 📁 Shared
│    ├── 📁 SmartWardrobe.Shared
│    │    ├── 📁 AppExceptions (BusinessException, TooManyRequestsException)
│    │    └── 📁 Constants (JwtSettings, RoleSlugs)
│    │
│    ├── 📁 SmartWardrobe.Shared.Domain
│    │    ├── 📁 Entities (BaseEntity, IAuditableEntity)
│    │    │    ├── 📄 User.cs
│    │    │    ├── 📄 WardrobeItem.cs
│    │    │    ├── 📄 Post.cs
│    │    │    └── (All 16 structural EF Core database models mapped here)
│    │    └── 📁 Repositories
│    │         └── 📄 IGenericRepository.cs
│    │
│    └── 📁 SmartWardrobe.Shared.Infrastructure
│         ├── 📄 AppDbContext.cs (Single context containing all 16 DbSets)
│         └── 📁 Persistence
│              └── 📄 GenericRepository.cs (Implements base CRUD operations)
│
├── 📁 Modules
│    ├── 📁 Identity
│    │    ├── 📁 SmartWardrobe.Identity.Contract
│    │    │    ├── 📁 Interfaces (IIdentityModuleContract, IPublicUserService)
│    │    │    └── 📄 SmartWardrobe.Identity.Contract.csproj
│    │    │
│    │    ├── 📁 SmartWardrobe.Identity.Domain
│    │    │    ├── 📁 Enums (Gender.cs)
│    │    │    ├── 📁 Repositories (IUserRepository.cs -> Formulates core data contracts)
│    │    │    └── 📄 SmartWardrobe.Identity.Domain.csproj
│    │    │
│    │    ├── 📁 SmartWardrobe.Identity.Application
│    │    │    ├── 📁 Features (Commands/Queries/Handlers grouped by Feature folders)
│    │    │    ├── 📁 DTOs (AuthRequest, UserResponse)
│    │    │    ├── 📁 Extensions (DependencyInjection.cs)
│    │    │    └── 📄 SmartWardrobe.Identity.Application.csproj
│    │    │
│    │    ├── 📁 SmartWardrobe.Identity.Infrastructure
│    │    │    ├── 📁 Persistence
│    │    │    │    └── 📁 Repositories (UserRepository.cs -> Inherits GenericRepository)
│    │    │    ├── 📁 Extensions (DependencyInjection.cs)
│    │    │    └── 📄 SmartWardrobe.Identity.Infrastructure.csproj
│    │    │
│    │    └── 📁 SmartWardrobe.Identity.Presentation
│    │         ├── 📁 Controllers (AuthController.cs, MeController.cs)
│    │         ├── 📁 Extensions (DependencyInjection.cs)
│    │         └── 📄 SmartWardrobe.Identity.Presentation.csproj
│    │
│    ├── 📁 Billing (Follows identical 5-project separation structure)
│    ├── 📁 WardrobeAI (Follows identical 5-project separation structure)
│    └── 📁 Community (Follows identical 5-project separation structure)
│
└── 📁 SmartWardrobe.WebApi (Main Startup Executable Gateway Project)
     ├── 📁 Extensions (DependencyInjectionExtensions, RateLimitingExtensions)
     ├── 📁 Middlewares (GlobalExceptionHandler)
     └── 📄 Program.cs (Invokes modular dependency registration layer hooks)

```

---

## III. CORE IMPLEMENTATION DESIGN PATTERNS

### 1. Centralized Database Context Boundary

The core engine enforces exactly one persistent access gateway. Individual module domain entities must never establish localized database context structures.

```csharp
namespace SmartWardrobe.Shared.Infrastructure;

public class AppDbContext : DbContext
{
    public AppDbContext(DbContextOptions<AppDbContext> options) : base(options) { }

    public DbSet<User> Users => Set<User>();
    public DbSet<WardrobeItem> WardrobeItems => Set<WardrobeItem>();
    public DbSet<Post> Posts => Set<Post>();
    // Contains all remaining 13 database model mappings

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        base.OnModelCreating(modelBuilder);
        // Applies configurations for constraint parameters and index mappings
    }
}

```

### 2. Extensible Generic Repository Pattern

Standard infrastructure repositories inherit boilerplate routines from the shared abstract repository layer while extending custom query specifications for heavy vector loops within their specific contract interfaces.

- **Domain Contract Interface (Located in Local Module Domain Project):**

```csharp
namespace SmartWardrobe.Modules.WardrobeAI.Domain.Repositories;

public interface IWardrobeRepository : IGenericRepository<WardrobeItem>
{
    // Custom vector operation signature bypasses pure LINQ limitations
    Task<List<WardrobeItem>> GetClosestWardrobeItemsAsync(Guid userId, float[] vector, int limit);
}

```

- **Infrastructure Data Implementation (Located in Local Module Infrastructure Project):**

```csharp
namespace SmartWardrobe.Modules.WardrobeAI.Infrastructure.Persistence.Repositories;

public class WardrobeRepository : GenericRepository<WardrobeItem>, IWardrobeRepository
{
    private readonly AppDbContext _context;

    public WardrobeRepository(AppDbContext context) : base(context)
    {
        _context = context;
    }

    public async Task<List<WardrobeItem>> GetClosestWardrobeItemsAsync(Guid userId, float[] vector, int limit)
    {
        var targetVectorString = $"[{string.Join(",", vector)}]";

        // Triggers pgvector hardware index directly via raw SQL querying extensions through shared context
        return await _context.WardrobeItems
            .FromSqlRaw("SELECT * FROM wardrobe_items WHERE user_id = {0} AND is_deleted = false ORDER BY embedding <=> {1}::vector LIMIT {2}",
                userId, targetVectorString, limit)
            .AsNoTracking()
            .ToListAsync();
    }
}

```

---

## IV. STRICT AGENT CODE-GENERATION LAWS

- **Comment Decoration Prohibition:** Absolutely DO NOT inject ordered list sequence numbering arrays (e.g., `1.`, `2.`, `01.`, `Step 1:`) inside codebase comment expressions. Use pure textual characters, functional headers, or horizontal line layouts (e.g., `// ===`, `// ---`) to detail method blocks.
- **MediatR Separation Boundaries:** Controllers must remain highly anemic, delegating execution immediately by dispatching internal requests. All Feature command/query descriptors, validation pipelines, and processing handler classes must live strictly inside the local module's `Application/Features` folder block.
- **Module Communication Guardrails:** Direct reference links across internal functional spaces of separate business modules are entirely blocked. All communication routines must path through exposed interface pipelines declared within target `[ModuleName].Contract` projects.
