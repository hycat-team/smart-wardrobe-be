# Config & Variables
export

WIRE_PATH = ./internal/di
GO_MAIN_REL_PATH = cmd/server/main.go
GO_BIN_OUT = bin/main.exe

.PHONY: install-tools tidy wire swagger build run dev clean help

help:
	@echo "PROJECT COMMANDS:"
	@echo "  make wire             : Generate Dependency Injection code (Google Wire)"
	@echo "  make swagger          : Generate Swagger specifications using Swag"
	@echo "  make tidy             : Format module and download missing dependencies"
	@echo "  make build            : Build the Go executable"
	@echo "  make run              : Build and Run the executable locally"
	@echo "  make dev              : Run full flow (tidy -> wire -> swagger -> run)"
	@echo "  make clean            : Clean binary files"
	@echo "  make install-tools    : Install necessary tools (wire, swag)"

wire:
	@echo "Running Google Wire..."
	@wire $(WIRE_PATH)/...
	@echo "Google Wire finished!"

swagger:
	@echo "Generating Swagger specifications..."
	@swag init -g $(GO_MAIN_REL_PATH) --output docs --parseDependency --parseInternal
	@echo "Swagger generation finished!"

tidy:
	@echo "Tidying module..."
	@go mod tidy
	@echo "Go mod tidy finished!"

build:
	@echo "Building application..."
	@go build -o $(GO_BIN_OUT) $(GO_MAIN_REL_PATH)
	@echo "Build success! Binary is at: $(GO_BIN_OUT)"

run: build
	@echo "Starting backend..."
	@./$(GO_BIN_OUT)

dev: tidy wire swagger run

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@echo "Clean done!"

install-tools:
	@echo "Installing tools..."
	@go install github.com/google/wire/cmd/wire@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "All tools installed."
