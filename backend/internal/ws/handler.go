package ws

import (
	"net/http"

	"github.com/404nfid/go-svelte-starter-kit/pkg/token"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Origin check is handled by the global CORS middleware
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS handles WebSocket upgrade requests.
// Authentication is done via a ?token= query parameter containing a valid JWT access token,
// since browsers cannot send custom headers during the WebSocket handshake.
func (h *Hub) ServeWS(tokenMgr *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		claims, err := tokenMgr.VerifyAccess(tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			http.Error(w, "invalid user", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := &Client{
			hub:    h,
			userID: userID,
			conn:   conn,
			send:   make(chan []byte, 256),
		}
		h.register <- client

		go client.writePump()
		go client.readPump()
	}
}
