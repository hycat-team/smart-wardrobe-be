# Schema Separation Mapping — Sprint 2 (US4)

**Trạng thái**: Thiết kế (feature `011-assess-schema-separation`)
**Ngày**: 2026-08-02
**Nguồn**: query `pg_tables`/`pg_constraint`/`pg_indexes` trên DB thực (`smart_wardrobe_db`), đối chiếu `database/scripts/closy_db-schema.sql`.

> **Đính chính**: DB thực có **45 bảng** (data-model.md ghi 42 — thiếu `provider_payment_events`, `provider_webhook_inbox`, `subscription_renewal_attempts`). Số bảng chuẩn = 45 + 1 sequence.

## 1. Bảng → schema đích (45 bảng + 1 sequence)

### identity (3)
`users`, `refresh_tokens`, `user_style_profiles`

### wardrobe (3)
`wardrobe_items`, `outfits`, `outfit_items`

### fashion (4)
`fashion_items`, `categories`, `conversational_contexts`, `messages`

### brand (15)
`brands`, `brand_members`, `brand_customers`, `brand_customer_claims`, `loyalty_programs`, `loyalty_tiers`, `loyalty_accounts`, `loyalty_point_transactions`, `loyalty_point_lots`, `brand_benefits`, `benefit_redemptions`, `brand_conversations`, `brand_conversation_messages`, `brand_items`, `digital_sample_responses`

### subscription (10 + 1 sequence)
`subscription_plans`, `user_subscriptions`, `user_daily_quotas`, `user_wallets`, `deposit_transactions`, `wallet_statements`, `provider_payment_events`, `provider_webhook_inbox`, `user_subscription_events`, `subscription_renewal_attempts` (+ sequence `deposit_transactions_order_code_seq`)

### aiexecutor (6)
`ai_cost_policies`, `ai_cost_policy_operations`, `ai_policy_grants`, `ai_usage_period_ledgers`, `ai_usage_events`, `ai_model_configs`

### dashboard (3)
`dashboard_admin_daily_snapshots`, `dashboard_brand_daily_snapshots`, `dashboard_user_wardrobe_summaries`

### shared (1)
`outbox_records`

## 2. FK xuyên module — xử lý theo Q1/Q2

### 2.1 CẮT (phi-users, enforce qua contract) — 9 FK

| # | Constraint | Bảng nguồn → Bảng đích | Đến module |
|---|-----------|--------------------------|------------|
| 1 | `subscription_plans_ai_cost_policy_id_fkey` | `subscription_plans.ai_cost_policy_id` → `ai_cost_policies` | subscription → aiexecutor |
| 2 | `user_ai_policy_grants_plan_id_fkey` | `ai_policy_grants.plan_id` → `subscription_plans` | aiexecutor → subscription |
| 3 | `user_ai_policy_grants_source_deposit_transaction_id_fkey` | `ai_policy_grants.source_deposit_transaction_id` → `deposit_transactions` | aiexecutor → subscription |
| 4 | `user_ai_policy_grants_source_event_id_fkey` | `ai_policy_grants.source_event_id` → `user_subscription_events` | aiexecutor → subscription |
| 5 | `fk_wardrobe_items_fashion_item` | `wardrobe_items.fashion_item_id` → `fashion_items` | wardrobe → fashion |
| 6 | `fk_outfit_items_fashion_item` | `outfit_items.fashion_item_id` → `fashion_items` | wardrobe → fashion |
| 7 | `brand_items_fashion_item_id_fkey` | `brand_items.fashion_item_id` → `fashion_items` | brand → fashion |
| 8 | `digital_sample_responses_outfit_id_fkey` | `digital_sample_responses.outfit_id` → `outfits` | brand → wardrobe |
| 9 | `dashboard_brand_daily_snapshots_brand_id_fkey` | `dashboard_brand_daily_snapshots.brand_id` → `brands` | dashboard → brand |

### 2.2 GIỮ (xuyên schema tới `users`) — 30 FK

FK trỏ tới `users` được GIỮ nguyên (PostgreSQL FK xuyên schema theo OID bảng — `SET SCHEMA` không phá constraint, pg_dump sẽ hiển thị schema-qualified `identity.users`):

