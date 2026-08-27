# Kế hoạch Triển khai: Refactor Contract Purity & Xử Lý Detour Call + Deadcode

**Branch**: `015-contract-purity-refactor` | **Ngày**: 2026-08-27 | **Spec**: [spec.md](spec.md)

**Input**: Đặc tả tính năng từ `specs/015-contract-purity-refactor/spec.md`

---

## Tóm tắt

Audit method-level toàn bộ 8 module phát hiện: 7 chuỗi detour call (wardrobe ↔ fashion), 2 contract leaky (wardrobe **4/8 alive leaky** — 7/15 nếu tính 3 deadcode trùng đã tách; brand 5/9 method **dùng chung handler+contract** nên tách `BrandItemContractUseCase` riêng), 12 method deadcode + **2 dead wiring** (wardrobe `aiService` + subscription `userBriefContract`). Mục tiêu: wardrobe sửa trực tiếp đã tách + brand tạo mới `BrandItemContractUseCase` ở `item/contractuc/` + thêm `FashionItemID` `fashionItemId` vào `WardrobeItemRes`, ép caller tự enrich bằng `fashionItemRepo` (đối với caller nội bộ fashion) hoặc `fashionContract` (đối với caller ngoài module) + `sliceutils.IndexBy`/`GroupBy` để tránh `nil`. Triển khai tuần tự P1→P1→P2→P3, trong đó US2 (Pure Contract) + US1 (Caller Enrich) tạo thành một khối nguyên tử (Atomic MVP Core) để loại trừ rủi ro nil pointer. Giữ 100% API surface. Viết `docs/system-design/contract-purity-pattern.md` trước (T002), cập nhật Constitution §II ref tới doc sau (T029, bump 1.7.0→1.8.0 MINOR).

**Hướng tiếp cận kỹ thuật**: Sửa trực tiếp `WardrobeContractUseCase` đã tách (bỏ 4 alive enrich + 3 deadcode sẽ xóa, thêm `FashionItemID` `fashionItemId` vào DTO/mapper) và **tạo mới** `BrandItemContractUseCase` (`brand/application/usecase/item/contractuc/`) cho brand vì `BrandItemUseCase` đang dùng chung handler+contract — không sửa trực tiếp để tránh handler mất enrich; contract uc mới pure (không `fashionContract`/`userBriefContract`), `BrandContract` chuyển sang inject `IBrandItemContractUseCase`. Update 7 caller fashion **chỉ gọi `fashionItemRepo` nội bộ** sau khi nhận pure result (`FashionItem == nil` nhưng có `FashionItemID` kèm nil guard), xóa 12 method deadcode (3 trùng leaky), sửa **2 dead wiring** (wardrobe `aiService` + subscription `userBriefContract`). Mỗi phase commit riêng, verify bằng `make lint`/`make test`/`make wire`.

---

## Bối cảnh Kỹ thuật

**Ngôn ngữ/Phiên bản**: Go 1.25+ (compiler), Wire cho DI

**Phụ thuộc chính**: Gin (HTTP), GORM + pgvector wrapper, Redis, RabbitMQ (outbox pattern), Elasticsearch, Cloudinary, Swag (Swagger), Goose (migration)

**Lưu trữ**: PostgreSQL (chính), Redis (cache), Elasticsearch (search index)

**Kiểm thử**: `go test` + testify/mock + `make test`, `make lint` (golangci-lint), audit script (shell + ripgrep)

**Nền tảng mục tiêu**: Linux server (Docker, single deployable)

**Loại dự án**: web-service — Modular Monolith REST API

**Mục tiêu hiệu năng**: Tổng số DB call tăng không quá 10% so với trước refactor (NFR-001) qua kiểm chứng tĩnh (Static Query-Count Verification). p95 latency giữ nguyên. Nếu vượt, optimize bằng batch query hoặc cache.

**Ràng buộc**: Không thay đổi API endpoint/DTO response (backward compatible). Triển khai trên branch `015-contract-purity-refactor`, tuần tự theo P1→P1→P2→P3; US2 + US1 đóng vai trò Atomic MVP Core. Không cần feature flag (dùng git revert).

