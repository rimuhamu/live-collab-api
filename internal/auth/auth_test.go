package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func setupTest(t *testing.T) (*AuthService, sqlmock.Sqlmock, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}

	authService := &AuthService{
		DB:        db,
		JWTSecret: "test-secret",
	}

	r := gin.Default()
	return authService, mock, r
}

func TestHashAndCheckPassword(t *testing.T) {
	password := "secret123"

	hashedPassword, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Error hashing password: %v", err)
	}

	if hashedPassword == password {
		t.Error("Hashed password should not be equal to itself")
	}

	if !CheckPasswordHash(password, hashedPassword) {
		t.Error("Password check failed")
	}

	if CheckPasswordHash("wrongpassword", hashedPassword) {
		t.Error("Wrong password check passed")
	}
}

func TestJWTToken(t *testing.T) {
	authService := &AuthService{JWTSecret: "test-secret"}
	userID := 123

	token, err := GenerateJWT(userID, authService.JWTSecret)
	if err != nil {
		t.Fatalf("Error generating JWT token: %v", err)
	}
	if token == "" {
		t.Fatal("Generated token is empty")
	}

	parsedID, err := authService.GetUserIDFromToken(token)
	if err != nil {
		t.Errorf("Error getting user id from token: %v", err)
	}
	if parsedID != userID {
		t.Errorf("Wrong user id. Expected %d, got %d", userID, parsedID)
	}
}

func TestRegister_Success(t *testing.T) {
	authService, mock, r := setupTest(t)
	defer authService.DB.Close()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (email, password) VALUES ($1, $2)")).
		WithArgs("test@example.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	r.POST("/register", authService.Register)

	payload := []byte(`{"email": "test@example.com", "password": "password123"}`)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	authService, mock, r := setupTest(t)
	defer authService.DB.Close()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users")).
		WillReturnError(errors.New("duplicate key value violates unique constraint"))

	r.POST("/register", authService.Register)

	payload := []byte(`{"email": "existing@example.com", "password": "password123"}`)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(payload))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate user, got %d", w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	authService, mock, r := setupTest(t)
	defer authService.DB.Close()

	password := "password123"
	hashedPassword, _ := HashPassword(password)
	userID := 1

	rows := sqlmock.NewRows([]string{"id", "password"}).AddRow(userID, hashedPassword)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, password FROM users WHERE email = $1")).
		WithArgs("user@example.com").
		WillReturnRows(rows)

	r.POST("/login", authService.Login)

	payload := []byte(`{"email": "user@example.com", "password": "password123"}`)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, ok := response["token"]; !ok {
		t.Error("Response should contain token")
	}

	if id, ok := response["user_id"].(float64); !ok || int(id) != userID {
		t.Errorf("Wrong user id. Expected %d, got %d", userID, response["user_id"])
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	authService, mock, r := setupTest(t)
	defer authService.DB.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, password FROM users")).
		WithArgs("wrong@example.com").
		WillReturnError(sql.ErrNoRows)

	r.POST("/login", authService.Login)

	payload := []byte(`{"email": "wrong@example.com", "password": "password123"}`)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(payload))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid user, got %d", w.Code)
	}
}

func TestMe_Success(t *testing.T) {
	authService, mock, r := setupTest(t)
	defer authService.DB.Close()

	userID := 1
	email := "user@example.com"
	createdAt := time.Now().Format(time.RFC3339)

	token, _ := GenerateJWT(userID, authService.JWTSecret)

	rows := sqlmock.NewRows([]string{"email", "created_at"}).AddRow(email, createdAt)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT email, created_at FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnRows(rows)

	r.GET("/me", authService.Me)

	req, _ := http.NewRequest("GET", "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())

	}
}
