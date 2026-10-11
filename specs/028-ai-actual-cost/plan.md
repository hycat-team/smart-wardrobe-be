# Implementation Plan: Chi phí AI thật phục vụ báo cáo lợi nhuận

**Branch**: `028-ai-actual-cost` | **Date**: 2026-10-11 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification từ `specs/028-ai-actual-cost/spec.md` + 4 clarify đã chốt (session 2026-10-11) + quyết định hậu clarify: embedding đếm inline bằng `countTokens` (R7) + `manual-seed.md` (seed tay 7 model).

## Summary

Bỏ dòng ép `actual_cost = 0` cho route free trong `AICostRepository.Confirm`,
luôn ghi chi phí thật tính từ pricing của model đã gọi; vision parse thêm
`usageMetadata` để có token thật; embedding đếm inline bằng `countTokens`
(fail-open về `usage_unreported`); lượt thiếu giá/thiếu token được
ghi mã `error_code` để đối soát (`pricing_config_missing`, `usage_unreported`).
Không đổi schema (trừ 1 field proto cộng thêm cho drill-down), không đổi logic
thu tiền/quota/chuyển tuyến của user, không migration code — giá model seed tay
qua `manual-seed.md`.

## Technical Context

**Language/Version**: Go 1.25+, multi-module workspace (`go.work`, mỗi service một `go.mod`)

**Primary Dependencies**: GORM, Gin, gRPC/protobuf (`contracts/ai/v1`), Wire DI, Goose (migrations nằm ở account service — tính năng này KHÔNG tạo migration)

**Storage**: PostgreSQL (`aiexecutor.ai_usage_events`, `ai_usage_period_ledgers`, `ai_model_configs`); Redis pricing cache 30 phút (`pricingCacheTTL` — giá seed mới cần chờ hết cache hoặc restart khi verify)

**Testing**: `go test -race -count=3` scoped + `make lint` (gate một phase); full `make check` trước push/tag

**Target Platform**: 3-service topology (Account / Commerce / AI Engine), chạy được cả VPS Compose lẫn managed — cùng image, chỉ khác config

**Project Type**: backend web-service (REST admin + gRPC liên service)

**Performance Goals**: Vision parse usage có sẵn trong response (0 API call thêm). Embedding không gọi thêm gì (spike T001 chứng minh `countTokens` 400 — gọi thêm chỉ tốn latency vô ích). Confirm thêm đúng 1 lookup pricing đã có sẵn.

**Constraints**: Không chặn user khi thiếu giá/thiếu token; không ước tính bịa số liệu (Q2-A); error metrics hiện tại (`Released`-only) không được tăng; deploy độc lập từng service (field proto mới phải cộng thêm, bỏ qua được); spike `countTokens` cho embedding model phải pass trước khi code nhánh embedding (R3/R7)

**Scale/Scope**: ~6 file nguồn + 1 proto + regen; 7 model trong seed tay; 0 migration

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- §I (deploy độc lập): PASS — field proto cộng thêm, client cũ bỏ qua; AI service deploy trước/sau commerce đều an toàn.
- §II (module boundary/contract): PASS — mọi thay đổi liên service đi qua `contracts/ai/v1` + regen; không import implementation cross-service.
- §IV (layering): PASS — repo chỉ implement interface domain; handler mỏng; mapper nằm `application/mapper` hoặc `contract/transport` hiện có.
- §V (error/transaction): PASS — dùng `apperror` và mã `error_code` là data, không phải error path; Confirm giữ transaction hiện có.
- §VI (AI cost control): PASS và là mục tiêu tính năng — reserve-first, fallback, hard-cap giữ nguyên; chỉ sửa nhánh ghi actual.
- Frontend guide (§Governance): CONDITIONAL — `AIEventDrillDown` thêm field → `tasks.md` bắt buộc có task viết `frontend-guide.md` (đánh dấu additive). Đã ghi trong `contracts/admin-api-delta.md`.
- Không thay đổi kiến trúc, không cần bump version constitution.

Re-check sau Phase 1: không phát sinh vi phạm mới (không migration, không endpoint mới, không DTO breaking).

## Project Structure

### Documentation (this feature)

```text
specs/028-ai-actual-cost/
├── plan.md              # File này
├── research.md          # Phase 0: 7 quyết định R1–R7
├── data-model.md        # Phase 1: ngữ nghĩa trường hiện có + 2 error_code
├── quickstart.md        # Phase 1: 6 kịch bản kiểm chứng + lệnh test
├── manual-seed.md       # Seed tay 7 model (dev tự chạy, đã có từ specify)
├── contracts/
│   └── admin-api-delta.md  # Phase 1: 1 field proto cộng thêm + API không đổi
├── checklists/
│   └── requirements.md  # Spec quality (16/16)
├── baseline-quota.md    # Runtime artifact do T025 sinh (không commit mẫu)
└── tasks.md             # Phase 2, đã tạo (T001–T026)
```

### Source Code (repository root)

```text
shared/infrastructure/ai/google/
└── google_vision_embed.go      # Parse usageMetadata + CallVisionWithUsage mới
shared/application/ai/
└── interfaces.go               # Thêm method ...WithUsage (cộng thêm, không đổi cũ)
services/ai/internal/modules/aiexecutor/
├── application/usecase/cost/usecase.go            # Confirm: 5 nhánh actual/error_code
├── application/usecase/orchestrator/orchestrator.go # Vision dùng WithUsage; embedding đếm inline countTokens
├── infrastructure/persistence/ai_cost_repo_impl.go # Confirm bỏ ép 0 + error_code; drill-down đọc error_code
├── domain/repositories/ai_cost_repo.go            # Interface Confirm thêm errorCode
├── domain/repositories/readmodels.go              # DrillDown DTO thêm ErrorCode
└── contract/.../transport/                        # Mapper proto field mới
contracts/ai/v1/
├── ai.proto                        # AIEventDrillDown.error_code = 7
└── ai_grpc.pb.go / ai.pb.go        # Regen
services/commerce/.../dashboard/    # KHÔNG CHẠM (tự đúng qua SUM actual)
```

**Structure Decision**: Sửa tại chỗ trong module `aiexecutor` (AI Engine) + shared AI provider + 1 field proto cộng thêm. Commerce giữ nguyên.

## Complexity Tracking

> Không có vi phạm constitution cần biện minh — bảng này để trống theo template.
