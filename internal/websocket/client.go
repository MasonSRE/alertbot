package websocket

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID uint, username, role string) *Client {
	clientID := uuid.New().String()
	
	client := &Client{
		conn:         conn,
		send:         make(chan *Message, 256),
		hub:          hub,
		id:           clientID,
		userID:       userID,
		username:     username,
		role:         role,
		filters:      make(map[string]interface{}),
		lastActivity: time.Now(),
		logger: hub.logger.WithFields(logrus.Fields{
			"client_id": clientID,
			"user_id":   userID,
			"username":  username,
		}),
	}
	
	return client
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.lastActivity = time.Now()
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	
	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.WithError(err).Error("WebSocket connection closed unexpectedly")
			}
			break
		}
		
		c.lastActivity = time.Now()
		c.handleMessage(messageBytes)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			// Send the message
			if err := c.conn.WriteJSON(message); err != nil {
				c.logger.WithError(err).Error("Failed to write message to WebSocket")
				return
			}
			
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(messageBytes []byte) {
	var message map[string]interface{}
	if err := json.Unmarshal(messageBytes, &message); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal client message")
		return
	}
	
	messageType, ok := message["type"].(string)
	if !ok {
		c.logger.Error("Message missing type field")
		return
	}
	
	c.logger.WithField("message_type", messageType).Debug("Received client message")
	
	switch messageType {
	case "subscribe":
		c.handleSubscribe(message)
	case "unsubscribe":
		c.handleUnsubscribe(message)
	case "ping":
		c.handlePing()
	case "get_filters":
		c.handleGetFilters()
	default:
		c.logger.WithField("message_type", messageType).Warn("Unknown message type")
	}
}

// handleSubscribe handles subscription requests from clients
func (c *Client) handleSubscribe(message map[string]interface{}) {
	filters, ok := message["filters"].(map[string]interface{})
	if !ok {
		c.logger.Error("Subscribe message missing or invalid filters")
		return
	}
	
	// Update client filters
	for key, value := range filters {
		c.filters[key] = value
	}
	
	c.logger.WithField("filters", c.filters).Debug("Client subscription updated")
	
	// Send confirmation
	response := &Message{
		Type: "subscription_updated",
		Data: map[string]interface{}{
			"filters": c.filters,
		},
		Timestamp: time.Now(),
	}
	
	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send subscription confirmation")
	}
}

// handleUnsubscribe handles unsubscription requests from clients
func (c *Client) handleUnsubscribe(message map[string]interface{}) {
	filterKeys, ok := message["filter_keys"].([]interface{})
	if !ok {
		// Clear all filters if no specific keys provided
		c.filters = make(map[string]interface{})
	} else {
		// Remove specific filter keys
		for _, keyInterface := range filterKeys {
			if key, ok := keyInterface.(string); ok {
				delete(c.filters, key)
			}
		}
	}
	
	c.logger.WithField("filters", c.filters).Debug("Client unsubscribed from filters")
	
	// Send confirmation
	response := &Message{
		Type: "unsubscription_confirmed",
		Data: map[string]interface{}{
			"filters": c.filters,
		},
		Timestamp: time.Now(),
	}
	
	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send unsubscription confirmation")
	}
}

// handlePing handles ping requests from clients
func (c *Client) handlePing() {
	response := &Message{
		Type: "pong",
		Data: map[string]interface{}{
			"server_time": time.Now(),
		},
		Timestamp: time.Now(),
	}
	
	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send pong response")
	}
}

// handleGetFilters handles requests for current filter information
func (c *Client) handleGetFilters() {
	response := &Message{
		Type: "current_filters",
		Data: map[string]interface{}{
			"filters": c.filters,
		},
		Timestamp: time.Now(),
	}
	
	select {
	case c.send <- response:
	default:
		c.logger.Warn("Failed to send current filters")
	}
}

// Start starts the client read and write pumps
func (c *Client) Start() {
	// Register the client with the hub
	c.hub.register <- c
	
	// Start the read and write pumps in separate goroutines
	go c.writePump()
	go c.readPump()
}

// SendMessage sends a message directly to this client
func (c *Client) SendMessage(messageType string, data interface{}) error {
	message := &Message{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
	}
	
	select {
	case c.send <- message:
		return nil
	default:
		return fmt.Errorf("client send channel is full")
	}
}

// GetInfo returns client information
func (c *Client) GetInfo() map[string]interface{} {
	return map[string]interface{}{
		"id":            c.id,
		"user_id":       c.userID,
		"username":      c.username,
		"role":          c.role,
		"filters":       c.filters,
		"last_activity": c.lastActivity,
	}
}