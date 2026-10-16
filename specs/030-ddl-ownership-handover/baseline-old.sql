--
-- PostgreSQL database dump
--

\restrict c1h3x7FcUsZ0gAx1RSfnhfntobo5hLXXVY2w5hqWg9FIvKQ2KolqhfIGvB0d4Xs

-- Dumped from database version 17.10 (Debian 17.10-1.pgdg12+1)
-- Dumped by pg_dump version 17.10 (Debian 17.10-1.pgdg12+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: aiexecutor; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA aiexecutor;


--
-- Name: brand; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA brand;


--
-- Name: community; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA community;


--
-- Name: dashboard; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA dashboard;


--
-- Name: fashion; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA fashion;


--
-- Name: identity; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA identity;


--
-- Name: shared; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA shared;


--
-- Name: subscription; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA subscription;


--
-- Name: wardrobe; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA wardrobe;


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: unaccent; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public;


--
-- Name: EXTENSION unaccent; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION unaccent IS 'text search dictionary that removes accents';


--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;


--
-- Name: EXTENSION vector; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION vector IS 'vector data type and ivfflat and hnsw access methods';


--
-- Name: immutable_unaccent(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.immutable_unaccent(text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $_$
  SELECT public.unaccent('public.unaccent'::regdictionary, $1)
$_$;


--
-- Name: campaigns_guard_immutable_budget(); Type: FUNCTION; Schema: subscription; Owner: -
--

CREATE FUNCTION subscription.campaigns_guard_immutable_budget() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF (NEW.quota, NEW.reserve, NEW.plan_slug)
           IS DISTINCT FROM (OLD.quota, OLD.reserve, OLD.plan_slug)
       AND EXISTS (
           SELECT 1 FROM subscription.campaign_claims
           WHERE campaign_code = OLD.code
       )
    THEN
        RAISE EXCEPTION
            'campaign % has grants; quota, reserve and plan_slug are immutable (026 FR-020/FR-021)',
            OLD.code
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: ai_cost_policies; Type: TABLE; Schema: aiexecutor; Owner: -
--

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
    CONSTRAINT ai_cost_policies_check CHECK ((((enforcement_mode)::text <> 'strict'::text) OR (hard_cost_micro_vnd IS NOT NULL))),
    CONSTRAINT ai_cost_policies_check1 CHECK ((compact_threshold_bps <= free_route_threshold_bps)),
    CONSTRAINT ai_cost_policies_compact_threshold_bps_check CHECK (((compact_threshold_bps >= 0) AND (compact_threshold_bps <= 10000))),
    CONSTRAINT ai_cost_policies_enforcement_mode_check CHECK (((enforcement_mode)::text = ANY (ARRAY[('strict'::character varying)::text, ('observe_only'::character varying)::text, ('free_only'::character varying)::text]))),
    CONSTRAINT ai_cost_policies_free_route_threshold_bps_check CHECK (((free_route_threshold_bps >= 0) AND (free_route_threshold_bps <= 10000))),
    CONSTRAINT ai_cost_policies_max_unknown_paid_requests_per_day_check CHECK ((max_unknown_paid_requests_per_day >= 0)),
    CONSTRAINT ai_cost_policies_period_days_check CHECK ((period_days > 0)),
    CONSTRAINT ai_cost_policies_unknown_hold_minutes_check CHECK ((unknown_hold_minutes > 0))
);


--
-- Name: ai_cost_policy_audit_logs; Type: TABLE; Schema: aiexecutor; Owner: -
--

CREATE TABLE aiexecutor.ai_cost_policy_audit_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    actor_user_id uuid NOT NULL,
    changed_at timestamp with time zone DEFAULT now() NOT NULL,
    request_ip character varying(64),
    policy_code character varying(100) NOT NULL,
    operation character varying(30) NOT NULL,
    field_name character varying(60) NOT NULL,
    old_value character varying(255),
    new_value character varying(255),
    action character varying(60) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: ai_cost_policy_operations; Type: TABLE; Schema: aiexecutor; Owner: -
--

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
    max_paid_attempts_per_day integer,
    paid_fallback_enabled boolean DEFAULT false NOT NULL,
    is_enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ai_cost_policy_operations_max_paid_attempts_per_day_check CHECK ((max_paid_attempts_per_day > 0)),
    CONSTRAINT ai_cost_policy_operations_normal_max_input_tokens_check CHECK ((normal_max_input_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_normal_max_output_tokens_check CHECK ((normal_max_output_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_operation_check CHECK (((operation)::text = ANY (ARRAY[('chat'::character varying)::text, ('outfit'::character varying)::text, ('summary'::character varying)::text, ('rewriter'::character varying)::text]))),
    CONSTRAINT ai_cost_policy_operations_reduced_max_input_tokens_check CHECK ((reduced_max_input_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_reduced_max_output_tokens_check CHECK ((reduced_max_output_tokens > 0))
);


--
-- Name: ai_model_configs; Type: TABLE; Schema: aiexecutor; Owner: -
--

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
    created_by uuid
);


--
-- Name: ai_policy_grants; Type: TABLE; Schema: aiexecutor; Owner: -
--

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
    CONSTRAINT user_ai_policy_grants_check CHECK (((effective_to IS NULL) OR (effective_to > effective_from))),
    CONSTRAINT user_ai_policy_grants_status_check CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('future'::character varying)::text, ('closed'::character varying)::text])))
);


--
-- Name: ai_usage_events; Type: TABLE; Schema: aiexecutor; Owner: -
--

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
    CONSTRAINT ai_usage_events_estimated_prompt_tokens_check CHECK (((estimated_prompt_tokens IS NULL) OR (estimated_prompt_tokens >= 0))),
    CONSTRAINT ai_usage_events_status_check CHECK (((status)::text = ANY (ARRAY[('reserved'::character varying)::text, ('in_flight'::character varying)::text, ('confirmed'::character varying)::text, ('released'::character varying)::text, ('unknown_usage'::character varying)::text, ('expired_unverified'::character varying)::text]))),
    CONSTRAINT ai_usage_events_token_count_latency_ms_check CHECK (((token_count_latency_ms IS NULL) OR (token_count_latency_ms >= 0))),
    CONSTRAINT ai_usage_events_token_estimation_method_check CHECK (((token_estimation_method IS NULL) OR ((token_estimation_method)::text = ANY (ARRAY[('local'::character varying)::text, ('provider_count'::character varying)::text]))))
);


--
-- Name: ai_usage_period_ledgers; Type: TABLE; Schema: aiexecutor; Owner: -
--

CREATE TABLE aiexecutor.ai_usage_period_ledgers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    grant_id uuid,
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
    CONSTRAINT ai_usage_period_ledgers_period_index_check CHECK ((period_index >= 0))
);


--
-- Name: benefit_redemptions; Type: TABLE; Schema: brand; Owner: -
--

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
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: brand_benefits; Type: TABLE; Schema: brand; Owner: -
--

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
    CONSTRAINT chk_brand_benefits_status CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('inactive'::character varying)::text, ('archived'::character varying)::text])))
);


--
-- Name: brand_conversation_messages; Type: TABLE; Schema: brand; Owner: -
--

CREATE TABLE brand.brand_conversation_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    conversation_id uuid NOT NULL,
    sender_user_id uuid,
    sender_role character varying(50) NOT NULL,
    message text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: brand_conversations; Type: TABLE; Schema: brand; Owner: -
--

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
    closed_by_user_id uuid
);


--
-- Name: brand_customer_claims; Type: TABLE; Schema: brand; Owner: -
--

CREATE TABLE brand.brand_customer_claims (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_customer_id uuid NOT NULL,
    claim_token_hash character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone,
    revoked_by_user_id uuid,
    revoked_reason character varying(255)
);


--
-- Name: brand_customers; Type: TABLE; Schema: brand; Owner: -
--

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
    CONSTRAINT chk_brand_customers_status CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('blocked'::character varying)::text, ('left'::character varying)::text])))
);


--
-- Name: brand_items; Type: TABLE; Schema: brand; Owner: -
--

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
    CONSTRAINT chk_brand_items_status CHECK (((status)::text = ANY (ARRAY[('draft'::character varying)::text, ('active'::character varying)::text, ('archived'::character varying)::text, ('failed'::character varying)::text])))
);


--
-- Name: brand_members; Type: TABLE; Schema: brand; Owner: -
--

CREATE TABLE brand.brand_members (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role character varying(50) NOT NULL,
    status character varying(50) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_brand_members_role CHECK (((role)::text = ANY (ARRAY[('owner'::character varying)::text, ('staff'::character varying)::text]))),
    CONSTRAINT chk_brand_members_status CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('invited'::character varying)::text, ('disabled'::character varying)::text])))
);


--
-- Name: brands; Type: TABLE; Schema: brand; Owner: -
--

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
    CONSTRAINT chk_brands_status CHECK (((status)::text = ANY (ARRAY[('pending_review'::character varying)::text, ('active'::character varying)::text, ('suspended'::character varying)::text, ('archived'::character varying)::text])))
);


--
-- Name: digital_sample_responses; Type: TABLE; Schema: brand; Owner: -
--

CREATE TABLE brand.digital_sample_responses (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_item_id uuid NOT NULL,
    user_id uuid NOT NULL,
    outfit_id uuid,
    vote_type character varying(50),
    rating integer,
    feedback_text text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: loyalty_accounts; Type: TABLE; Schema: brand; Owner: -
--

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
    CONSTRAINT chk_loyalty_accounts_current_points CHECK ((current_points >= 0)),
    CONSTRAINT chk_loyalty_accounts_lifetime_points CHECK ((lifetime_points >= 0)),
    CONSTRAINT chk_loyalty_accounts_total_spend CHECK ((total_spend >= (0)::numeric))
);


