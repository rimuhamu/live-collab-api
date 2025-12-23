package auth

import "github.com/gin-gonic/gin"

type Service interface {
	GetUserIDFromGinContext(c *gin.Context) (int, error)
}
