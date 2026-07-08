package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Event is the JSON envelope broadcast to all connected WebSocket clients.
type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Hub manages connected WebSocket clients and broadcasts events to them.
// It tracks two distinct connection pools:
//   - clients: dashboard (browser) connections — receives broadcasts
//   - agentConns: agent connections indexed by agent ID — bidirectional
type Hub struct {
	mu         sync.Mutex
	clients    map[*websocket.Conn]struct{}
	agentConns map[string]*websocket.Conn
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]struct{}),
		agentConns: make(map[string]*websocket.Conn),
	}
}

// Broadcast sends an event to every connected dashboard client.
//
// The mutex is held only long enough to snapshot the client list.
// Network writes happen after the lock is released so a slow or
// disconnected client cannot stall other hub operations or HTTP handlers.
// A 5-second write deadline prevents any single write from blocking forever.
func (h *Hub) Broadcast(event Event) {
	msg, err := json.Marshal(event)
	if err != nil {
		log.Printf("Hub: failed to marshal event: %v", err)
		return
	}

	// Snapshot clients under lock — release before doing any I/O.
	h.mu.Lock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		clients = append(clients, conn)
	}
	h.mu.Unlock()

	// Write to each client; collect any that have gone dead.
	var dead []*websocket.Conn
	for _, conn := range clients {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("Hub: removing dead client: %v", err)
			dead = append(dead, conn)
		}
	}

	// Remove dead clients.
	if len(dead) > 0 {
		h.mu.Lock()
		for _, conn := range dead {
			conn.Close()
			delete(h.clients, conn)
		}
		h.mu.Unlock()
	}
}

// SendToAgent sends a message to a specific agent's WebSocket connection.
// Returns an error if the agent is not currently connected.
func (h *Hub) SendToAgent(agentID string, msg any) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	h.mu.Lock()
	conn, ok := h.agentConns[agentID]
	h.mu.Unlock()

	if !ok {
		return fmt.Errorf("agent %q not connected", agentID)
	}

	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		return fmt.Errorf("failed to send to agent %q: %w", agentID, err)
	}
	return nil
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

func (h *Hub) addAgent(agentID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agentConns[agentID] = conn
}

func (h *Hub) removeAgent(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.agentConns, agentID)
}

// --- WebSocket upgrade handler ---

// Global upgrader with disabled CheckOrigin by default
// Will be overridden per-server instance with proper origin validation
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// WARNING: This should be overridden by Server.initWebSocketUpgrader()
		// Default deny to fail safe if initialization fails
		return false
	},
}

// handleWS upgrades the connection and registers it with the hub.
// Auth: accepts only JWT tokens from Authorization header or Sec-WebSocket-Protocol
// (browsers cannot set headers on WebSocket connections).
// Dashboard must use JWT — admin token not accepted here (use handleAgentWS for admin access).
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	// Try Sec-WebSocket-Protocol header as fallback (for browser clients).
	if token == "" {
		for _, proto := range r.Header["Sec-Websocket-Protocol"] {
			if strings.HasPrefix(proto, "bearer.") {
				token = strings.TrimPrefix(proto, "bearer.")
				break
			}
		}
	}
	// Dashboard connections must use valid JWT token (no admin token).
	if _, err := ValidateJWT(token, s.cfg.JWTSecret); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Echo back the bearer.* subprotocol so browsers accept the handshake.
	var upgradeHeader http.Header
	for _, proto := range r.Header["Sec-Websocket-Protocol"] {
		if strings.HasPrefix(proto, "bearer.") {
			upgradeHeader = http.Header{"Sec-Websocket-Protocol": []string{proto}}
			break
		}
	}
	conn, err := s.wsUpgrader.Upgrade(w, r, upgradeHeader)
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

// handleAgentWS upgrades an agent's connection, registers it by agent ID,
// and relays inbound update_progress events to dashboard clients.
// Auth: agent token (from Authorization header only).
func (s *Server) handleAgentWS(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	// Note: ?token= query param removed — tokens must be in Authorization header.

	// Accept both admin token and valid agent tokens.
	isAdmin := token == s.cfg.AdminToken
	if !isAdmin {
		if _, err := s.db.ValidateToken(token); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id query param required", http.StatusBadRequest)
		return
	}

	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Agent WS upgrade failed: %v", err)
		return
	}

	s.hub.addAgent(agentID, conn)
	log.Printf("Agent %q WS connected", agentID)

	go func() {
		defer func() {
			s.hub.removeAgent(agentID)
			conn.Close()
			log.Printf("Agent %q WS disconnected", agentID)
		}()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// Relay update_progress events from agent to dashboard clients.
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(msg, &envelope); err != nil {
				continue
			}
			msgType, _ := envelope["type"]
			var typeStr string
			json.Unmarshal(msgType, &typeStr)
			if typeStr == "update_progress" {
				var payload any
				json.Unmarshal(msg, &payload)
				s.hub.Broadcast(Event{Type: "update_progress", Payload: payload})
			}
		}
	}()
}
