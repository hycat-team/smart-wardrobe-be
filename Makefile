# Config and variables
export

# ---- 020 multi-module workspace (FR-038) ----
# Per-service targets take svc=<account|commerce|ai>, e.g. `make wire svc=account`.
MODULES := shared contracts services/account services/commerce services/ai
WIRE_VERSION := v0.7.0
SWAG_VERSION := v1.16.6
BUF_VERSION := v1.57.2
GO_BUILD_FLAGS := -trimpath -ldflags="-s -w"
BIN_DIR := bin

# Registry conventions (FR-042/FR-051, T007a):
# - VPS + AWS images → GHCR with existing CI token, tag <svc>:<git-sha> (immutable).
# - AR_* below is kept for manual registry builds (image-build/image-push).
AR_HOST ?= asia-southeast1-docker.pkg.dev
AR_PROJECT ?= closy-hycat
AR_REPO ?= closy
GHCR_OWNER ?= hycat
# Redirect to both: `2>/dev/null` for POSIX shells, `2>NUL` for Windows cmd.exe.
# Without the second form, every make run on Windows prints
# "The system cannot find the path specified." and GIT_SHA silently falls back to `dev`.
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null 2>NUL || echo dev)

# Migration (Goose) connection variables (shared by all services)
DB_HOST ?= localhost
DB_USER ?= admin
DB_PASSWORD ?= 123456
DB_NAME ?= smart_wardrobe_db
DB_PORT ?= 5432
DB_SSLMODE ?= disable
DB_TIMEZONE ?= Asia/Ho_Chi_Minh
MIGRATION_DSN := "host=$(DB_HOST) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) port=$(DB_PORT) sslmode=$(DB_SSLMODE) TimeZone=$(DB_TIMEZONE)"

# Host-run defaults for `make run` (multi-module: the binary runs from
# services/<svc>, so point it at the repo config + .env and at infra published
# on localhost). Override SERVER_PORT per service to run several at once, e.g.
#   make run svc=commerce SERVER_PORT=8102
SERVER_PORT ?= 8080
REDIS_HOST ?= localhost
RABBITMQ_HOST ?= localhost
DB_LISTENER_HOST ?= localhost
CONFIG_FILE ?= $(abspath configs/config.yaml)

# RabbitMQ topology reconcile (see the rabbitmq-* targets near nginx-routes).
# RABBITMQ_ENV_FILE points at the env holding the RABBITMQ_* settings: .env for
# the local stack, .env.production on the VPS, .env.aws on the AWS box. QUEUE and
# SPILL_DIR are optional overrides, e.g.
#   make rabbitmq-topology-check RABBITMQ_ENV_FILE=.env.production
#   make rabbitmq-topology-migrate RABBITMQ_ENV_FILE=.env.production QUEUE=subscription_plan_changed_queue
RABBITMQ_ENV_FILE ?= $(or $(env),.env)
QUEUE ?=
SPILL_DIR ?=

# Nginx ingress render (FR-028; runs on VPS)
NGINX_TEMPLATES_DIR := deployments/nginx/templates
NGINX_RENDER_DIR ?= /etc/nginx
# Server-block set to render: services | services-redirect | rollback
NGINX_MODE ?= services

# Extra flags for `make test` / `make test-race` (CI passes TEST_FLAGS=-short).
TEST_FLAGS ?=

ifeq ($(OS),Windows_NT)
GO_BIN_EXT := .exe
MKDIR_BIN_CMD := if not exist "$(subst /,\,$(BIN_DIR))" mkdir "$(subst /,\,$(BIN_DIR))"
CLEAN_BIN_CMD := if exist "$(subst /,\,$(BIN_DIR))" rmdir /S /Q "$(subst /,\,$(BIN_DIR))"
FMT_CHECK_CMD := powershell -Command "$$files = gofmt -l .; if ($$files) { Write-Error ($$files -join [Environment]::NewLine); exit 1 }"
# NOTE: do NOT pipe go test through a filter — the pipeline exit code becomes
# the filter's (findstr/grep return 1 when every line is filtered, e.g. a module
# with no test files) which both false-fails and masks real test failures.
TEST_CMD := go test $(TEST_FLAGS) ./...
RACE_TEST_CMD := go test -race $(TEST_FLAGS) ./...
# Relative path to the built binary when the recipe CWD is services/<svc>.
RUN_BIN = ..\..\$(subst /,\,$(BIN_DIR))\$(SVC)$(GO_BIN_EXT)
else
GO_BIN_EXT :=
MKDIR_BIN_CMD := mkdir -p "$(BIN_DIR)"
CLEAN_BIN_CMD := rm -rf "$(BIN_DIR)"
FMT_CHECK_CMD := test -z "$$(gofmt -l .)"
TEST_CMD := go test $(TEST_FLAGS) ./...
RACE_TEST_CMD := go test -race $(TEST_FLAGS) ./...
# Relative path to the built binary when the recipe CWD is services/<svc>.
RUN_BIN = ../../$(BIN_DIR)/$(SVC)$(GO_BIN_EXT)
endif

