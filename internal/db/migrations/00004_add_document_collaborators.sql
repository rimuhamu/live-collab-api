-- +goose Up
-- 00004_add_document_collaborators.sql
CREATE TABLE IF NOT EXISTS document_collaborators(
    id SERIAL PRIMARY KEY,
    document_id INT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(20) DEFAULT 'edit' CHECK (permission IN ('view', 'edit')),
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(document_id, user_id)
);

CREATE INDEX idx_document_collaborators_doc ON document_collaborators(document_id);
CREATE INDEX idx_document_collaborators_user ON document_collaborators(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_document_collaborators_user;
DROP INDEX IF EXISTS idx_document_collaborators_doc;
DROP TABLE IF EXISTS document_collaborators;