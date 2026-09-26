---
description: "Task list for feature 023-analyze-status-handling"
---

# Tasks: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Input**: Design documents from `/specs/023-analyze-status-handling/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Có bao gồm task test cho các đơn vị logic quan trọng (theo quy ước dự án: `make test-race` là gate bắt buộc).

**Organization**: Tasks grouped by user story để có thể triển khai/kiểm thử độc lập.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Có thể chạy song song (file khác nhau, không phụ thuộc task chưa xong)
- **[Story]**: US1–US4 tương ứng spec.md
- Bao gồm đường dẫn file chính xác

## Path Conventions

- Workspace multi-module: `shared/`, `services/commerce/`, `services/account/`, `services/ai/`
- Feature docs: `specs/023-analyze-status-handling/`
- Lưu ý: module `wardrobe`, `fashion`, `brand` đều thuộc service `commerce` → mọi regenerate Swagger dùng `make swagger svc=commerce` (không có `svc=brand`/`svc=wardrobe`)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Chuẩn bị migration, config, env cho toàn feature.

- [ ] T001 [P] Tạo Goose migration `services/commerce/database/migrations/<YYYYMMDDHHMMSS>_add_brand_item_failed_status.sql` (Up: drop + add `chk_brand_items_status` gồm `'failed'`, schema-qualify `brand.brand_items`; Down: revert về `draft|active|archived`)
- [ ] T002 [P] Thêm config `ReplayBufferSize` (mapstructure `replay_buffer_size`, default 100) và `ReplayBufferTTLSeconds` (mapstructure `replay_buffer_ttl_seconds`, default 300) vào struct `SSE` trong `shared/config/config.go`; thêm block `server.sse.replay_buffer_size` + `server.sse.replay_buffer_ttl_seconds` vào `configs/config.yaml` và `configs/config.production.yaml`; override env `SSE_REPLAY_BUFFER_SIZE` + `SSE_REPLAY_BUFFER_TTL_SECONDS` trong `shared/config/loader.go`; validation `> 0` cho cả hai trong `shared/config/validation.go`
- [ ] T003 [P] Cập nhật `.env.example`, `.env.production.example`, `.env.aws.example` với `SSE_REPLAY_BUFFER_SIZE` và `SSE_REPLAY_BUFFER_TTL_SECONDS` (comment tiếng Anh)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Transport realtime (Redis Pub/Sub) + buffer phát lại + chuyển các publisher hiện có. Chặn US2 và US4.

**⚠️ CRITICAL**: Hoàn thành trước khi bắt đầu US2/US4 (US1 độc lập, chỉ cần migration T001 nếu chạm brand).

- [ ] T004 [P] Thêm interface `IRealtimePublisher` (`Publish(ctx, taskId, data)`) và `IRealtimeRelay` (`Start(ctx)`/`Stop()`), kèm helper channel `sse:task:{taskId}`, trong `shared/application/sse/realtime.go`
- [ ] T005 [P] Implement `RedisRealtimePublisher` (`*redis.Client`, `PUBLISH sse:task:{taskId}`) trong `shared/infrastructure/messaging/redis_realtime_publisher.go`
- [ ] T006 [P] Implement `SSERelay` (`PSubscribe sse:task:*` → `ISSEManager.Publish`) trong `shared/presentation/sse_relay.go` (log lỗi, không chặn nghiệp vụ)
- [ ] T007 Mở rộng `SSEManager` trong `shared/application/sse/sse_manager.go`: ring buffer N event/`taskId` (N từ config `replay_buffer_size`), `Subscribe` phát lại buffer FIFO cho subscriber mới; loại event khi vượt N hoặc hết TTL (`replay_buffer_ttl_seconds`) — KHÔNG xóa buffer khi subscriber cuối cùng `Unsubscribe` (để reconnect muộn sau khi phiên kết thúc vẫn phát lại được)
- [ ] T008 Đăng ký provider + bind mới (`RedisRealtimePublisher` → `IRealtimePublisher`, `SSERelay`) trong `services/commerce/internal/di/wire.go`, rồi chạy `make wire`
- [ ] T009 Khởi động/dừng `SSERelay` trong `services/commerce/internal/bootstrap/app.go` cho mode `http|all` (KHÔNG đưa vào `bootstrap.Workers`)
- [ ] T010 Thay các lời gọi SSE publish hiện có sang `IRealtimePublisher`: `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/fashion_event_uc.go` (2 chỗ) và `services/commerce/internal/modules/brand/application/usecase/item/fashion_event_uc.go` (2 chỗ)

