package main

import (
	"live-collab-api/internal/auth"
	"live-collab-api/internal/config"
	"live-collab-api/internal/db"
	"live-collab-api/internal/documents"
	"live-collab-api/internal/events"
	"log"
	"net/http"

	_ "live-collab-api/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Live Collaboration API
// @version 1.0
// @description A collaborative document editing API with real-time features
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	cfg := config.LoadConfig()
	database := db.Connect(cfg.DBUrl)
	jwtSecret := cfg.JWTSecret

	authService := &auth.AuthService{
		DB:        database,
		JWTSecret: jwtSecret,
	}

	documentsHandler := &documents.DocumentHandler{
		DB:          database,
		AuthService: authService,
	}

	eventHandler := &events.EventHandler{
		DB:          database,
		AuthService: authService,
	}

	router := gin.Default()

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check endpoint
	// @Summary Health check
	// @Description Check if the API is running
	// @Tags health
	// @Produce json
	// @Success 200 {object} object{status=string}
	// @Router /health [get]
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/register", authService.Register)
	router.POST("/login", authService.Login)

	protected := router.Group("/")
	protected.Use(authService.AuthMiddleware())
	{
		router.POST("/documents", documentsHandler.Create)
		router.GET("/documents", documentsHandler.GetAll)
		router.GET("/documents/:id", documentsHandler.GetByID)
		router.PATCH("/documents/:id", documentsHandler.Update)
		router.DELETE("/documents/:id", documentsHandler.Delete)

		router.GET("/documents/:id/events", eventHandler.GetDocumentEvents)
		router.POST("/documents/:id/events", eventHandler.CreateDocumentEvent)
	}

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
