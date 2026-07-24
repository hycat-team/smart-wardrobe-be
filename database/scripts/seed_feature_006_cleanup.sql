-- Cleanup: Xoá toàn bộ seed data của feature 006 để chạy lại seed script
-- Chạy script này TRƯỚC KHI chạy seed_feature_006_dashboard.sql
-- Thứ tự TRUNCATE ngược chiều foreign key

TRUNCATE TABLE dashboard_user_wardrobe_summaries CASCADE;

TRUNCATE TABLE dashboard_brand_daily_snapshots CASCADE;
TRUNCATE TABLE dashboard_admin_daily_snapshots CASCADE;
TRUNCATE TABLE digital_sample_responses CASCADE;
TRUNCATE TABLE benefit_redemptions CASCADE;
TRUNCATE TABLE brand_conversation_messages CASCADE;
TRUNCATE TABLE brand_conversations CASCADE;
TRUNCATE TABLE brand_items CASCADE;
TRUNCATE TABLE ai_usage_events CASCADE;
TRUNCATE TABLE ai_usage_period_ledgers CASCADE;
TRUNCATE TABLE deposit_transactions CASCADE;
TRUNCATE TABLE loyalty_point_lots CASCADE;
TRUNCATE TABLE loyalty_point_transactions CASCADE;
TRUNCATE TABLE brand_customers CASCADE;
TRUNCATE TABLE brand_members CASCADE;
TRUNCATE TABLE brand_benefits CASCADE;
TRUNCATE TABLE user_subscriptions CASCADE;
TRUNCATE TABLE user_daily_quotas CASCADE;
TRUNCATE TABLE user_wallets CASCADE;
TRUNCATE TABLE user_style_profiles CASCADE;
TRUNCATE TABLE fashion_items CASCADE;
TRUNCATE TABLE brands CASCADE;
TRUNCATE TABLE loyalty_programs CASCADE;
TRUNCATE TABLE loyalty_tiers CASCADE;
TRUNCATE TABLE brand_customer_claims CASCADE;