**Checkpoint**: Transport realtime sẵn sàng cho cả 2 topology.

---

## Phase 3: User Story 1 - Rà soát & xác nhận món đồ AI không chắc chắn (Priority: P1) 🎯 MVP

**Goal**: Người dùng xác nhận món `needs_review` (có danh mục hợp lệ) → `inWardrobe`.

**Independent Test**: Tạo món `needsReview` có category hợp lệ → gọi `confirm-review` → status `inWardrobe` và tham gia gợi ý phối đồ.

- [ ] T011 [US1] Thêm method `ConfirmWardrobeReview(ctx, userID, itemID, now)` vào `services/commerce/internal/modules/wardrobe/domain/repositories/interfaces.go` + impl atomic (WHERE status = NeedsReview) trong `services/commerce/internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`
- [ ] T012 [P] [US1] Thêm `IsWearableCategory(ctx, categoryID uuid.UUID) (bool, error)` vào `services/commerce/internal/modules/fashion/presentation/contract/fashion_contract.go` + impl trong `fashion_contract_impl.go` (dùng ánh xạ category→wearing role, `other`/không có = false)
- [ ] T013 [US1] Implement `ConfirmWardrobeItemReview` trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go`: kiểm tra ownership + status `NeedsReview`; lấy category qua `IFashionContract.GetFashionItem` từ `item.FashionItemID` (nếu `FashionItemID == nil` hoặc fashion item/category nil → trả `ErrConfirmReviewCategoryRequired`); gọi `IsWearableCategory`; chuyển trạng thái nguyên tử và publish `wardrobe.created` trong `uow.Execute`
- [ ] T014 [P] [US1] Thêm lỗi `ErrConfirmReviewForbidden`, `ErrConfirmReviewCategoryRequired` trong `services/commerce/internal/modules/wardrobe/application/errors/errors.go`
- [ ] T015 [US1] Thêm method vào interface usecase (`.../application/interface/usecase/wardrobe_uc.go`), handler `ConfirmWardrobeItemReview` trong `.../wardrobe/presentation/handler/wardrobe_item_handler.go`, route `POST /wardrobe-items/:id/confirm-review` trong `.../wardrobe/presentation/router/router.go`
- [ ] T016 [P] [US1] Unit test UC confirm-review (ownership sai, status sai, thiếu category, thành công) trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/*_test.go`; đồng thời xác minh FR-003/SC-001 (món sau xác nhận xuất hiện trong gợi ý phối đồ) qua quickstart G1
- [ ] T017 [US1] Regenerate Swagger cho endpoint mới: `make swagger svc=commerce` (wardrobe/brand đều thuộc service commerce)

**Checkpoint**: US1 độc lập hoạt động và kiểm thử được.

---

## Phase 4: User Story 2 - Theo dõi tiến trình realtime đầy đủ, không mất sự kiện (Priority: P1)

**Goal**: Phát SSE `processing` cho từng món + phát lại khi reconnect.

**Independent Test**: Batch upload → SSE nhận `processing` từng món, rồi `completed/needs_review/failed`; reconnect giữa chừng nhận lại event.

- [ ] T018 [P] [US2] Thêm `Processing AnalyzeTaskStatus = "processing"` vào `shared/application/constants/analyzetaskstatus/analyzetaskstatus.go`
- [ ] T019 [P] [US2] Thêm topic `TopicFashionProcessingWardrobe`/`TopicFashionProcessingBrand` vào `shared/application/constants/eventconstants/eventconstants.go` và routing key alias trong `shared/infrastructure/messaging/rabbitmq_constants.go`
- [ ] T020 [US2] Bind thêm routing key processing vào 2 binding analyzed hiện có trong `shared/infrastructure/messaging/topology.go` (không tạo queue/DLQ mới)
- [ ] T021 [US2] Publish event `fashion.event.processing.*` ở đầu `ProcessBackgroundBatchUploadJob` trong `services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go` (trong `uow.Execute`, switch `job.ItemType`)
- [ ] T022 [US2] Thêm `HandleFashionProcessing` + dispatch nhánh `Processing` trong `HandleFashionEvent` tại `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/fashion_event_uc.go` + interface `.../interface/usecase/wardrobe_uc.go`; đảm bảo idempotent (không đổi DB nếu item không còn `Processing`; vẫn phát SSE nếu kênh mở) — FR-013
- [ ] T023 [US2] Thêm `HandleFashionProcessing` + dispatch nhánh `Processing` cho brand tại `services/commerce/internal/modules/brand/application/usecase/item/fashion_event_uc.go` + interface `.../interface/usecase/brand_item_uc.go`; đảm bảo idempotent (không đổi DB nếu item không còn trạng thái đang xử lý; vẫn phát SSE nếu kênh mở) — FR-013
- [ ] T024 [P] [US2] Unit test: SSE manager buffer/replay (gồm loại theo size + TTL, không xóa khi hết subscriber), dispatch `processing` → payload SSE đúng (wardrobe + brand), và idempotency khi nhận event `processing` trùng (không đổi DB dư thừa) — FR-013
- [ ] T025 [US2] Thêm annotation Swagger mô tả trạng thái `processing` trong handler wardrobe + brand (đều thuộc service commerce) và regenerate `make swagger svc=commerce`; KHÔNG chạy `swagger-combine` (dành cho T048)

