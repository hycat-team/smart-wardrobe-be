-- Database schema dump (schema-qualified theo module) - generated Sprint 2
-- 45 bảng + 1 sequence, tách theo module. Nguồn: query pg_catalog trên DB thực.

CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS wardrobe;
CREATE SCHEMA IF NOT EXISTS fashion;
CREATE SCHEMA IF NOT EXISTS brand;
CREATE SCHEMA IF NOT EXISTS subscription;
CREATE SCHEMA IF NOT EXISTS aiexecutor;
CREATE SCHEMA IF NOT EXISTS dashboard;
CREATE SCHEMA IF NOT EXISTS shared;


-- Table Definition
CREATE TABLE identity.refresh_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    token character varying(500) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    is_revoked boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX refresh_tokens_token_key ON identity.refresh_tokens USING btree (token);

-- Table Definition
CREATE TABLE identity.user_style_profiles (
    user_id uuid NOT NULL,
    taste_embedding vector(768),
    preferred_colors jsonb,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (user_id)
);

CREATE INDEX ustyles_embedding_cosine_idx ON identity.user_style_profiles USING hnsw (taste_embedding vector_cosine_ops);

-- Table Definition
CREATE TABLE identity.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    username character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    first_name character varying(255),
    last_name character varying(255),
    date_of_birth date,
    address character varying(255),
    gender integer,
    role_slug character varying(50) NOT NULL,
    body_profile jsonb,
    status smallint DEFAULT 0 NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    avatar_url character varying(500),
    avatar_public_id character varying(255),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX idx_users_email_lower ON identity.users USING btree (lower((email)::text));
CREATE UNIQUE INDEX idx_users_username_lower ON identity.users USING btree (lower((username)::text));

-- Table Definition
CREATE TABLE wardrobe.outfit_items (
    outfit_id uuid NOT NULL,
    position_x double precision DEFAULT 0.0 NOT NULL,
    position_y double precision DEFAULT 0.0 NOT NULL,
    scale double precision DEFAULT 1.0 NOT NULL,
    layer_order smallint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fashion_item_id uuid NOT NULL,
    item_context character varying(50) NOT NULL,
    PRIMARY KEY (outfit_id, fashion_item_id, item_context)
);

CREATE INDEX idx_outfit_items_fashion_item_context ON wardrobe.outfit_items USING btree (fashion_item_id, item_context);

-- Table Definition
CREATE TABLE wardrobe.outfits (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    cover_image_url character varying(500),
    cover_public_id character varying(255),
    status smallint DEFAULT 1 NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);


-- Table Definition
CREATE TABLE wardrobe.wardrobe_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    purchase_price numeric(12,2),
    status smallint DEFAULT 0 NOT NULL,
    item_type smallint DEFAULT 0 NOT NULL,
    last_used_at timestamp with time zone,
    is_deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fashion_item_id uuid,
    task_id uuid,
    PRIMARY KEY (id)
);

CREATE INDEX idx_wardrobe_items_fashion_item ON wardrobe.wardrobe_items USING btree (fashion_item_id);
CREATE INDEX idx_wardrobe_items_last_used_at ON wardrobe.wardrobe_items USING btree (last_used_at);
CREATE INDEX idx_wardrobe_items_task_id ON wardrobe.wardrobe_items USING btree (task_id);
CREATE INDEX idx_wardrobe_items_type_status ON wardrobe.wardrobe_items USING btree (item_type, is_deleted);
CREATE INDEX idx_wardrobe_items_user_fashion_item ON wardrobe.wardrobe_items USING btree (user_id, fashion_item_id);
CREATE INDEX idx_wardrobe_items_user_status ON wardrobe.wardrobe_items USING btree (user_id, is_deleted, status);

-- Table Definition
CREATE TABLE fashion.categories (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(100) NOT NULL,
    slug character varying(100) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX categories_name_key ON fashion.categories USING btree (name);
CREATE UNIQUE INDEX categories_slug_key ON fashion.categories USING btree (slug);

-- Table Definition
CREATE TABLE fashion.conversational_contexts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    context_summary text DEFAULT ''::text,
    is_archived boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);


-- Table Definition
CREATE TABLE fashion.fashion_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    category_id uuid,
    image_url character varying(500) NOT NULL,
    image_public_id character varying(255) NOT NULL,
    color character varying(50),
    color_hex character varying(7),
    color_hue double precision,
    color_saturation double precision,
    color_lightness double precision,
    style character varying(100),
    material character varying(100),
    pattern character varying(100),
    fit character varying(50),
    seasonality character varying(100),
    description text,
    embedding vector(768),
    processing_retry_count integer DEFAULT 0 NOT NULL,
    processing_version integer DEFAULT 0 NOT NULL,
    processing_started_at timestamp with time zone,
    last_processing_attempt_at timestamp with time zone,
    processing_error_reason text,
    review_reason character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX fitems_embedding_cosine_idx ON fashion.fashion_items USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_fashion_items_category_id ON fashion.fashion_items USING btree (category_id);
CREATE INDEX idx_fashion_items_lexical_search ON fashion.fashion_items USING gin (to_tsvector('simple'::regconfig, immutable_unaccent(lower((((((((((((((COALESCE(color, ''::character varying))::text || ' '::text) || (COALESCE(style, ''::character varying))::text) || ' '::text) || (COALESCE(material, ''::character varying))::text) || ' '::text) || (COALESCE(pattern, ''::character varying))::text) || ' '::text) || (COALESCE(fit, ''::character varying))::text) || ' '::text) || (COALESCE(seasonality, ''::character varying))::text) || ' '::text) || COALESCE(description, ''::text))))));

-- Table Definition
CREATE TABLE fashion.messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    context_id uuid NOT NULL,
    sender character varying(50) NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    is_summarized boolean DEFAULT false NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX idx_messages_context_created_at ON fashion.messages USING btree (context_id, created_at DESC);
CREATE INDEX idx_messages_context_unsummarized ON fashion.messages USING btree (context_id) WHERE (is_summarized = false);

