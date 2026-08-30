# Quickstart: Chuẩn hóa Codebase — Scan & Refactor (017)

**Date**: 2026-08-30 | **Feature**: [spec.md](spec.md) | **Contract**: [hygiene-scan-cli.md](contracts/hygiene-scan-cli.md)

Hướng dẫn xác minh end-to-end feature này sau khi implement. Không chứa code implementation —
chi tiết task nằm trong `tasks.md`.

## Prerequisites

- Go 1.25 theo go.mod; chạy được `make check` hiện hành (xem [local-setup](../../../docs/development/local-setup.md)).
- Repo ở nhánh `017-codebase-hygiene-refactor`.
- Không cần DB/Redis/RabbitMQ — scan thuần tĩnh.

## Scenario 1: Chạy scan lần đầu và kiểm tra báo cáo

1. `make hygiene-scan`
2. **Expected**: command exit 0, hoàn tất < 30 giây; sinh đúng 2 file trong
   `specs/017-codebase-hygiene-refactor/reports/`:
   `snapshot-<run_id>.json` và `scan-report.md`.
3. Mở `scan-report.md`, kiểm tra:
   - Section `## Exclusions` liệt kê `api/swagger/**` (generated) — và **không có** vi phạm nào
     trỏ vào `api/swagger/`.
   - Section `## By Rule` có đủ 8 rule H1–H8.
   - Ít nhất 1 vi phạm H1 fail (kỳ vọng: `internal/modules/brand/presentation/handler/brand_loyalty_handler.go`
     ~958 dòng > ngưỡng 800).
4. Kiểm chứng JSON: đọc `metrics.files_over_800` ≥ 1 và
   `metrics.cross_module_prohibited_imports` + `metrics.layer_skips` đủ phục vụ SC-001/SC-002.

## Scenario 2: Chỉnh xác của scan (SC-003 ≥ 90%)

1. Chọn ngẫu nhiên 20 vi phạm trong `## Violations` (hoặc toàn bộ nếu < 20).
2. Đối chiếu từng vi phạm với code thực tế (ngưỡng trong
   `docs/development/code-hygiene.md`).
3. **Expected**: ≥ 18/20 xác thực được → SC-003 pass. Kết quả audit ghi vào cuối `scan-report.md`.

## Scenario 3: Scan lặp lại và so sánh baseline (FR-004)

1. Sau một phase refactor, chạy lại `make hygiene-scan`.
2. **Expected**: snapshot JSON mới có run_id mới; `scan-report.md` section `## Summary` hiển thị
   so sánh với baseline lần trước (↑/↓/" cho từng nhóm); vi phạm đã refactor ở phase đó giảm về 0.

## Scenario 4: Refactor không đổi hành vi (FR-005)

1. Sau mỗi phase refactor: `make check && make test-race`.
2. **Expected**: toàn bộ pass; không sửa assertion test nào (chỉ import path khi bắt buộc).
3. `make wire` không phát hiện thay đổi DI (refactor không đụng DI).

## Scenario 5: Chuẩn chính thức đã chốt (FR-010)

1. Mở `docs/development/code-hygiene.md` — có đủ: 8 mục checklist + bảng ngưỡng đã chốt
   (500/800, 80, test 750/1200), hướng dẫn `make hygiene-scan`, ví dụ trước/sau.
2. `docs/system-design/design-pattern.md` và `AGENTS.md` có tham chiếu tới doc này.
3. Mọi vi phạm trong báo cáo mang RuleID khớp điều luật trong doc (SC-005).

## Definition of Done (feature level)

- [ ] SC-001: `metrics.files_over_800 == 0` (không tính generated).
- [ ] SC-002: `cross_module_prohibited_imports == 0 && layer_skips == 0`.
- [ ] SC-003: audit 20 mẫu ≥ 90% đúng, ghi trong báo cáo.
- [ ] SC-004: mọi phase pass `make check && make test-race`; 0 thay đổi hành vi.
- [ ] SC-005: 100% vi phạm quy chiếu được RuleID → code-hygiene.md.
- [ ] SC-006: giải thích được vị trí ≥ 5 chức năng đại diện chỉ qua tên folder/file.