--
-- Name: loyalty_point_lots; Type: TABLE; Schema: brand; Owner: -
--

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
    CONSTRAINT chk_loyalty_point_lots_earned_positive CHECK ((earned_points > 0)),
    CONSTRAINT chk_loyalty_point_lots_remaining_lte_earned CHECK ((remaining_points <= earned_points)),
    CONSTRAINT chk_loyalty_point_lots_remaining_non_negative CHECK ((remaining_points >= 0)),
    CONSTRAINT chk_loyalty_point_lots_status CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('consumed'::character varying)::text, ('expired'::character varying)::text])))
);


--
-- Name: loyalty_point_transactions; Type: TABLE; Schema: brand; Owner: -
--

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
    CONSTRAINT chk_loyalty_point_transactions_balance_after CHECK ((balance_after >= 0)),
    CONSTRAINT chk_loyalty_point_transactions_spend_amount CHECK (((spend_amount IS NULL) OR (spend_amount >= (0)::numeric)))
);


--
-- Name: loyalty_programs; Type: TABLE; Schema: brand; Owner: -
--

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
    CONSTRAINT chk_loyalty_programs_amount_per_point CHECK ((amount_per_point > (0)::numeric)),
    CONSTRAINT chk_loyalty_programs_point_expiry_days CHECK (((point_expiry_days IS NULL) OR (point_expiry_days > 0)))
);


--
-- Name: loyalty_tiers; Type: TABLE; Schema: brand; Owner: -
--

CREATE TABLE brand.loyalty_tiers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    rank integer NOT NULL,
    min_total_spend numeric(12,2) DEFAULT 0 NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_loyalty_tiers_min_total_spend CHECK ((min_total_spend >= (0)::numeric))
);


--
-- Name: follows; Type: TABLE; Schema: community; Owner: -
--

CREATE TABLE community.follows (
    follower_id uuid NOT NULL,
    followee_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_follows_no_self CHECK ((follower_id <> followee_id))
);


--
-- Name: post_comments; Type: TABLE; Schema: community; Owner: -
--

CREATE TABLE community.post_comments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    post_id uuid NOT NULL,
    user_id uuid NOT NULL,
    parent_comment_id uuid,
    root_comment_id uuid,
    content text NOT NULL,
    status character varying(20) DEFAULT 'published'::character varying NOT NULL,
    moderated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_post_comments_depth CHECK (((parent_comment_id IS NULL) OR ((root_comment_id IS NOT NULL) AND (parent_comment_id <> id)))),
    CONSTRAINT chk_post_comments_status CHECK (((status)::text = ANY (ARRAY[('published'::character varying)::text, ('hidden'::character varying)::text, ('deleted'::character varying)::text])))
);


--
-- Name: post_likes; Type: TABLE; Schema: community; Owner: -
--

CREATE TABLE community.post_likes (
    post_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: post_media; Type: TABLE; Schema: community; Owner: -
--

CREATE TABLE community.post_media (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    post_id uuid NOT NULL,
    media_type character varying(10) NOT NULL,
    media_url character varying(500) NOT NULL,
    public_id character varying(255),
    sort_order smallint DEFAULT 0 NOT NULL,
    CONSTRAINT chk_post_media_type CHECK (((media_type)::text = ANY (ARRAY[('image'::character varying)::text, ('video'::character varying)::text])))
);


--
-- Name: posts; Type: TABLE; Schema: community; Owner: -
--

CREATE TABLE community.posts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    public_id character varying(32) NOT NULL,
    user_id uuid NOT NULL,
    post_type character varying(20) NOT NULL,
    title character varying(150),
    content text DEFAULT ''::text NOT NULL,
    outfit_id uuid,
    like_count integer DEFAULT 0 NOT NULL,
    comment_count integer DEFAULT 0 NOT NULL,
    status character varying(20) DEFAULT 'published'::character varying NOT NULL,
    moderated_by uuid,
    hidden_reason character varying(30),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_posts_hidden_reason CHECK (((hidden_reason IS NULL) OR ((hidden_reason)::text = ANY (ARRAY[('admin'::character varying)::text, ('author_disabled'::character varying)::text])))),
    CONSTRAINT chk_posts_outfit_ref CHECK (((((post_type)::text = 'outfit'::text) AND (outfit_id IS NOT NULL)) OR (((post_type)::text = 'media'::text) AND (outfit_id IS NULL)))),
    CONSTRAINT chk_posts_post_type CHECK (((post_type)::text = ANY (ARRAY[('outfit'::character varying)::text, ('media'::character varying)::text]))),
    CONSTRAINT chk_posts_status CHECK (((status)::text = ANY (ARRAY[('published'::character varying)::text, ('hidden'::character varying)::text, ('deleted'::character varying)::text])))
);


--
-- Name: user_follow_stats; Type: TABLE; Schema: community; Owner: -
--

CREATE TABLE community.user_follow_stats (
    user_id uuid NOT NULL,
    follower_count integer DEFAULT 0 NOT NULL,
    following_count integer DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: dashboard_admin_daily_snapshots; Type: TABLE; Schema: dashboard; Owner: -
--

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
    payos_pending_count integer DEFAULT 0 NOT NULL,
    payos_failed_count integer DEFAULT 0 NOT NULL,
    payos_voided_count integer DEFAULT 0 NOT NULL
);


--
-- Name: dashboard_brand_daily_snapshots; Type: TABLE; Schema: dashboard; Owner: -
--

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
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: dashboard_user_wardrobe_summaries; Type: TABLE; Schema: dashboard; Owner: -
--

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
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: categories; Type: TABLE; Schema: fashion; Owner: -
--

CREATE TABLE fashion.categories (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(100) NOT NULL,
    slug character varying(100) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL
);


--
-- Name: conversational_contexts; Type: TABLE; Schema: fashion; Owner: -
--

CREATE TABLE fashion.conversational_contexts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    context_summary text DEFAULT ''::text,
    is_archived boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: fashion_items; Type: TABLE; Schema: fashion; Owner: -
--

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
    embedding public.vector(768),
    processing_retry_count integer DEFAULT 0 NOT NULL,
    processing_version integer DEFAULT 0 NOT NULL,
    processing_started_at timestamp with time zone,
    last_processing_attempt_at timestamp with time zone,
    processing_error_reason text,
    review_reason character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: messages; Type: TABLE; Schema: fashion; Owner: -
--

CREATE TABLE fashion.messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    context_id uuid NOT NULL,
    sender character varying(50) NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    is_summarized boolean DEFAULT false NOT NULL
);


--
-- Name: refresh_tokens; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.refresh_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    token character varying(500) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    is_revoked boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_name character varying(255),
    device_ip character varying(45),
    user_agent character varying(512),
    last_active_at timestamp with time zone
);


--
-- Name: user_activity_events; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.user_activity_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    activity_type character varying(20) NOT NULL,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    username character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255),
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
    provider character varying(50) DEFAULT 'local'::character varying NOT NULL,
    provider_id character varying(255),
    email_verified boolean DEFAULT false NOT NULL,
    created_by_admin boolean DEFAULT false NOT NULL
);


--
-- Name: COLUMN users.created_by_admin; Type: COMMENT; Schema: identity; Owner: -
--

COMMENT ON COLUMN identity.users.created_by_admin IS 'True only when the account was created through the administrator endpoint. Set at exactly one write site; never written from configuration.';


