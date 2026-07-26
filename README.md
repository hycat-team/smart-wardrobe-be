# SmartWardrobe Backend (Closy) — Golang Modular Monolith

Backend API for a smart wardrobe management platform, featuring AI-powered outfit recommendations, subscription management, and identity services. Built with Go following **Modular Monolith + Clean Architecture** principles.

---

## Tech Stack

| Component | Technology |
|---|---|
| Language | **Go 1.25.7** |
| HTTP Framework | **Gin** |
| ORM | **GORM** |
| DI Container | **Google Wire** |
| Database | **PostgreSQL 17 + pgvector** |
| Cache | **Redis 7** |
| Message Queue | **RabbitMQ** |
| Search Engine | **Elasticsearch 8** |
| Migration | **Goose** |
| API Docs | **Swagger (Swaggo)** |
| Config | **Viper** |
| Logging | **Zap** |
| Payment | **PayOS** |
| Media | **Cloudinary** |
| CI/CD | **GitHub Actions** |
| Deploy | **Docker multi-stage build + Docker Compose + Nginx + Certbot** |

---

## Prerequisites

- **Go** `>= 1.25.7` ([golang.org/dl](https://golang.org/dl/))
- **Docker & Docker Compose** — to run PostgreSQL, Redis, RabbitMQ, Elasticsearch
- **GNU Make** — built-in on macOS/Linux; on Windows use Chocolatey (`choco install make`) or Git Bash/WSL

---

## Local Setup

### 1. Clone & configure

```bash
cd smart-wardrobe-be
cp .env.example .env
```

Edit `.env` to match your environment.

### 2A. Run infrastructure only (recommended)

```bash
make docker-dev-up
```

This starts **PostgreSQL + Redis + RabbitMQ + Elasticsearch** (defined in `deployments/docker-compose.yml`).

Then run the Go backend:

```bash
make dev       # generate wire + swagger, build & run
```

### 2B. Or run the full stack

```bash
make docker-dev-up    # includes backend container
```

### 3. Install Go tools

```bash
make install-tools     # Wire + Swag CLI (pinned version)
make install-goose     # Goose CLI (migration)
```

---

## Makefile targets

### Dev workflow

```bash
make wire          # Google Wire DI generation
make swagger       # Swagger spec generation
make generate      # wire + swagger
make fmt           # format Go source
make run           # build & run
make dev           # generate + run
make clean         # remove binary + swagger
```

### Quality & test

```bash
make fmt-check     # check formatting
make tidy-check    # check go.mod tidiness
make generate-check# verify generated files are up-to-date
make vet           # go vet
make test          # go test ./...
make build         # build binary
make check         # CI suite: fmt-check → tidy-check → generate-check → vet → test → build
```

### Docker

```bash
make docker-build            # build production image
make docker-dev-up/down/stop # dev stack
make docker-prod-up/down/stop# production stack
```

### Migration (Goose)

```bash
make migration-create name=xxx  # create a new migration file
make migration-status           # check migration status
make migration-up               # apply pending migrations
make migration-down             # rollback last migration
make migration-version          # show current version
```

---

## API Documentation

After starting the server, visit:

```
http://localhost:8080/swagger
```

---

## Project Structure

```
.
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── api/                    # HTTP layer (middleware, routes)
│   ├── bootstrap/              # App startup + workers
│   ├── config/                 # Config loader (Viper)
│   ├── di/                     # Google Wire composition root
│   ├── modules/                # 7 business modules:
│   │   ├── aiexecutor/         #   AI execution engine
│   │   ├── brand/              #   Brand & loyalty
│   │   ├── dashboard/          #   Analytics
│   │   ├── fashion/            #   AI outfit recommendation
│   │   ├── identity/           #   Auth, users, OTP
│   │   ├── subscription/       #   Subscription & payment
│   │   └── wardrobe/           #   Wardrobe management
│   ├── pkg/                    # Shared utilities (logger, utils)
│   └── shared/                 # Shared domain, infra, presentation
├── deployments/                # Docker Compose files
├── database/
│   ├── init/                   # Init scripts
│   └── migrations/             # Goose migrations
├── api/swagger/                # Generated Swagger docs
├── configs/                    # YAML config files
├── docs/                       # Project documentation
└── .github/workflows/          # CI + Deploy pipelines
```

Each module in `internal/modules/` follows Clean Architecture:
- `domain/entities/` — domain models
- `domain/repositories/` — repository interfaces
- `application/usecase/` — business logic
- `infrastructure/persistence/` — GORM implementations
- `presentation/handler/` — HTTP handlers
- `presentation/worker/` — background workers

---

## Further Reading

- [Development Guide](docs/development/local-setup.md) — coding conventions, handler standards, routing, validation
- [System Architecture](docs/system-design/architecture.md) — module boundaries, design patterns, event-driven design
- [API docs](docs/api/) — endpoint specifications
- [Changelog](docs/project/changelog.md)
- [Agent Workflow](docs/development/agent-workflow.md)
