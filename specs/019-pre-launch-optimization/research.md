# Research: Tối Ưu Hóa Pre-Launch (019)

**Date**: 2026-09-06 | **Spec**: [spec.md](spec.md)

Mọi NEEDS CLARIFICATION trong Technical Context đã được giải quyết từ khảo sát code thực tế (file:line kèm theo). Không còn mục NEEDS CLARIFICATION.

## D1 — HTTP server timeouts + SSE compatibility (US1, FR-001, FR-003)

**Decision**: Gắn `ReadTimeout`/`WriteTimeout`/`IdleTimeout` cho `http.Server` tại [app.go:88](../../../internal/bootstrap/app.go) (giá trị từ config mới `server.read_timeout_seconds/write_timeout_seconds/idle_timeout_seconds`, mặc định 30/60/120). Các handler streaming (SSE) sẽ **gia hạn write deadline trong handler** qua `http.NewResponseController(w).SetWriteDeadline(...)` sau mỗi chunk ghi ra, nên stream dài không bị cắt bởi WriteTimeout toàn cục.

**Rationale**: `http.Server` không có timeout nào hiện tại (audit: app.go:88-91 chỉ có Addr+Handler). `GlobalTimeoutMiddleware` hiện có ([middleware/timeout.go](../../../internal/shared/presentation/middleware/timeout.go)) chỉ hủy request context, không bảo vệ kết nối TCP. `ResponseController` (Go 1.20+) là cơ chế chuẩn để per-handler mở rộng deadline mà không tắt timeout toàn cục — giữ defense-in-depth cho mọi route khác.

**Alternatives considered**:
- WriteTimeout rất lớn (10 phút) để "chứa" SSE → loại: vô hiệu hóa ý nghĩa timeout đối với mọi route thường.
- Tắt WriteTimeout, chỉ dựa GlobalTimeoutMiddleware → loại: không cắt được kết nối chậm ở tầng transport.

## D2 — Body-limit 3 tầng + registry route free (US1, FR-002)

**Decision**: Middleware mới `shared_middleware.BodyLimitMiddleware` đặt ngay sau `GlobalErrorHandler`/trước handler: wrap `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)` với limit phân theo (method+path pattern): mặc định 2 MB; tầng nâng 10 MB cho 4 nhóm route đã clarify; tầng free (registry cấu hình `server.body_limit.free_routes`, đang trống) bỏ qua wrap. Vượt ngưỡng → trả 413 qua apperror.

**Rationale**: `http.MaxBytesReader` là cơ chế chuẩn, tự ngắt đọc khi vượt và tương thích pipeline Gin; tier table khai báo dạng pattern `METHOD prefix` trong code (đếm được, test được), riêng tầng free là **config-driven** (yaml) đúng yêu cầu "thêm route free không đụng code middleware". Route patterns tầng nâng (khảo sát router thực tế):
- Chat AI + phối đồ: `POST /api/v1/ai/*` (chat sessions, messages, messages/stream, outfit-recommendations) — [fashion router.go:40-49](../../../internal/modules/fashion/presentation/router/router.go)
- Chat user↔brand: `POST /api/v1/brands/:brandId/conversation/*`, `POST /api/v1/brand-portal/brands/:brandId/conversations/*`
- Brand mutation + description: mọi `POST|PUT|PATCH /api/v1/brand-portal/**`, `POST|PUT|PATCH /api/v1/admin/brands/**` + route user-write brand (`/brands/claim`, `/brands/:id/join-loyalty`, `/brand-items/:id/feedbacks`, `/brand-benefits/:id/redeem`) — [brand router.go:38-155](../../../internal/modules/brand/presentation/router/router.go)

**Alternatives considered**:
- Miễn hoàn toàn 4 nhóm (bỏ ngưỡng) → leader đã chốt dời sang ngưỡng nâng 10 MB: giữ được defense-in-depth.
- Đếm body bằng cách đọc sẵn vào memory rồi kiểm tra → loại: cấp phát bộ nhớ trước khi quyết định, phản bản chất của giới hạn.