--
-- Name: goose_db_version; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.goose_db_version (
    id integer NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: goose_db_version_commerce; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.goose_db_version_commerce (
    id integer NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: goose_db_version_commerce_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.goose_db_version_commerce ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.goose_db_version_commerce_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: goose_db_version_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.goose_db_version ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.goose_db_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: admin_audit_logs; Type: TABLE; Schema: shared; Owner: -
--

CREATE TABLE shared.admin_audit_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    actor_user_id uuid NOT NULL,
    action character varying(100) NOT NULL,
    target_type character varying(50) NOT NULL,
    target_id uuid,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    request_ip character varying(45),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: outbox_records; Type: TABLE; Schema: shared; Owner: -
--

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
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: campaign_claims; Type: TABLE; Schema: subscription; Owner: -
--

CREATE TABLE subscription.campaign_claims (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    campaign_code character varying(64) NOT NULL,
    plan_slug character varying(100) NOT NULL,
    plan_id uuid NOT NULL,
    registered_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    granted_at timestamp with time zone DEFAULT now() NOT NULL,
    grant_bucket character varying(16) NOT NULL,
    fills_main_quota boolean DEFAULT false NOT NULL,
    CONSTRAINT chk_campaign_claims_bucket CHECK (((grant_bucket)::text = ANY (ARRAY[('main'::character varying)::text, ('reserve'::character varying)::text])))
);


--
-- Name: campaign_sweep_runs; Type: TABLE; Schema: subscription; Owner: -
--

CREATE TABLE subscription.campaign_sweep_runs (
    campaign_code character varying(64) NOT NULL,
    last_sweep_at timestamp with time zone,
    degraded boolean DEFAULT false NOT NULL,
    degraded_reason text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE campaign_sweep_runs; Type: COMMENT; Schema: subscription; Owner: -
--

COMMENT ON TABLE subscription.campaign_sweep_runs IS 'One row per campaign: the health of the FR-079 catch-up sweep, read by the operator look-up (FR-065) from every replica.';


--
-- Name: campaigns; Type: TABLE; Schema: subscription; Owner: -
--

CREATE TABLE subscription.campaigns (
    code character varying(64) NOT NULL,
    plan_slug character varying(100) NOT NULL,
    quota integer NOT NULL,
    reserve integer DEFAULT 0 NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone,
    closed_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_campaigns_code_format CHECK (((code)::text ~ '^[a-z0-9][a-z0-9-]{2,63}$'::text)),
    CONSTRAINT chk_campaigns_plan_slug_not_blank CHECK ((btrim((plan_slug)::text) <> ''::text)),
    CONSTRAINT chk_campaigns_quota CHECK ((quota >= 1)),
    CONSTRAINT chk_campaigns_reserve CHECK ((reserve >= 0)),
    CONSTRAINT chk_campaigns_window CHECK (((ends_at IS NULL) OR (ends_at > starts_at)))
);


--
-- Name: deposit_transactions; Type: TABLE; Schema: subscription; Owner: -
--

CREATE TABLE subscription.deposit_transactions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    amount numeric(12,2) NOT NULL,
    currency character varying(10) DEFAULT 'VND'::character varying NOT NULL,
    status smallint DEFAULT 0 NOT NULL,
    transaction_type character varying(50) NOT NULL,
    subscription_plan_id uuid,
    order_code bigint NOT NULL,
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
    refund_reason character varying(255)
);


--
-- Name: deposit_transactions_order_code_seq; Type: SEQUENCE; Schema: subscription; Owner: -
--

CREATE SEQUENCE subscription.deposit_transactions_order_code_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: deposit_transactions_order_code_seq; Type: SEQUENCE OWNED BY; Schema: subscription; Owner: -
--

ALTER SEQUENCE subscription.deposit_transactions_order_code_seq OWNED BY subscription.deposit_transactions.order_code;


--
-- Name: provider_payment_events; Type: TABLE; Schema: subscription; Owner: -
--

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
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: provider_webhook_inbox; Type: TABLE; Schema: subscription; Owner: -
--

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
    CONSTRAINT provider_webhook_inbox_processing_status_check CHECK (((processing_status)::text = ANY (ARRAY[('received'::character varying)::text, ('processing'::character varying)::text, ('retry_required'::character varying)::text, ('processed'::character varying)::text, ('investigation_required'::character varying)::text, ('failed'::character varying)::text])))
);


--
-- Name: subscription_plans; Type: TABLE; Schema: subscription; Owner: -
--

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
    plan_kind smallint NOT NULL,
    tier_rank integer NOT NULL,
    pricing_version bigint DEFAULT 1 NOT NULL,
    ai_cost_policy_id uuid NOT NULL
);


--
-- Name: subscription_renewal_attempts; Type: TABLE; Schema: subscription; Owner: -
--

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
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_daily_quotas; Type: TABLE; Schema: subscription; Owner: -
--

CREATE TABLE subscription.user_daily_quotas (
    user_id uuid NOT NULL,
    outfit_recommend_count integer DEFAULT 0 NOT NULL,
    ai_usage_count integer DEFAULT 0 NOT NULL,
    last_reset_date date DEFAULT CURRENT_DATE NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_subscription_events; Type: TABLE; Schema: subscription; Owner: -
--

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
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_subscriptions; Type: TABLE; Schema: subscription; Owner: -
--

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
    started_at timestamp with time zone NOT NULL,
    fallback_plan_id uuid,
    fallback_plan_code character varying(100),
    fallback_tier_rank integer,
    fallback_plan_kind smallint,
    fallback_benefit_snapshot jsonb,
    last_deposit_transaction_id uuid,
    version bigint DEFAULT 0 NOT NULL,
    CONSTRAINT chk_user_subscription_auto_renew CHECK (((current_plan_kind = 1) OR (is_auto_renew_enabled = false))),
    CONSTRAINT chk_user_subscription_fallback CHECK ((((fallback_plan_id IS NULL) AND (fallback_plan_code IS NULL) AND (fallback_tier_rank IS NULL) AND (fallback_plan_kind IS NULL) AND (fallback_benefit_snapshot IS NULL)) OR ((fallback_plan_id IS NOT NULL) AND (fallback_plan_code IS NOT NULL) AND (fallback_tier_rank IS NOT NULL) AND (fallback_plan_kind = 2) AND (fallback_benefit_snapshot IS NOT NULL) AND (current_plan_kind = 1) AND (current_tier_rank > fallback_tier_rank)))),
    CONSTRAINT chk_user_subscription_kind_expiry CHECK ((((current_plan_kind = 1) AND (expires_at IS NOT NULL)) OR ((current_plan_kind = ANY (ARRAY[0, 2])) AND (expires_at IS NULL))))
);


--
-- Name: user_wallets; Type: TABLE; Schema: subscription; Owner: -
--

CREATE TABLE subscription.user_wallets (
    user_id uuid NOT NULL,
    balance numeric(12,2) DEFAULT 0.00 NOT NULL,
    currency character varying(10) DEFAULT 'VND'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: wallet_statements; Type: TABLE; Schema: subscription; Owner: -
--

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
    renewal_attempt_key character varying(255)
);


--
-- Name: outfit_items; Type: TABLE; Schema: wardrobe; Owner: -
--

CREATE TABLE wardrobe.outfit_items (
    outfit_id uuid NOT NULL,
    position_x double precision DEFAULT 0.0 NOT NULL,
    position_y double precision DEFAULT 0.0 NOT NULL,
    scale double precision DEFAULT 1.0 NOT NULL,
    layer_order smallint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    fashion_item_id uuid NOT NULL,
    item_context character varying(50) NOT NULL
);


--
-- Name: outfits; Type: TABLE; Schema: wardrobe; Owner: -
--

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
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: wardrobe_items; Type: TABLE; Schema: wardrobe; Owner: -
--

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
    task_id uuid
);


--
-- Name: ai_cost_policies ai_cost_policies_code_version_key; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_cost_policies
    ADD CONSTRAINT ai_cost_policies_code_version_key UNIQUE (code, version);


--
-- Name: ai_cost_policies ai_cost_policies_pkey; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_cost_policies
    ADD CONSTRAINT ai_cost_policies_pkey PRIMARY KEY (id);


--
-- Name: ai_cost_policy_audit_logs ai_cost_policy_audit_logs_pkey; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_cost_policy_audit_logs
    ADD CONSTRAINT ai_cost_policy_audit_logs_pkey PRIMARY KEY (id);


--
-- Name: ai_cost_policy_operations ai_cost_policy_operations_pkey; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_cost_policy_operations
    ADD CONSTRAINT ai_cost_policy_operations_pkey PRIMARY KEY (id);


--
-- Name: ai_cost_policy_operations ai_cost_policy_operations_policy_id_operation_key; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_cost_policy_operations
    ADD CONSTRAINT ai_cost_policy_operations_policy_id_operation_key UNIQUE (policy_id, operation);


--
-- Name: ai_model_configs ai_model_configs_pkey; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_model_configs
    ADD CONSTRAINT ai_model_configs_pkey PRIMARY KEY (id);


--
-- Name: ai_usage_events ai_usage_events_pkey; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_events
    ADD CONSTRAINT ai_usage_events_pkey PRIMARY KEY (id);


--
-- Name: ai_usage_events ai_usage_events_request_id_key; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_events
    ADD CONSTRAINT ai_usage_events_request_id_key UNIQUE (request_id);


--
-- Name: ai_usage_period_ledgers ai_usage_period_ledgers_grant_id_period_index_key; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_period_ledgers
    ADD CONSTRAINT ai_usage_period_ledgers_grant_id_period_index_key UNIQUE (grant_id, period_index);


--
-- Name: ai_usage_period_ledgers ai_usage_period_ledgers_pkey; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_period_ledgers
    ADD CONSTRAINT ai_usage_period_ledgers_pkey PRIMARY KEY (id);


--
-- Name: ai_model_configs uq_provider_model_active; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_model_configs
    ADD CONSTRAINT uq_provider_model_active UNIQUE (provider, model, effective_from);


--
-- Name: ai_policy_grants user_ai_policy_grants_pkey; Type: CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_policy_grants
    ADD CONSTRAINT user_ai_policy_grants_pkey PRIMARY KEY (id);


--
-- Name: benefit_redemptions benefit_redemptions_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.benefit_redemptions
    ADD CONSTRAINT benefit_redemptions_pkey PRIMARY KEY (id);


--
-- Name: brand_benefits brand_benefits_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_benefits
    ADD CONSTRAINT brand_benefits_pkey PRIMARY KEY (id);


--
-- Name: brand_conversation_messages brand_conversation_messages_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_conversation_messages
    ADD CONSTRAINT brand_conversation_messages_pkey PRIMARY KEY (id);


--
-- Name: brand_conversations brand_conversations_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_conversations
    ADD CONSTRAINT brand_conversations_pkey PRIMARY KEY (id);


--
-- Name: brand_customer_claims brand_customer_claims_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_customer_claims
    ADD CONSTRAINT brand_customer_claims_pkey PRIMARY KEY (id);


--
-- Name: brand_customers brand_customers_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_customers
    ADD CONSTRAINT brand_customers_pkey PRIMARY KEY (id);


--
-- Name: brand_items brand_items_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_items
    ADD CONSTRAINT brand_items_pkey PRIMARY KEY (id);


--
-- Name: brand_members brand_members_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_members
    ADD CONSTRAINT brand_members_pkey PRIMARY KEY (id);


--
-- Name: brands brands_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brands
    ADD CONSTRAINT brands_pkey PRIMARY KEY (id);


--
-- Name: brands brands_slug_key; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brands
    ADD CONSTRAINT brands_slug_key UNIQUE (slug);


--
-- Name: digital_sample_responses digital_sample_responses_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.digital_sample_responses
    ADD CONSTRAINT digital_sample_responses_pkey PRIMARY KEY (id);


