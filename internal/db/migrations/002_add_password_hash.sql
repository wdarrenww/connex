-- +migrate Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';

-- +migrate Down
ALTER TABLE users DROP COLUMN IF EXISTS password_hash; 