**Ghi chú thiết kế phụ**: `apperror` hiện chưa có factory 413 — bổ sung `NewRequestEntityTooLarge(detail)` trong `dynamic_errors.go` theo đúng pattern các factory hiện hành (400/401/...).

## D3 — Gzip (US4, FR-009)

**Decision**: Thêm dependency `github.com/gin-contrib/gzip`; middleware đặt sau rate-limit, cấu hình `WithExcludedPaths` cho route SSE (`/api/v1/ai/chat/sessions/*/messages/stream`) và `WithExcludedPathsRegexs` cho extension tĩnh; bật/tắt bằng config `server.gzip_enabled` (mặc định bật ở production).

**Rationale**: Thư viện chuẩn của ecosystem Gin, hỗ trợ loại trừ path sẵn; không tự viết middleware nén. Response envelope JSON của dashboard/danh sách là text dữ liệu lớn → lợi ích nén rõ.

**Alternatives considered**:
- Tự viết gzip middleware → loại: trùng lặp thư viện cộng đồng, thêm bề mặt lỗi.
- Chỉ dựa reverse proxy tương lai → loại: hiện chưa có cam kết hạ tầng, để config toggle là an toàn cả hai thế giới.

## D4 — Cache tổng quan gói cước (US3, FR-007, FR-008)

**Decision**: Cache **intra-module** của subscription: interface `ISubscriptionOverviewCache` tại `application/interface/caching/overview_cache.go` (Get/Delete theo userID, entry chứa các field của `UserSubscriptionOverviewRes` + dấu thời điểm), impl Redis tại `infrastructure/caching/redis_overview_cache.go` (key `sub:overview:{userID}`, TTL `cache.overview_ttl_seconds` mặc định 60s). Đọc: `UserSubscriptionContract` impl ([subscription_contract_impl.go](../../../internal/modules/subscription/presentation/contract/subscription_contract_impl.go)) check cache → miss thì gọi usecase như cũ rồi fill cache. Invalidation: gọi `Delete(userID)` tại các điểm mutation trong cùng module — purchase apply (checkout uow apply), renewal workflow áp thành công, admin set plan active, set auto-renew toggle (IsAutoRenewEnabled thuộc overview), webhook reconciliation apply.

**Rationale**: Đúng Hiến pháp §II (không cross-module import mới — AI caller vẫn qua contract như cũ), §IV (interface application, impl infrastructure). Số đếm hạn mức (`UserSubscriptionDTO` qua `IUserQuotaContract`) không đụng vào — đúng FR-007. Invalidation in-module tránh phụ thuộc vòng event (subscription tự consume event của chính mình); TTL 60s là backstop cho mọi đường mutation bị sót.

**Alternatives considered**:
- Cache ở module fashion/wardrobe (bên gọi) → loại: phá ranh giới sở hữu dữ liệu, mỗi caller tự cache tạo phân mảnh.
- Invalidation qua RabbitMQ event mới → loại: vòng event tự-đ narative, phức tạp hơn gọi trực tiếp trong cùng module; PlanChanged event giữ nguyên vai trò hiện tại (grant AI).

## D5 — Migration lock đa instance (US4, FR-010)

**Decision**: **Không xây mới** — [migration.go](../../../internal/shared/infrastructure/db/migration.go) đã dùng `goose.WithSessionLocker(lock.NewPostgresSessionLocker())` (Postgres session-level advisory lock). Phạm vi còn lại: (1) thay `fmt.Println` bằng logger tập trung (trùng FR-012), (2) viết integration test chứng minh 2 tiến trình `RunMigrations` đồng thời chỉ 1 thực thi và instance kia chờ/thoát an toàn, (3) tài liệu hóa config deploy (không cần cấu hình thêm).

**Rationale**: Session-level advisory lock tự nhả khi phiên DB chết (đáp ứng edge case "khóa bị kẹt"), đúng nguyên tắc "tận dụng hiện trạng trước khi xây mới".