--
-- Name: loyalty_accounts loyalty_accounts_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_accounts
    ADD CONSTRAINT loyalty_accounts_pkey PRIMARY KEY (id);


--
-- Name: loyalty_point_lots loyalty_point_lots_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_lots
    ADD CONSTRAINT loyalty_point_lots_pkey PRIMARY KEY (id);


--
-- Name: loyalty_point_transactions loyalty_point_transactions_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_transactions
    ADD CONSTRAINT loyalty_point_transactions_pkey PRIMARY KEY (id);


--
-- Name: loyalty_programs loyalty_programs_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_programs
    ADD CONSTRAINT loyalty_programs_pkey PRIMARY KEY (id);


--
-- Name: loyalty_tiers loyalty_tiers_pkey; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_tiers
    ADD CONSTRAINT loyalty_tiers_pkey PRIMARY KEY (id);


--
-- Name: brand_conversations uq_brand_conversations_brand_user; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_conversations
    ADD CONSTRAINT uq_brand_conversations_brand_user UNIQUE (brand_id, user_id);


--
-- Name: brand_items uq_brand_items_brand_product_code; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_items
    ADD CONSTRAINT uq_brand_items_brand_product_code UNIQUE (brand_id, product_code);


--
-- Name: brand_members uq_brand_members_brand_user; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_members
    ADD CONSTRAINT uq_brand_members_brand_user UNIQUE (brand_id, user_id);


--
-- Name: loyalty_accounts uq_loyalty_accounts_brand_customer; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_accounts
    ADD CONSTRAINT uq_loyalty_accounts_brand_customer UNIQUE (brand_customer_id);


--
-- Name: loyalty_tiers uq_loyalty_tiers_brand_name; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_tiers
    ADD CONSTRAINT uq_loyalty_tiers_brand_name UNIQUE (brand_id, name);


--
-- Name: loyalty_tiers uq_loyalty_tiers_brand_rank; Type: CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_tiers
    ADD CONSTRAINT uq_loyalty_tiers_brand_rank UNIQUE (brand_id, rank);


--
-- Name: follows follows_pkey; Type: CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.follows
    ADD CONSTRAINT follows_pkey PRIMARY KEY (follower_id, followee_id);


--
-- Name: post_comments post_comments_pkey; Type: CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_comments
    ADD CONSTRAINT post_comments_pkey PRIMARY KEY (id);


--
-- Name: post_likes post_likes_pkey; Type: CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_likes
    ADD CONSTRAINT post_likes_pkey PRIMARY KEY (post_id, user_id);


--
-- Name: post_media post_media_pkey; Type: CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_media
    ADD CONSTRAINT post_media_pkey PRIMARY KEY (id);


--
-- Name: posts posts_pkey; Type: CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.posts
    ADD CONSTRAINT posts_pkey PRIMARY KEY (id);


--
-- Name: posts posts_public_id_key; Type: CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.posts
    ADD CONSTRAINT posts_public_id_key UNIQUE (public_id);


--
-- Name: user_follow_stats user_follow_stats_pkey; Type: CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.user_follow_stats
    ADD CONSTRAINT user_follow_stats_pkey PRIMARY KEY (user_id);


--
-- Name: dashboard_admin_daily_snapshots dashboard_admin_daily_snapshots_pkey; Type: CONSTRAINT; Schema: dashboard; Owner: -
--

ALTER TABLE ONLY dashboard.dashboard_admin_daily_snapshots
    ADD CONSTRAINT dashboard_admin_daily_snapshots_pkey PRIMARY KEY (id);


--
-- Name: dashboard_admin_daily_snapshots dashboard_admin_daily_snapshots_snapshot_date_key; Type: CONSTRAINT; Schema: dashboard; Owner: -
--

ALTER TABLE ONLY dashboard.dashboard_admin_daily_snapshots
    ADD CONSTRAINT dashboard_admin_daily_snapshots_snapshot_date_key UNIQUE (snapshot_date);


--
-- Name: dashboard_brand_daily_snapshots dashboard_brand_daily_snapshots_pkey; Type: CONSTRAINT; Schema: dashboard; Owner: -
--

ALTER TABLE ONLY dashboard.dashboard_brand_daily_snapshots
    ADD CONSTRAINT dashboard_brand_daily_snapshots_pkey PRIMARY KEY (id);


--
-- Name: dashboard_user_wardrobe_summaries dashboard_user_wardrobe_summaries_pkey; Type: CONSTRAINT; Schema: dashboard; Owner: -
--

ALTER TABLE ONLY dashboard.dashboard_user_wardrobe_summaries
    ADD CONSTRAINT dashboard_user_wardrobe_summaries_pkey PRIMARY KEY (user_id);


--
-- Name: dashboard_brand_daily_snapshots uq_brand_daily_snapshot; Type: CONSTRAINT; Schema: dashboard; Owner: -
--

ALTER TABLE ONLY dashboard.dashboard_brand_daily_snapshots
    ADD CONSTRAINT uq_brand_daily_snapshot UNIQUE (brand_id, snapshot_date);


--
-- Name: categories categories_name_key; Type: CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.categories
    ADD CONSTRAINT categories_name_key UNIQUE (name);


--
-- Name: categories categories_pkey; Type: CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.categories
    ADD CONSTRAINT categories_pkey PRIMARY KEY (id);


--
-- Name: categories categories_slug_key; Type: CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.categories
    ADD CONSTRAINT categories_slug_key UNIQUE (slug);


--
-- Name: conversational_contexts conversational_contexts_pkey; Type: CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.conversational_contexts
    ADD CONSTRAINT conversational_contexts_pkey PRIMARY KEY (id);


--
-- Name: fashion_items fashion_items_pkey; Type: CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.fashion_items
    ADD CONSTRAINT fashion_items_pkey PRIMARY KEY (id);


--
-- Name: messages messages_pkey; Type: CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.messages
    ADD CONSTRAINT messages_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_token_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.refresh_tokens
    ADD CONSTRAINT refresh_tokens_token_key UNIQUE (token);


--
-- Name: user_activity_events user_activity_events_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.user_activity_events
    ADD CONSTRAINT user_activity_events_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: goose_db_version_commerce goose_db_version_commerce_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.goose_db_version_commerce
    ADD CONSTRAINT goose_db_version_commerce_pkey PRIMARY KEY (id);


--
-- Name: goose_db_version goose_db_version_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.goose_db_version
    ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);


--
-- Name: admin_audit_logs admin_audit_logs_pkey; Type: CONSTRAINT; Schema: shared; Owner: -
--

ALTER TABLE ONLY shared.admin_audit_logs
    ADD CONSTRAINT admin_audit_logs_pkey PRIMARY KEY (id);


--
-- Name: outbox_records outbox_records_pkey; Type: CONSTRAINT; Schema: shared; Owner: -
--

ALTER TABLE ONLY shared.outbox_records
    ADD CONSTRAINT outbox_records_pkey PRIMARY KEY (id);


--
-- Name: campaign_claims campaign_claims_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.campaign_claims
    ADD CONSTRAINT campaign_claims_pkey PRIMARY KEY (id);


--
-- Name: campaign_sweep_runs campaign_sweep_runs_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.campaign_sweep_runs
    ADD CONSTRAINT campaign_sweep_runs_pkey PRIMARY KEY (campaign_code);


--
-- Name: campaigns campaigns_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.campaigns
    ADD CONSTRAINT campaigns_pkey PRIMARY KEY (code);


--
-- Name: deposit_transactions deposit_transactions_gateway_reference_key; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.deposit_transactions
    ADD CONSTRAINT deposit_transactions_gateway_reference_key UNIQUE (gateway_reference);


--
-- Name: deposit_transactions deposit_transactions_order_code_key; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.deposit_transactions
    ADD CONSTRAINT deposit_transactions_order_code_key UNIQUE (order_code);


--
-- Name: deposit_transactions deposit_transactions_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.deposit_transactions
    ADD CONSTRAINT deposit_transactions_pkey PRIMARY KEY (id);


--
-- Name: provider_payment_events provider_payment_events_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.provider_payment_events
    ADD CONSTRAINT provider_payment_events_pkey PRIMARY KEY (id);


--
-- Name: provider_payment_events provider_payment_events_provider_provider_reference_key; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.provider_payment_events
    ADD CONSTRAINT provider_payment_events_provider_provider_reference_key UNIQUE (provider, provider_reference);


--
-- Name: provider_webhook_inbox provider_webhook_inbox_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.provider_webhook_inbox
    ADD CONSTRAINT provider_webhook_inbox_pkey PRIMARY KEY (id);


--
-- Name: subscription_plans subscription_plans_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.subscription_plans
    ADD CONSTRAINT subscription_plans_pkey PRIMARY KEY (id);


--
-- Name: subscription_plans subscription_plans_slug_key; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.subscription_plans
    ADD CONSTRAINT subscription_plans_slug_key UNIQUE (slug);


--
-- Name: subscription_renewal_attempts subscription_renewal_attempts_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.subscription_renewal_attempts
    ADD CONSTRAINT subscription_renewal_attempts_pkey PRIMARY KEY (id);


--
-- Name: subscription_renewal_attempts subscription_renewal_attempts_renewal_attempt_key_key; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.subscription_renewal_attempts
    ADD CONSTRAINT subscription_renewal_attempts_renewal_attempt_key_key UNIQUE (renewal_attempt_key);


--
-- Name: user_daily_quotas user_daily_quotas_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_daily_quotas
    ADD CONSTRAINT user_daily_quotas_pkey PRIMARY KEY (user_id);


