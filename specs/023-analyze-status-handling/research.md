# Phase 0 Research: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Date**: 2026-09-26

Tài liệu này ghi lại các quyết định kỹ thuật (Decision / Rationale / Alternatives) để giải quyết mọi điểm chưa rõ trong Technical Context.

---

## R1. Truyền sự kiện realtime từ worker → http (last-mile)

**Bối cảnh**: `ISSEManager` (`shared/application/sse/sse_manager.go`) là map in-memory theo tiến trình. Consumer phát SSE chạy ở `commerce-worker`; handler SSE subscribe chạy ở `commerce-http`. Ở VPS `APP_MODE=all` cả hai cùng tiến trình; ở AWS tách riêng → sự kiện không tới client.

**Decision**: Thêm lớp transport **Redis Pub/Sub** làm chặng cuối:
- Interface mới `sse.IRealtimePublisher` (`Publish(ctx, taskId, data) error`) + `sse.IRealtimeRelay` (`Start(ctx)/Stop()`).
- Impl publisher publish lên channel `sse:task:{taskId}`; relay subscribe pattern `sse:task:*` và forward vào `ISSEManager.Publish` cục bộ (fan-out + buffer).
- Consumer (worker) dùng `IRealtimePublisher`; handler (http) dùng `ISSEManager`; relay chạy ở tiến trình http (mode `http|all`), bỏ qua `worker`.
- Publisher và relay đặt sau interface để có thể thay transport/kho chia sẻ sau này.

**Rationale**: Redis đã có sẵn ở mọi môi trường; Pub/Sub phù hợp tín hiệu tạm thời, độ trễ thấp, broadcast; hoạt động đồng nhất ở cả hai topology. Không vi phạm outbox vì SSE không phải write side-effect.

**Alternatives considered**:
- In-memory thuần: hỏng ở AWS tách tiến trình (đã xác minh).
- RabbitMQ fanout + queue mỗi http instance: phải chạy consumer ở `APP_MODE=http` (phá mô hình mode), bookkeeping queue phức tạp cho tín hiệu tạm thời → loại.
- Chạy consumer fashion trong http: phá tách biệt mode, tăng tải http → loại.

---

## R2. Sự kiện `processing` (G2) — thêm mới hay tận dụng hạ tầng có sẵn

**Decision**: Tận dụng **queue analyzed hiện có** thay vì tạo queue/worker/DLQ mới:
- Thêm `api.taskstatus.Processing = "processing"`; thêm routing keys `fashion.event.processing.wardrobe|.brand` và **bind thêm vào `QueueWardrobeFashionAnalyzed` / `QueueBrandFashionAnalyzed`** (như cách `analyzed` + `failed` đang dùng chung queue — `topology.go` L87-99).
- Tái dùng payload `FashionAnalyzedEventPayload` với `Status = processing` (các field `FashionBrief/Error` để trống).
- Consumer hiện có (`HandleFashionEvent`) dispatch thêm nhánh `Processing → HandleFashionProcessing` (chỉ phát SSE `processing`).
- Producer: `analyze_uc.go` `ProcessBackgroundBatchUploadJob` publish event processing **ngay đầu** (trong `uow.Execute`), trước khi gọi AI, dùng switch `job.ItemType == "brand"`.

**Rationale**: Ít thay đổi nhất; tương thích với pattern "analyzed + failed share queue"; topic `fashion.event.processing` không đụng `fashion.event.processing-exhausted`.

**Alternatives considered**: Queue/worker riêng cho processing → thêm 2 queue + 2 DLQ + 2 worker + wire; không cần thiết vì cùng xử lý "kết quả phân tích". Naming wildcard `fashion.event.processing.#` → loại (rủi ro đụng processing-exhausted).

---

## R3. Xác nhận `needs_review` — kiểm tra danh mục "hợp lệ" (G1)

**Decision**: Thêm method contract thuần `IFashionContract.IsWearableCategory(ctx, categoryID uuid.UUID) (bool, error)` (fashion đánh giá qua ánh xạ slug→wearing role; `other`/không có → không hợp lệ). Wardrobe UC `ConfirmWardrobeItemReview`:
1. Load item + kiểm tra ownership + status `NeedsReview`.
2. Lấy fashion item qua `GetFashionItem`; yêu cầu category != nil và `IsWearableCategory == true`.
3. Repo `ConfirmWardrobeReview` chuyển `NeedsReview → InWardrobe` atomic (WHERE status=NeedsReview).
4. Publish `wardrobe.created` trong `uow.Execute` (index ES).

**Rationale**: Giữ module boundary — wardrobe không import mapper fashion (`CategorySlugToWearingRole`); fashion là nơi sở hữu tri thức danh mục. Contract method chỉ đọc repo fashion → hợp lệ theo Pure Contract Usecase.

