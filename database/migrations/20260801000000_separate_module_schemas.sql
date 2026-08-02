-- +goose Up
-- +goose StatementBegin
-- Tách 45 bảng + 1 sequence từ schema public sang schema theo module.
-- Migration thuần SET SCHEMA (không đổi dữ liệu) → viết Down đầy đủ.
-- Cắt 9 FK xuyên module phi-users (enforce qua contract ở app layer).
-- FK trỏ tới users được GIỮ (FK xuyên schema theo OID bảng).

-- 1. Tạo schema theo module
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS wardrobe;
CREATE SCHEMA IF NOT EXISTS fashion;
CREATE SCHEMA IF NOT EXISTS brand;
CREATE SCHEMA IF NOT EXISTS subscription;
CREATE SCHEMA IF NOT EXISTS aiexecutor;
CREATE SCHEMA IF NOT EXISTS dashboard;
CREATE SCHEMA IF NOT EXISTS shared;

-- 2. Cắt 9 FK xuyên module phi-users (trước khi SET SCHEMA)
ALTER TABLE public.subscription_plans DROP CONSTRAINT IF EXISTS subscription_plans_ai_cost_policy_id_fkey;
ALTER TABLE public.ai_policy_grants DROP CONSTRAINT IF EXISTS user_ai_policy_grants_plan_id_fkey;
ALTER TABLE public.ai_policy_grants DROP CONSTRAINT IF EXISTS user_ai_policy_grants_source_deposit_transaction_id_fkey;
ALTER TABLE public.ai_policy_grants DROP CONSTRAINT IF EXISTS user_ai_policy_grants_source_event_id_fkey;
ALTER TABLE public.wardrobe_items DROP CONSTRAINT IF EXISTS fk_wardrobe_items_fashion_item;
ALTER TABLE public.outfit_items DROP CONSTRAINT IF EXISTS fk_outfit_items_fashion_item;
ALTER TABLE public.brand_items DROP CONSTRAINT IF EXISTS brand_items_fashion_item_id_fkey;
ALTER TABLE public.digital_sample_responses DROP CONSTRAINT IF EXISTS digital_sample_responses_outfit_id_fkey;
ALTER TABLE public.dashboard_brand_daily_snapshots DROP CONSTRAINT IF EXISTS dashboard_brand_daily_snapshots_brand_id_fkey;

