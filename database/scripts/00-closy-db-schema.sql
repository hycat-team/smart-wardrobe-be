-- Table Definition
CREATE TABLE "aiexecutor"."ai_usage_period_ledgers" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "grant_id" uuid,
    "user_id" uuid NOT NULL,
    "period_index" int4 NOT NULL,
    "period_start" timestamptz NOT NULL,
    "period_end" timestamptz NOT NULL,
    "paid_input_tokens" int8 NOT NULL DEFAULT 0,
    "paid_output_tokens" int8 NOT NULL DEFAULT 0,
    "free_input_tokens" int8 NOT NULL DEFAULT 0,
    "free_output_tokens" int8 NOT NULL DEFAULT 0,
    "actual_cost_micro_vnd" int8 NOT NULL DEFAULT 0,
    "reserved_cost_micro_vnd" int8 NOT NULL DEFAULT 0,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "aiexecutor"."ai_policy_grants" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "policy_id" uuid NOT NULL,
    "plan_id" uuid NOT NULL,
    "plan_code" varchar(100) NOT NULL,
    "tier_rank" int4 NOT NULL,
    "policy_snapshot" jsonb NOT NULL,
    "effective_from" timestamptz NOT NULL,
    "effective_to" timestamptz,
    "status" varchar(20) NOT NULL,
    "source_event_id" uuid,
    "source_deposit_transaction_id" uuid,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "aiexecutor"."ai_usage_events" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "request_id" uuid NOT NULL,
    "ledger_id" uuid,
    "user_id" uuid,
    "operation" varchar(30) NOT NULL,
    "logical_route" varchar(50) NOT NULL,
    "provider" varchar(50),
    "model" varchar(150),
    "input_usd_per_million" numeric(18,8),
    "output_usd_per_million" numeric(18,8),
    "usd_to_vnd" numeric(18,4),
    "prompt_tokens" int8 NOT NULL DEFAULT 0,
    "output_tokens" int8 NOT NULL DEFAULT 0,
    "thinking_tokens" int8 NOT NULL DEFAULT 0,
    "reserved_cost_micro_vnd" int8 NOT NULL DEFAULT 0,
    "actual_cost_micro_vnd" int8 NOT NULL DEFAULT 0,
    "estimated_max_cost_micro_vnd" int8 NOT NULL DEFAULT 0,
    "status" varchar(30) NOT NULL,
    "finish_reason" varchar(100),
    "error_code" varchar(100),
    "sent_at" timestamptz,
    "completed_at" timestamptz,
    "unknown_expires_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "estimated_prompt_tokens" int8,
    "token_estimation_method" varchar(30),
    "token_count_latency_ms" int8,
    "pricing_config_id" uuid,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "aiexecutor"."ai_model_configs" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "provider" varchar(50) NOT NULL,
    "model" varchar(100) NOT NULL,
    "input_usd_per_million" numeric(20,8) NOT NULL,
    "output_usd_per_million" numeric(20,8) NOT NULL,
    "currency" varchar(10) NOT NULL DEFAULT 'VND'::character varying,
    "usd_to_vnd" numeric(20,4) NOT NULL,
    "version" varchar(50) NOT NULL DEFAULT 'v1'::character varying,
    "effective_from" timestamptz NOT NULL DEFAULT now(),
    "effective_to" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "created_by" uuid,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "aiexecutor"."ai_cost_policy_audit_logs" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "actor_user_id" uuid NOT NULL,
    "changed_at" timestamptz NOT NULL DEFAULT now(),
    "request_ip" varchar(64),
    "policy_code" varchar(100) NOT NULL,
    "operation" varchar(30) NOT NULL,
    "field_name" varchar(60) NOT NULL,
    "old_value" varchar(255),
    "new_value" varchar(255),
    "action" varchar(60) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "aiexecutor"."ai_cost_policies" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "code" varchar(100) NOT NULL,
    "version" int8 NOT NULL,
    "name" varchar(150) NOT NULL,
    "enforcement_mode" varchar(30) NOT NULL,
    "period_days" int4 NOT NULL,
    "hard_cost_micro_vnd" int8,
    "compact_threshold_bps" int4 NOT NULL DEFAULT 8000,
    "free_route_threshold_bps" int4 NOT NULL DEFAULT 9200,
    "unknown_hold_minutes" int4 NOT NULL DEFAULT 1440,
    "max_unknown_paid_requests_per_day" int4 NOT NULL DEFAULT 2,
    "is_active" bool NOT NULL DEFAULT true,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "aiexecutor"."ai_cost_policy_operations" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "policy_id" uuid NOT NULL,
    "operation" varchar(30) NOT NULL,
    "normal_route" varchar(50) NOT NULL,
    "reduced_route" varchar(50) NOT NULL,
    "free_route" varchar(50) NOT NULL,
    "normal_max_input_tokens" int4 NOT NULL,
    "normal_max_output_tokens" int4 NOT NULL,
    "reduced_max_input_tokens" int4 NOT NULL,
    "reduced_max_output_tokens" int4 NOT NULL,
    "max_paid_attempts_per_day" int4,
    "paid_fallback_enabled" bool NOT NULL DEFAULT false,
    "is_enabled" bool NOT NULL DEFAULT true,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."loyalty_programs" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    "amount_per_point" numeric(12,2) NOT NULL,
    "point_expiry_days" int4,
    "rounding_mode" varchar(50) NOT NULL DEFAULT 'FLOOR'::character varying,
    "is_active" bool NOT NULL DEFAULT true,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."loyalty_tiers" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    "rank" int4 NOT NULL,
    "min_total_spend" numeric(12,2) NOT NULL DEFAULT 0,
    "description" text,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."loyalty_accounts" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "brand_customer_id" uuid NOT NULL,
    "user_id" uuid,
    "current_points" int4 NOT NULL DEFAULT 0,
    "lifetime_points" int4 NOT NULL DEFAULT 0,
    "total_spend" numeric(12,2) NOT NULL DEFAULT 0,
    "current_tier_id" uuid,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."loyalty_point_transactions" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "loyalty_account_id" uuid NOT NULL,
    "brand_id" uuid NOT NULL,
    "brand_customer_id" uuid NOT NULL,
    "user_id" uuid,
    "points_delta" int4 NOT NULL,
    "balance_after" int4 NOT NULL,
    "transaction_type" varchar(50) NOT NULL,
    "reason" varchar(255),
    "spend_amount" numeric(12,2),
    "reference_type" varchar(100),
    "reference_id" uuid,
    "expires_at" timestamptz,
    "idempotency_key" varchar(100),
    "created_by_user_id" uuid,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."brand_benefits" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "benefit_type" varchar(50) NOT NULL,
    "unlock_type" varchar(50) NOT NULL,
    "required_points" int4,
    "required_tier_id" uuid,
    "feature_code" varchar(100),
    "feature_config" jsonb,
    "status" varchar(50) NOT NULL DEFAULT 'ACTIVE'::character varying,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."digital_sample_responses" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_item_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "outfit_id" uuid,
    "vote_type" varchar(50),
    "rating" int4,
    "feedback_text" text,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."brand_customers" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "user_id" uuid,
    "customer_name" varchar(255),
    "phone_e164" varchar(50),
    "phone_hash" varchar(255),
    "external_customer_code" varchar(100),
    "joined_source" varchar(50) NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'ACTIVE'::character varying,
    "joined_at" timestamptz NOT NULL DEFAULT now(),
    "claimed_at" timestamptz,
    "created_by_member_id" uuid,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."brand_conversations" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'OPEN'::character varying,
    "last_message_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "user_last_read_at" timestamptz,
    "staff_last_read_at" timestamptz,
    "closed_at" timestamptz,
    "closed_by_user_id" uuid,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."brand_conversation_messages" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "conversation_id" uuid NOT NULL,
    "sender_user_id" uuid,
    "sender_role" varchar(50) NOT NULL,
    "message" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."brand_items" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "fashion_item_id" uuid,
    "product_code" varchar(100),
    "name" varchar(255) NOT NULL,
    "description" text,
    "price" numeric(12,2),
    "item_type" varchar(50) NOT NULL,
    "status" varchar(50) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "task_id" uuid,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."brands" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "slug" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "logo_url" varchar(500),
    "logo_public_id" varchar(255),
    "status" varchar(50) NOT NULL DEFAULT 'PENDING_REVIEW'::character varying,
    "created_by_user_id" uuid NOT NULL,
    "approved_by_user_id" uuid,
    "approved_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "background_url" varchar(500),
    "background_public_id" varchar(255),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."brand_members" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "role" varchar(50) NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'ACTIVE'::character varying,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."brand_customer_claims" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_customer_id" uuid NOT NULL,
    "claim_token_hash" varchar(255) NOT NULL,
    "expires_at" timestamptz NOT NULL,
    "consumed_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "revoked_at" timestamptz,
    "revoked_by_user_id" uuid,
    "revoked_reason" varchar(255),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."loyalty_point_lots" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "loyalty_account_id" uuid NOT NULL,
    "brand_id" uuid NOT NULL,
    "brand_customer_id" uuid NOT NULL,
    "user_id" uuid,
    "earn_transaction_id" uuid NOT NULL,
    "earned_points" int4 NOT NULL,
    "remaining_points" int4 NOT NULL,
    "expires_at" timestamptz,
    "status" varchar(50) NOT NULL DEFAULT 'ACTIVE'::character varying,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "brand"."benefit_redemptions" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "benefit_id" uuid NOT NULL,
    "brand_id" uuid NOT NULL,
    "brand_customer_id" uuid NOT NULL,
    "user_id" uuid,
    "points_spent" int4 NOT NULL DEFAULT 0,
    "status" varchar(50) NOT NULL DEFAULT 'REDEEMED'::character varying,
    "redeemed_at" timestamptz NOT NULL DEFAULT now(),
    "used_at" timestamptz,
    "expires_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "community"."posts" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "public_id" varchar(32) NOT NULL,
    "user_id" uuid NOT NULL,
    "post_type" varchar(20) NOT NULL,
    "title" varchar(150),
    "content" text NOT NULL DEFAULT ''::text,
    "outfit_id" uuid,
    "like_count" int4 NOT NULL DEFAULT 0,
    "comment_count" int4 NOT NULL DEFAULT 0,
    "status" varchar(20) NOT NULL DEFAULT 'published'::character varying,
    "moderated_by" uuid,
    "hidden_reason" varchar(30),
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "community"."post_media" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "post_id" uuid NOT NULL,
    "media_type" varchar(10) NOT NULL,
    "media_url" varchar(500) NOT NULL,
    "public_id" varchar(255),
    "sort_order" int2 NOT NULL DEFAULT 0,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "community"."post_likes" (
    "post_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("post_id","user_id")
);