**Quy mô/Phạm vi**: 8 module. Method-level: 2 contract leaky (wardrobe **4/8 alive leaky** — 7/15 nếu tính 3 deadcode trùng đã tách; brand 5/9 **dùng chung** nên tách riêng), category/subscription đã pure — không sửa (verify `1/1` + `8/8` bằng `rg`). 7 chuỗi detour đều qua wardrobe enrich. 12 method deadcode (3 trùng leaky) + **2 dead wiring**. 7 caller fashion cần ép enrich sau thuần hóa (chỉ qua `fashionContract` + `FashionItemID`, tránh nil).

---

## Kiểm tra Hiến pháp

*GATE: Phải pass trước Phase 0. Kiểm tra lại sau Phase 1.*

| Nguyên tắc | Trạng thái | Ghi chú |
|---|---|---|
| **I. Pragmatic Modular Monolith** | ✅ PASS | Không tạo deployable mới. Tất cả vẫn là single Gin service. Mỗi module vẫn sở hữu entities riêng. |
| **II. Strict Module Boundaries & DTO Contracts** | ✅ PASS (cải thiện) | Sửa trực tiếp contract usecase cho thuần (chỉ `wardrobeRepo`/`brandRepo` + mapper), caller enrich qua DTO `IndexBy`/`GroupBy`. Không import entity/repo/persistence cross-module. Sẽ bổ sung "Pure Contract Usecase" vào §II và ref tới `docs/system-design/contract-purity-pattern.md`. |
| **III. Asynchronous Event-Driven Processing** | ✅ PASS | 4 write contract methods (`DeleteWardrobeItem`, `UpdateItemStatus`, `SetFashionItemID`) là các synchronous transactional updates đơn giản của module sở hữu dữ liệu, được bảo toàn dưới dạng pure contract methods an toàn. Không đụng tới RabbitMQ/outbox trong phase này. |
| **IV. Clean Architecture & Strict Layer Dependency** | ✅ PASS | Tuân thủ `presentation → application → domain ← infrastructure`. Sửa trực tiếp contract usecase vẫn ở application layer, không skip layer, không tạo package mới. |
| **V. Standardized Transaction & Error Handling** | ✅ PASS | Giữ nguyên `IUnitOfWork.Execute` qua context. Không thay đổi error handling (apperror). Tách call thành 2 lần gọi riêng biệt nhưng vẫn trong cùng transaction nếu cần (caller chịu trách nhiệm). |
| **VI. AI Pipeline, Cost Control & Local Fallbacks** | ✅ PASS | Không đụng tới AI pipeline/LLM quota. |
| **VII. Design Patterns & Conventions Compliance** | ✅ PASS | Tuân thủ `docs/system-design/design-pattern.md` (chưa phát hiện xung đột). Cần đọc lại trước khi code. |

**Kết luận Gate**: ✅ PASS — Không có vi phạm cần justify. Phase 0 có thể bắt đầu.

---

## Cấu trúc Dự án

### Tài liệu (tính năng này)

```text
specs/015-contract-purity-refactor/
├── plan.md              # File này
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (NOT created by /speckit-plan)
```

### Mã nguồn (repository root)

```text
internal/
├── modules/
│   ├── wardrobe/
│   │   ├── application/usecase/wardrobe/contractuc/contract_uc.go  # sửa trực tiếp 4 alive leaky cho thuần (bỏ fashionContract/userSubContract) + thêm FashionItemID
│   │   ├── application/usecase/wardrobe/item/item_uc_read.go        # xóa dead wiring aiService
│   │   ├── presentation/contract/wardrobe_contract.go               # xóa 7 method deadcode
│   │   └── presentation/contract/wardrobe_contract_impl.go
│   ├── fashion/
│   │   ├── application/usecase/worker/worker_uc.go                  # 2 caller ép enrich sau pure
│   │   ├── application/usecase/ai/chat/workflows.go                 # 2 caller ép enrich
│   │   └── application/usecase/ai/recommendation/workflow_helpers.go # 2 caller ép enrich
│   ├── brand/
│   │   ├── application/usecase/item/usecase.go                      # GIỮ NGUYÊN cho handler (vẫn enrich) — không đụng
│   │   ├── application/usecase/item/helper.go                       # GIỮ NGUYÊN — handler vẫn dùng
│   │   ├── application/usecase/item/contractuc/contract_uc.go       # TẠO MỚI — BrandItemContractUseCase pure (không fashionContract/userBriefContract)
│   │   ├── application/usecase/item/contractuc/mapper.go            # TẠO MỚI — mapper thuần
│   │   ├── presentation/contract/brand_contract.go                  # xóa 2 method deadcode
│   │   └── presentation/contract/brand_contract_impl.go             # chuyển inject sang IBrandItemContractUseCase
│   ├── subscription/
│   │   ├── presentation/contract/subscription_contract.go           # xóa 3 method deadcode (không sửa usecase vì đã pure)
│   │   └── presentation/contract/subscription_contract_impl.go
│   └── ...
├── shared/
│   ├── application/dto/
│   └── infrastructure/db/
├── di/
│   └── wire_gen.go  # regenerate
└── docs/
    └── system-design/
        └── contract-purity-pattern.md  # SẼ TẠO (FR theo clarify)
```

