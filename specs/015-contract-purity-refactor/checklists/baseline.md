# Baseline Contract Purity Audit Report

**Date**: 2026-08-27
**Status**: FAILED (Pre-refactor state)

## Detected Issues

### 1. Pure Contract Violations
- `internal/modules/wardrobe/application/usecase/wardrobe/contractuc/contract_uc.go` imports:
  - `fashion/presentation/contract`
  - `subscription/presentation/contract`
- `internal/modules/brand/presentation/contract/brand_contract_impl.go` still injects leaky `IBrandItemUseCase`.

### 2. Contract Deadcode
The following dead methods are present in the interfaces and implementations:
- `CopyItemToUser`
- `VerifyItemsForPost`
- `GetItemsByIDs`
- `GetUserFashionItemIDs`
- `MarkProcessingFailed`
- `MarkProcessingNeedsReview`
- `CompleteProcessingSuccess`
- `CheckBrandFeatureAccess`
- `UpdateBrandItemStatus`
- `GetUserSubscription`
- `GetDefaultSubscriptionPlanID`
- `IsPremiumPlan`

### 3. Dead Wiring
- `internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_read.go` injects `aiService` (not used).
- `internal/modules/subscription/application/usecase/subscription/subscription_uc.go` injects `userBriefContract` (not used).

## Metrics Summary
- **Detour Calls**: 1 (proxy pattern search)
- **Pure Contract Violations**: 4
- **Contract Deadcode**: 35 (total line matches in presentation/contract)
- **Dead Wiring**: 4
