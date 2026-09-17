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
# - Cloud Run images → Artifact Registry (asia-southeast1), tag <svc>:<git-sha> (immutable).
# - VPS images (rollback + repatriation) → GHCR with existing CI token.
AR_HOST ?= asia-southeast1-docker.pkg.dev
AR_PROJECT ?= smart-wardrobe
AR_REPO ?= smart-wardrobe
GHCR_OWNER ?= hycat
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

# Migration (Goose) connection variables (shared by all services)
DB_HOST ?= localhost
DB_USER ?= admin
DB_PASSWORD ?= 123456
DB_NAME ?= smart_wardrobe_db
DB_PORT ?= 5432
DB_SSLMODE ?= disable
DB_TIMEZONE ?= Asia/Ho_Chi_Minh
MIGRATION_DSN := "host=$(DB_HOST) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) port=$(DB_PORT) sslmode=$(DB_SSLMODE) TimeZone=$(DB_TIMEZONE)"

# Cloud Run deploy defaults (FR-044..FR-048; manual helper — CI uses deploy-platform.yml)
CR_REGION ?= asia-southeast1
CR_CONCURRENCY ?= 40
CR_HTTP_MAX_INSTANCES ?= 2

# Nginx ingress render (FR-028; runs on VPS)
NGINX_TEMPLATES_DIR := deployments/nginx/templates
NGINX_ROUTES_DIR := deployments/nginx/api-v2-routes
NGINX_RENDER_DIR ?= /etc/nginx

ifeq ($(OS),Windows_NT)
GO_BIN_EXT := .exe
MKDIR_BIN_CMD := if not exist "$(subst /,\,$(BIN_DIR))" mkdir "$(subst /,\,$(BIN_DIR))"
CLEAN_BIN_CMD := if exist "$(subst /,\,$(BIN_DIR))" rmdir /S /Q "$(subst /,\,$(BIN_DIR))"
FMT_CHECK_CMD := powershell -Command "$$files = gofmt -l .; if ($$files) { Write-Error ($$files -join [Environment]::NewLine); exit 1 }"
TEST_CMD := go test ./... 2>&1 | findstr /V /C:"no test files"
RACE_TEST_CMD := go test -race ./... 2>&1 | findstr /V /C:"no test files"
else
GO_BIN_EXT :=
MKDIR_BIN_CMD := mkdir -p "$(BIN_DIR)"
CLEAN_BIN_CMD := rm -rf "$(BIN_DIR)"
FMT_CHECK_CMD := test -z "$$(gofmt -l .)"
TEST_CMD := go test ./... 2>&1 | grep -v "no test files"
RACE_TEST_CMD := go test -race ./... 2>&1 | grep -v "no test files"
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

.PHONY: help dev-help quality-help docker-help migration-help tools-help platform-help install-tools wire wire-all swagger swagger-all generate buf-gen buf-lint hygiene-scan fmt fmt-check tidy tidy-check vet test test-workspace test-race test-integration lint build build-all run dev check generate-check clean-swagger clean migrate-status migrate-up migrate-down migrate-create migrate-version docker-build image-build image-push deploy token-helper-build image-build-monolith-rollback nginx-reload docker-dev-up docker-dev-stop docker-dev-down docker-prod-up docker-prod-stop docker-prod-down install-goose

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
	@echo "  make platform-help  : Cloud Run / registry / ingress targets (020)"
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

platform-help:
	@echo "CLOUD RUN / REGISTRY / INGRESS COMMANDS (020):"
	@echo "  make image-build svc= : Build one service image (tag <svc>:<git-sha>)"
	@echo "  make image-push svc=  : Push one service image to Artifact Registry"
	@echo "  make deploy svc=      : Deploy one service to Cloud Run (HTTP, manual helper)"
	@echo "  make token-helper-build : Build token-helper binary + image (VPS)"
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
	@cd ../commerce && go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g cmd/commerce/main.go --output api/swagger --parseDependency --parseInternal --quiet
	@cd ../ai && go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g cmd/ai/main.go --output api/swagger --parseDependency --parseInternal --quiet
	@echo Swagger generation finished for all services.

generate: wire-all swagger-all

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
	@echo Tidying Go modules (workspace)...
	@cd shared && go mod tidy
	@cd contracts && go mod tidy
	@cd services/account && go mod tidy
	@cd services/commerce && go mod tidy
	@cd services/ai && go mod tidy
	@cd deployments/token-helper && go mod tidy
	@go work sync
	@echo Go module tidy finished.

tidy-check:
	@echo Checking whether go.mod and go.sum files are tidy...
	@cd shared && go mod tidy -diff
	@cd contracts && go mod tidy -diff
	@cd services/account && go mod tidy -diff
	@cd services/commerce && go mod tidy -diff
	@cd services/ai && go mod tidy -diff
	@cd deployments/token-helper && go mod tidy -diff
	@echo Go module files are tidy.

generate-check:
	@echo Checking generated Wire and Swagger files...
	@$(MAKE) generate
	@git diff --exit-code -- services/account/internal/di services/commerce/internal/di services/ai/internal/di services/account/api/swagger services/commerce/api/swagger services/ai/api/swagger
	@echo Generated files are up to date.

vet:
	@echo Running go vet (all workspace modules)...
	@cd shared && go vet ./...
	@cd contracts && go vet ./...
	@cd services/account && go vet ./...
	@cd services/commerce && go vet ./...
	@cd services/ai && go vet ./...
	@cd deployments/token-helper && go vet ./...
	@echo Go vet finished.

