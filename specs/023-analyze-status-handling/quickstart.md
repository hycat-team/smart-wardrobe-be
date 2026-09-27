# Quickstart: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Feature**: `023-analyze-status-handling`
**Date**: 2026-09-26

Hướng dẫn chạy và kiểm chứng end-to-end. Chi tiết hợp đồng xem [contracts/](./contracts/), dữ liệu xem [data-model.md](./data-model.md). Không chứa code triển khai.

---

## 0. Chuẩn bị

- Hạ tầng: PostgreSQL (pgvector), Redis, RabbitMQ — dùng `make dev` / docker compose local.
- Build & generate: `make wire` (sau khi thêm provider/lifecycle mới), Swagger regenerate.
- Chạy test: `make test-race`.
- Migration: `make migrate-create svc=commerce name=add_brand_item_failed_status` rồi `make migrate-up svc=commerce`.
- Biến môi trường mới (nếu cần): `SSE_REPLAY_BUFFER_SIZE`, `SSE_REPLAY_BUFFER_TTL_SECONDS`.

---

## 1. Kiểm chứng G1 — xử lý needs_review bằng retry kèm danh mục (US1)

**Điều kiện**: có một wardrobe item `status = needsReview` (`uncertain_category`).

1. Gọi `POST /api/v1/wardrobe-items/{id}/retry-analysis` với body `{ "categoryId": "<category hợp lệ>" }`.
2. Mở SSE `GET /api/v1/wardrobe-items/tasks/{taskId}/sse` (taskId từ response) và chờ event kết thúc.

**Kỳ vọng**:
- 200, `status = 3` (Processing) + có `taskId`.
- Khi AI xong: `status = 0` (inWardrobe) với **metadata đầy đủ** (`fashionItem` có color/style/…); danh mục đã chọn được giữ nguyên.
- Item xuất hiện trong kết quả gợi ý phối đồ (`POST /api/v1/ai/outfit-recommendations`) khi đủ bộ.
- Retry `needsReview` mà **thiếu/không hợp lệ** `categoryId` → từ chối (yêu cầu chọn danh mục).
- Endpoint cũ `POST /api/v1/wardrobe-items/{id}/confirm-review` → `404` (đã gỡ).

---

## 2. Kiểm chứng G2 + G7 — realtime `processing` + replay (US2)

1. `POST /api/v1/wardrobe-items/batch-upload` (nhiều ảnh) → nhận `taskId`.
2. Mở `GET /api/v1/wardrobe-items/tasks/{taskId}/sse`.
3. Quan sát event.

**Kỳ vọng**:
- Nhận event `status: "processing"` cho từng món, kèm `index/total`.
- Nhận lần lượt `completed`/`needs_review` (có `data`) hoặc `failed` (có `error`).
- **Replay**: ngắt kết nối giữa chừng rồi kết nối lại → nhận lại các event gần nhất (trong buffer ~100).
- **AWS topology** (nếu test được): `APP_MODE=http` + `APP_MODE=worker` tách container → event vẫn tới client nhờ Redis Pub/Sub. Kiểm tra key `sse:task:{taskId}` xuất hiện trên Redis khi worker publish.

---

## 3. Kiểm chứng G3 — guard trạng thái khi retry (US3)

1. Đưa item về `failed` (ảnh lỗi hoặc mock).
2. Gọi `POST /api/v1/wardrobe-items/{id}/retry-analysis` → item chuyển `processing`.
3. Gọi lại ngay khi item đang `processing`.
4. Thử gọi retry khi item đã `inWardrobe`.

**Kỳ vọng**:
- Lần 1: 200, `status = processing`, response có `taskId`.
- Lần 2 (đang xử lý): từ chối vì trạng thái không cho phép (không tạo phiên trùng).
- Retry khi đã sử dụng được: từ chối.

---

## 4. Kiểm chứng G4 — giữ danh mục thủ công (US3)

1. Với item `needsReview`, `PATCH` category sang giá trị người dùng chọn (vd `quan`).
2. Gọi `retry-analysis`; chờ AI xong.

**Kỳ vọng**: sau khi phân tích lại, `fashionItem.category.slug` vẫn là `quan` (không bị AI ghi đè).

---

## 5. Kiểm chứng G5 — brand `failed` + reason (US4)

1. Tạo brand item gây lỗi phân tích.
2. `GET /api/v1/brand-portal/brands/{brandId}/items/{itemId}`.

**Kỳ vọng**:
- `status = "failed"`, có `processingErrorReason`.
- `PATCH .../status` sang `active`/`draft` khi item đang `failed` → từ chối.
- Item AI không chắc → `status = "draft"` + `reviewReason`.

---

## 6. Kiểm chứng G6 — brand retry (US4)

1. Với item brand `failed`, gọi `POST /api/v1/brand-portal/brands/{brandId}/items/{itemId}/retry-analysis`.

**Kỳ vọng**:
- 200, trả `taskId` + item chuyển sang trạng thái đang xử lý.
- Subscribe SSE brand → nhận `processing` rồi `completed`/`failed`.
- Nhân viên brand khác (không sở hữu) → 403.

---

## 7. Kiểm chứng G8 — lọc danh mục rỗng (US5)

**Điều kiện**: người dùng có nhiều món; tồn tại một danh mục Z không có món nào thuộc người dùng (vd `mu_bao_hiem`), và một danh mục Y có món.