-- Table Definition
CREATE TABLE "community"."post_comments" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "post_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "parent_comment_id" uuid,
    "root_comment_id" uuid,
    "content" text NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'published'::character varying,
    "moderated_by" uuid,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "community"."follows" (
    "follower_id" uuid NOT NULL,
    "followee_id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("follower_id","followee_id")
);

-- Table Definition
CREATE TABLE "community"."user_follow_stats" (
    "user_id" uuid NOT NULL,
    "follower_count" int4 NOT NULL DEFAULT 0,
    "following_count" int4 NOT NULL DEFAULT 0,
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("user_id")
);

-- Table Definition
CREATE TABLE "dashboard"."dashboard_brand_daily_snapshots" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "snapshot_date" date NOT NULL,
    "total_members" int4 NOT NULL DEFAULT 0,
    "points_issued" int4 NOT NULL DEFAULT 0,
    "points_redeemed" int4 NOT NULL DEFAULT 0,
    "points_expiring_30d" int4 NOT NULL DEFAULT 0,
    "points_liability_value_vnd" numeric(12,2) NOT NULL DEFAULT 0.00,
    "active_vouchers" int4 NOT NULL DEFAULT 0,
    "vouchers_redeemed" int4 NOT NULL DEFAULT 0,
    "digital_sample_votes_count" int4 NOT NULL DEFAULT 0,
    "digital_sample_avg_rating" numeric(3,2) NOT NULL DEFAULT 0.00,
    "open_cs_tickets" int4 NOT NULL DEFAULT 0,
    "cs_first_response_avg_seconds" int4 NOT NULL DEFAULT 0,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "dashboard"."dashboard_user_wardrobe_summaries" (
    "user_id" uuid NOT NULL,
    "total_items" int4 NOT NULL DEFAULT 0,
    "total_wardrobe_value_vnd" numeric(12,2) NOT NULL DEFAULT 0.00,
    "average_cost_per_wear_vnd" numeric(12,2) NOT NULL DEFAULT 0.00,
    "underutilized_items_count" int4 NOT NULL DEFAULT 0,
    "underutilized_items" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "ai_quota_remaining" int4 NOT NULL DEFAULT 0,
    "expiring_vouchers_count" int4 NOT NULL DEFAULT 0,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("user_id")
);

