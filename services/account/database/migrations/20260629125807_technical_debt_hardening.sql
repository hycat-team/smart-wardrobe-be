-- +goose Up
ALTER TABLE brand_members DROP CONSTRAINT IF EXISTS chk_brand_members_role;

UPDATE brand_members
SET role = 'staff'
WHERE role IN ('manager', 'support_staff', 'marketer');

ALTER TABLE brand_members
    ADD CONSTRAINT chk_brand_members_role
    CHECK (role IN ('owner', 'staff'));

CREATE UNIQUE INDEX IF NOT EXISTS ux_brand_members_one_active_owner
ON brand_members(brand_id)
WHERE role = 'owner' AND status = 'active';

ALTER TABLE brand_customer_claims
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS revoked_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS revoked_reason VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_brand_customer_claims_active_customer
ON brand_customer_claims(brand_customer_id)
WHERE consumed_at IS NULL AND revoked_at IS NULL;

ALTER TABLE brand_conversations
    ADD COLUMN IF NOT EXISTS user_last_read_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS staff_last_read_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS closed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_brand_conversation_messages_unread
ON brand_conversation_messages(conversation_id, sender_role, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_brand_conversation_messages_unread;

ALTER TABLE brand_conversations
    DROP COLUMN IF EXISTS closed_by_user_id,
    DROP COLUMN IF EXISTS closed_at,
    DROP COLUMN IF EXISTS staff_last_read_at,
    DROP COLUMN IF EXISTS user_last_read_at;

DROP INDEX IF EXISTS idx_brand_customer_claims_active_customer;

ALTER TABLE brand_customer_claims
    DROP COLUMN IF EXISTS revoked_reason,
    DROP COLUMN IF EXISTS revoked_by_user_id,
    DROP COLUMN IF EXISTS revoked_at;

DROP INDEX IF EXISTS ux_brand_members_one_active_owner;

ALTER TABLE brand_members DROP CONSTRAINT IF EXISTS chk_brand_members_role;

ALTER TABLE brand_members
    ADD CONSTRAINT chk_brand_members_role
    CHECK (role IN ('owner', 'manager', 'support_staff', 'marketer'));
