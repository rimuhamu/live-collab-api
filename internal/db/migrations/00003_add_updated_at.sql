-- +goose Up
ALTER TABLE users
    ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();

ALTER TABLE documents
    ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();

ALTER TABLE events
    ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE documents
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE events
    DROP COLUMN IF EXISTS updated_at;