**Checkpoint**: US2 độc lập hoạt động và kiểm thử được.

---

## Phase 5: User Story 3 - Thử phân tích lại an toàn & không mất chỉnh sửa thủ công (Priority: P2)

**Goal**: Retry chỉ khi trạng thái hợp lệ, trả `taskId`, bảo toàn danh mục hiện có.

**Independent Test**: Retry khi đang xử lý/đã dùng được → từ chối; sửa danh mục rồi retry → danh mục giữ nguyên; response có `taskId`.

- [ ] T026 [US3] Thêm `PreserveCategory bool` vào `FashionAnalyzeJobDTO` trong `services/commerce/internal/modules/fashion/application/dto/fashion.go`
- [ ] T027 [US3] Cập nhật `RetryWardrobeAnalysis` trong `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go`: giữ guard trạng thái + claim nguyên tử, sinh `taskId` mới, set `PreserveCategory` khi fashion item đã có category, trả `taskId` trong response
- [ ] T028 [US3] Bỏ ghi đè `category_id` khi `PreserveCategory` && đã có category trong `buildCompletedItemUpdate` (`services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go`)
- [ ] T029 [US3] Thêm field `TaskID *string` (`json:"taskId,omitempty"`) vào `WardrobeItemRes` (`services/commerce/internal/modules/wardrobe/application/dto/item.go`) và populate tại `services/commerce/internal/modules/wardrobe/application/mapper/wardrobe.go` (mapper DTO phải nằm trong `application/mapper`)
- [ ] T030 [P] [US3] Unit test retry: guard trạng thái (đang xử lý/không hợp lệ), preserve category, trả taskId
- [ ] T031 [US3] Regenerate Swagger cho `retry-analysis`: `make swagger svc=commerce`

**Checkpoint**: US3 độc lập hoạt động và kiểm thử được.

---

## Phase 6: User Story 4 - Nhân viên nhãn hàng xử lý sản phẩm lỗi/cần rà soát (Priority: P2)

**Goal**: Brand phân biệt `failed` vs `draft`, hiển thị lý do, chặn đổi khỏi `failed`, có retry-analysis.

**Independent Test**: Brand item lỗi → `status=failed` + reason; đổi trạng thái thủ công bị từ chối; retry hoạt động và trả `taskId`.

