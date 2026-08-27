# Research: Pure Contract Refactor

**Feature**: `015-contract-purity-refactor`
**Ngày**: 2026-08-27 (cập nhật tránh overengineering)

---

## 1. Thuần hóa trực tiếp (không tạo *PureUseCase mới)

**Decision**: Sửa trực tiếp `WardrobeContractUseCase` (7 method) và 5 method leaky của `BrandItemUseCase` cho thuần (bỏ `fashionContract`/`userSubContract` enrich), không tạo interface `IPure*UseCase` mới. `Category`/`Subscription` đã pure ở method-level nên không sửa.

**Rationale**:
- Wardrobe đã có contract usecase riêng (`contractuc/contract_uc.go`) — sửa trực tiếp ít churn, không thêm file.
- Tạo `IPure*` mới sẽ duplicate interface và tăng Wire provider, là overengineering.
- 3/7 method leaky của wardrobe trùng deadcode (`CopyItemToUser`, `VerifyItemsForPost`, `GetItemsByIDs`) → xóa luôn, chỉ cần thuần hóa 4 method alive.

**Alternatives considered**:
- Tạo `IPure*` mới — loại vì overengineering, tăng 4 file/interface không cần thiết.
- Giữ enrich trong contract — loại vì tái tạo detour call 7 chuỗi.

---

## 2. Enrichment ở caller (nil risk)

**Decision**: Caller sau thuần hóa nhận `FashionItem == nil` → phải tự gọi `fashionContract.ListFashionItems`/`GetFashionItem` sau khi có IDs, stitch bằng `sliceutils.IndexBy`/`GroupBy` (Constitution §II). 7 caller fashion + 2 caller brand phải rà.

**Rationale**:
- Làm rõ trách nhiệm: fashion caller chịu enrich, không phụ thuộc wardrobe enrich ngầm.
- Tránh nil: nếu sót, UI thiếu ảnh/category/style.

**Alternatives considered**:
- Giữ enrich trong wardrobe 1 call — loại vì tái tạo detour.

---

## 3. Deadcode Verification

**Decision**: Dùng `rg` grep toàn bộ `internal/` loại trừ `*_test.go`, `*_impl.go` cho từng method name. 12 method deadcode đã verify; 3 trong số đó trùng leaky nên xóa giải quyết cả hai.

---

## 4. Wire

**Decision**: Không tách ProviderSet. Chỉ bỏ field `fashionContract`/`userSubContract` khỏi `WardrobeContractUseCase` constructor và `fashionContract` khỏi `BrandItemUseCase` helper path (nếu còn field). Regenerate `wire_gen.go`.

---

## 5. Constitution §II Update + Doc

**Decision**: Viết `docs/system-design/contract-purity-pattern.md` trước, sau đó cập nhật `.specify/memory/constitution.md` §II thêm "Pure Contract Usecase Principle" và link tới doc (clarify #2 + #3).

---

## 6. Performance NFR-001

**Decision**: Benchmark DB call count trước/sau với `BatchUploadWardrobeItems`, `OutfitRecommendation`. Nếu tăng >10% do caller enrich thêm thì gộp batch (1 `ListFashionItems` với nhiều IDs) hoặc cache trong cùng request.

---

## 7. Tài liệu tham khảo

- Audit method-level 2026-08-27 — wardrobe 7/15 leaky, brand 5/9 leaky, subscription/category đã pure.
- `docs/temp/detour-call-audit/02_wardrobe_fashion.md`
- `docs/system-design/design-pattern.md`
- `.specify/memory/constitution.md` §II, §IV
