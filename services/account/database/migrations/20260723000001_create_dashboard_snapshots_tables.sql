-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.dashboard_admin_daily_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_date DATE NOT NULL UNIQUE,
    total_users INT NOT NULL DEFAULT 0,
    new_users INT NOT NULL DEFAULT 0,
    active_users_dau INT NOT NULL DEFAULT 0,
    total_brands INT NOT NULL DEFAULT 0,
    pending_review_brands INT NOT NULL DEFAULT 0,
    ai_requests_count BIGINT NOT NULL DEFAULT 0,
    ai_error_count BIGINT NOT NULL DEFAULT 0,
    ai_paid_tokens BIGINT NOT NULL DEFAULT 0,
    ai_actual_cost_micro_vnd BIGINT NOT NULL DEFAULT 0,
    subscription_revenue_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    net_ai_margin_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    payos_transaction_count INT NOT NULL DEFAULT 0,
    payos_success_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_snapshots_date ON public.dashboard_admin_daily_snapshots(snapshot_date DESC);

CREATE TABLE IF NOT EXISTS public.dashboard_brand_daily_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES public.brands(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    total_members INT NOT NULL DEFAULT 0,
    points_issued INT NOT NULL DEFAULT 0,
    points_redeemed INT NOT NULL DEFAULT 0,
    points_expiring_30d INT NOT NULL DEFAULT 0,
    points_liability_value_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    active_vouchers INT NOT NULL DEFAULT 0,
    vouchers_redeemed INT NOT NULL DEFAULT 0,
    digital_sample_votes_count INT NOT NULL DEFAULT 0,
    digital_sample_avg_rating NUMERIC(3, 2) NOT NULL DEFAULT 0.00,
    open_cs_tickets INT NOT NULL DEFAULT 0,
    cs_first_response_avg_seconds INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_brand_daily_snapshot UNIQUE (brand_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_brand_snapshots_brand_date ON public.dashboard_brand_daily_snapshots(brand_id, snapshot_date DESC);

CREATE TABLE IF NOT EXISTS public.dashboard_brand_item_daily_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES public.brands(id) ON DELETE CASCADE,
    item_id UUID NOT NULL,
    snapshot_date DATE NOT NULL,
    votes_count INT NOT NULL DEFAULT 0,
    avg_rating NUMERIC(3, 2) NOT NULL DEFAULT 0.00,
    feedback_sentiment VARCHAR(20) NOT NULL DEFAULT 'positive',
    top_paired_categories JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_brand_item_daily_snapshot UNIQUE (brand_id, item_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_brand_item_snapshots ON public.dashboard_brand_item_daily_snapshots(brand_id, item_id, snapshot_date DESC);

CREATE TABLE IF NOT EXISTS public.dashboard_user_wardrobe_summaries (
    user_id UUID PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    total_items INT NOT NULL DEFAULT 0,
    total_wardrobe_value_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    average_cost_per_wear_vnd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    underutilized_items_count INT NOT NULL DEFAULT 0,
    underutilized_items JSONB NOT NULL DEFAULT '[]'::jsonb,
    ai_quota_remaining INT NOT NULL DEFAULT 0,
    expiring_vouchers_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_wardrobe_summaries_updated ON public.dashboard_user_wardrobe_summaries(updated_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.dashboard_user_wardrobe_summaries;
DROP TABLE IF EXISTS public.dashboard_brand_item_daily_snapshots;
DROP TABLE IF EXISTS public.dashboard_brand_daily_snapshots;
DROP TABLE IF EXISTS public.dashboard_admin_daily_snapshots;
-- +goose StatementEnd