**Alternatives considered**: Tự viết bảng lock/mutex ứng dụng → loại: goose đã cung cấp, tự viết thêm bề mặt lỗi.

## D6 — Cấp độ test & hạ tầng integration (US2, FR-004..FR-006, FR-014)

**Decision**: Unit test dùng mock repo/contract theo pattern hiện hành (vd [usecase_test.go](../../../internal/modules/brand/application/usecase/item/usecase_test.go)) chạy trong `make test-race`. Integration test theo **pattern env-skip** của messaging ([integration_test.go: testEnvOrSkip("TEST_DATABASE_URL")](../../../internal/shared/infrastructure/messaging/integration_test.go)) — file `*_integration_test.go` trong module subscription, skip khi thiếu env; mở rộng `make test-integration` chạy thêm `./internal/modules/subscription/...`.

**Rationale**: Clarify Q1 đã chốt unit + integration cho webhook/renewal; pattern env-skip đã tồn tại → chi phí hạ tầng gần bằng 0; DB test dùng Postgres test có sẵn qua docker (TEST_DATABASE_URL).

**Alternatives considered**: Testcontainers thêm dependency mới → loại: env-skip + docker compose test DB hiện có là đủ cho quy mô hiện tại.

## D7 — Phạm vi test cụ thể (US2)

**Decision**: Ma trận test (chi tiết trong data-model.md §Test Coverage Matrix):
- Purchase: happy path, purchase trùng đang chờ thanh toán, validate thất bại.
- Renewal (integration): gia hạn thành công, thất bại → fallback default plan.
- Webhook reconciliation (integration): payment success apply đúng 1 lần (idempotent), giao dịch trùng bị bỏ qua an toàn, số tiền lệch.
- Quota: vượt hạn mức bị chặn, reset theo ngày, ghi nhận sử dụng.
- RBAC (identity/brand middleware + router test): ma trận system admin/user × admin routes; brand owner/staff × portal routes; guest chặn.

**Rationale**: Bám sát acceptance scenarios của spec; mỗi luồng ≥1 happy + 2 edge theo SC-004.

## D8 — File splits (US5, FR-011) + fmt.Println (FR-012)

**Decision**:
- `wardrobe_item_handler.go` (501 dòng) → tách theo nhóm nghiệp vụ handler: `wardrobe_item_handler.go` (item CRUD) + `wardrobe_catalog_handler.go` (system catalog + upload signature) + `wardrobe_stats_handler.go` (stats/search hỗ trợ) — ranh giới chốt khi implement theo method thực tế, mỗi file < 500 dòng.
- `subscription_renewal_workflow.go` (495 dòng) → tách theo bước: orchestration chính + `renewal_payment.go` (luồng paid renewal, extract `executePaidRenewal` 90 dòng thành các bước nhỏ < 80 dòng) + `renewal_fallback.go` (fallback default plan).
- `fmt.Println` còn lại: [app.go:93-96,144,173](../../../internal/bootstrap/app.go) và [migration.go:42,52](../../../internal/shared/infrastructure/db/migration.go) → logger inject qua DI (bootstrap đã có logger; migration cần thêm tham số logger).

**Rationale**: Tách theo ranh giới nghiệp vụ đúng code-hygiene.md H1 + quy tắc "one file per business group" của mapper/handler pattern; pure refactor — compile-driven, không đổi hành vi.

## Constitution Re-check (post-design)

- §II: cache là intra-module subscription; không import chéo mới → ✅
- §III: không tạo event/queue mới; tái sử dụng logger/metrics hiện có (clarify Q2) → ✅
- §IV: mọi file mới đúng layer (middleware → shared/presentation; cache interface → application/interface; impl → infrastructure) → ✅
- §V: 413 qua apperror factory mới theo pattern → ✅
- §VI: không đụng quota/cost pipeline → ✅
- §VII: file mới tuân Handler Pattern và DTO convention 018 → ✅

**Kết luận**: 0 violation, không cần Complexity Tracking.
