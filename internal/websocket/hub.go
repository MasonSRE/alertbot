package websocket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"alertbot/internal/models"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to all clients
	broadcast chan *Message

	// Logger for hub operations
	logger *logrus.Logger

	// Mutex for thread-safe operations
	mutex sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	// The WebSocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan *Message

	// Hub reference
	hub *Hub

	// Client ID for identification
	id string

	// User information (if authenticated)
	userID   uint
	username string
	role     string

	// Subscription filters
	filters map[string]interface{}

	// Last activity time
	lastActivity time.Time

	// Logger with client context
	logger *logrus.Entry
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// NewClient creates a new WebSocket client
func (h *Hub) NewClient(conn *websocket.Conn, userID uint, username, role string) *Client {
	return NewClient(h, conn, userID, username, role)
}

// Run starts the hub and handles client registration/unregistration and broadcasting
func (h *Hub) Run() {
	defer h.cancel()
	
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info("WebSocket hub shutting down")
			return
			
		case client := <-h.register:
			h.registerClient(client)
			
		case client := <-h.unregister:
			h.unregisterClient(client)
			
		case message := <-h.broadcast:
			h.broadcastMessage(message)
			
		case <-ticker.C:
			h.pingClients()
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	h.clients[client] = true
	
	h.logger.WithFields(logrus.Fields{
		"client_id": client.id,
		"user_id":   client.userID,
		"username":  client.username,
	}).Info("WebSocket client connected")
	
	// Send welcome message
	welcomeMsg := &Message{
		Type: "welcome",
		Data: map[string]interface{}{
			"client_id": client.id,
			"server_time": time.Now(),
		},
		Timestamp: time.Now(),
	}
	
	select {
	case client.send <- welcomeMsg:
	default:
		h.closeClient(client)
	}
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		
		h.logger.WithFields(logrus.Fields{
			"client_id": client.id,
			"user_id":   client.userID,
			"username":  client.username,
		}).Info("WebSocket client disconnected")
	}
}

// broadcastMessage broadcasts a message to all connected clients
func (h *Hub) broadcastMessage(message *Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	for client := range h.clients {
		// Check if client should receive this message based on filters
		if h.shouldSendToClient(client, message) {
			select {
			case client.send <- message:
			default:
				h.closeClient(client)
			}
		}
	}
}

// shouldSendToClient determines if a message should be sent to a specific client
func (h *Hub) shouldSendToClient(client *Client, message *Message) bool {
	// If no filters are set, send all messages
	if len(client.filters) == 0 {
		return true
	}
	
	// Apply filters based on message type and content
	switch message.Type {
	case "alert_created", "alert_updated", "alert_resolved":
		// Check severity filter
		if severityFilter, ok := client.filters["severity"].(string); ok {
			if alert, ok := message.Data.(map[string]interface{}); ok {
				if alertSeverity, ok := alert["severity"].(string); ok {
					if severityFilter != alertSeverity {
						return false
					}
				}
			}
		}
		
		// Check status filter
		if statusFilter, ok := client.filters["status"].(string); ok {
			if alert, ok := message.Data.(map[string]interface{}); ok {
				if alertStatus, ok := alert["status"].(string); ok {
					if statusFilter != alertStatus {
						return false
					}
				}
			}
		}
	}
	
	return true
}

// closeClient closes a client connection
func (h *Hub) closeClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		client.conn.Close()
	}
}

// pingClients sends ping messages to all clients to keep connections alive
func (h *Hub) pingClients() {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	now := time.Now()
	for client := range h.clients {
		// Check if client has been inactive for too long
		if now.Sub(client.lastActivity) > pongWait {
			h.logger.WithField("client_id", client.id).Debug("Client timed out")
			h.closeClient(client)
			continue
		}
		
		// Send ping
		client.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			h.logger.WithError(err).WithField("client_id", client.id).Debug("Failed to send ping")
			h.closeClient(client)
		}
	}
}

// BroadcastAlertUpdate broadcasts an alert update to all connected clients
func (h *Hub) BroadcastAlertUpdate(alert *models.Alert, action string) {
	message := &Message{
		Type: fmt.Sprintf("alert_%s", action),
		Data: map[string]interface{}{
			"action": action,
			"alert":  alert,
		},
		Timestamp: time.Now(),
	}
	
	select {
	case h.broadcast <- message:
	default:
		h.logger.Warn("Broadcast channel is full, dropping message")
	}
}

// BroadcastSystemMessage broadcasts a system message to all connected clients
func (h *Hub) BroadcastSystemMessage(messageType string, data interface{}) {
	message := &Message{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
	}
	
	select {
	case h.broadcast <- message:
	default:
		h.logger.Warn("Broadcast channel is full, dropping system message")
	}
}

// GetClientCount returns the current number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetClients returns information about all connected clients
func (h *Hub) GetClients() []map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	clients := make([]map[string]interface{}, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, map[string]interface{}{
			"id":            client.id,
			"user_id":       client.userID,
			"username":      client.username,
			"role":          client.role,
			"last_activity": client.lastActivity,
			"filters":       client.filters,
		})
	}
	
	return clients
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	h.logger.Info("Shutting down WebSocket hub")
	h.cancel()
	
	// Close all client connections
	h.mutex.Lock()
	for client := range h.clients {
		client.conn.Close()
	}
	h.mutex.Unlock()
}