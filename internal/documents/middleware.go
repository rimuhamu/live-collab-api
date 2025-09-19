package documents

import (
	"live-collab-api/internal/auth"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func DocumentAccessMiddleware(authService *auth.AuthService, docService *DocumentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := authService.GetUserIDFromGinContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		documentIdStr := c.Param("id")
		if documentIdStr == "" {
			documentIdStr = c.Param("document_id")
		}

		documentId, err := strconv.Atoi(documentIdStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
			c.Abort()
			return
		}

		hasAccess, err := docService.HasDocumentAccess(userId, documentId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check document access"})
			c.Abort()
			return
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied - you don't own this document"})
			c.Abort()
			return
		}

		c.Set("userId", userId)
		c.Set("documentId", documentId)

		c.Next()
	}
}

func GetDocumentID(c *gin.Context) (int, bool) {
	docID, exists := c.Get("documentId")
	if !exists {
		return 0, false
	}
	return docID.(int), true
}
