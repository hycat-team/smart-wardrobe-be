# Implementation Plan: Bàn giao quyền sở hữu DDL (030)

**Branch**: `030-ddl-ownership-handover` | **Date**: 2026-10-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/030-ddl-ownership-handover/spec.md`

## Summary

Viết lại nội dung 11 file migration Account để gỡ toàn bộ DDL `aiexecutor.*`/`dashboard.*` (giữ nguyên tên file + version), dựng chain baseline độc lập đầy đủ ở cây AI (7 bảng, kèm rename `user_ai_policy_grants_check` → `ai_policy_grants_effective_window_check`) và 1 file baseline `dashboard.*` ở cây Commerce, đấu wiring embed cho AI, sửa `Makefile` truyền đúng bảng version từng service, vá init thiếu bảng chết, thêm gate lint 3 luật + target `migration-check` vào `make check`, cập nhật tham chiếu tên cũ + docs/ADR, rehearsal trên DB restore và cutover prod theo cửa sổ full-stop 1–2h do chủ hệ thống thao tác tay với rollback = restore backup. Không thay đổi hành vi nghiệp vụ, API hay DTO.

## Technical Context

**Language/Version**: Go 1.25+ (multi-module workspace `go.work`: `services/account`, `services/commerce`, `services/ai`, `shared`, `contracts`)

**Primary Dependencies**: pressly/goose v3.28.0 (provider API + session advisory lock), GORM, Wire DI, PostgreSQL 17 (pgvector image)

**Storage**: 1 PostgreSQL dùng chung, tách schema theo module; 3 bảng version độc lập (`goose_db_version` 35 dòng / `goose_db_version_commerce` 3 dòng / `goose_db_version_ai` chưa tồn tại — đo thật dev 2026-10-16)

**Testing**: `go test` (integration tests cần `TEST_DATABASE_URL`), rehearsal trên DB `verify_d06_*`, diff schema 2 đường, smoke grant + dashboard, gate đỏ/xanh

**Target Platform**: Docker Compose tự host (dev + prod VPS) — prod song song, dev tuần tự account-trước

**Project Type**: backend microservices (3 service độc lập triển khai) — thay đổi hạ tầng migration, không client-facing

**Performance Goals**: rename constraint là metadata-only (mili giây khi có lock); boot lại sau hội tụ là no-op; ngưỡng `lock_timeout` cụ thể chốt ở implement sau khi đo trên restore

**Constraints**: CẤM xóa volume / `down -v` mọi môi trường; CẤM target production cho verify; CẤM xóa file migration; CẤM migration mới chạm schema cấm trong lúc làm (FR-011, quy ước thủ công); rollback duy nhất = restore backup

**Scale/Scope**: 11 file Account viết lại nội dung + 2 file baseline mới + 1 embed wiring + 1 script gate + 1 init fix + 1 Makefile fix + docs/ADR; 0 dòng nghiệp vụ; ảnh hưởng mọi môi trường qua boot migration

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Nguyên tắc | Đánh giá |
|---|---|
| §I Independently Deployable Services | PASS — mục đích của spec chính là khôi phục §I cho tầng migration (mỗi service tự chạy DDL của mình, FR-035). Không thêm phụ thuộc deploy chéo; prod giữ song song. |
| §II Module boundaries / contracts | PASS — không đụng entity/contract/usecase; chỉ DDL + gate text. FK `identity.users` giữ nguyên theo ngoại lệ kiến trúc. |
| §III Outbox/events | PASS — không đụng event/outbox/worker. |
| §IV Clean architecture | PASS — file chạm: `.sql`, `migrations.go` embed, `di/providers.go` (đúng composition root), script gate, docs. Không bypass layer. |
| §V Error handling | PASS — precondition fail-fast dùng `RAISE EXCEPTION` rõ ràng trong migration (không phải apperror HTTP vì đây là boot-time, không phải request). |
| §VI AI pipeline | N/A — không đụng reservation/quota (UPDATE data ở `20260930150000` giữ nguyên, không replay). |
| §VII design-pattern.md | PASS — sẽ đọc bản mới nhất khi implement (ghi thành task đầu implement). |
| §VIII Portability / Goose | PASS — thuần PostgreSQL, không vendor feature; đúng `make migrate-create/up svc=<name>`; direct connection 5432 giữ nguyên (`migration.go` không đổi). |
| Governance: deferred | PASS — spec này *giải quyết* 027/D-06 (FR-010 đánh dấu resolved kèm dấu vết), không park nợ mới. Nợ phát sinh trong lúc làm (nếu có) vào `deferred.md` theo quy tắc. |
| Governance: frontend-guide | PASS — 0 thay đổi endpoint/event/DTO (ghi trong Assumptions) → miễn guide, sẽ ghi rõ trong `tasks.md`. |
| Governance: email volume | N/A — không flow gửi mail. |

*Re-check sau Phase 1 (2026-10-16)*: research R1–R6 và data-model/contracts/quickstart không đổi kết luận trên — không phát sinh vi phạm mới. Không có violation cần justify → mục Complexity Tracking để trống.

## Project Structure

### Documentation (this feature)

```text
specs/030-ddl-ownership-handover/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (R1–R6, đã xác minh bằng mã nguồn/DB thật)
├── data-model.md        # Phase 1 output (bản đồ sở hữu + ma trận hội tụ + sổ version)
├── quickstart.md        # Phase 1 output (V1 fresh-cách-ly, V2 rehearsal, V3 gate, V4 smoke)
├── contracts/
│   └── ownership-contract.md  # Phase 1 output (map chuẩn + 3 luật gate + điều khoản chuyển tiếp)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
services/account/database/migrations/
├── 20260622131631_add_ai_cost_policy.sql      # REWRITE CONTENT: gỡ DDL aiexecutor (giữ tên+version) [T007 xong]
├── 20260627143730_add_token_estimation_audit.sql  # REWRITE CONTENT: gỡ ALTER ai_usage_events (ai-only, no-op) [T033]
├── 20260627225700_update_ai_cost_quota_limits.sql # REWRITE CONTENT: gỡ UPDATE ai, giữ UPDATE subscription_plans [T034]
├── 20260629072235_lowercase_constants_and_constraints.sql # REWRITE CONTENT: gỡ dòng ai_*, giữ dòng brand_* [T035]
├── 20260704062159_init_ai_executor_module.sql # REWRITE CONTENT: gỡ tạo ai_model_configs + rename [T008 xong]
├── 20260723000001_create_dashboard_snapshots_tables.sql  # REWRITE CONTENT: gỡ tạo 3 bảng dashboard [T009 xong]
├── 20260724173038_fix_total_tokens_and_check_constraint.sql # REWRITE CONTENT: gỡ DDL/data ai_* (ai-only, no-op) [T036]
├── 20260801000000_separate_module_schemas.sql # REWRITE CONTENT: gỡ SET SCHEMA + FK ai/dashboard cả Up lẫn Down [T010 + T037]
├── 20260930120000_admin_premium_grant_schema.sql  # REWRITE CONTENT: gỡ 4 DDL aiexecutor [T011 xong]
├── 20260930150000_free_default_outfit_attempt_cap.sql  # REWRITE CONTENT: gỡ UPDATE aiexecutor (data) [T012 xong]
└── 20261011090002_add_payos_status_counts.sql # REWRITE CONTENT: gỡ ALTER dashboard [T013 xong]