- [ ] T032 [P] [US4] Thêm `Failed BrandItemStatus = "failed"` vào `services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus/brand_item_status.go`
- [ ] T033 [US4] Map `HandleFashionFailed` → `Failed` (thay vì `Draft`) trong `services/commerce/internal/modules/brand/application/usecase/item/fashion_event_uc.go`
- [ ] T034 [P] [US4] Mở rộng `FashionItemContractDTO` (`services/commerce/internal/modules/fashion/application/dto/contract_dto.go`) thêm `ProcessingErrorReason`, `ReviewReason`; đặt hàm map entity→DTO trong `services/commerce/internal/modules/fashion/application/mapper/` (tuân thủ Code Style: mapper trong `application/mapper`) và gọi từ `fashion_item_uc.go` (refactor `mapToFashionItemRes` nếu cần)
- [ ] T035 [P] [US4] Thêm `ProcessingErrorReason`, `ReviewReason` vào `BrandItemRes`/`BrandItemBriefRes` (`services/commerce/internal/modules/brand/application/dto/brand_item.go`) + map tại `services/commerce/internal/modules/brand/application/mapper/brand_item_mapper.go`
- [ ] T036 [US4] Guard `UpdateBrandItemStatus` (từ chối đặt thủ công `failed`; từ chối mọi thay đổi khi item đang `failed`) + lỗi `ErrBrandItemStatusLocked` trong `services/commerce/internal/modules/brand/application/errors/errors.go`; sửa tại `services/commerce/internal/modules/brand/application/usecase/item/crud_uc.go` (LƯU Ý: cùng file với T037 → chạy tuần tự, không song song)
- [ ] T037 [US4] Validate status + guard `failed` trong `UpdateBrandItem` (`services/commerce/internal/modules/brand/application/usecase/item/crud_uc.go`) (cùng file với T036 → chạy sau T036)
- [ ] T038 [US4] Clear `review_reason`/`processing_error_reason` khi thành công trong `completeProcessedItem` (`services/commerce/internal/modules/fashion/application/usecase/worker/analyze_uc.go`)
- [ ] T039 [US4] Implement `RetryBrandItemAnalysis(ctx, staffUserID, brandID, itemID) (*dto.BrandItemRes, error)` trong `services/commerce/internal/modules/brand/application/usecase/item/crud_uc.go` (hoặc file `retry_uc.go` mới cùng thư mục) + khai báo interface usecase; thêm method repo `ClaimManualAnalysisRetry(ctx, itemID, now) (*entities.BrandItem, bool, error)` vào `brand/domain/repositories/brand_item_interfaces.go` + impl trong `brand/infrastructure/persistence/brand_item_repo.go`; thêm error `ErrBrandRetryForbidden`, `ErrBrandRetryInProgress` trong `brand/application/errors/errors.go`; điều kiện: `failed`, hoặc `draft` + `reviewReason != nil`; sinh `taskId` mới, gọi `fashionContract.ResetProcessingState`, publish job analyze trong `uow.Execute`; trả `BrandItemRes` kèm `taskId`
- [ ] T040 [US4] Thêm handler + route `POST /brand-portal/brands/:brandId/items/:itemId/retry-analysis` trong `services/commerce/internal/modules/brand/presentation/handler/brand_portal_item_handler.go` + `.../presentation/router/router.go`
- [ ] T041 [P] [US4] Unit test: brand failed mapping, guard status, điều kiện retry (`failed` / `draft`+reason / draft thường bị từ chối)
- [ ] T042 [US4] Thêm annotation Swagger cho endpoint retry brand (trong `brand_portal_item_handler.go`) và regenerate `make swagger svc=commerce` (brand portal thuộc service commerce; `swagger-combine` để ở T048)

**Checkpoint**: US4 độc lập hoạt động và kiểm thử được.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Tài liệu, tài liệu frontend, kiểm chứng toàn cục.

- [ ] T043 [P] Cập nhật API docs: `docs/api/wardrobe/constants.md` (thêm `processing`), `docs/api/wardrobe/wardrobe-api.md` (SSE `processing`, replay, confirm-review, retry `taskId`), `docs/api/brand/brand-portal-api.md` + `docs/api/brand/constants.md` (brand `failed` + reason + retry), `docs/api/fashion/constants.md`
- [ ] T044 [P] Cập nhật `docs/system-design/event-driven-design.md` (topic `fashion.event.processing`, số queue/binding) và `docs/system-design/core-feature-sequences.md` (luồng realtime + replay)
- [ ] T045 Cập nhật `specs/023-analyze-status-handling/frontend-guide.md` bằng nội dung thật từ code/contracts: hoàn thiện mục 3 (bảng API "trước → sau" + ví dụ request/response), mục 4 (payload SSE mẫu cho từng trạng thái + reconnect/replay/taskId), mục 5 (mã lỗi & thông báo thật), mục 6 (checklist), gỡ bỏ mọi placeholder "sẽ chốt khi triển khai"
- [ ] T046 Kiểm tra & cập nhật lần cuối `specs/023-analyze-status-handling/frontend-guide.md` (BẮT BUỘC — sau khi mọi task triển khai xong): đọc thuần từ góc nhìn FE, đối chiếu **từng** FR/SC, từng user story (US1–US4), từng contract (`contracts/api-contracts.md`, `event-contracts.md`, `realtime-transport.contract.md`) và từng kịch bản `quickstart.md`; bảo đảm không bỏ sót status/endpoint/field/lỗi/event nào; diễn đạt rõ ràng, không mơ hồ; xác nhận FE triển khai được **chỉ với tài liệu này, không cần đọc code backend hay Swagger**; bổ sung sửa các thiếu sót phát hiện và ghi changelog
- [ ] T047 Chạy `make wire && make lint && make test-race` và kiểm chứng theo `specs/023-analyze-status-handling/quickstart.md` (6 kịch bản)
- [ ] T048 [P] Rà soát không còn TODO/placeholder; xác nhận migration chỉ còn `add_brand_item_failed_status`; chạy `make swagger-combine` để cập nhật bundle combined (nếu baseline yêu cầu)
- [ ] T049 [P] Xác minh tương thích ngược (FR-015/SC-008): thêm test đảm bảo client cũ không vỡ khi nhận trạng thái `processing` mới ở SSE và khi `taskId` là optional trong `WardrobeItemRes`; xác nhận `failed` của brand là giá trị mới additive (không phá payload cũ)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Không phụ thuộc — bắt đầu ngay.
- **Foundational (Phase 2)**: Phụ thuộc Setup; chặn US2 và US4. US1 chỉ cần T001 (nếu chạm brand) — không cần transport.
- **User Stories (Phase 3–6)**:
  - US1: sau T001, độc lập.
  - US2: sau Phase 2.
  - US3: **độc lập về logic** — T027 trả `taskId` không phụ thuộc US2. SSE transport (Phase 2) chỉ cần để client *tiêu thụ* sự kiện realtime; US3 vẫn kiểm thử độc lập qua response `taskId` + trạng thái item.
  - US4: sau Phase 2 (SSE transport) + T001; retry dùng chung hạ tầng với US3 (T027) nhưng độc lập nghiệp vụ brand.