-- 3. Di chuyển bảng theo module (index + sequence auto-owned tự đi theo bảng;
-- `deposit_transactions_order_code_seq` được OWNED BY deposit_transactions.order_code
-- nên sẽ tự động SET SCHEMA subscription khi bảng deposit_transactions di chuyển)
-- identity
ALTER TABLE public.users SET SCHEMA identity;
ALTER TABLE public.refresh_tokens SET SCHEMA identity;
ALTER TABLE public.user_style_profiles SET SCHEMA identity;
-- wardrobe
ALTER TABLE public.wardrobe_items SET SCHEMA wardrobe;
ALTER TABLE public.outfits SET SCHEMA wardrobe;
ALTER TABLE public.outfit_items SET SCHEMA wardrobe;
-- fashion
ALTER TABLE public.fashion_items SET SCHEMA fashion;
ALTER TABLE public.categories SET SCHEMA fashion;
ALTER TABLE public.conversational_contexts SET SCHEMA fashion;
ALTER TABLE public.messages SET SCHEMA fashion;
-- brand
ALTER TABLE public.brands SET SCHEMA brand;
ALTER TABLE public.brand_members SET SCHEMA brand;
ALTER TABLE public.brand_customers SET SCHEMA brand;
ALTER TABLE public.brand_customer_claims SET SCHEMA brand;
ALTER TABLE public.loyalty_programs SET SCHEMA brand;
ALTER TABLE public.loyalty_tiers SET SCHEMA brand;
ALTER TABLE public.loyalty_accounts SET SCHEMA brand;
ALTER TABLE public.loyalty_point_transactions SET SCHEMA brand;
ALTER TABLE public.loyalty_point_lots SET SCHEMA brand;
ALTER TABLE public.brand_benefits SET SCHEMA brand;
ALTER TABLE public.benefit_redemptions SET SCHEMA brand;
ALTER TABLE public.brand_conversations SET SCHEMA brand;
ALTER TABLE public.brand_conversation_messages SET SCHEMA brand;
ALTER TABLE public.brand_items SET SCHEMA brand;
ALTER TABLE public.digital_sample_responses SET SCHEMA brand;
-- subscription
ALTER TABLE public.subscription_plans SET SCHEMA subscription;
ALTER TABLE public.user_subscriptions SET SCHEMA subscription;
ALTER TABLE public.user_daily_quotas SET SCHEMA subscription;
ALTER TABLE public.user_wallets SET SCHEMA subscription;
ALTER TABLE public.deposit_transactions SET SCHEMA subscription;
ALTER TABLE public.wallet_statements SET SCHEMA subscription;
ALTER TABLE public.provider_payment_events SET SCHEMA subscription;
ALTER TABLE public.provider_webhook_inbox SET SCHEMA subscription;
ALTER TABLE public.user_subscription_events SET SCHEMA subscription;
ALTER TABLE public.subscription_renewal_attempts SET SCHEMA subscription;
-- aiexecutor
ALTER TABLE public.ai_cost_policies SET SCHEMA aiexecutor;
ALTER TABLE public.ai_cost_policy_operations SET SCHEMA aiexecutor;
ALTER TABLE public.ai_policy_grants SET SCHEMA aiexecutor;
ALTER TABLE public.ai_usage_period_ledgers SET SCHEMA aiexecutor;
ALTER TABLE public.ai_usage_events SET SCHEMA aiexecutor;
ALTER TABLE public.ai_model_configs SET SCHEMA aiexecutor;
-- dashboard
ALTER TABLE public.dashboard_admin_daily_snapshots SET SCHEMA dashboard;
ALTER TABLE public.dashboard_brand_daily_snapshots SET SCHEMA dashboard;
ALTER TABLE public.dashboard_user_wardrobe_summaries SET SCHEMA dashboard;
-- shared
ALTER TABLE public.outbox_records SET SCHEMA shared;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Rollback: đưa 45 bảng + 1 sequence về public, tái lập 9 FK xuyên module
-- (kiểm tra dữ liệu không orphan trước khi ADD CONSTRAINT), drop schema module.
-- (sequence auto-owned đi theo bảng deposit_transactions về public)