**Alternatives considered**:
- Wardrobe tự map slug: vi phạm boundary (import mapper fashion) hoặc lặp tri thức → loại.
- Đưa `WearingRole` vào contract DTO: rò rỉ tri thức phối đồ ra ngoài, nặng hơn nhu cầu → loại.
- Kiểm tra category != nil là đủ: không đủ (slug `other` vẫn không phối được) → loại.

---

## R4. Kiểm soát thử lại (G3) — chỉ theo trạng thái

**Decision**: KHÔNG thêm ràng buộc số lần/thời gian và KHÔNG thêm cột. `retry-analysis` chỉ được chấp nhận khi status ∈ {`Failed`, `NeedsReview`}; dùng claim nguyên tử (`WHERE status IN (...) → Processing`) để chống tạo phiên trùng khi có yêu cầu đồng thời.

**Rationale**: Bản thân trạng thái đã đóng vai trò guard: sau khi claim, status chuyển `Processing` nên mọi lần gọi lại bị từ chối cho tới khi có kết quả mới. Điều này loại bỏ được tốt nhất 2 cột hiếm dùng, config và error phụ trợ, giảm migration và độ phức tạp. Chi phí AI (nếu có chu trình fail→retry nhanh) được kiểm soát bởi chính sách quota AI hiện hữu (constitution §VI).

**Alternatives considered**:
- Cooldown 15s + max count (đã cân nhắc ban đầu): cần 2 cột + config + 2 error + test, giá trị thấp so với guard trạng thái → loại.
- Dùng `fashion_items.processing_retry_count`: là ngân sách retry **tự động**, bị `ResetProcessingState` reset và dùng để gate "bỏ cuộc" (`cleanup_uc.go:132`); trộn ngữ nghĩa → loại.

**Lịch sử**: FR-007 ban đầu yêu cầu cooldown + giới hạn; đã gỡ bỏ theo quyết định ngày 2026-09-26 (xem `spec.md` Clarifications).

---

## R5. Giữ danh mục người dùng sửa khi phân tích lại (G4)

**Decision**: Thêm `PreserveCategory bool` vào `FashionAnalyzeJobDTO`. `RetryWardrobeAnalysis` set `true` khi fashion item đã có category. `buildCompletedItemUpdate` bỏ `category_id` khỏi `updateMap` khi `PreserveCategory` && đã có category (vẫn cập nhật metadata/embedding khác).

**Rationale**: Bảo toàn danh mục hiện có đảm bảo không bao giờ ghi đè lựa chọn của người dùng (thỏa FR-008); nếu chưa có danh mục thì AI điền bình thường. Đơn giản, không cần cột/cờ mới.

**Alternatives considered**:
- Thêm cờ `categoryManuallySet` trên fashion item (chính xác nhất): cần migration + set cờ trong đường PATCH → thêm phức tạp chưa cần thiết → loại.
- Chỉ bảo toàn khi món ở `needs_review`: bỏ sót trường hợp user sửa danh mục rồi món `failed` (sẽ bị AI ghi đè) → loại.

**Xác nhận**: Quyết định B (bảo toàn khi đã có danh mục) đã được chốt ngày 2026-09-26 (xem `spec.md` Clarifications).

---

## R6. Brand: trạng thái `failed` + lý do (G5)

**Decision**:
- Thêm `branditemstatus.Failed = "failed"`.
- **Migration**: drop/re-add `chk_brand_items_status` để bao gồm `'failed'` (constraint hiện chỉ `draft|active|archived`).
- `HandleFashionFailed` set `Failed` (thay vì `Draft`); `needs_review` vẫn → `Draft`.
- `UpdateBrandItemStatus`: cho phép chuyển từ `Failed → Draft/Active`? **Không** — theo quyết định spec, `failed` là trạng thái cuối với thao tác thủ công: từ chối mọi thay đổi rời khỏi `failed` (kể cả `Failed→Draft/Active`), đồng thời từ chối đặt thủ công `failed`. Thêm error `ErrBrandItemStatusLocked`.
- `UpdateBrandItem` (PUT): bổ sung validate status (hiện cast không kiểm tra — bug) và chặn thay đổi khi item đang `failed`.
- **Lý do**: mở rộng `FashionItemContractDTO` thêm `ProcessingErrorReason`, `ReviewReason`; populate ở mapper fashion; thêm `processingErrorReason`/`reviewReason` vào `BrandItemRes` + map.
- **Dọn reason khi thành công**: bổ sung `review_reason: nil`, `processing_error_reason: nil` vào updateMap của `completeProcessedItem` (hiện không clear → lý do cũ còn sót).

**Rationale**: Tách `failed` (lỗi hệ thống) khỏi `draft` (cần duyệt); lý do lấy từ `fashion_items` qua contract (brand không có `fashionItemRepo`).