-- Table Definition
CREATE TABLE brand.benefit_redemptions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    benefit_id uuid NOT NULL,
    brand_id uuid NOT NULL,
    brand_customer_id uuid NOT NULL,
    user_id uuid,
    points_spent integer DEFAULT 0 NOT NULL,
    status character varying(50) DEFAULT 'REDEEMED'::character varying NOT NULL,
    redeemed_at timestamp with time zone DEFAULT now() NOT NULL,
    used_at timestamp with time zone,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX idx_benefit_redemptions_customer_status ON brand.benefit_redemptions USING btree (brand_customer_id, status);
CREATE INDEX idx_benefit_redemptions_user_status ON brand.benefit_redemptions USING btree (user_id, status);

-- Table Definition
CREATE TABLE brand.brand_benefits (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    benefit_type character varying(50) NOT NULL,
    unlock_type character varying(50) NOT NULL,
    required_points integer,
    required_tier_id uuid,
    feature_code character varying(100),
    feature_config jsonb,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_brand_benefits_status CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'inactive'::character varying, 'archived'::character varying])::text[])))
);

CREATE INDEX idx_brand_benefits_brand_status ON brand.brand_benefits USING btree (brand_id, status);

-- Table Definition
CREATE TABLE brand.brand_conversation_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    conversation_id uuid NOT NULL,
    sender_user_id uuid,
    sender_role character varying(50) NOT NULL,
    message text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX idx_brand_conversation_messages_conv_created ON brand.brand_conversation_messages USING btree (conversation_id, created_at);
CREATE INDEX idx_brand_conversation_messages_unread ON brand.brand_conversation_messages USING btree (conversation_id, sender_role, created_at);

-- Table Definition
CREATE TABLE brand.brand_conversations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    user_id uuid NOT NULL,
    status character varying(50) DEFAULT 'OPEN'::character varying NOT NULL,
    last_message_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_last_read_at timestamp with time zone,
    staff_last_read_at timestamp with time zone,
    closed_at timestamp with time zone,
    closed_by_user_id uuid,
    PRIMARY KEY (id)
);

CREATE INDEX idx_brand_conversations_brand_status ON brand.brand_conversations USING btree (brand_id, status);
CREATE UNIQUE INDEX uq_brand_conversations_brand_user ON brand.brand_conversations USING btree (brand_id, user_id);

-- Table Definition
CREATE TABLE brand.brand_customer_claims (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_customer_id uuid NOT NULL,
    claim_token_hash character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone,
    revoked_by_user_id uuid,
    revoked_reason character varying(255),
    PRIMARY KEY (id)
);

CREATE INDEX idx_brand_customer_claims_active_customer ON brand.brand_customer_claims USING btree (brand_customer_id) WHERE ((consumed_at IS NULL) AND (revoked_at IS NULL));
CREATE INDEX idx_brand_customer_claims_customer ON brand.brand_customer_claims USING btree (brand_customer_id);
CREATE UNIQUE INDEX uq_brand_customer_claims_token_hash ON brand.brand_customer_claims USING btree (claim_token_hash);

-- Table Definition
CREATE TABLE brand.brand_customers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    user_id uuid,
    customer_name character varying(255),
    phone_e164 character varying(50),
    phone_hash character varying(255),
    external_customer_code character varying(100),
    joined_source character varying(50) NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    joined_at timestamp with time zone DEFAULT now() NOT NULL,
    claimed_at timestamp with time zone,
    created_by_member_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_brand_customers_status CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'blocked'::character varying, 'left'::character varying])::text[])))
);

CREATE INDEX idx_brand_customers_brand_status ON brand.brand_customers USING btree (brand_id, status);
CREATE UNIQUE INDEX uq_brand_customers_brand_phone_hash ON brand.brand_customers USING btree (brand_id, phone_hash) WHERE (phone_hash IS NOT NULL);
CREATE UNIQUE INDEX uq_brand_customers_brand_user ON brand.brand_customers USING btree (brand_id, user_id) WHERE (user_id IS NOT NULL);

-- Table Definition
CREATE TABLE brand.brand_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    fashion_item_id uuid,
    product_code character varying(100),
    name character varying(255) NOT NULL,
    description text,
    price numeric(12,2),
    item_type character varying(50) NOT NULL,
    status character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_id uuid,
    PRIMARY KEY (id),
    CONSTRAINT chk_brand_items_status CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'active'::character varying, 'archived'::character varying])::text[])))
);

CREATE INDEX idx_brand_items_brand_status ON brand.brand_items USING btree (brand_id, status);
CREATE INDEX idx_brand_items_task_id ON brand.brand_items USING btree (task_id);
CREATE UNIQUE INDEX uq_brand_items_brand_product_code ON brand.brand_items USING btree (brand_id, product_code);
CREATE UNIQUE INDEX uq_brand_items_fashion_item ON brand.brand_items USING btree (fashion_item_id) WHERE (fashion_item_id IS NOT NULL);

-- Table Definition
CREATE TABLE brand.brand_members (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role character varying(50) NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_brand_members_role CHECK (((role)::text = ANY ((ARRAY['owner'::character varying, 'staff'::character varying])::text[]))),
    CONSTRAINT chk_brand_members_status CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'invited'::character varying, 'disabled'::character varying])::text[])))
);

CREATE INDEX idx_brand_members_user_status ON brand.brand_members USING btree (user_id, status);
CREATE UNIQUE INDEX uq_brand_members_brand_user ON brand.brand_members USING btree (brand_id, user_id);
CREATE UNIQUE INDEX ux_brand_members_one_active_owner ON brand.brand_members USING btree (brand_id) WHERE (((role)::text = 'owner'::text) AND ((status)::text = 'active'::text));

-- Table Definition
CREATE TABLE brand.brands (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    slug character varying(100) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    logo_url character varying(500),
    logo_public_id character varying(255),
    status character varying(50) DEFAULT 'PENDING_REVIEW'::character varying NOT NULL,
    created_by_user_id uuid NOT NULL,
    approved_by_user_id uuid,
    approved_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    background_url character varying(500),
    background_public_id character varying(255),
    PRIMARY KEY (id),
    CONSTRAINT chk_brands_status CHECK (((status)::text = ANY ((ARRAY['pending_review'::character varying, 'active'::character varying, 'suspended'::character varying, 'archived'::character varying])::text[])))
);

CREATE UNIQUE INDEX brands_slug_key ON brand.brands USING btree (slug);

