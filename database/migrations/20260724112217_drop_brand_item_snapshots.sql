-- +goose Up
DROP TABLE IF EXISTS dashboard_brand_item_daily_snapshots CASCADE;

-- +goose Down
CREATE TABLE dashboard_brand_item_daily_snapshots (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    brand_id uuid NOT NULL,
    item_id uuid NOT NULL,
    snapshot_date date NOT NULL,
    votes_count integer DEFAULT 0 NOT NULL,
    avg_rating numeric(3,2) DEFAULT 0 NOT NULL,
    feedback_sentiment character varying(20) DEFAULT 'positive'::character varying NOT NULL,
    top_paired_categories jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE dashboard_brand_item_daily_snapshots OWNER TO postgres;