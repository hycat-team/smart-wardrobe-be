# Research: Pure Contract Refactor

**Feature**: `015-contract-purity-refactor`
**Ngày**: 2026-08-27 (cập nhật tách Brand + 2 dead wiring)

---

## 1. Thuần hóa (sửa trực tiếp nếu đã tách, tạo mới nếu dùng chung)

**Decision**: Wardrobe đã tách (`contractuc/`) → sửa trực tiếp 4 alive leaky (3 deadcode trùng sẽ xóa) + thêm `FashionItemID` `fashionItemId`; Brand đang dùng chung `BrandItemUseCase` cho handler+contract (`brand_contract_impl.go:16` + `brand_portal_handler.go:17`) → **tạo mới** `BrandItemContractUseCase` ở `item/contractuc/` pure (không `fashionContract`/`userBriefContract`, chỉ `brandRepo`/`brandItemRepo`/`customerRepo`/`benefitUC`), giữ nguyên `BrandItemUseCase` cho handler. `Category` `1/1` + `Subscription` `8/8` đã pure method-level → không sửa.

**Rationale**:
- Wardrobe đã có contract usecase riêng — sửa trực tiếp ít churn.
- Brand shared: sửa `BrandItemUseCase` trực tiếp sẽ làm handler `GetBrandItemForStaff`...`ListBrandProductsForCustomer` mất `FashionItem` → break. Tách riêng thì handler giữ enrich, contract pure — đúng đề xuất "viết 1 contract usecase riêng cho contract đang dùng chung".
- Tạo `IPure*` cho wardrobe/category/subscription là overengineering (đã pure/tách) — chỉ brand cần tạo vì shared.
- 3/7 method leaky wardrobe trùng deadcode → xóa luôn, chỉ thuần hóa 4 alive.

**Alternatives considered**:
- Sửa trực tiếp `BrandItemUseCase` — loại vì handler mất enrich (đã rà `brand_portal_handler.go:17`).
- Tạo `IPure*` cho wardrobe/category — loại vì overengineering.

## 1b. Dead wiring thứ 2

**Decision**: Xóa `SubscriptionUseCase.userBriefContract` (`subscription_uc.go:41,59,76` — 0 call) cùng `WardrobeItemUseCase.aiService` — phát hiện khi rà shared usecase toàn 9 module.

**Rationale**: Inject không dùng → dead wiring, dọn cùng US4.

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

**Decision**: Wardrobe: bỏ `fashionContract`/`userSubContract` khỏi `WardrobeContractUseCase`; Brand: thêm ProviderSet `item/contractuc.NewBrandItemContractUseCase` + bind `IBrandItemContractUseCase` cho `BrandContract`, giữ `BrandItemUseCase` provider cho handler; Dead wiring: xóa `wardrobe aiService` + `subscription userBriefContract` providers; regenerate `wire_gen.go`.

---

## 5. Constitution §II Update + Doc

**Decision**: Viết `docs/system-design/contract-purity-pattern.md` trước, sau đó cập nhật `.specify/memory/constitution.md` §II thêm "Pure Contract Usecase Principle" và link tới doc (clarify #2 + #3).

---

## 6. Performance NFR-001 (dự đoán)

**Decision**: Benchmark **dự đoán** bằng đếm `fashionContract` trong code review (không chạy thật) cho 4 flow bị ảnh hưởng `GetWardrobeItemsByUserID`/`GetActive...`/`GetFailed...`/`GetStale...`. Nếu dự đoán tăng >10% do caller enrich thêm thì gộp batch `ListFashionItems` (1 call với nhiều IDs) hoặc cache trong cùng request.

---

## 7. Tài liệu tham khảo

- Audit method-level 2026-08-27 — wardrobe 7/15 leaky, brand 5/9 leaky, subscription/category đã pure.
- `docs/temp/detour-call-audit/02_wardrobe_fashion.md`
- `docs/system-design/design-pattern.md`
- `.specify/memory/constitution.md` §II, §IV