# FMT command dùng chung 2 OS (gofmt -w . tự động đệ quy tất cả các file .go)
FMT_CMD := gofmt -w .

# CLI overrides: chấp nhận cả chữ thường (svc=, name= — chuẩn tasks.md, dùng ở CI Linux)
# lẫn chữ hoa (SVC=, NAME= — một số bản make Windows nuốt override chữ thường).
SVC := $(or $(SVC),$(svc))
name := $(or $(name),$(NAME))
# NOTE: bare `export` above forces make to expand every variable for each recipe
# environment — unexport this guard so its $(error) only fires in recipes that use it.
SVC_REQ = $(if $(filter $(SVC),account commerce ai),,$(error This target requires svc=<account|commerce|ai> — e.g. make $@ svc=account))
unexport SVC_REQ

# Per-service goose version table (030 handover, FR-035): each service records
# into its own table so `make migrate-up svc=ai` never pollutes goose_db_version.
ifeq ($(SVC),commerce)
GOOSE_TABLE := goose_db_version_commerce
else ifeq ($(SVC),ai)
GOOSE_TABLE := goose_db_version_ai
else
GOOSE_TABLE := goose_db_version
endif

.PHONY: help dev-help quality-help docker-help migration-help tools-help platform-help install-tools wire wire-all swagger swagger-all swagger-combine generate buf-gen buf-lint hygiene-scan fmt fmt-check tidy tidy-check vet migration-check test test-workspace test-race test-integration lint build build-all run dev check generate-check clean-swagger clean migrate-status migrate-up migrate-down migrate-create migrate-version docker-build image-build image-push image-build-monolith-rollback nginx-reload nginx-check nginx-routes nginx-reload-local docker-dev-up docker-dev-stop docker-dev-down docker-prod-up docker-prod-stop docker-prod-down install-goose rabbitmq-topology-check rabbitmq-topology-migrate image-build-tool image-push-tool

help:
	@echo "========================================================================"
	@echo "SMART WARDROBE BUILD SYSTEM - HELP INDEX (020 multi-module workspace)"
	@echo "========================================================================"
	@echo "Run one of the following commands to list specific targets:"
	@echo "  make dev-help       : Local development workflow targets"
	@echo "  make quality-help   : CI quality checks, tests, and compilation"
	@echo "  make docker-help    : Docker Compose development & production targets"
	@echo "  make migration-help : Database migration (Goose) commands"
	@echo "  make tools-help     : Local development tools installation helper"
	@echo "  make platform-help  : Registry / ingress targets (020)"
	@echo "Per-service targets take svc=<account|commerce|ai>, e.g. make build svc=ai"
	@echo "========================================================================"

dev-help:
	@echo "DEVELOPMENT WORKFLOW COMMANDS:"
	@echo "  make wire svc=        : Generate Wire DI for one service"
	@echo "  make wire-all         : Generate Wire DI for all 3 services"
	@echo "  make swagger svc=     : Generate Swagger for one service"
	@echo "  make swagger-all      : Generate Swagger for all 3 services"
	@echo "  make generate         : Run wire-all + swagger-all"
	@echo "  make buf-gen          : Generate protobuf stubs into contracts/"
	@echo "  make hygiene-scan     : Run codebase hygiene scan (report in specs/)"
	@echo "  make fmt              : Format all Go source files"
	@echo "  make tidy             : Tidy all Go modules (workspace)"
	@echo "  make build svc=       : Build one service binary into bin/"
	@echo "  make build-all        : Build all 3 service binaries"
	@echo "  make run svc=         : Build and run one service locally"
	@echo "  make dev              : Run code generation and build all 3 services"
	@echo "  make clean            : Remove generated binaries and swagger docs"

