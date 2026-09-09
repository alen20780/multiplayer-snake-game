package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 5 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev/testing
	},
}

// Client is a middleman between the WebSocket connection and the Hub.
type Client struct {
	hub  *Hub
	game *Game
	conn *websocket.Conn
	send chan []byte
	id   string
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// ClientCount returns the number of active connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Run executes the hub event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WebSocket] Client connected: %s (Total: %d)", client.id, h.ClientCount())

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[WebSocket] Client disconnected: %s (Total: %d)", client.id, h.ClientCount())

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Slow client buffer full: close channel and prune
					log.Printf("[WebSocket] Dropping slow client %s", client.id)
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all active clients
func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		// Broadcast channel saturated
	}
}

// readPump pumps messages from the websocket connection to the game input loop
func (c *Client) readPump() {
	defer func() {
		c.game.RequestLeave(c.id)
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Error reading from client %s: %v", c.id, err)
			}
			break
		}

		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Printf("[WebSocket] Malformed message from %s: %v", c.id, err)
			continue
		}

		switch clientMsg.Type {
		case MsgTypeJoin:
			c.game.RequestJoin(c, clientMsg.Name)
		case MsgTypeInput:
			if clientMsg.Direction != "" {
				c.game.QueueInput(c.id, clientMsg.Direction)
			}
		case MsgTypePing:
			pongBytes, _ := json.Marshal(map[string]string{"type": MsgTypePong})
			c.sendMsg(pongBytes)
		}
	}
}

// writePump pumps messages from the client's send channel to the websocket connection
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
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain any additional queued messages into the same frame to save syscalls
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

// sendMsg non-blockingly queues a message to the client
func (c *Client) sendMsg(data []byte) {
	select {
	case c.send <- data:
	default:
	}
}

// SendGameOver sends a game over notification to the client
func (c *Client) SendGameOver(score int, reason string) {
	msg := GameOverMessage{
		Type:       MsgTypeGameOver,
		FinalScore: score,
		Reason:     reason,
	}
	bytes, _ := json.Marshal(msg)
	c.sendMsg(bytes)
}

// ServeWs handles websocket requests from peer
func ServeWs(hub *Hub, game *Game, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	clientID := fmt.Sprintf("%d", time.Now().UnixNano())

	client := &Client{
		hub:  hub,
		game: game,
		conn: conn,
		send: make(chan []byte, 128),
		id:   clientID,
	}

	client.hub.register <- client

	// Send initial Welcome message
	welcome := WelcomeMessage{
		Type:        MsgTypeWelcome,
		PlayerID:    clientID,
		WorldWidth:  game.config.WorldWidth,
		WorldHeight: game.config.WorldHeight,
		TickRate:    game.config.TickRate,
	}
	welcomeBytes, _ := json.Marshal(welcome)
	client.sendMsg(welcomeBytes)

	// Start goroutines for read and write
	go client.writePump()
	go client.readPump()
}
