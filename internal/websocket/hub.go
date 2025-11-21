package websocket

import (
	"sync"
)

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	// Registered clients by runID
	clients map[string]map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	mu sync.RWMutex
}

// Message represents a workflow event message
type Message struct {
	RunID   string      `json:"runId"`
	Type    string      `json:"type"` // step_start, step_complete, step_log, variable_change
	Payload interface{} `json:"payload"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.runID] == nil {
				h.clients[client.runID] = make(map[*Client]bool)
			}
			h.clients[client.runID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.runID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.runID)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.RunID]
			h.mu.RUnlock()

			for client := range clients {
				select {
				case client.send <- message:
				default:
					h.mu.Lock()
					close(client.send)
					delete(h.clients[message.RunID], client)
					h.mu.Unlock()
				}
			}
		}
	}
}

// Broadcast sends a message to all clients watching a runID
func (h *Hub) Broadcast(runID string, msgType string, payload interface{}) {
	h.broadcast <- &Message{
		RunID:   runID,
		Type:    msgType,
		Payload: payload,
	}
}

// Register registers a new client connection
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client connection
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
