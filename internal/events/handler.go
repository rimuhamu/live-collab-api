package events

import (
	"database/sql"
	"encoding/json"
	"errors"
	"live-collab-api/internal/auth"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	DB          *sql.DB
	AuthService *auth.AuthService
}

type Event struct {
	ID         int             `json:"id"`
	DocumentId int             `json:"document_id"`
	UserId     int             `json:"user_id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type CreateEventRequest struct {
	EventType string          `json:"event_type" binding:"required"`
	Payload   json.RawMessage `json:"payload" binding:"required"`
}

func (h *EventHandler) CreateDocumentEvent(c *gin.Context) {
	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unauthorized"})
		return
	}

	documentIdParam := c.Param("id")
	documentId, err := strconv.Atoi(documentIdParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document id"})
		return
	}

	var req CreateEventRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ownerId int
	err = h.DB.QueryRow("SELECT owner_id FROM documents WHERE id = $1", documentId).Scan(&ownerId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Document does not exist"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	if ownerId != userId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not authorized to create events for this document"})
		return
	}

	validEventTypes := map[string]bool{
		"text_insert":   true,
		"text_delete":   true,
		"text_replace":  true,
		"cursor_move":   true,
		"selection":     true,
		"document_save": true,
	}

	if !validEventTypes[req.EventType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event type"})
		return
	}

	var eventId int
	err = h.DB.QueryRow("INSERT INTO events (document_id, user_id, event_type, payload) VALUES ($1,$2,$3,$4) RETURNING id",
		documentId, userId, req.EventType, req.Payload).Scan(&eventId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event created", "event_id": eventId})
}

func (h *EventHandler) GetDocumentEvent(c *gin.Context) {
	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unauthorized"})
		return
	}

	documentIdParam := c.Param("id")
	documentId, err := strconv.Atoi(documentIdParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document id"})
		return
	}

	var ownerId int
	err = h.DB.QueryRow("SELECT owner_id FROM documents WHERE id = $1", documentId).Scan(&ownerId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Document does not exist"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	if ownerId != userId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not authorized to get events for this document"})
		return
	}

	limitParam := c.DefaultQuery("limit", "50")
	offsetParam := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetParam)
	if err != nil || offset <= 0 {
		offset = 0
	}

	rows, err := h.DB.Query(
		"SELECT id, document_id, user_id, event_type, payload, created_at, updated_at FROM events WHERE document_id = $1 ORDER BY created_at LIMIT $2 OFFSET $3",
		documentId, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query error", "detail": err.Error()})
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.ID, &event.DocumentId, &event.UserId, &event.EventType, &event.Payload, &event.CreatedAt, &event.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database scan error", "detail": err.Error()})
			return
		}
		events = append(events, event)
	}

	if events == nil {
		events = []Event{}
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "limit": limit, "offset": offset})

}
