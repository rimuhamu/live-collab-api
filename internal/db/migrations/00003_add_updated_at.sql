-- +goose Up
ALTER TABLE users
    ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();

ALTER TABLE documents
    ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();

ALTER TABLE events
    ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();