# Research — Củng cố Ranh Giới Module (Endpoint Ownership, Auth Middleware & Swagger Per-Module)

**Branch**: `010-module-ownership-refactor` | **Date**: 2026-08-01

## Mục đích

Giải quyết toàn bộ điểm NEEDS CLARIFICATION / câu hỏi nghiên cứu của plan trước khi design chi tiết.

---

## R1 — Route-registry test (đã triển khai rồi gỡ bỏ)

### Câu hỏi ban đầu
Làm sao snapshot tập route `/api/v1/*` để so sánh trước/sau refactor mà không cần chạy DB/Redis?

### Quyết định ban đầu (đã thực thi trong lúc refactor)
Dùng `engine.Routes()` của Gin — trả `gin.RoutesInfo` (Method + Path + Handler). Test `internal/api/routes/routes_test.go` dựng engine với handler **nil-dep** và so sánh tập `(method, path)` với golden file `testdata/routes_baseline.txt`. Baseline được chụp TRƯỚC khi refactor (150 routes) và xác nhận giữ nguyên 100% sau Đợt 1.

### Quyết định sau triển khai: GỠ BỎ
Sau khi feature hoàn tất, test + golden file + các Makefile target liên quan bị xóa hoàn toàn. Lý do:
- Giá trị chính (bảo vệ refactor 150 routes) đã hoàn thành.
- Route được khai báo tĩnh trong code — rủi ro "mất route" do vô ý rất thấp và dễ thấy trong code review.
- Swagger.json (130 paths) + `generate-check` đã là route inventory tự động.
- Tránh gánh nặng bảo trì: mỗi endpoint mới phải chạy `-update` khiến test thành false alarm.

Tham chiếu: spec Clarifications.

---

## R2 — Script split swagger per-module (FR-013)

### Câu hỏi
Dùng ngôn ngữ/tool nào để tách master swagger.json thành 6 spec per-module?

### Decision
**Go script** tại `internal/tools/swagger-split/main.go`, chạy qua `go run ./internal/tools/swagger-split -master api/swagger/swagger.json -mapping contracts/swagger-module-mapping.json -out api/swagger/modules`.

### Chức năng
1. Đọc master `swagger.json` (struct chuẩn của Swagger 2.0 / OpenAPI).
2. Đọc mapping tag→module (JSON: `{ "identity": ["Auth","Me","Admin Identity"], ... }`).
3. Với mỗi module: lọc `paths` — mỗi path operation thuộc module nếu **mọi tag của operation** đều nằm trong mapping module đó (an toàn nhất, tránh lọt operation multi-tag vào 2 module).
4. Giữ nguyên `info`, `definitions` (hoặc lọc `definitions` theo ref cần thiết), `securityDefinitions`.
5. Ghi `api/swagger/modules/<name>/swagger.json`.

### Rationale
- Cross-platform (Makefile đã có nhánh Windows/Unix riêng).
- Tái dùng struct JSON — không phải parse chuỗi như PowerShell.
- `make generate-check` diff được thư mục `api/swagger` (đã có trong `generate-check`).

### Alternatives considered
- PowerShell/jq: không ổn định cross-platform, parse JSON phức tạp.
- Chạy `swag init` N lần với `--tags`: chậm (N lần parse toàn tree), sinh `docs.go` thừa, dễ drift.

---

## R3 — Tag `Admin` dùng chung 3 module (Q1 Option B)

### Câu hỏi
Theo Q1 Option B (giữ tag nhóm, mỗi tag thuộc đúng 1 module), tag `Admin` hiện xuất hiện ở handler của identity, wardrobe, brand. Xử lý thế nào?

### Decision
Tách `@Tags Admin` thành tag riêng theo module:
- identity (admin users endpoints): `@Tags Admin Identity`
- wardrobe (admin wardrobe-items + categories): `@Tags Admin Wardrobe`
- brand (admin brands): `@Tags Admin Brand`

Các tag khác đã thuộc đúng 1 module (xem mapping đầy đủ trong contracts).

### Rationale
Q1 Option B: giữ phân nhóm, chỉ đảm bảo mỗi tag 1 module. Không gom về 1 tag/module. Tách `Admin` là thay đổi tối thiểu duy nhất cần thiết.

### Note
Đây là đổi tên hiển thị trong docs, không phải API contract (đã ghi ở spec edge case). Cần thông báo trong changelog/PR.

---

## R4 — Import cycle check (FR-006, SC-006)

### Câu hỏi
Đưa middleware vào `shared` và auth vào identity có tạo import cycle?

