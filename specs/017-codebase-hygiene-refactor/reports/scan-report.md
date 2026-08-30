# Hygiene Scan Report

**Run ID**: `20260830-200903` | **Generated**: 2026-08-30T20:09:03+07:00

Scanned 635 production files (+57 test files). Violations: **0 fail**, **80 warning**, **0 needs-confirmation**.

## Summary

Compared against baseline `20260830-193926` (↑ worse, ↓ better, = unchanged).

| Rule | fail | warning | needs-confirmation |
|------|------|---------|--------------------|
| H1 | 0 (=) | 51 (=) | 0 (=) |
| H2 | 0 (=) | 9 (=) | 0 (=) |
| H6 | 0 (=) | 20 (=) | 0 (=) |

## By Rule

| Key | fail | warning | needs-confirmation |
|-----|------|---------|--------------------|
| H1 | 0 | 51 | 0 |
| H2 | 0 | 9 | 0 |
| H6 | 0 | 20 | 0 |

## By Module

| Key | fail | warning | needs-confirmation |
|-----|------|---------|--------------------|
| admin | 0 | 2 | 0 |
| aiexecutor | 0 | 14 | 0 |
| brand | 0 | 15 | 0 |
| dashboard | 0 | 4 | 0 |
| fashion | 0 | 10 | 0 |
| identity | 0 | 3 | 0 |
| internal/bootstrap | 0 | 1 | 0 |
| internal/pkg | 0 | 1 | 0 |
| internal/shared | 0 | 10 | 0 |
| internal/tools | 0 | 1 | 0 |
| notification | 0 | 1 | 0 |
| subscription | 0 | 8 | 0 |
| wardrobe | 0 | 10 | 0 |

## Violations

