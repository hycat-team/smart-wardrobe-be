# Specification Quality Checklist: Xử lý tech debt cross-module & tách schema theo module

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-08-01
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- Feature gồm 3 phase: Phase 1 đánh giá/nghiên cứu (tiền đề) → Sprint 1 xử lý tech debt → Sprint 2 tách schema vật lý.
- User Story 1 (đánh giá) là phase bắt buộc làm trước, kết quả quyết định cách triển khai 2 sprint code.
- Tên kỹ thuật (tên file, bảng, FK, contract) được giữ vì bản thân feature mô tả việc xử lý các thành phần kỹ thuật đó — tương tự spec 010.
- Đã chạy clarify (session 2026-08-01), 5 câu hỏi được chốt và ghi vào `## Clarifications`: (1) identity tách schema riêng + giữ FK xuyên schema tới `users`; (2) cắt FK xuyên module trừ `users`, enforce qua contract; (3) không phân quyền DB-level, 1 connection chung; (4) migration thuần `SET SCHEMA` viết Down, migration đổi dữ liệu dùng Backup+Restore — đã đối chiếu `closy_db-schema.sql` (42 bảng + sequence ở `public`, thuần SET SCHEMA + drop/add constraint, không đổi dữ liệu) → migration tách schema viết Down đầy đủ; (5) eventual consistency qua outbox cho critical path worker.
- Đã chạy analyze và sửa toàn bộ 8 issue: I1 (số FK thống nhất: 9 phi-users cắt + ~30 giữ), I2 (method `GetStaleProcessingInfo`, không trùng `IWardrobeContract`), I3 (phân định đồng bộ vs outbox), U1 (chốt method `IFashionDashboardReader`), U2 (verify FK giữ), N1 (SC-007 latency), L1 (42 bảng), L2 (Phase 2.3). Re-analyze chỉ còn 2 LOW (L3 title Task 1.2.1, L4 mô tả 4 UPDATE hiện trạng) — đã xử lý.
- **Sprint 1 hoàn tất (2026-08-01)**: T001-T023 xong. Build + vet + test toàn bộ pass. Scan xác nhận: wardrobe không còn JOIN/UPDATE `fashion_items`, fashion không query `wardrobe_items`, dashboard không raw query bảng cross-module. Wire DI đã regenerate (idempotent). Còn T024-T034 (Sprint 2 — tách schema vật lý) chưa làm.
