package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Event is the JSON envelope broadcast to all connected WebSocket clients.
type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Hub manages connected WebSocket clients and broadcasts events to them.
type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func newHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

// Broadcast sends an event to every connected client.
func (h *Hub) Broadcast(event Event) {
	msg, err := json.Marshal(event)
	if err != nil {
		log.Printf("Hub: failed to marshal event: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("Hub: removing dead client: %v", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

func (h *Hub) add(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = struct{}{}
}

func (h *Hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
}

// --- WebSocket upgrade handler ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// handleWS upgrades the connection and registers it with the hub.
// Auth: accepts Bearer token from Authorization header OR ?token= query param
// (browsers cannot set headers on WebSocket connections).
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	// auth checked here instead of authMiddleware to support query param token
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token != s.cfg.AdminToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade failed: %v", err)
		return
	}

	s.hub.add(conn)
	log.Printf("WS client connected (%d total)", len(s.hub.clients))

	go func() {
		defer func() {
			s.hub.remove(conn)
			conn.Close()
			log.Printf("WS client disconnected")
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}