| Rule | Severity | Location | Detail |
|------|----------|----------|--------|
| H1 | warning | internal/bootstrap/app.go:78 `App.Run` | function spans 102 lines (threshold 80) |
| H1 | warning | internal/modules/aiexecutor/application/usecase/cost/usecase.go:43 `CostUseCase.Prepare` | function spans 93 lines (threshold 80) |
| H1 | warning | internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go:193 `OrchestratorUseCase.GenerateChatTextStream` | function spans 92 lines (threshold 80) |
| H1 | warning | internal/modules/brand/application/usecase/benefit/usecase_test.go | file has 1033 lines (warning threshold 750) |
| H1 | warning | internal/modules/brand/application/usecase/brand/member_uc.go:17 `BrandUseCase.AddBrandMembers` | function spans 164 lines (threshold 80) |
| H1 | warning | internal/modules/brand/application/usecase/claim/usecase.go:146 `BrandClaimUseCase.ClaimBrandCustomer` | function spans 85 lines (threshold 80) |
| H1 | warning | internal/modules/brand/application/usecase/item/crud_uc.go:30 `BrandItemUseCase.CreateBrandItem` | function spans 83 lines (threshold 80) |
| H1 | warning | internal/modules/brand/application/usecase/item/eligibility_uc.go:20 `BrandItemUseCase.ListEligibleBrandItemsForStyling` | function spans 97 lines (threshold 80) |
| H1 | warning | internal/modules/brand/application/usecase/loyalty/helper.go:109 `BrandLoyaltyUseCase.resolveBrandCustomerForPoints` | function spans 96 lines (threshold 80) |
| H1 | warning | internal/modules/brand/application/usecase/loyalty/points_uc.go:21 `BrandLoyaltyUseCase.GrantLoyaltyPoints` | function spans 131 lines (threshold 80) |
| H1 | warning | internal/modules/brand/presentation/router/router.go:37 `Router.Init` | function spans 121 lines (threshold 80) |
| H1 | warning | internal/modules/dashboard/application/usecase/admin_dashboard_uc_impl.go:103 `AdminDashboardUseCase.GetAIMarginAnalytics` | function spans 82 lines (threshold 80) |
| H1 | warning | internal/modules/dashboard/application/usecase/dynamic_query_execute_uc.go:20 `DynamicQueryUseCase.ExecuteDynamicQuery` | function spans 247 lines (threshold 80) |
| H1 | warning | internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go:110 `UserDashboardUseCase.GetWardrobeInsights` | function spans 87 lines (threshold 80) |
| H1 | warning | internal/modules/fashion/application/usecase/ai/chat/helper.go:27 `buildChatSystemPromptWithLimits` | function spans 119 lines (threshold 80) |
| H1 | warning | internal/modules/fashion/application/usecase/ai/recommendation/parser/parser.go:114 `LocalNLPParser.Parse` | function spans 82 lines (threshold 80) |
| H1 | warning | internal/modules/fashion/application/usecase/ai/recommendation/ranking/fallback_hsl.go:33 `RunLocalHSLMatching` | function spans 138 lines (threshold 80) |
| H1 | warning | internal/modules/fashion/application/usecase/ai/recommendation/ranking/scoring.go:51 `ScoreCandidateItem` | function spans 189 lines (threshold 80) |
| H1 | warning | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/prompt.go:57 `buildRecommendationPrompt` | function spans 90 lines (threshold 80) |
| H1 | warning | internal/modules/fashion/application/usecase/ai/recommendation/workflow_helpers.go:91 `OutfitRecommendationUseCase.filterCandidates` | function spans 289 lines (threshold 80) |
| H1 | warning | internal/modules/fashion/application/usecase/worker/cleanup_uc.go:99 `FashionWorkerUseCase.RecoverStaleProcessingItems` | function spans 81 lines (threshold 80) |
| H1 | warning | internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid.go:21 `FashionItemRepository.GetHybridCandidates` | function spans 102 lines (threshold 80) |
| H1 | warning | internal/modules/identity/application/mapper/user_mapper.go:43 `MapToUserRes` | function spans 87 lines (threshold 80) |
| H1 | warning | internal/modules/identity/application/usecase/session_uc.go:118 `SessionUseCase.RefreshToken` | function spans 106 lines (threshold 80) |
| H1 | warning | internal/modules/notification/infrastructure/communication/smtp_sender.go:28 `SmtpSender.Send` | function spans 83 lines (threshold 80) |
| H1 | warning | internal/modules/subscription/application/usecase/purchase/checkout.go:24 `SubscriptionPurchaseUseCase.CreateDirectPurchase` | function spans 151 lines (threshold 80) |
| H1 | warning | internal/modules/subscription/application/usecase/purchase/wallet.go:23 `SubscriptionPurchaseUseCase.PurchasePlanWithWallet` | function spans 153 lines (threshold 80) |
| H1 | warning | internal/modules/subscription/application/usecase/reconciliation/payment_reconciliation_usecase.go:82 `PaymentReconciliationUseCase.reconcile` | function spans 90 lines (threshold 80) |
| H1 | warning | internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go:340 `SubscriptionUseCase.executePaidRenewal` | function spans 91 lines (threshold 80) |
| H1 | warning | internal/modules/subscription/application/usecase/wallet/internal_wallet_transaction_workflow.go:29 `ProcessWalletTransaction` | function spans 104 lines (threshold 80) |
| H1 | warning | internal/modules/subscription/application/usecase/webhook/purchase_workflow.go:19 `PaymentWebhookUseCase.executeDirectPurchaseWorkflow` | function spans 126 lines (threshold 80) |
| H1 | warning | internal/modules/subscription/infrastructure/payment/payos/payos_service_impl.go:57 `PayOSService.CreateCheckoutSession` | function spans 98 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/application/mapper/search_mapper.go:13 `MapToSearchDocumentDTO` | function spans 91 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/application/usecase/outfit/outfit_uc.go:74 `OutfitUseCase.SaveOutfit` | function spans 119 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/application/usecase/outfit/outfit_uc.go:194 `OutfitUseCase.UpdateOutfit` | function spans 115 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/application/usecase/wardrobe/catalog/catalog_uc.go:117 `WardrobeCatalogUseCase.InitClosetFromCatalog` | function spans 103 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_read.go:90 `WardrobeItemUseCase.GetWardrobeItems` | function spans 114 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go:27 `WardrobeItemUseCase.CloneWardrobeItem` | function spans 138 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go:386 `WardrobeItemUseCase.BatchUploadWardrobeItems` | function spans 90 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/infrastructure/search/wardrobe_helper.go:14 `WardrobeSearchService.parseSearchWardrobeItemRes` | function spans 84 lines (threshold 80) |
| H1 | warning | internal/modules/wardrobe/infrastructure/search/wardrobe_search.go:15 `WardrobeSearchService.SearchItems` | function spans 81 lines (threshold 80) |
| H1 | warning | internal/pkg/logger/logger.go:27 `New` | function spans 82 lines (threshold 80) |
| H1 | warning | internal/shared/infrastructure/ai/google/google_text.go:17 `CallTextStream` | function spans 123 lines (threshold 80) |
| H1 | warning | internal/shared/infrastructure/ai/google/google_text.go:142 `CallText` | function spans 100 lines (threshold 80) |
| H1 | warning | internal/shared/infrastructure/ai/openai/openai.go:19 `CallTextStream` | function spans 116 lines (threshold 80) |
| H1 | warning | internal/shared/infrastructure/ai/openai/openai.go:137 `CallText` | function spans 84 lines (threshold 80) |
| H1 | warning | internal/shared/infrastructure/ai/vision_embedding_service.go:80 `AIService.EstimatePromptTokens` | function spans 83 lines (threshold 80) |
| H1 | warning | internal/shared/infrastructure/messaging/rabbitmq_connection.go:12 `RabbitMQClient.connect` | function spans 117 lines (threshold 80) |
| H1 | warning | internal/shared/presentation/middleware/error_handler.go:14 `GlobalErrorHandler` | function spans 85 lines (threshold 80) |
| H1 | warning | internal/shared/presentation/middleware/swagger_lock.go:12 `SwaggerLockMiddleware` | function spans 193 lines (threshold 80) |
| H1 | warning | internal/tools/swagger-split/main.go:44 `run` | function spans 116 lines (threshold 80) |
| H2 | warning | internal/modules/aiexecutor/domain/entities/table_name.go | file carries behavior for 6 types (AICostPolicy, AICostPolicyOperation, AIModelConfig, AIPolicyGrant, AIUsageEvent, AIUsagePeriodLedger); split by responsibility |
| H2 | warning | internal/modules/brand/domain/entities/table_name.go | file carries behavior for 15 types (BenefitRedemption, Brand, BrandBenefit, BrandConversation, BrandConversationMessage, BrandCustomer, BrandCustomerClaim, BrandItem, BrandMember, DigitalSampleResponse, LoyaltyAccount, LoyaltyPointLot, LoyaltyPointTransaction, LoyaltyProgram, LoyaltyTier); split by responsibility |
| H2 | warning | internal/modules/dashboard/domain/entities/dashboard_snapshots.go | file carries behavior for 3 types (DashboardAdminDailySnapshot, DashboardBrandDailySnapshot, DashboardUserWardrobeSummary); split by responsibility |
| H2 | warning | internal/modules/fashion/domain/entities/table_name.go | file carries behavior for 4 types (Category, ConversationalContext, FashionItem, Message); split by responsibility |
| H2 | warning | internal/modules/identity/domain/entities/table_name.go | file carries behavior for 3 types (RefreshToken, User, UserStyleProfile); split by responsibility |
| H2 | warning | internal/modules/subscription/domain/entities/table_name.go | file carries behavior for 10 types (DepositTransaction, ProviderPaymentEvent, ProviderWebhookInbox, SubscriptionPlan, SubscriptionRenewalAttempt, UserDailyQuota, UserSubscription, UserSubscriptionEvent, UserWallet, WalletStatement); split by responsibility |
| H2 | warning | internal/modules/wardrobe/domain/entities/table_name.go | file carries behavior for 3 types (Outfit, OutfitItem, WardrobeItem); split by responsibility |
| H2 | warning | internal/shared/application/event/consumer.go | file carries behavior for 3 types (Message, PermanentError, RetryableError); split by responsibility |
| H2 | warning | internal/shared/domain/entities/jsonb.go | file carries behavior for 3 types (JSONDocument, bodyProfile, preferredColors); split by responsibility |
| H6 | warning | internal/modules/admin/presentation/router/router.go `router.go` | potential duplication: file router.go exists in 8 modules (internal/modules/admin, internal/modules/aiexecutor, internal/modules/brand, internal/modules/dashboard, internal/modules/fashion, internal/modules/identity, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/admin/provider.go `provider.go` | potential duplication: file provider.go exists in 9 modules (internal/modules/admin, internal/modules/aiexecutor, internal/modules/brand, internal/modules/dashboard, internal/modules/fashion, internal/modules/identity, internal/modules/notification, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/aiexecutor/application/dto/dashboard_dto.go `dashboard_dto.go` | potential duplication: file dashboard_dto.go exists in 4 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/identity, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/aiexecutor/application/errors/errors.go `errors.go` | potential duplication: file errors.go exists in 6 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/dashboard, internal/modules/identity, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/aiexecutor/application/usecase/cost/helper.go `helper.go` | potential duplication: file helper.go exists in 3 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/fashion); audit before merging |
| H6 | warning | internal/modules/aiexecutor/application/usecase/cost/usecase.go `usecase.go` | potential duplication: file usecase.go exists in 3 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/fashion); audit before merging |
| H6 | warning | internal/modules/aiexecutor/application/usecase/orchestrator/plan.go `plan.go` | potential duplication: file plan.go exists in 2 modules (internal/modules/aiexecutor, internal/modules/subscription); audit before merging |
| H6 | warning | internal/modules/aiexecutor/domain/entities/base_types.go `base_types.go` | potential duplication: file base_types.go exists in 6 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/fashion, internal/modules/identity, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/aiexecutor/domain/entities/table_name.go `table_name.go` | potential duplication: file table_name.go exists in 6 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/fashion, internal/modules/identity, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/aiexecutor/domain/repositories/readmodels.go `readmodels.go` | potential duplication: file readmodels.go exists in 6 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/fashion, internal/modules/identity, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/aiexecutor/presentation/contract/dashboard_reader.go `dashboard_reader.go` | potential duplication: file dashboard_reader.go exists in 6 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/fashion, internal/modules/identity, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/aiexecutor/presentation/handler/admin_handler.go `admin_handler.go` | potential duplication: file admin_handler.go exists in 2 modules (internal/modules/aiexecutor, internal/modules/identity); audit before merging |
| H6 | warning | internal/modules/aiexecutor/presentation/worker/subscription_handler.go `subscription_handler.go` | potential duplication: file subscription_handler.go exists in 2 modules (internal/modules/aiexecutor, internal/modules/subscription); audit before merging |
| H6 | warning | internal/modules/brand/application/dto/chat.go `chat.go` | potential duplication: file chat.go exists in 2 modules (internal/modules/brand, internal/modules/fashion); audit before merging |
| H6 | warning | internal/modules/brand/application/usecase/item/contractuc/contract_uc.go `contract_uc.go` | potential duplication: file contract_uc.go exists in 2 modules (internal/modules/brand, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/brand/application/usecase/item/fashion_event_uc.go `fashion_event_uc.go` | potential duplication: file fashion_event_uc.go exists in 2 modules (internal/modules/brand, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/brand/domain/repositories/interfaces.go `interfaces.go` | potential duplication: file interfaces.go exists in 3 modules (internal/modules/brand, internal/modules/fashion, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/brand/presentation/worker/fashion_bind_worker.go `fashion_bind_worker.go` | potential duplication: file fashion_bind_worker.go exists in 2 modules (internal/modules/brand, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/brand/presentation/worker/fashion_event_worker.go `fashion_event_worker.go` | potential duplication: file fashion_event_worker.go exists in 2 modules (internal/modules/brand, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/fashion/application/dto/recommendation.go `recommendation.go` | potential duplication: file recommendation.go exists in 2 modules (internal/modules/fashion, internal/modules/wardrobe); audit before merging |
| H8 | warning | internal/modules/fashion/application/usecase/ai/chat/summary.go:30 `errors.New` | accepted-keep: one-off parse-guard errors inside summary JSON extraction; no chain to wrap |
| H8 | warning | internal/modules/fashion/application/usecase/ai/chat/summary.go:64 `errors.New` | accepted-keep: one-off parse-guard errors inside summary JSON extraction; no chain to wrap |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:71 `errors.New` | accepted-keep: one-off defensive parse guards; no chain to wrap |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:147 `errors.New` | accepted-keep: one-off defensive parse guards; no chain to wrap |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:154 `errors.New` | accepted-keep: one-off defensive parse guards; no chain to wrap |
| H5 | needs-confirmation | internal/modules/fashion/application/usecase/ai/chat/helper.go:148 `unreachable func: isWardrobeRelatedQuery` | accepted-keep: used by helpers_test.go to lock keyword-detection behavior |
| H5 | needs-confirmation | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/prompt.go:38 `unreachable func: BuildRecommendationPrompt` | accepted-keep: used by synthesis_test.go to lock the stylist prompt contract |
| H5 | needs-confirmation | internal/modules/fashion/application/usecase/ai/recommendation/types/structs.go:58 `unreachable func: FallbackCandidateCounts.Total` | accepted-keep: used by ranking tests (scoring_test.go) to assert fallback candidate counts |
| H5 | needs-confirmation | internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid_helpers.go:124 `unreachable func: buildRecommendationSeasonalityCondition` | accepted-keep: used by fashion_item_repo_hybrid_test.go to assert seasonality SQL |
| H5 | needs-confirmation | internal/modules/subscription/application/usecase/subscription/state_machine.go:139 `unreachable func: RestoreLifetimeFallbackIfExpired` | accepted-keep: exported state-machine transition, covered by state_machine_test.go |
| H5 | needs-confirmation | internal/modules/subscription/application/usecase/subscription/state_machine.go:202 `unreachable func: expiryFor` | accepted-keep: used by RestoreLifetimeFallbackIfExpired cluster and state_machine_test.go |
| H5 | needs-confirmation | internal/modules/subscription/application/usecase/subscription/state_machine.go:210 `unreachable func: AddProductDays` | accepted-keep: used by expiryFor and state_machine_test.go |
| H5 | needs-confirmation | internal/modules/subscription/application/usecase/subscription/state_machine.go:217 `unreachable func: clearFallback` | accepted-keep: used by RestoreLifetimeFallbackIfExpired |
| H5 | needs-confirmation | internal/modules/wardrobe/application/usecase/wardrobe/shared/lifecycle_decay.go:13 `unreachable func: CalculateLifecycleDecayFactor` | accepted-keep: used by lifecycle_decay_test.go to lock decay formula |
| H5 | needs-confirmation | internal/shared/domain/entities/jsonb.go:120 `unreachable func: JSONDocument.Scan` | accepted-keep: implements sql.Scanner — GORM contract, called by the database driver |
| H5 | needs-confirmation | internal/shared/domain/entities/vector.go:29 `unreachable func: Vector.Scan` | accepted-keep: implements sql.Scanner — GORM contract for pgvector column |
| H5 | needs-confirmation | internal/shared/presentation/middleware/ratelimit.go:78 `unreachable func: RateLimitMiddleware.Stop` | accepted-keep: shutdown hook for the cleanup goroutine started in NewRateLimitMiddleware |
| H5 | needs-confirmation | internal/shared/presentation/worker/base_worker.go:50 `unreachable func: BaseWorker.Name` | accepted-keep: BaseWorker framework accessor (pair with Start/Stop/Context/SetHealthCheck) |

## Exclusions

| Pattern | Reason |
|---------|--------|
| api/swagger/** | generated swagger artifacts |
| bin/** | build output |
| deployments/** | deployment configuration, no Go source |
| specs/** | specification documents |
| docs/** | documentation |
| database/** | goose SQL migrations |
| .git/** | vcs metadata |

## Needs Confirmation

13 finding(s) require manual verification before removal (see Violations table, rule H5).


## Final Audit — Phase 9 Convergence (2026-08-30, run 20260830-193926)

### Leak fix (FR-005 ngoại lệ security fix + T070–T076)

- Biên mapper (`apperror.From`/`Wrap`, `errorutils.MapErrorToProblem` đã xóa): error không nhận diện
  được trả `GenericInternalMessage`; raw text chỉ trong `Cause` (log) và `debugMessage` (dev-only).
- AI clients (OpenAI/Gemini): HTTP error của provider map thành static 503
  (`apperror.NewUpstreamUnavailable`), response body chỉ log.
- SSE chat stream: event `error` chỉ chứa message an toàn.
- Worker analyze → RabbitMQ → SSE task: `payload.Error` thay bằng reason an toàn từ catalog.
- Elasticsearch client: Message static, body ES chỉ log. Redis OTP: map 503 tại biên identity.
- SMTP mapping điểm identity: cause được gắn vào apperror để log giữ chi tiết.

### H8 (33 cảnh báo `fmt.Errorf` + 8 `errors.New`)

- 33 vị trí `fmt.Errorf` đã verify: toàn bộ message-only (không có error argument) → heuristic
  H8 được refine để không flag (thay vì chèn `%w` máy móc).
- 3× `errors.New("ai route resolved to local fallback...")` (orchestrator) → gộp thành sentinel
  `ErrLocalFallbackRoute` package-level (quy ước Option A / FR-003).
- 5× `errors.New` one-off còn lại (summary.go ×2, llm_rewriter.go ×3) → keep-reason trong
  accepted.json.

### H5 (32 candidate needs-confirmation)

- **19 symbol xóa** (bằng chứng: 0 tham chiếu ngoài khai báo, hoặc duplicate của bản đang dùng):
  4 sentinel admin errors (duplicate identity/brand), MapBrandMembers, MapBrandCustomers,
  parseValidDurationDays (bản benefit/helper.go), ErrInvalidChatIDFormat (fashion),
  ErrInvalidChatIDFormat + ErrMinimumWardrobeItemsRequired + ErrChatNotFound (wardrobe),
  RemoveVietnameseSigns (wardrobe, duplicate stringutils), buildChatSystemPrompt (wrapper 0
  caller), recommendationSearchDocumentSQL, MapErrorToProblem (chỉ test dùng), parseSSEStream
  (bản shared/ai, duplicate openai/google), DefaultClient, repositories.Offset,
  repositories.BuildPaginationMetadata.
- **13 symbol giữ với keep-reason** (interface contract / framework accessor / test-referenced):
  Scan ×2 (sql.Scanner), BaseWorker.Name, RateLimitMiddleware.Stop, FallbackCandidateCounts.Total,
  buildRecommendationSeasonalityCondition, RestoreLifetimeFallbackIfExpired, expiryFor,
  AddProductDays, clearFallback, BuildRecommendationPrompt, isWardrobeRelatedQuery,
  CalculateLifecycleDecayFactor — ghi trong `internal/tools/hygiene-scan/accepted.json` (FR-011).

### Kết quả số lần scan cuối (run 20260830-193926 / verify 20260830-200903)

- violations: **0 fail, 80 warning** (H1 51, H2 9, H6 20), **0 needs-confirmation**.
- accepted-keep: **18** (13 H5 + 5 H8).
- metrics: files_over_500=0, files_over_800=0, cross_module_prohibited_imports=0, layer_skips=0.
- Kiểm chứng: `make wire` ✓, `make lint` ✓ (fmt-check + vet), `make test-race` ✓ exit 0.
- Snapshots giữ 3 mốc đối chiếu: `164823` (trước refactor) → `180210` (sau Phase 1–8, commit
  27c69ab) → `193926` (sau Phase 9).
