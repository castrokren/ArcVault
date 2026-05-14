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
// Slow or dead clients are removed silently.
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
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade failed: %v", err)
		return
	}

	s.hub.add(conn)
	log.Printf("WS client connected (%d total)", len(s.hub.clients))

	// keep connection alive; remove on disconnect
	go func() {
		defer func() {
			s.hub.remove(conn)
			conn.Close()
			log.Printf("WS client disconnected")
		}()
		for {
			// ReadMessage blocks; returns error on close
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}
