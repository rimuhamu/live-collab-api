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
// @Description Create a new document for the authenticated user. Requires valid JWT token.
// @Tags documents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateDocumentRequest true "Document data"
// @Success 201 {object} CreateDocumentResponse "Document created successfully"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /documents [post]
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
// @Summary Get document
// @Description Get a specific document. User can only access documents they own.
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Success 200 {object} DocumentResponse "Document details"
// @Failure 400 {object} ErrorResponse "Invalid document ID"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} ErrorResponse "Document not found or access denied"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /documents/{id} [get]
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

// Update godoc
// @Summary Update document
// @Description Update a document (title, content, or content type). User can only update documents they own.
// @Tags documents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Param request body UpdateDocumentRequest true "Document update data"
// @Success 200 {object} MessageResponse "Document updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} ErrorResponse "Document not found or access denied"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /documents/{id} [patch]
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

// Delete godoc
// @Summary Delete document
// @Description Delete a document. User can only delete documents they own. This action cannot be undone.
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Success 200 {object} MessageResponse "Document deleted successfully"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} ErrorResponse "Document not found or access denied"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /documents/{id} [delete]
func (dh *DocumentHandler) Delete(c *gin.Context) {
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

type CreateDocumentRequest struct {
	Title       string `json:"title" binding:"required" example:"My Document"`
	Content     string `json:"content" example:"Document content here"`
	ContentType string `json:"content_type" example:"text/plain"`
}

type UpdateDocumentRequest struct {
	Title       string `json:"title" binding:"required" example:"Updated Document"`
	Content     string `json:"content" example:"Updated content"`
	ContentType string `json:"content_type" example:"text/plain"`
}

type DocumentResponse struct {
	ID          int    `json:"id" example:"1"`
	Title       string `json:"title" example:"My Document"`
	Content     string `json:"content" example:"Document content"`
	ContentType string `json:"content_type" example:"text/plain"`
	OwnerID     int    `json:"owner_id" example:"1"`
	CreatedAt   string `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

type CreateDocumentResponse struct {
	Message     string `json:"message" example:"Document created successfully"`
	DocumentID  int    `json:"document_id" example:"1"`
	Title       string `json:"title" example:"My Document"`
	OwnerID     int    `json:"owner_id" example:"1"`
	Content     string `json:"content" example:"Document content"`
	ContentType string `json:"content_type" example:"text/plain"`
}

type DocumentListResponse struct {
	Documents []DocumentResponse `json:"documents"`
}

type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}
