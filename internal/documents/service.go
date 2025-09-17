package documents

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type DocumentService struct {
	DB *sql.DB
}

type Document struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	OwnerId     string `json:"owner_id"`
	CreatedAt   string `json:"created_at"`
}

type Event struct {
	ID         string                 `json:"id"`
	DocumentId string                 `json:"document_id"`
	UserId     string                 `json:"user_id"`
	EventType  string                 `json:"event_type"`
	Payload    map[string]interface{} `json:"payload"`
	CreatedAt  string                 `json:"created_at"`
}

func (ds *DocumentService) CreateDocument(title string, content string, contentType string, ownerId int) (*Document, error) {
	var doc Document
	err := ds.DB.QueryRow(`
		INSERT INTO documents (title, content, content_type, owner_id, created_at)
		VALUES ($1, $2, $3, $4, now())
		RETURNING id, title, content, content_type, owner_id, created_at
`, title, content, contentType, ownerId).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.ContentType, &doc.OwnerId, &doc.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("error creating document: %v", err)
	}
	return &doc, nil
}

func (ds *DocumentService) GetDocument(documentId int) (*Document, error) {
	var doc Document
	err := ds.DB.QueryRow(`
		SELECT id, title, content, content_type, owner_id, created_at
		FROM documents WHERE id = $1`, documentId).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.ContentType, &doc.OwnerId, &doc.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("error getting document: %v", err)
	}

	return &doc, nil
}

func (ds *DocumentService) GetUserDocuments(userId string) ([]Document, error) {
	rows, err := ds.DB.Query(`
		SELECT id, title, content, content_type, owner_id, created_at
		FROM documents WHERE owner_id = $1 ORDER BY created_at DESC`, userId)
	if err != nil {
		return nil, fmt.Errorf("error getting user documents: %v", err)
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.ContentType, &doc.OwnerId, &doc.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan document: %v", err)
		}
		documents = append(documents, doc)
	}
	return documents, nil
}

func (ds *DocumentService) UpdateDocument(documentId int, title string, content string) error {
	result, err := ds.DB.Exec("UPDATE documents SET title = $1, content = $2 WHERE id = $3", title, content, documentId)
	if err != nil {
		return fmt.Errorf("error updating document: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no document with id %v has been updated", documentId)
	}

	return nil
}

func (ds *DocumentService) DeleteDocument(documentId int) error {
	tx, err := ds.DB.Begin()
	if err != nil {
		return fmt.Errorf("error beginning transaction: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM events WHERE document_id = $1", documentId)
	if err != nil {
		return fmt.Errorf("failed to events from document: %v", err)
	}

	result, err := tx.Exec("DELETE FROM documents WHERE id = $1", documentId)
	if err != nil {
		return fmt.Errorf("failed to delete document: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no document with id %v has been deleted", documentId)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

func (ds *DocumentService) GetDocumentEvents(documentId int, limit int) ([]Event, error) {
	rows, err := ds.DB.Query(`
		SELECT id, document_id, user_id, event_type, payload, created_at
		FROM events WHERE document_id = $1
		ORDER BY created_at DESC LIMIT $2
	`, documentId, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to query events: %v", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		var payload []byte

		if err := rows.Scan(&event.ID, &event.DocumentId, &event.UserId, &event.EventType, &payload, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %v", err)
		}

		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event payload: %v", err)
		}

		events = append(events, event)
	}
	return events, nil
}

func (ds *DocumentService) HasDocumentAccess(userId, DocumentId int) (bool, error) {
	var ownerId int
	err := ds.DB.QueryRow("SELECT owner_id FROM documents WHERE id = $1", DocumentId).Scan(&ownerId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check document access: %v", err)
	}

	// TODO: add collaboration
	return ownerId == userId, nil
}
