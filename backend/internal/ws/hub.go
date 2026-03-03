package ws

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10 // 54s
	maxMessageSize = 8192
)

// Client represents a single WebSocket connection belonging to a user.
type Client struct {
	hub    *Hub
	userID uuid.UUID
	conn   *websocket.Conn
	send   chan []byte
}

type broadcastMsg struct {
	userID  uuid.UUID
	payload []byte
}

// Hub maintains the set of active WebSocket clients and routes messages to them.
// All map mutations are serialized through the Run() goroutine — no mutex required.
type Hub struct {
	// clients maps userID → set of active connections (multiple tabs supported)
	clients    map[uuid.UUID]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastMsg
	done       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]struct{}),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
		broadcast:  make(chan broadcastMsg, 256),
		done:       make(chan struct{}),
	}
}

// Run processes hub events. Call it in a dedicated goroutine.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			if _, ok := h.clients[c.userID]; !ok {
				h.clients[c.userID] = make(map[*Client]struct{})
			}
			h.clients[c.userID][c] = struct{}{}
			slog.Debug("ws: client registered", "user_id", c.userID, "total_connections", len(h.clients[c.userID]))

		case c := <-h.unregister:
			if conns, ok := h.clients[c.userID]; ok {
				if _, exists := conns[c]; exists {
					delete(conns, c)
					close(c.send)
					if len(conns) == 0 {
						delete(h.clients, c.userID)
					}
				}
			}

		case msg := <-h.broadcast:
			conns, ok := h.clients[msg.userID]
			if !ok {
				slog.Debug("ws: no connected clients for user, dropping push", "user_id", msg.userID)
				continue
			}
			for c := range conns {
				select {
				case c.send <- msg.payload:
				default:
					// Slow client — drop and remove
					delete(conns, c)
					close(c.send)
				}
			}
			if len(conns) == 0 {
				delete(h.clients, msg.userID)
			}

		case <-h.done:
			return
		}
	}
}

// Stop shuts down the hub.
func (h *Hub) Stop() { close(h.done) }

// Send enqueues a payload for all connections belonging to userID.
func (h *Hub) Send(userID uuid.UUID, payload []byte) {
	select {
	case h.broadcast <- broadcastMsg{userID: userID, payload: payload}:
	default:
		slog.Warn("ws: broadcast channel full, dropping message", "user_id", userID)
	}
}

// readPump pumps incoming frames from the connection. Runs in its own goroutine.
// Its only job is pong-handling and detecting disconnects; clients are read-only from the server.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint:errcheck
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint:errcheck
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Debug("ws read error", "user_id", c.userID, "err", err)
			}
			break
		}
	}
}

// writePump pumps messages from the send channel to the connection. Runs in its own goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{}) //nolint:errcheck
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
