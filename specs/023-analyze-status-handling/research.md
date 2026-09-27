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

---

## R11. Phân biệt "không lọc" vs "lọc nhưng rỗng" (G8)

**Bối cảnh**: Commit `7b1c046a` ("eliminate cross-module DB access via contracts") thay việc lọc theo danh mục từ JOIN `categories.slug` bằng cách resolve danh mục → `[]fashionItemIDs` qua `IFashionContract.SearchFashionItemIDs` rồi lọc `fashion_item_id IN (...)`. Tầng repository dùng `if len(fashionItemIDs) > 0` nên khi danh mục/từ khóa không khớp fashion item nào (slice rỗng) thì bỏ qua bộ lọc ⇒ trả về **toàn bộ** món (hồi quy). Ba đường bị ảnh hưởng: `GetWardrobeItems`, `GetSystemCatalogWardrobeItems` (fallback DB search) và `catalog_uc.go`.

**Decision**:
- Thêm tham số boolean tường minh `filterByFashionItems bool` vào `buildUserWardrobeFilterQuery` / `buildWardrobeItemSearchQuery` và các method public tương ứng (`CountByUserIDAndFilters`, `GetByUserIDAndFiltersPaginated`, `CountItems`, `GetItemsPaginated`, `GetItems`).
- Khi `filterByFashionItems == true`: luôn áp `fashion_item_id IN ?` (kể cả slice rỗng ⇒ 0 kết quả).
- Khi `filterByFashionItems == false`: bỏ hoàn toàn bộ lọc ⇒ trả tất cả món của người dùng, kể cả `fashion_item_id` NULL.
- Usecase xác định cờ: `GetWardrobeItems` ⇒ `categorySlug != nil`; catalog/search ⇒ `queryStr != nil || categorySlug != nil`.
- Short-circuit: khi cờ bật nhưng danh sách rỗng, usecase trả kết quả rỗng ngay, KHÔNG truy vấn DB (tránh SQL `IN ()` vô nghĩa).

**Rationale**: Đặt "ý định lọc" tường minh ở lớp gọi thay vì suy diễn từ độ dài mảng; giữ nguyên ranh giới module; sửa đúng gốc tại 1 điểm (repository) cho cả 3 đường.

**Alternatives considered**:
- Dùng con trỏ `*[]uuid.UUID` (nil = không lọc, rỗng = lọc rỗng): cũng đúng nhưng dễ nhầm nil/rỗng ở call site; boolean tường minh dễ đọc hơn → chọn boolean.
- Chỉ short-circuit ở usecase, không đổi chữ ký repo: vá được 3 call site hiện tại nhưng để lại anti-pattern ở tầng repository, dễ tái phát → loại.
- Để SQL `IN ()` xử lý: tùy biến thể DB/GORM, kém rõ ràng và vẫn truy vấn DB vô ích → loại.

**Liên quan**: `GetByUserID(ctx, userID, nil)` (dùng cho locked-map, outfit…) luôn truyền danh sách rỗng với ý nghĩa "không lọc"; giữ nguyên hành vi "không lọc" (không truyền cờ, hoặc cờ false).

---

## R12. Realtime best-effort + observability (G9)

**Bối cảnh**: Realtime dùng Redis Pub/Sub làm last-mile (R1). Redis có thể tạm thời không khả dụng. Nếu lỗi publish/relay làm fail nghiệp vụ thì upload/confirm/retry hỏng oan, dù DB mới là nguồn sự thật. Đồng thời, best-effort nếu không có tín hiệu quan sát sẽ khiến mất sự kiện âm thầm, khó vận hành.

**Decision**:
- **Best-effort publish/relay**: `IRealtimePublisher.Publish` và relay xử lý lỗi Redis bằng cách **log `warn` kèm `taskId` rồi trả về** (không propagate error lên usecase/worker). Write side-effect nghiệp vụ vẫn đi qua outbox/RabbitMQ (không đổi).
- **Observability**: thêm counters/gauges in-memory trong package `shared/application/sse` (`realtime_metrics.go`) theo đúng pattern `shared/infrastructure/messaging/metrics.go` (`atomic.Int64`):
  - `realtime_publish_total`, `realtime_publish_failed`
  - `realtime_relay_total`, `realtime_relay_failed`
  - gauge `realtime_subscribers` (tổng subscriber đang mở, cập nhật khi Subscribe/Unsubscribe — tăng/giảm atomic).
  - Hàm `GetRealtimeMetrics() map[string]int64` phơi qua payload health `GET /api/v1/commerce/health` (thêm khóa `"realtime"`), cạnh `"elasticsearch"` hiện có.