**Alternatives considered**:
- Thêm cột `status_reason` trên `brand_items`: trùng lặp dữ liệu đã có ở `fashion_items` → loại.
- Thêm trạng thái `needs_review` riêng cho brand: spec chọn chỉ thêm `failed`, giữ `draft` cho cần duyệt → loại.
- Cho nhân viên đổi khỏi `failed`: trái quyết định spec → loại.

---

## R7. Brand retry-analysis (G6)

**Decision**: Endpoint `POST /brand-portal/brands/{brandId}/items/{itemId}/retry-analysis` (nhóm `portalMember`, Owner+Staff). UC `RetryBrandItemAnalysis`:
1. Kiểm tra brand sở hữu item + status ∈ {`Failed`, `Draft` khi review} — chỉ cho phép khi đang lỗi/cần rà soát.
2. Kiểm tra trạng thái cho phép retry (không dùng count/cooldown — xem R4).
3. Reset processing state fashion (`fashionContract.ResetProcessingState`).
4. Sinh `taskID` mới, claim status → `Draft` (đang xử lý), publish lại job analyze (`TopicFashionAnalyze`) với `PreserveCategory` nếu cần.
5. Trả `BrandItemRes` + `taskId` để subscribe SSE.

**Rationale**: Đối xứng với wardrobe; tận dụng `FashionItemContractDTO` (đã có `ImageUrl`, `Category`). Brand item đã có `FashionItemID` sẵn nên không cần luồng bind.

**Alternatives considered**: Dùng `TopicFashionInit` (như wardrobe): init flow dành cho item chưa có fashion record; brand đã có → dùng `TopicFashionAnalyze` phù hợp hơn.

---

## R8. Buffer phát lại SSE (G7)

**Decision**: Mở rộng `SSEManager` giữ ring buffer N event/taskId (config `server.sse.replay_buffer_size`, default 100); `Subscribe(taskId)` phát lại toàn bộ buffer hiện có cho subscriber mới trước khi stream tiếp. Vượt N → loại event cũ nhất; buffer cũng bị loại theo TTL (`server.sse.replay_buffer_ttl_seconds`, default 300). Buffer KHÔNG bị xóa ngay khi subscriber cuối cùng ngắt kết nối, để client reconnect muộn (kể cả sau khi phiên kết thúc) vẫn nhận được sự kiện.

**Rationale**: Giải quyết reconnect mà không cần refetch; in-memory đủ cho hiện tại; đặt sau interface để nâng lên kho chia sẻ sau.

**Alternatives considered**: `Last-Event-ID` đầy đủ (chuẩn SSE): phức tạp hơn, chưa cần → hoãn. Redis list buffer: tăng phụ thuộc, làm sau nếu cần đa instance.

---

## R9. Vị trí migration & ownership

**Decision**: Migration mới đặt tại `services/commerce/database/migrations/` (module `brand` thuộc Commerce), đặt tên `<ts>_add_brand_item_failed_status.sql`, **schema-qualify** bảng (`brand.brand_items`) vì Goose DSN không có `search_path`. Thực hiện qua `make migrate-create svc=commerce` / `make migrate-up svc=commerce`.

**Nội dung migration** (chỉ còn 1 thay đổi — không thêm cột; xem R4):
- Drop + re-add `chk_brand_items_status` bao gồm `'failed'`.

**Rationale**: Owning service của module là Commerce; migration lịch sử nằm ở Account chỉ do lịch sử (`docs/development/local-setup.md:315-323`). `wardrobe_items.status` **không** có CHECK nên không cần sửa cho status.

**Rủi ro cần xác nhận khi implement**: entity `BrandItem` tag `default:DRAFT` (chữ hoa) không khớp DB (lowercase, không default) — không dựa vào default tag; luôn set status tường minh.

**Alternatives considered**: Đặt migration ở `account`: trái quy tắc "owning service" cho thay đổi mới → loại.

---

## R10. Lifecycle relay ở HTTP process

**Decision**: Thêm component relay (Start/Stop) vào `App` và khởi động trong `App.Run()` cho mode `http|all`, dừng khi shutdown. Không đưa vào `bootstrap.Workers` (vì `Workers` chỉ chạy ở `worker|all`, sẽ không chạy ở http).

**Rationale**: Tôn trọng ràng buộc FR-044/SC-018 (`APP_MODE=http` không chạy consumers/outbox/cron); relay không phải consumer nghiệp vụ, chỉ là adapter hạ tầng nhẹ.

**Alternatives considered**: Thêm relay vào `Workers.All()`: sẽ không chạy ở http → hỏng mục tiêu. Thêm nhánh đặc biệt trong `startWorkers`: che khuất quy tắc mode → loại.