-- Table Definition
CREATE TABLE brand.digital_sample_responses (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_item_id uuid NOT NULL,
    user_id uuid NOT NULL,
    outfit_id uuid,
    vote_type character varying(50),
    rating integer,
    feedback_text text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX idx_digital_sample_responses_item_user ON brand.digital_sample_responses USING btree (brand_item_id, user_id);

-- Table Definition
CREATE TABLE brand.loyalty_accounts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    brand_customer_id uuid NOT NULL,
    user_id uuid,
    current_points integer DEFAULT 0 NOT NULL,
    lifetime_points integer DEFAULT 0 NOT NULL,
    total_spend numeric(12,2) DEFAULT 0 NOT NULL,
    current_tier_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_loyalty_accounts_current_points CHECK ((current_points >= 0)),
    CONSTRAINT chk_loyalty_accounts_lifetime_points CHECK ((lifetime_points >= 0)),
    CONSTRAINT chk_loyalty_accounts_total_spend CHECK ((total_spend >= (0)::numeric))
);

CREATE INDEX idx_loyalty_accounts_brand_tier ON brand.loyalty_accounts USING btree (brand_id, current_tier_id);
CREATE UNIQUE INDEX uq_loyalty_accounts_brand_customer ON brand.loyalty_accounts USING btree (brand_customer_id);
CREATE UNIQUE INDEX uq_loyalty_accounts_brand_user ON brand.loyalty_accounts USING btree (brand_id, user_id) WHERE (user_id IS NOT NULL);

-- Table Definition
CREATE TABLE brand.loyalty_point_lots (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    loyalty_account_id uuid NOT NULL,
    brand_id uuid NOT NULL,
    brand_customer_id uuid NOT NULL,
    user_id uuid,
    earn_transaction_id uuid NOT NULL,
    earned_points integer NOT NULL,
    remaining_points integer NOT NULL,
    expires_at timestamp with time zone,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_loyalty_point_lots_earned_positive CHECK ((earned_points > 0)),
    CONSTRAINT chk_loyalty_point_lots_remaining_lte_earned CHECK ((remaining_points <= earned_points)),
    CONSTRAINT chk_loyalty_point_lots_remaining_non_negative CHECK ((remaining_points >= 0)),
    CONSTRAINT chk_loyalty_point_lots_status CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'consumed'::character varying, 'expired'::character varying])::text[])))
);

CREATE INDEX idx_loyalty_point_lots_account_active_expiry ON brand.loyalty_point_lots USING btree (loyalty_account_id, expires_at) WHERE (((status)::text = 'ACTIVE'::text) AND (remaining_points > 0));
CREATE INDEX idx_loyalty_point_lots_brand_customer ON brand.loyalty_point_lots USING btree (brand_customer_id);
CREATE INDEX idx_loyalty_point_lots_expiry_worker ON brand.loyalty_point_lots USING btree (expires_at) WHERE (((status)::text = 'ACTIVE'::text) AND (remaining_points > 0) AND (expires_at IS NOT NULL));
CREATE UNIQUE INDEX ux_loyalty_point_lots_earn_transaction ON brand.loyalty_point_lots USING btree (earn_transaction_id);

-- Table Definition
CREATE TABLE brand.loyalty_point_transactions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    loyalty_account_id uuid NOT NULL,
    brand_id uuid NOT NULL,
    brand_customer_id uuid NOT NULL,
    user_id uuid,
    points_delta integer NOT NULL,
    balance_after integer NOT NULL,
    transaction_type character varying(50) NOT NULL,
    reason character varying(255),
    spend_amount numeric(12,2),
    reference_type character varying(100),
    reference_id uuid,
    expires_at timestamp with time zone,
    idempotency_key character varying(100),
    created_by_user_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_loyalty_point_transactions_balance_after CHECK ((balance_after >= 0)),
    CONSTRAINT chk_loyalty_point_transactions_spend_amount CHECK (((spend_amount IS NULL) OR (spend_amount >= (0)::numeric)))
);

CREATE INDEX idx_loyalty_point_transactions_account_created ON brand.loyalty_point_transactions USING btree (loyalty_account_id, created_at);
CREATE INDEX idx_loyalty_point_transactions_brand_user ON brand.loyalty_point_transactions USING btree (brand_id, user_id);
CREATE INDEX idx_loyalty_point_transactions_customer_created ON brand.loyalty_point_transactions USING btree (brand_customer_id, created_at);
CREATE INDEX idx_loyalty_point_transactions_expires_at ON brand.loyalty_point_transactions USING btree (expires_at) WHERE (expires_at IS NOT NULL);
CREATE UNIQUE INDEX uq_loyalty_point_transactions_brand_idempotency ON brand.loyalty_point_transactions USING btree (brand_id, idempotency_key) WHERE (idempotency_key IS NOT NULL);

-- Table Definition
CREATE TABLE brand.loyalty_programs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    amount_per_point numeric(12,2) NOT NULL,
    point_expiry_days integer,
    rounding_mode character varying(50) DEFAULT 'FLOOR'::character varying NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_loyalty_programs_amount_per_point CHECK ((amount_per_point > (0)::numeric)),
    CONSTRAINT chk_loyalty_programs_point_expiry_days CHECK (((point_expiry_days IS NULL) OR (point_expiry_days > 0)))
);

CREATE UNIQUE INDEX uq_loyalty_programs_active_brand ON brand.loyalty_programs USING btree (brand_id) WHERE (is_active = true);

-- Table Definition
CREATE TABLE brand.loyalty_tiers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    rank integer NOT NULL,
    min_total_spend numeric(12,2) DEFAULT 0 NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_loyalty_tiers_min_total_spend CHECK ((min_total_spend >= (0)::numeric))
);

CREATE UNIQUE INDEX uq_loyalty_tiers_brand_name ON brand.loyalty_tiers USING btree (brand_id, name);
CREATE UNIQUE INDEX uq_loyalty_tiers_brand_rank ON brand.loyalty_tiers USING btree (brand_id, rank);