--
-- Name: user_subscription_events user_subscription_events_event_key_key; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscription_events
    ADD CONSTRAINT user_subscription_events_event_key_key UNIQUE (event_key);


--
-- Name: user_subscription_events user_subscription_events_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscription_events
    ADD CONSTRAINT user_subscription_events_pkey PRIMARY KEY (id);


--
-- Name: user_subscriptions user_subscriptions_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscriptions
    ADD CONSTRAINT user_subscriptions_pkey PRIMARY KEY (user_id);


--
-- Name: user_wallets user_wallets_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_wallets
    ADD CONSTRAINT user_wallets_pkey PRIMARY KEY (user_id);


--
-- Name: campaign_claims ux_campaign_claims_user; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.campaign_claims
    ADD CONSTRAINT ux_campaign_claims_user UNIQUE (user_id);


--
-- Name: wallet_statements wallet_statements_pkey; Type: CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.wallet_statements
    ADD CONSTRAINT wallet_statements_pkey PRIMARY KEY (id);


--
-- Name: outfit_items outfit_items_pkey; Type: CONSTRAINT; Schema: wardrobe; Owner: -
--

ALTER TABLE ONLY wardrobe.outfit_items
    ADD CONSTRAINT outfit_items_pkey PRIMARY KEY (outfit_id, fashion_item_id, item_context);


--
-- Name: outfits outfits_pkey; Type: CONSTRAINT; Schema: wardrobe; Owner: -
--

ALTER TABLE ONLY wardrobe.outfits
    ADD CONSTRAINT outfits_pkey PRIMARY KEY (id);


--
-- Name: wardrobe_items wardrobe_items_pkey; Type: CONSTRAINT; Schema: wardrobe; Owner: -
--

ALTER TABLE ONLY wardrobe.wardrobe_items
    ADD CONSTRAINT wardrobe_items_pkey PRIMARY KEY (id);


--
-- Name: ix_ai_cost_policy_audit_logs_actor_user_id; Type: INDEX; Schema: aiexecutor; Owner: -
--

CREATE INDEX ix_ai_cost_policy_audit_logs_actor_user_id ON aiexecutor.ai_cost_policy_audit_logs USING btree (actor_user_id);


--
-- Name: ix_ai_cost_policy_audit_logs_changed_at; Type: INDEX; Schema: aiexecutor; Owner: -
--

CREATE INDEX ix_ai_cost_policy_audit_logs_changed_at ON aiexecutor.ai_cost_policy_audit_logs USING btree (changed_at);


--
-- Name: ix_ai_policy_grants_source_event_id; Type: INDEX; Schema: aiexecutor; Owner: -
--

CREATE INDEX ix_ai_policy_grants_source_event_id ON aiexecutor.ai_policy_grants USING btree (source_event_id);


--
-- Name: ix_ai_usage_events_unknown; Type: INDEX; Schema: aiexecutor; Owner: -
--

CREATE INDEX ix_ai_usage_events_unknown ON aiexecutor.ai_usage_events USING btree (status, unknown_expires_at) WHERE ((status)::text = ANY (ARRAY[('reserved'::character varying)::text, ('unknown_usage'::character varying)::text]));


--
-- Name: ix_ai_usage_events_user_day; Type: INDEX; Schema: aiexecutor; Owner: -
--

CREATE INDEX ix_ai_usage_events_user_day ON aiexecutor.ai_usage_events USING btree (user_id, created_at, status);


--
-- Name: ix_user_ai_policy_grants_active; Type: INDEX; Schema: aiexecutor; Owner: -
--

CREATE INDEX ix_user_ai_policy_grants_active ON aiexecutor.ai_policy_grants USING btree (user_id, effective_from, effective_to);


--
-- Name: ux_ledgers_user_period_fallback; Type: INDEX; Schema: aiexecutor; Owner: -
--

CREATE UNIQUE INDEX ux_ledgers_user_period_fallback ON aiexecutor.ai_usage_period_ledgers USING btree (user_id, period_index) WHERE (grant_id IS NULL);


--
-- Name: idx_benefit_redemptions_customer_status; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_benefit_redemptions_customer_status ON brand.benefit_redemptions USING btree (brand_customer_id, status);


--
-- Name: idx_benefit_redemptions_user_status; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_benefit_redemptions_user_status ON brand.benefit_redemptions USING btree (user_id, status);


--
-- Name: idx_brand_benefits_brand_status; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_benefits_brand_status ON brand.brand_benefits USING btree (brand_id, status);


--
-- Name: idx_brand_conversation_messages_conv_created; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_conversation_messages_conv_created ON brand.brand_conversation_messages USING btree (conversation_id, created_at);


--
-- Name: idx_brand_conversation_messages_unread; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_conversation_messages_unread ON brand.brand_conversation_messages USING btree (conversation_id, sender_role, created_at);


--
-- Name: idx_brand_conversations_brand_status; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_conversations_brand_status ON brand.brand_conversations USING btree (brand_id, status);


--
-- Name: idx_brand_customer_claims_active_customer; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_customer_claims_active_customer ON brand.brand_customer_claims USING btree (brand_customer_id) WHERE ((consumed_at IS NULL) AND (revoked_at IS NULL));


--
-- Name: idx_brand_customer_claims_customer; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_customer_claims_customer ON brand.brand_customer_claims USING btree (brand_customer_id);


--
-- Name: idx_brand_customers_brand_status; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_customers_brand_status ON brand.brand_customers USING btree (brand_id, status);


--
-- Name: idx_brand_items_brand_status; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_items_brand_status ON brand.brand_items USING btree (brand_id, status);


--
-- Name: idx_brand_items_task_id; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_items_task_id ON brand.brand_items USING btree (task_id);


--
-- Name: idx_brand_members_user_status; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_brand_members_user_status ON brand.brand_members USING btree (user_id, status);


--
-- Name: idx_digital_sample_responses_item_user; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_digital_sample_responses_item_user ON brand.digital_sample_responses USING btree (brand_item_id, user_id);


--
-- Name: idx_loyalty_accounts_brand_tier; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_loyalty_accounts_brand_tier ON brand.loyalty_accounts USING btree (brand_id, current_tier_id);


--
-- Name: idx_loyalty_point_lots_account_active_expiry; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_loyalty_point_lots_account_active_expiry ON brand.loyalty_point_lots USING btree (loyalty_account_id, expires_at) WHERE (((status)::text = 'ACTIVE'::text) AND (remaining_points > 0));


--
-- Name: idx_loyalty_point_lots_brand_customer; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_loyalty_point_lots_brand_customer ON brand.loyalty_point_lots USING btree (brand_customer_id);


--
-- Name: idx_loyalty_point_lots_expiry_worker; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_loyalty_point_lots_expiry_worker ON brand.loyalty_point_lots USING btree (expires_at) WHERE (((status)::text = 'ACTIVE'::text) AND (remaining_points > 0) AND (expires_at IS NOT NULL));


--
-- Name: idx_loyalty_point_transactions_account_created; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_loyalty_point_transactions_account_created ON brand.loyalty_point_transactions USING btree (loyalty_account_id, created_at);


--
-- Name: idx_loyalty_point_transactions_brand_user; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_loyalty_point_transactions_brand_user ON brand.loyalty_point_transactions USING btree (brand_id, user_id);


--
-- Name: idx_loyalty_point_transactions_customer_created; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_loyalty_point_transactions_customer_created ON brand.loyalty_point_transactions USING btree (brand_customer_id, created_at);


--
-- Name: idx_loyalty_point_transactions_expires_at; Type: INDEX; Schema: brand; Owner: -
--

CREATE INDEX idx_loyalty_point_transactions_expires_at ON brand.loyalty_point_transactions USING btree (expires_at) WHERE (expires_at IS NOT NULL);


--
-- Name: uq_brand_customer_claims_token_hash; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX uq_brand_customer_claims_token_hash ON brand.brand_customer_claims USING btree (claim_token_hash);


--
-- Name: uq_brand_customers_brand_phone_hash; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX uq_brand_customers_brand_phone_hash ON brand.brand_customers USING btree (brand_id, phone_hash) WHERE (phone_hash IS NOT NULL);


--
-- Name: uq_brand_customers_brand_user; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX uq_brand_customers_brand_user ON brand.brand_customers USING btree (brand_id, user_id) WHERE (user_id IS NOT NULL);


--
-- Name: uq_brand_items_fashion_item; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX uq_brand_items_fashion_item ON brand.brand_items USING btree (fashion_item_id) WHERE (fashion_item_id IS NOT NULL);


--
-- Name: uq_loyalty_accounts_brand_user; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX uq_loyalty_accounts_brand_user ON brand.loyalty_accounts USING btree (brand_id, user_id) WHERE (user_id IS NOT NULL);


--
-- Name: uq_loyalty_point_transactions_brand_idempotency; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX uq_loyalty_point_transactions_brand_idempotency ON brand.loyalty_point_transactions USING btree (brand_id, idempotency_key) WHERE (idempotency_key IS NOT NULL);


--
-- Name: uq_loyalty_programs_active_brand; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX uq_loyalty_programs_active_brand ON brand.loyalty_programs USING btree (brand_id) WHERE (is_active = true);


--
-- Name: ux_brand_members_one_active_owner; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX ux_brand_members_one_active_owner ON brand.brand_members USING btree (brand_id) WHERE (((role)::text = 'owner'::text) AND ((status)::text = 'active'::text));


--
-- Name: ux_loyalty_point_lots_earn_transaction; Type: INDEX; Schema: brand; Owner: -
--

CREATE UNIQUE INDEX ux_loyalty_point_lots_earn_transaction ON brand.loyalty_point_lots USING btree (earn_transaction_id);