test:
	@echo Running Go tests (all workspace modules)...
	@cd shared && $(TEST_CMD)
	@cd contracts && $(TEST_CMD)
	@cd services/account && $(TEST_CMD)
	@cd services/commerce && $(TEST_CMD)
	@cd services/ai && $(TEST_CMD)
	@cd deployments/token-helper && $(TEST_CMD)
	@echo Go tests finished.

test-workspace: test

test-race:
	@echo Running Go tests with race detector (all workspace modules)...
	@cd shared && $(RACE_TEST_CMD)
	@cd contracts && $(RACE_TEST_CMD)
	@cd services/account && $(RACE_TEST_CMD)
	@cd services/commerce && $(RACE_TEST_CMD)
	@cd services/ai && $(RACE_TEST_CMD)
	@cd deployments/token-helper && $(RACE_TEST_CMD)
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
	@cd services/$(SVC) && ../../$(BIN_DIR)/$(SVC)$(GO_BIN_EXT)

dev: generate build-all

check: fmt-check tidy-check generate-check vet test-race build-all

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
	@goose -dir ./services/$(SVC)/database/migrations postgres $(MIGRATION_DSN) status

migrate-up:
	$(SVC_REQ)
	@goose -dir ./services/$(SVC)/database/migrations postgres $(MIGRATION_DSN) up

migrate-down:
	$(SVC_REQ)
	@goose -dir ./services/$(SVC)/database/migrations postgres $(MIGRATION_DSN) down

migrate-create:
	$(SVC_REQ)
ifndef name
	$(error migrate-create requires name=xxx — e.g. make migrate-create svc=account name=add_feature)
endif
	@goose -dir ./services/$(SVC)/database/migrations create $(name) sql

migrate-version:
	$(SVC_REQ)
	@goose -dir ./services/$(SVC)/database/migrations postgres $(MIGRATION_DSN) version

docker-build:
	$(SVC_REQ)
	@echo Building production Docker image for $(SVC)...
	@docker build -f services/$(SVC)/Dockerfile -t smart-wardrobe-$(SVC):local .
	@echo Docker image build finished.

image-build:
	$(SVC_REQ)
	@echo Building $(SVC) image...
	@docker build -f services/$(SVC)/Dockerfile -t $(AR_HOST)/$(AR_PROJECT)/$(AR_REPO)/$(SVC):$(GIT_SHA) .
	@echo Image built: $(AR_HOST)/$(AR_PROJECT)/$(AR_REPO)/$(SVC):$(GIT_SHA)

image-push:
	$(SVC_REQ)
	@echo Pushing $(SVC) image to Artifact Registry...
	@docker push $(AR_HOST)/$(AR_PROJECT)/$(AR_REPO)/$(SVC):$(GIT_SHA)
	@echo Image pushed.

deploy:
	$(SVC_REQ)
	@echo Deploying $(SVC) to Cloud Run (manual helper; CI uses deploy-platform.yml)...
	@gcloud run deploy smart-wardrobe-$(SVC) --image $(AR_HOST)/$(AR_PROJECT)/$(AR_REPO)/$(SVC):$(GIT_SHA) --region $(CR_REGION) --platform managed --concurrency $(CR_CONCURRENCY) --max-instances $(CR_HTTP_MAX_INSTANCES)
	@echo Deploy finished.

token-helper-build:
	@echo Building token-helper binary and image (VPS)...
	@$(MKDIR_BIN_CMD)
	@go build $(GO_BUILD_FLAGS) -o "$(BIN_DIR)/token-helper$(GO_BIN_EXT)" ./deployments/token-helper
	@docker build -f deployments/token-helper/Dockerfile -t smart-wardrobe-token-helper:local .
	@echo token-helper build finished.

image-build-monolith-rollback:
	@echo Building rollback image FROM tag monolith-legacy (run once before funded window ends)...
	@git worktree add /tmp/monolith-legacy monolith-legacy
	@cp deployments/monolith-legacy/Dockerfile /tmp/monolith-legacy/Dockerfile.legacy
	@docker build -f /tmp/monolith-legacy/Dockerfile.legacy -t monolith-legacy:$(GIT_SHA) /tmp/monolith-legacy
	@git worktree remove --force /tmp/monolith-legacy
	@echo Rollback image built. Push to GHCR (T097a) with: docker tag + docker push.

nginx-reload:
	@echo Rendering Nginx templates and reloading (VPS)...
	@mkdir -p $(NGINX_RENDER_DIR)/api-v2-routes
	@envsubst '$${API_DOMAIN} $${NGINX_CLIENT_MAX_BODY_SIZE}' < $(NGINX_TEMPLATES_DIR)/default.conf.template > $(NGINX_RENDER_DIR)/default.conf
	@envsubst '$${API_V2_DOMAIN} $${NGINX_CLIENT_MAX_BODY_SIZE} $${ACCOUNT_RUN_URL} $${COMMERCE_RUN_URL} $${AI_RUN_URL} $${TOKEN_HELPER_URL}' < $(NGINX_TEMPLATES_DIR)/api-v2.conf.template > $(NGINX_RENDER_DIR)/api-v2.conf
	@envsubst '$${ACCOUNT_RUN_URL}' < deployments/nginx/api-v2-routes/account.conf > $(NGINX_RENDER_DIR)/api-v2-routes/account.conf
	@envsubst '$${COMMERCE_RUN_URL}' < deployments/nginx/api-v2-routes/commerce.conf > $(NGINX_RENDER_DIR)/api-v2-routes/commerce.conf
	@envsubst '$${AI_RUN_URL}' < deployments/nginx/api-v2-routes/ai.conf > $(NGINX_RENDER_DIR)/api-v2-routes/ai.conf
	@nginx -t
	@nginx -s reload
	@echo Nginx reload finished.

docker-dev-up:
	@docker compose -f deployments/docker-compose.yml --env-file .env up -d

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