-- Table Definition
CREATE TABLE subscription.deposit_transactions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    amount numeric(12,2) NOT NULL,
    currency character varying(10) DEFAULT 'VND'::character varying NOT NULL,
    status smallint DEFAULT 0 NOT NULL,
    transaction_type character varying(50) NOT NULL,
    subscription_plan_id uuid,
    order_code bigint DEFAULT nextval('deposit_transactions_order_code_seq'::regclass) NOT NULL,
    gateway_reference character varying(255),
    gateway_details text,
    payment_url character varying(500),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    provider character varying(50) DEFAULT 'PAYOS'::character varying NOT NULL,
    payment_link_id character varying(255),
    provider_status character varying(50),
    successful_provider_reference character varying(255),
    expires_at timestamp with time zone,
    next_reconciliation_at timestamp with time zone,
    reconciliation_attempts integer DEFAULT 0 NOT NULL,
    processing_token uuid,
    processing_lease_until timestamp with time zone,
    last_provider_error_code character varying(100),
    last_provider_error_at timestamp with time zone,
    failure_reason character varying(100),
    cancelled_at timestamp with time zone,
    expired_at timestamp with time zone,
    plan_code character varying(100),
    plan_name character varying(100),
    tier_rank integer,
    plan_kind smallint,
    purchased_duration_days integer,
    expected_amount numeric(12,2) NOT NULL,
    benefit_snapshot jsonb,
    pricing_version bigint,
    benefit_resolution character varying(100),
    benefit_applied_at timestamp with time zone,
    benefit_result_snapshot jsonb,
    wallet_credit_amount numeric(12,2),
    subscription_version_before bigint,
    subscription_version_after bigint,
    refunded_at timestamp with time zone,
    refund_amount numeric(12,2),
    refund_reason character varying(255),
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX deposit_transactions_order_code_key ON subscription.deposit_transactions USING btree (order_code);
CREATE INDEX idx_deposit_reconciliation ON subscription.deposit_transactions USING btree (next_reconciliation_at, expires_at) WHERE (status = ANY (ARRAY[0, 3, 4, 5]));
CREATE UNIQUE INDEX ux_active_direct_purchase_per_user ON subscription.deposit_transactions USING btree (user_id) WHERE (((transaction_type)::text = 'DIRECT_PURCHASE'::text) AND (status = ANY (ARRAY[0, 3, 4, 5])));

-- Table Definition
CREATE TABLE subscription.provider_payment_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    provider character varying(50) NOT NULL,
    provider_reference character varying(255) NOT NULL,
    event_code character varying(100) NOT NULL,
    order_code bigint NOT NULL,
    payment_link_id character varying(255),
    amount numeric(12,2) NOT NULL,
    currency character varying(10) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX provider_payment_events_provider_provider_reference_key ON subscription.provider_payment_events USING btree (provider, provider_reference);

-- Table Definition
CREATE TABLE subscription.provider_webhook_inbox (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    provider character varying(50) NOT NULL,
    provider_reference character varying(255),
    event_code character varying(100) NOT NULL,
    order_code bigint NOT NULL,
    payment_link_id character varying(255),
    amount numeric(12,2) NOT NULL,
    currency character varying(10) NOT NULL,
    canonical_payload_hash character varying(64) NOT NULL,
    raw_payload jsonb NOT NULL,
    processing_status character varying(50) NOT NULL,
    processing_attempts integer DEFAULT 0 NOT NULL,
    next_processing_at timestamp with time zone,
    processing_token uuid,
    processing_lease_until timestamp with time zone,
    processing_error text,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    processed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT provider_webhook_inbox_processing_status_check CHECK (((processing_status)::text = ANY ((ARRAY['received'::character varying, 'processing'::character varying, 'retry_required'::character varying, 'processed'::character varying, 'investigation_required'::character varying, 'failed'::character varying])::text[])))
);

CREATE INDEX idx_webhook_inbox_processing ON subscription.provider_webhook_inbox USING btree (next_processing_at, processing_lease_until) WHERE ((processing_status)::text = ANY ((ARRAY['RECEIVED'::character varying, 'RETRY_REQUIRED'::character varying, 'PROCESSING'::character varying])::text[]));
CREATE UNIQUE INDEX ux_webhook_payload_hash ON subscription.provider_webhook_inbox USING btree (provider, canonical_payload_hash) WHERE (provider_reference IS NULL);
CREATE UNIQUE INDEX ux_webhook_provider_reference ON subscription.provider_webhook_inbox USING btree (provider, provider_reference, event_code) WHERE (provider_reference IS NOT NULL);

