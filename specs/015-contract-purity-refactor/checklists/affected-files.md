# Affected Files Checklist: Contract Purity Refactor

**Date**: 2026-08-27
**Purpose**: Track files modified during implementation

## Modules and Files

### Wardrobe Module
- `[ ]` [contract_uc.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go) (FR-001, FR-008)
- `[ ]` [dto/wardrobe.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/wardrobe/application/dto/wardrobe.go) (FR-001a DTO field addition)
- `[ ]` [mapper/wardrobe.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/wardrobe/application/mapper/wardrobe.go) (FR-001a Mapper update)
- `[ ]` [presentation/contract/wardrobe_contract.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/wardrobe/presentation/contract/wardrobe_contract.go) (FR-007)
- `[ ]` [presentation/contract/wardrobe_contract_impl.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/wardrobe/presentation/contract/wardrobe_contract_impl.go) (FR-007)
- `[ ]` [provider.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/wardrobe/provider.go) (FR-009)
- `[ ]` [item/item_uc_read.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_read.go) (FR-011 dead wiring)

### Brand Module
- `[ ]` [presentation/contract/brand_contract.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/brand/presentation/contract/brand_contract.go) (FR-008)
- `[ ]` [presentation/contract/brand_contract_impl.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/brand/presentation/contract/brand_contract_impl.go) (FR-003, FR-008)
- `[ ]` [application/provider.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/brand/application/provider.go) (FR-003)
- `[ ]` [provider.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/brand/provider.go) (FR-003)
- `[NEW] [ ]` [brand_item_contract_uc.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/brand/application/interface/usecase/brand_item_contract_uc.go) (FR-003 interface)
- `[NEW] [ ]` [contract_uc.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/brand/application/usecase/item/contractuc/contract_uc.go) (FR-003 implementation)
- `[NEW] [ ]` [mapper.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/brand/application/usecase/item/contractuc/mapper.go) (FR-003 mapping helper)

### Fashion Module
- `[ ]` [worker/worker_uc.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/fashion/application/usecase/worker/worker_uc.go) (FR-005 caller enrich)
- `[ ]` [ai/chat/workflows.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/chat/workflows.go) (FR-005 caller enrich)
- `[ ]` [ai/chat/workflows_test.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/chat/workflows_test.go) (FR-023 mock update)
- `[ ]` [ai/recommendation/workflow_helpers.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go) (FR-005 caller enrich)

### Subscription Module
- `[ ]` [presentation/contract/subscription_contract.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/subscription/presentation/contract/subscription_contract.go) (FR-009)
- `[ ]` [presentation/contract/subscription_contract_impl.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/subscription/presentation/contract/subscription_contract_impl.go) (FR-009)
- `[ ]` [subscription_uc.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription/subscription_uc.go) (FR-011 dead wiring)
- `[ ]` [provider.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/modules/subscription/provider.go) (FR-012 wire update)

### Shared & Dependency Injection
- `[ ]` [wire_gen.go](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/internal/di/wire_gen.go) (DI regeneration)
- `[ ]` [constitution.md](file:///C:/0LamViec/_projects/hycat/closy/smart-wardrobe-be/.specify/memory/constitution.md) (Constitution update)
