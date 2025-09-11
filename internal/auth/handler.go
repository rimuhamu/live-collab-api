package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Register godoc
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User registration data"
// @Success 201 {object} MessageResponse "User created successfully"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 409 {object} ErrorResponse "User already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /register [post]
func (s *AuthService) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password hashing failed"})
		return
	}

	_, err = s.DB.Exec("INSERT INTO users (email, password) VALUES ($1, $2)", req.Email, hash)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT token for accessing protected endpoints
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User login credentials"
// @Success 200 {object} LoginResponse "Login successful with JWT token"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 401 {object} ErrorResponse "Invalid credentials"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /login [post]
func (s *AuthService) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int
	var hash string
	err := s.DB.QueryRow("SELECT id, password FROM users WHERE email = $1", req.Email).Scan(&id, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	if !CheckPasswordHash(req.Password, hash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := GenerateJWT(id, s.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"user_id": id,
	})

}

// Swagger models for auth endpoints

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type LoginResponse struct {
	Token  string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MzY5OTc4NTYsInVzZXJfaWQiOjF9.Xg2Lv8K3oPHx9vXzF2dA1kT7mN8qR5wE"`
	UserID int    `json:"user_id" example:"1"`
}

type MessageResponse struct {
	Message string `json:"message" example:"User created successfully"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
}
