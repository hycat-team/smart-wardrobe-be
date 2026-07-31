# Research — Củng cố Ranh Giới Module (Endpoint Ownership, Auth Middleware & Swagger Per-Module)

**Branch**: `010-module-ownership-refactor` | **Date**: 2026-08-01

## Mục đích

Giải quyết toàn bộ điểm NEEDS CLARIFICATION / câu hỏi nghiên cứu của plan trước khi design chi tiết.

---

## R1 — Route-registry test không cần DB (FR-017)

### Câu hỏi
Làm sao snapshot tập route `/api/v1/*` để so sánh trước/sau refactor mà không cần chạy DB/Redis (vì `make test` chạy local không có infra)?

### Decision
Dùng `engine.Routes()` của Gin — trả `gin.RoutesInfo` (Method + Path + Handler). Viết 1 test ở `internal/api/routes/` dựng engine bằng `NewEngine(cfg, appRouter, log, rateLimit)` với các router module/handler **nil-dep** (constructor của router chỉ gán field, không deref lúc khởi tạo — đã xác minh từ pattern hiện có). Test so sánh tập `(method, path)` của engine với **golden file** snapshot.

### Cách hoạt động
1. **Trước refactor (Đợt 0 bắt đầu)**: tạo file `internal/api/routes/routes_test.go` + golden `testdata/routes_baseline.txt`. Chạy 1 lần để chụp baseline (130 paths) từ code hiện tại.
2. **Sau mỗi đợt**: `make test` chạy test so sánh baseline với `engine.Routes()` thực tế.
3. Nếu path/method thay đổi → test fail → bắt buộc xem xét breaking change.

### Lưu ý
- Handler nil-dep: `WardrobeItemHandler{}` rỗng vẫn đăng ký được route (route chỉ wrap `shared_pres.WrapHandler(h.Method)` — method value không bị gọi lúc khởi tạo).
- Middleware: `NewRateLimitMiddleware(cfg)` cần cfg thật; dùng config test tối thiểu. `AuthMiddleware` cần `ITokenBlacklistService` — có thể nil vì `Handle()` không gọi lúc dựng.
- Golden file phải commit cùng code.

### Rationale
Không cần infra, deterministic, chạy trong CI. Phát hiện chính xác "mất route" hoặc "đổi method" — vấn đề rủi ro nhất của refactor di chuyển route.

### Alternatives considered
- Dựng app thật + DB/Redis: nặng, không chạy được `make test` local.
- So sánh swagger.json trước/sau: swagger sinh từ annotation handler (không từ router) nên KHÔNG phát hiện route bị mất khi di chuyển router — chỉ phát hiện đổi annotation. Không đủ.
- Manual smoke test: dễ sót với 130 paths.

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

Gin không conflict vì các path phân nhóm khác nhau (không có wildcard cùng cấp trùng nhau). Route-registry test sẽ bắt lỗi nếu có conflict (Gin panic khi dựng).

### Rationale
Giữ thứ tự gần với hiện tại để giảm rủi ro; route-registry test phát hiện conflict sớm.

---

## Kết luận

Mọi điểm mơ hồ của Technical Context đã được giải quyết. Không còn NEEDS CLARIFICATION. Sẵn sàng chuyển sang Phase 1 (data-model, contracts, quickstart).
