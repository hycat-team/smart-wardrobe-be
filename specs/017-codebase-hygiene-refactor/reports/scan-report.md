# Hygiene Scan Report

**Run ID**: `20260830-203513` | **Generated**: 2026-08-30T20:35:13+07:00

Scanned 635 production files (+57 test files). Violations: **0 fail**, **50 warning**, **0 needs-confirmation**.

## Summary

Compared against baseline `20260830-203425` (↑ worse, ↓ better, = unchanged).

| Rule | fail | warning | needs-confirmation |
|------|------|---------|--------------------|
| H1 | 0 (=) | 50 (=) | 0 (=) |

## By Rule

| Key | fail | warning | needs-confirmation |
|-----|------|---------|--------------------|
| H1 | 0 | 50 | 0 |

## By Module

| Key | fail | warning | needs-confirmation |
|-----|------|---------|--------------------|
| aiexecutor | 0 | 2 | 0 |
| brand | 0 | 7 | 0 |
| dashboard | 0 | 3 | 0 |
| fashion | 0 | 8 | 0 |
| identity | 0 | 2 | 0 |
| internal/bootstrap | 0 | 1 | 0 |
| internal/pkg | 0 | 1 | 0 |
| internal/shared | 0 | 8 | 0 |
| internal/tools | 0 | 1 | 0 |
| notification | 0 | 1 | 0 |
| subscription | 0 | 7 | 0 |
| wardrobe | 0 | 9 | 0 |

## Violations

| Rule | Severity | Location | Detail |
|------|----------|----------|--------|
| H1 | warning | internal/bootstrap/app.go:78 `App.Run` | function spans 102 lines (threshold 80) |
| H1 | warning | internal/modules/aiexecutor/application/usecase/cost/usecase.go:43 `CostUseCase.Prepare` | function spans 93 lines (threshold 80) |
| H1 | warning | internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go:193 `OrchestratorUseCase.GenerateChatTextStream` | function spans 92 lines (threshold 80) |
| H1 | warning | internal/modules/brand/application/usecase/benefit/usecase_test.go | accepted-keep: test file length accepted (table-driven fixtures); threshold already relaxed +50% |
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
| H1 | warning | internal/shared/presentation/middleware/error_handler.go:14 `GlobalErrorHandler` | function spans 89 lines (threshold 80) |
| H1 | warning | internal/shared/presentation/middleware/swagger_lock.go:12 `SwaggerLockMiddleware` | function spans 193 lines (threshold 80) |
| H1 | warning | internal/tools/swagger-split/main.go:44 `run` | function spans 116 lines (threshold 80) |
| H2 | warning | internal/modules/dashboard/domain/entities/dashboard_snapshots.go | accepted-keep: three daily-snapshot aggregates (admin/brand/user) serve the same dashboard concern |
| H2 | warning | internal/shared/application/event/consumer.go | accepted-keep: Message/PermanentError/RetryableError serve one concern — the event consumer contract |
| H2 | warning | internal/shared/domain/entities/jsonb.go | accepted-keep: JSONDocument plus its small parse helpers serve one concern (JSONB column handling) |
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


## Final Audit — Phase 9 Convergence (2026-08-30, run 20260830-193926; tune heuristic run 20260830-203513)

### Leak fix (FR-005 ngoại lệ security fix + T070–T076)

- Biên mapper (`apperror.From`/`Wrap`): error không nhận diện được trả `GenericInternalMessage`;
  raw text chỉ trong `Cause` (log) và `debugMessage` (chỉ `appEnv == "development"`).
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
- 5× `errors.New` one-off còn lại → keep-reason trong accepted.json.

### H5 (32 candidate needs-confirmation)

- **19 symbol xóa** (bằng chứng: 0 tham chiếu ngoài khai báo, hoặc duplicate của bản đang dùng) —
  liệt kê đầy đủ trong commit 27c69ab..be5578e và h5-verification.json.
- **13 symbol giữ với keep-reason** (interface contract / framework accessor / test-referenced)
  trong `internal/tools/hygiene-scan/accepted.json` (FR-011).

### Tune heuristic H2/H6 (2026-08-30, sau review developer)

- H2: miễn `table_name.go` (boilerplate `TableName()` GORM gom theo module là chủ đích);
  3 file H2 còn lại giữ với keep-reason (consumer.go, jsonb.go, dashboard_snapshots.go —
  cùng mối quan tâm).
- H6: same-name-file heuristic nghỉ hổ — kiến trúc modular monolith lặp lại archetype thư mục ở
  mọi module nên filename equality không bao giờ chỉ ra trùng lặp thật; H6 chuyển thành audit
  thủ công theo code-hygiene.md. Test khóa: H6 không bao giờ emit tự động.

### Kết quả số lần scan cuối (run 20260830-203513)

- violations: **0 fail, 50 warning — toàn bộ H1 function-length (nợ kỹ thuật thật, theo dõi xu
  hướng ↓)**, 0 needs-confirmation.
- accepted-keep: **22**.
- metrics: files_over_500=0, files_over_800=0, cross_module_prohibited_imports=0, layer_skips=0.
- Kiểm chứng: `make wire` ✓, `make lint` ✓, `make test-race` ✓ exit 0.
- Snapshots giữ 3 mốc đối chiếu: `164823` (trước refactor) → `180210` (sau Phase 1–8) →
  `203513` (sau Phase 9 + tune heuristic).
