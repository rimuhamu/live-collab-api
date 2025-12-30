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

// CreateDocumentEvent godoc
// @Summary Create document event
// @Description Create a new event for collaborative editing (text operations, cursor movements, etc.). User can only create events for documents they own.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Param request body CreateEventRequest true "Event data with type and payload"
// @Success 201 {object} CreateEventResponse "Event created successfully"
// @Failure 400 {object} ErrorResponse "Invalid input data or event type"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} ErrorResponse "Access denied - you don't own this document"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /documents/{id}/events [post]
func (h *EventHandler) CreateDocumentEvent(c *gin.Context) {
	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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

	var temp interface{}
	if err := json.Unmarshal([]byte(req.Payload), &temp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON in payload"})
		return
	}

	var ownerId int
	err = h.DB.QueryRow("SELECT owner_id FROM documents WHERE id = $1", documentId).Scan(&ownerId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document does not exist"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	hasEditPermission := false
	if ownerId == userId {
		hasEditPermission = true
	} else {
		var permission string
		err = h.DB.QueryRow(`
			SELECT permission FROM document_collaborators 
			WHERE document_id = $1 AND user_id = $2
		`, documentId, userId).Scan(&permission)

		if err == nil && permission == "edit" {
			hasEditPermission = true
		}
	}

	if !hasEditPermission {
		c.JSON(http.StatusForbidden, gin.H{"error": "You need edit permission to create events for this document"})
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

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Event created",
		"event_id":    eventId,
		"event_type":  req.EventType,
		"document_id": documentId,
		"user_id":     userId,
	})
}

// GetDocumentEvents godoc
// @Summary Get document events
// @Description Get all events for a specific document with pagination. User can only access events for documents they own.
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Param limit query int false "Number of events to return (default 50, max 1000)" default(50)
// @Param offset query int false "Number of events to skip (default 0)" default(0)
// @Success 200 {object} EventListResponse "List of events with pagination info"
// @Failure 400 {object} ErrorResponse "Invalid document ID or parameters"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} ErrorResponse "Access denied - you don't own this document"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /documents/{id}/events [get]
func (h *EventHandler) GetDocumentEvents(c *gin.Context) {
	userId, err := h.AuthService.GetUserIDFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	documentIdParam := c.Param("id")
	documentId, err := strconv.Atoi(documentIdParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document id"})
		return
	}

	var hasAccess bool
	err = h.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM documents WHERE id = $1 AND owner_id = $2
			UNION
			SELECT 1 FROM document_collaborators WHERE document_id = $1 AND user_id = $2
		)
	`, documentId, userId).Scan(&hasAccess)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied - you don't have access to this document"})
		return
	}

	limitParam := c.DefaultQuery("limit", "50")
	offsetParam := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetParam)
	if err != nil || offset < 0 {
		offset = 0
	}

	rows, err := h.DB.Query(
		"SELECT id, document_id, user_id, event_type, payload, created_at, updated_at FROM events WHERE document_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
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

// swagger models for events

type EventResponse struct {
	ID         int       `json:"id" example:"1"`
	DocumentID int       `json:"document_id" example:"1"`
	UserID     int       `json:"user_id" example:"1"`
	EventType  string    `json:"event_type" example:"text_insert"`
	Payload    string    `json:"payload" example:"{\"position\":10,\"text\":\"Hello\"}"`
	CreatedAt  time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt  time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

type EventListResponse struct {
	Events []EventResponse `json:"events"`
	Limit  int             `json:"limit" example:"50"`
	Offset int             `json:"offset" example:"0"`
	Total  int             `json:"total" example:"3"`
}

type CreateEventRequest struct {
	EventType string `json:"event_type" binding:"required" example:"text_insert" enums:"text_insert,text_delete,text_replace,cursor_move,selection,document_save,document_open,user_join,user_leave"`
	Payload   string `json:"payload" binding:"required" example:"{\"position\":10,\"text\":\"Hello World\",\"timestamp\":\"2024-01-15T10:30:00Z\"}"`
}

type CreateEventResponse struct {
	Message    string `json:"message" example:"Event created successfully"`
	EventID    int    `json:"event_id" example:"1"`
	EventType  string `json:"event_type" example:"text_insert"`
	DocumentID int    `json:"document_id" example:"1"`
	UserID     int    `json:"user_id" example:"1"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}
