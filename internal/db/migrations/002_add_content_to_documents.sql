-- Add content and content_type columns to existing documents table
ALTER TABLE documents
    ADD COLUMN content TEXT,
    ADD COLUMN content_type VARCHAR(50) DEFAULT 'text/plain';