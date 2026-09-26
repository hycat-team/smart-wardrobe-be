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

## 1. Kiểm chứng G1 — confirm-review (US1)

**Điều kiện**: có một wardrobe item `status = needsReview`, có fashion item với category hợp lệ (vd slug `ao`).

1. Đặt category hợp lệ qua `PATCH /api/v1/wardrobe-items/{id}` (`categoryId`).
2. Gọi `POST /api/v1/wardrobe-items/{id}/confirm-review`.

**Kỳ vọng**:
- 200, `status = inWardrobe`.
- Item xuất hiện trong kết quả gợi ý phối đồ (`POST /api/v1/ai/outfit-recommendations`) khi đủ bộ.
- Gọi lại confirm khi item không còn `needsReview` → 403.
- Item `needsReview` chưa có category hợp lệ → từ chối (yêu cầu chọn danh mục).

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

## 7. Kiểm chứng tài liệu frontend (FR-017/SC-010)

- Mở `specs/023-analyze-status-handling/frontend-guide.md`.
- Xác nhận có đủ: kịch bản từng trạng thái, bảng API "trước → sau" + ví dụ, luồng SSE/replay/taskId, checklist.
- Thử một lập trình viên FE (hoặc tự đối chiếu) thực hiện tích hợp chỉ dựa vào tài liệu.

---

## 8. Kiểm thử tự động (gợi ý)

- Unit: UC confirm-review (ownership/status/category), retry (status guard/preserve), brand status guard, SSE manager buffer/replay, `IsWearableCategory`.
- Consumer: dispatch `processing` → SSE payload đúng.
- Integration (env-gated): routing key `fashion.event.processing.*` bind đúng queue; Redis pub/sub relay tới subscriber.
- Lint/boundaries: `make lint`, kiểm tra không import chéo module.

## 9. Tiêu chí hoàn tất nhanh

- `make wire && make lint && make test-race` PASS.
- Cả 6 kịch bản mục 1–6 đạt kỳ vọng.
- Swagger hiển thị endpoint mới; `frontend-guide.md` hoàn chỉnh.