--
-- Name: idx_comments_post_parent_created; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_comments_post_parent_created ON community.post_comments USING btree (post_id, parent_comment_id, created_at);


--
-- Name: idx_comments_root_created; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_comments_root_created ON community.post_comments USING btree (root_comment_id, created_at);


--
-- Name: idx_follows_followee_created; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_follows_followee_created ON community.follows USING btree (followee_id, created_at DESC);


--
-- Name: idx_follows_follower_created; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_follows_follower_created ON community.follows USING btree (follower_id, created_at DESC);


--
-- Name: idx_post_likes_post; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_post_likes_post ON community.post_likes USING btree (post_id, created_at DESC);


--
-- Name: idx_post_likes_user; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_post_likes_user ON community.post_likes USING btree (user_id, created_at DESC);


--
-- Name: idx_post_media_post_sort; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_post_media_post_sort ON community.post_media USING btree (post_id, sort_order);


--
-- Name: idx_posts_outfit; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_posts_outfit ON community.posts USING btree (outfit_id) WHERE ((outfit_id IS NOT NULL) AND ((status)::text = 'published'::text));


--
-- Name: idx_posts_published_created; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_posts_published_created ON community.posts USING btree (created_at DESC, id DESC) WHERE ((status)::text = 'published'::text);


--
-- Name: idx_posts_search; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_posts_search ON community.posts USING gin (lower("left"(content, 200)) public.gin_trgm_ops) WHERE ((status)::text = 'published'::text);


--
-- Name: idx_posts_type_created; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_posts_type_created ON community.posts USING btree (post_type, created_at DESC) WHERE ((status)::text = 'published'::text);


--
-- Name: idx_posts_user_created; Type: INDEX; Schema: community; Owner: -
--

CREATE INDEX idx_posts_user_created ON community.posts USING btree (user_id, created_at DESC) WHERE ((status)::text = 'published'::text);


--
-- Name: idx_admin_snapshots_date; Type: INDEX; Schema: dashboard; Owner: -
--

CREATE INDEX idx_admin_snapshots_date ON dashboard.dashboard_admin_daily_snapshots USING btree (snapshot_date DESC);


--
-- Name: idx_brand_snapshots_brand_date; Type: INDEX; Schema: dashboard; Owner: -
--

CREATE INDEX idx_brand_snapshots_brand_date ON dashboard.dashboard_brand_daily_snapshots USING btree (brand_id, snapshot_date DESC);


--
-- Name: idx_user_wardrobe_summaries_updated; Type: INDEX; Schema: dashboard; Owner: -
--

CREATE INDEX idx_user_wardrobe_summaries_updated ON dashboard.dashboard_user_wardrobe_summaries USING btree (updated_at DESC);


--
-- Name: fitems_embedding_cosine_idx; Type: INDEX; Schema: fashion; Owner: -
--

CREATE INDEX fitems_embedding_cosine_idx ON fashion.fashion_items USING hnsw (embedding public.vector_cosine_ops);


--
-- Name: idx_fashion_items_category_id; Type: INDEX; Schema: fashion; Owner: -
--

CREATE INDEX idx_fashion_items_category_id ON fashion.fashion_items USING btree (category_id);


--
-- Name: idx_fashion_items_lexical_search; Type: INDEX; Schema: fashion; Owner: -
--

CREATE INDEX idx_fashion_items_lexical_search ON fashion.fashion_items USING gin (to_tsvector('simple'::regconfig, public.immutable_unaccent(lower((((((((((((((COALESCE(color, ''::character varying))::text || ' '::text) || (COALESCE(style, ''::character varying))::text) || ' '::text) || (COALESCE(material, ''::character varying))::text) || ' '::text) || (COALESCE(pattern, ''::character varying))::text) || ' '::text) || (COALESCE(fit, ''::character varying))::text) || ' '::text) || (COALESCE(seasonality, ''::character varying))::text) || ' '::text) || COALESCE(description, ''::text))))));


--
-- Name: idx_messages_context_created_at; Type: INDEX; Schema: fashion; Owner: -
--

CREATE INDEX idx_messages_context_created_at ON fashion.messages USING btree (context_id, created_at DESC);


--
-- Name: idx_messages_context_unsummarized; Type: INDEX; Schema: fashion; Owner: -
--

CREATE INDEX idx_messages_context_unsummarized ON fashion.messages USING btree (context_id) WHERE (is_summarized = false);


--
-- Name: idx_user_activity_events_occurred_at; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_user_activity_events_occurred_at ON identity.user_activity_events USING btree (occurred_at);


--
-- Name: idx_user_activity_events_user_occurred_at; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_user_activity_events_user_occurred_at ON identity.user_activity_events USING btree (user_id, occurred_at);


--
-- Name: idx_users_email_lower; Type: INDEX; Schema: identity; Owner: -
--

CREATE UNIQUE INDEX idx_users_email_lower ON identity.users USING btree (lower((email)::text));


--
-- Name: idx_users_provider_provider_id; Type: INDEX; Schema: identity; Owner: -
--

CREATE UNIQUE INDEX idx_users_provider_provider_id ON identity.users USING btree (provider, provider_id) WHERE (provider_id IS NOT NULL);


--
-- Name: idx_users_username_lower; Type: INDEX; Schema: identity; Owner: -
--

CREATE UNIQUE INDEX idx_users_username_lower ON identity.users USING btree (lower((username)::text));


--
-- Name: idx_admin_audit_logs_actor; Type: INDEX; Schema: shared; Owner: -
--

CREATE INDEX idx_admin_audit_logs_actor ON shared.admin_audit_logs USING btree (actor_user_id);


--
-- Name: idx_admin_audit_logs_created_at; Type: INDEX; Schema: shared; Owner: -
--

CREATE INDEX idx_admin_audit_logs_created_at ON shared.admin_audit_logs USING btree (created_at);


--
-- Name: idx_admin_audit_logs_target; Type: INDEX; Schema: shared; Owner: -
--

CREATE INDEX idx_admin_audit_logs_target ON shared.admin_audit_logs USING btree (target_type, target_id);


--
-- Name: idx_outbox_records_status_created_at; Type: INDEX; Schema: shared; Owner: -
--

CREATE INDEX idx_outbox_records_status_created_at ON shared.outbox_records USING btree (status, created_at);


--
-- Name: idx_outbox_records_status_published_at; Type: INDEX; Schema: shared; Owner: -
--

CREATE INDEX idx_outbox_records_status_published_at ON shared.outbox_records USING btree (status, published_at);


--
-- Name: idx_deposit_reconciliation; Type: INDEX; Schema: subscription; Owner: -
--

CREATE INDEX idx_deposit_reconciliation ON subscription.deposit_transactions USING btree (next_reconciliation_at, expires_at) WHERE (status = ANY (ARRAY[0, 3, 4, 5]));


--
-- Name: idx_user_subscriptions_expires; Type: INDEX; Schema: subscription; Owner: -
--

CREATE INDEX idx_user_subscriptions_expires ON subscription.user_subscriptions USING btree (expires_at, user_id) WHERE (current_plan_kind = 1);


--
-- Name: idx_user_subscriptions_plan_id; Type: INDEX; Schema: subscription; Owner: -
--

CREATE INDEX idx_user_subscriptions_plan_id ON subscription.user_subscriptions USING btree (subscription_plan_id);


--
-- Name: ix_campaign_claims_campaign_bucket; Type: INDEX; Schema: subscription; Owner: -
--

CREATE INDEX ix_campaign_claims_campaign_bucket ON subscription.campaign_claims USING btree (campaign_code, grant_bucket);


--
-- Name: ix_campaign_claims_campaign_granted; Type: INDEX; Schema: subscription; Owner: -
--

CREATE INDEX ix_campaign_claims_campaign_granted ON subscription.campaign_claims USING btree (campaign_code, granted_at DESC);


--
-- Name: ix_campaigns_starts_at; Type: INDEX; Schema: subscription; Owner: -
--

CREATE INDEX ix_campaigns_starts_at ON subscription.campaigns USING btree (starts_at DESC);


--
-- Name: ux_active_direct_purchase_per_user; Type: INDEX; Schema: subscription; Owner: -
--

CREATE UNIQUE INDEX ux_active_direct_purchase_per_user ON subscription.deposit_transactions USING btree (user_id) WHERE (((transaction_type)::text = 'DIRECT_PURCHASE'::text) AND (status = ANY (ARRAY[3, 0, 4, 5])));


--
-- Name: ux_campaign_claims_watermark; Type: INDEX; Schema: subscription; Owner: -
--

CREATE UNIQUE INDEX ux_campaign_claims_watermark ON subscription.campaign_claims USING btree (campaign_code) WHERE fills_main_quota;


--
-- Name: ux_single_active_default_free_plan; Type: INDEX; Schema: subscription; Owner: -
--

CREATE UNIQUE INDEX ux_single_active_default_free_plan ON subscription.subscription_plans USING btree (plan_kind) WHERE ((plan_kind = 0) AND (is_active = true));


--
-- Name: ux_wallet_statement_renewal_attempt; Type: INDEX; Schema: subscription; Owner: -
--

CREATE UNIQUE INDEX ux_wallet_statement_renewal_attempt ON subscription.wallet_statements USING btree (renewal_attempt_key) WHERE (renewal_attempt_key IS NOT NULL);


--
-- Name: ux_webhook_payload_hash; Type: INDEX; Schema: subscription; Owner: -
--

CREATE UNIQUE INDEX ux_webhook_payload_hash ON subscription.provider_webhook_inbox USING btree (provider, canonical_payload_hash) WHERE (provider_reference IS NULL);


