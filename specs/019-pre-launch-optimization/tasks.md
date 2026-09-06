---
description: "Task list for pre-launch optimization (hardening, performance, maintainability)"
---

# Tasks: Tối Ưu Hóa Pre-Launch — Hardening, Hiệu Năng và Độ Bảo Trì

**Input**: Design documents from `/specs/019-pre-launch-optimization/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/http-behavior.md ✅, quickstart.md ✅ (đã áp remediation của /speckit-analyze: F1–F6)

**Tests**: Có yêu cầu tường minh trong spec (US2 là bản thân bộ test + FR-014 chính sách cấp độ test) — task test là bắt buộc.

**Organization**: Nhóm theo user story (US1 hardening → US2 test suite → US3 cache → US4 gzip/migration → US5 hygiene), mỗi story độc lập kiểm thử được.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Chạy song song được (khác file, không phụ thuộc task chưa xong)
- **[Story]**: User story sở hữu task (US1..US5)
- Đường dẫn file tuyệt đối so với repo root

## Path Conventions

Go module `smart-wardrobe-be` — paths theo cấu trúc `internal/` hiện hành (xem plan.md §Project Structure).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Dependency, error factory và cấu hình dùng chung cho nhiều story

- [x] T001 [P] Thêm dependency `github.com/gin-contrib/gzip` vào go.mod (`go get github.com/gin-contrib/gzip`)
- [x] T002 [P] Thêm factory `NewRequestEntityTooLarge(detail)` (HTTP 413, title tiếng Việt theo pattern hiện hành) vào `internal/shared/application/constants/apperror/dynamic_errors.go` + test factory trong `dynamic_errors_test.go`
- [x] T003 [P] Mở rộng config: `Server.ReadTimeoutSeconds/WriteTimeoutSeconds/IdleTimeoutSeconds/GzipEnabled` + struct `BodyLimit{DefaultMB, ElevatedMB, FreeRoutes []string}` trong `internal/config/config.go`, cập nhật file config yaml mẫu với mặc định 30/60/120, gzip=true, 2/10/[]

**Checklist Phase 1**: `go build ./...` pass sau mỗi task.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Ràng buộc cấu hình dùng chung trước khi story sử dụng

- [x] T004 Thêm validation khởi động trong `internal/bootstrap/startup_validator.go`: ở production bắt buộc `read/write/idle timeout > 0`, `body_limit.default_mb > 0`, `elevated_mb >= default_mb` — fail fast với message rõ ràng

---

## Phase 3: User Story 1 — HTTP Hardening: Timeouts + Body Limit (P1)

**Story Goal**: Server từ chối kết nối treo và body vượt ngưỡng theo mô hình 3 tầng, không phá SSE.

**Independent Test**: curl body >2MB vào route mặc định → 413 chuẩn; body lớn vào route tầng nâng → không 413; slowloris unit test → đóng sau read timeout; stream handler không bị cắt write deadline (verify stream > 60s — xem T010). Xem quickstart.md §US1.

- [x] T005 [US1] Tạo middleware `internal/shared/presentation/middleware/body_limit.go`: wrap `http.MaxBytesReader` theo tier table (free registry config → elevated patterns `POST /api/v1/ai/*`, brand chat/portal/admin, user-write brand → default 2MB) — pattern chi tiết tại research.md D2; trả 413 qua `NewRequestEntityTooLarge`; log có cấu trúc method/path/tier/content-length khi từ chối
- [x] T006 [P] [US1] Unit test `internal/shared/presentation/middleware/body_limit_test.go`: 413 đúng envelope ở tầng default; route tầng nâng nhận >2MB; route trong free registry bỏ qua wrap; sai pattern không nắn tier
- [x] T007 [US1] Đăng ký `BodyLimitMiddleware` trong `internal/api/routes/router.go` (sau GlobalErrorHandler, trước rate-limit)
- [x] T008 [US1] Gia hạn write deadline cho SSE: handler `StreamChatMessage` trong `internal/modules/fashion/presentation/handler/fashion_ai_handler.go` dùng `http.NewResponseController(c.Writer).SetWriteDeadline(...)` theo từng chunk ghi ra (không đổi hành vi stream khác)
- [x] T009 [US1] Gắn `ReadTimeout/WriteTimeout/IdleTimeout` cho `http.Server` trong `internal/bootstrap/app.go` đọc từ config mới
- [x] T010 [US1] Xác minh theo quickstart.md §US1: chạy curl 413 + unit slowloris/timeout + **verify SSE dài: stream giả lập kéo dài > write timeout (60s) vẫn truyền dữ liệu không bị cắt**; `make test-race ./internal/shared/presentation/middleware/...` pass

---

## Phase 4: User Story 2 — Bộ Test Billing / Quota / RBAC (P1)

**Story Goal**: Mạng an toàn tự động cho luồng tiền, hạn mức và phân quyền — unit cho mọi use case, integration DB thật cho webhook + renewal.

**Independent Test**: `make test-race` (unit) và `make test-integration` (integration, env-skip `TEST_DATABASE_URL`) pass; ma trận kịch bản khớp data-model.md §5.

- [x] T011 [US2] Unit test `internal/modules/subscription/application/usecase/purchase/checkout_test.go` (mock repo/uow): mua happy path; đang có giao dịch pending → tái sử dụng payment link; validate purchase thất bại
- [x] T012 [P] [US2] Unit test `internal/modules/subscription/application/usecase/quota/quota_test.go`: vượt hạn mức AI bị chặn với lỗi hạn mức; reset theo ngày khôi phục; ghi nhận sử dụng (outfit/chat) đúng
- [x] T013 [P] [US2] Unit test RBAC: `internal/modules/identity/presentation/middleware/` (roles authorize matrix: admin/user × admin routes; guest bị chặn) + test phân quyền brand portal trong `internal/modules/brand/presentation/middleware/` (owner/staff × portal routes)
- [x] T014 [US2] Unit test renewal workflow `internal/modules/subscription/application/usecase/subscription/renewal_test.go` (mock repo/uow/contract): phân nhánh gói finite/paid; toggle auto-renew đổi luồng; lỗi tải plan → fallback default plan
- [x] T015 [US2] Integration test `internal/modules/subscription/application/usecase/payment/webhook_integration_test.go` — **hàm test đặt tên tiền tố `TestIntegration...`** để khớp `-run Integration` của Makefile (pattern env-skip `TEST_DATABASE_URL` theo messaging integration_test.go): payment success apply đúng 1 lần (idempotent tái phát cùng order); giao dịch trùng bị bỏ qua an toàn; lệch số tiền → trạng thái phù hợp
- [x] T016 [US2] Integration test `internal/modules/subscription/application/usecase/subscription/renewal_integration_test.go` — **hàm test đặt tên tiền tố `TestIntegration...`**: gia hạn thành công apply đúng; thất bại → fallback default plan đúng trạng thái
- [x] T017 [US2] Mở rộng `Makefile` target `test-integration` chạy thêm `go test -race -run Integration ./internal/modules/subscription/...` (giữ nguyên messaging test hiện hành)

---

## Phase 5: User Story 3 — Cache Tổng Quan Gói Cước (P2)

**Story Goal**: AI hot path đọc overview qua cache theo user, TTL ≤60s, invalidation tại mutation, quota không qua cache.

**Independent Test**: unit cache hit/miss/fallback + invalidation; log invalidation xuất hiện khi mua/gia hạn; tắt Redis request AI vẫn chạy. Xem quickstart.md §US3.

- [x] T018 [US3] Tạo interface `ISubscriptionOverviewCache` (Get/Delete theo userID) tại `internal/modules/subscription/application/interface/caching/overview_cache.go` + cache entry struct theo data-model.md §3
- [x] T019 [US3] Redis impl `internal/modules/subscription/infrastructure/caching/redis_overview_cache.go` (key `sub:overview:{userID}`, TTL từ config `cache.overview_ttl_seconds` mặc định 60, lỗi Redis → miss an toàn) + wire provider trong `internal/di/wire.go`
- [x] T020 [US3] Read-through tại `internal/modules/subscription/presentation/contract/subscription_contract_impl.go`: cache hit trả entry → map sang `UserSubscriptionContractDTO`; miss → gọi usecase như cũ rồi fill cache (mapper tại `application/mapper/subscription.go`)
- [x] T021 [US3] Gọi `Delete(userID)` + log invalidation tại các mutation: apply sau checkout (`purchase/checkout.go`), renewal apply thành công (`subscription_renewal_workflow.go`), admin set plan active, set auto-renew toggle, webhook reconciliation apply
- [x] T022 [US3] Unit test cache: hit/miss/fallback-lỗi-Redis; invalidation được gọi ở từng mutation point; xác nhận quota (`IUserQuotaContract`) không đụng cache

---

## Phase 6: User Story 4 — Gzip + Migration Đa Instance (P2)

**Story Goal**: Nén phản hồi JSON loại trừ SSE; chứng minh migration lock an toàn đa instance.

**Independent Test**: curl gzip giảm ≥50% kích thước, giải nén khớp 100%; SSE không nén; 2 tiến trình migration đồng thời không xung đột. Xem quickstart.md §US4.

- [ ] T023 [US4] Đăng ký gzip middleware trong `internal/api/routes/router.go` (`gin-contrib/gzip`, toggle `server.gzip_enabled`, `WithExcludedPaths` cho route stream `/api/v1/ai/chat/sessions/*/messages/stream`)
- [ ] T024 [P] [US4] Inject logger vào `internal/shared/infrastructure/db/migration.go`: thay `fmt.Println` bằng log có cấu trúc (bắt đầu/kết thúc/từng migration/kết quả lock) — giữ nguyên session locker
- [ ] T025 [US4] Integration test `internal/shared/infrastructure/db/migration_integration_test.go` (env-skip, hàm test tiền tố `TestIntegration...`): 2 provider `RunMigrations` đồng thời trên cùng DB — không xung đột, kết quả idempotent
- [ ] T026 [US4] Xác minh quickstart.md §US4: curl gzip size ≤50% + giải nén diff 100% khớp; SSE trả không nén

---

## Phase 7: User Story 5 — Hygiene Pass (P3)

**Story Goal**: Tách 2 file chạm ngưỡng 500 dòng thành các file nghiệp vụ <500 dòng, hàm <80 dòng; dọn hết fmt.Println — pure refactor.

**Independent Test**: `wc -l` các file kết quả đạt ngưỡng; `grep fmt.Println` production rỗng; `make test-race` pass; swagger diff không đổi schema. Xem quickstart.md §US5.

- [ ] T027 [US5] Tách `internal/modules/wardrobe/presentation/handler/wardrobe_item_handler.go` (501 dòng) theo ranh giới data-model.md §6: `wardrobe_item_handler.go` (CRUD item) + `wardrobe_catalog_handler.go` (system catalog + upload signature) — di chuyển message constants tương ứng, không đổi hành vi/route
- [ ] T028 [US5] Tách `internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go` (495 dòng) thành orchestration + `renewal_payment.go` + `renewal_fallback.go`; extract `executePaidRenewal` (90 dòng) thành các hàm bước <80 dòng — pure refactor, không đổi logic
- [ ] T029 [US5] Thay `fmt.Println` còn lại trong `internal/bootstrap/app.go` (banner + shutdown) bằng logger đã inject — **chạy sau T024** (migration.go cũng thay fmt.Println trong US4); xác nhận `grep -rn "fmt.Println" internal --include="*.go" | grep -v _test` trả rỗng
- [ ] T030 [US5] Xác minh quickstart.md §US5: `wc -l` đạt ngưỡng, `make test-race` pass sau tách

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Gate cuối cùng toàn feature

- [ ] T031 Chạy `make generate` (wire + swagger) và audit swagger diff theo SC-008: không field công khai nào bị xóa/đổi — ghi kết quả vào quickstart checklist
- [ ] T032 Chạy gate toàn phần: `make lint` + `make test-race` + `make test-integration` (khi có TEST_DATABASE_URL) + đi lần lượt quickstart.md từng section — đối chiếu SC-001..SC-008

---

## Dependencies & Execution Order

### Story phases

```text
Phase 1 (Setup T001-T003) → Phase 2 (T004)
  → Phase 3 (US1: T005-T010)   [P1 — ưu tiên đầu]
  → Phase 4 (US2: T011-T017)   [P1 — độc lập US1, chạy song song được sau Phase 2]
  → Phase 5 (US3: T018-T022)   [P2 — độc lập US1/US2, phụ thuộc T003 (config TTL)]
  → Phase 6 (US4: T023-T026)   [P2 — T023 phụ thuộc T001/T003; T024 độc lập]
  → Phase 7 (US5: T027-T030)   [P3 — làm CUỐI: tách file đụng file mà US3/US4 cũng sửa
                                 (renewal workflow, bootstrap) → tránh merge conflict]
  → Phase 8 (Polish: T031-T032)
