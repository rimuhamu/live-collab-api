package websocket

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"live-collab-api/internal/auth"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var allowedOrigins = strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}

		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == strings.TrimSpace(allowedOrigin) {
				return true
			}
		}
		return false
	},
}

type WebSocketHandler struct {
	Hub         *Hub
	DB          *sql.DB
	AuthService *auth.AuthService
}

func (ws *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// get document id
	documentIdStr := c.Param("document_id")
	documentId, err := strconv.Atoi(documentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document id"})
		return
	}

	// authenticate user from token
	userId, err := ws.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	}

	// verify user access to document
	if !ws.hasDocumentAccess(userId, documentId) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket Upgrade Error: %v\n", err)
		return
	}

	client := &Client{
		ID:         uuid.New().String(),
		DocumentId: documentId,
		UserId:     userId,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		Hub:        ws.Hub,
	}

	ws.Hub.register <- client

	go client.writePump()
	go client.readPump(ws)
}

func (c *Client) readPump(ws *WebSocketHandler) {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageData, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var message Message
		if err := json.Unmarshal(messageData, &message); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		switch message.Type {
		case "edit":
			ws.handleEditMessage(&message)
		case "cursor":
			ws.handleCursorMessage(&message)
		default:
			log.Printf("Unknown message type: %v", message.Type)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(time.Second * 54)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (ws *WebSocketHandler) handleEditMessage(message *Message) {
	// parse edit event from payload
	payloadBytes, err := json.Marshal(message.Payload)
	if err != nil {
		log.Printf("Error marshaling payload: %v", err)
		return
	}

	var editEvent EditEvent
	if err := json.Unmarshal(payloadBytes, &editEvent); err != nil {
		log.Printf("Error unmarshaling payload: %v", err)
		return
	}

	currentVersion, err := ws.getCurrentDocumentVersion(message.DocumentId)
	if err != nil {
		log.Printf("Error getting current document version: %v", err)
		return
	}

	message.Version = currentVersion + 1

	if err := ws.persistEvent(message); err != nil {
		log.Printf("Error persisting event: %v", err)
		return
	}

	if err := ws.applyEditToDocument(message.DocumentId, &editEvent); err != nil {
		log.Printf("Error applying edit to document: %v", err)
		return
	}

	ws.Hub.BroadcastMessage(message)

	log.Printf("Processed edit event for document %d, version %d", message.DocumentId, message.Version)
}

func (ws *WebSocketHandler) handleCursorMessage(message *Message) {
	ws.Hub.BroadcastMessage(message)
}

func (ws *WebSocketHandler) hasDocumentAccess(userId, documentId int) bool {
	var ownerId int
	err := ws.DB.QueryRow("SELECT owner_id FROM documents WHERE id = $1", documentId).Scan(&ownerId)
	if err != nil {
		return false
	}

	return ownerId == userId
}

func (ws *WebSocketHandler) getCurrentDocumentVersion(documentID int) (int, error) {
	var version int
	err := ws.DB.QueryRow(`
		SELECT COALESCE(MAX(CAST(payload->>'version' AS INTEGER)), 0) 
		FROM events 
		WHERE document_id = $1 AND event_type = 'edit'
	`, documentID).Scan(&version)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return version, nil
}

func (ws *WebSocketHandler) persistEvent(message *Message) error {
	payloadJSON, err := json.Marshal(map[string]interface{}{
		"type":      message.Type,
		"version":   message.Version,
		"timestamp": message.Timestamp,
		"payload":   message.Payload,
	})
	if err != nil {
		return err
	}

	_, err = ws.DB.Exec(`
		INSERT INTO events (document_id, user_id, event_type, payload, created_at) 
		VALUES ($1, $2, $3, $4, NOW())
	`, message.DocumentId, message.UserId, message.Type, payloadJSON)

	return err
}

func (ws *WebSocketHandler) applyEditToDocument(documentId int, edit *EditEvent) error {
	var content string
	err := ws.DB.QueryRow("SELECT COALESCE(content, '') FROM documents WHERE id = $1", documentId).Scan(&content)
	if err != nil {
		return fmt.Errorf("failed to get document content: %v", err)
	}

	newContent := ws.applyEdit(content, edit)

	_, err = ws.DB.Exec("UPDATE documents SET content = $1 WHERE id = $2", newContent, documentId)
	return err
}

func (ws *WebSocketHandler) applyEdit(content string, edit *EditEvent) string {
	runes := []rune(content)

	switch edit.Operation {
	case "insert":
		if edit.Position > len(runes) {
			edit.Position = len(runes)
		}
		insertRunes := []rune(edit.Content)
		result := make([]rune, 0, len(runes)+len(insertRunes))
		result = append(result, runes[:edit.Position]...)
		result = append(result, insertRunes...)
		result = append(result, runes[edit.Position:]...)
		return string(result)

	case "delete":
		if edit.Position >= len(runes) {
			return content
		}
		endPosition := edit.Position + edit.Length
		if endPosition > len(runes) {
			endPosition = len(runes)
		}
		result := make([]rune, 0, len(runes)-edit.Length)
		result = append(result, runes[:edit.Position]...)
		result = append(result, runes[endPosition:]...)
		return string(result)

	default:
		return content
	}
}
