-- +goose Up

CREATE TABLE IF NOT EXISTS loyalty_point_lots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    loyalty_account_id UUID NOT NULL REFERENCES loyalty_accounts(id) ON DELETE CASCADE,
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    brand_customer_id UUID NOT NULL REFERENCES brand_customers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    earn_transaction_id UUID NOT NULL REFERENCES loyalty_point_transactions(id) ON DELETE RESTRICT,

    earned_points INT NOT NULL,
    remaining_points INT NOT NULL,

    expires_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_loyalty_point_lots_earned_positive
        CHECK (earned_points > 0),

    CONSTRAINT chk_loyalty_point_lots_remaining_non_negative
        CHECK (remaining_points >= 0),

    CONSTRAINT chk_loyalty_point_lots_remaining_lte_earned
        CHECK (remaining_points <= earned_points),

    CONSTRAINT chk_loyalty_point_lots_status
        CHECK (status IN ('ACTIVE', 'CONSUMED', 'EXPIRED'))
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_loyalty_point_lots_earn_transaction
ON loyalty_point_lots(earn_transaction_id);

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_account_active_expiry
ON loyalty_point_lots(loyalty_account_id, expires_at)
WHERE status = 'ACTIVE' AND remaining_points > 0;

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_expiry_worker
ON loyalty_point_lots(expires_at)
WHERE status = 'ACTIVE' AND remaining_points > 0 AND expires_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_brand_customer
ON loyalty_point_lots(brand_customer_id);

INSERT INTO loyalty_point_lots (
    loyalty_account_id,
    brand_id,
    brand_customer_id,
    user_id,
    earn_transaction_id,
    earned_points,
    remaining_points,
    expires_at,
    status,
    created_at,
    updated_at
)
SELECT
    t.loyalty_account_id,
    t.brand_id,
    t.brand_customer_id,
    t.user_id,
    t.id,
    t.points_delta,
    t.points_delta,
    t.expires_at,
    CASE
        WHEN t.expires_at IS NOT NULL AND t.expires_at <= NOW() THEN 'EXPIRED'
        ELSE 'ACTIVE'
    END,
    t.created_at,
    NOW()
FROM loyalty_point_transactions t
WHERE t.transaction_type = 'EARN'
  AND t.points_delta > 0
  AND NOT EXISTS (
      SELECT 1
      FROM loyalty_point_lots l
      WHERE l.earn_transaction_id = t.id
  );

-- +goose Down

DROP INDEX IF EXISTS idx_loyalty_point_lots_brand_customer;
DROP INDEX IF EXISTS idx_loyalty_point_lots_expiry_worker;
DROP INDEX IF EXISTS idx_loyalty_point_lots_account_active_expiry;
DROP INDEX IF EXISTS ux_loyalty_point_lots_earn_transaction;

DROP TABLE IF EXISTS loyalty_point_lots;
