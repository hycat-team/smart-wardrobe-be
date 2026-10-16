-- +goose Up
-- +goose StatementBegin
-- Move 36 tables + 1 sequence from schema public into module schemas.
-- (030 handover: aiexecutor x6 and dashboard x3 are created directly in
-- their own schemas by the owning services, no relocation here.)
-- Pure SET SCHEMA migration (no data change); Down fully provided.
-- Cut 5 cross-module non-users FKs (enforced via contract at app layer).
-- FKs to users are KEPT (cross-schema FKs follow the table OID).

-- 1. Create module schemas
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS wardrobe;
CREATE SCHEMA IF NOT EXISTS fashion;
CREATE SCHEMA IF NOT EXISTS brand;
CREATE SCHEMA IF NOT EXISTS subscription;
-- 030 handover: CREATE SCHEMA aiexecutor/dashboard moved to the owning services.
CREATE SCHEMA IF NOT EXISTS shared;

-- 2. Cut 5 cross-module non-users FKs (before SET SCHEMA)
ALTER TABLE public.subscription_plans DROP CONSTRAINT IF EXISTS subscription_plans_ai_cost_policy_id_fkey;
ALTER TABLE public.wardrobe_items DROP CONSTRAINT IF EXISTS fk_wardrobe_items_fashion_item;
ALTER TABLE public.outfit_items DROP CONSTRAINT IF EXISTS fk_outfit_items_fashion_item;
ALTER TABLE public.brand_items DROP CONSTRAINT IF EXISTS brand_items_fashion_item_id_fkey;
ALTER TABLE public.digital_sample_responses DROP CONSTRAINT IF EXISTS digital_sample_responses_outfit_id_fkey;

-- 3. Move tables per module (indexes + auto-owned sequence follow tables;
-- `deposit_transactions_order_code_seq` is OWNED BY deposit_transactions.order_code
-- so it moves to subscription together with the deposit_transactions table)
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
-- 030 handover: aiexecutor x6 + dashboard x3 are created directly in their own
-- schemas by the AI/Commerce baselines, no relocation here.
-- shared
ALTER TABLE public.outbox_records SET SCHEMA shared;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Rollback: move 36 tables + 1 sequence back to public, re-add 4 cross-module FKs
-- (030 handover: aiexecutor/dashboard no longer move in Up, so Down
-- reverses nothing for them; their schemas belong to the owning services).
-- (check for orphan rows before ADD CONSTRAINT), drop module schemas.
-- (auto-owned sequence follows deposit_transactions back to public)

-- 1. Move tables back to public (indexes follow tables)
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
-- 030 handover: no aiexecutor/dashboard relocation in Up, so none to reverse.
-- shared
ALTER TABLE shared.outbox_records SET SCHEMA public;

-- 3. re-add 4 cross-module FKs - check for orphan rows before ADD CONSTRAINT
DO $$
BEGIN
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
END $$;

ALTER TABLE public.wardrobe_items
    ADD CONSTRAINT fk_wardrobe_items_fashion_item FOREIGN KEY (fashion_item_id) REFERENCES public.fashion_items(id) ON DELETE RESTRICT;
ALTER TABLE public.outfit_items
    ADD CONSTRAINT fk_outfit_items_fashion_item FOREIGN KEY (fashion_item_id) REFERENCES public.fashion_items(id) ON DELETE RESTRICT;
ALTER TABLE public.brand_items
    ADD CONSTRAINT brand_items_fashion_item_id_fkey FOREIGN KEY (fashion_item_id) REFERENCES public.fashion_items(id) ON DELETE CASCADE;
ALTER TABLE public.digital_sample_responses
    ADD CONSTRAINT digital_sample_responses_outfit_id_fkey FOREIGN KEY (outfit_id) REFERENCES public.outfits(id) ON DELETE SET NULL;

-- 4. Drop module schemas (empty after moving tables back to public)
DROP SCHEMA IF EXISTS identity;
DROP SCHEMA IF EXISTS wardrobe;
DROP SCHEMA IF EXISTS fashion;
DROP SCHEMA IF EXISTS brand;
DROP SCHEMA IF EXISTS subscription;
-- 030 handover: aiexecutor/dashboard schemas belong to their owners.
DROP SCHEMA IF EXISTS shared;
-- +goose StatementEnd
