# Quickstart Validation: Platform Migration & 3 Services

**Feature**: `020-platform-migration` | **Date**: 2026-09-15

Hướng dẫn kiểm chứng end-to-end cho feature. Đây là **run/verify guide**, không phải mã triển khai.

Tham chiếu: [spec.md](./spec.md) · [data-model.md](./data-model.md) · [contracts/](./contracts/inter-service-grpc.md) · ADR 021 (`docs/decisions/021-funded-platforms-and-repatriation.md`)

## Prerequisites

- Go 1.25+, Docker Desktop, GNU Make.
- `buf` CLI (cài qua `make tool`) cho sinh mã gRPC.
- Postgres + Redis + RabbitMQ local qua `deployments/docker-compose.yml`.
- Tài khoản funded (Cloudflare Pro, Supabase Pro, CloudAMQP Tiger, Redis Cloud Essentials) +
  trial GCP có billing cap cho scenario Cloud.
- Biến môi trường copy từ `.env.example`.

## VS1 — Elasticsearch tắt, hệ thống vẫn hoạt động (US1, SC-001, SC-007)

1. Cấu hình: `ELASTICSEARCH_ENABLED=false` (mặc định), trỏ `ELASTICSEARCH_ADDRESSES` rỗng.
2. Chạy compose local (3 service).
3. Gọi tìm kiếm trang phục hệ thống, ví dụ:
   `GET /api/v1/wardrobe/catalog/items?query=...`
4. **Kỳ vọng**: trả 200 + kết quả từ database; log có cảnh báo fallback, không có lỗi lặp.
5. Thêm/sửa/xóa một system catalog item.
6. **Kỳ vọng**: thao tác thành công; depth queue `elasticsearch_sync_queue` và `.dlq` **không tăng**.
7. Bật lại `ELASTICSEARCH_ENABLED=true` + địa chỉ hợp lệ, restart.
8. **Kỳ vọng**: initial sync chạy, dữ liệu được đánh index lại không cần thao tác tay.

## VS2 — Khởi động trên pooler, outbox vẫn phát (US2, SC-002)

1. Cấu hình DB runtime trỏ Supabase pooler (managed, transaction mode 6543); direct DSN trỏ 5432.
2. Tắt listener (`OUTBOX_LISTENER_ENABLED=false`) để mô phỏng `LISTEN` không khả dụng, chạy service.
3. **Kỳ vọng**: app khởi động thành công; log cảnh báo listener bị tắt/không khả dụng.
4. Tạo một bản ghi nghiệp vụ sinh event (ví dụ tạo wardrobe item).
5. **Kỳ vọng**: outbox worker phát event qua poll ticker trong vài giây; `outbox_pending` giảm về 0.
6. Đặt `DATABASE_MIGRATE_ON_START=false`, restart.
7. **Kỳ vọng**: app không tự chạy migration; chạy `make migration-up` (DSN session) thành công.

## VS3 — Build 3 binary service, monolith đã xóa (US5, SC-006)

```bash
go work sync
go build ./services/account/... ./services/commerce/... ./services/ai/...
```

**Kỳ vọng**: cả 3 build thành công; `cmd/server` không còn tồn tại trên `main`; import
implementation xuyên service fail ở compile-time. Trong CI, job build 3 binary bắt buộc pass
(FR-020). Tag `monolith-legacy` tồn tại và frozen.

## VS4 — Topology 3 service + JWT + origin-auth (US5, FR-019/FR-022..024, FR-028)

1. Chạy `account`, `commerce`, `ai` local với gRPC bật; gọi API liên service (ví dụ tạo outfit
   cần dữ liệu fashion + subscription quota).
   - **Kỳ vọng**: kết quả đúng; giao tiếp liên service qua gRPC; correlation ID xuyên suốt.
2. Gọi service thiếu `X-Gateway-Secret`.
   - **Kỳ vọng**: 401/403 (trừ `/api/v1/health` vẫn 200).
