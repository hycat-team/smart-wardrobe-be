# Quickstart & Verification Guide: 018-contract-dto-refactor

**Feature**: `018-contract-dto-refactor`
**Date**: 2026-09-03
**Status**: Draft

---

## 1. Mục Đích & Tiền Đề Kiểm Thử (Prerequisites)

Tài liệu này hướng dẫn các kịch bản kiểm thử tĩnh, kiểm thử biên dịch và kiểm thử tích hợp nhằm xác thực rằng việc refactor DTO của Contract không làm phát sinh lỗi biên dịch và không làm vỡ các API Response public của hệ thống.

**Tiền đề kỹ thuật**:
- Go version 1.23+
- Docker & Docker Compose (cho PostgreSQL, Redis nếu chạy test integration)
- Makefile và các công cụ linting (`golangci-lint`, `wire`)

---

## 2. Kịch Bản Kiểm Tra Biên Dịch & Type Integrity (Compiler Gate)

Sau khi áp dụng Strict Cutover trên các interface contract, chạy các lệnh kiểm tra tự động:

```bash
# 1. Tái sinh Dependency Injection
make wire

# 2. Kiểm tra biên dịch toàn bộ workspace
go build ./...

# 3. Chạy linter kiểm tra cú pháp và quy chuẩn
make lint
```

**Kết quả mong đợi**:
- `go build ./...` hoàn tất không có bất kỳ lỗi biên dịch nào.
- Toàn bộ các caller của `IWardrobeContract` và `IBrandContract` đã được chuyển sang nhận kiểu `ContractDTO`.

---

## 3. Kịch Bản Kiểm Thử Tự Động (Automated Test Suites)

Chạy bộ unit test và race detection cho các module bị tác động:

```bash
# Chạy test cho module Wardrobe
go test -v -race ./internal/modules/wardrobe/...

# Chạy test cho module Brand
go test -v -race ./internal/modules/brand/...

# Chạy test cho module Fashion (nơi gọi nhiều contract nhất)
go test -v -race ./internal/modules/fashion/...

# Chạy test cho module Dashboard
go test -v -race ./internal/modules/dashboard/...

# Chạy toàn bộ test suite dự án
make test-race
```

**Kết quả mong đợi**:
- 100% các bài test (bao gồm mock contract) đều PASS.

---

## 4. Kịch Bản Xác Thực Không Gây Lỗi Hồi Quy API (Zero API Regression)

Xác thực rằng HTTP API response của các endpoint sau vẫn giữ nguyên định dạng JSON có đầy đủ `fashionItem` và `brandItem` cho Client:

### Scenario 1: Lấy danh sách tủ đồ người dùng
- **Endpoint**: `GET /api/v1/wardrobe/items`
- **Header**: `Authorization: Bearer <user_token>`
- **Kỳ vọng**:
  - HTTP Status: `200 OK`
  - Body JSON chứa mảng `items`, mỗi item có:
    ```json
    {
      "id": "...",
      "price": 100000,
      "status": "active",
      "fashionItem": {
        "id": "...",
        "imageUrl": "https://...",
        "category": { ... }
      },
      "brandItem": { ... }
    }
    ```
  - Trường `fashionItem` và `brandItem` không bị rỗng/thiếu so với trước khi refactor.

### Scenario 2: Kiểm tra dọn dẹp tủ đồ mồ côi (Worker Safety)
- **Kỳ vọng**:
  - Khi có trang phục tải lên bị lỗi (`FashionItemID == nil`), worker `CleanupFailedItems` dọn dẹp tủ đồ bình thường, log ghi nhận xử lý an toàn mà không phát sinh nil pointer dereference panic.
