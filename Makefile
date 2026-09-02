# Config and variables
export

WIRE_PATH := ./internal/di
SWAG_MAIN_FILE := ./cmd/server/main.go
SWAG_OUTPUT_DIR := ./api/swagger
GO_BUILD_PACKAGE := ./cmd/server
BIN_DIR := bin

WIRE_VERSION := v0.7.0
SWAG_VERSION := v1.16.6

# Goose Migration variables
DB_HOST ?= localhost
DB_USER ?= admin
DB_PASSWORD ?= 123456
DB_NAME ?= smart_wardrobe_db
DB_PORT ?= 5432
DB_SSLMODE ?= disable
DB_TIMEZONE ?= Asia/Ho_Chi_Minh
MIGRATION_DIR := ./database/migrations
MIGRATION_DSN := "host=$(DB_HOST) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) port=$(DB_PORT) sslmode=$(DB_SSLMODE) TimeZone=$(DB_TIMEZONE)"

ifeq ($(OS),Windows_NT)
GO_BIN_OUT := $(BIN_DIR)/main.exe
RUN_BIN := .\$(subst /,\,$(GO_BIN_OUT))
MKDIR_BIN_CMD := if not exist "$(subst /,\,$(BIN_DIR))" mkdir "$(subst /,\,$(BIN_DIR))"
CLEAN_BIN_CMD := if exist "$(subst /,\,$(BIN_DIR))" rmdir /S /Q "$(subst /,\,$(BIN_DIR))"
CLEAN_SWAGGER_CMD := if exist api\swagger rmdir /S /Q api\swagger
FMT_CHECK_CMD := powershell -Command "$$files = gofmt -l .; if ($$files) { Write-Error ($$files -join [Environment]::NewLine); exit 1 }"
TEST_CMD := go test ./... 2>&1 | findstr /V /C:"no test files"
RACE_TEST_CMD := go test -race ./... 2>&1 | findstr /V /C:"no test files"
else
GO_BIN_OUT := $(BIN_DIR)/main
RUN_BIN := ./$(GO_BIN_OUT)
MKDIR_BIN_CMD := mkdir -p "$(BIN_DIR)"
CLEAN_BIN_CMD := rm -rf "$(BIN_DIR)"
CLEAN_SWAGGER_CMD := rm -rf ./api/swagger
FMT_CHECK_CMD := test -z "$$(gofmt -l .)"
TEST_CMD := go test ./... 2>&1 | grep -v "no test files"
RACE_TEST_CMD := go test -race ./... 2>&1 | grep -v "no test files"
endif

# FMT command dùng chung 2 OS (gofmt -w . tự động đệ quy tất cả các file .go)
FMT_CMD := gofmt -w .

.PHONY: help dev-help quality-help docker-help migration-help tools-help install-tools wire swagger generate hygiene-scan fmt fmt-check tidy tidy-check vet test test-race test-integration lint build run dev clean clean-swagger check generate-check docker-build migration-status migration-up migration-down migration-create migration-version install-goose docker-dev-up docker-dev-stop docker-dev-down docker-prod-up docker-prod-stop docker-prod-down

help:
	@echo "========================================================================"
	@echo "SMART WARDROBE BUILD SYSTEM - HELP INDEX"
	@echo "========================================================================"
	@echo "Run one of the following commands to list specific targets:"
	@echo "  make dev-help       : Local development workflow targets"
	@echo "  make quality-help   : CI quality checks, tests, and compilation"
	@echo "  make docker-help    : Docker Compose development & production targets"
	@echo "  make migration-help : Database migration (Goose) commands"
	@echo "  make tools-help     : Local development tools installation helper"
	@echo "========================================================================"

dev-help:
	@echo "DEVELOPMENT WORKFLOW COMMANDS:"
	@echo "  make wire             : Generate dependency injection code (Google Wire)"
	@echo "  make swagger          : Generate Swagger API specifications"
	@echo "  make generate         : Run both Wire and Swagger code generation"
	@echo "  make hygiene-scan     : Run codebase hygiene scan (feature 017, report in specs/)"
	@echo "  make fmt              : Format all Go source files"
	@echo "  make tidy             : Tidy Go modules"
	@echo "  make run              : Build and run the Go backend locally"
	@echo "  make dev              : Run code generation and start backend locally"
	@echo "  make clean            : Remove generated binaries and swagger docs"

quality-help:
	@echo "QUALITY ASSURANCE & TESTING COMMANDS:"
	@echo "  make fmt-check        : Verify Go source formatting without making edits"
	@echo "  make tidy-check       : Verify go.mod and go.sum files are clean"
	@echo "  make generate-check   : Verify generated Wire and Swagger files are up-to-date"
	@echo "  make vet              : Run static analysis using 'go vet'"
	@echo "  make test             : Run all Go unit and integration tests"
	@echo "  make test-race        : Run all Go tests with race detector enabled"
	@echo "  make test-integration : Run messaging integration tests (needs TEST_RABBITMQ_URL / TEST_DATABASE_URL)"
	@echo "  make lint             : Run fmt-check + go vet"
	@echo "  make build            : Compile Go backend binary"
	@echo "  make check            : Run all checks sequentially (CI check suite)"