services/ai/database/migrations/
├── migrations.go        # NEW: embed FS (copy mẫu Account)
└── 2026XXXXXX_*.sql     # NEW: chain baseline 7 bảng aiexecutor (idempotent, kèm rename)

services/commerce/database/migrations/
└── 2026XXXXXX_*.sql     # NEW: baseline 3 bảng dashboard (idempotent)

services/ai/internal/di/
├── providers.go         # EDIT: provideMigrationsFS return EmbedFS
└── wire_gen.go          # REGEN via make wire

shared/tools/ (hoặc vị trí Makefile quy ước)
└── migration-ownership-check.*  # NEW: script gate đọc contracts/ownership-contract.md

Makefile                  # EDIT: target migration-check + đấu vào check; FIX migrate-status/up/down/version truyền -table đúng từng service (goose_db_version[_commerce|_ai]) [T039]
database/init/01-schema.sql   # EDIT: bổ sung CREATE TABLE public.user_style_profiles đã mất do drift (khớp shape trong 20261011090000-Down) [T038]
specs/027-grant-repair-unblock/deferred.md  # EDIT: D-06 resolved
docs/decisions/           # NEW ADR bàn giao (số tiếp theo)
```

**Structure Decision**: Không thêm service/module/package nghiệp vụ. Mọi thay đổi nằm trong 3 cây migration hiện có + wiring DI có sẵn + 1 script gate + docs. Toàn bộ là diện "hạ tầng migration", tách bạch khỏi code nghiệp vụ.

## Complexity Tracking

> Không có violation cần justify (Constitution Check pass toàn bộ, re-check sau Phase 1 giữ nguyên).
