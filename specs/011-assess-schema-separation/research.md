# Research: Xử lý tech debt cross-module & tách schema theo module

**Phase 0 output** — tổng hợp nghiên cứu và quyết định kỹ thuật trước khi design.

## 1. Cross-module DB access hiện hữu

### Decision
- Wardrobe module đang JOIN/UPDATE trực tiếp bảng `fashion_items`/`categories` (12 vị trí trong `wardrobe_repo.go`).
- Fashion module query bảng `wardrobe_items` (`GetHybridCandidates`, `fashion_item_repo_hybrid.go:44-46`).
- Dashboard module query trực tiếp bảng của brand/subscription/aiexecutor/wardrobe/fashion (18 hàm trong 3 repo).

### Rationale
- Xác nhận đúng 2 tech debt docs (sau khi bổ sung discrepancy).
- Tất cả đều vi phạm Hiến pháp I & II — cần đưa qua contract trước khi tách schema.

### Alternatives considered
- Giữ raw query cross-module + exception hiến pháp → bị loại vì tách schema sẽ fail ngay (bảng ở schema khác).
- Giữ nguyên, không tách schema → mục tiêu cuối của feature không đạt.

## 2. Xử lý vector search `GetHybridCandidates`

### Decision
- Wardrobe expose `GetUserFashionItemIDs(ctx, userID)` qua `IWardrobeContract` (chỉ `SELECT fashion_item_id ...`, không trả entity).
- Fashion query `fashion_items WHERE id IN (?) ORDER BY embedding <=> ?` trên bảng của mình.
- **Đính chính**: HNSW index nằm trên `fashion_items.embedding` (không phải wardrobe_items); query `WHERE id IN + ORDER BY embedding` tận dụng HNSW tốt hơn query join hiện tại.

### Rationale
- Tuân thủ Hiến pháp I & II 100% (không cần exception).
- Giữ HNSW index — không hồi quy hiệu năng (SC-006).
- Tủ user nhỏ (Standard 50 items, Premium vài trăm) nên tập `id IN` nhỏ, sort nhanh.

### Alternatives considered
- Giữ raw query + exception hiến pháp → bị loại (vi phạm triệt để hơn).
- Query `fashion_items` HNSW top-K toàn cục rồi lọc theo user → bị loại (có thể thiếu kết quả vì top-K toàn cục không thuộc tủ user, phải chọn K lớn).

## 3. Xử lý bảng hub `users` (Q1)

### Decision
- Tách identity ra schema riêng; giữ FK xuyên schema tới `users` (PostgreSQL hỗ trợ FK xuyên schema).
- ~30 FK tới `users` được giữ, referential integrity không đổi.

### Rationale
- `users` là hub trung tâm bị gần như mọi module tham chiếu; cắt hết FK → mất integrity cấp DB, rủi ro cao.
- PostgreSQL cho phép FK xuyên schema nên không cần đánh đổi.

### Alternatives considered
- Schema `core`/`shared` chứa users → bị loại (đi lệch module ownership).
- Cắt hết FK, UUID thuần → bị loại (mất integrity, rủi ro dữ liệu lỗi).
- Giữ users ở public → bị loại (identity không tuân thủ boundary).

## 4. Xử lý FK xuyên module còn lại (Q2)

### Decision
- Cắt toàn bộ FK xuyên module (trừ FK tới `users`), enforce integrity qua contract ở app layer.
- Danh sách 9 FK cắt: `ai_policy_grants.plan_id/source_event_id/source_deposit_transaction_id`, `subscription_plans.ai_cost_policy_id`, `wardrobe_items.fashion_item_id`, `outfit_items.fashion_item_id`, `brand_items.fashion_item_id`, `digital_sample_responses.outfit_id`, `dashboard_brand_daily_snapshots.brand_id`.

