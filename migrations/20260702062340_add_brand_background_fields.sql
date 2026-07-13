-- +goose Up
ALTER TABLE brands ADD COLUMN background_url VARCHAR(500);
ALTER TABLE brands ADD COLUMN background_public_id VARCHAR(255);

-- +goose Down
ALTER TABLE brands DROP COLUMN background_url;
ALTER TABLE brands DROP COLUMN background_public_id;
