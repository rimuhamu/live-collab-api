package websocket

import (
	"database/sql"
	"encoding/json"
	"live-collab-api/internal/auth"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func setupWebSocketTest(t *testing.T) (*WebSocketHandler, sqlmock.Sqlmock, *gin.Engine, *auth.AuthService, *Hub) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}

	authService := &auth.AuthService{
		DB:        db,
		JWTSecret: "test-secret",
	}

	hub := NewHub()
	go hub.Run()

	wsHandler := &WebSocketHandler{
		Hub:         hub,
		DB:          db,
		AuthService: authService,
	}

	r := gin.Default()
	return wsHandler, mock, r, authService, hub
}

func TestHub_NewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.clients == nil {
		t.Error("clients map not initialized")
	}

	if hub.register == nil {
		t.Error("register channel not initialized")
	}

	if hub.unregister == nil {
		t.Error("unregister channel not initialized")
	}

	if hub.broadcast == nil {
		t.Error("broadcast channel not initialized")
	}
}

func TestHub_RegisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		ID:         "test-client-1",
		DocumentId: 1,
		UserId:     1,
		Permission: "edit",
		Send:       make(chan []byte, 256),
		Hub:        hub,
	}

	hub.register <- client

	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	if hub.clients[1] == nil {
		t.Error("Document clients map not created")
	}

	if hub.clients[1]["test-client-1"] == nil {
		t.Error("Client not registered")
	}

	if hub.clients[1]["test-client-1"].ID != "test-client-1" {
		t.Errorf("Expected client ID 'test-client-1', got %s", hub.clients[1]["test-client-1"].ID)
	}
}

func TestHub_UnregisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		ID:         "test-client-1",
		DocumentId: 1,
		UserId:     1,
		Permission: "edit",
		Send:       make(chan []byte, 256),
		Hub:        hub,
	}

	hub.register <- client
	time.Sleep(100 * time.Millisecond)

	hub.unregister <- client
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	if hub.clients[1] != nil && hub.clients[1]["test-client-1"] != nil {
		t.Error("Client was not unregistered")
	}
}

func TestHub_BroadcastMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{
		ID:         "client-1",
		DocumentId: 1,
		UserId:     1,
		Permission: "edit",
		Send:       make(chan []byte, 256),
		Hub:        hub,
	}

	client2 := &Client{
		ID:         "client-2",
		DocumentId: 1,
		UserId:     2,
		Permission: "view",
		Send:       make(chan []byte, 256),
		Hub:        hub,
	}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(100 * time.Millisecond)

	message := &Message{
		Type:       "edit",
		DocumentId: 1,
		UserId:     1,
		Payload: map[string]interface{}{
			"operation": "insert",
			"position":  0,
			"content":   "Hello",
		},
	}

	hub.BroadcastMessage(message)
	time.Sleep(100 * time.Millisecond)

	select {
	case msg := <-client1.Send:
		var receivedMsg Message
		if err := json.Unmarshal(msg, &receivedMsg); err != nil {
			t.Errorf("Error unmarshaling message: %v", err)
		}
		if receivedMsg.Type != "edit" {
			t.Errorf("Expected message type 'edit', got %s", receivedMsg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Client 1 did not receive message")
	}

	select {
	case msg := <-client2.Send:
		var receivedMsg Message
		if err := json.Unmarshal(msg, &receivedMsg); err != nil {
			t.Errorf("Error unmarshaling message: %v", err)
		}
		if receivedMsg.Type != "edit" {
			t.Errorf("Expected message type 'edit', got %s", receivedMsg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Client 2 did not receive message")
	}
}

func TestHub_GetDocumentClientCount(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{
		ID:         "client-1",
		DocumentId: 1,
		UserId:     1,
		Permission: "edit",
		Send:       make(chan []byte, 256),
		Hub:        hub,
	}

	client2 := &Client{
		ID:         "client-2",
		DocumentId: 1,
		UserId:     2,
		Permission: "edit",
		Send:       make(chan []byte, 256),
		Hub:        hub,
	}

	count := hub.GetDocumentClientCount(1)
	if count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}

	hub.register <- client1
	time.Sleep(50 * time.Millisecond)

	count = hub.GetDocumentClientCount(1)
	if count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}

	hub.register <- client2
	time.Sleep(50 * time.Millisecond)

	count = hub.GetDocumentClientCount(1)
	if count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}
}