### Rationale
- Đúng tinh thần module boundary (Hiến pháp I) — mỗi schema độc lập thực sự.
- Giữ FK xuyên schema giữa module phi-users làm 2 schema vẫn ràng buộc vật lý lẫn nhau, giảm ý nghĩa tách.

### Alternatives considered
- Giữ toàn bộ FK xuyên schema → bị loại (2 schema vẫn coupling).
- Soft-check trigger → bị loại (phức tạp, chi phí cao).

## 5. Phân quyền DB-level (Q3)

### Decision
- Không phân quyền theo module; vẫn dùng 1 connection chung truy cập mọi schema.
- Tách schema chỉ về mặt tổ chức/ownership.

### Rationale
- Modular monolith 1 service; nhiều role/connection phức tạp DSN/DI/GORM mà giá trị bảo mật thấp (app đã kiểm soát qua contract).
- Đúng tinh thần "pragmatic" của Hiến pháp I.

### Alternatives considered
- Nhiều role/connection theo module → bị loại (phức tạp, ít lợi ích).

## 6. Migration rollback (Q4)

### Decision
- Migration tách schema thuộc nhóm "thuần chuyển schema": `CREATE SCHEMA` + `SET SCHEMA` (table/sequence/index) + `DROP CONSTRAINT`/`ADD CONSTRAINT`, không đổi dữ liệu → **viết Down đầy đủ**.
- Khi Down tái lập FK xuyên module đã cắt: check dữ liệu không orphan trước `ADD CONSTRAINT`.

### Rationale
- Xét schema thực tế (`closy_db-schema.sql`): 42 bảng + sequence `deposit_transactions_order_code_seq` đều ở `public`, không có UPDATE/INSERT...SELECT/merge/drop cột → an toàn để viết Down.

### Alternatives considered
- Migration đổi dữ liệu → không viết Down, dùng Backup + Restore (áp dụng cho migration khác, không phải feature này).

## 7. Atomicity cross-module critical path worker (Q5)

### Decision
- Chuyển sang eventual consistency: wardrobe ghi `wardrobe_items` + publish outbox event trong cùng transaction (transactional outbox, Hiến pháp III); fashion consume event và update `fashion_items` bất đồng bộ.
- Xử lý race condition claim: row lock `FOR UPDATE` trên bảng sở hữu + `processing_version` check + idempotent consumer.

### Rationale
- Hiến pháp III: cross-module write side effects nên dùng event publishing.
- Giữ được độ trễ nhỏ chấp nhận được, không phá worker critical path.

### Alternatives considered
- Truyền transaction qua contract (atomic tuyệt đối) → bị loại theo Q5 (người dùng chọn eventual consistency).
- Giữ raw access exception cho critical path → bị loại (vi phạm Hiến pháp I).

## 8. GORM / search_path config

### Decision
- DSN thêm `search_path` = tất cả schema module (identity, wardrobe, fashion, brand, subscription, aiexecutor, dashboard, shared, public).
- GORM `NamingStrategy` giữ nguyên (bảng tìm qua search_path); raw SQL không cần sửa vì search_path resolve đúng schema.

### Rationale
- 1 connection chung + search_path là cách đơn giản nhất để mọi query hiện có (không schema-qualified) hoạt động đúng sau khi bảng di chuyển.

### Alternatives considered
- Schema-qualified tên bảng trong từng raw SQL → bị loại (phải sửa hàng loạt, dễ sót).
- Nhiều connection theo schema → bị loại (Q3).

## 9. Tech stack & công cụ

- PostgreSQL 16 + pgvector (HNSW index: `fitems_embedding_cosine_idx` trên `fashion_items`, `ustyles_embedding_cosine_idx` trên `user_style_profiles`).
- GORM (SingularTable, không prefix), Goose migration (`-- +goose Up/Down`), Google Wire DI.
- Outbox pattern đã có sẵn (`outbox_records` + `internal/shared/infrastructure/messaging/outbox.go`).
- Lệnh verify: `make vet`, `make test`, `make check`; migration: `make migration-up/down/status`.
