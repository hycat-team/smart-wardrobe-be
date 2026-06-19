# 👕 SmartWardrobe Backend (Golang Modular Monolith)

SmartWardrobe Backend is an API service system developed using **Golang** based on a clean, high-performance, and maintainable **Modular Monolith** architecture (Clean Architecture). The system provides features for smart wardrobe management, AI-driven outfit recommendations, subscription plan management, and secure identity services (Identity).

---

## 🛠️ Prerequisites

Before getting started, make sure your computer has the following tools installed:

1. **Go Compiler (v1.24.+)**
    - Download and install at: [golang.org/dl](https://golang.org/dl/)
    - Verify installation: `go version`
2. **Docker & Docker Desktop (or Docker Compose)**
    - Required to run PostgreSQL (with pgvector) and Redis in the local environment.
    - Download and install at: [docker.com](https://www.docker.com/)
3. **GNU Make**
    - _Windows:_ Recommended to install via [Chocolatey](https://chocolatey.org/) (`choco install make`) or use Git Bash / WSL.
    - _macOS:_ Pre-installed or install via Homebrew (`brew install make`).
    - _Linux:_ `sudo apt install make` (Ubuntu/Debian).

---

## 🚀 Local Run Guide

Follow these steps to set up and run the project on your machine:

### Step 1: Clone Project & Configure Environment

1. Navigate to the project directory:
    ```bash
    cd smart-wardrobe-be
    ```
2. Create the `.env` file from the example:
    ```bash
    cp .env.example .env
    ```
    _Note:_ Open the newly created `.env` file and configure the database connection, Redis, JWT secret, or Gmail SMTP for sending real OTP codes (if needed).

### Step 2: Run Project Using Docker Compose

You have two options for running the application locally using Docker Compose:

#### Option A: Run Only Database & Redis (Recommended for direct Go development)

```bash
docker-compose up postgres redis -d
```

_Note:_ After that, run the Go application locally using the `make dev` or `go run` command.

#### Option B: Run the Full Stack Including Backend App

```bash
docker-compose up -d --build
```

This command will automatically build a lightweight Docker image for the Go Backend (using multi-stage builds, about ~15MB) and run it with full database and cache integration.

_Check status:_ Run the `docker ps` command to ensure the 3 containers (`postgres-smartwardrobe`, `redis-smartwardrobe`, `backend-smartwardrobe`) are running properly.

### Step 3: Install Specialized Go Development Tools

The project uses **Google Wire** (Dependency Injection) and **Swag** (Swagger Generator). Install them using the pre-configured Make command:

```bash
make install-tools
```

_(Or run the following commands manually if Make is not available)_:

```bash
go install github.com/google/wire/cmd/wire@latest
go install github.com/swaggo/swag/cmd/swag@latest
```

---

## 💻 Development & Running Workflow

The project includes automated commands in the `Makefile` to simplify development workflows.

### Fastest Development Command (Full Flow)

When starting development or after modifying anything related to Dependency Injection (declaring new Providers/Usecases) or updating Swagger APIs:

```bash
make dev
```

This command automatically executes:

1. `go mod tidy` to clean up and download dependencies.
2. `wire` to generate Dependency Injection code automatically.
3. `swag` to update Swagger API documentation.
4. Compiles and starts the server immediately.

---

### Available Make Commands Detail

| Command              | Meaning                                                  | Equivalent Manual Command                                                         |
| :------------------- | :------------------------------------------------------- | :-------------------------------------------------------------------------------- |
| `make install-tools` | Installs `wire` and `swag` CLIs locally                  | `go install ...`                                                                  |
| `make tidy`          | Syncs and downloads missing Go packages                  | `go mod tidy`                                                                     |
| `make wire`          | Automatically generates Dependency Injection code        | `wire ./internal/di/...`                                                          |
| `make swagger`       | Regenerates Swagger API documentation                    | `swag init -g cmd/server/main.go --output docs --parseDependency --parseInternal` |
| `make build`         | Compiles the project into an executable in `bin/`        | `go build -o bin/main.exe cmd/server/main.go`                                     |
| `make run`           | Runs the compiled executable                             | `./bin/main.exe`                                                                  |
| `make dev`           | Runs full flow: tidy, wire, swagger, and run             | _(Combination of the commands above)_                                             |
| `make clean`         | Cleans up the `bin/` binary directory                    | `rm -rf bin/`                                                                     |

---

## 🔍 Verify & Test the APIs

Upon successful server startup, the console will print a clean log containing the API documentation links:

```text
==========================================================
SmartWardrobe BE is running on port: 8080
Swagger UI is available at: http://localhost:8080/swagger
==========================================================
```

- **API Documentation (Swagger UI):** Hold `Ctrl` and click the link [http://localhost:8080/swagger](http://localhost:8080/swagger) to open the interactive API docs directly in your browser.