docker-help:
	@echo "DOCKER COMPOSE COMMANDS:"
	@echo "  make docker-build     : Build local production Docker image"
	@echo "  make docker-dev-up    : Start local development stack in background"
	@echo "  make docker-dev-stop  : Stop local development stack without removing volumes"
	@echo "  make docker-dev-down  : Stop local development stack and remove volumes"
	@echo "  make docker-prod-up   : Start production Docker stack in background"
	@echo "  make docker-prod-stop : Stop production Docker stack without removing volumes"
	@echo "  make docker-prod-down : Stop production Docker stack and remove volumes"

migration-help:
	@echo "DATABASE MIGRATION COMMANDS (GOOSE):"
	@echo "  make migration-status : Check database migration status"
	@echo "  make migration-up     : Run pending database migrations"
	@echo "  make migration-down   : Rollback last database migration step"
	@echo "  make migration-create name=xxx : Create a new SQL migration file"
	@echo "  make migration-version: View current database migration version"

tools-help:
	@echo "DEVELOPMENT TOOLS INSTALLATION HELPERS:"
	@echo "  make install-tools    : Install pinned Wire and Swag tool binaries"
	@echo "  make install-goose    : Install Goose migration CLI binary"

install-tools:
	@echo Installing pinned development tools...
	@go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	@go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
	@echo Development tools installed.

install-goose:
	@echo Installing Goose CLI tool...
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo Goose CLI installed.

wire:
	@echo Running Google Wire...
	@go run github.com/google/wire/cmd/wire@$(WIRE_VERSION) $(WIRE_PATH)/...
	@echo Google Wire finished.

swagger:
	@echo Generating Swagger specifications...
	@go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g $(SWAG_MAIN_FILE) --output $(SWAG_OUTPUT_DIR) --parseDependency --parseInternal --exclude internal/modules/community --quiet
	@go run ./internal/tools/swagger-split -master $(SWAG_OUTPUT_DIR)/swagger.json -mapping $(SWAG_OUTPUT_DIR)/swagger-module-mapping.json -out $(SWAG_OUTPUT_DIR)/modules
	@echo Swagger generation finished.

generate: wire swagger

hygiene-scan:
	@echo "Running codebase hygiene scan (specs/017-codebase-hygiene-refactor)..."
	@go run ./internal/tools/hygiene-scan
	@echo "Hygiene scan finished. Report: specs/017-codebase-hygiene-refactor/reports/scan-report.md"

fmt:
	@echo Formatting Go source files...
	@$(FMT_CMD)
	@echo Go source formatting finished.

fmt-check:
	@echo Checking Go source formatting...
	@$(FMT_CHECK_CMD)
	@echo Go source formatting is valid.

tidy:
	@echo Tidying Go modules...
	@go mod tidy
	@echo Go module tidy finished.

tidy-check:
	@echo Checking whether go.mod and go.sum are tidy...
	@go mod tidy -diff
	@echo Go module files are tidy.

generate-check:
	@echo Checking generated Wire and Swagger files...
	@$(MAKE) generate
	@git diff --exit-code -- $(WIRE_PATH) $(SWAG_OUTPUT_DIR)
	@echo Generated files are up to date.

vet:
	@echo Running go vet...
	@go vet ./...
	@echo Go vet finished.

test:
	@echo Running Go tests...
	@$(TEST_CMD)
	@echo Go tests finished.

test-race:
	@echo Running Go tests with race detector...
	@$(RACE_TEST_CMD)
	@echo Go race tests finished.

lint: fmt-check vet
	@echo Lint finished.

test-integration:
	@echo Running integration tests - requires TEST_RABBITMQ_URL / TEST_DATABASE_URL...
	@go test -race -run Integration ./internal/shared/infrastructure/messaging/
	@echo Integration tests finished.

build:
	@echo Building application for the current operating system...
	@$(MKDIR_BIN_CMD)
	@go build -trimpath -ldflags="-s -w" -o "$(GO_BIN_OUT)" $(GO_BUILD_PACKAGE)
	@echo Build succeeded. Binary: $(GO_BIN_OUT)

run: build
	@echo Starting backend...
	@$(RUN_BIN)

dev: generate run

check: fmt-check tidy-check generate-check vet test build

clean-swagger:
	@echo Cleaning generated Swagger files...
	@$(CLEAN_SWAGGER_CMD)
	@echo Swagger clean finished.

clean: clean-swagger
	@echo Cleaning generated binaries...
	@$(CLEAN_BIN_CMD)
	@echo Clean finished.

migration-status:
	@goose -dir $(MIGRATION_DIR) postgres $(MIGRATION_DSN) status

migration-up:
	@goose -dir $(MIGRATION_DIR) postgres $(MIGRATION_DSN) up

migration-down:
	@goose -dir $(MIGRATION_DIR) postgres $(MIGRATION_DSN) down

migration-create:
	@goose -dir $(MIGRATION_DIR) create $(name) sql

migration-version:
	@goose -dir $(MIGRATION_DIR) postgres $(MIGRATION_DSN) version

docker-build:
	@echo Building production Docker image...
	@docker build --pull --target production -t backend-smartwardrobe:local .
	@echo Docker image build finished.

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