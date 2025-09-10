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
	jwtSecret := cfg.JWTSecret

	authService := &auth.AuthService{
		DB:        database,
		JWTSecret: jwtSecret,
	}

	router := gin.Default()

	router.POST("/register", authService.Register)
	router.POST("/login", authService.Login)

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
