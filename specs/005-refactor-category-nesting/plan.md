# Implementation Plan: Refactor Category Nesting

**Branch**: `005-refactor-category-nesting` | **Date**: 2026-07-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/005-refactor-category-nesting/spec.md`

## Summary

This feature refactors the flat category fields (`CategoryID`, `CategoryName`, `CategorySlug`) inside `FashionItemRes` and `FashionItemBriefRes` to a nested object `Category *CategoryBriefRes`. It also eliminates the duplicate root-level `Category` fields on `WardrobeItemRes`/`SearchWardrobeItemRes` so that all modules reuse the nested category design from the `fashion` module contract.

## Technical Context

**Language/Version**: Go (v1.25.+)

**Primary Dependencies**: Gin Web Framework, Google Wire

**Storage**: PostgreSQL (GORM), Elasticsearch client

**Testing**: go test / make test

**Target Platform**: Linux / Windows local env

**Project Type**: web-service (modular monolith)

**Performance Goals**: N/A (no network/db changes, purely DTO refactoring)

**Constraints**: Complies with Project Constitution layer boundaries.

**Scale/Scope**: Refactoring affects multiple usecases and mappers across `fashion`, `wardrobe`, and `brand` modules.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I (Pragmatic Modular Monolith)**: Passed. Modules continue to use Google Wire and keep their boundaries.
- **Principle II (Strict Module Boundaries & DTO Contracts)**: Passed. Cross-module communications continue to go through public contract DTOs. Tái sử dụng `CategoryBriefRes` giúp thống nhất contract, loại bỏ struct DTO tự định nghĩa dư thừa ở module wardrobe và brand.
- **Principle IV (Clean Architecture)**: Passed. Mappers will be updated within their respective module `application/mapper` folders.

## Project Structure

### Documentation (this feature)

```text
specs/005-refactor-category-nesting/
├── spec.md              # Feature specification
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── checklists/
    └── requirements.md  # Specification Quality Checklist
```

### Source Code (repository root)

```text
internal/modules/
├── fashion/
│   ├── application/
│   │   ├── dto/contract_dtos.go
│   │   └── usecase/
│   │       ├── fashion_item/fashion_item_uc.go
│   │       ├── worker/worker_uc.go
│   │       └── ai/recommendation/...
│   └── presentation/contract/fashion_contract_impl.go
├── wardrobe/
│   ├── application/
│   │   ├── dto/wardrobe.go
│   │   └── mapper/
│   │       ├── wardrobe.go
│   │       └── search_mapper.go
│   └── infrastructure/search/wardrobe_helper.go
└── brand/
    ├── application/
    │   ├── dto/brand_item.go
    │   └── mapper/brand_item_mapper.go
```

**Structure Decision**: Single modular monolith project structure. Files are refactored in-place under `internal/modules/`.

## Proposed Changes

### Module `fashion` (Nguồn dữ liệu gốc)

#### [MODIFY] [contract_dtos.go](../../internal/modules/fashion/application/dto/contract_dtos.go)
- Khai báo thêm struct `CategoryBriefRes`.
- Cấu trúc lại `FashionItemRes` và `FashionItemBriefRes` sử dụng `Category *CategoryBriefRes` lồng thay vì các trường phẳng.

#### [MODIFY] [fashion_contract_impl.go](../../internal/modules/fashion/presentation/contract/fashion_contract_impl.go)
- Sửa đổi hàm ánh xạ để gán đối tượng danh mục lồng.

#### [MODIFY] [fashion_item_uc.go](../../internal/modules/fashion/application/usecase/fashion_item/fashion_item_uc.go)
- Cập nhật logic trả về chi tiết trang phục.

#### [MODIFY] AI Recommendation Engine & Usecases
- [workflow_helpers.go](../../internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go)
- [synthesis/prompt.go](../../internal/modules/fashion/application/usecase/ai/recommendation/synthesis/prompt.go)
- [synthesis/synthesis_test.go](../../internal/modules/fashion/application/usecase/ai/recommendation/synthesis/synthesis_test.go)
- [ranking/scoring.go](../../internal/modules/fashion/application/usecase/ai/recommendation/ranking/scoring.go)
- [ranking/ranking.go](../../internal/modules/fashion/application/usecase/ai/recommendation/ranking/ranking.go)
- [ranking/scoring_test.go](../../internal/modules/fashion/application/usecase/ai/recommendation/ranking/scoring_test.go)
- [ranking/pool_expansion.go](../../internal/modules/fashion/application/usecase/ai/recommendation/ranking/pool_expansion.go)
- Cập nhật truy cập từ trường phẳng sang nested category (`.Category.Name` / `.Category.Slug`).

#### [MODIFY] [worker_uc.go](../../internal/modules/fashion/application/usecase/worker/worker_uc.go)
- Cập nhật logic xử lý sau AI phân tích ảnh.

---

### Module `wardrobe` (Tiêu thụ dữ liệu)

#### [MODIFY] [wardrobe.go DTO](../../internal/modules/wardrobe/application/dto/wardrobe.go)
- Loại bỏ `Category` ở root của `WardrobeItemRes`/`SearchWardrobeItemRes`. Sửa đổi struct `FashionItemRes` nội bộ.

#### [MODIFY] [outfit.go DTO](../../internal/modules/wardrobe/application/dto/outfit.go)
- Cập nhật cấu trúc lồng.

#### [MODIFY] Mappers & Search helpers
- [wardrobe.go mapper](../../internal/modules/wardrobe/application/mapper/wardrobe.go)
- [outfit.go mapper](../../internal/modules/wardrobe/application/mapper/outfit.go)
- [search_mapper.go](../../internal/modules/wardrobe/application/mapper/search_mapper.go)
- [wardrobe_helper.go](../../internal/modules/wardrobe/infrastructure/search/wardrobe_helper.go)
- Đồng bộ hóa chuyển đổi sang dạng lồng và xử lý parse Elasticsearch hits.

---

### Module `brand` (Tiêu thụ dữ liệu)

#### [MODIFY] [brand_item.go DTO](../../internal/modules/brand/application/dto/brand_item.go)
- Cập nhật cấu trúc `FashionItemRes` nội bộ.

#### [MODIFY] [brand_item_mapper.go](../../internal/modules/brand/application/mapper/brand_item_mapper.go)
- Đồng bộ ánh xạ dạng lồng.