-- 1. Di chuyển bảng về public (index tự đi theo bảng)
-- identity
ALTER TABLE identity.users SET SCHEMA public;
ALTER TABLE identity.refresh_tokens SET SCHEMA public;
ALTER TABLE identity.user_style_profiles SET SCHEMA public;
-- wardrobe
ALTER TABLE wardrobe.wardrobe_items SET SCHEMA public;
ALTER TABLE wardrobe.outfits SET SCHEMA public;
ALTER TABLE wardrobe.outfit_items SET SCHEMA public;
-- fashion
ALTER TABLE fashion.fashion_items SET SCHEMA public;
ALTER TABLE fashion.categories SET SCHEMA public;
ALTER TABLE fashion.conversational_contexts SET SCHEMA public;
ALTER TABLE fashion.messages SET SCHEMA public;
-- brand
ALTER TABLE brand.brands SET SCHEMA public;
ALTER TABLE brand.brand_members SET SCHEMA public;
ALTER TABLE brand.brand_customers SET SCHEMA public;
ALTER TABLE brand.brand_customer_claims SET SCHEMA public;
ALTER TABLE brand.loyalty_programs SET SCHEMA public;
ALTER TABLE brand.loyalty_tiers SET SCHEMA public;
ALTER TABLE brand.loyalty_accounts SET SCHEMA public;
ALTER TABLE brand.loyalty_point_transactions SET SCHEMA public;
ALTER TABLE brand.loyalty_point_lots SET SCHEMA public;
ALTER TABLE brand.brand_benefits SET SCHEMA public;
ALTER TABLE brand.benefit_redemptions SET SCHEMA public;
ALTER TABLE brand.brand_conversations SET SCHEMA public;
ALTER TABLE brand.brand_conversation_messages SET SCHEMA public;
ALTER TABLE brand.brand_items SET SCHEMA public;
ALTER TABLE brand.digital_sample_responses SET SCHEMA public;
-- subscription
ALTER TABLE subscription.subscription_plans SET SCHEMA public;
ALTER TABLE subscription.user_subscriptions SET SCHEMA public;
ALTER TABLE subscription.user_daily_quotas SET SCHEMA public;
ALTER TABLE subscription.user_wallets SET SCHEMA public;
ALTER TABLE subscription.deposit_transactions SET SCHEMA public;
ALTER TABLE subscription.wallet_statements SET SCHEMA public;
ALTER TABLE subscription.provider_payment_events SET SCHEMA public;
ALTER TABLE subscription.provider_webhook_inbox SET SCHEMA public;
ALTER TABLE subscription.user_subscription_events SET SCHEMA public;
ALTER TABLE subscription.subscription_renewal_attempts SET SCHEMA public;
-- aiexecutor
ALTER TABLE aiexecutor.ai_cost_policies SET SCHEMA public;
ALTER TABLE aiexecutor.ai_cost_policy_operations SET SCHEMA public;
ALTER TABLE aiexecutor.ai_policy_grants SET SCHEMA public;
ALTER TABLE aiexecutor.ai_usage_period_ledgers SET SCHEMA public;
ALTER TABLE aiexecutor.ai_usage_events SET SCHEMA public;
ALTER TABLE aiexecutor.ai_model_configs SET SCHEMA public;
-- dashboard
ALTER TABLE dashboard.dashboard_admin_daily_snapshots SET SCHEMA public;
ALTER TABLE dashboard.dashboard_brand_daily_snapshots SET SCHEMA public;
ALTER TABLE dashboard.dashboard_user_wardrobe_summaries SET SCHEMA public;
-- shared
ALTER TABLE shared.outbox_records SET SCHEMA public;