-- Table Definition
CREATE TABLE "dashboard"."dashboard_admin_daily_snapshots" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "snapshot_date" date NOT NULL,
    "total_users" int4 NOT NULL DEFAULT 0,
    "new_users" int4 NOT NULL DEFAULT 0,
    "active_users_dau" int4 NOT NULL DEFAULT 0,
    "total_brands" int4 NOT NULL DEFAULT 0,
    "pending_review_brands" int4 NOT NULL DEFAULT 0,
    "ai_requests_count" int8 NOT NULL DEFAULT 0,
    "ai_error_count" int8 NOT NULL DEFAULT 0,
    "ai_paid_tokens" int8 NOT NULL DEFAULT 0,
    "ai_actual_cost_micro_vnd" int8 NOT NULL DEFAULT 0,
    "subscription_revenue_vnd" numeric(12,2) NOT NULL DEFAULT 0.00,
    "net_ai_margin_vnd" numeric(12,2) NOT NULL DEFAULT 0.00,
    "payos_transaction_count" int4 NOT NULL DEFAULT 0,
    "payos_success_count" int4 NOT NULL DEFAULT 0,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "payos_pending_count" int4 NOT NULL DEFAULT 0,
    "payos_failed_count" int4 NOT NULL DEFAULT 0,
    "payos_voided_count" int4 NOT NULL DEFAULT 0,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "fashion"."conversational_contexts" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "title" varchar(255) NOT NULL,
    "context_summary" text DEFAULT ''::text,
    "is_archived" bool NOT NULL DEFAULT false,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "fashion"."messages" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "context_id" uuid NOT NULL,
    "sender" varchar(50) NOT NULL,
    "content" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "is_summarized" bool NOT NULL DEFAULT false,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "fashion"."fashion_items" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "category_id" uuid,
    "image_url" varchar(500) NOT NULL,
    "image_public_id" varchar(255) NOT NULL,
    "color" varchar(50),
    "color_hex" varchar(7),
    "color_hue" float8,
    "color_saturation" float8,
    "color_lightness" float8,
    "style" varchar(100),
    "material" varchar(100),
    "pattern" varchar(100),
    "fit" varchar(50),
    "seasonality" varchar(100),
    "description" text,
    "embedding" vector,
    "processing_retry_count" int4 NOT NULL DEFAULT 0,
    "processing_version" int4 NOT NULL DEFAULT 0,
    "processing_started_at" timestamptz,
    "last_processing_attempt_at" timestamptz,
    "processing_error_reason" text,
    "review_reason" varchar(100),
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "fashion"."categories" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "name" varchar(100) NOT NULL,
    "slug" varchar(100) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "sort_order" int4 NOT NULL DEFAULT 0,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "identity"."users" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "username" varchar(255) NOT NULL,
    "email" varchar(255) NOT NULL,
    "password_hash" varchar(255),
    "first_name" varchar(255),
    "last_name" varchar(255),
    "date_of_birth" date,
    "address" varchar(255),
    "gender" int4,
    "role_slug" varchar(50) NOT NULL,
    "body_profile" jsonb,
    "status" int2 NOT NULL DEFAULT 0,
    "is_deleted" bool NOT NULL DEFAULT false,
    "avatar_url" varchar(500),
    "avatar_public_id" varchar(255),
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "provider" varchar(50) NOT NULL DEFAULT 'local'::character varying,
    "provider_id" varchar(255),
    "email_verified" bool NOT NULL DEFAULT false,
    "created_by_admin" bool NOT NULL DEFAULT false,
    PRIMARY KEY ("id")
);

-- Column Comment
COMMENT ON COLUMN "identity"."users"."created_by_admin" IS 'True only when the account was created through the administrator endpoint. Set at exactly one write site; never written from configuration.';