-- Table Definition
CREATE TABLE subscription.subscription_plans (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    slug character varying(100) NOT NULL,
    name character varying(100) NOT NULL,
    price numeric(12,2) DEFAULT 0.00 NOT NULL,
    max_wardrobe_items integer NOT NULL,
    max_outfits integer NOT NULL,
    ai_outfit_daily_quota integer NOT NULL,
    ai_chat_daily_quota integer NOT NULL,
    duration_days integer,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    plan_kind smallint DEFAULT 0 NOT NULL,
    tier_rank integer DEFAULT 0 NOT NULL,
    pricing_version bigint DEFAULT 1 NOT NULL,
    ai_cost_policy_id uuid NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX subscription_plans_slug_key ON subscription.subscription_plans USING btree (slug);
CREATE UNIQUE INDEX ux_single_active_default_free_plan ON subscription.subscription_plans USING btree (plan_kind) WHERE ((plan_kind = 0) AND (is_active = true));

-- Table Definition
CREATE TABLE subscription.subscription_renewal_attempts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    renewal_attempt_key character varying(255) NOT NULL,
    user_id uuid NOT NULL,
    expected_plan_id uuid NOT NULL,
    expected_expires_at timestamp with time zone NOT NULL,
    expected_subscription_version bigint NOT NULL,
    status character varying(50) NOT NULL,
    attempt_count integer DEFAULT 0 NOT NULL,
    last_error_code character varying(100),
    last_error_message text,
    processing_token uuid,
    processing_lease_until timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX subscription_renewal_attempts_renewal_attempt_key_key ON subscription.subscription_renewal_attempts USING btree (renewal_attempt_key);

-- Table Definition
CREATE TABLE subscription.user_daily_quotas (
    user_id uuid NOT NULL,
    outfit_recommend_count integer DEFAULT 0 NOT NULL,
    ai_usage_count integer DEFAULT 0 NOT NULL,
    last_reset_date date DEFAULT CURRENT_DATE NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (user_id)
);


-- Table Definition
CREATE TABLE subscription.user_subscription_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_key character varying(255) NOT NULL,
    user_id uuid NOT NULL,
    event_type character varying(100) NOT NULL,
    from_plan_code character varying(100),
    from_tier_rank integer,
    to_plan_code character varying(100),
    to_tier_rank integer,
    source_deposit_transaction_id uuid,
    actor_admin_id uuid,
    occurred_at timestamp with time zone NOT NULL,
    effective_at timestamp with time zone NOT NULL,
    metadata jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX user_subscription_events_event_key_key ON subscription.user_subscription_events USING btree (event_key);

-- Table Definition
CREATE TABLE subscription.user_subscriptions (
    user_id uuid NOT NULL,
    subscription_plan_id uuid NOT NULL,
    expires_at timestamp with time zone,
    is_auto_renew_enabled boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    current_plan_code character varying(100) NOT NULL,
    current_tier_rank integer NOT NULL,
    current_plan_kind smallint NOT NULL,
    current_benefit_snapshot jsonb DEFAULT '{}'::jsonb NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    fallback_plan_id uuid,
    fallback_plan_code character varying(100),
    fallback_tier_rank integer,
    fallback_plan_kind smallint,
    fallback_benefit_snapshot jsonb,
    last_deposit_transaction_id uuid,
    version bigint DEFAULT 0 NOT NULL,
    PRIMARY KEY (user_id),
    CONSTRAINT chk_user_subscription_auto_renew CHECK (((current_plan_kind = 1) OR (is_auto_renew_enabled = false))),
    CONSTRAINT chk_user_subscription_fallback CHECK ((((fallback_plan_id IS NULL) AND (fallback_plan_code IS NULL) AND (fallback_tier_rank IS NULL) AND (fallback_plan_kind IS NULL) AND (fallback_benefit_snapshot IS NULL)) OR ((fallback_plan_id IS NOT NULL) AND (fallback_plan_code IS NOT NULL) AND (fallback_tier_rank IS NOT NULL) AND (fallback_plan_kind = 2) AND (fallback_benefit_snapshot IS NOT NULL) AND (current_plan_kind = 1) AND (current_tier_rank > fallback_tier_rank)))),
    CONSTRAINT chk_user_subscription_kind_expiry CHECK ((((current_plan_kind = 1) AND (expires_at IS NOT NULL)) OR ((current_plan_kind = ANY (ARRAY[0, 2])) AND (expires_at IS NULL))))
);

CREATE INDEX idx_user_subscriptions_expires ON subscription.user_subscriptions USING btree (expires_at, user_id) WHERE (current_plan_kind = 1);
CREATE INDEX idx_user_subscriptions_plan_id ON subscription.user_subscriptions USING btree (subscription_plan_id);

-- Table Definition
CREATE TABLE subscription.user_wallets (
    user_id uuid NOT NULL,
    balance numeric(12,2) DEFAULT 0.00 NOT NULL,
    currency character varying(10) DEFAULT 'VND'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (user_id)
);


-- Table Definition
CREATE TABLE subscription.wallet_statements (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    amount numeric(12,2) NOT NULL,
    transaction_type character varying(50) NOT NULL,
    previous_balance numeric(12,2) NOT NULL,
    new_balance numeric(12,2) NOT NULL,
    reference_id uuid,
    description character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    source_plan_code character varying(100),
    source_tier_rank integer,
    active_tier_rank_at_completion integer,
    renewal_attempt_key character varying(255),
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX ux_wallet_statement_renewal_attempt ON subscription.wallet_statements USING btree (renewal_attempt_key) WHERE (renewal_attempt_key IS NOT NULL);

-- Table Definition
CREATE TABLE aiexecutor.ai_cost_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    code character varying(100) NOT NULL,
    version bigint NOT NULL,
    name character varying(150) NOT NULL,
    enforcement_mode character varying(30) NOT NULL,
    period_days integer NOT NULL,
    hard_cost_micro_vnd bigint,
    compact_threshold_bps integer DEFAULT 8000 NOT NULL,
    free_route_threshold_bps integer DEFAULT 9200 NOT NULL,
    unknown_hold_minutes integer DEFAULT 1440 NOT NULL,
    max_unknown_paid_requests_per_day integer DEFAULT 2 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT ai_cost_policies_check CHECK ((((enforcement_mode)::text <> 'strict'::text) OR (hard_cost_micro_vnd IS NOT NULL))),
    CONSTRAINT ai_cost_policies_check1 CHECK ((compact_threshold_bps <= free_route_threshold_bps)),
    CONSTRAINT ai_cost_policies_compact_threshold_bps_check CHECK (((compact_threshold_bps >= 0) AND (compact_threshold_bps <= 10000))),
    CONSTRAINT ai_cost_policies_enforcement_mode_check CHECK (((enforcement_mode)::text = ANY ((ARRAY['strict'::character varying, 'observe_only'::character varying, 'free_only'::character varying])::text[]))),
    CONSTRAINT ai_cost_policies_free_route_threshold_bps_check CHECK (((free_route_threshold_bps >= 0) AND (free_route_threshold_bps <= 10000))),
    CONSTRAINT ai_cost_policies_max_unknown_paid_requests_per_day_check CHECK ((max_unknown_paid_requests_per_day >= 0)),
    CONSTRAINT ai_cost_policies_period_days_check CHECK ((period_days > 0)),
    CONSTRAINT ai_cost_policies_unknown_hold_minutes_check CHECK ((unknown_hold_minutes > 0))
);

CREATE UNIQUE INDEX ai_cost_policies_code_version_key ON aiexecutor.ai_cost_policies USING btree (code, version);

-- Table Definition
CREATE TABLE aiexecutor.ai_cost_policy_operations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    policy_id uuid NOT NULL,
    operation character varying(30) NOT NULL,
    normal_route character varying(50) NOT NULL,
    reduced_route character varying(50) NOT NULL,
    free_route character varying(50) NOT NULL,
    normal_max_input_tokens integer NOT NULL,
    normal_max_output_tokens integer NOT NULL,
    reduced_max_input_tokens integer NOT NULL,
    reduced_max_output_tokens integer NOT NULL,
    max_paid_attempts_per_day integer NOT NULL,
    paid_fallback_enabled boolean DEFAULT false NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT ai_cost_policy_operations_max_paid_attempts_per_day_check CHECK ((max_paid_attempts_per_day > 0)),
    CONSTRAINT ai_cost_policy_operations_normal_max_input_tokens_check CHECK ((normal_max_input_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_normal_max_output_tokens_check CHECK ((normal_max_output_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_operation_check CHECK (((operation)::text = ANY ((ARRAY['chat'::character varying, 'outfit'::character varying, 'summary'::character varying, 'rewriter'::character varying])::text[]))),
    CONSTRAINT ai_cost_policy_operations_reduced_max_input_tokens_check CHECK ((reduced_max_input_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_reduced_max_output_tokens_check CHECK ((reduced_max_output_tokens > 0))
);

CREATE UNIQUE INDEX ai_cost_policy_operations_policy_id_operation_key ON aiexecutor.ai_cost_policy_operations USING btree (policy_id, operation);

-- Table Definition
CREATE TABLE aiexecutor.ai_model_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    provider character varying(50) NOT NULL,
    model character varying(100) NOT NULL,
    input_usd_per_million numeric(20,8) NOT NULL,
    output_usd_per_million numeric(20,8) NOT NULL,
    currency character varying(10) DEFAULT 'VND'::character varying NOT NULL,
    usd_to_vnd numeric(20,4) NOT NULL,
    version character varying(50) DEFAULT 'v1'::character varying NOT NULL,
    effective_from timestamp with time zone DEFAULT now() NOT NULL,
    effective_to timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by uuid,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX uq_provider_model_active ON aiexecutor.ai_model_configs USING btree (provider, model, effective_from);

-- Table Definition
CREATE TABLE aiexecutor.ai_policy_grants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    plan_id uuid NOT NULL,
    plan_code character varying(100) NOT NULL,
    tier_rank integer NOT NULL,
    policy_snapshot jsonb NOT NULL,
    effective_from timestamp with time zone NOT NULL,
    effective_to timestamp with time zone,
    status character varying(20) NOT NULL,
    source_event_id uuid,
    source_deposit_transaction_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT user_ai_policy_grants_check CHECK (((effective_to IS NULL) OR (effective_to > effective_from))),
    CONSTRAINT user_ai_policy_grants_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'future'::character varying, 'closed'::character varying])::text[])))
);

