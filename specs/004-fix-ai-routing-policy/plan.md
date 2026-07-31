# Implementation Plan: Giải quyết nợ kỹ thuật định tuyến AI Routing Policy

**Branch**: `004-fix-ai-routing-policy` | **Date**: 2026-07-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-fix-ai-routing-policy/spec.md`

## Summary

Dự án sẽ chuyển đổi luồng định tuyến và gọi AI từ cơ chế tĩnh (hardcode model) sang cơ chế động dựa vào kết quả quyết định định tuyến (`decision.Route` trả về từ `CostUseCase.Prepare`). Khi route là `free`, hệ thống tự động đổi sang cấu hình model miễn phí. Khi route là `local`, hệ thống sẽ trả về lỗi fallback để kích hoạt thuật toán Go thuần cục bộ.

## Technical Context

**Language/Version**: Go 1.25.+

**Primary Dependencies**: Gin web framework, GORM, shopspring/decimal, Google Wire

**Storage**: PostgreSQL, Redis

**Testing**: go test

**Target Platform**: Linux server / local development

**Project Type**: web-service

**Performance Goals**: <200ms p95 overhead for policy resolution

**Constraints**: dynamic route resolution without changing external module APIs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Modular Monolith**: Khớp hoàn toàn. Các thay đổi chỉ diễn ra cục bộ trong module `aiexecutor` và lớp interface presentation của nó.
- **AI pipeline & Cost control**: Khớp hoàn toàn. Đây chính là tính năng giải quyết nợ kỹ thuật cho nguyên tắc thứ VI của Hiến pháp (AI Pipeline, Cost Control & Local Fallbacks).

## Project Structure

### Documentation (this feature)

```text
specs/004-fix-ai-routing-policy/
├── plan.md              # This file
├── research.md          # Research findings
├── data-model.md        # Relevant entities summary
└── quickstart.md        # Hướng dẫn chạy thử nghiệm & xác thực
```

### Source Code (repository root)

```text
internal/modules/aiexecutor/
├── application/
│   ├── dto/
│   │   └── options.go
│   └── usecase/
│       ├── cost/
│       │   └── usecase.go
│       └── orchestrator/
│           ├── orchestrator.go
│           ├── plan.go
│           └── service.go
├── domain/
│   ├── constants/
│   │   ├── aioperation/
│   │   └── airoute/
│   └── entities/
│       ├── policy.go
│       └── usage.go
└── infrastructure/
    └── persistence/
        └── ai_cost_repo_impl.go
```

## Proposed Changes

### [Component Name]

#### [MODIFY] [plan.go](../../internal/modules/aiexecutor/application/usecase/orchestrator/plan.go)
- Điều chỉnh hàm `resolveProviderModel` để nhận thêm tham số `route airoute.AIRoute`.
- Trả về `cfg.AI.FreeTextPrimary` khi `route == airoute.Free || route == airoute.Local`.

#### [MODIFY] [orchestrator.go](../../internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go)
- Cập nhật logic các hàm `GenerateChatText`, `GenerateRecommendationText`, `GenerateChatTextStream`.
- Sau khi gọi `costUC.Prepare(ctx, plan)`, kiểm tra `decision.Route`:
  - Nếu là `airoute.Local`: Trực tiếp trả về lỗi `errors.New("ai route resolved to local fallback: bypassing remote api call")` mà không thực hiện gọi API LLM.
  - Nếu là `airoute.Free`: Cập nhật lại `plan.Provider` và `plan.Model` sang mô hình Free trước khi gọi API LLM.

#### [MODIFY] [.env.example](../../.env.example) và [.env.production.example](../../.env.production.example)
- Đảm bảo các cấu hình `FREE_TEXT_PRIMARY_...` nằm ở dưới cùng của phần AI. (Đã thực hiện ở Specify phase).

## Verification Plan

### Automated Tests
- Chạy unit tests cho orchestrator và cost usecase:
  ```bash
  go test ./internal/modules/aiexecutor/application/usecase/...
  ```
- Biên dịch toàn bộ dự án:
  ```bash
  go build ./...
  ```

### Manual Verification
- Cấu hình hạn mức của một standard user về 0, thực hiện gửi yêu cầu gợi ý trang phục và kiểm tra xem hệ thống có tự động fallback về thuật toán phối màu HSL cục bộ và trả về cờ `IsFallback: true` hay không.