quality-help:
	@echo "QUALITY ASSURANCE & TESTING COMMANDS:"
	@echo "  make fmt-check        : Verify Go source formatting without making edits"
	@echo "  make tidy-check       : Verify all go.mod/go.sum files are clean"
	@echo "  make generate-check   : Verify generated Wire and Swagger files are up-to-date"
	@echo "  make vet              : Run static analysis using 'go vet' (all modules)"
	@echo "  make migration-check  : Verify migration DDL ownership (030 contract)"
	@echo "  make test             : Run all Go unit and integration tests (all modules)"
	@echo "  make test-workspace   : Alias of test across the go.work workspace"
	@echo "  make test-race        : Run all Go tests with race detector enabled"
	@echo "  make test-integration : Run messaging integration tests (needs TEST_RABBITMQ_URL / TEST_DATABASE_URL)"
	@echo "  make lint             : Run fmt-check + go vet"
	@echo "  make check            : Run all checks sequentially (workspace CI suite)"

docker-help:
	@echo "DOCKER COMPOSE COMMANDS:"
	@echo "  make docker-build svc= : Build one service production image (context = repo root)"
	@echo "  make docker-dev-up    : Start local development stack in background"
	@echo "  make docker-dev-stop  : Stop local development stack without removing volumes"
	@echo "  make docker-dev-down  : Stop local development stack and remove volumes"
	@echo "  make docker-prod-up   : Start production Docker stack in background"
	@echo "  make docker-prod-stop : Stop production Docker stack without removing volumes"
	@echo "  make docker-prod-down : Stop production Docker stack and remove volumes"

migration-help:
	@echo "DATABASE MIGRATION COMMANDS (GOOSE, per-service — FR-035):"
	@echo "  make migrate-status svc=          : Check migration status for one service"
	@echo "  make migrate-up svc=              : Run pending migrations for one service"
	@echo "  make migrate-down svc=            : Rollback last migration step for one service"
	@echo "  make migrate-create svc= name=xxx : Create a new SQL migration in the owning service"
	@echo "  make migrate-version svc=         : View current migration version for one service"

tools-help:
	@echo "DEVELOPMENT TOOLS INSTALLATION HELPERS:"
	@echo "  make install-tools    : Install pinned Wire, Swag and Buf tool binaries"
	@echo "  make install-goose    : Install Goose migration CLI binary"
	@echo ""
	@echo "RabbitMQ topology (guards the 406 queue-type mismatch; see deployments/scripts/rabbitmq-topology/):"
	@echo "  make rabbitmq-topology-check env=  : Compare broker queue types against the registry (read-only)"
	@echo "  make rabbitmq-topology-migrate env=: Drain, delete, recreate and replay drifted queues"
	@echo "  make image-build-tool              : Build the rabbitmq-topology ops image"
	@echo "  (env= is shorthand: pass RABBITMQ_ENV_FILE=.env.production for VPS, .env.aws for AWS)"

platform-help:
	@echo "REGISTRY / INGRESS COMMANDS (020):"
	@echo "  make image-build svc= : Build one service image (tag <svc>:<git-sha>)"
	@echo "  make image-push svc=  : Push one service image to the registry"
	@echo "  make nginx-reload     : Render Nginx templates and reload (VPS)"
	@echo "  make image-build-monolith-rollback : Build rollback image FROM tag monolith-legacy (once)"

install-tools:
	@echo Installing pinned development tools...
	@go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	@go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
	@go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	@echo Development tools installed.

install-goose:
	@echo Installing Goose CLI tool...
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo Goose CLI installed.

wire:
	$(SVC_REQ)
	@echo Running Google Wire for service $(SVC)...
	@go run github.com/google/wire/cmd/wire@$(WIRE_VERSION) ./services/$(SVC)/internal/di/...
	@echo Google Wire finished for $(SVC).

wire-all:
	@echo Running Google Wire for all services...
	@go run github.com/google/wire/cmd/wire@$(WIRE_VERSION) ./services/account/internal/di/...
	@go run github.com/google/wire/cmd/wire@$(WIRE_VERSION) ./services/commerce/internal/di/...
	@go run github.com/google/wire/cmd/wire@$(WIRE_VERSION) ./services/ai/internal/di/...
	@echo Google Wire finished for all services.

