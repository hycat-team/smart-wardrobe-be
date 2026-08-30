# Hygiene Scan Report

**Run ID**: `20260830-180210` | **Generated**: 2026-08-30T18:02:10+07:00

Scanned 637 production files (+58 test files). Violations: **0 fail**, **121 warning**, **32 needs-confirmation**.

## Summary

Compared against baseline `20260830-174633` (↑ worse, ↓ better, = unchanged).

| Rule | fail | warning | needs-confirmation |
|------|------|---------|--------------------|
| H1 | 0 (=) | 51 (=) | 0 (=) |
| H2 | 0 (=) | 9 (=) | 0 (=) |
| H5 | 0 (=) | 0 (=) | 32 (=) |
| H6 | 0 (=) | 20 (=) | 0 (=) |
| H8 | 0 (=) | 41 (=) | 0 (=) |

## By Rule

| Key | fail | warning | needs-confirmation |
|-----|------|---------|--------------------|
| H1 | 0 | 51 | 0 |
| H2 | 0 | 9 | 0 |
| H5 | 0 | 0 | 32 |
| H6 | 0 | 20 | 0 |
| H8 | 0 | 41 | 0 |

## By Module

| Key | fail | warning | needs-confirmation |
|-----|------|---------|--------------------|
| admin | 0 | 3 | 4 |
| aiexecutor | 0 | 18 | 0 |
| brand | 0 | 15 | 3 |
| dashboard | 0 | 4 | 0 |
| fashion | 0 | 30 | 7 |
| identity | 0 | 3 | 0 |
| internal/bootstrap | 0 | 1 | 0 |
| internal/pkg | 0 | 1 | 1 |
| internal/shared | 0 | 10 | 8 |
| internal/tools | 0 | 1 | 0 |
| notification | 0 | 4 | 0 |
| subscription | 0 | 20 | 4 |
| wardrobe | 0 | 11 | 5 |

## Violations

