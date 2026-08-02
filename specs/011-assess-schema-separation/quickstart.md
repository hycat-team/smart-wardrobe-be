# Quickstart: Xử lý tech debt cross-module & tách schema theo module

**Phase 1 output** — hướng dẫn validation end-to-end sau khi triển khai.

## Prerequisites

- PostgreSQL 16 + pgvector (`CREATE EXTENSION vector`), RabbitMQ, Redis, Cloudinary — chạy qua `make docker-dev-up` hoặc `docker-compose up -d`.
- Config: copy `.env.example` → `.env`, set `DB_NAME=smart_wardrobe_db`.
- Migration lên schema hiện tại: `make migration-up`.
- Seed demo (dashboard): `psql -f database/scripts/seed_feature_006_dashboard.sql`.

## Sprint 1 — Validate sau khi refactor

### 1. Cross-module DB access đã hết

```bash
make vet          # go vet pass
make test         # unit + integration pass
```

Scan xác nhận không còn raw access cross-module:
```bash
# không được còn JOIN/UPDATE fashion_items trong wardrobe_repo
# không được còn Table("wardrobe_items") trong fashion repo
# không được còn raw query bảng module khác trong dashboard repos
```

### 2. Vector search (GetHybridCandidates) vẫn hoạt động + dùng HNSW

1. Chạy API AI recommendation (vd: `POST /api/v1/fashion/ai/recommendations`) với user có wardrobe items.
2. Kết quả trả về danh sách gợi ý đúng (không lỗi, đủ top-K).
3. Verify HNSW index được dùng:
   ```sql
   EXPLAIN (ANALYZE, BUFFERS)
   SELECT id FROM fashion_items WHERE id IN (...) ORDER BY embedding <=> '...' LIMIT 20;
   -- phải thấy "Index Scan using fitems_embedding_cosine_idx"
   ```

### 3. Worker processing + worker-direct (Sprint 1)

1. Trigger AI processing (upload item → worker claim).
2. Kiểm tra `wardrobe_items` status update (Processing → InWardrobe/Failed/NeedsReview) và `fashion_items` được ghi metadata + embedding qua fashion repo (worker-direct, không qua outbox cho claim/processing state):
   ```sql
   SELECT id, processing_version, review_reason FROM fashion_items WHERE id = '<fashion_item_id>';
   ```
3. Kiểm tra fashion worker ghi `fashion_items` + đổi status `wardrobe_items` qua contract (`UpdateItemStatus`).
4. Test race condition claim: 2 worker claim cùng item → chỉ 1 thắng (version check trên `fashion_items`).
5. Đo latency claim retry ≤ +100ms so với baseline trước refactor (SC-007).
6. Verify các thao tác đồng bộ qua contract vẫn hoạt động: `GetStaleProcessingInfo`, `ReassignCategoryForCatalogItems`.
7. Recovery worker `RecoverStaleProcessingItems` republish stale items hoặc mark failed khi retry cạn.

### 4. Dashboard APIs

Chạy toàn bộ API dashboard (user/brand/admin) so sánh response với trước refactor:
- `GET /me/dashboard/*`
- `GET /brand/dashboard/*`
- `GET /admin/dashboard/*`

Empty data → zero value (không lỗi), dữ liệu giống hệt trước refactor.

## Sprint 2 — Validate sau khi tách schema

### 1. Migration round-trip

```bash
make migration-up     # tách schema
make migration-status # toàn bộ applied
# chạy toàn bộ test suite trên schema tách
make migration-down   # rollback về public
make migration-status # toàn bộ applied
```

Verify:
- Không mất dữ liệu (so sánh count trước/sau round-trip).
- Down tái lập FK xuyên module không lỗi (không orphan).

### 2. Schema mapping

```sql
-- Mỗi bảng ở đúng schema module
SELECT schemaname, tablename FROM pg_tables
WHERE schemaname IN ('identity','wardrobe','fashion','brand','subscription','aiexecutor','dashboard','shared')
ORDER BY schemaname, tablename;
```

- `deposit_transactions_order_code_seq` ở schema `subscription`.
- Sequence vẫn hoạt động (tạo deposit transaction → order_code tăng).

### 3. Search_path hoạt động

1. App khởi động với DSN `search_path=public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared` (`public` đứng đầu để `goose_db_version` resolve — bảng goose luôn ở public).
2. Chạy các luồng nghiệp vụ chính:
   - Worker processing (claim/retry/complete).
   - AI recommendation (vector search + HNSW).
   - Dashboard (user/brand/admin).
   - Payment (PayOS webhook → deposit → wallet → subscription).
   - Loyalty (earn/redeem/expire).
3. Verify FK tới `users` vẫn enforce (vd: xoá user → cascade delete; chèn bản ghi vi phạm `user_id` không tồn tại → bị chặn bởi FK).

### 4. Full quality gate

```bash
make check   # fmt + tidy + generate + vet + test + build pass
```

## Expected outcomes

- **SC-001/002/003** (tài liệu): đối chiếu docs vs code không còn discrepancy; §VII không còn bảng ma; FK xuyên module liệt kê đủ (9 FK phi-users cắt + ~30 FK tới `users` giữ).
- **SC-004** (Sprint 1): 0 cross-module DB access; `go vet` + test pass.
- **SC-005** (Sprint 2): toàn bộ test suite + luồng nghiệp vụ chính pass trên schema tách.
- **SC-006**: HNSW index vẫn dùng trong vector search sau 2 sprint.
- **SC-007**: Latency claim retry không tăng quá +100ms so với trước refactor.
