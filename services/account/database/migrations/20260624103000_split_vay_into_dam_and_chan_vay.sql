-- +goose Up
UPDATE categories
SET name = 'Đầm',
    slug = 'dam',
    updated_at = NOW()
WHERE slug = 'vay';

INSERT INTO categories (id, name, slug, created_at, updated_at)
SELECT
    '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a09'::uuid,
    'Chân váy',
    'chan-vay',
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1
    FROM categories
    WHERE slug = 'chan-vay'
);

-- +goose Down
DELETE FROM categories
WHERE slug = 'chan-vay';

UPDATE categories
SET name = 'Váy',
    slug = 'vay',
    updated_at = NOW()
WHERE slug = 'dam';
