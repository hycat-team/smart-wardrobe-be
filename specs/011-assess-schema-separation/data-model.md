# Data Model: Xử lý tech debt cross-module & tách schema theo module

**Phase 1 output** — bản đồ schema hiện tại và schema đích theo module.

## 1. Schema hiện tại (single `public`)

Nguồn: `database/scripts/closy_db-schema.sql` (42 bảng + 1 sequence, tất cả ở `public`).

### 1.1 Bảng thuộc module sở hữu

| Module | Bảng |
|--------|------|
| **identity** | `users`, `refresh_tokens`, `user_style_profiles` |
| **wardrobe** | `wardrobe_items`, `outfits`, `outfit_items` |
| **fashion** | `fashion_items`, `categories`, `conversational_contexts`, `messages` |
| **brand** | `brands`, `brand_members`, `brand_customers`, `brand_customer_claims`, `loyalty_programs`, `loyalty_tiers`, `loyalty_accounts`, `loyalty_point_transactions`, `loyalty_point_lots`, `brand_benefits`, `benefit_redemptions`, `brand_conversations`, `brand_conversation_messages`, `brand_items`, `digital_sample_responses` |
| **subscription** | `subscription_plans`, `user_subscriptions`, `user_daily_quotas`, `user_wallets`, `deposit_transactions`, `wallet_statements`, `provider_payment_events`, `provider_webhook_inbox`, `user_subscription_events`, `subscription_renewal_attempts` |
| **aiexecutor** | `ai_cost_policies`, `ai_cost_policy_operations`, `ai_policy_grants`, `ai_usage_period_ledgers`, `ai_usage_events`, `ai_model_configs` |
| **dashboard** | `dashboard_admin_daily_snapshots`, `dashboard_brand_daily_snapshots`, `dashboard_user_wardrobe_summaries` |
| **shared** | `outbox_records` |

Sequence: `deposit_transactions_order_code_seq` (thuộc subscription, đi cùng `deposit_transactions`).

## 2. Schema đích (multi-schema theo module)

```
identity     → users, refresh_tokens, user_style_profiles
wardrobe     → wardrobe_items, outfits, outfit_items
fashion      → fashion_items, categories, conversational_contexts, messages
brand        → (15 bảng brand/loyalty/benefit)
subscription → (10 bảng) + deposit_transactions_order_code_seq
aiexecutor   → (6 bảng ai_*)
dashboard    → (3 bảng dashboard_*)
shared       → outbox_records
```

## 3. FK xuyên module — xử lý theo Q1/Q2

### 3.1 Giữ (FK xuyên schema tới `users`) — ~30 FK

`users` là hub trung tâm; các FK sau GIỮ, trỏ schema-qualified tới `identity.users`:

- `refresh_tokens.user_id`, `user_style_profiles.user_id` (identity nội bộ)
- `wallet_statements.user_id`, `user_subscription_events.user_id/actor_admin_id`, `subscription_renewal_attempts.user_id` (subscription)
- `ai_policy_grants.user_id`, `ai_usage_period_ledgers.user_id`, `ai_usage_events.user_id`, `ai_model_configs.created_by` (aiexecutor)
- `outfits.user_id` (wardrobe)
- `brand_customers.user_id`, `loyalty_accounts.user_id`, `loyalty_point_transactions.user_id/created_by_user_id`, `benefit_redemptions.user_id`, `brand_conversations.user_id/closed_by_user_id`, `brand_conversation_messages.sender_user_id`, `digital_sample_responses.user_id`, `brands.created_by_user_id/approved_by_user_id`, `brand_members.user_id`, `brand_customer_claims.revoked_by_user_id`, `loyalty_point_lots.user_id` (brand)
- `user_subscriptions.user_id`, `user_daily_quotas.user_id`, `user_wallets.user_id`, `deposit_transactions.user_id` (subscription)
- `dashboard_brand_daily_snapshots` → giữ brand_id? KHÔNG — thuộc mục cắt (dashboard→brand)
- `dashboard_user_wardrobe_summaries.user_id` (dashboard→identity, giữ)
- `conversational_contexts.user_id` (fashion→identity, giữ)

### 3.2 Cắt (FK xuyên module phi-users) — 9 FK

| FK | Từ module → Đến | Lý do cắt |
|----|------------------|-----------|
| `ai_policy_grants.plan_id` | aiexecutor → subscription | xuyên module |
| `ai_policy_grants.source_event_id` | aiexecutor → subscription | xuyên module |
| `ai_policy_grants.source_deposit_transaction_id` | aiexecutor → subscription | xuyên module |
| `subscription_plans.ai_cost_policy_id` | subscription → aiexecutor | xuyên module |
| `wardrobe_items.fashion_item_id` | wardrobe → fashion | xuyên module |
| `outfit_items.fashion_item_id` | wardrobe → fashion | xuyên module |
| `brand_items.fashion_item_id` | brand → fashion | xuyên module |
| `digital_sample_responses.outfit_id` | brand → wardrobe | xuyên module |
| `dashboard_brand_daily_snapshots.brand_id` | dashboard → brand | xuyên module |

> Lưu ý: danh sách đối chiếu từ `closy_db-schema.sql`; cần rà soát lại chính xác tên constraint ở phase thực hiện.

## 4. Entity & state transitions liên quan

### 4.1 Wardrobe item processing state (critical path worker)

`wardrobe_items.status`: InWardrobe(0) → Processing → InWardrobe / Failed / NeedsReview.

`fashion_items` processing fields (thuộc fashion module): `processing_retry_count`, `processing_version`, `processing_started_at`, `last_processing_attempt_at`, `processing_error_reason`, `review_reason`.

Sau Sprint 1 (Q5): wardrobe ghi `wardrobe_items` + publish outbox event; fashion consume event update `fashion_items`. **Phân định (I3)**: claim retry + processing state (mark failed / needs review / complete success) → outbox event `wardrobe.processing.state`; đọc stale info (`GetStaleProcessingInfo`) + reassign category (`ReassignCategoryForCatalogItems`) → đồng bộ qua `IFashionContract`.

### 4.2 GetHybridCandidates (AI recommendation)

Input: `userID`, `semanticVector`, `lexicalTerms`, `excludedTerms`, `hardFilters`, `limit`, `rrfK`.
Output: `[]HybridCandidate`.

Refactor (Q2 research): lấy `GetUserFashionItemIDs(userID)` từ wardrobe contract → query `fashion_items WHERE id IN (?) [filters] ORDER BY embedding <=> ? LIMIT ?` → RRF merge.

## 5. Validation rules giữ nguyên

- Không thay đổi schema dữ liệu (chỉ di chuyển schema + drop/add constraint).
- Không đổi semantics dữ liệu → migration thuần `SET SCHEMA` viết Down đầy đủ (Q4).
- Không thêm cột/bảng mới trong Sprint 2 (trừ event/outbox cần thiết trong Sprint 1).
