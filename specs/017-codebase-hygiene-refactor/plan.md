# Implementation Plan: Chuẩn hóa Codebase — Scan & Refactor theo Checklist Hygiene

**Branch**: `017-codebase-hygiene-refactor` | **Date**: 2026-08-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/017-codebase-hygiene-refactor/spec.md`

## Summary

Xây dựng một công cụ scan tĩnh (`internal/tools/hygiene-scan`) chạy được lặp lại để phát hiện vi
phạm hygiene trong codebase Go (~645 file, 9 module) theo 8 mục checklist đã chốt trong spec
(ngưỡng: file >500 warning / >800 fail; hàm >80; file test nới lỏng 50%), xuất báo cáo markdown +
JSON snapshot vào `specs/017-codebase-hygiene-refactor/reports/`. Sau đó refactor theo từng phase
(giữ nguyên hành vi, mỗi phase được xác minh bằng `make check`) để đưa các vi phạm fail về 0;
commit chỉ thực hiện một lần khi toàn bộ tasks.md hoàn tất, đồng thời chốt chuẩn
thành doc mới `docs/development/code-hygiene.md` (tham chiếu từ `design-pattern.md` và `AGENTS.md`).

## Technical Context

**Language/Version**: Go 1.25 (đã pinned trong go.mod; tool dùng thư viện chuẩn `go/parser`,
`go/ast`, không thêm dependency nặng)

**Primary Dependencies**: Thư viện chuẩn Go (`go/parser`, `go/ast`, `go/token`) cho phân tích tĩnh;
`golang.org/x/tools/cmd/deadcode` (chạy bằng `go run`) làm nguồn tham vấn cho dead code. Không
thêm dependency mới vào runtime của service.

**Storage**: N/A (tool đọc source, ghi báo cáo ra file; không đụng DB)

**Testing**: `go test ./...` + `make check` (fmt-check, tidy-check, generate-check, vet, test,
build); tool scan có unit test riêng cho rule engine

**Target Platform**: Windows (dev) + Linux (CI) — tool phải chạy được trên cả hai qua `go run`

**Project Type**: Modular monolith (Gin) + developer CLI tool (`internal/tools/hygiene-scan`,
theo pattern `swagger-split` hiện có, chạy qua target Makefile mới `make hygiene-scan`)

**Performance Goals**: Scan toàn bộ codebase (<700 file Go) hoàn tất < 30 giây trên máy dev thông
thường

**Constraints**: Tool chỉ đọc source, không tự sửa code; refactor thực hiện thủ công theo phase
và mỗi phase phải pass `make check` (bao gồm test-race theo git-workflow) trước khi chuyển
phase; commit chỉ thực hiện MỘT LẦN khi toàn bộ tasks.md hoàn tất (yêu cầu developer
2026-08-30); không thay đổi hành vi runtime; không thêm dependency runtime

**Scale/Scope**: ~645 file Go, 9 module nghiệp vụ + shared; 10 file production đã vượt 500 dòng
(1 file fail >800: `brand_loyalty_handler.go` 958 dòng)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Áp dụng cho feature này | Trạng thái |
|-----------|------------------------|------------|
| §I Pragmatic Modular Monolith | Tool không thuộc runtime DI; chỉ là dev tool dưới `internal/tools/` như `swagger-split` | ✅ Pass |
| §II Strict Module Boundaries & DTO Contracts | Rule scan chéo module được mã hóa đúng theo bảng Allowed/Prohibited trong constitution (entities, repositories, persistence bị cấm chéo module); refactor vi phạm đi qua contract/DTO, không tạo exception ad-hoc | ✅ Pass |
| §III Async Event-Driven | N/A — không đụng messaging/outbox | ✅ N/A |
| §IV Clean Architecture Layering | Rule phân lớp mã hóa đúng `presentation → application → domain ← infrastructure`; refactor giữ nguyên hướng phụ thuộc; handler vẫn mỏng sau khi tách file | ✅ Pass |
| §V Transaction & Error Handling | Rule error-handling kiểm tra dùng `apperror` factory; refactor không đổi cách mapping lỗi | ✅ Pass |
| §VI AI Pipeline | N/A — không đụng AI pipeline | ✅ N/A |
| §VII Design Patterns & Conventions | `docs/development/code-hygiene.md` mới phải nhất quán với `design-pattern.md` (mapper nằm trong `application/mapper`, suffix `_helper.go`, đặt tên `Get...`); design-pattern.md giữ vai trò nguồn chuẩn implementation, code-hygiene.md chỉ thêm ngưỡng hygiene và tham chiếu chéo | ✅ Pass |

**Post-design re-check (sau Phase 1)**: Không phát sinh vi phạm mới — tool scan thuần đọc, không
import module nào của runtime; báo cáo nằm trong `specs/`, không ảnh hưởng build runtime.

## Project Structure

### Documentation (this feature)

```text
specs/017-codebase-hygiene-refactor/
├── plan.md              # This file
├── research.md          # Phase 0 output: quyết định kỹ thuật (tool, deadcode, report format)
├── data-model.md        # Phase 1 output: mô hình dữ liệu của scan report & rule
├── quickstart.md        # Phase 1 output: hướng dẫn chạy scan & đối chiếu
├── contracts/           # Phase 1 output: contract CLI của hygiene-scan + schema báo cáo
│   └── hygiene-scan-cli.md
├── checklists/          # hygiene.md (21 mục), requirements.md (16 mục)
├── reports/             # Đầu ra khi chạy tool (scan-report.md + snapshot JSON) — tạo ở Phase implement
└── tasks.md             # Phase 2 output (/speckit-tasks) — CHƯA tạo
```

### Source Code (repository root)

```text
internal/tools/hygiene-scan/     # Dev tool scan tĩnh thuần đọc (pattern: swagger-split)
├── main.go                      # CLI entry, flag parsing, orchestration
├── rules.go                     # 8 nhóm rule hygiene + ngưỡng đã chốt
├── rules_test.go                # Unit test rule engine (fixture file giả lập)
├── report.go                    # Render markdown + JSON snapshot, so sánh baseline
└── report_test.go

docs/development/code-hygiene.md # Chuẩn hygiene chính thức (checklist + ngưỡng + ví dụ trước/sau)
Makefile                         # Thêm target `hygiene-scan` + mục trong help
internal/modules/**              # Refactor theo phase (tách file god-file, sửa import vi phạm)
```

**Structure Decision**: Tool đặt tại `internal/tools/hygiene-scan` theo đúng pattern hiện có
(`internal/tools/swagger-split` chạy bằng `go run` trong Makefile) — không tạo project riêng,
không đụng Wire DI, không ảnh hưởng binary runtime. Báo cáo (artifact theo thời điểm) nằm trong
`specs/017-.../reports/` theo quyết định clarify; chuẩn chính thức (ổn định) nằm trong
`docs/development/`.

## Complexity Tracking

> Không có vi phạm constitution cần justify.