- identity: `refresh_tokens_user_id_fkey`, `user_style_profiles_user_id_fkey`
- subscription: `wallet_statements_user_id_fkey`, `user_subscription_events_user_id_fkey`, `user_subscription_events_actor_admin_id_fkey`, `subscription_renewal_attempts_user_id_fkey`, `user_subscriptions_user_id_fkey`, `deposit_transactions_user_id_fkey`, `user_daily_quotas_user_id_fkey`, `user_wallets_user_id_fkey`
- aiexecutor: `ai_model_configs_created_by_fkey`, `ai_usage_events_user_id_fkey`, `ai_usage_period_ledgers_user_id_fkey`, `user_ai_policy_grants_user_id_fkey`
- wardrobe: `outfits_user_id_fkey`, `wardrobe_items_user_id_fkey`
- fashion: `conversational_contexts_user_id_fkey`
- brand: `brand_customers_user_id_fkey`, `brand_conversations_user_id_fkey`, `brand_conversations_closed_by_user_id_fkey`, `brand_conversation_messages_sender_user_id_fkey`, `brand_customer_claims_revoked_by_user_id_fkey`, `brands_approved_by_user_id_fkey`, `brands_created_by_user_id_fkey`, `brand_members_user_id_fkey`, `benefit_redemptions_user_id_fkey`, `digital_sample_responses_user_id_fkey`, `loyalty_accounts_user_id_fkey`, `loyalty_point_lots_user_id_fkey`, `loyalty_point_transactions_user_id_fkey`, `loyalty_point_transactions_created_by_user_id_fkey`
- dashboard: `dashboard_user_wardrobe_summaries_user_id_fkey`

### 2.3 FK nội bộ module — GIỮ (không đổi)

Toàn bộ FK còn lại (41 FK) trỏ giữa các bảng cùng module sở hữu — `SET SCHEMA` theo bảng tự giữ, không cần drop/add.

## 3. Index

Khi `ALTER TABLE ... SET SCHEMA`, mọi index của bảng (kể cả unique/partial/expression/HNSW `fitems_embedding_cosine_idx`, `ustyles_embedding_cosine_idx`, GIN `idx_fashion_items_lexical_search`) tự động chuyển schema theo bảng — không cần `ALTER INDEX` riêng.

## 4. Sequence

`deposit_transactions_order_code_seq` → `ALTER SEQUENCE public.deposit_transactions_order_code_seq SET SCHEMA subscription`.

## 5. Access Matrix — module nào đọc/ghi schema nào (qua contract)

> 1 connection chung (Q3): mọi module **có thể** query mọi schema qua `search_path` — không phân quyền DB-level. Boundary enforce ở **app layer** (Hiến pháp II): chỉ truy cập dữ liệu module khác qua `presentation/contract/`, cấm import entity/repo cross-module (SC-004 verified).

| Module (caller) | Sở hữu schema | Truy cập schema module khác | Kênh contract |
|---|---|---|---|
| **wardrobe** | `wardrobe` | `fashion` (đọc category/fashion item; write reassign category) | `IFashionContract` |
| **fashion** | `fashion` | `wardrobe` (đọc item, update status, delete) | `IWardrobeContract` |
|  |  | `subscription` (đọc quota/overview) | `IUserSubscriptionContract` |
|  |  | `brand` (đọc brand item cho styling) | `IBrandContract` |
|  |  | `identity` (đọc user) | `IUserContract` |
| **brand** | `brand` | `identity` (đọc user theo id/username/email) | `IUserContract` |
| **subscription** | `subscription` | `identity` (auth middleware — router) | `IAuthMiddleware` |
| **aiexecutor** | `aiexecutor` | `subscription` (đọc overview/quotas) | `IUserSubscriptionContract` |
| **identity** | `identity` | `notification` (gửi email/SMS) | `INotificationContract` |
|  |  | `subscription` (đọc subscription overview) | `IUserSubscriptionContract` |
| **dashboard** | `dashboard` | `aiexecutor` | `IAICostDashboardReader` |
|  |  | `brand` | `IBrandDashboardReader` |
|  |  | `fashion` | `IFashionDashboardReader` |
|  |  | `identity` | `IUserDashboardReader` |
|  |  | `subscription` | `ISubscriptionDashboardReader` |
|  |  | `wardrobe` | `IWardrobeDashboardReader` |

**Chiều dữ liệu**: dashboard chỉ **đọc** qua DashboardReader; fashion/wardrobe/brand có đọc + ghi hạn chế qua method contract được expose (`IFashionContract.ReassignCategoryForCatalogItems`, `IWardrobeContract.UpdateItemStatus`, ...); FK cross-module đã cắt → mọi ghi chéo qua contract, không raw UPDATE.

## 6. Down (rollback)

1. `ALTER SEQUENCE ... SET SCHEMA public`.
2. `SET SCHEMA public` ngược lại cho 45 bảng.
3. Tái lập 9 FK cắt — **check dữ liệu không orphan trước `ADD CONSTRAINT`**.
4. `DROP SCHEMA` các module (sau khi xác nhận rỗng).
