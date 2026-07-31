# Contract: Route-Registry Snapshot Test (FR-017)

**Branch**: `010-module-ownership-refactor` | **Date**: 2026-08-01

## Mục đích

Kiểm chứng 100% endpoint `/api/v1/*` giữ nguyên path + method sau Đợt 0 và Đợt 1 (SC-001). Bảo vệ refactor khỏi breaking change do mất route hoặc đổi method khi di chuyển route về module.

## Cách hoạt động

Dùng `engine.Routes()` của Gin → `[]gin.RoutesInfo` (mỗi item có `Method`, `Path`). Test dựng engine với handler nil-dep (không cần DB), so sánh tập `(method, path)` với golden file.

## Vị trí file

- Test: `internal/api/routes/routes_test.go`
- Golden: `internal/api/routes/testdata/routes_baseline.txt`

## Format golden file

Mỗi dòng một route, sắp xếp theo `method` rồi `path`:

```
GET /api/v1/health
GET /api/v1/auth/...
POST /api/v1/wardrobe-items/:id/clone
...
```

## Cách dựng engine trong test

```go
cfg := testConfig() // config tối thiểu (JWT secret, timeout, rate-limit 0-ish)
appRouter := &AppRouter{
    IdentityRouter:     &identity_router.Router{},  // hoặc NewRouter với nil handler
    SubscriptionRouter: &subscription_router.Router{},
    // ... 6 module
}
rateLimit := shared_middleware.NewRateLimitMiddleware(cfg) // goroutine cleanup job
engine := NewEngine(cfg, appRouter, log, rateLimit)
```

> Handler bên trong router có thể là giá trị zero (`nil` field) vì route chỉ wrap `shared_pres.WrapHandler(h.Method)` — method value không bị gọi lúc dựng.

## Trình tự thực hiện

1. **Bước 0 (trước refactor)**: viết test + sinh golden `routes_baseline.txt` từ code HIỆN TẠI (10 router cũ). Commit baseline.
2. **Sau Đợt 0**: test phải pass (không đổi route).
3. **Sau Đợt 1**: test phải pass (route giữ nguyên dù di chuyển).
4. Nếu fail → đối chiếu diff để biết route nào thay đổi → sửa hoặc xác nhận cố ý.

## Ràng buộc

- Golden file phải được tạo TRƯỚC refactor — nếu tạo sau, baseline sai, test vô nghĩa (spec Edge Cases).
- Test nằm trong `make test` (go test ./...).
- Không phụ thuộc DB/Redis/network.

## Cách dùng

```bash
go test ./internal/api/routes/ -run TestRoutesBaseline -update   # sinh/cập nhật golden
go test ./internal/api/routes/                                    # verify
```