CREATE INDEX ix_user_ai_policy_grants_active ON aiexecutor.ai_policy_grants USING btree (user_id, effective_from, effective_to);

-- Table Definition
CREATE TABLE aiexecutor.ai_usage_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_id uuid NOT NULL,
    ledger_id uuid,
    user_id uuid,
    operation character varying(30) NOT NULL,
    logical_route character varying(50) NOT NULL,
    provider character varying(50),
    model character varying(150),
    input_usd_per_million numeric(18,8),
    output_usd_per_million numeric(18,8),
    usd_to_vnd numeric(18,4),
    prompt_tokens bigint DEFAULT 0 NOT NULL,
    output_tokens bigint DEFAULT 0 NOT NULL,
    thinking_tokens bigint DEFAULT 0 NOT NULL,
    reserved_cost_micro_vnd bigint DEFAULT 0 NOT NULL,
    actual_cost_micro_vnd bigint DEFAULT 0 NOT NULL,
    estimated_max_cost_micro_vnd bigint DEFAULT 0 NOT NULL,
    status character varying(30) NOT NULL,
    finish_reason character varying(100),
    error_code character varying(100),
    sent_at timestamp with time zone,
    completed_at timestamp with time zone,
    unknown_expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    estimated_prompt_tokens bigint,
    token_estimation_method character varying(30),
    token_count_latency_ms bigint,
    pricing_config_id uuid,
    PRIMARY KEY (id),
    CONSTRAINT ai_usage_events_estimated_prompt_tokens_check CHECK (((estimated_prompt_tokens IS NULL) OR (estimated_prompt_tokens >= 0))),
    CONSTRAINT ai_usage_events_status_check CHECK (((status)::text = ANY ((ARRAY['reserved'::character varying, 'in_flight'::character varying, 'confirmed'::character varying, 'released'::character varying, 'unknown_usage'::character varying, 'expired_unverified'::character varying])::text[]))),
    CONSTRAINT ai_usage_events_token_count_latency_ms_check CHECK (((token_count_latency_ms IS NULL) OR (token_count_latency_ms >= 0))),
    CONSTRAINT ai_usage_events_token_estimation_method_check CHECK (((token_estimation_method IS NULL) OR ((token_estimation_method)::text = ANY ((ARRAY['local'::character varying, 'provider_count'::character varying])::text[]))))
);

CREATE UNIQUE INDEX ai_usage_events_request_id_key ON aiexecutor.ai_usage_events USING btree (request_id);
CREATE INDEX ix_ai_usage_events_unknown ON aiexecutor.ai_usage_events USING btree (status, unknown_expires_at) WHERE ((status)::text = ANY ((ARRAY['reserved'::character varying, 'unknown_usage'::character varying])::text[]));
CREATE INDEX ix_ai_usage_events_user_day ON aiexecutor.ai_usage_events USING btree (user_id, created_at, status);

-- Table Definition
CREATE TABLE aiexecutor.ai_usage_period_ledgers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    grant_id uuid NOT NULL,
    user_id uuid NOT NULL,
    period_index integer NOT NULL,
    period_start timestamp with time zone NOT NULL,
    period_end timestamp with time zone NOT NULL,
    paid_input_tokens bigint DEFAULT 0 NOT NULL,
    paid_output_tokens bigint DEFAULT 0 NOT NULL,
    free_input_tokens bigint DEFAULT 0 NOT NULL,
    free_output_tokens bigint DEFAULT 0 NOT NULL,
    actual_cost_micro_vnd bigint DEFAULT 0 NOT NULL,
    reserved_cost_micro_vnd bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT ai_usage_period_ledgers_period_index_check CHECK ((period_index >= 0))
);

CREATE UNIQUE INDEX ai_usage_period_ledgers_grant_id_period_index_key ON aiexecutor.ai_usage_period_ledgers USING btree (grant_id, period_index);

