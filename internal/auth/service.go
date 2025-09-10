package auth

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	DB        *sql.DB
	JWTSecret string
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateJWT(userId int, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userId,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *AuthService) GetUserIDFromToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte(s.JWTSecret), nil
	})

	if err != nil {
		return 0, fmt.Errorf("invalid token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIdValue, exists := claims["user_id"]
		if !exists {
			return 0, fmt.Errorf("user_id not found in token")
		}

		// Convert to int (handle the float64 JSON unmarshalling issue)
		switch v := userIdValue.(type) {
		case float64:
			return int(v), nil
		case int:
			return v, nil
		case string:
			return strconv.Atoi(v)
		default:
			return 0, fmt.Errorf("invalid user_id type in token: %T", v)
		}
	}

	return 0, fmt.Errorf("invalid token claims")
}

func (s *AuthService) GetUserIDFromAuthHeader(authHeader string) (int, error) {
	if authHeader == "" {
		return 0, fmt.Errorf("authorization header missing")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, fmt.Errorf("invalid authorization header format")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	return s.GetUserIDFromToken(tokenString)
}

func (s *AuthService) GetUserIDFromGinContext(c *gin.Context) (int, error) {
	authHeader := c.GetHeader("Authorization")
	return s.GetUserIDFromAuthHeader(authHeader)
}
