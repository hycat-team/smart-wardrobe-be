# Quickstart & Verification Guide - Refactor Category Nesting

This guide provides scenarios to verify that the refactoring compiles, runs, and outputs the correct nested Category structures instead of flat fields.

## Prerequisites
- Docker Compose running with Elasticsearch, PostgreSQL, and Redis.
- Google Wire CLI or Makefile setup.

## Verification Steps

### Step 1: Code Compilation
Execute the build command:
```bash
make build
```

### Step 2: Running Unit Tests
Validate that AI recommendation scoring, ranking, and synthesis logic still work:
```bash
make test
```

### Step 3: Run the Application & Verify HTTP Endpoint
Start the server:
```bash
make dev
```

Send a request to:
```bash
GET http://localhost:8080/api/v1/system-catalog/wardrobe-items
```

Verify that in the JSON response:
- The top-level `category` field is completely removed from the `items` list objects.
- Each `items[i].fashionItem` contains a nested `category` field with the following shape:
  ```json
  "category": {
    "id": "uuid-value",
    "name": "Áo Thun",
    "slug": "ao-thun"
  }
  ```
