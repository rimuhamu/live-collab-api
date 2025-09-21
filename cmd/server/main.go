package main

import (
	"live-collab-api/internal/auth"
	"live-collab-api/internal/config"
	"live-collab-api/internal/db"
	"live-collab-api/internal/documents"
	"live-collab-api/internal/events"
	"live-collab-api/internal/websocket"
	"log"
	"net/http"

	_ "live-collab-api/docs"

	"github.com/gin-contrib/cors"
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

	documentService := &documents.DocumentService{
		DB: database,
	}

	documentsHandler := &documents.DocumentHandler{
		DocumentService: documentService,
		AuthService:     authService,
	}

	eventsHandler := &events.EventHandler{
		DB:          database,
		AuthService: authService,
	}

	hub := websocket.NewHub()
	go hub.Run()

	wsService := &websocket.WebSocketHandler{
		Hub:         hub,
		DB:          database,
		AuthService: authService,
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendUrl, "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

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

	protected := router.Group("/api")
	protected.Use(authService.AuthMiddleware())
	{
		protected.GET("/me", authService.Me)

		protected.POST("/documents", documentsHandler.CreateDocument)
		protected.GET("/documents/:id", documentsHandler.GetUserDocuments)
		protected.GET("/documents/:id", documentsHandler.GetDocumentEvents)

		protected.POST("/documents/:id/events", eventsHandler.CreateDocumentEvent)
		protected.GET("/documents/:id/events", eventsHandler.GetDocumentEvents)

		docAccess := protected.Group("")
		docAccess.Use(documents.DocumentAccessMiddleware(authService, documentService))
		{
			protected.GET("/documents", documentsHandler.GetDocument)
			protected.PATCH("/documents/:id", documentsHandler.UpdateDocument)
			protected.DELETE("/documents/:id", documentsHandler.DeleteDocument)
		}

	}

	router.GET("/ws/:document_id", wsService.HandleWebSocket)

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
