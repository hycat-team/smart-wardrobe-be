# Implementation Plan: Audit Response Structs & BriefRes Pattern

**Branch**: `013-briefres-audit` | **Date**: 2026-08-22 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/013-briefres-audit/spec.md`

## Summary

Feature 013 gồm 2 pha:
1. **Audit** (đã xong ở plan): quét toàn diện response DTOs trong 9 modules, dispatch 9 sub-agents (1/module) tạo report tại `reports/`, tổng hợp 12 BriefRes mới đề xuất + 6 điểm bảo mật.
2. **Implement** (giai đoạn tasks/implement): tạo các struct BriefRes mới, trim field overdelivery, bổ sung nested data underdelivery, cập nhật mapper + handler + swagger theo mức ưu tiên. KHÔNG đổi DB schema, không đổi request DTO. Cuối cùng tạo `frontend-note.md` (breaking change policy đã chốt).

## Technical Context

**Language/Version**: Go (v1.25+) / Gin framework

**Primary Dependencies**: GORM (repository), Google Wire (DI), Swag (swagger)

**Storage**: PostgreSQL (KHÔNG thay đổi schema trong feature này)

**Testing**: `make test` (Go unit tests), `make check` (fmt + go.mod tidy + wire + swagger)

**Target Platform**: Backend Linux server (single deployable)

**Project Type**: Web-service (Gin modular monolith), feature này là analysis/documentation

**Performance Goals**: Giảm JSON payload cho frontend (không đo trực tiếp trong feature audit; benchmark sau khi refactor)

**Constraints**:
- KHÔNG sửa database schema, entity GORM, request DTO
- Chỉ response DTO + mapper
- Struct dùng chung admin/portal + user → tách user-variant, không sửa struct gốc
- Cross-module chỉ import `application/dto` (constitution §II)

**Scale/Scope**: 9 modules, ~100 response structs được audit; 12 BriefRes mới đề xuất; ~50 endpoints bị ảnh hưởng (liệt kê trong contracts/)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Trạng thái | Ghi chú |
|------|-----------|---------|
| §I Modular Monolith — không cross-module DB joins | ✅ PASS | Đề xuất chỉ dùng DTO + contract, không join entity |
| §II Module Boundaries — cross-module qua contract/DTO | ✅ PASS | UserBriefRes đặt identity, các module import `application/dto` (được phép) |
| §IV Clean Architecture — mapper trong application/mapper | ✅ PASS | Mọi mapper đề xuất đặt `application/mapper/` |
| §V Error Handling — apperror + handler thin | ✅ PASS | Không thay đổi error handling |
| §VI Naming — `Get...` prefix, camelCase JSON | ✅ PASS | Struct BriefRes theo convention `<Entity>BriefRes`, JSON camelCase |
| Language — Swagger/message tiếng Việt, code English | ✅ PASS | Report/spec tiếng Việt; code identifiers English |
| KHÔNG sửa DB schema | ✅ PASS | Không migration |

**Re-check sau Phase 1**: Không có vi phạm. Complexity Tracking không cần điền.

## Project Structure

### Documentation (this feature)

```text
specs/013-briefres-audit/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output — tổng hợp findings 9 reports
├── data-model.md        # Phase 1 output — 12 BriefRes đề xuất + nested relationships
├── quickstart.md        # Phase 1 output — validation guide
├── contracts/
│   └── affected-api-contracts.md   # Phase 1 output — API bị ảnh hưởng
├── reports/             # Phase 0 output — 9 file, 1/module
│   ├── admin.md
│   ├── aiexecutor.md
│   ├── brand.md
│   ├── dashboard.md
│   ├── fashion.md
│   ├── identity.md
│   ├── notification.md
│   ├── subscription.md
│   └── wardrobe.md
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── frontend-note.md     # TẠO Ở CUỐI implement + verify (KHÔNG tạo ở plan)
```

### Source Code (repository root)

Feature này không thay đổi source code ở giai đoạn plan (audit). Giai đoạn implement (tasks) sẽ thay đổi các file sau theo module:

```text
internal/modules/
├── brand/application/dto/          # B1,B2,B4,B5,B6 + sửa response structs
├── brand/application/mapper/       # MapBrandBrief, MapBenefitBrief, ...
├── identity/application/dto/       # B3 UserBriefRes
├── identity/application/mapper/    # MapUserBrief
├── wardrobe/application/dto/       # B7,B8
├── fashion/application/dto/        # B9,B10
├── subscription/application/dto/   # B11
├── aiexecutor/application/dto/     # B12
└── * /presentation/handler/        # đổi response struct trong handler/swagger
```

**Structure Decision**: Giai đoạn plan (audit) không thay đổi cấu trúc source code — file docs nằm trong `specs/013-briefres-audit/` theo convention speckit. Giai đoạn implement sẽ tạo/refactor các file DTO + mapper + handler theo bảng trên.

## Complexity Tracking

Không có vi phạm Constitution → không cần điền.