swagger:
	$(SVC_REQ)
	@echo Generating Swagger for service $(SVC)...
	@cd services/$(SVC) && go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g cmd/$(SVC)/main.go --output api/swagger --parseDependency --parseInternal --quiet
	@echo Swagger generation finished for $(SVC).

swagger-all:
	@echo Generating Swagger for all services...
	@cd services/account && go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g cmd/account/main.go --output api/swagger --parseDependency --parseInternal --quiet
	@cd services/commerce && go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g cmd/commerce/main.go --output api/swagger --parseDependency --parseInternal --quiet
	@cd services/ai && go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g cmd/ai/main.go --output api/swagger --parseDependency --parseInternal --quiet
	@echo Swagger generation finished for all services.

generate: wire-all swagger-all swagger-combine nginx-routes

buf-gen:
	@echo Generating protobuf stubs into contracts/...
	@buf generate
	@echo Protobuf generation finished.

buf-lint:
	@echo Linting protobuf contracts...
	@buf lint
	@echo Protobuf lint finished.

hygiene-scan:
	@echo "Running codebase hygiene scan..."
	@go run ./shared/tools/hygiene-scan
	@echo "Hygiene scan finished."

fmt:
	@echo Formatting Go source files...
	@$(FMT_CMD)
	@echo Go source formatting finished.

fmt-check:
	@echo Checking Go source formatting...
	@$(FMT_CHECK_CMD)
	@echo Go source formatting is valid.

tidy:
	@echo "Tidying Go modules (workspace)..."
	@cd shared && go mod tidy
	@cd contracts && go mod tidy
	@cd services/account && go mod tidy
	@cd services/commerce && go mod tidy
	@cd services/ai && go mod tidy
	@go work sync
	@echo Go module tidy finished.

tidy-check:
	@echo Checking whether go.mod and go.sum files are tidy...
	@cd shared && go mod tidy -diff
	@cd contracts && go mod tidy -diff
	@cd services/account && go mod tidy -diff
	@cd services/commerce && go mod tidy -diff
	@cd services/ai && go mod tidy -diff
	@echo Go module files are tidy.

generate-check:
	@echo Checking generated Wire, Swagger and Nginx route map files...
	@$(MAKE) generate
	@git diff --exit-code -- services/account/internal/di services/commerce/internal/di services/ai/internal/di services/account/api/swagger services/commerce/api/swagger services/ai/api/swagger deployments/nginx
	@echo Generated files are up to date.

vet:
	@echo "Running go vet (all workspace modules)..."
	@cd shared && go vet ./...
	@cd contracts && go vet ./...
	@cd services/account && go vet ./...
	@cd services/commerce && go vet ./...
	@cd services/ai && go vet ./...
	@echo Go vet finished.

# 030 DDL ownership handover (FR-009): the gate reads the schema lists from
# specs/030-ddl-ownership-handover/contracts/ownership-contract.md, so it
# follows map changes without edits here. `go run` is portable, no OS branch.
migration-check:
	@echo Checking migration DDL ownership against the ownership contract...
	@go run ./shared/tools/migration-ownership-check
	@echo Migration ownership check finished.

test:
	@echo "Running Go tests (all workspace modules)..."
	@cd shared && $(TEST_CMD)
	@cd contracts && $(TEST_CMD)
	@cd services/account && $(TEST_CMD)
	@cd services/commerce && $(TEST_CMD)
	@cd services/ai && $(TEST_CMD)
	@echo Go tests finished.

test-workspace: test

test-race:
	@echo "Running Go tests with race detector (all workspace modules)..."
	@cd shared && $(RACE_TEST_CMD)
	@cd contracts && $(RACE_TEST_CMD)
	@cd services/account && $(RACE_TEST_CMD)
	@cd services/commerce && $(RACE_TEST_CMD)
	@cd services/ai && $(RACE_TEST_CMD)
	@echo Go race tests finished.

lint: fmt-check vet
	@echo Lint finished.

test-integration:
	@echo Running integration tests - requires TEST_RABBITMQ_URL / TEST_DATABASE_URL...
	@cd services/account && go test -race -run Integration ./internal/modules/subscription/...
	@cd shared && go test -race -run Integration ./infrastructure/messaging/...
	@echo Integration tests finished.