3. Verify JWT ES256 bằng public key cục bộ (không gọi mạng); blacklist qua Redis.
4. **Contract test**: bộ test hành vi contract (in-process + gRPC) — cả hai adapter pass.
5. **Generate-check**: `buf lint && buf generate` → không có diff.
6. **Swagger parity**: hợp docs 3 service đủ 9 section; thiếu section là fail.

## VS5 — Repatriation về VPS: 3 service + Nginx gateway (US4/US7, SC-004, SC-008)

1. Từ Supabase (môi trường Cloud) tạo dump; dựng VPS (hoặc local prod-like) theo
   `deployments/docker-compose.prod.yml` (3 service + postgres/redis/rabbitmq + nginx).
2. Restore dump; đặt profile `vps`: DB trỏ docker host, `sslmode=disable`, TLS tắt,
   `migrate_on_start=true`; Nginx gateway dùng route sinh từ single-source.
3. Trỏ DNS test về VPS; gọi ma trận host × route + SSE + auth + upload.
4. **Kỳ vọng**: hệ thống chạy đầy đủ; chỉ khác biệt cấu hình/DNS, không sửa code; đo thời gian
   cán đích ≤ 4h (SC-004).
5. Route parity Worker ↔ Nginx 100% (SC-012).

## VS6 — Cutover drill trên môi trường Cloud: NS-delegate → Worker → SaaS (US3, FR-027, SC-003, SC-009)

Thực hiện runbook trên môi trường Cloud:

1. NS-delegate `api-v2...` → zone Cloudflare Pro active → Worker route → 3 Cloud Run (môi trường Cloud).
2. Migrate dữ liệu VPS → Supabase (1 database dùng chung); drill dùng `database.drill_schema` rồi
   xoá; smoke host × route; test SSE dài (heartbeat 15s, chunk deadline 30s)
   (align timeout 3 tầng); test rate-limit (2 rule) + WAF managed.
3. Drill SaaS custom hostname + redirect `api-v2` → domain chính.
4. **Kỳ vọng**: cutover production-like hoàn tất; health + smoke pass; đo thời gian ≤ 2h.
5. Chạy reverse cutover (về VPS) trong cùng đợt drill; đo thời gian; ghi lại bước lệch vào runbook.

## VS7 — Kiểm tra chất lượng bắt buộc

```bash
make wire && make lint && make test-race
```

**Kỳ vọng**: pass. Đây là gate cho mọi commit (Governance, constitution 3.0.0).

## VS8 — Teardown trial + funding clock (US6/US7, SC-010, SC-011)

1. Chạy runbook teardown trên môi trường Cloud: xóa service Cloud Run trial, verify không còn resource
   tính phí; đối chiếu billing trial về 0 (SC-010).
2. Drill funding clock T-30/14/7/1 trên giấy + restore drill thật 1 lần: backup Supabase,
   restore vào Postgres local, verify checksum (SC-008/SC-011).
3. **Kỳ vọng**: teardown sạch; restore drill pass; checklist T-minus có người ký xác nhận.

## Ma trận truy vết (rút gọn)

| Scenario | User Story | FR | SC |
|----------|-----------|----|----|
| VS1 | US1 | FR-001..003 | SC-001, SC-007 |
| VS2 | US2 | FR-009..012 | SC-002 |
| VS3 | US5 | FR-018..021 | SC-006 |
| VS4 | US5 | FR-019/FR-022..024/FR-028/FR-030 | SC-012 |
| VS5 | US4/US7 | FR-004..008/FR-032 | SC-004, SC-005, SC-008, SC-012 |
| VS6 | US3 | FR-013..017, FR-025..029 | SC-003, SC-009 |
| VS7 | — | Governance | — |
| VS8 | US6/US7 | FR-031/FR-032 | SC-008, SC-010, SC-011 |
