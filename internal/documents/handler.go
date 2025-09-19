package documents

import (
	"live-collab-api/internal/auth"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	DocumentService *DocumentService
	AuthService     *auth.AuthService
}

// CreateDocument godoc
// @Summary Create a new document
// @Description Create a new document for collaborative editing. Optionally include initial content that will be tracked as the first edit event.
// @Tags documents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateDocumentRequest true "Document creation data"
// @Success 201 {object} DocumentResponse "Document created successfully"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing JWT token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/documents [post]
func (dh *DocumentHandler) CreateDocument(c *gin.Context) {
	userID, err := dh.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	document, err := dh.DocumentService.CreateDocument(req.Title, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create document"})
		return
	}

	c.JSON(http.StatusCreated, document)
}

// GetDocument godoc
// @Summary Get document by ID
// @Description Retrieve a specific document by its ID. User can only access documents they own.
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Success 200 {object} DocumentResponse "Document details"
// @Failure 400 {object} ErrorResponse "Invalid document ID"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing JWT token"
// @Failure 403 {object} ErrorResponse "Access denied - you don't own this document"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/documents/{id} [get]
func (dh *DocumentHandler) GetDocument(c *gin.Context) {
	userId, err := dh.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	documentIdStr := c.Param("id")
	documentId, err := strconv.Atoi(documentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	hasAccess, err := dh.DocumentService.HasDocumentAccess(userId, documentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access to document"})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access forbidden"})
		return
	}

	document, err := dh.DocumentService.GetDocument(documentId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, document)
}

// GetUserDocuments godoc
// @Summary Get all user documents
// @Description Retrieve all documents owned by the authenticated user, ordered by creation date (newest first)
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DocumentListResponse "List of user documents"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing JWT token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/documents [get]
func (dh *DocumentHandler) GetUserDocuments(c *gin.Context) {
	userId, err := dh.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	documents, err := dh.DocumentService.GetUserDocuments(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get documents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"documents": documents})
}

// UpdateDocument godoc
// @Summary Update document title
// @Description Update a document's title. User can only update documents they own. Content updates should be done via WebSocket for real-time collaboration.
// @Tags documents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Param request body UpdateDocumentRequest true "Document update data"
// @Success 200 {object} MessageResponse "Document updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid input data or document ID"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing JWT token"
// @Failure 403 {object} ErrorResponse "Access denied - you don't own this document"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/documents/{id} [put]
func (dh *DocumentHandler) UpdateDocument(c *gin.Context) {
	userId, err := dh.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	documentIdStr := c.Param("id")
	documentId, err := strconv.Atoi(documentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	hasAccess, err := dh.DocumentService.HasDocumentAccess(userId, documentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access to document"})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access forbidden"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dh.DocumentService.UpdateDocumentTitle(documentId, req.Title); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document updated successfully"})
}

// DeleteDocument godoc
// @Summary Delete document
// @Description Delete a document and all its associated events. User can only delete documents they own. This action cannot be undone.
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Success 200 {object} MessageResponse "Document deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid document ID"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing JWT token"
// @Failure 403 {object} ErrorResponse "Access denied - you don't own this document"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/documents/{id} [delete]
func (dh *DocumentHandler) DeleteDocument(c *gin.Context) {
	userId, err := dh.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	documentIdStr := c.Param("id")
	documentId, err := strconv.Atoi(documentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	hasAccess, err := dh.DocumentService.HasDocumentAccess(userId, documentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access to document"})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access forbidden"})
		return
	}

	if err := dh.DocumentService.DeleteDocument(documentId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}

// GetDocumentEvents godoc
// @Summary Get document edit events
// @Description Retrieve edit events for a specific document with optional pagination. User can only access events for documents they own. Events are ordered by creation date (newest first).
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Param limit query int false "Maximum number of events to return (default: 100, max: 1000)" default(100)
// @Success 200 {object} EventListResponse "List of document events"
// @Failure 400 {object} ErrorResponse "Invalid document ID or parameters"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing JWT token"
// @Failure 403 {object} ErrorResponse "Access denied - you don't own this document"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/documents/{id}/events [get]
func (dh *DocumentHandler) GetDocumentEvents(c *gin.Context) {
	userId, err := dh.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	documentIdStr := c.Param("id")
	documentId, err := strconv.Atoi(documentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	hasAccess, err := dh.DocumentService.HasDocumentAccess(userId, documentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access to document"})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access forbidden"})
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	events, err := dh.DocumentService.GetDocumentEvents(documentId, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get document events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// swagger models for documents

// CreateDocumentRequest represents the request body for creating a document
type CreateDocumentRequest struct {
	Title   string `json:"title" binding:"required" example:"My Collaborative Document"`
	Content string `json:"content" example:"Initial content for the document"`
}

// UpdateDocumentRequest represents the request body for updating a document
type UpdateDocumentRequest struct {
	Title string `json:"title" binding:"required" example:"Updated Document Title"`
}

// DocumentResponse represents a document in API responses
type DocumentResponse struct {
	ID          int    `json:"id" example:"1"`
	Title       string `json:"title" example:"My Collaborative Document"`
	Content     string `json:"content" example:"Document content here"`
	ContentType string `json:"content_type" example:"text/plain"`
	OwnerID     int    `json:"owner_id" example:"1"`
	CreatedAt   string `json:"created_at" example:"2025-09-19T10:30:00Z"`
}

// DocumentListResponse represents a list of documents
type DocumentListResponse struct {
	Documents []DocumentResponse `json:"documents"`
}

// EventResponse represents a document event in API responses
type EventResponse struct {
	ID         int                    `json:"id" example:"1"`
	DocumentID int                    `json:"document_id" example:"1"`
	UserID     int                    `json:"user_id" example:"1"`
	EventType  string                 `json:"event_type" example:"edit"`
	Payload    map[string]interface{} `json:"payload" example:"{\"type\":\"edit\",\"version\":1,\"payload\":{\"op\":\"insert\",\"pos\":0,\"content\":\"Hello\"}}"`
	CreatedAt  string                 `json:"created_at" example:"2025-09-19T10:30:00Z"`
}

// EventListResponse represents a list of document events
type EventListResponse struct {
	Events []EventResponse `json:"events"`
	Count  int             `json:"count" example:"10"`
	Limit  int             `json:"limit" example:"100"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Error message describing what went wrong"`
}
