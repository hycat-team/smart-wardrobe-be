# Quickstart: Pure Contract Refactor

**Feature**: `015-contract-purity-refactor`
**Ngày**: 2026-08-27

---

## Mục đích

Hướng dẫn verify feature hoạt động end-to-end sau khi refactor.

---

## Điều kiện tiên quyết

- Branch `015-contract-purity-refactor` đã checkout.
- Go 1.25+, `make` có sẵn.
- Đã chạy `make wire` thành công trước đó (wire_gen.go tồn tại).

---

## Bước 1: Audit 3 metric (phải về 0 sau refactor)

Chạy audit script (sẽ tạo trong tasks):

```powershell
# Detour call — số chuỗi 2-hop
scripts/audit-contract-purity.ps1 -Check detour

# Pure violation — số contract impl leaky
scripts/audit-contract-purity.ps1 -Check pure

# Deadcode — số method không có caller
scripts/audit-contract-purity.ps1 -Check deadcode
```

**Kỳ vọng sau refactor**: `detour: 0`, `pure: 0`, `deadcode: 0`.

Nếu script chưa có, dùng thủ công:

```powershell
# Deadcode ví dụ
Select-String -Path "internal/modules/wardrobe/presentation/contract/wardrobe_contract.go" -Pattern "CopyItemToUser" | ForEach-Object { $_.Line }
# Rồi grep caller:
Select-String -Path "internal" -Pattern "CopyItemToUser" -Recurse
```

---

## Bước 2: Build & Lint

```powershell
make wire
make build
make lint
```

**Kỳ vọng**: Không lỗi. `wire_gen.go` regenerate thành công.

---

## Bước 3: Test

```powershell
make test
```

**Kỳ vọng**: Tất cả test pass. Nếu test fake dùng method đã xóa → update fake trước.

---

## Bước 4: Manual E2E (4 case)

### Case 1: Upload
```powershell
make dev
# POST /api/v1/wardrobe/items:batch-upload  (2 ảnh)
# GET /api/v1/wardrobe/items  → verify 2 items, status = Processing
```

### Case 2: Retry analysis
```powershell
# POST /api/v1/wardrobe/items/{id}/retry-analysis
# Poll GET /api/v1/wardrobe/items/{id} mỗi 200ms → FashionItemID xuất hiện trong 1-2s
```

### Case 3: Cleanup failed
```powershell
# Trigger FashionWorker.CleanupFailedItems (hoặc đợi cron)
# GET /api/v1/wardrobe/items → số item failed giảm
```

### Case 4: Outfit recommendation
```powershell
# POST /api/v1/fashion/outfits/recommend  (với wardrobe items)
# Verify response có fashion data enrich (sau khi caller tự enrich)
```

**Kỳ vọng**: Kết quả giống trước refactor (functional behavior không đổi — FR-006).

---

## Bước 5: Benchmark NFR-001

```powershell
# So sánh số DB call trước/sau (ví dụ log count)
go test -bench=BenchmarkBatchUploadWardrobeItems -benchtime=1x
go test -bench=BenchmarkOutfitRecommendation -benchtime=1x
```

**Kỳ vọng**: Tăng không quá 10%. Nếu vượt → optimize batch query (gộp ListFashionItems).

---

## Bước 6: Constitution & Doc

- Verify `docs/system-design/contract-purity-pattern.md` tồn tại.
- Verify `.specify/memory/constitution.md` §II có đoạn Pure Contract + link tới doc.

---

## Tham khảo

- [spec.md](spec.md)
- [research.md](research.md)
- [data-model.md](data-model.md)
- [contracts/internal-contracts.md](contracts/internal-contracts.md)
