-- Sequence and defined type
CREATE SEQUENCE IF NOT EXISTS deposit_transactions_order_code_seq;

-- Table Definition
CREATE TABLE "public"."users" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "username" varchar(255) NOT NULL,
    "email" varchar(255) NOT NULL,
    "password_hash" varchar(255) NOT NULL,
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
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX idx_users_email_lower ON public.users USING btree (lower((email)::text));
CREATE UNIQUE INDEX idx_users_username_lower ON public.users USING btree (lower((username)::text));

-- Table Definition
CREATE TABLE "public"."messages" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "context_id" uuid NOT NULL,
    "sender" varchar(50) NOT NULL,
    "content" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "is_summarized" bool NOT NULL DEFAULT false,
    CONSTRAINT "messages_context_id_fkey" FOREIGN KEY ("context_id") REFERENCES "public"."conversational_contexts"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE INDEX idx_messages_context_created_at ON public.messages USING btree (context_id, created_at DESC);
CREATE INDEX idx_messages_context_unsummarized ON public.messages USING btree (context_id) WHERE (is_summarized = false);

-- Table Definition
CREATE TABLE "public"."categories" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "name" varchar(100) NOT NULL,
    "slug" varchar(100) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "sort_order" int4 NOT NULL DEFAULT 0,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX categories_name_key ON public.categories USING btree (name);
CREATE UNIQUE INDEX categories_slug_key ON public.categories USING btree (slug);

-- Table Definition
CREATE TABLE "public"."user_daily_quotas" (
    "user_id" uuid NOT NULL,
    "outfit_recommend_count" int4 NOT NULL DEFAULT 0,
    "ai_usage_count" int4 NOT NULL DEFAULT 0,
    "last_reset_date" date NOT NULL DEFAULT CURRENT_DATE,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "user_daily_quotas_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("user_id")
);

-- Table Definition
CREATE TABLE "public"."user_style_profiles" (
    "user_id" uuid NOT NULL,
    "taste_embedding" vector,
    "preferred_colors" jsonb,
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "user_style_profiles_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("user_id")
);


-- Indices
CREATE INDEX ustyles_embedding_cosine_idx ON public.user_style_profiles USING hnsw (taste_embedding vector_cosine_ops);

-- Table Definition
CREATE TABLE "public"."refresh_tokens" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "token" varchar(500) NOT NULL,
    "expires_at" timestamptz NOT NULL,
    "is_revoked" bool NOT NULL DEFAULT false,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "refresh_tokens_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX refresh_tokens_token_key ON public.refresh_tokens USING btree (token);

-- Table Definition
CREATE TABLE "public"."conversational_contexts" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "title" varchar(255) NOT NULL,
    "context_summary" text DEFAULT ''::text,
    "is_archived" bool NOT NULL DEFAULT false,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "conversational_contexts_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "public"."wardrobe_items" (
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
    CONSTRAINT "fk_wardrobe_items_fashion_item" FOREIGN KEY ("fashion_item_id") REFERENCES "public"."fashion_items"("id") ON DELETE RESTRICT,
    CONSTRAINT "wardrobe_items_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE INDEX idx_wardrobe_items_last_used_at ON public.wardrobe_items USING btree (last_used_at);
CREATE INDEX idx_wardrobe_items_user_fashion_item ON public.wardrobe_items USING btree (user_id, fashion_item_id);
CREATE INDEX idx_wardrobe_items_task_id ON public.wardrobe_items USING btree (task_id);

-- Table Definition
CREATE TABLE "public"."outfits" (
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
    CONSTRAINT "outfits_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "public"."user_wallets" (
    "user_id" uuid NOT NULL,
    "balance" numeric(12,2) NOT NULL DEFAULT 0.00,
    "currency" varchar(10) NOT NULL DEFAULT 'VND'::character varying,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "user_wallets_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("user_id")
);

