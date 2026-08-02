# Technical Debt: Cross-Module Raw Database Access

**Phát hiện**: Sprint 2 (tech-debt remediation)  
**Trạng thái**: ĐANG XỬ LÝ trong Sprint 1 (feature `011-assess-schema-separation`)  
**Mức độ**: HIGH (vi phạm Constitution I & II)

## Vấn đề

Cả `wardrobe` và `fashion` module đang JOIN/UPDATE entity của module kia trực tiếp qua SQL raw, vi phạm:

- **Constitution I**: *"Cross-module database joins/preloads on entities of other modules are strictly prohibited"*
- **Constitution II**: *"A module must never import another module's entities or persistence structures"*

---

## 1. Wardrobe module → fashion_items

**File**: `internal/modules/wardrobe/infrastructure/persistence/wardrobe_repo.go`

| Dòng | Hành vi | Chi tiết |
|------|---------|----------|
| 67-68 | `LEFT JOIN fashion_items ON ...` | Query search với fashion_items + categories |
| 200-201 | `LEFT JOIN fashion_items ON ...` | `GetStaleProcessingItems` — truy vấn stale items |
| 234 | `UPDATE fashion_items SET ...` | Claim retry: update trực tiếp processing_version |
| 253-254 | `LEFT JOIN fashion_items ON ...` | `ClaimStaleProcessingRetry` + check processing_version |
| 266 | `UPDATE fashion_items SET ...` | Increment `processing_retry_count` |
| 331-332 | `LEFT JOIN fashion_items ON ...` | `updateProcessingState` claim + check version |
| 376 | `UPDATE fashion_items SET ...` | `updateProcessingState` — update nhiều fields |
| 409-410 | `LEFT JOIN fashion_items` + `categories` | `CountItems` query |
| 424-425 | `LEFT JOIN fashion_items` + `categories` | `GetItems` query |
| 433 | WHERE filter | Search condition trên `fashion_items.color, style, ...` |
| 460-461 | `LEFT JOIN fashion_items ON ...` | `CountByCategoryIDAndItemType` |
| 468 | `UPDATE fashion_items SET category_id = ...` | Update category_id của fashion_items |

---

## 2. Fashion module → wardrobe_items

**File**: `internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid.go`

| Dòng | Hành vi | Chi tiết |
|------|---------|----------|
| 44-46 | `Table("wardrobe_items")` + `LEFT JOIN fashion_items` | `GetHybridCandidates` — search vector + full-text |

---

## 3. Import domain constants cross-module

Import `domain/constants/` của module khác được Hiến pháp II cho phép (exception), NHƯNG phải dùng symbol (không hardcode giá trị):

| File | Dòng | Import | Ghi chú |
|------|------|--------|---------|
| `internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid.go` | 7 | `wardrobe/domain/constants/wardrobestatus` | Hợp lệ (dùng symbol `wardrobestatus.InWardrobe`), không hardcode |

**Lưu ý**: Các vi phạm hardcode giá trị domain constant (không dùng symbol) đã được xử lý ở feature 009. Nếu phát hiện thêm trong quá trình refactor, thay bằng symbol import.

---

## Hướng giải quyết (đã chốt — feature 011)

### Đọc dữ liệu (fashion → wardrobe_items)
- Fashion module gọi `IWardrobeContract.GetUserFashionItemIDs(ctx, userID)` qua `presentation/contract/` (chỉ `SELECT fashion_item_id`, không trả entity).
- Fashion module query `fashion_items WHERE id IN (?) ORDER BY embedding <=> ?` trên bảng của mình — tận dụng HNSW index (đính chính: HNSW index nằm trên `fashion_items`, không phải wardrobe_items).
- Dùng `sliceutils.IndexBy` / `sliceutils.GroupBy` để stitch data (Constitution II).

### Ghi dữ liệu (wardrobe → fashion_items)
- Wardrobe module gọi `IFashionContract` thay vì UPDATE trực tiếp bảng fashion_items (Constitution I + II).
- **Phân định**: claim retry + processing state (mark failed / needs review / complete success) → chuyển sang outbox event `wardrobe.processing.state` do fashion consumer xử lý (eventual consistency, Hiến pháp III); đọc stale info (`GetStaleProcessingInfo`) + reassign category (`ReassignCategoryForCatalogItems`) → giữ đồng bộ qua `IFashionContract`.
- `IWardrobeContract` đã có sẵn `GetStaleProcessingItems`, `MarkProcessingFailed`, `MarkProcessingNeedsReview`, `CompleteProcessingSuccess` (operate trên `wardrobe_items`) — minh chứng hướng fix khả thi.

### Rủi ro
- Vector similarity search (`fashion_items.embedding <=> ?`) — phương án `GetUserFashionItemIDs` + query `fashion_items WHERE id IN (...) ORDER BY embedding <=> ?` giữ HNSW index (đính chính nhận định trước rằng HNSW bị giảm hiệu năng).
- Batch UPDATE (`processing_retry_count`, `processing_version`) là critical path của worker — chuyển sang outbox event, cần row lock `FOR UPDATE` + version check + idempotent consumer để tránh race condition.
