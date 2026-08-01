# Sprint 1 — Kết quả đánh giá & refactor (T001-T023)

**Ngày**: 2026-08-01
**Branch**: `refactor/assess-schema-separation`

> Lưu ý: Các tài liệu chi tiết ban đầu đặt tại `docs/temp/technical-debt/` (thư mục cá nhân, bị gitignore). Nội dung cốt lõi được sao lưu trong `research.md`, `data-model.md`, `contracts/contract-design.md` và file này để lưu trữ trên git.

## 1. Phân loại vi phạm cross-module (đã xử lý)

| Nhóm | Mô tả | Mức độ | Xử lý |
|------|-------|--------|-------|
| wardrobe → fashion | `wardrobe_repo.go` JOIN/UPDATE `fashion_items`/`categories` | HIGH | ✅ Refactor qua `IFashionContract` + worker ghi 2 phía |
| fashion → wardrobe | `GetHybridCandidates` query `wardrobe_items` | HIGH | ✅ `GetUserFashionItemIDs` + query `fashion_items` |
| dashboard → 5 module | 3 repo dashboard query bảng brand/subscription/aiexecutor/wardrobe/fashion | LOW | ✅ Refactor qua contract reader |

## 2. Kết quả Sprint 1 (T001-T023)

### Phase 1.1 — Đánh giá
- Cập nhật `docs/system-design/architecture.md` §VII: bảng mapping thực tế (42 bảng theo module), ghi chú FK xuyên module.
- 2 tech debt docs + tài liệu đánh giá khả thi (bản nháp trong `docs/temp/`, không commit).

### Phase 1.2 — Wardrobe ↔ Fashion
- `IFashionContract` + `GetStaleProcessingInfo`, `ReassignCategoryForCatalogItems`, `ResetProcessingState`, `SearchFashionItemIDs`, `GetCategoryIDsByFashionItemIDs`.
- `IWardrobeContract` + `GetUserFashionItemIDs`.
- `wardrobe_repo.go`: xóa toàn bộ JOIN/UPDATE `fashion_items` (12 vị trí). Processing methods chỉ còn update `wardrobe_items`.
- `GetHybridCandidates`: query `fashion_items` + trả id/scores; caller stitch wardrobe qua contract (giữ HNSW index).
- Fashion analyze worker: ghi `fashion_items` qua `fashionItemRepo` + `wardrobe_items` qua `wardrobeContract.UpdateItemStatus` (đúng module boundary).
- `RetryWardrobeAnalysis`: gọi `IFashionContract.ResetProcessingState` sau khi claim.

### Phase 1.3 — Dashboard
- Thêm ~18 method analytics vào 5 module qua use case + repo + reader:
  - `IBrandDashboardAnalyticsUseCase` (mới, 11 method) + `IBrandDashboardReader` + 11 method.
  - `ISubscriptionDashboardReader` + 3 method, `IAICostDashboardReader` + 3 method.
  - `IWardrobeDashboardReader` + `GetUserWardrobeFashionItemIDs`.
  - `IFashionDashboardReader` (mới) + `GetCategoriesByIDs`, `GetCategoryIDsByFashionItemIDs`.
- 3 dashboard repos: xóa 18 method cross-module, chỉ còn snapshot CRUD.
- Dashboard use cases: chuyển từ `[]map` sang typed reader DTO + stitch bằng `sliceutils`.
- Unit test: brand analytics use case (5 test), fashion worker (2 test), sửa mocks brand/subscription/dashboard.

## 3. Verification

- ✅ `go build ./...` pass
- ✅ `go vet ./...` pass
- ✅ `go test ./...` — ALL TESTS PASS
- ✅ `make tidy-check` pass
- ✅ `make wire` idempotent (wire_gen.go ổn định)
- ✅ Scan: wardrobe 0 tham chiếu `fashion_items`; fashion không query `wardrobe_items`; dashboard 0 raw query bảng cross-module.

## 4. Còn lại (Sprint 2 — T024-T034)

- Migration tách schema (`CREATE SCHEMA` + `SET SCHEMA` 42 bảng + sequence, cắt 9 FK xuyên module phi-users, giữ ~30 FK tới users).
- Cập nhật DSN `search_path` + GORM.
- Verify HNSW index + toàn bộ luồng nghiệp vụ trên schema tách.