build:
	$(SVC_REQ)
	@echo Building service $(SVC)...
	@$(MKDIR_BIN_CMD)
	@go build $(GO_BUILD_FLAGS) -o "$(BIN_DIR)/$(SVC)$(GO_BIN_EXT)" ./services/$(SVC)/cmd/$(SVC)
	@echo Build succeeded. Binary: $(BIN_DIR)/$(SVC)$(GO_BIN_EXT)

build-all:
	@echo Building all 3 service binaries...
	@$(MKDIR_BIN_CMD)
	@go build $(GO_BUILD_FLAGS) -o "$(BIN_DIR)/account$(GO_BIN_EXT)" ./services/account/cmd/account
	@go build $(GO_BUILD_FLAGS) -o "$(BIN_DIR)/commerce$(GO_BIN_EXT)" ./services/commerce/cmd/commerce
	@go build $(GO_BUILD_FLAGS) -o "$(BIN_DIR)/ai$(GO_BIN_EXT)" ./services/ai/cmd/ai
	@echo Build succeeded. Binaries in $(BIN_DIR)/

run: build
	$(SVC_REQ)
	@echo Starting $(SVC) service...
	@cd services/$(SVC) && $(RUN_BIN)

dev: generate build-all

check: fmt-check tidy-check generate-check nginx-check vet migration-check test-race build-all

clean-swagger:
	@echo Cleaning generated Swagger files...
ifeq ($(OS),Windows_NT)
	@if exist services\account\api\swagger rmdir /S /Q services\account\api\swagger
	@if exist services\commerce\api\swagger rmdir /S /Q services\commerce\api\swagger
	@if exist services\ai\api\swagger rmdir /S /Q services\ai\api\swagger
else
	@rm -rf services/account/api/swagger services/commerce/api/swagger services/ai/api/swagger
endif
	@echo Swagger clean finished.

clean: clean-swagger
	@echo Cleaning generated binaries...
	@$(CLEAN_BIN_CMD)
	@echo Clean finished.

migrate-status:
	$(SVC_REQ)
	@goose -table $(GOOSE_TABLE) -dir ./services/$(SVC)/database/migrations postgres $(MIGRATION_DSN) status

migrate-up:
	$(SVC_REQ)
	@goose -table $(GOOSE_TABLE) -dir ./services/$(SVC)/database/migrations postgres $(MIGRATION_DSN) up

migrate-down:
	$(SVC_REQ)
	@goose -table $(GOOSE_TABLE) -dir ./services/$(SVC)/database/migrations postgres $(MIGRATION_DSN) down

migrate-create:
	$(SVC_REQ)
ifndef name
	$(error migrate-create requires name=xxx — e.g. make migrate-create svc=account name=add_feature)
endif
	@goose -dir ./services/$(SVC)/database/migrations create $(name) sql

migrate-version:
	$(SVC_REQ)
	@goose -table $(GOOSE_TABLE) -dir ./services/$(SVC)/database/migrations postgres $(MIGRATION_DSN) version

docker-build:
	$(SVC_REQ)
	@echo Building production Docker image for $(SVC)...
	@docker build -f services/$(SVC)/Dockerfile -t closy-$(SVC):local .
	@echo Docker image build finished.

image-build:
	$(SVC_REQ)
	@echo Building $(SVC) image...
	@docker build -f services/$(SVC)/Dockerfile -t $(AR_HOST)/$(AR_PROJECT)/$(AR_REPO)/$(SVC):$(GIT_SHA) .
	@echo Image built: $(AR_HOST)/$(AR_PROJECT)/$(AR_REPO)/$(SVC):$(GIT_SHA)

image-push:
	$(SVC_REQ)
	@echo Pushing $(SVC) image to the registry...
	@docker push $(AR_HOST)/$(AR_PROJECT)/$(AR_REPO)/$(SVC):$(GIT_SHA)
	@echo Image pushed.