-- Table Definition
CREATE TABLE "identity"."user_activity_events" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "activity_type" varchar(20) NOT NULL,
    "occurred_at" timestamptz NOT NULL DEFAULT now(),
    "created_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "identity"."refresh_tokens" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "token" varchar(500) NOT NULL,
    "expires_at" timestamptz NOT NULL,
    "is_revoked" bool NOT NULL DEFAULT false,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "device_name" varchar(255),
    "device_ip" varchar(45),
    "user_agent" varchar(512),
    "last_active_at" timestamptz,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "public"."goose_db_version_commerce" (
    "id" int4 NOT NULL,
    "version_id" int8 NOT NULL,
    "is_applied" bool NOT NULL,
    "tstamp" timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "public"."goose_db_version" (
    "id" int4 NOT NULL,
    "version_id" int8 NOT NULL,
    "is_applied" bool NOT NULL,
    "tstamp" timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "shared"."admin_audit_logs" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "actor_user_id" uuid NOT NULL,
    "action" varchar(100) NOT NULL,
    "target_type" varchar(50) NOT NULL,
    "target_id" uuid,
    "payload" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "request_ip" varchar(45),
    "created_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "shared"."outbox_records" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "topic" varchar(255) NOT NULL,
    "payload" jsonb NOT NULL,
    "headers" jsonb DEFAULT '{}'::jsonb,
    "status" varchar(50) NOT NULL DEFAULT 'pending'::character varying,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "published_at" timestamptz,
    "attempts" int4 NOT NULL DEFAULT 0,
    "last_error" text,
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "subscription"."user_daily_quotas" (
    "user_id" uuid NOT NULL,
    "outfit_recommend_count" int4 NOT NULL DEFAULT 0,
    "ai_usage_count" int4 NOT NULL DEFAULT 0,
    "last_reset_date" date NOT NULL DEFAULT CURRENT_DATE,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("user_id")
);

-- Table Definition
CREATE TABLE "subscription"."wallet_statements" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "amount" numeric(12,2) NOT NULL,
    "transaction_type" varchar(50) NOT NULL,
    "previous_balance" numeric(12,2) NOT NULL,
    "new_balance" numeric(12,2) NOT NULL,
    "reference_id" uuid,
    "description" varchar(255) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "source_plan_code" varchar(100),
    "source_tier_rank" int4,
    "active_tier_rank_at_completion" int4,
    "renewal_attempt_key" varchar(255),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "subscription"."user_subscriptions" (
    "user_id" uuid NOT NULL,
    "subscription_plan_id" uuid NOT NULL,
    "expires_at" timestamptz,
    "is_auto_renew_enabled" bool NOT NULL DEFAULT false,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "current_plan_code" varchar(100) NOT NULL,
    "current_tier_rank" int4 NOT NULL,
    "current_plan_kind" int2 NOT NULL,
    "current_benefit_snapshot" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "started_at" timestamptz NOT NULL,
    "fallback_plan_id" uuid,
    "fallback_plan_code" varchar(100),
    "fallback_tier_rank" int4,
    "fallback_plan_kind" int2,
    "fallback_benefit_snapshot" jsonb,
    "last_deposit_transaction_id" uuid,
    "version" int8 NOT NULL DEFAULT 0,
    PRIMARY KEY ("user_id")
);

-- Table Definition
CREATE TABLE "subscription"."user_wallets" (
    "user_id" uuid NOT NULL,
    "balance" numeric(12,2) NOT NULL DEFAULT 0.00,
    "currency" varchar(10) NOT NULL DEFAULT 'VND'::character varying,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("user_id")
);

-- Table Definition
CREATE TABLE "subscription"."deposit_transactions" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "amount" numeric(12,2) NOT NULL,
    "currency" varchar(10) NOT NULL DEFAULT 'VND'::character varying,
    "status" int2 NOT NULL DEFAULT 0,
    "transaction_type" varchar(50) NOT NULL,
    "subscription_plan_id" uuid,
    "order_code" int8 NOT NULL,
    "gateway_reference" varchar(255),
    "gateway_details" text,
    "payment_url" varchar(500),
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "provider" varchar(50) NOT NULL DEFAULT 'PAYOS'::character varying,
    "payment_link_id" varchar(255),
    "provider_status" varchar(50),
    "successful_provider_reference" varchar(255),
    "expires_at" timestamptz,
    "next_reconciliation_at" timestamptz,
    "reconciliation_attempts" int4 NOT NULL DEFAULT 0,
    "processing_token" uuid,
    "processing_lease_until" timestamptz,
    "last_provider_error_code" varchar(100),
    "last_provider_error_at" timestamptz,
    "failure_reason" varchar(100),
    "cancelled_at" timestamptz,
    "expired_at" timestamptz,
    "plan_code" varchar(100),
    "plan_name" varchar(100),
    "tier_rank" int4,
    "plan_kind" int2,
    "purchased_duration_days" int4,
    "expected_amount" numeric(12,2) NOT NULL,
    "benefit_snapshot" jsonb,
    "pricing_version" int8,
    "benefit_resolution" varchar(100),
    "benefit_applied_at" timestamptz,
    "benefit_result_snapshot" jsonb,
    "wallet_credit_amount" numeric(12,2),
    "subscription_version_before" int8,
    "subscription_version_after" int8,
    "refunded_at" timestamptz,
    "refund_amount" numeric(12,2),
    "refund_reason" varchar(255),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "subscription"."provider_payment_events" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "provider" varchar(50) NOT NULL,
    "provider_reference" varchar(255) NOT NULL,
    "event_code" varchar(100) NOT NULL,
    "order_code" int8 NOT NULL,
    "payment_link_id" varchar(255),
    "amount" numeric(12,2) NOT NULL,
    "currency" varchar(10) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "subscription"."provider_webhook_inbox" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "provider" varchar(50) NOT NULL,
    "provider_reference" varchar(255),
    "event_code" varchar(100) NOT NULL,
    "order_code" int8 NOT NULL,
    "payment_link_id" varchar(255),
    "amount" numeric(12,2) NOT NULL,
    "currency" varchar(10) NOT NULL,
    "canonical_payload_hash" varchar(64) NOT NULL,
    "raw_payload" jsonb NOT NULL,
    "processing_status" varchar(50) NOT NULL,
    "processing_attempts" int4 NOT NULL DEFAULT 0,
    "next_processing_at" timestamptz,
    "processing_token" uuid,
    "processing_lease_until" timestamptz,
    "processing_error" text,
    "received_at" timestamptz NOT NULL DEFAULT now(),
    "processed_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "subscription"."subscription_renewal_attempts" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "renewal_attempt_key" varchar(255) NOT NULL,
    "user_id" uuid NOT NULL,
    "expected_plan_id" uuid NOT NULL,
    "expected_expires_at" timestamptz NOT NULL,
    "expected_subscription_version" int8 NOT NULL,
    "status" varchar(50) NOT NULL,
    "attempt_count" int4 NOT NULL DEFAULT 0,
    "last_error_code" varchar(100),
    "last_error_message" text,
    "processing_token" uuid,
    "processing_lease_until" timestamptz,
    "completed_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "subscription"."campaign_sweep_runs" (
    "campaign_code" varchar(64) NOT NULL,
    "last_sweep_at" timestamptz,
    "degraded" bool NOT NULL DEFAULT false,
    "degraded_reason" text,
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("campaign_code")
);

-- Table Definition
CREATE TABLE "subscription"."campaign_claims" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "campaign_code" varchar(64) NOT NULL,
    "plan_slug" varchar(100) NOT NULL,
    "plan_id" uuid NOT NULL,
    "registered_at" timestamptz NOT NULL,
    "expires_at" timestamptz NOT NULL,
    "granted_at" timestamptz NOT NULL DEFAULT now(),
    "grant_bucket" varchar(16) NOT NULL,
    "fills_main_quota" bool NOT NULL DEFAULT false,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "subscription"."campaigns" (
    "code" varchar(64) NOT NULL,
    "plan_slug" varchar(100) NOT NULL,
    "quota" int4 NOT NULL,
    "reserve" int4 NOT NULL DEFAULT 0,
    "starts_at" timestamptz NOT NULL,
    "ends_at" timestamptz,
    "closed_at" timestamptz,
    "version" int4 NOT NULL DEFAULT 1,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("code")
);

-- Table Definition
CREATE TABLE "subscription"."user_subscription_events" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "event_key" varchar(255) NOT NULL,
    "user_id" uuid NOT NULL,
    "event_type" varchar(100) NOT NULL,
    "from_plan_code" varchar(100),
    "from_tier_rank" int4,
    "to_plan_code" varchar(100),
    "to_tier_rank" int4,
    "source_deposit_transaction_id" uuid,
    "actor_admin_id" uuid,
    "occurred_at" timestamptz NOT NULL,
    "effective_at" timestamptz NOT NULL,
    "metadata" jsonb,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "subscription"."subscription_plans" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "slug" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "price" numeric(12,2) NOT NULL DEFAULT 0.00,
    "max_wardrobe_items" int4 NOT NULL,
    "max_outfits" int4 NOT NULL,
    "ai_outfit_daily_quota" int4 NOT NULL,
    "ai_chat_daily_quota" int4 NOT NULL,
    "duration_days" int4,
    "is_active" bool NOT NULL DEFAULT true,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "plan_kind" int2 NOT NULL,
    "tier_rank" int4 NOT NULL,
    "pricing_version" int8 NOT NULL DEFAULT 1,
    "ai_cost_policy_id" uuid NOT NULL,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "wardrobe"."outfits" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "cover_image_url" varchar(500),
    "cover_public_id" varchar(255),
    "status" int2 NOT NULL DEFAULT 1,
    "is_deleted" bool NOT NULL DEFAULT false,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "wardrobe"."wardrobe_items" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "purchase_price" numeric(12,2),
    "status" int2 NOT NULL DEFAULT 0,
    "item_type" int2 NOT NULL DEFAULT 0,
    "last_used_at" timestamptz,
    "is_deleted" bool NOT NULL DEFAULT false,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "fashion_item_id" uuid,
    "task_id" uuid,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "wardrobe"."outfit_items" (
    "outfit_id" uuid NOT NULL,
    "position_x" float8 NOT NULL DEFAULT 0.0,
    "position_y" float8 NOT NULL DEFAULT 0.0,
    "scale" float8 NOT NULL DEFAULT 1.0,
    "layer_order" int2 NOT NULL DEFAULT 1,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "fashion_item_id" uuid NOT NULL,
    "item_context" varchar(50) NOT NULL,
    PRIMARY KEY ("outfit_id","fashion_item_id","item_context")
);



-- Indices
CREATE UNIQUE INDEX ai_usage_period_ledgers_grant_id_period_index_key ON aiexecutor.ai_usage_period_ledgers USING btree (grant_id, period_index);
CREATE UNIQUE INDEX ux_ledgers_user_period_fallback ON aiexecutor.ai_usage_period_ledgers USING btree (user_id, period_index) WHERE (grant_id IS NULL);

ALTER TABLE "aiexecutor"."ai_usage_period_ledgers" ADD FOREIGN KEY ("grant_id") REFERENCES "aiexecutor"."ai_policy_grants"("id") ON DELETE CASCADE;

ALTER TABLE "aiexecutor"."ai_usage_period_ledgers" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX user_ai_policy_grants_pkey ON aiexecutor.ai_policy_grants USING btree (id);
CREATE INDEX ix_user_ai_policy_grants_active ON aiexecutor.ai_policy_grants USING btree (user_id, effective_from, effective_to);
CREATE INDEX ix_ai_policy_grants_source_event_id ON aiexecutor.ai_policy_grants USING btree (source_event_id);

ALTER TABLE "aiexecutor"."ai_policy_grants" ADD FOREIGN KEY ("policy_id") REFERENCES "aiexecutor"."ai_cost_policies"("id") ON DELETE RESTRICT;

ALTER TABLE "aiexecutor"."ai_policy_grants" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX ai_usage_events_request_id_key ON aiexecutor.ai_usage_events USING btree (request_id);
CREATE INDEX ix_ai_usage_events_user_day ON aiexecutor.ai_usage_events USING btree (user_id, created_at, status);
CREATE INDEX ix_ai_usage_events_unknown ON aiexecutor.ai_usage_events USING btree (status, unknown_expires_at) WHERE ((status)::text = ANY ((ARRAY['reserved'::character varying, 'unknown_usage'::character varying])::text[]));

ALTER TABLE "aiexecutor"."ai_usage_events" ADD FOREIGN KEY ("ledger_id") REFERENCES "aiexecutor"."ai_usage_period_ledgers"("id") ON DELETE CASCADE;

ALTER TABLE "aiexecutor"."ai_usage_events" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

ALTER TABLE "aiexecutor"."ai_usage_events" ADD FOREIGN KEY ("pricing_config_id") REFERENCES "aiexecutor"."ai_model_configs"("id");

-- Indices
CREATE UNIQUE INDEX uq_provider_model_active ON aiexecutor.ai_model_configs USING btree (provider, model, effective_from);

ALTER TABLE "aiexecutor"."ai_model_configs" ADD FOREIGN KEY ("created_by") REFERENCES "identity"."users"("id");

-- Indices
CREATE INDEX ix_ai_cost_policy_audit_logs_actor_user_id ON aiexecutor.ai_cost_policy_audit_logs USING btree (actor_user_id);
CREATE INDEX ix_ai_cost_policy_audit_logs_changed_at ON aiexecutor.ai_cost_policy_audit_logs USING btree (changed_at);

-- Indices
CREATE UNIQUE INDEX ai_cost_policies_code_version_key ON aiexecutor.ai_cost_policies USING btree (code, version);

-- Indices
CREATE UNIQUE INDEX ai_cost_policy_operations_policy_id_operation_key ON aiexecutor.ai_cost_policy_operations USING btree (policy_id, operation);

ALTER TABLE "aiexecutor"."ai_cost_policy_operations" ADD FOREIGN KEY ("policy_id") REFERENCES "aiexecutor"."ai_cost_policies"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX uq_loyalty_programs_active_brand ON brand.loyalty_programs USING btree (brand_id) WHERE (is_active = true);

ALTER TABLE "brand"."loyalty_programs" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX uq_loyalty_tiers_brand_rank ON brand.loyalty_tiers USING btree (brand_id, rank);
CREATE UNIQUE INDEX uq_loyalty_tiers_brand_name ON brand.loyalty_tiers USING btree (brand_id, name);

ALTER TABLE "brand"."loyalty_tiers" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX uq_loyalty_accounts_brand_customer ON brand.loyalty_accounts USING btree (brand_customer_id);
CREATE UNIQUE INDEX uq_loyalty_accounts_brand_user ON brand.loyalty_accounts USING btree (brand_id, user_id) WHERE (user_id IS NOT NULL);
CREATE INDEX idx_loyalty_accounts_brand_tier ON brand.loyalty_accounts USING btree (brand_id, current_tier_id);

ALTER TABLE "brand"."loyalty_accounts" ADD FOREIGN KEY ("brand_customer_id") REFERENCES "brand"."brand_customers"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."loyalty_accounts" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."loyalty_accounts" ADD FOREIGN KEY ("current_tier_id") REFERENCES "brand"."loyalty_tiers"("id") ON DELETE SET NULL;

ALTER TABLE "brand"."loyalty_accounts" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

-- Indices
CREATE UNIQUE INDEX uq_loyalty_point_transactions_brand_idempotency ON brand.loyalty_point_transactions USING btree (brand_id, idempotency_key) WHERE (idempotency_key IS NOT NULL);
CREATE INDEX idx_loyalty_point_transactions_account_created ON brand.loyalty_point_transactions USING btree (loyalty_account_id, created_at);
CREATE INDEX idx_loyalty_point_transactions_customer_created ON brand.loyalty_point_transactions USING btree (brand_customer_id, created_at);
CREATE INDEX idx_loyalty_point_transactions_brand_user ON brand.loyalty_point_transactions USING btree (brand_id, user_id);
CREATE INDEX idx_loyalty_point_transactions_expires_at ON brand.loyalty_point_transactions USING btree (expires_at) WHERE (expires_at IS NOT NULL);

ALTER TABLE "brand"."loyalty_point_transactions" ADD FOREIGN KEY ("brand_customer_id") REFERENCES "brand"."brand_customers"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."loyalty_point_transactions" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."loyalty_point_transactions" ADD FOREIGN KEY ("created_by_user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

ALTER TABLE "brand"."loyalty_point_transactions" ADD FOREIGN KEY ("loyalty_account_id") REFERENCES "brand"."loyalty_accounts"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."loyalty_point_transactions" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

-- Indices
CREATE INDEX idx_brand_benefits_brand_status ON brand.brand_benefits USING btree (brand_id, status);

ALTER TABLE "brand"."brand_benefits" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."brand_benefits" ADD FOREIGN KEY ("required_tier_id") REFERENCES "brand"."loyalty_tiers"("id") ON DELETE SET NULL;

-- Indices
CREATE INDEX idx_digital_sample_responses_item_user ON brand.digital_sample_responses USING btree (brand_item_id, user_id);

ALTER TABLE "brand"."digital_sample_responses" ADD FOREIGN KEY ("brand_item_id") REFERENCES "brand"."brand_items"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."digital_sample_responses" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX uq_brand_customers_brand_user ON brand.brand_customers USING btree (brand_id, user_id) WHERE (user_id IS NOT NULL);
CREATE UNIQUE INDEX uq_brand_customers_brand_phone_hash ON brand.brand_customers USING btree (brand_id, phone_hash) WHERE (phone_hash IS NOT NULL);
CREATE INDEX idx_brand_customers_brand_status ON brand.brand_customers USING btree (brand_id, status);

ALTER TABLE "brand"."brand_customers" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."brand_customers" ADD FOREIGN KEY ("created_by_member_id") REFERENCES "brand"."brand_members"("id") ON DELETE SET NULL;

ALTER TABLE "brand"."brand_customers" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

-- Indices
CREATE UNIQUE INDEX uq_brand_conversations_brand_user ON brand.brand_conversations USING btree (brand_id, user_id);
CREATE INDEX idx_brand_conversations_brand_status ON brand.brand_conversations USING btree (brand_id, status);

ALTER TABLE "brand"."brand_conversations" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."brand_conversations" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."brand_conversations" ADD FOREIGN KEY ("closed_by_user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

-- Indices
CREATE INDEX idx_brand_conversation_messages_conv_created ON brand.brand_conversation_messages USING btree (conversation_id, created_at);
CREATE INDEX idx_brand_conversation_messages_unread ON brand.brand_conversation_messages USING btree (conversation_id, sender_role, created_at);

ALTER TABLE "brand"."brand_conversation_messages" ADD FOREIGN KEY ("conversation_id") REFERENCES "brand"."brand_conversations"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."brand_conversation_messages" ADD FOREIGN KEY ("sender_user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

-- Indices
CREATE UNIQUE INDEX uq_brand_items_brand_product_code ON brand.brand_items USING btree (brand_id, product_code);
CREATE INDEX idx_brand_items_brand_status ON brand.brand_items USING btree (brand_id, status);
CREATE UNIQUE INDEX uq_brand_items_fashion_item ON brand.brand_items USING btree (fashion_item_id) WHERE (fashion_item_id IS NOT NULL);
CREATE INDEX idx_brand_items_task_id ON brand.brand_items USING btree (task_id);

ALTER TABLE "brand"."brand_items" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX brands_slug_key ON brand.brands USING btree (slug);

ALTER TABLE "brand"."brands" ADD FOREIGN KEY ("approved_by_user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

ALTER TABLE "brand"."brands" ADD FOREIGN KEY ("created_by_user_id") REFERENCES "identity"."users"("id") ON DELETE RESTRICT;

-- Indices
CREATE UNIQUE INDEX uq_brand_members_brand_user ON brand.brand_members USING btree (brand_id, user_id);
CREATE INDEX idx_brand_members_user_status ON brand.brand_members USING btree (user_id, status);
CREATE UNIQUE INDEX ux_brand_members_one_active_owner ON brand.brand_members USING btree (brand_id) WHERE (((role)::text = 'owner'::text) AND ((status)::text = 'active'::text));

ALTER TABLE "brand"."brand_members" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."brand_members" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX uq_brand_customer_claims_token_hash ON brand.brand_customer_claims USING btree (claim_token_hash);
CREATE INDEX idx_brand_customer_claims_customer ON brand.brand_customer_claims USING btree (brand_customer_id);
CREATE INDEX idx_brand_customer_claims_active_customer ON brand.brand_customer_claims USING btree (brand_customer_id) WHERE ((consumed_at IS NULL) AND (revoked_at IS NULL));

ALTER TABLE "brand"."brand_customer_claims" ADD FOREIGN KEY ("brand_customer_id") REFERENCES "brand"."brand_customers"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."brand_customer_claims" ADD FOREIGN KEY ("revoked_by_user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

-- Indices
CREATE UNIQUE INDEX ux_loyalty_point_lots_earn_transaction ON brand.loyalty_point_lots USING btree (earn_transaction_id);
CREATE INDEX idx_loyalty_point_lots_account_active_expiry ON brand.loyalty_point_lots USING btree (loyalty_account_id, expires_at) WHERE (((status)::text = 'ACTIVE'::text) AND (remaining_points > 0));
CREATE INDEX idx_loyalty_point_lots_expiry_worker ON brand.loyalty_point_lots USING btree (expires_at) WHERE (((status)::text = 'ACTIVE'::text) AND (remaining_points > 0) AND (expires_at IS NOT NULL));
CREATE INDEX idx_loyalty_point_lots_brand_customer ON brand.loyalty_point_lots USING btree (brand_customer_id);

ALTER TABLE "brand"."loyalty_point_lots" ADD FOREIGN KEY ("brand_customer_id") REFERENCES "brand"."brand_customers"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."loyalty_point_lots" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."loyalty_point_lots" ADD FOREIGN KEY ("earn_transaction_id") REFERENCES "brand"."loyalty_point_transactions"("id") ON DELETE RESTRICT;

ALTER TABLE "brand"."loyalty_point_lots" ADD FOREIGN KEY ("loyalty_account_id") REFERENCES "brand"."loyalty_accounts"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."loyalty_point_lots" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

-- Indices
CREATE INDEX idx_benefit_redemptions_user_status ON brand.benefit_redemptions USING btree (user_id, status);
CREATE INDEX idx_benefit_redemptions_customer_status ON brand.benefit_redemptions USING btree (brand_customer_id, status);

ALTER TABLE "brand"."benefit_redemptions" ADD FOREIGN KEY ("benefit_id") REFERENCES "brand"."brand_benefits"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."benefit_redemptions" ADD FOREIGN KEY ("brand_id") REFERENCES "brand"."brands"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."benefit_redemptions" ADD FOREIGN KEY ("brand_customer_id") REFERENCES "brand"."brand_customers"("id") ON DELETE CASCADE;

ALTER TABLE "brand"."benefit_redemptions" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE SET NULL;

-- Indices
CREATE UNIQUE INDEX posts_public_id_key ON community.posts USING btree (public_id);
CREATE INDEX idx_posts_published_created ON community.posts USING btree (created_at DESC, id DESC) WHERE ((status)::text = 'published'::text);
CREATE INDEX idx_posts_user_created ON community.posts USING btree (user_id, created_at DESC) WHERE ((status)::text = 'published'::text);
CREATE INDEX idx_posts_type_created ON community.posts USING btree (post_type, created_at DESC) WHERE ((status)::text = 'published'::text);
CREATE INDEX idx_posts_outfit ON community.posts USING btree (outfit_id) WHERE ((outfit_id IS NOT NULL) AND ((status)::text = 'published'::text));
CREATE INDEX idx_posts_search ON community.posts USING gin (lower("left"(content, 200)) gin_trgm_ops) WHERE ((status)::text = 'published'::text);

ALTER TABLE "community"."posts" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id");

-- Indices
CREATE INDEX idx_post_media_post_sort ON community.post_media USING btree (post_id, sort_order);

ALTER TABLE "community"."post_media" ADD FOREIGN KEY ("post_id") REFERENCES "community"."posts"("id") ON DELETE CASCADE;

-- Indices
CREATE INDEX idx_post_likes_post ON community.post_likes USING btree (post_id, created_at DESC);
CREATE INDEX idx_post_likes_user ON community.post_likes USING btree (user_id, created_at DESC);

ALTER TABLE "community"."post_likes" ADD FOREIGN KEY ("post_id") REFERENCES "community"."posts"("id") ON DELETE CASCADE;

ALTER TABLE "community"."post_likes" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id");

-- Indices
CREATE INDEX idx_comments_post_parent_created ON community.post_comments USING btree (post_id, parent_comment_id, created_at);
CREATE INDEX idx_comments_root_created ON community.post_comments USING btree (root_comment_id, created_at);

ALTER TABLE "community"."post_comments" ADD FOREIGN KEY ("parent_comment_id") REFERENCES "community"."post_comments"("id") ON DELETE CASCADE;

ALTER TABLE "community"."post_comments" ADD FOREIGN KEY ("post_id") REFERENCES "community"."posts"("id") ON DELETE CASCADE;

ALTER TABLE "community"."post_comments" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id");

-- Indices
CREATE INDEX idx_follows_follower_created ON community.follows USING btree (follower_id, created_at DESC);
CREATE INDEX idx_follows_followee_created ON community.follows USING btree (followee_id, created_at DESC);

ALTER TABLE "community"."follows" ADD FOREIGN KEY ("followee_id") REFERENCES "identity"."users"("id");

ALTER TABLE "community"."follows" ADD FOREIGN KEY ("follower_id") REFERENCES "identity"."users"("id");

ALTER TABLE "community"."user_follow_stats" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id");

-- Indices
CREATE UNIQUE INDEX uq_brand_daily_snapshot ON dashboard.dashboard_brand_daily_snapshots USING btree (brand_id, snapshot_date);
CREATE INDEX idx_brand_snapshots_brand_date ON dashboard.dashboard_brand_daily_snapshots USING btree (brand_id, snapshot_date DESC);

-- Indices
CREATE INDEX idx_user_wardrobe_summaries_updated ON dashboard.dashboard_user_wardrobe_summaries USING btree (updated_at DESC);

ALTER TABLE "dashboard"."dashboard_user_wardrobe_summaries" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX dashboard_admin_daily_snapshots_snapshot_date_key ON dashboard.dashboard_admin_daily_snapshots USING btree (snapshot_date);
CREATE INDEX idx_admin_snapshots_date ON dashboard.dashboard_admin_daily_snapshots USING btree (snapshot_date DESC);

ALTER TABLE "fashion"."conversational_contexts" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE INDEX idx_messages_context_created_at ON fashion.messages USING btree (context_id, created_at DESC);
CREATE INDEX idx_messages_context_unsummarized ON fashion.messages USING btree (context_id) WHERE (is_summarized = false);

ALTER TABLE "fashion"."messages" ADD FOREIGN KEY ("context_id") REFERENCES "fashion"."conversational_contexts"("id") ON DELETE CASCADE;

-- Indices
CREATE INDEX idx_fashion_items_category_id ON fashion.fashion_items USING btree (category_id);
CREATE INDEX fitems_embedding_cosine_idx ON fashion.fashion_items USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_fashion_items_lexical_search ON fashion.fashion_items USING gin (to_tsvector('simple'::regconfig, immutable_unaccent(lower((((((((((((((COALESCE(color, ''::character varying))::text || ' '::text) || (COALESCE(style, ''::character varying))::text) || ' '::text) || (COALESCE(material, ''::character varying))::text) || ' '::text) || (COALESCE(pattern, ''::character varying))::text) || ' '::text) || (COALESCE(fit, ''::character varying))::text) || ' '::text) || (COALESCE(seasonality, ''::character varying))::text) || ' '::text) || COALESCE(description, ''::text))))));

ALTER TABLE "fashion"."fashion_items" ADD FOREIGN KEY ("category_id") REFERENCES "fashion"."categories"("id") ON DELETE RESTRICT;

-- Indices
CREATE UNIQUE INDEX categories_name_key ON fashion.categories USING btree (name);
CREATE UNIQUE INDEX categories_slug_key ON fashion.categories USING btree (slug);

-- Indices
CREATE UNIQUE INDEX idx_users_email_lower ON identity.users USING btree (lower((email)::text));
CREATE UNIQUE INDEX idx_users_username_lower ON identity.users USING btree (lower((username)::text));
CREATE UNIQUE INDEX idx_users_provider_provider_id ON identity.users USING btree (provider, provider_id) WHERE (provider_id IS NOT NULL);

-- Indices
CREATE INDEX idx_user_activity_events_occurred_at ON identity.user_activity_events USING btree (occurred_at);
CREATE INDEX idx_user_activity_events_user_occurred_at ON identity.user_activity_events USING btree (user_id, occurred_at);

ALTER TABLE "identity"."user_activity_events" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX refresh_tokens_token_key ON identity.refresh_tokens USING btree (token);

ALTER TABLE "identity"."refresh_tokens" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE INDEX idx_admin_audit_logs_actor ON shared.admin_audit_logs USING btree (actor_user_id);
CREATE INDEX idx_admin_audit_logs_target ON shared.admin_audit_logs USING btree (target_type, target_id);
CREATE INDEX idx_admin_audit_logs_created_at ON shared.admin_audit_logs USING btree (created_at);

ALTER TABLE "shared"."admin_audit_logs" ADD FOREIGN KEY ("actor_user_id") REFERENCES "identity"."users"("id");

-- Indices
CREATE INDEX idx_outbox_records_status_published_at ON shared.outbox_records USING btree (status, published_at);
CREATE INDEX idx_outbox_records_status_created_at ON shared.outbox_records USING btree (status, created_at);

ALTER TABLE "subscription"."user_daily_quotas" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX ux_wallet_statement_renewal_attempt ON subscription.wallet_statements USING btree (renewal_attempt_key) WHERE (renewal_attempt_key IS NOT NULL);

ALTER TABLE "subscription"."wallet_statements" ADD FOREIGN KEY ("reference_id") REFERENCES "subscription"."deposit_transactions"("id") ON DELETE SET NULL;

ALTER TABLE "subscription"."wallet_statements" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE INDEX idx_user_subscriptions_plan_id ON subscription.user_subscriptions USING btree (subscription_plan_id);
CREATE INDEX idx_user_subscriptions_expires ON subscription.user_subscriptions USING btree (expires_at, user_id) WHERE (current_plan_kind = 1);

ALTER TABLE "subscription"."user_subscriptions" ADD FOREIGN KEY ("fallback_plan_id") REFERENCES "subscription"."subscription_plans"("id") ON DELETE RESTRICT;

ALTER TABLE "subscription"."user_subscriptions" ADD FOREIGN KEY ("subscription_plan_id") REFERENCES "subscription"."subscription_plans"("id") ON DELETE RESTRICT;

ALTER TABLE "subscription"."user_subscriptions" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

ALTER TABLE "subscription"."user_wallets" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX deposit_transactions_order_code_key ON subscription.deposit_transactions USING btree (order_code);
CREATE UNIQUE INDEX deposit_transactions_gateway_reference_key ON subscription.deposit_transactions USING btree (gateway_reference);
CREATE UNIQUE INDEX ux_active_direct_purchase_per_user ON subscription.deposit_transactions USING btree (user_id) WHERE (((transaction_type)::text = 'DIRECT_PURCHASE'::text) AND (status = ANY (ARRAY[3, 0, 4, 5])));
CREATE INDEX idx_deposit_reconciliation ON subscription.deposit_transactions USING btree (next_reconciliation_at, expires_at) WHERE (status = ANY (ARRAY[0, 3, 4, 5]));

ALTER TABLE "subscription"."deposit_transactions" ADD FOREIGN KEY ("subscription_plan_id") REFERENCES "subscription"."subscription_plans"("id") ON DELETE SET NULL;

ALTER TABLE "subscription"."deposit_transactions" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX provider_payment_events_provider_provider_reference_key ON subscription.provider_payment_events USING btree (provider, provider_reference);

-- Indices
CREATE UNIQUE INDEX ux_webhook_provider_reference ON subscription.provider_webhook_inbox USING btree (provider, provider_reference, event_code) WHERE (provider_reference IS NOT NULL);
CREATE UNIQUE INDEX ux_webhook_payload_hash ON subscription.provider_webhook_inbox USING btree (provider, canonical_payload_hash) WHERE (provider_reference IS NULL);

-- Indices
CREATE UNIQUE INDEX subscription_renewal_attempts_renewal_attempt_key_key ON subscription.subscription_renewal_attempts USING btree (renewal_attempt_key);

ALTER TABLE "subscription"."subscription_renewal_attempts" ADD FOREIGN KEY ("expected_plan_id") REFERENCES "subscription"."subscription_plans"("id");

ALTER TABLE "subscription"."subscription_renewal_attempts" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Comments
COMMENT ON TABLE "subscription"."campaign_sweep_runs" IS 'One row per campaign: the health of the FR-079 catch-up sweep, read by the operator look-up (FR-065) from every replica.';

-- Indices
CREATE UNIQUE INDEX ux_campaign_claims_user ON subscription.campaign_claims USING btree (user_id);
CREATE UNIQUE INDEX ux_campaign_claims_watermark ON subscription.campaign_claims USING btree (campaign_code) WHERE fills_main_quota;
CREATE INDEX ix_campaign_claims_campaign_bucket ON subscription.campaign_claims USING btree (campaign_code, grant_bucket);
CREATE INDEX ix_campaign_claims_campaign_granted ON subscription.campaign_claims USING btree (campaign_code, granted_at DESC);

ALTER TABLE "subscription"."campaign_claims" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE INDEX ix_campaigns_starts_at ON subscription.campaigns USING btree (starts_at DESC);

-- Indices
CREATE UNIQUE INDEX user_subscription_events_event_key_key ON subscription.user_subscription_events USING btree (event_key);

ALTER TABLE "subscription"."user_subscription_events" ADD FOREIGN KEY ("actor_admin_id") REFERENCES "identity"."users"("id");

ALTER TABLE "subscription"."user_subscription_events" ADD FOREIGN KEY ("source_deposit_transaction_id") REFERENCES "subscription"."deposit_transactions"("id");

ALTER TABLE "subscription"."user_subscription_events" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE UNIQUE INDEX subscription_plans_slug_key ON subscription.subscription_plans USING btree (slug);
CREATE UNIQUE INDEX ux_single_active_default_free_plan ON subscription.subscription_plans USING btree (plan_kind) WHERE ((plan_kind = 0) AND (is_active = true));

ALTER TABLE "wardrobe"."outfits" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE INDEX idx_wardrobe_items_last_used_at ON wardrobe.wardrobe_items USING btree (last_used_at);
CREATE INDEX idx_wardrobe_items_user_fashion_item ON wardrobe.wardrobe_items USING btree (user_id, fashion_item_id);
CREATE INDEX idx_wardrobe_items_task_id ON wardrobe.wardrobe_items USING btree (task_id);
CREATE INDEX idx_wardrobe_items_user_status ON wardrobe.wardrobe_items USING btree (user_id, is_deleted, status);
CREATE INDEX idx_wardrobe_items_fashion_item ON wardrobe.wardrobe_items USING btree (fashion_item_id);
CREATE INDEX idx_wardrobe_items_type_status ON wardrobe.wardrobe_items USING btree (item_type, is_deleted);

ALTER TABLE "wardrobe"."wardrobe_items" ADD FOREIGN KEY ("user_id") REFERENCES "identity"."users"("id") ON DELETE CASCADE;

-- Indices
CREATE INDEX idx_outfit_items_fashion_item_context ON wardrobe.outfit_items USING btree (fashion_item_id, item_context);

ALTER TABLE "wardrobe"."outfit_items" ADD FOREIGN KEY ("outfit_id") REFERENCES "wardrobe"."outfits"("id") ON DELETE CASCADE;

