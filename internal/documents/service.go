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
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	OwnerId     int    `json:"owner_id"`
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

type Collaborator struct {
	ID         int    `json:"id"`
	DocumentID int    `json:"document_id"`
	UserID     int    `json:"user_id"`
	Email      string `json:"email"`
	Permission string `json:"permission"`
	CreatedAt  string `json:"created_at"`
}

func (ds *DocumentService) CreateDocument(title string, ownerId int) (*Document, error) {
	var doc Document
	err := ds.DB.QueryRow(`
		INSERT INTO documents (title, owner_id, content, content_type, created_at)
		VALUES ($1, $2, '', 'text/plain', now())
		RETURNING id, title, content, content_type, owner_id, created_at
	`, title, ownerId).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.ContentType, &doc.OwnerId, &doc.CreatedAt)

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

func (ds *DocumentService) GetUserDocuments(userId int) ([]Document, error) {
	rows, err := ds.DB.Query(`
		SELECT DISTINCT d.id, d.title, d.content, d.content_type, d.owner_id, d.created_at
		FROM documents d
		LEFT JOIN document_collaborators dc ON d.id = dc.document_id
		WHERE d.owner_id = $1 OR dc.user_id = $1
		ORDER BY d.created_at DESC`, userId)
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

func (ds *DocumentService) UpdateDocumentTitle(documentId int, title string) error {
	result, err := ds.DB.Exec("UPDATE documents SET title = $1 WHERE id = $2", title, documentId)
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
		return fmt.Errorf("failed to delete events from document: %v", err)
	}

	_, err = tx.Exec("DELETE FROM document_collaborators WHERE document_id = $1", documentId)
	if err != nil {
		return fmt.Errorf("failed to delete collaborators from document: %v", err)
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

func (ds *DocumentService) HasDocumentAccess(userId, documentId int) (bool, error) {
	var hasAccess bool
	// Check if the user is the owner or collaborator of the document
	err := ds.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM documents WHERE id = $1 AND owner_id = $2
			UNION
			SELECT 1 FROM document_collaborators WHERE document_id = $1 AND user_id = $2
		)
	`, documentId, userId).Scan(&hasAccess)

	if err != nil {
		return false, fmt.Errorf("failed to check document access: %v", err)
	}

	return hasAccess, nil
}

func (ds *DocumentService) IsDocumentOwner(userId, documentId int) (bool, error) {
	var ownerId int
	err := ds.DB.QueryRow("SELECT owner_id FROM documents WHERE id = $1", documentId).Scan(&ownerId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check document ownership: %v", err)
	}

	return ownerId == userId, nil
}

func (ds *DocumentService) AddCollaborator(documentId, userId int, permission string) error {
	if permission != "view" && permission != "edit" {
		return fmt.Errorf("invalid permission: must be 'view' or 'edit'")
	}

	_, err := ds.DB.Exec(`
		INSERT INTO document_collaborators (document_id, user_id, permission)
		VALUES ($1, $2, $3)
		ON CONFLICT (document_id, user_id) 
		DO UPDATE SET permission = $3
	`, documentId, userId, permission)

	if err != nil {
		return fmt.Errorf("failed to add collaborator: %v", err)
	}

	return nil
}

func (ds *DocumentService) RemoveCollaborator(documentId, userId int) error {
	result, err := ds.DB.Exec(`
		DELETE FROM document_collaborators 
		WHERE document_id = $1 AND user_id = $2
	`, documentId, userId)

	if err != nil {
		return fmt.Errorf("failed to remove collaborator: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("collaborator not found")
	}

	return nil
}

func (ds *DocumentService) GetCollaborators(documentId int) ([]Collaborator, error) {
	rows, err := ds.DB.Query(`
		SELECT dc.id, dc.document_id, dc.user_id, u.email, dc.permission, dc.created_at
		FROM document_collaborators dc
		JOIN users u ON dc.user_id = u.id
		WHERE dc.document_id = $1
		ORDER BY dc.created_at DESC
	`, documentId)

	if err != nil {
		return nil, fmt.Errorf("failed to get collaborators: %v", err)
	}
	defer rows.Close()

	var collaborators []Collaborator
	for rows.Next() {
		var collab Collaborator
		if err := rows.Scan(&collab.ID, &collab.DocumentID, &collab.UserID, &collab.Email, &collab.Permission, &collab.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan collaborator: %v", err)
		}
		collaborators = append(collaborators, collab)
	}

	return collaborators, nil
}

func (ds *DocumentService) GetCollaboratorPermission(documentId, userId int) (string, error) {
	var permission string
	err := ds.DB.QueryRow(`
		SELECT permission FROM document_collaborators 
		WHERE document_id = $1 AND user_id = $2
	`, documentId, userId).Scan(&permission)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get permission: %v", err)
	}

	return permission, nil
}
