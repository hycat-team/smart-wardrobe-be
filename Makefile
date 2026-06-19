# Config and variables
export

WIRE_PATH := ./internal/di
SWAG_MAIN_FILE := ./cmd/server/main.go
SWAG_OUTPUT_DIR := ./docs
GO_BUILD_PACKAGE := ./cmd/server
BIN_DIR := bin

WIRE_VERSION := v0.7.0
SWAG_VERSION := v1.16.6

ifeq ($(OS),Windows_NT)
GO_BIN_OUT := $(BIN_DIR)/main.exe
RUN_BIN := .\$(subst /,\,$(GO_BIN_OUT))
MKDIR_BIN_CMD := if not exist "$(subst /,\,$(BIN_DIR))" mkdir "$(subst /,\,$(BIN_DIR))"
CLEAN_BIN_CMD := if exist "$(subst /,\,$(BIN_DIR))" rmdir /S /Q "$(subst /,\,$(BIN_DIR))"
FMT_CMD := powershell -NoProfile -ExecutionPolicy Bypass -Command "$$files = @(Get-ChildItem -Path . -Filter '*.go' -File -Recurse | Where-Object { $$_.FullName -notmatch '[\\/](vendor)[\\/]' }); foreach ($$file in $$files) { & gofmt -w $$file.FullName; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE } }"
FMT_CHECK_CMD := powershell -NoProfile -ExecutionPolicy Bypass -Command "$$files = @(Get-ChildItem -Path . -Filter '*.go' -File -Recurse | Where-Object { $$_.FullName -notmatch '[\\/](vendor)[\\/]' }); $$bad = @(); foreach ($$file in $$files) { $$result = & gofmt -l $$file.FullName; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE }; if ($$result) { $$bad += $$result } }; if ($$bad.Count -gt 0) { Write-Host 'The following Go files are not formatted:'; $$bad | ForEach-Object { Write-Host $$_ }; exit 1 }"
else
GO_BIN_OUT := $(BIN_DIR)/main
RUN_BIN := ./$(GO_BIN_OUT)
MKDIR_BIN_CMD := mkdir -p "$(BIN_DIR)"
CLEAN_BIN_CMD := rm -rf "$(BIN_DIR)"
FMT_CMD := find . -type f -name '*.go' -not -path './vendor/*' -exec gofmt -w {} +
FMT_CHECK_CMD := test -z "$$(find . -type f -name '*.go' -not -path './vendor/*' -exec gofmt -l {} +)"
endif

.PHONY: help install-tools wire swagger generate fmt fmt-check tidy tidy-check vet test build run dev clean check generate-check docker-build

help:
	@echo PROJECT COMMANDS:
	@echo make wire          : Generate dependency injection code
	@echo make swagger       : Generate Swagger specifications
	@echo make generate      : Generate Wire and Swagger files
	@echo make fmt           : Format Go source files
	@echo make fmt-check     : Check Go source formatting
	@echo make tidy          : Tidy Go modules
	@echo make tidy-check    : Verify go.mod and go.sum
	@echo make vet           : Run go vet
	@echo make test          : Run Go tests
	@echo make build         : Build the Go executable
	@echo make run           : Build and run the executable
	@echo make dev           : Generate and run the application
	@echo make check         : Run all CI quality checks
	@echo make clean         : Remove generated binaries
	@echo make install-tools : Install pinned Wire and Swag versions
	@echo make docker-build  : Build the production Docker image locally

install-tools:
	@echo Installing pinned development tools...
	@go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	@go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
	@echo Development tools installed.

wire:
	@echo Running Google Wire...
	@go run github.com/google/wire/cmd/wire@$(WIRE_VERSION) $(WIRE_PATH)/...
	@echo Google Wire finished.

swagger:
	@echo Generating Swagger specifications...
	@go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g $(SWAG_MAIN_FILE) --output $(SWAG_OUTPUT_DIR) --parseDependency --parseInternal
	@echo Swagger generation finished.

generate: wire swagger

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
	@go test ./...
	@echo Go tests finished.

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

clean:
	@echo Cleaning generated binaries...
	@$(CLEAN_BIN_CMD)
	@echo Clean finished.

docker-build:
	@echo Building production Docker image...
	@docker build --pull --target production -t backend-smartwardrobe:local .
	@echo Docker image build finished.