| Rule | Severity | Location | Detail |
|------|----------|----------|--------|
| H1 | warning | internal/bootstrap/app.go:78 `App.Run` | function spans 102 lines (threshold 80) |
| H1 | warning | internal/modules/aiexecutor/application/usecase/cost/usecase.go:43 `CostUseCase.Prepare` | function spans 93 lines (threshold 80) |
| H1 | warning | internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go:189 `OrchestratorUseCase.GenerateChatTextStream` | function spans 92 lines (threshold 80) |
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
| H1 | warning | internal/modules/fashion/application/usecase/ai/chat/helper.go:32 `buildChatSystemPromptWithLimits` | function spans 119 lines (threshold 80) |
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
| H1 | warning | internal/shared/presentation/middleware/error_handler.go:14 `GlobalErrorHandler` | function spans 81 lines (threshold 80) |
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
| H6 | warning | internal/modules/admin/application/errors/errors.go `errors.go` | potential duplication: file errors.go exists in 7 modules (internal/modules/admin, internal/modules/aiexecutor, internal/modules/brand, internal/modules/dashboard, internal/modules/identity, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/admin/presentation/router/router.go `router.go` | potential duplication: file router.go exists in 8 modules (internal/modules/admin, internal/modules/aiexecutor, internal/modules/brand, internal/modules/dashboard, internal/modules/fashion, internal/modules/identity, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/admin/provider.go `provider.go` | potential duplication: file provider.go exists in 9 modules (internal/modules/admin, internal/modules/aiexecutor, internal/modules/brand, internal/modules/dashboard, internal/modules/fashion, internal/modules/identity, internal/modules/notification, internal/modules/subscription, internal/modules/wardrobe); audit before merging |
| H6 | warning | internal/modules/aiexecutor/application/dto/dashboard_dto.go `dashboard_dto.go` | potential duplication: file dashboard_dto.go exists in 4 modules (internal/modules/aiexecutor, internal/modules/brand, internal/modules/identity, internal/modules/wardrobe); audit before merging |
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
| H8 | warning | internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go:93 `errors.New` | use case constructs bare errors.New; use apperror factory or domain error constructor |
| H8 | warning | internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go:151 `errors.New` | use case constructs bare errors.New; use apperror factory or domain error constructor |
| H8 | warning | internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go:221 `errors.New` | use case constructs bare errors.New; use apperror factory or domain error constructor |
| H8 | warning | internal/modules/aiexecutor/application/usecase/orchestrator/plan.go:21 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/aiexecutor/application/usecase/pricing/usecase.go:43 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/chat/summary.go:30 `errors.New` | use case constructs bare errors.New; use apperror factory or domain error constructor |
| H8 | warning | internal/modules/fashion/application/usecase/ai/chat/summary.go:64 `errors.New` | use case constructs bare errors.New; use apperror factory or domain error constructor |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:71 `errors.New` | use case constructs bare errors.New; use apperror factory or domain error constructor |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:83 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:147 `errors.New` | use case constructs bare errors.New; use apperror factory or domain error constructor |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:154 `errors.New` | use case constructs bare errors.New; use apperror factory or domain error constructor |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:173 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:217 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:220 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:231 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go:246 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/generation.go:69 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/generation.go:90 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/json.go:38 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/json.go:52 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/json.go:57 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/json.go:61 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/prompt.go:54 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/worker/analyze_uc.go:187 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/fashion/application/usecase/worker/shared/fashion_helper.go:32 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/notification/application/usecase/email/outbox_email_uc.go:53 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/notification/application/usecase/email/outbox_email_uc.go:58 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/notification/application/usecase/email/outbox_email_uc.go:94 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/state_machine.go:67 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/state_machine.go:73 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/state_machine.go:78 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/state_machine.go:85 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/state_machine.go:90 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/state_machine.go:180 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go:134 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go:267 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go:435 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go:440 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go:443 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go:449 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H8 | warning | internal/modules/wardrobe/application/usecase/wardrobe/search_sync/search_sync_uc.go:57 `fmt.Errorf` | use case uses fmt.Errorf without %w; wrap the cause or use apperror |
| H5 | needs-confirmation | internal/modules/admin/application/errors/errors.go:7 `unreachable func: ErrUserNotFound` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/admin/application/errors/errors.go:11 `unreachable func: ErrBrandMemberNotFound` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/admin/application/errors/errors.go:15 `unreachable func: ErrCannotRemoveLastOwner` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/admin/application/errors/errors.go:19 `unreachable func: ErrCannotDemoteLastOwner` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/brand/application/mapper/brand.go:123 `unreachable func: MapBrandMembers` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/brand/application/mapper/brand.go:195 `unreachable func: MapBrandCustomers` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/brand/application/usecase/benefit/helper.go:9 `unreachable func: parseValidDurationDays` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/fashion/application/errors/chat_errors.go:11 `unreachable func: ErrInvalidChatIDFormat` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/fashion/application/usecase/ai/chat/helper.go:20 `unreachable func: buildChatSystemPrompt` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/fashion/application/usecase/ai/chat/helper.go:153 `unreachable func: isWardrobeRelatedQuery` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/fashion/application/usecase/ai/recommendation/synthesis/prompt.go:38 `unreachable func: BuildRecommendationPrompt` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/fashion/application/usecase/ai/recommendation/types/structs.go:58 `unreachable func: FallbackCandidateCounts.Total` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid_helpers.go:124 `unreachable func: buildRecommendationSeasonalityCondition` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid_helpers.go:191 `unreachable func: recommendationSearchDocumentSQL` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/subscription/application/usecase/subscription/state_machine.go:139 `unreachable func: RestoreLifetimeFallbackIfExpired` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/subscription/application/usecase/subscription/state_machine.go:202 `unreachable func: expiryFor` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/subscription/application/usecase/subscription/state_machine.go:210 `unreachable func: AddProductDays` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/subscription/application/usecase/subscription/state_machine.go:217 `unreachable func: clearFallback` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/wardrobe/application/errors/errors.go:44 `unreachable func: ErrInvalidChatIDFormat` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/wardrobe/application/errors/errors.go:104 `unreachable func: ErrMinimumWardrobeItemsRequired` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/wardrobe/application/errors/errors.go:108 `unreachable func: ErrChatNotFound` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/wardrobe/application/usecase/wardrobe/shared/lifecycle_decay.go:13 `unreachable func: CalculateLifecycleDecayFactor` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/modules/wardrobe/application/usecase/wardrobe/shared/string_helper.go:6 `unreachable func: RemoveVietnameseSigns` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/pkg/utils/errorutils/errorutils.go:199 `unreachable func: MapErrorToProblem` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/shared/domain/entities/jsonb.go:120 `unreachable func: JSONDocument.Scan` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/shared/domain/entities/vector.go:29 `unreachable func: Vector.Scan` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/shared/infrastructure/ai/helper.go:56 `unreachable func: parseSSEStream` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/shared/infrastructure/httptransport/transport.go:35 `unreachable func: DefaultClient` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/shared/infrastructure/repositories/pagination.go:8 `unreachable func: Offset` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/shared/infrastructure/repositories/pagination.go:17 `unreachable func: BuildPaginationMetadata` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/shared/presentation/middleware/ratelimit.go:78 `unreachable func: RateLimitMiddleware.Stop` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |
| H5 | needs-confirmation | internal/shared/presentation/worker/base_worker.go:50 `unreachable func: BaseWorker.Name` | unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing |

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

32 finding(s) require manual verification before removal (see Violations table, rule H5).


---

## Final Audit (feature 017 — run 20260830-180210)

### SC-003 accuracy audit (20-sample, 2026-08-30)

20 vi phạm lấy mẫu định kỳ từ snapshot, đối chiếu trực tiếp với mã nguồn: **20/20 đúng = 100%**
(ngưỡng SC-003 ≥ 90%). Phân bố mẫu: H1 ×11 (ngưỡng dòng, vị trí hàm), H2 ×1, H6 ×4, H8 ×4.

### SC-001 — file length: PASS

`files_over_500 = 0`, `files_over_800 = 0`. Baseline đầu (run 20260830-164823): 10 file > 500
(1 file fail 958 dòng) — toàn bộ god file đã tách theo responsibility.

### SC-002 — module boundary & layering: PASS

`cross_module_prohibited_imports = 0`, `layer_skips = 0`. Baseline đầu: 11 layer skips + 8
domain-repository→application/dto imports. Đã sửa qua: outbox port về `shared/domain/repositories`,
ports `IRawMessagePublisher`/`IOutboxGaugeProbe` ở `shared/application/event`, notification email
usecase tách khỏi contract, read-model DTOs về `domain/repositories` (alias tương thích trong
`application/dto`).

### H5 dead code audit (T033)

98 candidates: **66 đã xóa** (bằng chứng: symbol chỉ xuất hiện tại file khai báo — xem
`h5-verification.json`; backup tại `reports/h5-backup/`), **32 giữ nguyên** needs-confirmation
(symbol còn tham chiếu trong repo — cần xác nhận tay từng trường hợp).

### H6 duplication audit (T034)

20 cụm nghi vấn = file trùng tên (`helper.go`, `errors.go`...) giữa các module — idiom Go hợp lệ
trong modular monolith; gộp sẽ vi phạm ranh giới module (CHK013) → giữ nguyên có lý do.

### H8 error handling audit (T036)

41 cảnh báo (8 `errors.New` trần + 33 `fmt.Errorf` thiếu `%w`). Mỗi site cần quyết định mapping
HTTP status riêng theo §V; chuyển đổi loạt sẽ đổi hành vi API (vi phạm FR-005) → giữ nguyên làm
danh sách hygiene pass tiếp theo.

### H7 naming (T035)

0 vi phạm — không có method repository dùng tiền tố `Find...`.

### SC-006 — navigability (2026-08-30)

Định vị được 5 chức năng đại diện chỉ qua tên folder/file:
1. Cộng điểm loyalty brand → `internal/modules/brand/application/usecase/loyalty/points_uc.go`
2. Worker phân tích AI trang phục → `internal/modules/fashion/application/usecase/worker/analyze_uc.go`
3. Cost control AI → `internal/modules/aiexecutor/application/usecase/cost/`
4. Quản trị user → `internal/modules/identity/application/usecase/admin_uc.go`
5. Dynamic dashboard query → `internal/modules/dashboard/application/usecase/dynamic_query_execute_uc.go`

### Sự cố đã khắc phục trong phase 8

Regenerate swagger phát hiện 6 block swagger annotation bị đứt khỏi hàm sau khi tách file handler
(các endpoint POST /brand-portal/brands, loyalty points/benefits/claim-token, feedbacks, lots mất
tags hoặc biến mất khỏi spec). Đã gắn lại đầy đủ; swagger-split sinh lại đúng 154 paths / 8 module
specs như baseline HEAD.
