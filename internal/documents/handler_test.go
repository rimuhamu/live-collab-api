package documents_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"live-collab-api/internal/documents"
	"live-collab-api/internal/documents/mocks"
)

func TestCreateDocument(t *testing.T) {
	// Setup Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock Services
	mockDocService := mocks.NewMockService(ctrl)
	mockAuthService := mocks.NewMockAuthService(ctrl)

	// Setup Handler with Mocks
	handler := &documents.DocumentHandler{
		DocumentService: mockDocService,
		AuthService:     mockAuthService,
	}

	// Setup Gin
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/documents", handler.CreateDocument)

	t.Run("Success", func(t *testing.T) {
		// Prepare Request
		reqBody := documents.CreateDocumentRequest{
			Title: "Test Document",
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/documents", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer valid_token")
		req.Header.Set("Content-Type", "application/json")

		// Expectations
		userID := 123
		createdDoc := &documents.Document{
			ID:      "1",
			Title:   "Test Document",
			OwnerId: "123",
		}

		mockAuthService.EXPECT().GetUserIDFromGinContext(gomock.Any()).Return(userID, nil)
		mockDocService.EXPECT().CreateDocument(reqBody.Title, userID).Return(createdDoc, nil)

		// Perform Request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusCreated, w.Code)

		var response documents.Document
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, createdDoc.ID, response.ID)
		assert.Equal(t, createdDoc.Title, response.Title)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		// Prepare Request
		reqBody := documents.CreateDocumentRequest{
			Title: "Test Document",
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/documents", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")

		// Expectations
		mockAuthService.EXPECT().GetUserIDFromGinContext(gomock.Any()).Return(0, errors.New("auth error"))

		// Perform Request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("BadRequest", func(t *testing.T) {
		// Prepare Request with missing title
		reqBody := map[string]string{}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/documents", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer valid_token")
		req.Header.Set("Content-Type", "application/json")

		// Expectations
		userID := 123
		mockAuthService.EXPECT().GetUserIDFromGinContext(gomock.Any()).Return(userID, nil)

		// Perform Request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {
		// Prepare Request
		reqBody := documents.CreateDocumentRequest{
			Title: "Test Document",
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/documents", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer valid_token")
		req.Header.Set("Content-Type", "application/json")

		// Expectations
		userID := 123
		mockAuthService.EXPECT().GetUserIDFromGinContext(gomock.Any()).Return(userID, nil)
		mockDocService.EXPECT().CreateDocument(reqBody.Title, userID).Return(nil, errors.New("db error"))

		// Perform Request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