**Quyết định cấu trúc**: Giữ nguyên monolith. Wardrobe đã tách nên **sửa trực tiếp** `WardrobeContractUseCase`; brand đang dùng chung nên **tạo mới** `BrandItemContractUseCase` ở `item/contractuc/` — không đụng `BrandItemUseCase` handler để tránh mất enrich. Thêm 1 doc trong `docs/system-design/`.

---

## Theo dõi Độ phức tạp

> Chỉ điền nếu Kiểm tra Hiến pháp có vi phạm cần biện minh

| Vi phạm | Tại sao cần | Phương án đơn giản hơn bị loại vì |
|---|---|---|
| — | — | — |

Không có vi phạm. Bảng để trống.

---

## Phase 0: Research (Tóm tắt — chi tiết ở research.md)

| Chủ đề | Quyết định |
|---|---|
| Thuần hóa | Wardrobe đã tách: sửa trực tiếp `WardrobeContractUseCase` 4 alive leaky + 3 deadcode (7/15 nếu tính trùng), thêm `FashionItemID` `fashionItemId` vào DTO/mapper; Brand dùng chung handler+contract: **tạo mới** `BrandItemContractUseCase` ở `item/contractuc/` với 5 method pure (bỏ `fashionContract`/`userBriefContract`), giữ nguyên `BrandItemUseCase` cho handler; `Category` `1/1` + `Subscription` `8/8` đã pure — verify bằng `rg` |
| Enrichment ở caller (nil risk) | Caller sau thuần hóa nhận `FashionItem == nil` → phải tự gọi `fashionItemRepo.GetByIDs`/`GetByID` (nếu cùng module fashion) hoặc `fashionContract.ListFashionItems` (nếu khác module) + `sliceutils.IndexBy`/`GroupBy` để stitch, tránh nil |
| Deadcode verification | `rg` để xác nhận không có caller ngoài `_test.go`/`_impl.go`; trong đó 3 method trùng leaky nên xóa luôn |
| Wire | Wardrobe: bỏ `fashionContract`/`userSubContract` khỏi `WardrobeContractUseCase`; Brand: thêm `item/contractuc` ProviderSet + bind `IBrandItemContractUseCase`; Dead wiring: xóa `wardrobe aiService` + `subscription userBriefContract` providers; regenerate `wire_gen.go` |
| Constitution §II update | Thêm đoạn "Pure Contract Usecase Principle" + link tới `docs/system-design/contract-purity-pattern.md` sau khi doc hoàn tất |
| Performance NFR-001 | Dự đoán (không chạy thật) bằng đếm số lần gọi `fashionItemRepo` (nội bộ) hoặc `fashionContract` (ngoài module) trước/sau cho flow `GetWardrobeItemsByUserID`/`GetActive...`/`GetFailed...`/`GetStale...`; nếu dự đoán tăng >10% thì gộp batch `ListFashionItems`/`GetByIDs` |

## Phase 1: Design (Tóm tắt — chi tiết ở data-model.md, contracts/, quickstart.md)

- **Data Model**: Không thêm entity/DB migration. Wardrobe sửa trực tiếp 4 alive + thêm `FashionItemID`; brand **tạo mới** `BrandItemContractUseCase` ở `item/contractuc/` pure + 2 dead wiring (`wardrobe aiService` + `subscription userBriefContract`) — mô tả ở `data-model.md`.
- **Contracts**: API endpoint/DTO không đổi. Thu hẹp contract interface (xóa 12 method, 3 trong số đó trùng leaky) và thuần hóa impl — mô tả ở `contracts/`.
- **Quickstart**: Hướng dẫn verify: `rg` audit 0/0/0, `make lint`, `make test`, `make wire && make build`, check nil enrich — ở `quickstart.md`.
