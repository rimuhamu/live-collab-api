package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *AuthService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := s.GetUserIDFromGinContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "detail": "Invalid or missing authentication token"})
			c.Abort()
			return
		}

		c.Set("userId", userId)
		c.Next()
	}
}
