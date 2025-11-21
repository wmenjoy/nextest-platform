package handler

import (
	"net/http"

	"test-management-service/internal/websocket"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - should be restricted in production
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub *websocket.Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v2")
	{
		api.GET("/workflows/runs/:runId/stream", h.StreamWorkflowRun)
	}
}

// StreamWorkflowRun establishes WebSocket connection for workflow run
func (h *WebSocketHandler) StreamWorkflowRun(c *gin.Context) {
	runID := c.Param("runId")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runId is required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	client := websocket.NewClient(h.hub, conn, runID)
	h.hub.Register(client)

	// Start goroutines
	go client.WritePump()
	go client.ReadPump()
}
