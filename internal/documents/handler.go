package documents

import (
	"database/sql"
	"errors"
	"fmt"
	"live-collab-api/internal/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	DB          *sql.DB
	AuthService *auth.AuthService
}

// Create godoc
// @Summary Create a new document
// @Description Create a new document for the authenticated user
// @Tags documents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object{title=string,content=string,content_type=string} true "Document data"
// @Success 201 {object} object{message=string,document_id=int,title=string,owner_id=int,content=string,content_type=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /documents [post]
func (h *DocumentHandler) Create(c *gin.Context) {
	var req struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Content     string `json:"content"`
		ContentType string `json:"content_type"`
		OwnerID     int    `json:"owner_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":  "Unauthorized",
			"detail": err.Error(),
		})
		return
	}

	var documentId int
	err = h.DB.QueryRow(
		"INSERT INTO documents (title, content, content_type, owner_id) VALUES ($1, $2, $3, $4) RETURNING id",
		req.Title, req.Content, req.ContentType, userId,
	).Scan(&documentId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create document"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Document created successfully",
		"document_id":  documentId,
		"title":        req.Title,
		"owner_id":     userId,
		"content":      req.Content,
		"content_type": req.ContentType,
	})
}

// GetByID godoc
// @Summary Get document by ID
// @Description Get a specific document by ID (must be owned by user)
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Success 200 {object} object{id=int,title=string,content=string,content_type=string,owner_id=int,created_at=string,updated_at=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /documents/{id} [get]
func (h *DocumentHandler) GetByID(c *gin.Context) {
	documentId := c.Param("id")

	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var doc struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Content     string `json:"content"`
		ContentType string `json:"content_type"`
		OwnerID     int    `json:"owner_id"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
	}

	err = h.DB.QueryRow("SELECT id, title, content, content_type, owner_id, created_at, updated_at FROM documents WHERE id = $1 AND owner_id = $2",
		documentId, userId,
	).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.ContentType, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}
	c.JSON(http.StatusOK, doc)
}

// GetAll godoc
// @Summary Get all user documents
// @Description Get all documents owned by the authenticated user
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{documents=[]object}
// @Failure 401 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /documents [get]
func (h *DocumentHandler) GetAll(c *gin.Context) {
	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	fmt.Printf("DEBUG: Looking for documents for user ID: %d\n", userId)

	rows, err := h.DB.Query("Select id, title, content, content_type, owner_id, created_at, updated_at FROM documents WHERE owner_id = $1", userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var documents []map[string]interface{}
	rowCount := 0
	for rows.Next() {
		rowCount++
		var doc struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			Content     string `json:"content"`
			ContentType string `json:"content_type"`
			OwnerID     int    `json:"owner_id"`
			CreatedAt   string `json:"created_at"`
			UpdatedAt   string `json:"updated_at"`
		}

		if err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.ContentType, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			fmt.Printf("DEBUG: Scan error: %v\n", err)
			continue
		}

		documents = append(documents, map[string]interface{}{
			"id":           doc.ID,
			"title":        doc.Title,
			"content":      doc.Content,
			"content_type": doc.ContentType,
			"owner_id":     doc.OwnerID,
			"created_at":   doc.CreatedAt,
			"updated_at":   doc.UpdatedAt,
		})
	}

	fmt.Printf("DEBUG: Total rows processed: %d\n", rowCount)
	if documents == nil {
		documents = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{"documents": documents})
}

// Update godoc
// @Summary Update document
// @Description Update a document (must be owned by user)
// @Tags documents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Param request body object{title=string,content=string,content_type=string} true "Document update data"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /documents/{id} [patch]
func (h *DocumentHandler) Update(c *gin.Context) {
	documentId := c.Param("id")
	var req struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		ContentType string `json:"content_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.DB.Exec("UPDATE documents SET title = $1, content = $2, content_type = $3 WHERE id = $4 AND owner_id = $5",
		req.Title, req.Content, req.ContentType, documentId, userId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found or not authorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document updated successfully"})
}

// Delete godoc
// @Summary Delete document
// @Description Delete a document (must be owned by user)
// @Tags documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Success 200 {object} object{message=string}
// @Failure 401 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /documents/{id} [delete]
func (h *DocumentHandler) Delete(c *gin.Context) {
	documentId := c.Param("id")

	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result, err := h.DB.Exec("DELETE FROM documents WHERE id = $1 AND owner_id = $2", documentId, userId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}