- **Client fallback**: client tự refetch/polling khi không nhận được sự kiện (đã ghi trong spec FR-025). Không thêm cơ chế retry/buffer phía server cho last-mile.

**Rationale**: SSE là kênh tiện ích, không phải nguồn sự thật ⇒ cô lập lỗi để không ảnh hưởng nghiệp vụ. Metrics in-memory tận dụng hạ tầng health sẵn có, không thêm Prometheus/OTel (tránh dependency & chi phí). Đủ để đo tỉ lệ thất bại phục vụ SC-013/SC-014.

**Alternatives considered**:
- Retry + buffer sự kiện chờ Redis hồi phục: tăng phức tạp, độ trễ, và Redis down thường kéo theo cache hỏng — không đáng → loại.
- Coi Redis là bắt buộc, lỗi publish trả lỗi: có thể làm fail upload/retry → loại.
- Prometheus/OpenTelemetry: thêm dependency/hạ tầng, vượt nhu cầu hiện tại → loại (hoãn).
- Distributed tracing xuyên tiến trình: không cần để đạt SC-013/SC-014 → loại.

---

## R13. Phân loại ảnh không hợp lệ + mã hóa lý do + brand exhausted (G10)

**Bối cảnh**: `needs_review` hiện được kích hoạt bởi `is_single_item=false` (`analyze_uc.go:52`), gộp chung "ảnh đa món/toàn thân" (không sửa được) với "một món nhưng chưa chắc category". Hệ quả: ảnh xấu vẫn confirm được → lọt vào tủ đồ/phối đồ; retry cùng ảnh vô nghĩa (deterministic) và tốn AI cost. Lý do đang là câu thông báo tiếng Việt. Brand không nhận `processing-exhausted` (topology chỉ bind wardrobe + ES) nên có thể kẹt `draft` với `reviewReason=null`.

**Decision**:
- **Tách tín hiệu AI**: thêm `category_confident` (bool) bên cạnh `is_single_item`. `is_single_item` = ảnh có đúng một món; `category_confident` chỉ có nghĩa khi `is_single_item=true`.
- **Định tuyến**: `!is_single_item` ⇒ `failed` + lý do `multiple_items_detected`/`full_body_outfit_detected`; `is_single_item && !category_confident` ⇒ `needs_review` + `uncertain_category`; còn lại ⇒ `completed`.
- **Mã hóa lý do**: catalog mã cố định (review: `uncertain_category`; error: `multiple_items_detected`, `full_body_outfit_detected`, `analysis_temporary_error`, `auto_retry_exceeded`); SSE `error` cũng là mã; hằng số dùng chung trong `fashion/domain/constants/analysisreason`.
- **Gate**: confirm chỉ cho `uncertain_category`; retry chặn với ảnh không hợp lệ, chỉ cho lỗi tạm thời/hết retry hoặc `needs_review`.
- **Vị trí hiển thị**: reason thuộc fashion item, chỉ nested trong `fashionItem`; bỏ field root ở `BrandItemRes`/`BrandItemBriefRes`; mở rộng `FashionItemBriefRes` + `FashionItemRes`.
- **camelCase**: `FashionItemContractDTO` (+ `CategoryContractDTO`) serialize camelCase.
- **Brand exhausted (Option 1)**: quét stale theo `fashion_items` (fashion sở hữu processing state), map owner qua contract (wardrobe + brand), payload `processing-exhausted` thêm `ItemType`/`ItemID`, thêm queue + worker brand ⇒ brand item `failed` + lý do.

**Rationale**: Ảnh không hợp lệ là lỗi dữ liệu, không phải lỗi hệ thống → `failed` + mã lý do, chặn confirm/retry; giữ `needs_review` cho trường hợp thực sự fixable. Mã lý do ổn định cho FE i18n. Fashion sở hữu trạng thái xử lý nên quét tập trung, tránh lặp logic wardrobe/brand.

