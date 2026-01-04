package documents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"live-collab-api/internal/auth"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func setupDocumentTest(t *testing.T) (*DocumentHandler, sqlmock.Sqlmock, *gin.Engine, *auth.AuthService) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}

	authService := &auth.AuthService{
		DB:        db,
		JWTSecret: "test-secret",
	}

	documentService := &DocumentService{
		DB: db,
	}

	documentHandler := &DocumentHandler{
		DocumentService: documentService,
		AuthService:     authService,
	}

	r := gin.Default()
	return documentHandler, mock, r, authService
}

func TestCreateDocument_Success(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO documents (title, owner_id, content, content_type, created_at)")).
		WithArgs("My Test Document", userID, "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "content_type", "owner_id", "created_at"}).
			AddRow(1, "My Test Document", "", "text/plain", userID, "2025-01-04T10:00:00Z"))

	r.POST("/documents", handler.CreateDocument)

	payload := []byte(`{"title": "My Test Document"}`)
	req, _ := http.NewRequest("POST", "/documents", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["title"] != "My Test Document" {
		t.Errorf("Expected title 'My Test Document', got %v", response["title"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestCreateDocument_MissingTitle(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	r.POST("/documents", handler.CreateDocument)

	payload := []byte(`{}`)
	req, _ := http.NewRequest("POST", "/documents", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestCreateDocument_NoAuth(t *testing.T) {
	handler, mock, r, _ := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	r.POST("/documents", handler.CreateDocument)

	payload := []byte(`{"title": "Test Document"}`)
	req, _ := http.NewRequest("POST", "/documents", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestCreateDocument_WithContent(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	createdAt := "2025-01-04T10:00:00Z"
	expectedContent := "Initial content here"

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO documents (title, owner_id, content, content_type, created_at)")).
		WithArgs("Document with Content", userID, expectedContent).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "content_type", "owner_id", "created_at"}).
			AddRow(1, "Document with Content", expectedContent, "text/plain", userID, createdAt))

	r.POST("/documents", handler.CreateDocument)

	payload := []byte(`{"title": "Document with Content", "content": "Initial content here"}`)
	req, _ := http.NewRequest("POST", "/documents", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestGetDocument_Success(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, content, content_type, owner_id, created_at FROM documents WHERE id = $1")).
		WithArgs(documentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "content_type", "owner_id", "created_at"}).
			AddRow(documentID, "Test Document", "Content here", "text/plain", userID, "2025-01-04T10:00:00Z"))

	r.GET("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.GetDocument)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/documents/%d", documentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestGetDocument_InvalidID(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	r.GET("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.GetDocument)

	req, _ := http.NewRequest("GET", "/documents/invalid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	documentID := 999
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	r.GET("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.GetDocument)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/documents/%d", documentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestGetDocument_NoAccess(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	r.GET("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.GetDocument)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/documents/%d", documentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestGetDocument_AsCollaborator(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 2 // Collaborator, not owner
	ownerID := 1
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, content, content_type, owner_id, created_at FROM documents WHERE id = $1")).
		WithArgs(documentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "content_type", "owner_id", "created_at"}).
			AddRow(documentID, "Shared Document", "Content", "text/plain", ownerID, "2025-01-04T10:00:00Z"))

	r.GET("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.GetDocument)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/documents/%d", documentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestGetUserDocuments_Success(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	rows := sqlmock.NewRows([]string{"id", "title", "content", "content_type", "owner_id", "created_at"}).
		AddRow(1, "Document 1", "Content 1", "text/plain", userID, "2025-01-04T10:00:00Z").
		AddRow(2, "Document 2", "Content 2", "text/plain", userID, "2025-01-04T11:00:00Z")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT d.id, d.title, d.content, d.content_type, d.owner_id, d.created_at FROM documents d")).
		WithArgs(userID).
		WillReturnRows(rows)

	r.GET("/documents", handler.GetUserDocuments)

	req, _ := http.NewRequest("GET", "/documents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	documents := response["documents"].([]interface{})
	if len(documents) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(documents))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestGetUserDocuments_OwnedAndShared(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	otherUserID := 2
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	rows := sqlmock.NewRows([]string{"id", "title", "content", "content_type", "owner_id", "created_at"}).
		AddRow(1, "My Document", "Content", "text/plain", userID, "2025-01-04T10:00:00Z").
		AddRow(2, "Shared Document", "Content", "text/plain", otherUserID, "2025-01-04T11:00:00Z")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT d.id, d.title, d.content, d.content_type, d.owner_id, d.created_at FROM documents d")).
		WithArgs(userID).
		WillReturnRows(rows)

	r.GET("/documents", handler.GetUserDocuments)

	req, _ := http.NewRequest("GET", "/documents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	documents := response["documents"].([]interface{})
	if len(documents) != 2 {
		t.Errorf("Expected 2 documents (owned and shared), got %d", len(documents))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestUpdateDocument_Success(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE documents SET title = $1 WHERE id = $2")).
		WithArgs("Updated Title", documentID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r.PATCH("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.UpdateDocument)

	payload := []byte(`{"title": "Updated Title"}`)
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/documents/%d", documentID), bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestUpdateDocument_NoAuth(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	documentID := 1

	r.PATCH("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.UpdateDocument)

	payload := []byte(`{"title": "Updated Title"}`)
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/documents/%d", documentID), bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestUpdateDocument_NotOwner(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 2
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	r.PATCH("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.UpdateDocument)

	payload := []byte(`{"title": "Updated Title"}`)
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/documents/%d", documentID), bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestDeleteDocument_Success(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM events WHERE document_id = $1")).
		WithArgs(documentID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM document_collaborators WHERE document_id = $1")).
		WithArgs(documentID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM documents WHERE id = $1")).
		WithArgs(documentID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	r.DELETE("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.DeleteDocument)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/documents/%d", documentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestDeleteDocument_NoAuth(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	documentID := 1

	r.DELETE("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.DeleteDocument)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/documents/%d", documentID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestDeleteDocument_NotOwner(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 2
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	r.DELETE("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.DeleteDocument)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/documents/%d", documentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestDeleteDocument_NotFound(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	documentID := 999
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	r.DELETE("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.DeleteDocument)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/documents/%d", documentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestDeleteDocument_CascadeEvents(t *testing.T) {
	handler, mock, r, authService := setupDocumentTest(t)
	defer handler.DocumentService.DB.Close()

	userID := 1
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectBegin()

	// Delete events associated with the document
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM events WHERE document_id = $1")).
		WithArgs(documentID).
		WillReturnResult(sqlmock.NewResult(0, 3)) // 3 events deleted

	// Delete collaborators
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM document_collaborators WHERE document_id = $1")).
		WithArgs(documentID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Delete the document
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM documents WHERE id = $1")).
		WithArgs(documentID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	r.DELETE("/documents/:id", DocumentAccessMiddleware(authService, handler.DocumentService), handler.DeleteDocument)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/documents/%d", documentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}