--
-- Name: ux_webhook_provider_reference; Type: INDEX; Schema: subscription; Owner: -
--

CREATE UNIQUE INDEX ux_webhook_provider_reference ON subscription.provider_webhook_inbox USING btree (provider, provider_reference, event_code) WHERE (provider_reference IS NOT NULL);


--
-- Name: idx_outfit_items_fashion_item_context; Type: INDEX; Schema: wardrobe; Owner: -
--

CREATE INDEX idx_outfit_items_fashion_item_context ON wardrobe.outfit_items USING btree (fashion_item_id, item_context);


--
-- Name: idx_wardrobe_items_fashion_item; Type: INDEX; Schema: wardrobe; Owner: -
--

CREATE INDEX idx_wardrobe_items_fashion_item ON wardrobe.wardrobe_items USING btree (fashion_item_id);


--
-- Name: idx_wardrobe_items_last_used_at; Type: INDEX; Schema: wardrobe; Owner: -
--

CREATE INDEX idx_wardrobe_items_last_used_at ON wardrobe.wardrobe_items USING btree (last_used_at);


--
-- Name: idx_wardrobe_items_task_id; Type: INDEX; Schema: wardrobe; Owner: -
--

CREATE INDEX idx_wardrobe_items_task_id ON wardrobe.wardrobe_items USING btree (task_id);


--
-- Name: idx_wardrobe_items_type_status; Type: INDEX; Schema: wardrobe; Owner: -
--

CREATE INDEX idx_wardrobe_items_type_status ON wardrobe.wardrobe_items USING btree (item_type, is_deleted);


--
-- Name: idx_wardrobe_items_user_fashion_item; Type: INDEX; Schema: wardrobe; Owner: -
--

CREATE INDEX idx_wardrobe_items_user_fashion_item ON wardrobe.wardrobe_items USING btree (user_id, fashion_item_id);


--
-- Name: idx_wardrobe_items_user_status; Type: INDEX; Schema: wardrobe; Owner: -
--

CREATE INDEX idx_wardrobe_items_user_status ON wardrobe.wardrobe_items USING btree (user_id, is_deleted, status);


--
-- Name: campaigns trg_campaigns_immutable_budget; Type: TRIGGER; Schema: subscription; Owner: -
--

CREATE TRIGGER trg_campaigns_immutable_budget BEFORE UPDATE ON subscription.campaigns FOR EACH ROW EXECUTE FUNCTION subscription.campaigns_guard_immutable_budget();


--
-- Name: ai_cost_policy_operations ai_cost_policy_operations_policy_id_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_cost_policy_operations
    ADD CONSTRAINT ai_cost_policy_operations_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES aiexecutor.ai_cost_policies(id) ON DELETE CASCADE;


--
-- Name: ai_model_configs ai_model_configs_created_by_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_model_configs
    ADD CONSTRAINT ai_model_configs_created_by_fkey FOREIGN KEY (created_by) REFERENCES identity.users(id);


--
-- Name: ai_usage_events ai_usage_events_ledger_id_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_events
    ADD CONSTRAINT ai_usage_events_ledger_id_fkey FOREIGN KEY (ledger_id) REFERENCES aiexecutor.ai_usage_period_ledgers(id) ON DELETE CASCADE;


--
-- Name: ai_usage_events ai_usage_events_pricing_config_id_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_events
    ADD CONSTRAINT ai_usage_events_pricing_config_id_fkey FOREIGN KEY (pricing_config_id) REFERENCES aiexecutor.ai_model_configs(id);


--
-- Name: ai_usage_events ai_usage_events_user_id_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_events
    ADD CONSTRAINT ai_usage_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: ai_usage_period_ledgers ai_usage_period_ledgers_grant_id_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_period_ledgers
    ADD CONSTRAINT ai_usage_period_ledgers_grant_id_fkey FOREIGN KEY (grant_id) REFERENCES aiexecutor.ai_policy_grants(id) ON DELETE CASCADE;


--
-- Name: ai_usage_period_ledgers ai_usage_period_ledgers_user_id_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_usage_period_ledgers
    ADD CONSTRAINT ai_usage_period_ledgers_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: ai_policy_grants user_ai_policy_grants_policy_id_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_policy_grants
    ADD CONSTRAINT user_ai_policy_grants_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES aiexecutor.ai_cost_policies(id) ON DELETE RESTRICT;


--
-- Name: ai_policy_grants user_ai_policy_grants_user_id_fkey; Type: FK CONSTRAINT; Schema: aiexecutor; Owner: -
--

ALTER TABLE ONLY aiexecutor.ai_policy_grants
    ADD CONSTRAINT user_ai_policy_grants_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: benefit_redemptions benefit_redemptions_benefit_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.benefit_redemptions
    ADD CONSTRAINT benefit_redemptions_benefit_id_fkey FOREIGN KEY (benefit_id) REFERENCES brand.brand_benefits(id) ON DELETE CASCADE;


--
-- Name: benefit_redemptions benefit_redemptions_brand_customer_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.benefit_redemptions
    ADD CONSTRAINT benefit_redemptions_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand.brand_customers(id) ON DELETE CASCADE;


--
-- Name: benefit_redemptions benefit_redemptions_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.benefit_redemptions
    ADD CONSTRAINT benefit_redemptions_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: benefit_redemptions benefit_redemptions_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.benefit_redemptions
    ADD CONSTRAINT benefit_redemptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: brand_benefits brand_benefits_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_benefits
    ADD CONSTRAINT brand_benefits_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: brand_benefits brand_benefits_required_tier_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_benefits
    ADD CONSTRAINT brand_benefits_required_tier_id_fkey FOREIGN KEY (required_tier_id) REFERENCES brand.loyalty_tiers(id) ON DELETE SET NULL;


--
-- Name: brand_conversation_messages brand_conversation_messages_conversation_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_conversation_messages
    ADD CONSTRAINT brand_conversation_messages_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES brand.brand_conversations(id) ON DELETE CASCADE;


