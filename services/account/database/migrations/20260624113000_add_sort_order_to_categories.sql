-- +goose Up
ALTER TABLE categories
ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;

UPDATE categories
SET sort_order = CASE slug
    WHEN 'ao' THEN 10
    WHEN 'quan' THEN 20
    WHEN 'chan-vay' THEN 30
    WHEN 'dam' THEN 40
    WHEN 'ao-khoac' THEN 50
    WHEN 'giay' THEN 60
    WHEN 'mu' THEN 70
    WHEN 'phu-kien' THEN 80
    WHEN 'other' THEN 90
    ELSE sort_order
END
WHERE slug IN ('ao', 'quan', 'chan-vay', 'dam', 'ao-khoac', 'giay', 'mu', 'phu-kien', 'other');

-- +goose Down
ALTER TABLE categories
DROP COLUMN IF EXISTS sort_order;
