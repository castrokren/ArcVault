// Package ws maintains a persistent WebSocket connection from the agent to the
// coordinator. It receives update_command messages and relays update_progress
// events back through the same connection.
package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"arcvault/agent/updater"
)

// Client holds the configuration for the agent WebSocket connection.
type Client struct {
	AgentID        string
	CoordinatorURL string // http(s)://host:port
	AuthToken      string
}

type inboundMsg struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

type progressMsg struct {
	Type    string `json:"type"`
	AgentID string `json:"agent_id"`
	Step    string `json:"step"`
	Pct     int    `json:"pct"`
}

// Start connects to the coordinator's /ws/agent endpoint and processes
// messages until the process exits. It reconnects automatically on disconnect.
func (c *Client) Start() {
	wsURL := buildWSURL(c.CoordinatorURL, c.AgentID, c.AuthToken)
	for {
		if err := c.run(wsURL); err != nil {
			log.Printf("Agent WS: disconnected (%v) — reconnecting in 5s", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (c *Client) run(url string) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	log.Printf("Agent WS: connected to coordinator")

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var msg inboundMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		if msg.Type == "update_command" {
			go c.handleUpdateCommand(conn, msg)
		} else if msg.Type == "rollback_command" {
			go c.handleRollbackCommand(conn)
		}
	}
}

func (c *Client) handleUpdateCommand(conn *websocket.Conn, msg inboundMsg) {
	log.Printf("Agent WS: received update_command version=%s", msg.Version)

	send := func(step string, pct int) {
		evt := progressMsg{
			Type:    "update_progress",
			AgentID: c.AgentID,
			Step:    step,
			Pct:     pct,
		}
		raw, _ := json.Marshal(evt)
		if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
			log.Printf("Agent WS: failed to send progress (%s %d%%): %v", step, pct, err)
		}
	}

	if err := updater.HandleUpdateCommand(msg.Version, msg.URL, send); err != nil {
		log.Printf("Agent WS: update failed: %v", err)
		send("error", 0)
	}
}

func (c *Client) handleRollbackCommand(conn *websocket.Conn) {
	log.Printf("Agent WS: received rollback_command")

	send := func(step string, pct int) {
		evt := progressMsg{
			Type:    "rollback_progress",
			AgentID: c.AgentID,
			Step:    step,
			Pct:     pct,
		}
		raw, _ := json.Marshal(evt)
		if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
			log.Printf("Agent WS: failed to send rollback progress (%s %d%%): %v", step, pct, err)
		}
	}

	exePath, err := getAgentBinaryPath()
	if err != nil {
		log.Printf("Agent WS: rollback failed - could not determine binary path: %v", err)
		send("error", 0)
		return
	}

	if err := updater.Rollback(exePath, send); err != nil {
		log.Printf("Agent WS: rollback failed: %v", err)
		send("error", 0)
	}
}

func getAgentBinaryPath() (string, error) {
	return os.Executable()
}

// buildWSURL converts an http(s):// coordinator URL to a ws(s):// URL
// for the /ws/agent endpoint with agent_id and token query params.
func buildWSURL(coordinatorURL, agentID, token string) string {
	wsBase := strings.Replace(coordinatorURL, "http://", "ws://", 1)
	wsBase = strings.Replace(wsBase, "https://", "wss://", 1)
	return fmt.Sprintf("%s/ws/agent?agent_id=%s&token=%s", wsBase, agentID, token)
}
