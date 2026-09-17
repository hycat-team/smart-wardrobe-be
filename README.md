# SmartWardrobe Backend (Closy) — 3-Service Go Workspace

Backend API for a smart wardrobe management platform, featuring AI-powered outfit
recommendations, subscription management, and identity services. Built with Go as
**3 independently deployable services** on a multi-module workspace (`go.work`),
following **Clean Architecture** with compiler-enforced service boundaries.

> Monolith (`cmd/server`) is retired from `main` — see the frozen tag
> `monolith-legacy` for emergency reference. Runtime is always the 3-service
> topology (local compose, VPS compose, or Cloud Run).

---

## Services

| Service | Module | Business modules | Hosts gRPC | Calls |
|---------|--------|------------------|------------|-------|
| Account | `services/account` | identity, subscription, admin, notification | Identity, Subscription, Admin | — |
| Commerce Core | `services/commerce` | wardrobe, fashion, brand, dashboard | — | Account, AI |
| AI Engine | `services/ai` | aiexecutor | AIOrchestrator, AIDashboard | Account |

Shared: `shared/` (config, infra, presentation, DTO, pkg) · `contracts/` (proto +
stubs + gRPC clients). See `docs/system-design/microservices/`.

---

## Tech Stack

| Component | Technology |
|---|---|
| Language | **Go 1.25.7** (workspace `go.work`, 6 modules) |
| HTTP Framework | **Gin** (per service) |
| Inter-service RPC | **gRPC + protobuf** (`buf`) |
| ORM | **GORM** |
| DI Container | **Google Wire** (per-service composition) |
| Database | **PostgreSQL 17 + pgvector** (Supabase Pro managed / docker self-host) |
| Cache | **Redis 7** (Redis Cloud TLS managed / docker self-host) |
| Message Queue | **RabbitMQ** (CloudAMQP TLS managed / docker self-host) |
| Search Engine | **Elasticsearch 8 — OFF by default** (`ELASTICSEARCH_ENABLED=false`, DB fallback) |
| Migration | **Goose** (per-service, own version table) |
| API Docs | **Swagger per service** (9 sections combined, behind `swagger_access_code`) |
| Config | **Viper** (env-driven, no hardcoded hosts) |
| Logging | **Zap** (JSON on managed, console locally) |
| Payment / Media | **PayOS** / **Cloudinary** |
| CI/CD | **GitHub Actions** (workspace matrix, Artifact Registry, GHCR for VPS) |
| Deploy | **Cloud Run** (managed) / **Docker Compose + Nginx + Certbot** (VPS) |

---

## Prerequisites

- **Go** `>= 1.25.7`
- **Docker & Docker Compose** — Postgres, Redis, RabbitMQ locally
- **GNU Make** — built-in on macOS/Linux; on Windows use Chocolatey/Git Bash/WSL
- **Buf CLI** (`make install-tools`) — only when changing `contracts/**/*.proto`

---

## Local Setup

### 1. Clone & configure

```bash
cd smart-wardrobe-be
cp .env.example .env
```

### 2. Run infrastructure + services

```bash
make docker-dev-up          # local compose: infra (+ optional service images)
make build-all              # build account, commerce, ai into bin/
make run svc=account        # run one service locally (svc=account|commerce|ai)
make dev                    # generate (wire+swagger) + build all
```

Each service listens on `PORT` (Cloud Run injects it) else `SERVER_PORT`, and
serves `GET /api/v1/health`. Run mode via `APP_MODE=http|worker|all` (default `all`).

### 3. Install Go tools

```bash
make install-tools     # Wire + Swag + Buf CLI (pinned versions)
make install-goose     # Goose CLI (migration)
```

---

## Makefile targets (per-service: `svc=account|commerce|ai`)

```bash
make wire svc= / swagger svc= / build svc= / run svc=   # one service
make wire-all / swagger-all / build-all                # all 3 services
make buf-gen / buf-lint                                # protobuf contracts/
make migrate-up svc= / migrate-create svc= name=xxx    # per-service Goose
make image-build svc= / image-push svc= / deploy svc=  # Cloud Run images
make test / test-race / vet / lint                     # all workspace modules
make check                                             # CI suite
```

---

## API Documentation

Each service serves its own docs (behind `swagger_access_code`); combined they
cover 9 sections (identity, subscription, admin, notification, wardrobe,
fashion, brand, dashboard, aiexecutor):

```
http://localhost:8080/swagger   # per-service, when running locally
```

---

## Project Structure

```text
.
├── go.work                        # 5 modules + token-helper (no monolith)
├── services/account|commerce|ai/  # one Go module each: cmd/<svc>, internal/{bootstrap,di,modules}, api/swagger, database/migrations, Dockerfile
├── shared/                        # config, infra, presentation, DTO, pkg (+ tools)
├── contracts/<owner>/v1/          # *.proto, *.pb.go, client/ (+ grpcx helpers)
├── deployments/                   # docker-compose.yml (local), docker-compose.prod.yml (VPS + rollback), nginx/, token-helper/, monolith-legacy/
├── database/init/                 # infra init SQL (not migrations)
├── configs/                       # shared YAML (parameterised, no hardcoded hosts)
├── docs/                          # development / system-design / operations / project
└── .github/workflows/             # ci.yml, deploy-platform.yml, deploy-vps-services.yml, deploy-production.yml (rollback)
```

Each module follows Clean Architecture (`domain/` ← `application/` ←
`presentation/`, `infrastructure` implements domain). Cross-service calls go
through `contracts/` (gRPC) or `presentation/contract/` DTOs in-process — never
domain entities. See `docs/system-design/microservices/` and the Constitution
(`.specify/memory/constitution.md`).

---

## Operations (managed ⇄ VPS)

- Runbooks: `docs/operations/ha-tang/` (Supabase, Redis, AMQP, Cloud Run, Nginx
  ingress, ES-off, parallel, data, DNS/CI, repatriation, teardown, secret revocation).
- Cutover `api.closy` → Cloud Run is **manual, owner-triggered** with rollback:
  `docs/operations/manual-cutover-api-closy-to-cloudrun.md`.
- Funding clock + repatriation: `docs/project/funding-and-platform-lifecycle.md`.

---

## Further Reading

- [Development Guide](docs/development/local-setup.md) — conventions, handlers, routing, validation
- [System Architecture](docs/system-design/architecture.md) — service topology, boundaries, events
- [Microservices guides](docs/system-design/microservices/) — contracts, gRPC, auth, CI
- [Agent Workflow](docs/development/agent-workflow.md)