1. `GET /api/v1/wardrobe-items?categorySlug=Z`.
2. `GET /api/v1/wardrobe-items?categorySlug=Y`.
3. `GET /api/v1/wardrobe-items` (không truyền danh mục).
4. `GET /api/v1/wardrobe-items/catalog?query=<từ-khóa-không-khớp>` (fallback DB search với `ELASTICSEARCH_ENABLED=false`).
5. (Repo nội bộ) `GetByUserID` cho locked-map vẫn truyền "không lọc".

**Kỳ vọng**:
- (1) `data` rỗng, `total = 0` — KHÔNG trả toàn bộ tủ đồ.
- (2) chỉ trả các món thuộc danh mục Y.
- (3) trả tất cả món của người dùng, kể cả món `fashion_item_id` NULL.
- (4) kết quả rỗng, không trả toàn bộ catalog.
- Log/trace: case (1) và (4) không phát sinh truy vấn `IN (...)` ở repository (usecase short-circuit).

---

## 8. Kiểm chứng G9 — realtime best-effort + observability (FR-025/FR-026)

**Điều kiện**: hạ tầng chạy bình thường; có thể chủ động chặn Redis (dừng container Redis hoặc chặn cổng) để mô phỏng gián đoạn.

1. `GET /api/v1/commerce/health` → ghi lại khối `realtime` (counters/gauges ban đầu).
2. Bắt đầu một phiên `batch-upload` / mở SSE (`GET /api/v1/wardrobe-items/tasks/{taskId}/sse`) → gauge `realtime_subscribers` tăng.
3. Ngắt Redis (hoặc cấu hình sai endpoint Redis) rồi phát sinh sự kiện (upload/retry).
4. Kiểm tra log service: có `warn` kèm `taskId` cho lỗi publish/relay.
5. `GET /api/v1/commerce/health` → `realtime_publish_total`/`realtime_publish_failed` (hoặc relay) tăng.
6. Khôi phục Redis, phát sinh sự kiện mới → client nhận lại sự kiện bình thường.

**Kỳ vọng**:
- Khi Redis down: upload/retry vẫn trả thành công (200/202); item vẫn được tạo/cập nhật ở DB (SC-013).
- Mỗi lỗi publish/relay có ≥1 dòng log `warn` kèm `taskId` (SC-014).
- Counters phơi qua health đo được tỉ lệ thất bại; không có dependency/hạ tầng mới.

---

## 9. Kiểm chứng tài liệu frontend (FR-017/SC-010)

- Mở `specs/023-analyze-status-handling/frontend-guide.md`.
- Xác nhận có đủ: kịch bản từng trạng thái, bảng API "trước → sau" + ví dụ, luồng SSE/replay/taskId, checklist.
- Thử một lập trình viên FE (hoặc tự đối chiếu) thực hiện tích hợp chỉ dựa vào tài liệu.

---

## 10. Kiểm thử tự động (gợi ý)

- Unit: UC retry kèm danh mục (validate danh mục, chế độ danh mục cố định), retry (status guard/preserve), brand status guard, SSE manager buffer/replay, `IsWearableCategory`.
- Unit G8: `GetWardrobeItems`/`GetSystemCatalogWardrobeItems`/catalog — danh mục rỗng ⇒ 0; không lọc ⇒ toàn bộ; cờ `filterByFashionItems` truyền đúng xuống repo mock.
- Unit G9: publisher/relay trả nil (không lỗi) khi Redis lỗi + counter `*_failed` tăng; `GetRealtimeMetrics` phản ánh subscribe/unsubscribe.
- Consumer: dispatch `processing` → SSE payload đúng.
- Integration (env-gated): routing key `fashion.event.processing.*` bind đúng queue; Redis pub/sub relay tới subscriber.
- Lint/boundaries: `make lint`, kiểm tra không import chéo module.

## 11. Kiểm chứng G10 — phân loại ảnh & mã lý do (US6)

**Điều kiện**: dùng ảnh test: (a) ảnh nhiều món/toàn thân; (b) ảnh một món nhưng khó phân loại category.

1. Upload ảnh (a) → chờ realtime.
2. Upload ảnh (b) → chờ realtime.
3. `GET` món (wardrobe detail / brand item) → đọc `fashionItem.reviewReason` / `fashionItem.processingErrorReason`.
4. Gọi `retry-analysis` (không kèm category) với món từ (a) và món từ (b).
5. Ép brand item quá hạn + hết retry (mock thời gian/processing_retry_count) → chạy recovery worker.

**Kỳ vọng**:
- (a) ⇒ `failed` + reason `multiple_items_detected`/`full_body_outfit_detected`; `retry-analysis` bị từ chối (ảnh không hợp lệ).
- (b) ⇒ `needs_review` + `uncertain_category`; xử lý bằng `retry-analysis` kèm `categoryId` (xem mục 1).
- Reason **chỉ** nằm trong `fashionItem` (camelCase); không có ở root.
- SSE event `failed` có `error` là mã (không phải câu thông báo).
- Brand item quá hạn/hết retry ⇒ `failed` + `auto_retry_exceeded` (không kẹt `draft`).

---

## 12. Tiêu chí hoàn tất nhanh

- `make wire && make lint && make test-race` PASS.
- Cả kịch bản mục 1–11 đạt kỳ vọng (bao gồm G11 — retry kèm danh mục cho `needs_review`).
- Swagger hiển thị endpoint mới; `frontend-guide.md` hoàn chỉnh.