-- Table Definition
CREATE TABLE dashboard.dashboard_admin_daily_snapshots (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    snapshot_date date NOT NULL,
    total_users integer DEFAULT 0 NOT NULL,
    new_users integer DEFAULT 0 NOT NULL,
    active_users_dau integer DEFAULT 0 NOT NULL,
    total_brands integer DEFAULT 0 NOT NULL,
    pending_review_brands integer DEFAULT 0 NOT NULL,
    ai_requests_count bigint DEFAULT 0 NOT NULL,
    ai_error_count bigint DEFAULT 0 NOT NULL,
    ai_paid_tokens bigint DEFAULT 0 NOT NULL,
    ai_actual_cost_micro_vnd bigint DEFAULT 0 NOT NULL,
    subscription_revenue_vnd numeric(12,2) DEFAULT 0.00 NOT NULL,
    net_ai_margin_vnd numeric(12,2) DEFAULT 0.00 NOT NULL,
    payos_transaction_count integer DEFAULT 0 NOT NULL,
    payos_success_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX dashboard_admin_daily_snapshots_snapshot_date_key ON dashboard.dashboard_admin_daily_snapshots USING btree (snapshot_date);
CREATE INDEX idx_admin_snapshots_date ON dashboard.dashboard_admin_daily_snapshots USING btree (snapshot_date DESC);

-- Table Definition
CREATE TABLE dashboard.dashboard_brand_daily_snapshots (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    snapshot_date date NOT NULL,
    total_members integer DEFAULT 0 NOT NULL,
    points_issued integer DEFAULT 0 NOT NULL,
    points_redeemed integer DEFAULT 0 NOT NULL,
    points_expiring_30d integer DEFAULT 0 NOT NULL,
    points_liability_value_vnd numeric(12,2) DEFAULT 0.00 NOT NULL,
    active_vouchers integer DEFAULT 0 NOT NULL,
    vouchers_redeemed integer DEFAULT 0 NOT NULL,
    digital_sample_votes_count integer DEFAULT 0 NOT NULL,
    digital_sample_avg_rating numeric(3,2) DEFAULT 0.00 NOT NULL,
    open_cs_tickets integer DEFAULT 0 NOT NULL,
    cs_first_response_avg_seconds integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX idx_brand_snapshots_brand_date ON dashboard.dashboard_brand_daily_snapshots USING btree (brand_id, snapshot_date DESC);
CREATE UNIQUE INDEX uq_brand_daily_snapshot ON dashboard.dashboard_brand_daily_snapshots USING btree (brand_id, snapshot_date);

-- Table Definition
CREATE TABLE dashboard.dashboard_user_wardrobe_summaries (
    user_id uuid NOT NULL,
    total_items integer DEFAULT 0 NOT NULL,
    total_wardrobe_value_vnd numeric(12,2) DEFAULT 0.00 NOT NULL,
    average_cost_per_wear_vnd numeric(12,2) DEFAULT 0.00 NOT NULL,
    underutilized_items_count integer DEFAULT 0 NOT NULL,
    underutilized_items jsonb DEFAULT '[]'::jsonb NOT NULL,
    ai_quota_remaining integer DEFAULT 0 NOT NULL,
    expiring_vouchers_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (user_id)
);

CREATE INDEX idx_user_wardrobe_summaries_updated ON dashboard.dashboard_user_wardrobe_summaries USING btree (updated_at DESC);

-- Table Definition
CREATE TABLE shared.outbox_records (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    topic character varying(255) NOT NULL,
    payload jsonb NOT NULL,
    headers jsonb DEFAULT '{}'::jsonb,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    published_at timestamp with time zone,
    attempts integer DEFAULT 0 NOT NULL,
    last_error text,
    PRIMARY KEY (id)
);

CREATE INDEX idx_outbox_records_created_at ON shared.outbox_records USING btree (created_at);
CREATE INDEX idx_outbox_records_status ON shared.outbox_records USING btree (status);

-- Foreign Key Constraints (giữ; 9 FK cross-module đã cắt, enforce qua contract)
ALTER TABLE aiexecutor.ai_cost_policy_operations
    ADD CONSTRAINT ai_cost_policy_operations_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES ai_cost_policies(id) ON DELETE CASCADEALTER TABLE aiexecutor.ai_model_configs
    ADD CONSTRAINT ai_model_configs_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id)ALTER TABLE aiexecutor.ai_policy_grants
    ADD CONSTRAINT user_ai_policy_grants_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES ai_cost_policies(id) ON DELETE RESTRICT,
    ADD CONSTRAINT user_ai_policy_grants_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE aiexecutor.ai_usage_events
    ADD CONSTRAINT ai_usage_events_ledger_id_fkey FOREIGN KEY (ledger_id) REFERENCES ai_usage_period_ledgers(id) ON DELETE CASCADE,
    ADD CONSTRAINT ai_usage_events_pricing_config_id_fkey FOREIGN KEY (pricing_config_id) REFERENCES ai_model_configs(id),
    ADD CONSTRAINT ai_usage_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE aiexecutor.ai_usage_period_ledgers
    ADD CONSTRAINT ai_usage_period_ledgers_grant_id_fkey FOREIGN KEY (grant_id) REFERENCES ai_policy_grants(id) ON DELETE CASCADE,
    ADD CONSTRAINT ai_usage_period_ledgers_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE brand.benefit_redemptions
    ADD CONSTRAINT benefit_redemptions_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand_customers(id) ON DELETE CASCADE,
    ADD CONSTRAINT benefit_redemptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT benefit_redemptions_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE,
    ADD CONSTRAINT benefit_redemptions_benefit_id_fkey FOREIGN KEY (benefit_id) REFERENCES brand_benefits(id) ON DELETE CASCADEALTER TABLE brand.brand_benefits
    ADD CONSTRAINT brand_benefits_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE,
    ADD CONSTRAINT brand_benefits_required_tier_id_fkey FOREIGN KEY (required_tier_id) REFERENCES loyalty_tiers(id) ON DELETE SET NULLALTER TABLE brand.brand_conversation_messages
    ADD CONSTRAINT brand_conversation_messages_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES brand_conversations(id) ON DELETE CASCADE,
    ADD CONSTRAINT brand_conversation_messages_sender_user_id_fkey FOREIGN KEY (sender_user_id) REFERENCES users(id) ON DELETE SET NULLALTER TABLE brand.brand_conversations
    ADD CONSTRAINT brand_conversations_closed_by_user_id_fkey FOREIGN KEY (closed_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT brand_conversations_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    ADD CONSTRAINT brand_conversations_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADEALTER TABLE brand.brand_customer_claims
    ADD CONSTRAINT brand_customer_claims_revoked_by_user_id_fkey FOREIGN KEY (revoked_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT brand_customer_claims_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand_customers(id) ON DELETE CASCADEALTER TABLE brand.brand_customers
    ADD CONSTRAINT brand_customers_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE,
    ADD CONSTRAINT brand_customers_created_by_member_id_fkey FOREIGN KEY (created_by_member_id) REFERENCES brand_members(id) ON DELETE SET NULL,
    ADD CONSTRAINT brand_customers_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULLALTER TABLE brand.brand_items
    ADD CONSTRAINT brand_items_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADEALTER TABLE brand.brand_members
    ADD CONSTRAINT brand_members_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE,
    ADD CONSTRAINT brand_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE brand.brands
    ADD CONSTRAINT brands_created_by_user_id_fkey FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT brands_approved_by_user_id_fkey FOREIGN KEY (approved_by_user_id) REFERENCES users(id) ON DELETE SET NULLALTER TABLE brand.digital_sample_responses
    ADD CONSTRAINT digital_sample_responses_brand_item_id_fkey FOREIGN KEY (brand_item_id) REFERENCES brand_items(id) ON DELETE CASCADE,
    ADD CONSTRAINT digital_sample_responses_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE brand.loyalty_accounts
    ADD CONSTRAINT loyalty_accounts_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand_customers(id) ON DELETE CASCADE,
    ADD CONSTRAINT loyalty_accounts_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE,
    ADD CONSTRAINT loyalty_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT loyalty_accounts_current_tier_id_fkey FOREIGN KEY (current_tier_id) REFERENCES loyalty_tiers(id) ON DELETE SET NULLALTER TABLE brand.loyalty_point_lots
    ADD CONSTRAINT loyalty_point_lots_earn_transaction_id_fkey FOREIGN KEY (earn_transaction_id) REFERENCES loyalty_point_transactions(id) ON DELETE RESTRICT,
    ADD CONSTRAINT loyalty_point_lots_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT loyalty_point_lots_loyalty_account_id_fkey FOREIGN KEY (loyalty_account_id) REFERENCES loyalty_accounts(id) ON DELETE CASCADE,
    ADD CONSTRAINT loyalty_point_lots_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand_customers(id) ON DELETE CASCADE,
    ADD CONSTRAINT loyalty_point_lots_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADEALTER TABLE brand.loyalty_point_transactions
    ADD CONSTRAINT loyalty_point_transactions_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand_customers(id) ON DELETE CASCADE,
    ADD CONSTRAINT loyalty_point_transactions_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE,
    ADD CONSTRAINT loyalty_point_transactions_created_by_user_id_fkey FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT loyalty_point_transactions_loyalty_account_id_fkey FOREIGN KEY (loyalty_account_id) REFERENCES loyalty_accounts(id) ON DELETE CASCADE,
    ADD CONSTRAINT loyalty_point_transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULLALTER TABLE brand.loyalty_programs
    ADD CONSTRAINT loyalty_programs_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADEALTER TABLE brand.loyalty_tiers
    ADD CONSTRAINT loyalty_tiers_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADEALTER TABLE dashboard.dashboard_user_wardrobe_summaries
    ADD CONSTRAINT dashboard_user_wardrobe_summaries_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE fashion.conversational_contexts
    ADD CONSTRAINT conversational_contexts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE fashion.fashion_items
    ADD CONSTRAINT fashion_items_category_id_fkey FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICTALTER TABLE fashion.messages
    ADD CONSTRAINT messages_context_id_fkey FOREIGN KEY (context_id) REFERENCES conversational_contexts(id) ON DELETE CASCADEALTER TABLE identity.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE identity.user_style_profiles
    ADD CONSTRAINT user_style_profiles_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE subscription.deposit_transactions
    ADD CONSTRAINT deposit_transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    ADD CONSTRAINT deposit_transactions_subscription_plan_id_fkey FOREIGN KEY (subscription_plan_id) REFERENCES subscription_plans(id) ON DELETE SET NULLALTER TABLE subscription.subscription_renewal_attempts
    ADD CONSTRAINT subscription_renewal_attempts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    ADD CONSTRAINT subscription_renewal_attempts_expected_plan_id_fkey FOREIGN KEY (expected_plan_id) REFERENCES subscription_plans(id)ALTER TABLE subscription.user_daily_quotas
    ADD CONSTRAINT user_daily_quotas_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE subscription.user_subscription_events
    ADD CONSTRAINT user_subscription_events_source_deposit_transaction_id_fkey FOREIGN KEY (source_deposit_transaction_id) REFERENCES deposit_transactions(id),
    ADD CONSTRAINT user_subscription_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    ADD CONSTRAINT user_subscription_events_actor_admin_id_fkey FOREIGN KEY (actor_admin_id) REFERENCES users(id)ALTER TABLE subscription.user_subscriptions
    ADD CONSTRAINT user_subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    ADD CONSTRAINT user_subscriptions_fallback_plan_id_fkey FOREIGN KEY (fallback_plan_id) REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    ADD CONSTRAINT user_subscriptions_subscription_plan_id_fkey FOREIGN KEY (subscription_plan_id) REFERENCES subscription_plans(id) ON DELETE RESTRICTALTER TABLE subscription.user_wallets
    ADD CONSTRAINT user_wallets_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE subscription.wallet_statements
    ADD CONSTRAINT wallet_statements_reference_id_fkey FOREIGN KEY (reference_id) REFERENCES deposit_transactions(id) ON DELETE SET NULL,
    ADD CONSTRAINT wallet_statements_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE wardrobe.outfit_items
    ADD CONSTRAINT outfit_items_outfit_id_fkey FOREIGN KEY (outfit_id) REFERENCES outfits(id) ON DELETE CASCADEALTER TABLE wardrobe.outfits
    ADD CONSTRAINT outfits_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADEALTER TABLE wardrobe.wardrobe_items
    ADD CONSTRAINT wardrobe_items_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

