# Contract: hygiene-scan CLI

**Feature**: 017-codebase-hygiene-refactor | **Date**: 2026-08-30

Contract cho dev tool nội bộ `internal/tools/hygiene-scan` (thuần đọc source, ghi báo cáo ra
`specs/017-codebase-hygiene-refactor/reports/`). Đây là "giao diện" duy nhất tool cam kết với
người dùng (developer) — không có API/endpoint runtime.

## Invocation

```bash
# Cách chuẩn (Makefile target mới):
make hygiene-scan

# Tương đương gọi trực tiếp:
go run ./internal/tools/hygiene-scan [flags]
```

## Flags

| Flag | Bắt buộc | Mặc định | Ý nghĩa |
|------|----------|----------|---------|
| `-root` | Không | `.` (repo root tự dò qua go.mod) | Thư mục gốc để scan |
| `-out` | Không | `specs/017-codebase-hygiene-refactor/reports/` | Thư mục ghi snapshot + report |
| `-baseline` | Không | snapshot JSON mới nhất trong `-out` (nếu có) | File snapshot JSON để so sánh trước/sau |
| `-format` | Không | `all` | `md` \| `json` \| `all` |

## Exit codes

| Code | Ý nghĩa |
|------|---------|
| 0 | Scan thành công (dù còn vi phạm — tool chỉ báo cáo, không chặn) |
| 2 | Lỗi tool (không đọc được root, không ghi được out) |

## Output contract

Mỗi lần chạy ghi (theo `-out`):

1. `snapshot-<YYYYMMDD-HHMMSS>.json` — schema theo [data-model.md §3](../data-model.md);
   trường `metrics` phải đủ để kiểm chứng SC-001/SC-002 bằng 1 phép đọc JSON.
2. `scan-report.md` — báo cáo markdown, thứ tự section bắt buộc:
   1. `## Summary` — tổng vi phạm theo severity + so với baseline (↑/↓/=");
   2. `## By Rule` — bảng 8 rule H1–H8 (fail/warning/needs-confirmation);
   3. `## By Module` — bảng theo module;
   4. `## Violations` — danh sách chi tiết sort theo severity rồi file;
   5. `## Exclusions` — các thư mục/pattern loại trừ kèm lý do;
   6. `## Needs Confirmation` — dead code/archived cần xác nhận tay.

Markdown phải render được trên GitHub (bảng GFM, không HTML).

## Behavioral guarantees

- Tool KHÔNG bao giờ ghi/sửa file source nào — chỉ đọc source và ghi đúng 2 file output trên.
- Kết quả phải deterministic trên cùng một cây source (sort ổn định; không timestamp lẫn vào
  nội dung violations).
- Generated files (`// Code generated ... DO NOT EDIT.`, `api/swagger/**`) không bao giờ xuất
  hiện trong `violations`, chỉ xuất hiện trong `exclusions`.
- `provider.go`, `internal/di/**`, `internal/bootstrap/**` được miễn rule "bỏ qua tầng"
  (nơi dựng Wire) nhưng KHÔNG được miễn rule import chéo module bị cấm.
- Tất cả vi phạm phải mang RuleID khớp bảng rule trong data-model.md (SC-005: quy chiếu được
  về điều luật trong `docs/development/code-hygiene.md` sau khi doc được chốt).
