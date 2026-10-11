-- Cleanup: Xoá toàn bộ seed data của feature 006 để chạy lại seed script
-- Chạy script này TRƯỚC KHI chạy seed_feature_006_dashboard.sql
-- Thứ tự TRUNCATE ngược chiều foreign key

TRUNCATE TABLE dashboard.dashboard_user_wardrobe_summaries CASCADE;

TRUNCATE TABLE dashboard.dashboard_brand_daily_snapshots CASCADE;
TRUNCATE TABLE dashboard.dashboard_admin_daily_snapshots CASCADE;
TRUNCATE TABLE brand.digital_sample_responses CASCADE;
TRUNCATE TABLE brand.benefit_redemptions CASCADE;
TRUNCATE TABLE brand.brand_conversation_messages CASCADE;
TRUNCATE TABLE brand.brand_conversations CASCADE;
TRUNCATE TABLE brand.brand_items CASCADE;
TRUNCATE TABLE aiexecutor.ai_usage_events CASCADE;
TRUNCATE TABLE aiexecutor.ai_usage_period_ledgers CASCADE;
TRUNCATE TABLE subscription.deposit_transactions CASCADE;
TRUNCATE TABLE brand.loyalty_point_lots CASCADE;
TRUNCATE TABLE brand.loyalty_point_transactions CASCADE;
TRUNCATE TABLE brand.brand_customers CASCADE;
TRUNCATE TABLE brand.brand_members CASCADE;
TRUNCATE TABLE brand.brand_benefits CASCADE;
TRUNCATE TABLE subscription.user_subscriptions CASCADE;
TRUNCATE TABLE subscription.user_daily_quotas CASCADE;
TRUNCATE TABLE subscription.user_wallets CASCADE;
-- identity.user_style_profiles đã bị xoá (ADR 015, migration 20261011090000)
TRUNCATE TABLE fashion.fashion_items CASCADE;
TRUNCATE TABLE brand.brands CASCADE;
TRUNCATE TABLE brand.loyalty_programs CASCADE;
TRUNCATE TABLE brand.loyalty_tiers CASCADE;
TRUNCATE TABLE brand.brand_customer_claims CASCADE;
