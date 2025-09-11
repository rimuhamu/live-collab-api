package main

import (
	"live-collab-api/internal/auth"
	"live-collab-api/internal/config"
	"live-collab-api/internal/db"
	"live-collab-api/internal/documents"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

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

	router := gin.Default()

	router.POST("/register", authService.Register)
	router.POST("/login", authService.Login)

	router.POST("/documents", documentsHandler.Create)
	router.GET("/documents", documentsHandler.GetAll)
	router.GET("/documents/:id", documentsHandler.GetByID)
	router.PATCH("/documents/:id", documentsHandler.Update)
	router.DELETE("/documents/:id", documentsHandler.Delete)

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