- **Polish (Phase 7)**: sau tất cả story. **T046 phụ thuộc toàn bộ T001–T045.** T049 độc lập, có thể chạy song song sau khi US1–US4 xong.

### User Story Dependencies

- **US1 (P1)**: độc lập (chỉ cần migration nếu đụng brand — không).
- **US2 (P1)**: độc lập sau Foundational.
- **US3 (P2)**: độc lập về logic (không phụ thuộc US2). SSE transport chỉ cần cho trải nghiệm realtime phía client.
- **US4 (P2)**: độc lập sau Foundational + migration T001.

### Within Each User Story

- Models/constants → services → handlers/router → tests → swagger.
- Story hoàn tất và kiểm thử trước khi sang story kế.

### Parallel Opportunities

- Setup: T001, T002, T003 song song.
- Foundational: T004, T005, T006 song song; T007 phụ thuộc T002; T008 sau T004–T007; T009 sau T008; T010 sau T008.
- US1: T012, T014, T016 song song với T011/T013/T015.
- US2: T018, T019 song song.
- US4: T032, T034, T035 song song.
- Polish: T043, T044, T048, T049 song song.

---

## Parallel Example: User Story 1

```bash
# Song song (khác file, không phụ thuộc):
Task: "T012 [P] [US1] Thêm IsWearableCategory vào IFashionContract + impl"
Task: "T014 [P] [US1] Thêm ErrConfirmReviewForbidden/ErrConfirmReviewCategoryRequired"
Task: "T016 [P] [US1] Unit test confirm-review"

# Tuần tự sau đó:
Task: "T011 [US1] Repo ConfirmWardrobeReview"
Task: "T013 [US1] UC ConfirmWardrobeItemReview (depends T011, T012)"
Task: "T015 [US1] Handler + route (depends T013, T014)"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1: Setup (T001–T003).
2. Phase 3: US1 (T011–T017) — có thể làm ngay sau T001.
3. **STOP & VALIDATE**: chạy kịch bản G1 trong quickstart.
4. Demo nếu sẵn sàng.

### Incremental Delivery

1. Setup + Foundational → transport sẵn sàng.
2. US1 → validate → MVP.
3. US2 → validate → realtime đầy đủ.
4. US3 → validate → retry an toàn.
5. US4 → validate → parity brand.
6. Polish → cập nhật docs, **hoàn thiện frontend-guide (T045–T046)**, kiểm chứng (T047).

### Parallel Team Strategy

- Dev A: US1; Dev B: US2 (sau Foundational); Dev C: US4 (sau Foundational + T001).
- US3 có thể chạy song song với US2 (logic retry độc lập); chỉ cần Phase 2 cho trải nghiệm realtime.

---

## Notes

- `[P]` = khác file, không phụ thuộc task chưa xong.
- `[Story]` map tới user story để truy vết.
- Mỗi story độc lập, có checkpoint kiểm thử riêng.
- **T046 là cổng bắt buộc cuối cùng** cho `frontend-guide.md`: không được coi feature hoàn tất nếu tài liệu FE còn thiếu/mơ hồ.
- Commit theo từng task/nhóm logic; tham chiếu `docs/development/git-workflow.md`.
- Tránh task mơ hồ; không sửa cùng file ở nhiều task song song.
- Regenerate Swagger (`make swagger svc=commerce`) lặp theo từng story là **chủ đích** (incremental delivery); T048 chỉ chạy `make swagger-combine` để cập nhật bundle combined.