func TestWebSocketHandler_HasDocumentAccess_Owner(t *testing.T) {
	wsHandler, mock, _, _, _ := setupWebSocketTest(t)
	defer wsHandler.DB.Close()

	userID := 1
	documentID := 1

	mock.ExpectQuery(regexp.QuoteMeta("SELECT owner_id FROM documents WHERE id = $1")).
		WithArgs(documentID).
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(userID))

	hasAccess, permission := wsHandler.hasDocumentAccess(userID, documentID)

	if !hasAccess {
		t.Error("Expected owner to have access")
	}

	if permission != "owner" {
		t.Errorf("Expected permission 'owner', got '%s'", permission)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestWebSocketHandler_HasDocumentAccess_Collaborator(t *testing.T) {
	wsHandler, mock, _, _, _ := setupWebSocketTest(t)
	defer wsHandler.DB.Close()

	userID := 2
	ownerID := 1
	documentID := 1

	mock.ExpectQuery(regexp.QuoteMeta("SELECT owner_id FROM documents WHERE id = $1")).
		WithArgs(documentID).
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(ownerID))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT permission FROM document_collaborators")).
		WithArgs(documentID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"permission"}).AddRow("edit"))

	hasAccess, permission := wsHandler.hasDocumentAccess(userID, documentID)

	if !hasAccess {
		t.Error("Expected collaborator to have access")
	}

	if permission != "edit" {
		t.Errorf("Expected permission 'edit', got '%s'", permission)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestWebSocketHandler_HasDocumentAccess_NoAccess(t *testing.T) {
	wsHandler, mock, _, _, _ := setupWebSocketTest(t)
	defer wsHandler.DB.Close()

	userID := 3
	ownerID := 1
	documentID := 1

	mock.ExpectQuery(regexp.QuoteMeta("SELECT owner_id FROM documents WHERE id = $1")).
		WithArgs(documentID).
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(ownerID))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT permission FROM document_collaborators")).
		WithArgs(documentID, userID).
		WillReturnError(sql.ErrNoRows)

	hasAccess, permission := wsHandler.hasDocumentAccess(userID, documentID)

	if hasAccess {
		t.Error("Expected user to not have access")
	}

	if permission != "" {
		t.Errorf("Expected empty permission, got '%s'", permission)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestWebSocketHandler_ApplyEdit_Insert(t *testing.T) {
	wsHandler, _, _, _, _ := setupWebSocketTest(t)
	defer wsHandler.DB.Close()

	edit := &EditEvent{
		Operation: "insert",
		Position:  5,
		Content:   "World",
	}

	result := wsHandler.applyEdit("Hello", edit)
	expected := "HelloWorld"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestWebSocketHandler_ApplyEdit_InsertMiddle(t *testing.T) {
	wsHandler, _, _, _, _ := setupWebSocketTest(t)
	defer wsHandler.DB.Close()

	edit := &EditEvent{
		Operation: "insert",
		Position:  5,
		Content:   " Beautiful",
	}

	result := wsHandler.applyEdit("Hello World", edit)
	expected := "Hello Beautiful World"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestWebSocketHandler_ApplyEdit_Delete(t *testing.T) {
	wsHandler, _, _, _, _ := setupWebSocketTest(t)
	defer wsHandler.DB.Close()

	edit := &EditEvent{
		Operation: "delete",
		Position:  5,
		Length:    6,
	}

	result := wsHandler.applyEdit("Hello World", edit)
	expected := "Hello"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// Integration test
func TestWebSocketHandler_FullIntegration(t *testing.T) {
	wsHandler, mock, _, authService, hub := setupWebSocketTest(t)
	defer wsHandler.DB.Close()

	userID := 1
	documentID := 1
	token, _ := auth.GenerateJWT(userID, authService.JWTSecret)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT owner_id FROM documents WHERE id = $1")).
		WithArgs(documentID).
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(userID))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		c.Params = gin.Params{{Key: "document_id", Value: "1"}}
		wsHandler.HandleWebSocket(c)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Add("Authorization", "Bearer "+token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(200 * time.Millisecond)

	count := hub.GetDocumentClientCount(documentID)
	if count != 1 {
		t.Errorf("Expected 1 active client, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}
