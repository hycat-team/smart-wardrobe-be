-- +goose Up
-- Sprint 1: Refund fields
ALTER TABLE deposit_transactions
    ADD COLUMN refunded_at TIMESTAMPTZ,
    ADD COLUMN refund_amount NUMERIC(12,2),
    ADD COLUMN refund_reason VARCHAR(255);

-- Sprint 1: Webhook inbox 'failed' status
ALTER TABLE provider_webhook_inbox
    DROP CONSTRAINT IF EXISTS provider_webhook_inbox_processing_status_check;
ALTER TABLE provider_webhook_inbox
    ADD CONSTRAINT provider_webhook_inbox_processing_status_check
        CHECK (processing_status IN ('received', 'processing', 'retry_required', 'processed', 'investigation_required', 'failed'));

-- Sprint 2: Wardrobe indexes
CREATE INDEX IF NOT EXISTS idx_wardrobe_items_user_status ON wardrobe_items(user_id, is_deleted, status);
CREATE INDEX IF NOT EXISTS idx_wardrobe_items_fashion_item ON wardrobe_items(fashion_item_id);
CREATE INDEX IF NOT EXISTS idx_wardrobe_items_type_status ON wardrobe_items(item_type, is_deleted);

-- Sprint 3: Outbox table
CREATE TABLE IF NOT EXISTS outbox_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    headers JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE INDEX idx_outbox_records_status ON outbox_records (status);
CREATE INDEX idx_outbox_records_created_at ON outbox_records (created_at);

-- +goose Down
DROP TABLE IF EXISTS outbox_records;

DROP INDEX IF EXISTS idx_wardrobe_items_type_status;
DROP INDEX IF EXISTS idx_wardrobe_items_fashion_item;
DROP INDEX IF EXISTS idx_wardrobe_items_user_status;

ALTER TABLE provider_webhook_inbox
    DROP CONSTRAINT IF EXISTS provider_webhook_inbox_processing_status_check;
ALTER TABLE provider_webhook_inbox
    ADD CONSTRAINT provider_webhook_inbox_processing_status_check
        CHECK (processing_status IN ('received', 'processing', 'retry_required', 'processed', 'investigation_required'));

ALTER TABLE deposit_transactions
    DROP COLUMN refunded_at,
    DROP COLUMN refund_amount,
    DROP COLUMN refund_reason;
