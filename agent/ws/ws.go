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

	"arcvault/agent/config"
	"arcvault/agent/updater"
)

// JobCanceller allows the WS client to cancel running jobs on the runner.
type JobCanceller interface {
	CancelJob(jobID string) bool
}

// PollTrigger allows the WS client to trigger an immediate job poll on the runner.
type PollTrigger interface {
	PollNow()
}

// Client holds the configuration for the agent WebSocket connection.
type Client struct {
	AgentID        string
	CoordinatorURL string   // http(s)://host:port (single, backward compat)
	Coordinators   []string // list of coordinator URLs for failover
	AuthToken      string
	CACertFile     string
	Canceller      JobCanceller // used to cancel running jobs
	Poller         PollTrigger  // triggers immediate job poll
	lastSuccessfulCoordinator string // track which coordinator the agent is currently homed to
}

type inboundMsg struct {
	Type        string `json:"type"`
	Version     string `json:"version"`
	URL         string `json:"url"`
	ChecksumURL string `json:"checksum_url"`
	JobID       string `json:"job_id"`
}

type progressMsg struct {
	Type    string `json:"type"`
	AgentID string `json:"agent_id"`
	Step    string `json:"step"`
	Pct     int    `json:"pct"`
}

// Start connects to the coordinator's /ws/agent endpoint and processes
// messages until the process exits. It reconnects automatically on disconnect,
// trying each coordinator in the list with exponential backoff.
func (c *Client) Start() {
	// Build coordinator list: use Coordinators if provided, otherwise fall back to single CoordinatorURL
	coordinators := c.Coordinators
	if len(coordinators) == 0 && c.CoordinatorURL != "" {
		coordinators = []string{c.CoordinatorURL}
	}
	if len(coordinators) == 0 {
		log.Fatal("Agent WS: no coordinators configured")
	}

	backoff := 30 * time.Second
	coordinatorIndex := 0

	for {
		// Try each coordinator in round-robin
		coordinator := coordinators[coordinatorIndex%len(coordinators)]
		wsURL := buildWSURL(coordinator, c.AgentID, c.AuthToken)

		if err := c.run(wsURL); err != nil {
			log.Printf("Agent WS: disconnected from %s (%v) — trying next coordinator in %s", coordinator, err, backoff)
			coordinatorIndex++

			// After trying all coordinators, apply exponential backoff
			if coordinatorIndex%len(coordinators) == 0 {
				time.Sleep(backoff)
				backoff *= 2
				if backoff > 120*time.Second {
					backoff = 120 * time.Second
				}
			}
		} else {
			// Successful connection
			c.lastSuccessfulCoordinator = coordinator
			backoff = 30 * time.Second // Reset backoff on success
			coordinatorIndex++ // Try next one on reconnect
		}
	}
}

func (c *Client) run(url string) error {
	// Build TLS config for WebSocket client
	tlsConfig, err := config.BuildTLSConfig(c.CACertFile)
	if err != nil {
		return fmt.Errorf("failed to build TLS config: %w", err)
	}

	// Create a custom dialer with TLS config
	dialer := &websocket.Dialer{
		Proxy:            websocket.DefaultDialer.Proxy,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  tlsConfig,
	}

	conn, _, err := dialer.Dial(url, nil)
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
		} else if msg.Type == "cancel_command" {
			go c.handleCancelCommand(msg)
		} else if msg.Type == "poll_now" {
			go c.handlePollNow()
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

	if err := updater.HandleUpdateCommand(msg.Version, msg.URL, msg.ChecksumURL, send); err != nil {
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

func (c *Client) handleCancelCommand(msg inboundMsg) {
	log.Printf("Agent WS: received cancel_command for job %s", msg.JobID)
	if c.Canceller == nil {
		log.Printf("Agent WS: no canceller set, cannot cancel job %s", msg.JobID)
		return
	}
	if ok := c.Canceller.CancelJob(msg.JobID); ok {
		log.Printf("Agent WS: cancelled job %s", msg.JobID)
	} else {
		log.Printf("Agent WS: job %s not found or already finished", msg.JobID)
	}
}

func (c *Client) handlePollNow() {
	log.Printf("Agent WS: received poll_now — triggering immediate job poll")
	if c.Poller != nil {
		c.Poller.PollNow()
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
