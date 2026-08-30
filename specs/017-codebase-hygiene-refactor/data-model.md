# Data Model: Chuẩn hóa Codebase — Scan & Refactor (017)

**Date**: 2026-08-30 | **Feature**: [spec.md](spec.md)

Feature này không có dữ liệu runtime/DB. "Key Entities" dưới đây là mô hình dữ liệu nội bộ của
tool `hygiene-scan` và schema của các artifact báo cáo.

## 1. ChecklistRule (rule engine trong tool)

| Trường | Kiểu | Ghi chú |
|--------|------|---------|
| ID | string | `H1`..`H8` (8 nhóm hygiene theo spec FR-002) |
| Category | enum | file-length, struct-per-file, folder-org, boundary, dead-code, duplication, naming, error-handling |
| Severity | enum | `fail` / `warning` / `needs-confirmation` |
| AppliesTo | predicate | loại file áp dụng (production/test/generated/excluded) |
| Threshold | int | ngưỡng định lượng (nếu có) |

**Ngưỡng đã chốt (Session 2026-08-30)**:

| Rule | Warning | Fail | Ghi chú |
|------|---------|------|---------|
| H1 file-length (production) | >500 dòng | >800 dòng | Loại trừ generated |
| H1 file-length (test) | >750 dòng | >1200 dòng | Nới +50% |
| H1 func-length | >80 dòng | — (warning only) | Per-function |
| H2 struct-per-file | >2 struct không cùng mối quan tâm | — | "Cùng mối quan tâm" = cùng phục vụ một use case trong file; miễn table_name.go (boilerplate GORM) |
| H4 boundary / layering | — | import cấm, bỏ qua tầng | Bảng Allowed/Prohibited Constitution §II/§IV; `provider.go`, `di`, `bootstrap` được miễn rule bỏ qua tầng |
| H5 dead-code | — | — | Mọi kết quả gắn `needs-confirmation` (FR-008) |
| H6 duplication | — (heuristic đã nghỉ) | — | Audit thủ công — same-name-file heuristic nghỉ hổ 2026-08-30, tool không tự emit H6 |
| H7 naming | lệch chuẩn | — | Heuristic theo design-pattern.md |
| H8 error-handling | lệch chuẩn | — | Heuristic apperror / %w / WrapHandler. Miễn: sentinel package-level `var ErrXxx = errors.New(...)` (tên biến tiền tố `Err`) và `fmt.Errorf` message-only (không có argument kiểu error) — chỉ flag `fmt.Errorf` có error argument thiếu `%w` (FR-003) |

## 2. Violation (một vi phạm)

| Trường | Kiểu | Ràng buộc |
|--------|------|-----------|
| RuleID | string | Phải trỏ về ChecklistRule tồn tại (SC-005) |
| Location | {file, line?, symbol?} | File path tương đối repo |
| Severity | enum | fail / warning / needs-confirmation |
| Detail | string | Mô tả ngắn, tiếng Anh (chuẩn nội bộ) |
| Status | enum | open / refactored / accepted-with-reason |

Trạng thái `accepted-with-reason` dùng cho dead code cần xác nhận tay và archived module
(chỉ ghi nhận), và cho finding advisory khớp keep-list
`internal/tools/hygiene-scan/accepted.json` (FR-011) — finding vẫn được liệt kê trong báo cáo
nhưng không tính vào tổng đếm ở Summary/By Rule/By Module.

## 3. ScanRun (một lần scan) — schema snapshot JSON

```text
{
  "run_id": "YYYYMMDD-HHMMSS",
  "generated_at": RFC3339,
  "scope": {
    "scanned_files": int,
    "exclusions": ["api/swagger/**", "bin/**", ...],   // lý do loại trừ kèm theo
    "test_files": int
  },
  "summary": {
    "by_rule":  { "<H1>": {"fail": n, "warning": n, "needs_confirmation": n}, ... },
    "by_module": { "<module-name>": {"fail": n, "warning": n}, ... }
  },
  "violations": [ Violation... ],
  "acceptedKeeps": int,   // số finding khớp keep-list (FR-011); optional
  "metrics": {
    "files_over_500": int, "files_over_800": int,
    "cross_module_prohibited_imports": int,
    "layer_skips": int
  }
}
```

**Artifact trong `specs/017-codebase-hygiene-refactor/reports/`**:

- `snapshot-<run_id>.json` — snapshot máy đọc được (schema trên), input cho so sánh baseline.
- `scan-report.md` — báo cáo người đọc: tổng kết theo rule/module, danh sách vi phạm theo
  severity, phần "So với lần trước" (khi có `-baseline`), phần loại trừ, phần dead code
  needs-confirmation.

## 4. HygieneStandard (văn bản chuẩn — `docs/development/code-hygiene.md`)

Không phải dữ liệu tool, nhưng là deliverable bắt buộc (FR-010). Cấu trúc bắt buộc của doc:

1. Mục đích + phạm vi áp dụng (ai áp, áp cho phần nào của repo)
2. 8 mục checklist với ngưỡng bảng ở trên (nguồn sự thật ngưỡng)
3. Quy tắc chạy scan (`make hygiene-scan`) và ý nghĩa severity
4. Ví dụ trước/sau cho từng loại refactor điển hình (god-file, import vi phạm)
5. Tham chiếu ngược: constitution §II/§IV, design-pattern.md, spec 017

Quan hệ tài liệu: constitution = kiến trúc; design-pattern.md = conventions implementation;
code-hygiene.md = ngưỡng hygiene + quy trình scan (không định nghĩa lại quy ước đã có ở
design-pattern.md, chỉ tham chiếu).