-- Table Definition
CREATE TABLE "public"."user_subscriptions" (
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
    "started_at" timestamptz NOT NULL DEFAULT now(),
    "fallback_plan_id" uuid,
    "fallback_plan_code" varchar(100),
    "fallback_tier_rank" int4,
    "fallback_plan_kind" int2,
    "fallback_benefit_snapshot" jsonb,
    "last_deposit_transaction_id" uuid,
    "version" int8 NOT NULL DEFAULT 0,
    CONSTRAINT "user_subscriptions_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    CONSTRAINT "user_subscriptions_subscription_plan_id_fkey" FOREIGN KEY ("subscription_plan_id") REFERENCES "public"."subscription_plans"("id") ON DELETE RESTRICT,
    CONSTRAINT "user_subscriptions_fallback_plan_id_fkey" FOREIGN KEY ("fallback_plan_id") REFERENCES "public"."subscription_plans"("id") ON DELETE RESTRICT,
    PRIMARY KEY ("user_id")
);


-- Indices
CREATE INDEX idx_user_subscriptions_plan_id ON public.user_subscriptions USING btree (subscription_plan_id);
CREATE INDEX idx_user_subscriptions_expires ON public.user_subscriptions USING btree (expires_at, user_id) WHERE (current_plan_kind = 1);

-- Table Definition
CREATE TABLE "public"."goose_db_version" (
    "id" int4 NOT NULL,
    "version_id" int8 NOT NULL,
    "is_applied" bool NOT NULL,
    "tstamp" timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY ("id")
);

-- Table Definition
CREATE TABLE "public"."deposit_transactions" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "user_id" uuid NOT NULL,
    "amount" numeric(12,2) NOT NULL,
    "currency" varchar(10) NOT NULL DEFAULT 'VND'::character varying,
    "status" int2 NOT NULL DEFAULT 0,
    "transaction_type" varchar(50) NOT NULL,
    "subscription_plan_id" uuid,
    "order_code" int8 NOT NULL DEFAULT nextval('deposit_transactions_order_code_seq'::regclass),
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
    CONSTRAINT "deposit_transactions_subscription_plan_id_fkey" FOREIGN KEY ("subscription_plan_id") REFERENCES "public"."subscription_plans"("id") ON DELETE SET NULL,
    CONSTRAINT "deposit_transactions_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX deposit_transactions_order_code_key ON public.deposit_transactions USING btree (order_code);
CREATE UNIQUE INDEX ux_active_direct_purchase_per_user ON public.deposit_transactions USING btree (user_id) WHERE (((transaction_type)::text = 'DIRECT_PURCHASE'::text) AND (status = ANY (ARRAY[0, 3, 4, 5])));
CREATE INDEX idx_deposit_reconciliation ON public.deposit_transactions USING btree (next_reconciliation_at, expires_at) WHERE (status = ANY (ARRAY[0, 3, 4, 5]));

