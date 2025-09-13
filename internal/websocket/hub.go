package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID         string
	DocumentId int
	UserId     int
	Conn       *websocket.Conn
	Send       chan []byte
	Hub        *Hub
}

type Message struct {
	Type       string      `json:"type"`
	DocumentId int         `json:"document_id"`
	UserId     int         `json:"user_id"`
	Version    int         `json:"version"`
	Payload    interface{} `json:"payload"`
	Timestamp  int64       `json:"timestamp"`
}

type EditEvent struct {
	Operation string `json:"operation"`
	Position  int    `json:"position"`
	Content   string `json:"content,omitempty"`
	Length    int    `json:"length,omitempty"`
}

type Hub struct {
	clients    map[int]map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mutex      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToDocument(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.clients[client.DocumentId] == nil {
		h.clients[client.DocumentId] = make(map[string]*Client)
	}

	h.clients[client.DocumentId][client.ID] = client

	log.Printf("Client %s is connected to document %d. Total clients for doc: %d:\n",
		client.ID, client.DocumentId, len(h.clients[client.DocumentId]))

	confirmMsg := &Message{
		Type:       "connected",
		DocumentId: client.DocumentId,
		UserId:     client.UserId,
	}

	if data, err := json.Marshal(confirmMsg); err == nil {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(h.clients[client.DocumentId], client.ID)
		}
	}

}

func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clients, exists := h.clients[client.DocumentId]; exists {
		if _, exists := clients[client.ID]; exists {
			delete(clients, client.ID)
			close(client.Send)

			log.Printf("Client %s is disconnected from document %d. Remaining clients: %d",
				client.ID, client.DocumentId, len(clients))

			if len(clients) == 0 {
				delete(h.clients, client.DocumentId)
			}
		}

	}
}

func (h *Hub) broadcastToDocument(message *Message) {
	h.mutex.RLock()
	clients := h.clients[message.DocumentId]
	h.mutex.RUnlock()

	if clients == nil {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling message: %v", err)
		return
	}

	for clientId, client := range clients {
		if client == nil {
			continue
		}

		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(clients, clientId)
		}
	}
}

func (h *Hub) GetDocumentClientCount(documentId int) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if clients, exists := h.clients[documentId]; exists {
		return len(clients)
	}
	return 0
}

func (h *Hub) BroadcastMessage(message *Message) {
	h.broadcast <- message
}