### Decision
Không. Đã xác minh qua grep:
- `shared/presentation/middleware/*.go` chỉ import: `config`, `logger`, `errorutils`, `apperror`, `roleslug`, `contextutils`, `x/time/rate`, `gin`, `zap` — **không** import `internal/modules/*`.
- `identity/presentation/contract` chỉ import `gin`.
- `auth.go` (đặt tại `identity/presentation/middleware/`) import `identityerrors`, `identity_security`, `contextutils`, `jwtutils`, `apperror`, `jwttype` — toàn bộ đều là identity-module-internal hoặc shared/pkg → không vòng.

### Rationale
Chuỗi import là DAG: `shared/presentation/middleware` → (chỉ shared/pkg/config); `identity/presentation/middleware` → identity internal + shared. Không có shared→module hay module→shared→module cycle.

---

## R5 — Wire bind interface IAuthMiddleware

### Câu hỏi
`NewAuthMiddleware` trả `*middleware.AuthMiddleware` (concrete). Consumer (router module) cần `identity_contract.IAuthMiddleware`. Wire xử lý thế nào?

### Decision
Trong `internal/modules/identity/presentation/provider.go` (hoặc module provider):

```go
var ProviderSet = wire.NewSet(
    middleware.NewAuthMiddleware,
    wire.Bind(new(contract.IAuthMiddleware), new(*middleware.AuthMiddleware)),
    // ... hiện có
)
```

Các router module nhận param `auth identity_contract.IAuthMiddleware`. Wire auto-resolve interface qua Bind.

### Rationale
Pattern chuẩn Wire cho interface; dự án đã dùng `wire.Bind` tương tự trong `shared/provider.go`.

---

## R6 — errorutils noise-candidates cần cập nhật path

### Câu hỏi
`internal/pkg/utils/errorutils/errorutils.go:22-27` hardcode path middleware cũ. Khi di chuyển có ảnh hưởng không?

### Decision
Có — cập nhật `noiseCandidateFunctions`:
- `internal/api/middleware.CORSMiddleware` → `internal/shared/presentation/middleware.CORSMiddleware`
- `internal/api/routes.NewEngine.(*RateLimitMiddleware).Handle` → `internal/shared/presentation/middleware.(*RateLimitMiddleware).Handle` (RateLimitMiddleware giờ ở shared)

Giữ nguyên `internal/api/routes.NewEngine.GlobalErrorHandler` và `GlobalTimeoutMiddleware` (NewEngine vẫn ở `internal/api/routes`).

### Test
`internal/pkg/utils/errorutils/errorutils_test.go:60-65,129-132` có stack frame hardcode path `internal/api/middleware/...` — phải cập nhật theo path mới, nếu không test fail.

### Rationale
Filter stack trace dựa trên prefix match — path sai = stack trace nhiễu, test fail.

---

## R7 — Thứ tự gọi Init giữa các module router (route conflict)

### Câu hỏi
NewEngine gọi `Init(api)` cho 6 router module theo thứ tự nào để tránh Gin panic wildcard conflict?

### Decision
Thứ tự hiện tại trong `router.go` giữ nguyên tương đối, nhóm theo module:
1. `IdentityRouter.Init(api)` — `/auth`, `/me`, `/admin/users`
2. `SubscriptionRouter.Init(api)` — `/subscriptions`
3. `WardrobeRouter.Init(api)` — `/wardrobe-items`, `/me/wardrobe-items`, `/system-catalog`, `/outfits`, `/me/outfits`, `/categories`, `/admin/...`
4. `DashboardRouter.Init(api)` — `/dashboard/analytics`, `/me/dashboard`, `/brand/dashboard`, `/admin/dashboard`
5. `BrandRouter.Init(api)` — `/brands`, `/brand-items`, `/brand-benefits`, `/me/brand-loyalties`, `/brand-portal`, `/admin/brands`
6. `FashionRouter.Init(api)` — `/ai`

Gin không conflict vì các path phân nhóm khác nhau (không có wildcard cùng cấp trùng nhau). Khi khởi động app, Gin sẽ panic nếu có conflict path — dùng smoke test `make run` để xác nhận.

### Rationale
Giữ thứ tự gần với hiện tại để giảm rủi ro; conflict path sẽ bị Gin panic khi dựng engine và bắt được qua smoke test.

---

## Kết luận

Mọi điểm mơ hồ của Technical Context đã được giải quyết. Không còn NEEDS CLARIFICATION. Sẵn sàng chuyển sang Phase 1 (data-model, contracts, quickstart).
