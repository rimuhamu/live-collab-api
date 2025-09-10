package main

import (
	"live-collab-api/internal/auth"
	"live-collab-api/internal/config"
	"live-collab-api/internal/db"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()
	database := db.Connect(cfg.DBUrl)

	authHandler := &auth.AuthHandler{DB: database, JWTSecret: cfg.JWTSecret}

	router := gin.Default()
	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)

	log.Println("Server running on :8080")
	http.ListenAndServe(":8080", router)
}
