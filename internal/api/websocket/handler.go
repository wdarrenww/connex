package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"connex/pkg/logger"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Message types for WebSocket communication
const (
	MessageTypeChat   = "chat"
	MessageTypeSystem = "system"
	MessageTypeAuth   = "auth"
	MessageTypePing   = "ping"
	MessageTypePong   = "pong"
	MessageTypeError  = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
	Room      string      `json:"room,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   string
	Room     string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	mu       sync.Mutex
	lastPing time.Time
}

// Hub manages all WebSocket connections
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	rooms      map[string]map[*Client]bool
	redis      *redis.Client
	jwtSecret  string
	logger     *logger.Logger
	mu         sync.RWMutex
}

// Handler handles WebSocket connections
type Handler struct {
	hub       *Hub
	upgrader  websocket.Upgrader
	logger    *logger.Logger
	jwtSecret string
}

// NewHandler creates a new WebSocket handler
func NewHandler(jwtSecret string, redisClient *redis.Client) *Handler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Allow localhost for development and your domain for production
			return origin == "http://localhost:3000" ||
				origin == "https://your-frontend-domain.com" ||
				origin == "http://localhost:8080"
		},
	}

	hub := &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[string]map[*Client]bool),
		redis:      redisClient,
		jwtSecret:  jwtSecret,
		logger:     logger.GetGlobal(),
	}

	handler := &Handler{
		hub:       hub,
		upgrader:  upgrader,
		logger:    logger.GetGlobal(),
		jwtSecret: jwtSecret,
	}

	// Start the hub
	go hub.run()

	return handler
}

// HandleWebSocket handles WebSocket upgrade and connection
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Rate limiting check
	if !h.checkRateLimit(r) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", zap.String("error", err.Error()))
		return
	}

	// Create client
	client := &Client{
		ID:       generateClientID(),
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      h.hub,
		lastPing: time.Now(),
	}

	// Authenticate client (optional)
	if token := r.URL.Query().Get("token"); token != "" {
		if userID, err := h.authenticateToken(token); err == nil {
			client.UserID = userID
		}
	}

	// Register client
	h.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// authenticateToken validates JWT token and returns user ID
func (h *Handler) authenticateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.jwtSecret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userID, ok := claims["user_id"].(string); ok {
			return userID, nil
		}
	}

	return "", fmt.Errorf("invalid token")
}

// checkRateLimit implements basic rate limiting
func (h *Handler) checkRateLimit(r *http.Request) bool {
	// Simple IP-based rate limiting
	ip := r.RemoteAddr
	key := fmt.Sprintf("ws_rate_limit:%s", ip)

	ctx := context.Background()
	count, err := h.hub.redis.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		h.logger.Error("Rate limit check failed", zap.String("error", err.Error()))
		return true // Allow if Redis is down
	}

	if count >= 10 { // 10 connections per minute
		return false
	}

	// Increment counter
	h.hub.redis.Incr(ctx, key)
	h.hub.redis.Expire(ctx, key, time.Minute)

	return true
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// run manages the hub's main loop
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if client.Room != "" {
				if h.rooms[client.Room] == nil {
					h.rooms[client.Room] = make(map[*Client]bool)
				}
				h.rooms[client.Room][client] = true
			}
			h.mu.Unlock()

			h.logger.Info("Client connected", zap.String("client_id", client.ID), zap.String("user_id", client.UserID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				if client.Room != "" {
					delete(h.rooms[client.Room], client)
					if len(h.rooms[client.Room]) == 0 {
						delete(h.rooms, client.Room)
					}
				}
			}
			h.mu.Unlock()

			h.logger.Info("Client disconnected", zap.String("client_id", client.ID), zap.String("user_id", client.UserID))

		case message := <-h.broadcast:
			h.mu.RLock()
			if message.Room != "" {
				// Broadcast to room
				for client := range h.rooms[message.Room] {
					select {
					case client.Send <- h.serializeMessage(message):
					default:
						close(client.Send)
						delete(h.clients, client)
					}
				}
			} else {
				// Broadcast to all clients
				for client := range h.clients {
					select {
					case client.Send <- h.serializeMessage(message):
					default:
						close(client.Send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// serializeMessage converts Message to JSON
func (h *Hub) serializeMessage(msg *Message) []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to serialize message", zap.String("error", err.Error()))
		return []byte(`{"type":"error","data":"Internal error"}`)
	}
	return data
}

// Broadcast sends a message to all clients or a specific room
func (h *Hub) Broadcast(msg *Message) {
	h.broadcast <- msg
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(4096) // 4KB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("WebSocket read error", zap.String("error", err.Error()))
			}
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendError("Invalid message format")
			continue
		}

		// Handle different message types
		c.handleMessage(&msg)
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
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

// handleMessage processes incoming messages
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MessageTypePing:
		c.sendPong()
	case MessageTypeChat:
		c.handleChatMessage(msg)
	case MessageTypeAuth:
		c.handleAuthMessage(msg)
	default:
		c.sendError("Unknown message type")
	}
}

// handleChatMessage processes chat messages
func (c *Client) handleChatMessage(msg *Message) {
	if c.UserID == "" {
		c.sendError("Authentication required")
		return
	}

	// Broadcast to room or all clients
	broadcastMsg := &Message{
		Type:      MessageTypeChat,
		Data:      msg.Data,
		Timestamp: time.Now(),
		UserID:    c.UserID,
		Room:      c.Room,
	}

	c.Hub.Broadcast(broadcastMsg)
}

// handleAuthMessage processes authentication messages
func (c *Client) handleAuthMessage(msg *Message) {
	// Handle room joining, etc.
	if room, ok := msg.Data.(map[string]interface{})["room"].(string); ok {
		c.mu.Lock()
		c.Room = room
		c.mu.Unlock()

		c.sendMessage(&Message{
			Type: MessageTypeSystem,
			Data: map[string]interface{}{
				"message": "Joined room: " + room,
			},
			Timestamp: time.Now(),
		})
	}
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(msg *Message) {
	data := c.Hub.serializeMessage(msg)
	select {
	case c.Send <- data:
	default:
		close(c.Send)
		c.Hub.unregister <- c
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(message string) {
	c.sendMessage(&Message{
		Type: MessageTypeError,
		Data: map[string]interface{}{
			"error": message,
		},
		Timestamp: time.Now(),
	})
}

// sendPong sends a pong response
func (c *Client) sendPong() {
	c.sendMessage(&Message{
		Type:      MessageTypePong,
		Data:      map[string]interface{}{"timestamp": time.Now().Unix()},
		Timestamp: time.Now(),
	})
}