**Alternatives considered**:
- Đổi hết `needs_review` → `failed`: mất use-case "một món nhưng chưa chắc category" và trộn ngữ nghĩa lỗi hệ thống vs lỗi dữ liệu → loại.
- Thêm status mới `invalid`/`rejected`: semantic sạch nhưng cần migration cho brand + FE xử lý thêm → loại (dùng `failed` + reason).
- Chỉ brand-only recovery (không refactor scan): lặp logic và khó phân biệt "draft chờ duyệt" vs "draft đang xử lý" → loại.
- Giữ lý do là message tiếng Việt: FE khó i18n và so khớp chuỗi → loại.

**Ngoài phạm vi** (không làm đợt này): ẩn mặc định ảnh không hợp lệ khỏi closet; flow "thay ảnh" (replace image).

**Clarifications bổ sung (2026-09-27)**:
- `category_confident` thiếu/null + `is_single_item=true` ⇒ mặc định **false** ⇒ `needs_review` (không tự động completed).
- Mâu thuẫn `is_single_item=false` + `review_reason=uncertain_category` ⇒ ưu tiên tính hợp lệ ảnh ⇒ `failed` + `multiple_items_detected`.
- Thử lại thất bại ở bước publish job ⇒ ghi `processingErrorReason=analysis_temporary_error` (qua fashion item) cùng lúc set `failed` (FR-036).

---

## R14. Bỏ confirm-review, gộp vào retry kèm danh mục (G11)

**Bối cảnh**: Với G10 (không lưu metadata ở `needs_review`), `confirm-review` chỉ set danh mục + trạng thái mà fashion item **rỗng** metadata/embedding ⇒ món vào sử dụng nghèo dữ liệu cho gợi ý phối đồ. Đồng thời `job.CategoryID` hiện **không** được đưa vào prompt phân tích ⇒ nếu chỉ truyền danh mục vào job, AI vẫn có thể trả `category_confident=false` ⇒ **vòng lặp `needs_review`**.

**Decision**:
- **Bỏ hẳn `confirm-review`** (route/handler/UC/interface/repo claim confirm/errors/tests) — breaking, ghi rõ trong tài liệu frontend.
- `retry-analysis` nhận `categoryId` (wardrobe + brand):
  - `NeedsReview` (`uncertain_category`) ⇒ **bắt buộc** danh mục hợp lệ (từ request hoặc danh mục có sẵn); validate `IsWearableCategory`; persist qua `UpdateFashionItemCategory`.
  - `Failed` (transient) ⇒ danh mục tùy chọn.
  - Vẫn chặn retry với ảnh không hợp lệ.
- **Chế độ "danh mục cố định"**: thêm cờ `FixedCategory` vào `FashionAnalyzeJobDTO`; worker dùng **prompt metadata-only** theo danh mục đã biết, **bỏ gate** `is_single_item`/`category_confident`, `PreserveCategory=true` ⇒ chỉ sinh metadata + embedding, giữ danh mục ⇒ hết vòng lặp.
- **Prompt fail-fast**: initial analyze trả JSON tối giản khi ảnh không hợp lệ / `category_confident=false`; giữ **1 call không `ResponseSchema`** (schema strict buộc đủ field ⇒ vô hiệu fail-fast).
- Brand nhận `categoryId` qua chính retry request (chưa có API set category khác).

**Rationale**: Một đường duy nhất (retry) vừa đảm bảo metadata đầy đủ, vừa tránh endpoint rỗng nghĩa; chế độ danh mục cố định loại bỏ vòng lặp; fail-fast giảm output tokens.

**Alternatives considered**:
- Giữ confirm nhưng cho chạy AI (async): trùng lặp retry → gộp cho gọn → loại.
- Confirm sync + enrichment async: cửa sổ "usable nhưng thiếu metadata" → loại.
- Chỉ truyền `CategoryID` mà không đổi prompt: vòng lặp `needs_review` → loại.
- Bật `ResponseSchema` cho analyze: vô hiệu fail-fast → loại.
- Thêm `CategoryID` vào `UpdateBrandItemReq` thay vì retry request: thừa endpoint, brand chọn luôn ở retry → loại.

**Ngoài phạm vi**: không giữ alias deprecated cho confirm.
