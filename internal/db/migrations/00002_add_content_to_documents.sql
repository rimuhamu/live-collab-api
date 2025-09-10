-- +goose Up
ALTER TABLE documents
    ADD COLUMN content TEXT,
    ADD COLUMN content_type VARCHAR(50) DEFAULT 'text/plain';

-- +goose Down
ALTER TABLE documents
    DROP COLUMN IF EXISTS content,
    DROP COLUMN IF EXISTS content_type;