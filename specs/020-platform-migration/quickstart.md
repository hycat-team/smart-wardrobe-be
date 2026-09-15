# Quickstart Validation: Platform Migration & 3 Services

**Feature**: `020-platform-migration` | **Date**: 2026-09-15

Hướng dẫn kiểm chứng end-to-end cho feature. Đây là **run/verify guide**, không phải mã triển khai.

Tham chiếu: [spec.md](./spec.md) · [data-model.md](./data-model.md) · [contracts/](./contracts/inter-service-grpc.md) · ADR 021 (`docs/decisions/021-funded-platforms-and-repatriation.md`)

## Prerequisites

- Go 1.25+, Docker Desktop, GNU Make.
- `buf` CLI (cài qua `make tool`) cho sinh mã gRPC.
- Postgres + Redis + RabbitMQ local qua `deployments/docker-compose.yml`.
- Tài khoản funded (Cloudflare Pro — funded nhưng **KHÔNG nằm trong đường API**, xem ADR 023; Supabase Pro, CloudAMQP Tiger, Redis Cloud Essentials) +
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

## VS2 — Khởi động trên pooler, outbox vẫn phát (US2, FR-046, SC-002, SC-020)

1. Cấu hình DB runtime trỏ Supabase pooler (managed, transaction mode 6543); direct DSN trỏ 5432.
2. Xác nhận runtime dùng `default_query_exec_mode=simple_protocol` và **`search_path` set server-side**
   (`ALTER ROLE`/`ALTER DATABASE`), không truyền qua DSN client.
   - **Kỳ vọng**: query resolve đúng 9 schema; **0 lỗi prepared statement** khi chạy nhiều transaction
     qua pooler (SC-020).
3. Tắt listener (`OUTBOX_LISTENER_ENABLED=false`) để mô phỏng `LISTEN` không khả dụng, chạy service.
4. **Kỳ vọng**: app khởi động thành công; log cảnh báo listener bị tắt/không khả dụng.
5. Tạo một bản ghi nghiệp vụ sinh event (ví dụ tạo wardrobe item).
6. **Kỳ vọng**: outbox worker phát event qua poll ticker trong vài giây; `outbox_pending` giảm về 0.
7. Đặt `DATABASE_MIGRATE_ON_START=false`, restart.
8. **Kỳ vọng**: app không tự chạy migration; chạy `make migration-up` (DSN session/direct) thành công.
9. Verify advisory lock chạy trên **direct 5432** (không qua pooler 6543).

## VS3 — Build 3 binary service, monolith đã xóa (US5, SC-006)

```bash
go work sync
go build ./services/account/... ./services/commerce/... ./services/ai/...
```

**Kỳ vọng**: cả 3 build thành công; `cmd/server` không còn tồn tại trên `main`; import
implementation xuyên service fail ở compile-time. Trong CI, job build 3 binary bắt buộc pass
(FR-020). Tag `monolith-legacy` tồn tại và frozen.

## VS4 — Topology 3 service + JWT + ingress auth (US5, FR-019/FR-022..024, FR-033)

1. Chạy `account`, `commerce`, `ai` local với gRPC bật; gọi API liên service (ví dụ tạo outfit
   cần dữ liệu fashion + subscription quota).
   - **Kỳ vọng**: kết quả đúng; giao tiếp liên service qua gRPC; correlation ID xuyên suốt.
2. Auth ở ingress là **Cloud Run IAM** (không `allUsers`); app không yêu cầu header gateway secret.
   - **Kỳ vọng**: request tới Cloud Run thiếu ID token hợp lệ bị 401/403; `Authorization` (Closy JWT)
     tới app nguyên vẹn.
3. Verify JWT ES256 bằng public key cục bộ (không gọi mạng); blacklist qua Redis.
4. **Contract test**: bộ test hành vi contract (in-process + gRPC) — cả hai adapter pass.
5. **Generate-check**: `buf lint && buf generate` → không có diff.
6. **Swagger parity**: hợp docs 3 service đủ 9 section; thiếu section là fail.

## VS5 — Repatriation về VPS: 3 service + Nginx (US4/US7, FR-055/FR-056, SC-004, SC-008, SC-029, SC-030)

1. Từ Supabase (môi trường Cloud) tạo dump; dựng VPS (hoặc local prod-like) theo
   `deployments/docker-compose.prod.yml` (3 service + postgres/redis/rabbitmq + nginx).
2. Restore dump; đặt profile `vps`: DB trỏ docker host, `sslmode=disable`, TLS tắt,
   `migrate_on_start=true`; Nginx route `api-v2` tới 3 service.
3. **Deploy qua workflow** `deploy-vps-services.yml` (matrix 3 service → GHCR → compose, pin digest).
   - **Kỳ vọng**: deploy từng service độc lập thành công; image pin theo digest (SC-030).
4. **Rollback khẩn cấp**: chạy `deploy-production.yml` (dispatch, build từ tag `monolith-legacy`) →
   compose profile `rollback`; xác nhận `api.closy` phục vụ monolith.
5. Trỏ DNS test về VPS; gọi ma trận host × route + SSE + auth + upload.
   - **Kỳ vọng**: `api-v2` **301 → `api.closy`**; DNS `api-v2` xoá sau grace period (SC-029).
6. **Kỳ vọng**: hệ thống chạy đầy đủ; chỉ khác biệt cấu hình/DNS, không sửa code; đo thời gian
   cán đích ≤ 4h (SC-004).
7. Smoke test route `api-v2` tới đủ 3 service (SC-012).

## VS6 — Song song trên môi trường Cloud: Vercel DNS → VPS Nginx → Cloud Run (US3, FR-013..017, FR-025..029, FR-033..040, FR-042, FR-047..FR-052, SC-003/SC-009/SC-013/SC-016/SC-017/SC-021..SC-026)