```

### Cross-story dependencies chính

- T005/T007 (body limit) phụ thuộc T002 (413 factory) + T003 (config tiers)
- T009 (server timeouts) phụ thuộc T003 + T004
- T019/T020 (cache) phụ thuộc T003 (TTL config); T021 phụ thuộc T019
- T023 (gzip) phụ thuộc T001 + T003
- T025 (migration integration test) phụ thuộc T024 (logger, env-skip pattern)
- T027/T028 (tách file) phải chạy sau T014/T015/T016/T021 để tránh đụng cùng file renewal workflow
- T029 (fmt.Println cleanup "rỗng") phụ thuộc T024 đã thay migration.go trước

### Parallel Opportunities

- Trong Phase 3: T006 song song với T005 (test viết sau nhưng file riêng), T008 song song T009
- Toàn bộ Phase 4 (T011-T017) chạy song song với Phase 3 — khác hẳn phạm vi file
- T012, T013, T014 song song với nhau và với T011
- T024 song song với T023 (file khác nhau)

## Implementation Strategy

- **MVP First (US1 + US2)**: sau Phase 3+4, hệ thống đã harden và có mạng an toàn luồng tiền — đây là mức tối thiểu an toàn để launch.
- **Incremental**: US3 (hiệu năng) → US4 (vận hành) → US5 (bảo trì) mỗi story là một increment test được độc lập, commit theo story.
- **Validation loop**: mỗi story kết thúc bằng gate riêng (mục Independent Test) trước khi sang story kế; gate toàn phần ở Phase 8 đối chiếu 8 success criteria.
- Lưu ý Git Bash: dùng `make test`/`make test-race`/`make lint` (đã fix shell-aware), không chạy `make hygiene-scan` nếu không được phép.