-- 3. Tái lập 9 FK xuyên module — check dữ liệu không orphan trước khi ADD CONSTRAINT
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.subscription_plans sp LEFT JOIN public.ai_cost_policies acp ON sp.ai_cost_policy_id = acp.id WHERE sp.ai_cost_policy_id IS NOT NULL AND acp.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in subscription_plans.ai_cost_policy_id';
    END IF;
    IF EXISTS (SELECT 1 FROM public.ai_policy_grants g LEFT JOIN public.subscription_plans sp ON g.plan_id = sp.id WHERE g.plan_id IS NOT NULL AND sp.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in ai_policy_grants.plan_id';
    END IF;
    IF EXISTS (SELECT 1 FROM public.ai_policy_grants g LEFT JOIN public.deposit_transactions dt ON g.source_deposit_transaction_id = dt.id WHERE g.source_deposit_transaction_id IS NOT NULL AND dt.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in ai_policy_grants.source_deposit_transaction_id';
    END IF;
    IF EXISTS (SELECT 1 FROM public.ai_policy_grants g LEFT JOIN public.user_subscription_events e ON g.source_event_id = e.id WHERE g.source_event_id IS NOT NULL AND e.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in ai_policy_grants.source_event_id';
    END IF;
    IF EXISTS (SELECT 1 FROM public.wardrobe_items wi LEFT JOIN public.fashion_items fi ON wi.fashion_item_id = fi.id WHERE wi.fashion_item_id IS NOT NULL AND fi.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in wardrobe_items.fashion_item_id';
    END IF;
    IF EXISTS (SELECT 1 FROM public.outfit_items oi LEFT JOIN public.fashion_items fi ON oi.fashion_item_id = fi.id WHERE oi.fashion_item_id IS NOT NULL AND fi.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in outfit_items.fashion_item_id';
    END IF;
    IF EXISTS (SELECT 1 FROM public.brand_items bi LEFT JOIN public.fashion_items fi ON bi.fashion_item_id = fi.id WHERE bi.fashion_item_id IS NOT NULL AND fi.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in brand_items.fashion_item_id';
    END IF;
    IF EXISTS (SELECT 1 FROM public.digital_sample_responses dsr LEFT JOIN public.outfits o ON dsr.outfit_id = o.id WHERE dsr.outfit_id IS NOT NULL AND o.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in digital_sample_responses.outfit_id';
    END IF;
    IF EXISTS (SELECT 1 FROM public.dashboard_brand_daily_snapshots s LEFT JOIN public.brands b ON s.brand_id = b.id WHERE s.brand_id IS NOT NULL AND b.id IS NULL) THEN
        RAISE EXCEPTION 'orphan rows in dashboard_brand_daily_snapshots.brand_id';
    END IF;
END $$;

ALTER TABLE public.subscription_plans
    ADD CONSTRAINT subscription_plans_ai_cost_policy_id_fkey FOREIGN KEY (ai_cost_policy_id) REFERENCES public.ai_cost_policies(id) ON DELETE RESTRICT;
ALTER TABLE public.ai_policy_grants
    ADD CONSTRAINT user_ai_policy_grants_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.subscription_plans(id) ON DELETE RESTRICT;
ALTER TABLE public.ai_policy_grants
    ADD CONSTRAINT user_ai_policy_grants_source_deposit_transaction_id_fkey FOREIGN KEY (source_deposit_transaction_id) REFERENCES public.deposit_transactions(id) ON DELETE SET NULL;
ALTER TABLE public.ai_policy_grants
    ADD CONSTRAINT user_ai_policy_grants_source_event_id_fkey FOREIGN KEY (source_event_id) REFERENCES public.user_subscription_events(id) ON DELETE SET NULL;
ALTER TABLE public.wardrobe_items
    ADD CONSTRAINT fk_wardrobe_items_fashion_item FOREIGN KEY (fashion_item_id) REFERENCES public.fashion_items(id) ON DELETE RESTRICT;
ALTER TABLE public.outfit_items
    ADD CONSTRAINT fk_outfit_items_fashion_item FOREIGN KEY (fashion_item_id) REFERENCES public.fashion_items(id) ON DELETE RESTRICT;
ALTER TABLE public.brand_items
    ADD CONSTRAINT brand_items_fashion_item_id_fkey FOREIGN KEY (fashion_item_id) REFERENCES public.fashion_items(id) ON DELETE CASCADE;
ALTER TABLE public.digital_sample_responses
    ADD CONSTRAINT digital_sample_responses_outfit_id_fkey FOREIGN KEY (outfit_id) REFERENCES public.outfits(id) ON DELETE SET NULL;
ALTER TABLE public.dashboard_brand_daily_snapshots
    ADD CONSTRAINT dashboard_brand_daily_snapshots_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES public.brands(id) ON DELETE CASCADE;

-- 4. Drop schema module (đã trống sau khi chuyển bảng về public)
DROP SCHEMA IF EXISTS identity;
DROP SCHEMA IF EXISTS wardrobe;
DROP SCHEMA IF EXISTS fashion;
DROP SCHEMA IF EXISTS brand;
DROP SCHEMA IF EXISTS subscription;
DROP SCHEMA IF EXISTS aiexecutor;
DROP SCHEMA IF EXISTS dashboard;
DROP SCHEMA IF EXISTS shared;
-- +goose StatementEnd