--
-- Name: brand_conversation_messages brand_conversation_messages_sender_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_conversation_messages
    ADD CONSTRAINT brand_conversation_messages_sender_user_id_fkey FOREIGN KEY (sender_user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: brand_conversations brand_conversations_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_conversations
    ADD CONSTRAINT brand_conversations_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: brand_conversations brand_conversations_closed_by_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_conversations
    ADD CONSTRAINT brand_conversations_closed_by_user_id_fkey FOREIGN KEY (closed_by_user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: brand_conversations brand_conversations_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_conversations
    ADD CONSTRAINT brand_conversations_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: brand_customer_claims brand_customer_claims_brand_customer_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_customer_claims
    ADD CONSTRAINT brand_customer_claims_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand.brand_customers(id) ON DELETE CASCADE;


--
-- Name: brand_customer_claims brand_customer_claims_revoked_by_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_customer_claims
    ADD CONSTRAINT brand_customer_claims_revoked_by_user_id_fkey FOREIGN KEY (revoked_by_user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: brand_customers brand_customers_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_customers
    ADD CONSTRAINT brand_customers_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: brand_customers brand_customers_created_by_member_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_customers
    ADD CONSTRAINT brand_customers_created_by_member_id_fkey FOREIGN KEY (created_by_member_id) REFERENCES brand.brand_members(id) ON DELETE SET NULL;


--
-- Name: brand_customers brand_customers_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_customers
    ADD CONSTRAINT brand_customers_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: brand_items brand_items_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_items
    ADD CONSTRAINT brand_items_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: brand_members brand_members_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_members
    ADD CONSTRAINT brand_members_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: brand_members brand_members_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brand_members
    ADD CONSTRAINT brand_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: brands brands_approved_by_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brands
    ADD CONSTRAINT brands_approved_by_user_id_fkey FOREIGN KEY (approved_by_user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: brands brands_created_by_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.brands
    ADD CONSTRAINT brands_created_by_user_id_fkey FOREIGN KEY (created_by_user_id) REFERENCES identity.users(id) ON DELETE RESTRICT;


--
-- Name: digital_sample_responses digital_sample_responses_brand_item_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.digital_sample_responses
    ADD CONSTRAINT digital_sample_responses_brand_item_id_fkey FOREIGN KEY (brand_item_id) REFERENCES brand.brand_items(id) ON DELETE CASCADE;


--
-- Name: digital_sample_responses digital_sample_responses_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.digital_sample_responses
    ADD CONSTRAINT digital_sample_responses_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: loyalty_accounts loyalty_accounts_brand_customer_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_accounts
    ADD CONSTRAINT loyalty_accounts_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand.brand_customers(id) ON DELETE CASCADE;


--
-- Name: loyalty_accounts loyalty_accounts_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_accounts
    ADD CONSTRAINT loyalty_accounts_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: loyalty_accounts loyalty_accounts_current_tier_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_accounts
    ADD CONSTRAINT loyalty_accounts_current_tier_id_fkey FOREIGN KEY (current_tier_id) REFERENCES brand.loyalty_tiers(id) ON DELETE SET NULL;


--
-- Name: loyalty_accounts loyalty_accounts_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_accounts
    ADD CONSTRAINT loyalty_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: loyalty_point_lots loyalty_point_lots_brand_customer_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_lots
    ADD CONSTRAINT loyalty_point_lots_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand.brand_customers(id) ON DELETE CASCADE;


--
-- Name: loyalty_point_lots loyalty_point_lots_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_lots
    ADD CONSTRAINT loyalty_point_lots_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: loyalty_point_lots loyalty_point_lots_earn_transaction_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_lots
    ADD CONSTRAINT loyalty_point_lots_earn_transaction_id_fkey FOREIGN KEY (earn_transaction_id) REFERENCES brand.loyalty_point_transactions(id) ON DELETE RESTRICT;


--
-- Name: loyalty_point_lots loyalty_point_lots_loyalty_account_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_lots
    ADD CONSTRAINT loyalty_point_lots_loyalty_account_id_fkey FOREIGN KEY (loyalty_account_id) REFERENCES brand.loyalty_accounts(id) ON DELETE CASCADE;


--
-- Name: loyalty_point_lots loyalty_point_lots_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_lots
    ADD CONSTRAINT loyalty_point_lots_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: loyalty_point_transactions loyalty_point_transactions_brand_customer_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_transactions
    ADD CONSTRAINT loyalty_point_transactions_brand_customer_id_fkey FOREIGN KEY (brand_customer_id) REFERENCES brand.brand_customers(id) ON DELETE CASCADE;


--
-- Name: loyalty_point_transactions loyalty_point_transactions_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_transactions
    ADD CONSTRAINT loyalty_point_transactions_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: loyalty_point_transactions loyalty_point_transactions_created_by_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_transactions
    ADD CONSTRAINT loyalty_point_transactions_created_by_user_id_fkey FOREIGN KEY (created_by_user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: loyalty_point_transactions loyalty_point_transactions_loyalty_account_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_transactions
    ADD CONSTRAINT loyalty_point_transactions_loyalty_account_id_fkey FOREIGN KEY (loyalty_account_id) REFERENCES brand.loyalty_accounts(id) ON DELETE CASCADE;


--
-- Name: loyalty_point_transactions loyalty_point_transactions_user_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_point_transactions
    ADD CONSTRAINT loyalty_point_transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE SET NULL;


--
-- Name: loyalty_programs loyalty_programs_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_programs
    ADD CONSTRAINT loyalty_programs_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: loyalty_tiers loyalty_tiers_brand_id_fkey; Type: FK CONSTRAINT; Schema: brand; Owner: -
--

ALTER TABLE ONLY brand.loyalty_tiers
    ADD CONSTRAINT loyalty_tiers_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brand.brands(id) ON DELETE CASCADE;


--
-- Name: follows follows_followee_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.follows
    ADD CONSTRAINT follows_followee_id_fkey FOREIGN KEY (followee_id) REFERENCES identity.users(id);


--
-- Name: follows follows_follower_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.follows
    ADD CONSTRAINT follows_follower_id_fkey FOREIGN KEY (follower_id) REFERENCES identity.users(id);


--
-- Name: post_comments post_comments_parent_comment_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_comments
    ADD CONSTRAINT post_comments_parent_comment_id_fkey FOREIGN KEY (parent_comment_id) REFERENCES community.post_comments(id) ON DELETE CASCADE;


--
-- Name: post_comments post_comments_post_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_comments
    ADD CONSTRAINT post_comments_post_id_fkey FOREIGN KEY (post_id) REFERENCES community.posts(id) ON DELETE CASCADE;


--
-- Name: post_comments post_comments_user_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_comments
    ADD CONSTRAINT post_comments_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id);


--
-- Name: post_likes post_likes_post_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_likes
    ADD CONSTRAINT post_likes_post_id_fkey FOREIGN KEY (post_id) REFERENCES community.posts(id) ON DELETE CASCADE;


--
-- Name: post_likes post_likes_user_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_likes
    ADD CONSTRAINT post_likes_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id);


--
-- Name: post_media post_media_post_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.post_media
    ADD CONSTRAINT post_media_post_id_fkey FOREIGN KEY (post_id) REFERENCES community.posts(id) ON DELETE CASCADE;


--
-- Name: posts posts_user_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.posts
    ADD CONSTRAINT posts_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id);


--
-- Name: user_follow_stats user_follow_stats_user_id_fkey; Type: FK CONSTRAINT; Schema: community; Owner: -
--

ALTER TABLE ONLY community.user_follow_stats
    ADD CONSTRAINT user_follow_stats_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id);


--
-- Name: dashboard_user_wardrobe_summaries dashboard_user_wardrobe_summaries_user_id_fkey; Type: FK CONSTRAINT; Schema: dashboard; Owner: -
--

ALTER TABLE ONLY dashboard.dashboard_user_wardrobe_summaries
    ADD CONSTRAINT dashboard_user_wardrobe_summaries_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: conversational_contexts conversational_contexts_user_id_fkey; Type: FK CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.conversational_contexts
    ADD CONSTRAINT conversational_contexts_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: fashion_items fashion_items_category_id_fkey; Type: FK CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.fashion_items
    ADD CONSTRAINT fashion_items_category_id_fkey FOREIGN KEY (category_id) REFERENCES fashion.categories(id) ON DELETE RESTRICT;


--
-- Name: messages messages_context_id_fkey; Type: FK CONSTRAINT; Schema: fashion; Owner: -
--

ALTER TABLE ONLY fashion.messages
    ADD CONSTRAINT messages_context_id_fkey FOREIGN KEY (context_id) REFERENCES fashion.conversational_contexts(id) ON DELETE CASCADE;


--
-- Name: refresh_tokens refresh_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: user_activity_events user_activity_events_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.user_activity_events
    ADD CONSTRAINT user_activity_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: admin_audit_logs admin_audit_logs_actor_user_id_fkey; Type: FK CONSTRAINT; Schema: shared; Owner: -
--

ALTER TABLE ONLY shared.admin_audit_logs
    ADD CONSTRAINT admin_audit_logs_actor_user_id_fkey FOREIGN KEY (actor_user_id) REFERENCES identity.users(id);


--
-- Name: deposit_transactions deposit_transactions_subscription_plan_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.deposit_transactions
    ADD CONSTRAINT deposit_transactions_subscription_plan_id_fkey FOREIGN KEY (subscription_plan_id) REFERENCES subscription.subscription_plans(id) ON DELETE SET NULL;


--
-- Name: deposit_transactions deposit_transactions_user_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.deposit_transactions
    ADD CONSTRAINT deposit_transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: campaign_claims fk_campaign_claims_user; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.campaign_claims
    ADD CONSTRAINT fk_campaign_claims_user FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: subscription_renewal_attempts subscription_renewal_attempts_expected_plan_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.subscription_renewal_attempts
    ADD CONSTRAINT subscription_renewal_attempts_expected_plan_id_fkey FOREIGN KEY (expected_plan_id) REFERENCES subscription.subscription_plans(id);


--
-- Name: subscription_renewal_attempts subscription_renewal_attempts_user_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.subscription_renewal_attempts
    ADD CONSTRAINT subscription_renewal_attempts_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: user_daily_quotas user_daily_quotas_user_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_daily_quotas
    ADD CONSTRAINT user_daily_quotas_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: user_subscription_events user_subscription_events_actor_admin_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscription_events
    ADD CONSTRAINT user_subscription_events_actor_admin_id_fkey FOREIGN KEY (actor_admin_id) REFERENCES identity.users(id);


--
-- Name: user_subscription_events user_subscription_events_source_deposit_transaction_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscription_events
    ADD CONSTRAINT user_subscription_events_source_deposit_transaction_id_fkey FOREIGN KEY (source_deposit_transaction_id) REFERENCES subscription.deposit_transactions(id);


--
-- Name: user_subscription_events user_subscription_events_user_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscription_events
    ADD CONSTRAINT user_subscription_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: user_subscriptions user_subscriptions_fallback_plan_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscriptions
    ADD CONSTRAINT user_subscriptions_fallback_plan_id_fkey FOREIGN KEY (fallback_plan_id) REFERENCES subscription.subscription_plans(id) ON DELETE RESTRICT;


--
-- Name: user_subscriptions user_subscriptions_subscription_plan_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscriptions
    ADD CONSTRAINT user_subscriptions_subscription_plan_id_fkey FOREIGN KEY (subscription_plan_id) REFERENCES subscription.subscription_plans(id) ON DELETE RESTRICT;


--
-- Name: user_subscriptions user_subscriptions_user_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_subscriptions
    ADD CONSTRAINT user_subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: user_wallets user_wallets_user_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.user_wallets
    ADD CONSTRAINT user_wallets_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: wallet_statements wallet_statements_reference_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.wallet_statements
    ADD CONSTRAINT wallet_statements_reference_id_fkey FOREIGN KEY (reference_id) REFERENCES subscription.deposit_transactions(id) ON DELETE SET NULL;


--
-- Name: wallet_statements wallet_statements_user_id_fkey; Type: FK CONSTRAINT; Schema: subscription; Owner: -
--

ALTER TABLE ONLY subscription.wallet_statements
    ADD CONSTRAINT wallet_statements_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: outfit_items outfit_items_outfit_id_fkey; Type: FK CONSTRAINT; Schema: wardrobe; Owner: -
--

ALTER TABLE ONLY wardrobe.outfit_items
    ADD CONSTRAINT outfit_items_outfit_id_fkey FOREIGN KEY (outfit_id) REFERENCES wardrobe.outfits(id) ON DELETE CASCADE;


--
-- Name: outfits outfits_user_id_fkey; Type: FK CONSTRAINT; Schema: wardrobe; Owner: -
--

ALTER TABLE ONLY wardrobe.outfits
    ADD CONSTRAINT outfits_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- Name: wardrobe_items wardrobe_items_user_id_fkey; Type: FK CONSTRAINT; Schema: wardrobe; Owner: -
--

ALTER TABLE ONLY wardrobe.wardrobe_items
    ADD CONSTRAINT wardrobe_items_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict c1h3x7FcUsZ0gAx1RSfnhfntobo5hLXXVY2w5hqWg9FIvKQ2KolqhfIGvB0d4Xs