Thực hiện runbook trên môi trường Cloud (song song; **không cutover**):

1. Thêm bản ghi Vercel cho `api-v2.closy.hycat.online` → VPS; mở rộng cert Let's Encrypt; thêm
   server block Nginx `api-v2` route tới 3 Cloud Run; **giữ nguyên** `api.closy` → monolith.
2. Deploy token helper + Nginx `auth_request`; bật Cloud Run IAM (`run.invoker`, không `allUsers`);
   smoke test: thiếu ID token bị từ chối; có token → 200; `Authorization` tới app đúng.
3. **Deploy độc lập từng service** (`make image-build svc=ai && make image-push svc=ai && make deploy svc=ai`)
   không rebuild account/commerce; xác nhận image tag theo SHA của riêng service đó (SC-017).
   - Pull image từ **Artifact Registry** thành công (SC-025); secrets tiêm từ GitHub Actions secrets,
     **0 giá trị secret in ra log** (SC-026).
4. Migrate dữ liệu VPS → Supabase (1 database dùng chung); drill dùng `database.drill_schema` rồi
   xoá; smoke host × route; test SSE dài (heartbeat 15s, chunk deadline 30s, buffering off).
5. **gRPC giữa 3 service**: deploy HTTP service với `--use-http2`; verify `grpc.health.v1` +
   call thực Commerce→Account/AI, AI→Account (SC-022).
6. **Ngân sách capacity**: cap HTTP `max-instances=2`, worker `min=max=1`, `concurrency=40`;
   đo Supabase pool size + test tải nhẹ → tổng client conn ≤ giới hạn gói (AMQP ≤100, Redis ≤256)
   với headroom ≥20% (SC-021).
7. **Kỳ vọng**: `api-v2` chạy qua Nginx↔Cloud Run; `api.closy` **0 thay đổi routing**; 0 request
   `api-v2` đi qua Cloudflare (SC-016); deploy 1 service không ảnh hưởng 2 service còn lại; upload/media
   (Cloudinary) hoạt động (SC-027); `GET /api/v1/health` pass (SC-023).
8. Kiểm tra `api.closy` vẫn phục vụ monolith bình thường trong suốt quá trình.

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

## VS9 — Cutover thủ công `api.closy` → Cloud Run + rollback (FR-041, SC-015)

Chỉ chạy **khi owner quyết định** `api-v2` đủ ổn định. Drill trên môi trường Cloud trước:

1. Chạy runbook `docs/operations/manual-cutover-api-closy-to-cloudrun.md`: điều kiện tiên quyết
   (`api-v2` ổn định, dữ liệu đã đồng bộ), đổi Nginx `api.closy` → Cloud Run (hoặc DNS), kiểm thử.
2. **Rollback**: đổi Nginx/DNS `api.closy` về monolith; xác nhận API cũ hoạt động lại.
3. **Kỳ vọng**: cutover + rollback chạy được theo runbook; đo thời gian; không sửa mã nghiệp vụ;
   ghi lại bước lệch.

## VS10 — Runtime topology & app-mode (US3/US5, FR-044/FR-045/FR-049/FR-050/FR-054, SC-018/SC-019/SC-023/SC-024/SC-028)

1. Chạy service với `APP_MODE=http` → **0 consumer/outbox/cron** khởi động; `GET /api/v1/health` OK (SC-018).
2. Chạy `APP_MODE=worker` → consumer/outbox/cron khởi động; HTTP tối giản phục vụ health trên `PORT`
   (dùng CPU always-on khi deploy Cloud Run).
3. Chạy `APP_MODE=all` → hành vi như monolith hiện tại (local/VPS).
4. **Worker singleton**: deploy worker `min=max=1`; verify cron không chạy trùng; advisory lock bảo vệ
   job không idempotent (subscription renewal, loyalty expiry) khi instance restart/overlap (SC-019).
5. **Port**: chạy với `PORT=9090` → app bind đúng 9090, không hardcode 8080 (SC-023).
6. **Timezone distroless**: chạy image nonroot → giờ `Asia/Ho_Chi_Minh` đúng, 0 lỗi `LoadLocation` (SC-024).
7. **Rate-limit Redis-backed**: mô phỏng 2 instance → 429 nhất quán cho rule login/OTP + API chung,
   ngưỡng **không reset** khi instance scale (SC-028).

## Ma trận truy vết (rút gọn)

| Scenario | User Story | FR | SC |
|----------|-----------|----|----|
| VS1 | US1 | FR-001..003 | SC-001, SC-007 |
| VS2 | US2 | FR-009..012, FR-046 | SC-002, SC-020 |
| VS3 | US5 | FR-018..021 | SC-006 |
| VS4 | US5 | FR-019/FR-022..024/FR-030/FR-033 | SC-012, SC-013 |
| VS5 | US4/US7 | FR-004..008/FR-032/FR-055/FR-056 | SC-004, SC-005, SC-008, SC-012, SC-029, SC-030 |
| VS6 | US3 | FR-013..017, FR-025..029, FR-033..040, FR-042, FR-047..FR-052 | SC-003, SC-009, SC-013, SC-016, SC-017, SC-021..SC-026 |
| VS7 | — | Governance | — |
| VS8 | US6/US7 | FR-031/FR-032 | SC-008, SC-010, SC-011 |
| VS9 | US3 (cutover thủ công) | FR-041 | SC-015 |
| VS10 | US3/US5 (runtime) | FR-044/FR-045/FR-049/FR-050/FR-054 | SC-018, SC-019, SC-023, SC-024, SC-028 |