image-build-monolith-rollback:
	@echo "Building rollback image FROM tag monolith-legacy (run once before funded window ends)..."
	@git worktree add /tmp/monolith-legacy monolith-legacy
	@cp deployments/monolith-legacy/Dockerfile /tmp/monolith-legacy/Dockerfile.legacy
	@docker build -f /tmp/monolith-legacy/Dockerfile.legacy -t monolith-legacy:$(GIT_SHA) /tmp/monolith-legacy
	@git worktree remove --force /tmp/monolith-legacy
	@echo "Rollback image built. Push to GHCR (T097a) with: docker tag + docker push."

nginx-reload:
	@echo "Rendering Nginx templates (NGINX_MODE=$(NGINX_MODE)) and reloading (VPS)..."
	@mkdir -p $(NGINX_RENDER_DIR)/conf.d
	@rm -f $(NGINX_RENDER_DIR)/conf.d/*.conf
	@for f in $(NGINX_TEMPLATES_DIR)/$(NGINX_MODE)/*.template; do \
		out=$$(basename $$f .template); \
		envsubst '$${API_DOMAIN} $${API_V2_DOMAIN} $${NGINX_CLIENT_MAX_BODY_SIZE}' < $$f > $(NGINX_RENDER_DIR)/conf.d/$$out; \
		echo "  rendered conf.d/$$out"; \
	done
	@nginx -t
	@nginx -s reload
	@echo "Nginx reload finished (NGINX_MODE=$(NGINX_MODE))."

nginx-check:
	@echo Checking Nginx route map covers all API paths...
	@go run ./shared/tools/nginx-check

nginx-routes: swagger-all
	@echo Generating Nginx route map from swagger...
	@go run ./shared/tools/nginx-routes
	@echo NOTE: if the local dev stack is running, run 'make nginx-reload-local' to apply the new routes.

# ---------------------------------------------------------------------------
# RabbitMQ topology reconcile — guards the one failure mode that crash-loops a
# service: the registry declares a queue type the broker already holds as a
# different one. RabbitMQ cannot change x-queue-type in place (406).
# `check` is read-only and also runs in CI; `migrate` deletes and recreates
# queues and is deliberately never automatic.
# ---------------------------------------------------------------------------
rabbitmq-topology-check:
	@echo Checking RabbitMQ queue types against the registry...
	@go run ./shared/tools/rabbitmq-topology --env-file "$(RABBITMQ_ENV_FILE)" --mode check $(if $(QUEUE),--queue $(QUEUE),) $(if $(STRICT),--strict,)
	@echo NOTE: on a box run deployments/scripts/rabbitmq-topology/01-check-drift.sh instead.

rabbitmq-topology-migrate:
	@echo Migrating drifted RabbitMQ queues (drain, delete, redeclare, replay)...
	@go run ./shared/tools/rabbitmq-topology --env-file "$(RABBITMQ_ENV_FILE)" --mode migrate --yes $(if $(QUEUE),--queue $(QUEUE),) $(if $(SPILL_DIR),--spill-dir $(SPILL_DIR),)
	@echo Restart the services so they bind to the recreated queues.

image-build-tool:
	@echo "Building rabbitmq-topology tool image (rabbitmq-topology:$(GIT_SHA))..."
	@docker build -f shared/tools/rabbitmq-topology/Dockerfile -t rabbitmq-topology:$(GIT_SHA) -t rabbitmq-topology:latest .

image-push-tool:
	@echo "Pushing rabbitmq-topology tool image..."
	@docker push rabbitmq-topology:$(GIT_SHA)

nginx-reload-local:
	@echo "Reloading local dev Nginx (docker compose)..."
	@docker exec nginx-closy-dev nginx -t
	@docker exec nginx-closy-dev nginx -s reload
	@echo Local Nginx reloaded.

swagger-combine: swagger-all
	@echo Combining Swagger specs into monolith-style bundle...
	@go run ./shared/tools/swagger-combine

docker-dev-up:
	@docker compose -f deployments/docker-compose.yml --env-file .env up -d --build

docker-dev-stop:
	@docker compose -f deployments/docker-compose.yml --env-file .env stop

docker-dev-down:
	@docker compose -f deployments/docker-compose.yml --env-file .env down -v

docker-prod-up:
	@docker compose -f deployments/docker-compose.prod.yml --env-file .env.production up -d

docker-prod-stop:
	@docker compose -f deployments/docker-compose.prod.yml --env-file .env.production stop

docker-prod-down:
	@docker compose -f deployments/docker-compose.prod.yml --env-file .env.production down -v
