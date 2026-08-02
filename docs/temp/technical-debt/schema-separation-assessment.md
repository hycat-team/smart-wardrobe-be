# Đánh giá khả thi tách database schema theo module

**Trạng thái**: Đánh giá (feature `011-assess-schema-separation`)
**Ngày**: 2026-08-01

## 1. Phân loại & ưu tiên vi phạm cross-module

| Nhóm | Mô tả | Mức độ | Số lượng | Xử lý |
|------|-------|--------|----------|-------|
| wardrobe → fashion | `wardrobe_repo.go` JOIN/UPDATE `fashion_items`/`categories` | **HIGH** (gồm UPDATE ghi) | 12 vị trí | Refactor qua `IFashionContract` + outbox (Sprint 1) |
| fashion → wardrobe | `GetHybridCandidates` query `wardrobe_items` | **HIGH** | 1 function | `GetUserFashionItemIDs` + query `fashion_items WHERE id IN (...) ORDER BY embedding` (Sprint 1) |
| dashboard → 5 module | 3 file repo dashboard query bảng brand/subscription/aiexecutor/wardrobe/fashion | **LOW** (read-only) | 18 hàm | Refactor qua contract reader (Sprint 1) |

## 2. FK xuyên module

### 2.1 CẮT (phi-users, enforce qua contract) — 9 FK

| FK | Từ module → Đến |
|----|------------------|
| `ai_policy_grants.plan_id` | aiexecutor → subscription |
| `ai_policy_grants.source_event_id` | aiexecutor → subscription |
| `ai_policy_grants.source_deposit_transaction_id` | aiexecutor → subscription |
| `subscription_plans.ai_cost_policy_id` | subscription → aiexecutor |
| `wardrobe_items.fashion_item_id` | wardrobe → fashion |
| `outfit_items.fashion_item_id` | wardrobe → fashion |
| `brand_items.fashion_item_id` | brand → fashion |
| `digital_sample_responses.outfit_id` | brand → wardrobe |
| `dashboard_brand_daily_snapshots.brand_id` | dashboard → brand |

**Khả năng cắt**: Toàn bộ 9 FK CẮT ĐƯỢC sau khi Sprint 1 hoàn tất (mọi luồng ghi đều qua contract). Khi Down tái lập FK phải check dữ liệu không orphan trước `ADD CONSTRAINT`.

### 2.2 GIỮ (xuyên schema tới `users`) — ~30 FK

Gần như mọi module có FK trỏ tới `users` (identity, wardrobe, fashion, brand ×15, subscription ×10, aiexecutor, dashboard). PostgreSQL hỗ trợ FK xuyên schema → GIỮ nguyên, không cắt, referential integrity không đổi.

## 3. Kết luận khả thi

**KẾT LUẬN: Tách schema theo module KHẢ THI**, với điều kiện Sprint 1 (xử lý tech debt) hoàn tất trước.

### Chi phí ước lượng

- **Sprint 1 (tech debt)**: refactor ~30 file (contract interface + impl + use case + repo + DI) — chi phí trung bình.
- **Sprint 2 (tách schema)**: 1 migration thuần `SET SCHEMA` (42 bảng + 1 sequence) + cấu hình DSN `search_path` + verify — chi phí thấp vì không đổi dữ liệu.

### Rủi ro

1. **Migration consolidated** (`20260628000000_closy_b2b2c_rebuild.sql`) chứa nhiều phần Up/Down lồng nhau — không được phá, migration mới phải độc lập.
2. **`users` hub trung tâm**: giữ FK xuyên schema (Q1) — không cần cắt, nhưng cần đảm bảo search_path resolve đúng schema.
3. **Vector search**: phương án `GetUserFashionItemIDs` + query `fashion_items WHERE id IN (...) ORDER BY embedding <=> ?` tận dụng HNSW index (đính chính nhận định ban đầu).
4. **Worker critical path**: chuyển sang eventual consistency qua outbox — cần row lock `FOR UPDATE` + version check + idempotent consumer (Q5).

### Thứ tự thực hiện

1. **Sprint 1 — Tech debt** (feature 011): đánh giá → wardrobe↔fashion → dashboard.
2. **Sprint 2 — Tách schema**: migration `SET SCHEMA` + `search_path` + verify toàn bộ luồng nghiệp vụ.

### Giới hạn chấp nhận được

- Eventual consistency giữa `wardrobe_items` và `fashion_items` (độ trễ nhỏ, outbox đảm bảo at-least-once).
- 1 connection chung, không phân quyền DB-level (tách chỉ về mặt tổ chức/ownership).
- Cắt FK xuyên module phi-users → integrity enforce qua contract ở app layer (mất integrity cấp DB, cần test bù).