-- Table Definition
CREATE TABLE "public"."wallet_statements" (
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
    CONSTRAINT "wallet_statements_reference_id_fkey" FOREIGN KEY ("reference_id") REFERENCES "public"."deposit_transactions"("id") ON DELETE SET NULL,
    CONSTRAINT "wallet_statements_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX ux_wallet_statement_renewal_attempt ON public.wallet_statements USING btree (renewal_attempt_key) WHERE (renewal_attempt_key IS NOT NULL);

-- Table Definition
CREATE TABLE "public"."provider_payment_events" (
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


-- Indices
CREATE UNIQUE INDEX provider_payment_events_provider_provider_reference_key ON public.provider_payment_events USING btree (provider, provider_reference);

-- Table Definition
CREATE TABLE "public"."provider_webhook_inbox" (
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


-- Indices
CREATE UNIQUE INDEX ux_webhook_provider_reference ON public.provider_webhook_inbox USING btree (provider, provider_reference, event_code) WHERE (provider_reference IS NOT NULL);
CREATE UNIQUE INDEX ux_webhook_payload_hash ON public.provider_webhook_inbox USING btree (provider, canonical_payload_hash) WHERE (provider_reference IS NULL);
CREATE INDEX idx_webhook_inbox_processing ON public.provider_webhook_inbox USING btree (next_processing_at, processing_lease_until) WHERE ((processing_status)::text = ANY ((ARRAY['RECEIVED'::character varying, 'RETRY_REQUIRED'::character varying, 'PROCESSING'::character varying])::text[]));

-- Table Definition
CREATE TABLE "public"."user_subscription_events" (
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
    CONSTRAINT "user_subscription_events_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    CONSTRAINT "user_subscription_events_source_deposit_transaction_id_fkey" FOREIGN KEY ("source_deposit_transaction_id") REFERENCES "public"."deposit_transactions"("id"),
    CONSTRAINT "user_subscription_events_actor_admin_id_fkey" FOREIGN KEY ("actor_admin_id") REFERENCES "public"."users"("id"),
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX user_subscription_events_event_key_key ON public.user_subscription_events USING btree (event_key);

-- Table Definition
CREATE TABLE "public"."subscription_renewal_attempts" (
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
    CONSTRAINT "subscription_renewal_attempts_expected_plan_id_fkey" FOREIGN KEY ("expected_plan_id") REFERENCES "public"."subscription_plans"("id"),
    CONSTRAINT "subscription_renewal_attempts_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX subscription_renewal_attempts_renewal_attempt_key_key ON public.subscription_renewal_attempts USING btree (renewal_attempt_key);

-- Table Definition
CREATE TABLE "public"."ai_cost_policies" (
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


-- Indices
CREATE UNIQUE INDEX ai_cost_policies_code_version_key ON public.ai_cost_policies USING btree (code, version);

-- Table Definition
CREATE TABLE "public"."ai_cost_policy_operations" (
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
    "max_paid_attempts_per_day" int4 NOT NULL,
    "paid_fallback_enabled" bool NOT NULL DEFAULT false,
    "is_enabled" bool NOT NULL DEFAULT true,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "ai_cost_policy_operations_policy_id_fkey" FOREIGN KEY ("policy_id") REFERENCES "public"."ai_cost_policies"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX ai_cost_policy_operations_policy_id_operation_key ON public.ai_cost_policy_operations USING btree (policy_id, operation);

-- Table Definition
CREATE TABLE "public"."subscription_plans" (
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
    "plan_kind" int2 NOT NULL DEFAULT 0,
    "tier_rank" int4 NOT NULL DEFAULT 0,
    "pricing_version" int8 NOT NULL DEFAULT 1,
    "ai_cost_policy_id" uuid NOT NULL,
    CONSTRAINT "subscription_plans_ai_cost_policy_id_fkey" FOREIGN KEY ("ai_cost_policy_id") REFERENCES "public"."ai_cost_policies"("id") ON DELETE RESTRICT,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX subscription_plans_slug_key ON public.subscription_plans USING btree (slug);
CREATE UNIQUE INDEX ux_single_active_default_free_plan ON public.subscription_plans USING btree (plan_kind) WHERE ((plan_kind = 0) AND (is_active = true));

-- Table Definition
CREATE TABLE "public"."ai_usage_events" (
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
    CONSTRAINT "ai_usage_events_ledger_id_fkey" FOREIGN KEY ("ledger_id") REFERENCES "public"."ai_usage_period_ledgers"("id") ON DELETE CASCADE,
    CONSTRAINT "ai_usage_events_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    CONSTRAINT "ai_usage_events_pricing_config_id_fkey" FOREIGN KEY ("pricing_config_id") REFERENCES "public"."ai_model_configs"("id"),
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX ai_usage_events_request_id_key ON public.ai_usage_events USING btree (request_id);
CREATE INDEX ix_ai_usage_events_user_day ON public.ai_usage_events USING btree (user_id, created_at, status);
CREATE INDEX ix_ai_usage_events_unknown ON public.ai_usage_events USING btree (status, unknown_expires_at) WHERE ((status)::text = ANY ((ARRAY['reserved'::character varying, 'unknown_usage'::character varying])::text[]));

-- Table Definition
CREATE TABLE "public"."brand_members" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "role" varchar(50) NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'ACTIVE'::character varying,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "brand_members_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "brand_members_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_brand_members_brand_user ON public.brand_members USING btree (brand_id, user_id);
CREATE INDEX idx_brand_members_user_status ON public.brand_members USING btree (user_id, status);
CREATE UNIQUE INDEX ux_brand_members_one_active_owner ON public.brand_members USING btree (brand_id) WHERE (((role)::text = 'owner'::text) AND ((status)::text = 'active'::text));

-- Table Definition
CREATE TABLE "public"."ai_usage_period_ledgers" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "grant_id" uuid NOT NULL,
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
    CONSTRAINT "ai_usage_period_ledgers_grant_id_fkey" FOREIGN KEY ("grant_id") REFERENCES "public"."ai_policy_grants"("id") ON DELETE CASCADE,
    CONSTRAINT "ai_usage_period_ledgers_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX ai_usage_period_ledgers_grant_id_period_index_key ON public.ai_usage_period_ledgers USING btree (grant_id, period_index);

-- Table Definition
CREATE TABLE "public"."outfit_items" (
    "outfit_id" uuid NOT NULL,
    "position_x" float8 NOT NULL DEFAULT 0.0,
    "position_y" float8 NOT NULL DEFAULT 0.0,
    "scale" float8 NOT NULL DEFAULT 1.0,
    "layer_order" int2 NOT NULL DEFAULT 1,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "fashion_item_id" uuid NOT NULL,
    "item_context" varchar(50) NOT NULL,
    CONSTRAINT "fk_outfit_items_fashion_item" FOREIGN KEY ("fashion_item_id") REFERENCES "public"."fashion_items"("id") ON DELETE RESTRICT,
    CONSTRAINT "outfit_items_outfit_id_fkey" FOREIGN KEY ("outfit_id") REFERENCES "public"."outfits"("id") ON DELETE CASCADE,
    PRIMARY KEY ("outfit_id","fashion_item_id","item_context")
);


-- Indices
CREATE INDEX idx_outfit_items_fashion_item_context ON public.outfit_items USING btree (fashion_item_id, item_context);

-- Table Definition
CREATE TABLE "public"."brands" (
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
    CONSTRAINT "brands_created_by_user_id_fkey" FOREIGN KEY ("created_by_user_id") REFERENCES "public"."users"("id") ON DELETE RESTRICT,
    CONSTRAINT "brands_approved_by_user_id_fkey" FOREIGN KEY ("approved_by_user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX brands_slug_key ON public.brands USING btree (slug);

-- Table Definition
CREATE TABLE "public"."ai_policy_grants" (
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
    CONSTRAINT "user_ai_policy_grants_policy_id_fkey" FOREIGN KEY ("policy_id") REFERENCES "public"."ai_cost_policies"("id") ON DELETE RESTRICT,
    CONSTRAINT "user_ai_policy_grants_plan_id_fkey" FOREIGN KEY ("plan_id") REFERENCES "public"."subscription_plans"("id") ON DELETE RESTRICT,
    CONSTRAINT "user_ai_policy_grants_source_event_id_fkey" FOREIGN KEY ("source_event_id") REFERENCES "public"."user_subscription_events"("id") ON DELETE SET NULL,
    CONSTRAINT "user_ai_policy_grants_source_deposit_transaction_id_fkey" FOREIGN KEY ("source_deposit_transaction_id") REFERENCES "public"."deposit_transactions"("id") ON DELETE SET NULL,
    CONSTRAINT "user_ai_policy_grants_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX user_ai_policy_grants_pkey ON public.ai_policy_grants USING btree (id);
CREATE INDEX ix_user_ai_policy_grants_active ON public.ai_policy_grants USING btree (user_id, effective_from, effective_to);

-- Table Definition
CREATE TABLE "public"."fashion_items" (
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
    CONSTRAINT "fashion_items_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "public"."categories"("id") ON DELETE RESTRICT,
    PRIMARY KEY ("id")
);


-- Indices
CREATE INDEX idx_fashion_items_category_id ON public.fashion_items USING btree (category_id);
CREATE INDEX fitems_embedding_cosine_idx ON public.fashion_items USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_fashion_items_lexical_search ON public.fashion_items USING gin (to_tsvector('simple'::regconfig, immutable_unaccent(lower((((((((((((((COALESCE(color, ''::character varying))::text || ' '::text) || (COALESCE(style, ''::character varying))::text) || ' '::text) || (COALESCE(material, ''::character varying))::text) || ' '::text) || (COALESCE(pattern, ''::character varying))::text) || ' '::text) || (COALESCE(fit, ''::character varying))::text) || ' '::text) || (COALESCE(seasonality, ''::character varying))::text) || ' '::text) || COALESCE(description, ''::text))))));

-- Table Definition
CREATE TABLE "public"."loyalty_point_transactions" (
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
    CONSTRAINT "loyalty_point_transactions_loyalty_account_id_fkey" FOREIGN KEY ("loyalty_account_id") REFERENCES "public"."loyalty_accounts"("id") ON DELETE CASCADE,
    CONSTRAINT "loyalty_point_transactions_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "loyalty_point_transactions_brand_customer_id_fkey" FOREIGN KEY ("brand_customer_id") REFERENCES "public"."brand_customers"("id") ON DELETE CASCADE,
    CONSTRAINT "loyalty_point_transactions_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    CONSTRAINT "loyalty_point_transactions_created_by_user_id_fkey" FOREIGN KEY ("created_by_user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_loyalty_point_transactions_brand_idempotency ON public.loyalty_point_transactions USING btree (brand_id, idempotency_key) WHERE (idempotency_key IS NOT NULL);
CREATE INDEX idx_loyalty_point_transactions_account_created ON public.loyalty_point_transactions USING btree (loyalty_account_id, created_at);
CREATE INDEX idx_loyalty_point_transactions_customer_created ON public.loyalty_point_transactions USING btree (brand_customer_id, created_at);
CREATE INDEX idx_loyalty_point_transactions_brand_user ON public.loyalty_point_transactions USING btree (brand_id, user_id);
CREATE INDEX idx_loyalty_point_transactions_expires_at ON public.loyalty_point_transactions USING btree (expires_at) WHERE (expires_at IS NOT NULL);

-- Table Definition
CREATE TABLE "public"."loyalty_tiers" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    "rank" int4 NOT NULL,
    "min_total_spend" numeric(12,2) NOT NULL DEFAULT 0,
    "description" text,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "loyalty_tiers_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_loyalty_tiers_brand_rank ON public.loyalty_tiers USING btree (brand_id, rank);
CREATE UNIQUE INDEX uq_loyalty_tiers_brand_name ON public.loyalty_tiers USING btree (brand_id, name);

-- Table Definition
CREATE TABLE "public"."loyalty_accounts" (
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
    CONSTRAINT "loyalty_accounts_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "loyalty_accounts_brand_customer_id_fkey" FOREIGN KEY ("brand_customer_id") REFERENCES "public"."brand_customers"("id") ON DELETE CASCADE,
    CONSTRAINT "loyalty_accounts_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    CONSTRAINT "loyalty_accounts_current_tier_id_fkey" FOREIGN KEY ("current_tier_id") REFERENCES "public"."loyalty_tiers"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_loyalty_accounts_brand_user ON public.loyalty_accounts USING btree (brand_id, user_id) WHERE (user_id IS NOT NULL);
CREATE INDEX idx_loyalty_accounts_brand_tier ON public.loyalty_accounts USING btree (brand_id, current_tier_id);
CREATE UNIQUE INDEX uq_loyalty_accounts_brand_customer ON public.loyalty_accounts USING btree (brand_customer_id);

-- Table Definition
CREATE TABLE "public"."brand_customer_claims" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_customer_id" uuid NOT NULL,
    "claim_token_hash" varchar(255) NOT NULL,
    "expires_at" timestamptz NOT NULL,
    "consumed_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "revoked_at" timestamptz,
    "revoked_by_user_id" uuid,
    "revoked_reason" varchar(255),
    CONSTRAINT "brand_customer_claims_brand_customer_id_fkey" FOREIGN KEY ("brand_customer_id") REFERENCES "public"."brand_customers"("id") ON DELETE CASCADE,
    CONSTRAINT "brand_customer_claims_revoked_by_user_id_fkey" FOREIGN KEY ("revoked_by_user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_brand_customer_claims_token_hash ON public.brand_customer_claims USING btree (claim_token_hash);
CREATE INDEX idx_brand_customer_claims_customer ON public.brand_customer_claims USING btree (brand_customer_id);
CREATE INDEX idx_brand_customer_claims_active_customer ON public.brand_customer_claims USING btree (brand_customer_id) WHERE ((consumed_at IS NULL) AND (revoked_at IS NULL));

-- Table Definition
CREATE TABLE "public"."brand_customers" (
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
    CONSTRAINT "brand_customers_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "brand_customers_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    CONSTRAINT "brand_customers_created_by_member_id_fkey" FOREIGN KEY ("created_by_member_id") REFERENCES "public"."brand_members"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_brand_customers_brand_user ON public.brand_customers USING btree (brand_id, user_id) WHERE (user_id IS NOT NULL);
CREATE UNIQUE INDEX uq_brand_customers_brand_phone_hash ON public.brand_customers USING btree (brand_id, phone_hash) WHERE (phone_hash IS NOT NULL);
CREATE INDEX idx_brand_customers_brand_status ON public.brand_customers USING btree (brand_id, status);

-- Table Definition
CREATE TABLE "public"."loyalty_programs" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_id" uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    "amount_per_point" numeric(12,2) NOT NULL,
    "point_expiry_days" int4,
    "rounding_mode" varchar(50) NOT NULL DEFAULT 'FLOOR'::character varying,
    "is_active" bool NOT NULL DEFAULT true,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "loyalty_programs_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_loyalty_programs_active_brand ON public.loyalty_programs USING btree (brand_id) WHERE (is_active = true);

-- Table Definition
CREATE TABLE "public"."brand_benefits" (
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
    CONSTRAINT "brand_benefits_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "brand_benefits_required_tier_id_fkey" FOREIGN KEY ("required_tier_id") REFERENCES "public"."loyalty_tiers"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE INDEX idx_brand_benefits_brand_status ON public.brand_benefits USING btree (brand_id, status);

-- Table Definition
CREATE TABLE "public"."benefit_redemptions" (
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
    CONSTRAINT "benefit_redemptions_benefit_id_fkey" FOREIGN KEY ("benefit_id") REFERENCES "public"."brand_benefits"("id") ON DELETE CASCADE,
    CONSTRAINT "benefit_redemptions_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "benefit_redemptions_brand_customer_id_fkey" FOREIGN KEY ("brand_customer_id") REFERENCES "public"."brand_customers"("id") ON DELETE CASCADE,
    CONSTRAINT "benefit_redemptions_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE INDEX idx_benefit_redemptions_user_status ON public.benefit_redemptions USING btree (user_id, status);
CREATE INDEX idx_benefit_redemptions_customer_status ON public.benefit_redemptions USING btree (brand_customer_id, status);

-- Table Definition
CREATE TABLE "public"."brand_conversations" (
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
    CONSTRAINT "brand_conversations_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "brand_conversations_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    CONSTRAINT "brand_conversations_closed_by_user_id_fkey" FOREIGN KEY ("closed_by_user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_brand_conversations_brand_user ON public.brand_conversations USING btree (brand_id, user_id);
CREATE INDEX idx_brand_conversations_brand_status ON public.brand_conversations USING btree (brand_id, status);

-- Table Definition
CREATE TABLE "public"."brand_conversation_messages" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "conversation_id" uuid NOT NULL,
    "sender_user_id" uuid,
    "sender_role" varchar(50) NOT NULL,
    "message" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "brand_conversation_messages_conversation_id_fkey" FOREIGN KEY ("conversation_id") REFERENCES "public"."brand_conversations"("id") ON DELETE CASCADE,
    CONSTRAINT "brand_conversation_messages_sender_user_id_fkey" FOREIGN KEY ("sender_user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    PRIMARY KEY ("id")
);


-- Indices
CREATE INDEX idx_brand_conversation_messages_conv_created ON public.brand_conversation_messages USING btree (conversation_id, created_at);
CREATE INDEX idx_brand_conversation_messages_unread ON public.brand_conversation_messages USING btree (conversation_id, sender_role, created_at);

-- Table Definition
CREATE TABLE "public"."loyalty_point_lots" (
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
    CONSTRAINT "loyalty_point_lots_loyalty_account_id_fkey" FOREIGN KEY ("loyalty_account_id") REFERENCES "public"."loyalty_accounts"("id") ON DELETE CASCADE,
    CONSTRAINT "loyalty_point_lots_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "loyalty_point_lots_brand_customer_id_fkey" FOREIGN KEY ("brand_customer_id") REFERENCES "public"."brand_customers"("id") ON DELETE CASCADE,
    CONSTRAINT "loyalty_point_lots_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE SET NULL,
    CONSTRAINT "loyalty_point_lots_earn_transaction_id_fkey" FOREIGN KEY ("earn_transaction_id") REFERENCES "public"."loyalty_point_transactions"("id") ON DELETE RESTRICT,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX ux_loyalty_point_lots_earn_transaction ON public.loyalty_point_lots USING btree (earn_transaction_id);
CREATE INDEX idx_loyalty_point_lots_account_active_expiry ON public.loyalty_point_lots USING btree (loyalty_account_id, expires_at) WHERE (((status)::text = 'ACTIVE'::text) AND (remaining_points > 0));
CREATE INDEX idx_loyalty_point_lots_expiry_worker ON public.loyalty_point_lots USING btree (expires_at) WHERE (((status)::text = 'ACTIVE'::text) AND (remaining_points > 0) AND (expires_at IS NOT NULL));
CREATE INDEX idx_loyalty_point_lots_brand_customer ON public.loyalty_point_lots USING btree (brand_customer_id);

-- Table Definition
CREATE TABLE "public"."brand_items" (
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
    CONSTRAINT "brand_items_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands"("id") ON DELETE CASCADE,
    CONSTRAINT "brand_items_fashion_item_id_fkey" FOREIGN KEY ("fashion_item_id") REFERENCES "public"."fashion_items"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_brand_items_brand_product_code ON public.brand_items USING btree (brand_id, product_code);
CREATE INDEX idx_brand_items_brand_status ON public.brand_items USING btree (brand_id, status);
CREATE UNIQUE INDEX uq_brand_items_fashion_item ON public.brand_items USING btree (fashion_item_id) WHERE (fashion_item_id IS NOT NULL);
CREATE INDEX idx_brand_items_task_id ON public.brand_items USING btree (task_id);

-- Table Definition
CREATE TABLE "public"."digital_sample_responses" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "brand_item_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "outfit_id" uuid,
    "vote_type" varchar(50),
    "rating" int4,
    "feedback_text" text,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "digital_sample_responses_brand_item_id_fkey" FOREIGN KEY ("brand_item_id") REFERENCES "public"."brand_items"("id") ON DELETE CASCADE,
    CONSTRAINT "digital_sample_responses_outfit_id_fkey" FOREIGN KEY ("outfit_id") REFERENCES "public"."outfits"("id") ON DELETE SET NULL,
    CONSTRAINT "digital_sample_responses_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE INDEX idx_digital_sample_responses_item_user ON public.digital_sample_responses USING btree (brand_item_id, user_id);

-- Table Definition
CREATE TABLE "public"."ai_model_configs" (
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
    CONSTRAINT "ai_model_configs_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."users"("id"),
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_provider_model_active ON public.ai_model_configs USING btree (provider, model, effective_from);

